package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStoreCreatesLocalPaths(t *testing.T) {
	root := t.TempDir()

	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	for _, path := range []string{
		filepath.Join(root, "data", "app.db"),
		filepath.Join(root, "logs", "app.log"),
		filepath.Join(root, "generated"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	root := t.TempDir()

	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	if err := store.migrate(t.Context()); err != nil {
		t.Fatalf("second migrate() error = %v", err)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("migration version count = %d, want 1", count)
	}
}

func TestSettingsSaveLoad(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	initial, err := store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if initial.Provider != defaultProvider {
		t.Fatalf("Provider = %q, want %q", initial.Provider, defaultProvider)
	}
	if initial.APIKeyConfigured {
		t.Fatal("APIKeyConfigured = true, want false")
	}

	saved, err := store.SaveSettings(SaveSettingsInput{
		Provider: "openrouter",
		Model:    "deepseek/test",
	})
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if saved.Model != "deepseek/test" {
		t.Fatalf("saved Model = %q, want deepseek/test", saved.Model)
	}

	loaded, err := store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() after save error = %v", err)
	}
	if loaded.Model != "deepseek/test" {
		t.Fatalf("loaded Model = %q, want deepseek/test", loaded.Model)
	}
}

func TestEnvLocalParseWriteWithoutExposingKey(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	secret := "sk-test-secret"
	status, err := store.SaveAPIKey(SaveAPIKeyInput{APIKey: secret})
	if err != nil {
		t.Fatalf("SaveAPIKey() error = %v", err)
	}
	if !status.APIKeyConfigured {
		t.Fatal("APIKeyConfigured = false, want true")
	}
	if strings.Contains(status.APIKeySource, secret) {
		t.Fatal("status exposed API key")
	}
	content, err := os.ReadFile(filepath.Join(store.root, ".env.local"))
	if err != nil {
		t.Fatalf("read .env.local: %v", err)
	}
	if string(content) != "OPENROUTER_API_KEY="+secret+"\n" {
		t.Fatalf(".env.local content = %q", string(content))
	}
}

func TestOpenRouterEnvVarPrecedence(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	if err := writeEnvLocalValue(store.envLocalPath(), "OPENROUTER_API_KEY", "sk-from-file"); err != nil {
		t.Fatalf("writeEnvLocalValue() error = %v", err)
	}
	t.Setenv("OPENROUTER_API_KEY", "sk-from-env")

	key, source := store.APIKeyForProvider("openrouter")
	if key != "sk-from-env" {
		t.Fatalf("key = %q, want env key", key)
	}
	if source != "OPENROUTER_API_KEY environment" {
		t.Fatalf("source = %q, want OPENROUTER_API_KEY environment", source)
	}
}

func TestBuildChatCompletionRequestShape(t *testing.T) {
	body, err := buildChatCompletionRequest("")
	if err != nil {
		t.Fatalf("buildChatCompletionRequest() error = %v", err)
	}
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if request["model"] != defaultModel {
		t.Fatalf("model = %v, want %s", request["model"], defaultModel)
	}
	if _, ok := request["messages"].([]any); !ok {
		t.Fatal("messages is missing")
	}
	if request["temperature"] != float64(0) {
		t.Fatalf("temperature = %v, want 0", request["temperature"])
	}
}

func TestLLMMissingKeyFailsWithoutNetwork(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENROUTER_API_KEY", "")
	called := false
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})}

	result, err := store.TestLLM(t.Context(), client)
	if err != nil {
		t.Fatalf("TestLLM() error = %v", err)
	}
	if result.Success {
		t.Fatal("Success = true, want false")
	}
	if result.Error == "" {
		t.Fatal("Error is empty")
	}
	if called {
		t.Fatal("network client was called without API key")
	}
}

