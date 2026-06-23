package main

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestLatexResumeTemplateParsesWithLiteralLatexBraces(t *testing.T) {
	funcs := template.FuncMap{
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"tex": sanitizeLaTeX,
	}
	tmpl, err := template.New("resume").Delims("[[", "]]").Funcs(funcs).Parse(latexResumeTemplate())
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, ResumeJSON{
		Headline:    "Backend Engineer",
		Summary:     "Backend engineer focused on production systems.",
		ContactLine: "email@example.com | Melbourne, VIC",
		Skills: []ResumeSkill{
			{Category: "Languages", Items: []string{"Python", "Go", "TypeScript"}},
			{Category: "Backend", Items: []string{"FastAPI", "AWS"}},
		},
		Experience: []ResumeEntry{{Company: "ExampleCo", Title: "Fullstack Engineer", Location: "Remote", StartDate: "06/2025", EndDate: "Present", Bullets: []string{"Built production APIs."}}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "Languages") {
		t.Fatalf("skills section missing Languages category, got: %s", text)
	}
	if !strings.Contains(text, "Professional Summary") {
		t.Fatalf("missing Professional Summary section")
	}
	if strings.Contains(text, "[[") || strings.Contains(text, "]]") {
		t.Fatalf("rendered template still contains delimiters: %s", text)
	}
}

func TestValidateResumeJSONDetectsMissingHeadline(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	result, err := store.ValidateResumeJSON(ResumeJSON{Headline: ""}, 1)
	if err != nil {
		t.Fatalf("ValidateResumeJSON() error = %v", err)
	}
	if !listContains(result.TitleIssues, "headline is empty") {
		t.Fatalf("expected headline empty error, got: %+v", result.TitleIssues)
	}
}

func TestPolishResumeSummaryRemovesRoboticPhrases(t *testing.T) {
	summary := polishResumeSummary("cloud software engineer for Software Engineer - AI/ML with 3+ years of experience focused on product_platform_delivery across 3 professional contexts and workflows. Recent work includes shipping backend product features.")
	for _, unwanted := range []string{"product_platform_delivery", "professional contexts", "workflows", "for Software Engineer - AI/ML", "Recent work includes"} {
		if strings.Contains(summary, unwanted) {
			t.Fatalf("summary contains %q: %s", unwanted, summary)
		}
	}
	if !strings.Contains(summary, "product delivery") || !strings.Contains(summary, "systems") || !strings.Contains(summary, "Experience includes") {
		t.Fatalf("summary not humanized: %s", summary)
	}
}

func TestHumanizeValueThemeUsesPlainLanguage(t *testing.T) {
	got := humanizeValueTheme("product_platform_delivery")
	if got != "shipping backend product features" {
		t.Fatalf("humanizeValueTheme() = %q", got)
	}
}

func TestSortResumeEntriesBySourceOrderPreservesTimeline(t *testing.T) {
	entries := []ResumeEntry{
		{Company: "Metadata Services", Title: "IT Systems & Web Developer"},
		{Company: "Sitespace", Title: "Backend Engineer"},
		{Company: "Tata Consultancy Services (TCS)", Title: "Software Engineer"},
	}
	sections := []SourceSection{
		{Heading: "Backend Engineer (Founding Team) | Sitespace | Remote", SectionType: "experience", Content: "Backend Engineer | Sitespace | Remote", SortOrder: 10},
		{Heading: "Software Engineer | Tata Consultancy Services (TCS) | Chennai, India", SectionType: "experience", Content: "Software Engineer | Tata Consultancy Services (TCS) | Chennai, India", SortOrder: 20},
		{Heading: "IT Systems & Web Developer (Part-time) | Metadata Services | Melbourne, Australia", SectionType: "experience", Content: "IT Systems & Web Developer (Part-time) | Metadata Services | Melbourne, Australia", SortOrder: 30},
	}

	sortResumeEntriesBySourceOrder(entries, sections)

	want := []string{"Sitespace", "Tata Consultancy Services (TCS)", "Metadata Services"}
	for i, company := range want {
		if entries[i].Company != company {
			t.Fatalf("entry %d company = %q, want %q; entries=%+v", i, entries[i].Company, company, entries)
		}
	}
}

func TestHumanizeResumeBulletRemovesOverSpecificTone(t *testing.T) {
	got := humanizeResumeBullet("Containerized backend workflows with Python 3.11 — leveraged cutting-edge integration workflows for deployment.")
	for _, unwanted := range []string{"Python 3.11", "leveraged", "workflows", "—", "cutting-edge"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(unwanted)) {
			t.Fatalf("bullet contains %q: %s", unwanted, got)
		}
	}
	if !strings.Contains(got, "Python") || !strings.Contains(got, "backend logic") || !strings.Contains(got, "integrations") {
		t.Fatalf("bullet not humanized: %s", got)
	}
}

