package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	jobFetchBodyLimit = 4 * 1024 * 1024
	jobFetchMinText   = 200
	jobFetchMaxText   = 50000
)

type FetchJobDescriptionInput struct {
	URL string `json:"url"`
}

type FetchJobDescriptionResult struct {
	Company  string   `json:"company"`
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	RawText  string   `json:"raw_text"`
	Source   string   `json:"source"`
	Warnings []string `json:"warnings"`
}

func (s *Store) FetchJobDescription(ctx context.Context, input FetchJobDescriptionInput, client *http.Client) (FetchJobDescriptionResult, error) {
	jobURL, err := normalizeJobFetchURL(input.URL)
	if err != nil {
		return FetchJobDescriptionResult{}, err
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jobURL, nil)
	if err != nil {
		return FetchJobDescriptionResult{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) JD Tailor/1.0")
	req.Header.Set("Accept", "text/html,text/plain;q=0.9,*/*;q=0.2")

	resp, err := client.Do(req)
	if err != nil {
		return FetchJobDescriptionResult{}, fmt.Errorf("fetch job description: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FetchJobDescriptionResult{}, fmt.Errorf("job posting fetch failed: HTTP %d", resp.StatusCode)
	}
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	mediaType = strings.ToLower(mediaType)
	if mediaType != "" && mediaType != "text/html" && mediaType != "application/xhtml+xml" && mediaType != "text/plain" {
		return FetchJobDescriptionResult{}, fmt.Errorf("unsupported job posting content type: %s", mediaType)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, jobFetchBodyLimit+1))
	if err != nil {
		return FetchJobDescriptionResult{}, err
	}
	if len(body) > jobFetchBodyLimit {
		return FetchJobDescriptionResult{}, errors.New("job posting response is too large")
	}

	extracted := scrapeExtractedJob{}
	if mediaType == "text/plain" {
		extracted.RawText = normalizePastedText(string(body))
		extracted.Source = "plain_text"
	} else {
		extracted = extractJobDescriptionHTML(string(body))
	}
	text := normalizePastedText(extracted.RawText)
	if len(text) > jobFetchMaxText {
		text = strings.TrimSpace(text[:jobFetchMaxText])
		extracted.Warnings = append(extracted.Warnings, "Extracted text was truncated to 50000 characters.")
	}
	if len(text) < jobFetchMinText {
		return FetchJobDescriptionResult{}, errors.New("extracted job description is too short; the page may require login or JavaScript")
	}
	details := inferJobDetails(text)
	company := strings.TrimSpace(extracted.Company)
	if company == "" {
		company = details.Company
	}
	title := strings.TrimSpace(extracted.Title)
	if title == "" {
		title = details.Title
	}
	return FetchJobDescriptionResult{Company: company, Title: title, URL: jobURL, RawText: text, Source: extracted.Source, Warnings: extracted.Warnings}, nil
}

func normalizeJobFetchURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("job posting URL is required")
	}
	if !strings.Contains(raw, "://") && regexp.MustCompile(`^[A-Za-z0-9.-]+\.[A-Za-z]{2,}(/.*)?$`).MatchString(raw) {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", errors.New("invalid job posting URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("job posting URL must use http or https")
	}
	if rewritten := normalizeLinkedInJobURL(u); rewritten != "" {
		return rewritten, nil
	}
	return u.String(), nil
}

func normalizeLinkedInJobURL(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	if host != "linkedin.com" && !strings.HasSuffix(host, ".linkedin.com") {
		return ""
	}
	if id := u.Query().Get("currentJobId"); regexp.MustCompile(`^\d+$`).MatchString(id) {
		return "https://www.linkedin.com/jobs/view/" + id
	}
	if ids := u.Query().Get("originToLandingJobPostings"); ids != "" {
		for _, id := range strings.Split(ids, ",") {
			id = strings.TrimSpace(id)
			if regexp.MustCompile(`^\d+$`).MatchString(id) {
				return "https://www.linkedin.com/jobs/view/" + id
			}
		}
	}
	return ""
}