func TestLLMUsesOpenRouterChatCompletionsShape(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	if _, err := store.SaveSettings(SaveSettingsInput{Provider: "openrouter", Model: "deepseek/test"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("authorization header not set")
		}
		if r.Header.Get("X-OpenRouter-Title") != "JD Tailor" {
			t.Fatalf("OpenRouter title header not set")
		}
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != "deepseek/test" || len(request.Messages) == 0 {
			t.Fatalf("bad request: %+v", request)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"JD Tailor LLM check"}}]}`))
	}))
	defer server.Close()

	originalURL := openRouterChatCompletionsURLForTest(server.URL)
	defer originalURL()
	result, err := store.TestLLM(t.Context(), server.Client())
	if err != nil {
		t.Fatalf("TestLLM() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("Success = false, error = %q", result.Error)
	}
	if result.Text != "JD Tailor LLM check" {
		t.Fatalf("Text = %q", result.Text)
	}
}

func TestEventLogging(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if err := store.LogEvent("warning", "test event"); err != nil {
		t.Fatalf("LogEvent() error = %v", err)
	}

	events, err := store.GetRecentEvents(5)
	if err != nil {
		t.Fatalf("GetRecentEvents() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("GetRecentEvents() returned no events")
	}
	if events[0].Message != "test event" {
		t.Fatalf("latest event = %q, want test event", events[0].Message)
	}
}

func TestCandidateProfileSaveLoad(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	saved, err := store.SaveCandidateProfile(CandidateProfile{
		Contact: CandidateContact{
			FullName: "Roshan Example",
			Email:    "roshan@example.com",
			Links:    []string{"https://portfolio.example", "https://portfolio.example"},
			Verified: true,
		},
		Records: []CandidateProfileRecord{
			{
				RecordType:   "employment",
				Organization: "Acme",
				Role:         "Backend Engineer",
				StartDate:    "2024",
				EndDate:      "2025",
				Value:        "Official title: Backend Engineer",
				Verified:     true,
			},
			{
				RecordType: "blocked_alias",
				Value:      "Do not claim ML Engineer",
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveCandidateProfile() error = %v", err)
	}
	if saved.Contact.FullName != "Roshan Example" {
		t.Fatalf("FullName = %q", saved.Contact.FullName)
	}
	if len(saved.Contact.Links) != 1 {
		t.Fatalf("Links = %+v, want deduplicated single link", saved.Contact.Links)
	}
	if !saved.Contact.Verified {
		t.Fatal("Contact.Verified = false, want true")
	}

	loaded, err := store.GetCandidateProfile()
	if err != nil {
		t.Fatalf("GetCandidateProfile() error = %v", err)
	}
	if len(loaded.Records) != 2 {
		t.Fatalf("Records len = %d, want 2", len(loaded.Records))
	}
	if !loaded.Records[0].Verified && !loaded.Records[1].Verified {
		t.Fatal("expected at least one verified profile record")
	}
}

func TestCandidateSourceAndSectionDetection(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		SourceType: "current_resume",
		Title:      "Resume",
		RawText:    "SUMMARY\nBuilt APIs.\n\nTECHNICAL SKILLS\nGo, React",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	if source.ID == 0 {
		t.Fatal("source ID was not assigned")
	}

	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("sections len = %d, want 2", len(sections))
	}
	if sections[0].SectionType != "summary" || sections[1].SectionType != "skills" {
		t.Fatalf("section types = %q, %q", sections[0].SectionType, sections[1].SectionType)
	}

	updated, err := store.UpdateSourceSection(UpdateSourceSectionInput{
		ID:          sections[0].ID,
		Heading:     "Professional Summary",
		SectionType: "summary",
		Content:     "Built backend APIs.",
	})
	if err != nil {
		t.Fatalf("UpdateSourceSection() error = %v", err)
	}
	if updated.Content != "Built backend APIs." {
		t.Fatalf("updated content = %q", updated.Content)
	}
}

func TestImportCandidateSourceFileAllowsTextMarkdownLatexOnly(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	mdPath := filepath.Join(root, "notes.md")
	if err := os.WriteFile(mdPath, []byte("# Project\nBuilt APIs."), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}
	source, err := store.ImportCandidateSourceFile(ImportCandidateSourceFileInput{Path: mdPath})
	if err != nil {
		t.Fatalf("ImportCandidateSourceFile() md error = %v", err)
	}
	if source.RawText == "" || source.FilePath != mdPath {
		t.Fatalf("source = %+v", source)
	}

	texPath := filepath.Join(root, "resume.tex")
	texContent := `\section{Experience}