func TestNormalizeHumanResumeTextRemovesAIStylePunctuationAndFiller(t *testing.T) {
	got := polishResumeSummary("Dynamic software engineer — leveraged cutting-edge systems in order to enhance efficiency and drive growth.")
	for _, unwanted := range []string{"—", "leveraged", "cutting-edge", "in order to", "enhance efficiency", "drive growth", "Dynamic"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(unwanted)) {
			t.Fatalf("summary contains %q: %s", unwanted, got)
		}
	}
}

func TestPolishResumeSummaryRemovesGenericAISummaryShape(t *testing.T) {
	got := polishResumeSummary("Backend software engineer with 3+ years of experience with a practical focus on experience building and operating customer-facing software in modern engineering. Comfortable working across Docker, Next.js, AWS, Bash, Elastic Stack, Elasticsearch, with experience turning product requirements into reliable backend features.")
	for _, unwanted := range []string{"practical focus", "comfortable working across", "experience with a practical focus", "with experience turning", "experience building and operating"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(unwanted)) {
			t.Fatalf("summary contains %q: %s", unwanted, got)
		}
	}
	if !strings.Contains(got, "building and operating customer-facing software") {
		t.Fatalf("summary lost human work focus: %s", got)
	}
}

func TestNormalizeHumanEditCleansSummaryAndPreservesOrder(t *testing.T) {
	original := ResumeJSON{
		Headline: "Roshan Ravikumar",
		Summary:  "Backend engineer with production API experience.",
		Experience: []ResumeEntry{
			{Company: "Sitespace", Title: "Backend Engineer", Bullets: []string{"Built APIs."}, ClaimIDs: []int64{1}, BulletIDs: []int64{10}},
			{Company: "TCS", Title: "Software Engineer", Bullets: []string{"Supported services."}, ClaimIDs: []int64{2}, BulletIDs: []int64{20}},
		},
	}
	edited := ResumeJSON{
		Summary: "cloud software engineer for Software Engineer - AI/ML. Recent work includes shipping backend product features.",
		Experience: []ResumeEntry{
			{Company: "Wrong", Title: "Wrong", Bullets: []string{"Leveraged Python 3.11 workflows."}},
			{Company: "Wrong", Title: "Wrong", Bullets: []string{"Utilized backend workflows."}},
		},
	}

	got := normalizeResumeJSONForHumanEdit(original, edited)

	if got.Experience[0].Company != "Sitespace" || got.Experience[1].Company != "TCS" {
		t.Fatalf("experience order/details changed: %+v", got.Experience)
	}
	for _, unwanted := range []string{"for Software Engineer - AI/ML", "Recent work includes", "Leveraged", "Python 3.11", "workflows"} {
		if strings.Contains(strings.ToLower(got.Summary+" "+strings.Join(got.Experience[0].Bullets, " ")+" "+strings.Join(got.Experience[1].Bullets, " ")), strings.ToLower(unwanted)) {
			t.Fatalf("normalized resume contains %q: %+v", unwanted, got)
		}
	}
}