type scrapeExtractedJob struct {
	Company  string
	Title    string
	RawText  string
	Source   string
	Warnings []string
}

func extractJobDescriptionHTML(rawHTML string) scrapeExtractedJob {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return scrapeExtractedJob{RawText: htmlTextFallback(rawHTML), Source: "html_fallback", Warnings: []string{"HTML parsing failed; used text fallback."}}
	}
	meta := collectJobMeta(doc)
	blocks := collectSemanticJobBlocks(doc)
	best := ""
	for _, block := range blocks {
		text := normalizePastedText(extractVisibleText(block))
		if len(text) > len(best) {
			best = text
		}
	}
	if len(best) < jobFetchMinText {
		best = normalizePastedText(extractVisibleText(doc))
	}
	if isJobFetchLoginGate(best, meta["title"]) {
		return scrapeExtractedJob{RawText: "", Source: "login_gate", Warnings: []string{"The site returned a login page instead of the job description."}}
	}
	company, title := splitJobTitleMeta(meta["title"])
	if title == "" {
		title = firstNonEmptyScrape(meta["og:title"], meta["twitter:title"], meta["title"])
	}
	if company == "" {
		company = firstNonEmptyScrape(meta["og:site_name"], meta["application-name"])
	}
	return scrapeExtractedJob{Company: company, Title: cleanMetaText(title), RawText: best, Source: "html"}
}

func isJobFetchLoginGate(text string, title string) bool {
	lowerText := strings.ToLower(normalizePastedText(text))
	lowerTitle := strings.ToLower(title)
	if strings.Contains(lowerTitle, "linkedin login") || strings.Contains(lowerTitle, "sign in | linkedin") {
		return true
	}
	loginTerms := 0
	for _, term := range []string{"linkedin login", "sign in", "user agreement", "privacy policy", "cookie policy", "linkedin corporation"} {
		if strings.Contains(lowerText, term) {
			loginTerms++
		}
	}
	return loginTerms >= 4
}

func collectJobMeta(n *html.Node) map[string]string {
	meta := map[string]string{}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Data == "title" {
				meta["title"] = normalizePastedText(extractVisibleText(node))
			}
			if node.Data == "meta" {
				key := firstNonEmptyScrape(attr(node, "property"), attr(node, "name"))
				if key != "" && attr(node, "content") != "" {
					meta[strings.ToLower(key)] = attr(node, "content")
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return meta
}

func collectSemanticJobBlocks(n *html.Node) []*html.Node {
	var blocks []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && !skipJobNode(node) {
			data := strings.ToLower(node.Data)
			marker := strings.ToLower(attr(node, "class") + " " + attr(node, "id") + " " + attr(node, "data-testid"))
			if data == "main" || data == "article" || strings.Contains(marker, "job") || strings.Contains(marker, "description") || strings.Contains(marker, "posting") {
				blocks = append(blocks, node)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return blocks
}

func extractVisibleText(n *html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if skipJobNode(node) {
			return
		}
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				parts = append(parts, text)
			}
		}
		if node.Type == html.ElementNode && (node.Data == "p" || node.Data == "li" || node.Data == "br" || strings.HasPrefix(node.Data, "h")) {
			parts = append(parts, "\n")
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(parts, " ")
}

func skipJobNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch strings.ToLower(n.Data) {
	case "script", "style", "noscript", "svg", "header", "footer", "nav", "aside", "form", "button", "select", "option", "input", "textarea":
		return true
	}
	return false
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func htmlTextFallback(raw string) string {
	return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(raw, " ")
}

func splitJobTitleMeta(title string) (string, string) {
	title = cleanMetaText(title)
	for _, sep := range []string{" | ", " - ", " at "} {
		before, after, ok := strings.Cut(title, sep)
		if ok {
			if sep == " at " {
				return cleanMetaText(after), cleanMetaText(before)
			}
			return cleanMetaText(after), cleanMetaText(before)
		}
	}
	return "", title
}

func cleanMetaText(text string) string {
	return strings.Trim(strings.Join(strings.Fields(text), " "), "|- ")
}

func firstNonEmptyScrape(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