Built FastAPI services.`
	if err := os.WriteFile(texPath, []byte(texContent), 0o644); err != nil {
		t.Fatalf("write tex: %v", err)
	}
	latexSource, err := store.ImportCandidateSourceFile(ImportCandidateSourceFileInput{Path: texPath})
	if err != nil {
		t.Fatalf("ImportCandidateSourceFile() tex error = %v", err)
	}
	if !strings.Contains(latexSource.RawText, "Experience") || !strings.Contains(latexSource.RawText, "Built FastAPI services.") || latexSource.FilePath != texPath {
		t.Fatalf("latexSource = %+v", latexSource)
	}
	if strings.Contains(latexSource.RawText, `\section`) {
		t.Fatalf("latexSource RawText still contains LaTeX formatting: %q", latexSource.RawText)
	}

	pdfPath := filepath.Join(root, "resume.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF"), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	if _, err := store.ImportCandidateSourceFile(ImportCandidateSourceFileInput{Path: pdfPath}); err == nil {
		t.Fatal("ImportCandidateSourceFile() pdf error = nil, want unsupported extension error")
	}
}

func TestLatexResumeImportCleansReadableContent(t *testing.T) {
	raw := `% comment
\documentclass{article}
\begin{document}
\begin{center}
\textbf{\Huge \scshape Roshan Ravikumar} \\
\href{mailto:roshrwork@gmail.com}{roshrwork@gmail.com} $|$ Melbourne, VIC
\end{center}
\section{Experience}
\resumeSubheading{Sitespace $|$ \href{https://sitespace.com.au}{Website}}{Remote}{Fullstack Engineer}{06/2025 -- Present}
\resumeItemListStart
\begin{itemize}[leftmargin=0.15in, label=]
\resumeItem{Built and shipped the FastAPI/PostgreSQL backend for a construction planning platform.}
\end{itemize}
\resumeItemListEnd
\end{document}`

	cleaned := normalizeRawSourceText(raw)
	for _, forbidden := range []string{`\documentclass`, `\begin`, `\section`, `\resumeItem`, `\href`, `\textbf`, "leftmargin", "label=", "ListStart", "ListEnd"} {
		if strings.Contains(cleaned, forbidden) {
			t.Fatalf("cleaned text contains %q: %s", forbidden, cleaned)
		}
	}
	for _, required := range []string{"Roshan Ravikumar", "roshrwork@gmail.com", "Experience", "Sitespace", "Fullstack Engineer", "Built and shipped the FastAPI/PostgreSQL backend"} {
		if !strings.Contains(cleaned, required) {
			t.Fatalf("cleaned text missing %q: %s", required, cleaned)
		}
	}
}

func TestDetectSectionsUsesLatexResumeSections(t *testing.T) {
	raw := `\begin{document}
\begin{center}
\textbf{\Huge \scshape Roshan Ravikumar}
\end{center}
\section{Professional Summary}
\small{Backend engineer.}
\section{Technical Skills}
\textbf{Languages}{: Go, Python}
\section{Experience}
\resumeSubheading{Sitespace}{Remote}{Fullstack Engineer}{06/2025 -- Present}
\resumeItem{Built FastAPI systems.}
\resumeSubheading{SecondCo}{Remote}{Backend Engineer}{01/2024 -- 05/2025}
\resumeItem{Built queue workers.}
\section{Projects}
\resumeProjectHeading{\textbf{CueMate}}{03/2026 -- Present}
\resumeItem{Built a recommendation platform.}
\section{Education}
\resumeSubheading{Monash University}{Melbourne}{Master of Information Technology}{08/2022 -- 07/2024}
\end{document}`

	sections := detectSections(raw)
	if len(sections) != 6 {
		t.Fatalf("sections len = %d, want 6: %+v", len(sections), sections)
	}
	want := []string{"Professional Summary", "Technical Skills", "Sitespace - Fullstack Engineer", "SecondCo - Backend Engineer", "CueMate", "Monash University - Master of Information Technology"}
	for index, heading := range want {
		if sections[index].Heading != heading {
			t.Fatalf("heading[%d] = %q, want %q", index, sections[index].Heading, heading)
		}
	}
	if sections[2].SectionType != "experience" || sections[4].SectionType != "project" || sections[5].SectionType != "education" {
		t.Fatalf("section types = %q, %q, %q", sections[2].SectionType, sections[4].SectionType, sections[5].SectionType)
	}
	if strings.Contains(sections[2].Content, `\resumeItem`) || !strings.Contains(sections[2].Content, "Built FastAPI systems.") || strings.Contains(sections[2].Content, "Built queue workers.") {
		t.Fatalf("experience section content not cleaned: %q", sections[2].Content)
	}
}

func TestRefineExtractedFactsFallsBackToBulletEvidence(t *testing.T) {
	section := SourceSection{
		Heading:     "Sitespace - Fullstack Engineer",
		SectionType: "experience",
		Content: "Sitespace | Remote\n" +
			"Fullstack Engineer | 06/2025 -- Present\n" +
			"- Built FastAPI systems.\n",
	}

	facts := refineExtractedFacts(section, []extractedFact{{
		FactText:      section.Content,
		EvidenceQuote: section.Content,
		Confidence:    "medium",
	}})
	if len(facts) != 1 {
		t.Fatalf("facts len = %d, want 1: %+v", len(facts), facts)
	}
	if facts[0].FactText != "Built FastAPI systems." || facts[0].EvidenceQuote != "- Built FastAPI systems." {
		t.Fatalf("fallback fact = %+v", facts[0])
	}
}

func TestDraftCandidateProfileFromLatexSourceNeedsVerification(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		SourceType: "current_resume",
		Title:      "Resume",
		RawText: `\begin{document}
\begin{center}
\textbf{\Huge \scshape Roshan Ravikumar} \\
\href{mailto:roshrwork@gmail.com}{roshrwork@gmail.com} $|$ Melbourne, VIC
\end{center}
\section{Experience}
\resumeSubheading{Sitespace}{Remote}{Fullstack Engineer}{06/2025 -- Present}
\section{Education}
\resumeSubheading{Monash University}{Melbourne, Australia}{Master of Information Technology}{08/2022 -- 07/2024}
\end{document}`,
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	profile, err := store.DraftCandidateProfileFromSource(source.ID)
	if err != nil {
		t.Fatalf("DraftCandidateProfileFromSource() error = %v", err)
	}
	if profile.Contact.FullName != "Roshan Ravikumar" || profile.Contact.Email != "roshrwork@gmail.com" {
		t.Fatalf("draft contact = %+v", profile.Contact)
	}
	if profile.Contact.Verified {
		t.Fatal("draft contact was verified automatically")
	}
	if len(profile.Records) == 0 {
		t.Fatal("draft profile had no records")
	}
	for _, record := range profile.Records {
		if record.Verified {
			t.Fatalf("draft record was verified automatically: %+v", record)
		}
	}
}

func TestDeleteCandidateSourceCascadesContext(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:   "Notes",
		RawText: "PROJECTS\nBuilt FastAPI backend.",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}
	if _, err := store.insertExtractedFacts(sections[0], []extractedFact{{
		FactText:      "Built FastAPI backend.",
		EvidenceQuote: "Built FastAPI backend.",
		Confidence:    "high",
	}}); err != nil {
		t.Fatalf("insertExtractedFacts() error = %v", err)
	}
	if err := store.DeleteCandidateSource(DeleteInput{ID: source.ID}); err != nil {
		t.Fatalf("DeleteCandidateSource() error = %v", err)
	}
	remainingSections, err := store.ListSourceSections(0)
	if err != nil {
		t.Fatalf("ListSourceSections() error = %v", err)
	}
	if len(remainingSections) != 0 {
		t.Fatalf("remaining sections = %+v", remainingSections)
	}
	remainingFacts, err := store.ListEvidenceFacts("all")
	if err != nil {
		t.Fatalf("ListEvidenceFacts() error = %v", err)
	}
	if len(remainingFacts) != 0 {
		t.Fatalf("remaining facts = %+v", remainingFacts)
	}
}

func TestParseExtractedFactsRequiresEvidenceQuotes(t *testing.T) {
	_, err := parseExtractedFacts(`{"facts":[{"fact_text":"Built APIs","confidence":"high"}]}`)
	if err == nil {
		t.Fatal("parseExtractedFacts() error = nil, want missing evidence error")
	}

	facts, err := parseExtractedFacts("```json\n{\"facts\":[{\"fact_text\":\"Built APIs\",\"evidence_quote\":\"Built APIs\",\"technologies\":[\"Go\"],\"confidence\":\"high\",\"risk_flags\":[]}]}\n```")
	if err != nil {
		t.Fatalf("parseExtractedFacts() error = %v", err)
	}
	if len(facts) != 1 || facts[0].EvidenceQuote != "Built APIs" {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestInsertExtractedFactsReviewPolicy(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:   "Notes",
		RawText: "PROJECTS\nBuilt FastAPI backend.\nImproved latency by 30%.",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}

	inserted, err := store.insertExtractedFacts(sections[0], []extractedFact{
		{
			FactText:      "Built FastAPI backend.",
			EvidenceQuote: "Built FastAPI backend.",
			Confidence:    "high",
		},
		{
			FactText:      "Improved latency by 30%.",
			EvidenceQuote: "Improved latency by 30%.",
			Confidence:    "high",
			RiskFlags:     []string{"unclear_metric"},
		},
	})
	if err != nil {
		t.Fatalf("insertExtractedFacts() error = %v", err)
	}
	if inserted[0].Status != factStatusApproved || !inserted[0].AutoApproved {
		t.Fatalf("first fact status = %+v, want auto-approved", inserted[0])
	}
	if inserted[1].Status != factStatusNeedsReview || inserted[1].AutoApproved {
		t.Fatalf("second fact status = %+v, want needs_review", inserted[1])
	}

	reviewed, err := store.UpdateEvidenceFactReview(UpdateEvidenceFactReviewInput{
		ID:            inserted[1].ID,
		FactText:      inserted[1].FactText,
		EvidenceQuote: inserted[1].EvidenceQuote,
		Confidence:    inserted[1].Confidence,
		RiskFlags:     inserted[1].RiskFlags,
		Status:        factStatusApproved,
		ReviewNote:    "Metric came from source note.",
	})
	if err != nil {
		t.Fatalf("UpdateEvidenceFactReview() error = %v", err)
	}
	if reviewed.Status != factStatusApproved || reviewed.ReviewNote == "" {
		t.Fatalf("reviewed = %+v", reviewed)
	}
}

func TestTectonicStatusRepoLocalOnly(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	status := store.TectonicStatus()
	if status.Status != "missing" {
		t.Fatalf("Status = %q, want missing", status.Status)
	}
	if !strings.HasSuffix(status.ExecutablePath, filepath.Join("tools", "tectonic", "tectonic.exe")) {
		t.Fatalf("ExecutablePath = %q, want repo-local tectonic.exe", status.ExecutablePath)
	}
}

func TestTectonicDownloadURLSelectsWindowsAsset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"assets": [
				{"name": "tectonic-0.16.9-x86_64-unknown-linux-gnu.tar.gz", "browser_download_url": "linux"},
				{"name": "tectonic-0.16.9-x86_64-pc-windows-msvc.zip", "browser_download_url": "windows"}
			]
		}`))
	}))
	defer server.Close()

	previous := tectonicLatestReleaseURL
	tectonicLatestReleaseURL = server.URL
	defer func() {
		tectonicLatestReleaseURL = previous
	}()

	url, err := tectonicDownloadURL(t.Context(), server.Client())
	if err != nil {
		t.Fatalf("tectonicDownloadURL() error = %v", err)
	}
	if url != "windows" {
		t.Fatalf("url = %q, want windows", url)
	}
}

func TestRenderSamplePDFWithFakeExecutable(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	fakePath := store.tectonicPath()
	if err := os.MkdirAll(filepath.Dir(fakePath), 0o755); err != nil {
		t.Fatalf("mkdir fake path: %v", err)
	}
	currentExe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	data, err := os.ReadFile(currentExe)
	if err != nil {
		t.Fatalf("read test exe: %v", err)
	}
	if err := os.WriteFile(fakePath, data, 0o755); err != nil {
		t.Fatalf("write fake exe: %v", err)
	}
	originalCommand := execCommandContext
	execCommandContext = func(_ context.Context, _ string, args ...string) *exec.Cmd {
		testArgs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
		cmd := exec.Command(currentExe, testArgs...)
		cmd.Env = append(os.Environ(), "JDTAILOR_FAKE_TECTONIC=1")
		return cmd
	}
	defer func() {
		execCommandContext = originalCommand
	}()

	result, err := store.RenderSamplePDF(t.Context())
	if err != nil {
		t.Fatalf("RenderSamplePDF() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("Success = false, error = %q", result.Error)
	}
	if _, err := os.Stat(result.TexPath); err != nil {
		t.Fatalf("tex missing: %v", err)
	}
	if _, err := os.Stat(result.PDFPath); err != nil {
		t.Fatalf("pdf missing: %v", err)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("JDTAILOR_FAKE_TECTONIC") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--outdir" && i+1 < len(os.Args) {
			if err := os.WriteFile(filepath.Join(os.Args[i+1], "sample.pdf"), []byte("%PDF-1.4\n"), 0o644); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	os.Exit(1)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func openAIResponsesURLForTest(url string) func() {
	previous := openAIResponsesURL
	openAIResponsesURL = url
	return func() {
		openAIResponsesURL = previous
	}
}

func openRouterChatCompletionsURLForTest(url string) func() {
	previous := openRouterChatCompletionsURL
	openRouterChatCompletionsURL = url
	return func() {
		openRouterChatCompletionsURL = previous
	}
}
