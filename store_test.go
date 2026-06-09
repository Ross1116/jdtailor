package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestGenerateLLMTextUsesJSONModeAndDetectsLength(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	if _, err := store.SaveSettings(SaveSettingsInput{Provider: "openrouter", Model: "deepseek/test"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.ResponseFormat == nil || request.ResponseFormat.Type != "json_object" {
			t.Fatalf("response format = %+v, want json_object", request.ResponseFormat)
		}
		if call == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"ok\":true}"},"finish_reason":"stop"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"ok\":"},"finish_reason":"length"}]}`))
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	text, err := store.GenerateLLMText(t.Context(), server.Client(), "Return JSON.", "Return JSON.", 64)
	if err != nil || text != `{"ok":true}` {
		t.Fatalf("GenerateLLMText() = %q, %v", text, err)
	}
	if _, err := store.GenerateLLMText(t.Context(), server.Client(), "Return JSON.", "Return JSON.", 64); err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("GenerateLLMText() truncation error = %v", err)
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
	if !strings.Contains(facts[0].FactText, "tools=FastAPI") || !strings.Contains(facts[0].FactText, "actions=built") || facts[0].EvidenceQuote != "- Built FastAPI systems." {
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

func TestDeleteAllEvidenceFactsClearsDependentJobOutputs(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Title:   "Backend Engineer",
		RawText: "Must build APIs.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	requirements, err := store.replaceJobRequirements(job, []parsedJobRequirement{{
		Category:        "must_have",
		RequirementText: "Must build APIs.",
		Keywords:        []string{"api"},
		Priority:        "high",
		SourceQuote:     "Must build APIs.",
	}})
	if err != nil {
		t.Fatalf("replaceJobRequirements() error = %v", err)
	}
	_, facts := createFactsForJobTests(t, store)
	if _, err := store.replaceJobMatches(job.ID, []parsedJobMatch{{
		RequirementID:  requirements[0].ID,
		FactID:         facts[0].ID,
		Score:          0.8,
		CoverageStatus: "strong",
	}}, requirements, []factPromptContext{{ID: facts[0].ID, Status: facts[0].Status}}); err != nil {
		t.Fatalf("replaceJobMatches() error = %v", err)
	}
	if _, err := store.replaceBulletDrafts(job.ID, []parsedBulletDraft{{
		RequirementID: requirements[0].ID,
		FactIDs:       []int64{facts[0].ID},
		DraftText:     "Built API workflows with reliable backend delivery.",
	}}, requirements, []factPromptContext{{ID: facts[0].ID, SectionHeading: "Sitespace", SectionType: "experience"}}); err != nil {
		t.Fatalf("replaceBulletDrafts() error = %v", err)
	}
	if _, err := store.GenerateFitAnalysis(job.ID); err != nil {
		t.Fatalf("GenerateFitAnalysis() error = %v", err)
	}
	if err := store.DeleteAllEvidenceFacts(); err != nil {
		t.Fatalf("DeleteAllEvidenceFacts() error = %v", err)
	}
	factsAfter, err := store.ListEvidenceFacts("all")
	if err != nil {
		t.Fatalf("ListEvidenceFacts() error = %v", err)
	}
	matchesAfter, err := store.ListJobFactMatches(job.ID)
	if err != nil {
		t.Fatalf("ListJobFactMatches() error = %v", err)
	}
	draftsAfter, err := store.ListTailoredBulletDrafts(job.ID)
	if err != nil {
		t.Fatalf("ListTailoredBulletDrafts() error = %v", err)
	}
	if len(factsAfter) != 0 || len(matchesAfter) != 0 || len(draftsAfter) != 0 {
		t.Fatalf("remaining facts/matches/drafts = %+v/%+v/%+v", factsAfter, matchesAfter, draftsAfter)
	}
	if _, err := store.GetFitAnalysis(job.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetFitAnalysis() error = %v, want sql.ErrNoRows", err)
	}
}

func TestParseExtractedFactsRequiresEvidenceQuotes(t *testing.T) {
	_, err := parseExtractedFacts(`{"facts":[{"fact_text":"Built APIs","confidence":"high"}]}`)
	if err == nil {
		t.Fatal("parseExtractedFacts() error = nil, want missing evidence error")
	}
	if _, err := parseExtractedFacts(""); err == nil || !strings.Contains(err.Error(), "empty fact response") {
		t.Fatalf("parseExtractedFacts() empty error = %v, want empty fact response", err)
	}

	facts, err := parseExtractedFacts("```json\n{\"facts\":[{\"fact_text\":\"Built APIs\",\"evidence_quote\":\"Built APIs\",\"technologies\":[\"Go\"],\"confidence\":\"high\",\"risk_flags\":[]}]}\n```")
	if err != nil {
		t.Fatalf("parseExtractedFacts() error = %v", err)
	}
	if len(facts) != 1 || facts[0].EvidenceQuote != "Built APIs" {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestExtractEvidenceFactsFallsBackOnEmptyLLMResponse(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	if _, err := store.SaveSettings(SaveSettingsInput{Provider: "openrouter", Model: "deepseek/test"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:   "Resume",
		RawText: "PROJECTS\nCueMate | 2026\n- Built FastAPI recommendation APIs.",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	facts, err := store.ExtractEvidenceFacts(t.Context(), ExtractEvidenceFactsInput{
		SourceID:  source.ID,
		SectionID: sections[0].ID,
	}, server.Client())
	if err != nil {
		t.Fatalf("ExtractEvidenceFacts() error = %v", err)
	}
	if len(facts) != 1 || !strings.Contains(facts[0].FactText, "tools=FastAPI") || facts[0].Confidence != "high" {
		t.Fatalf("facts = %+v", facts)
	}
}

func TestFallbackFactsExtractAtomsTechnologiesAndConfidence(t *testing.T) {
	section := SourceSection{
		Heading:     "Load testing",
		SectionType: "project",
		Content:     "CueMate | 2026\n- Added Locust load-test coverage for login, project listing, asset listing, and booking creation against staging-style workflows.",
	}
	facts := fallbackFactsFromSection(section)
	if len(facts) != 4 {
		t.Fatalf("facts len = %d, want 4: %+v", len(facts), facts)
	}
	parts := make([]string, 0, len(facts))
	for _, fact := range facts {
		parts = append(parts, fact.FactText)
	}
	combined := strings.Join(parts, "\n")
	if strings.Contains(combined, "Added Locust") ||
		!strings.Contains(combined, "tools=Locust") ||
		!strings.Contains(combined, "scope=login") ||
		!strings.Contains(combined, "environment=staging-style workflows") ||
		!strings.Contains(combined, "outcome=test coverage") {
		t.Fatalf("facts not atomized: %+v", facts)
	}
	if len(facts[0].Technologies) != 1 || facts[0].Technologies[0] != "Locust" {
		t.Fatalf("technologies = %+v, want Locust", facts[0].Technologies)
	}
	if facts[0].Confidence != "medium" || len(facts[0].RiskFlags) == 0 {
		t.Fatalf("confidence/risk = %q/%+v, want medium with staging risk", facts[0].Confidence, facts[0].RiskFlags)
	}
}

func TestInsertExtractedFactsRepairsParaphrasedEvidenceQuote(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:   "Resume",
		RawText: "PROJECTS\nBuilt features end to end across APIs, integrations, and workflow logic.",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}
	inserted, err := store.insertExtractedFacts(sections[0], []extractedFact{{
		FactText:      "actions=built; artifact=backend features",
		EvidenceQuote: "Hands-on experience designing core backend architecture and building features end to end, from system boundaries to APIs, integrations, and workflow logic.",
		Confidence:    "medium",
	}})
	if err != nil {
		t.Fatalf("insertExtractedFacts() error = %v", err)
	}
	if len(inserted) != 1 || inserted[0].EvidenceQuote != "Built features end to end across APIs, integrations, and workflow logic." {
		t.Fatalf("inserted = %+v", inserted)
	}
	if inserted[0].OriginHeading == "" || inserted[0].OriginType == "" || len(inserted[0].Context) == 0 {
		t.Fatalf("origin context missing: %+v", inserted[0])
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

func TestJobContextMigrationCreatesTables(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	for _, table := range []string{"job_descriptions", "job_requirements", "job_fact_matches", "tailored_bullet_drafts"} {
		var name string
		if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}

func TestJobDescriptionCascadeWorkflow(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		URL:     "https://example.test/job",
		RawText: "We need FastAPI experience.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	updated, err := store.UpdateJobDescription(UpdateJobDescriptionInput{
		ID:      job.ID,
		Company: "Acme Co",
		Title:   "Senior Backend Engineer",
		URL:     job.URL,
		RawText: job.RawText,
	})
	if err != nil {
		t.Fatalf("UpdateJobDescription() error = %v", err)
	}
	if updated.Company != "Acme Co" || updated.Title != "Senior Backend Engineer" {
		t.Fatalf("updated job = %+v", updated)
	}

	requirements, err := store.replaceJobRequirements(updated, []parsedJobRequirement{{
		Category:        "must_have",
		RequirementText: "FastAPI experience",
		Keywords:        []string{"FastAPI"},
		Priority:        "high",
		SourceQuote:     "FastAPI experience",
	}})
	if err != nil {
		t.Fatalf("replaceJobRequirements() error = %v", err)
	}
	section, facts := createFactsForJobTests(t, store)
	matches, err := store.replaceJobMatches(updated.ID, []parsedJobMatch{{
		RequirementID:  requirements[0].ID,
		FactID:         facts[0].ID,
		Score:          0.9,
		Rationale:      "Direct FastAPI evidence.",
		CoverageStatus: "strong",
	}}, requirements, []factPromptContext{{
		ID:             facts[0].ID,
		Status:         facts[0].Status,
		Confidence:     facts[0].Confidence,
		RiskFlags:      facts[0].RiskFlags,
		FactText:       facts[0].FactText,
		EvidenceQuote:  facts[0].EvidenceQuote,
		SectionHeading: section.Heading,
	}})
	if err != nil {
		t.Fatalf("replaceJobMatches() error = %v", err)
	}
	if matches[0].FactStatus != facts[0].Status {
		t.Fatalf("match fact status = %q, want %q", matches[0].FactStatus, facts[0].Status)
	}
	if _, err := store.replaceBulletDrafts(updated.ID, []parsedBulletDraft{{
		RequirementID: requirements[0].ID,
		FactIDs:       []int64{facts[0].ID},
		DraftText:     "Built FastAPI APIs for production planning workflows.",
		Rationale:     "Uses exact FastAPI evidence.",
	}}, requirements, []factPromptContext{{ID: facts[0].ID}}); err != nil {
		t.Fatalf("replaceBulletDrafts() error = %v", err)
	}

	if err := store.DeleteJobDescription(DeleteInput{ID: updated.ID}); err != nil {
		t.Fatalf("DeleteJobDescription() error = %v", err)
	}
	for table, column := range map[string]string{
		"job_requirements":       "job_id",
		"job_fact_matches":       "job_id",
		"tailored_bullet_drafts": "job_id",
	} {
		var count int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE `+column+` = ?`, updated.ID).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", table, count)
		}
	}
}

func TestParseJobRequirementsValidation(t *testing.T) {
	requirements, err := parseJobRequirements(`{"requirements":[{"category":"must_have","requirement_text":"FastAPI","keywords":["FastAPI"],"priority":"high","source_quote":"FastAPI"}]}`)
	if err != nil {
		t.Fatalf("parseJobRequirements() error = %v", err)
	}
	if len(requirements) != 1 || requirements[0].Category != "must_have" {
		t.Fatalf("requirements = %+v", requirements)
	}
	if _, err := parseJobRequirements(`{"requirements":[]}`); err == nil {
		t.Fatal("parseJobRequirements() empty error = nil")
	}
	if _, err := parseJobRequirements(`{"requirements":[{"requirement_text":"FastAPI"}]}`); err == nil {
		t.Fatal("parseJobRequirements() missing quote error = nil")
	}
}

func TestParseJobRequirementsFiltersBoilerplate(t *testing.T) {
	requirements, err := parseJobRequirements(`{"requirements":[
		{"category":"must_have","requirement_text":"Slater and Gordon Lawyers logo","keywords":["Slater","Gordon","Lawyers","logo"],"priority":"high","source_quote":"Slater and Gordon Lawyers logo"},
		{"category":"must_have","requirement_text":"Software Engineer","keywords":["Software","Engineer"],"priority":"high","source_quote":"Software Engineer"},
		{"category":"responsibility","requirement_text":"Melbourne, Victoria, Australia · 55 minutes ago · 4 people clicked apply","keywords":["Melbourne","apply"],"priority":"high","source_quote":"Melbourne, Victoria, Australia · 55 minutes ago · 4 people clicked apply"},
		{"category":"must_have","requirement_text":"Promoted by hirer · Responses managed off LinkedIn","keywords":["Promoted","LinkedIn"],"priority":"high","source_quote":"Promoted by hirer · Responses managed off LinkedIn"},
		{"category":"must_have","requirement_text":"Your profile matches some required qualifications","keywords":["profile","matches"],"priority":"high","source_quote":"Your profile matches some required qualifications"},
		{"category":"must_have","requirement_text":"Retry Premium for A$0","keywords":["Premium"],"priority":"medium","source_quote":"Retry Premium for A$0"},
		{"category":"must_have","requirement_text":"About the job","keywords":["About"],"priority":"medium","source_quote":"About the job"},
		{"category":"must_have","requirement_text":"What are we looking for?","keywords":["looking"],"priority":"medium","source_quote":"What are we looking for?"},
		{"category":"seniority","requirement_text":"The role is a 12-month max term contract for a Software Engineer.","keywords":["12-month","Software Engineer"],"priority":"high","source_quote":"Software Engineer | 12 Month Max Term"},
		{"category":"domain","requirement_text":"Role is within a leading personal injury and class actions law firm.","keywords":["class actions","law firm"],"priority":"medium","source_quote":"Slater and Gordon Lawyers are a leading personal injury and class actions law firm"},
		{"category":"must_have","requirement_text":"Strong experience building scalable distributed cloud applications.","keywords":["cloud applications","distributed","scalable"],"priority":"high","source_quote":"Strong experience building and supporting scalable, distributed cloud applications."}
	]}`)
	if err != nil {
		t.Fatalf("parseJobRequirements() error = %v", err)
	}
	if len(requirements) != 1 || !strings.Contains(requirements[0].RequirementText, "cloud") {
		t.Fatalf("requirements = %+v", requirements)
	}
}

func TestJobLLMWorkflowUsesAllFactStatusesAndDraftReview(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	if _, err := store.SaveSettings(SaveSettingsInput{Provider: "openrouter", Model: "deepseek/test"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		RawText: "Must have FastAPI experience. Nice to have Linux automation.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	_, facts := createFactsForJobTests(t, store)

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		content := request.Messages[len(request.Messages)-1].Content
		switch call {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"requirements\":[{\"category\":\"must_have\",\"requirement_text\":\"FastAPI experience\",\"keywords\":[\"FastAPI\"],\"priority\":\"high\",\"source_quote\":\"FastAPI experience\"}]}"}}]}`))
		case 2:
			if !strings.Contains(content, facts[0].Status) || !strings.Contains(content, facts[0].FactText) {
				t.Fatalf("match prompt missing relevant approved fact: %s", content)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"matches\":[{\"requirement_id\":1,\"fact_id\":1,\"score\":0.9,\"rationale\":\"Direct evidence\",\"coverage_status\":\"strong\"},{\"requirement_id\":1,\"fact_id\":1,\"score\":0,\"rationale\":\"No evidence\",\"coverage_status\":\"gap\"},{\"requirement_id\":1,\"fact_id\":999,\"score\":1,\"rationale\":\"Invalid\",\"coverage_status\":\"strong\"}]}"}}]}`))
		case 3:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"drafts\":[{\"requirement_id\":1,\"fact_ids\":[1,999],\"draft_text\":\"Built FastAPI APIs for planning workflows.\",\"rationale\":\"Direct evidence\",\"risk_flags\":[]}]}"}}]}`))
		default:
			t.Fatalf("unexpected LLM call %d", call)
		}
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	requirements, err := store.ParseJobDescription(t.Context(), job.ID, server.Client())
	if err != nil {
		t.Fatalf("ParseJobDescription() error = %v", err)
	}
	matches, err := store.BuildJobMatchMap(t.Context(), job.ID, server.Client())
	if err != nil {
		t.Fatalf("BuildJobMatchMap() error = %v", err)
	}
	if len(matches) != 1 || matches[0].RequirementID != requirements[0].ID || matches[0].FactID != facts[0].ID {
		t.Fatalf("matches = %+v", matches)
	}
	drafts, err := store.GenerateTailoredBulletDrafts(t.Context(), job.ID, server.Client())
	if err != nil {
		t.Fatalf("GenerateTailoredBulletDrafts() error = %v", err)
	}
	if len(drafts) != 1 || len(drafts[0].FactIDs) != 1 || drafts[0].FactIDs[0] != facts[0].ID {
		t.Fatalf("drafts = %+v", drafts)
	}
	updated, err := store.UpdateTailoredBulletDraft(UpdateTailoredBulletDraftInput{
		ID:        drafts[0].ID,
		DraftText: "Built FastAPI APIs tailored to Acme planning workflows.",
		Rationale: "Edited by user.",
		Status:    "accepted",
		RiskFlags: []string{"needs_review_source"},
	})
	if err != nil {
		t.Fatalf("UpdateTailoredBulletDraft() error = %v", err)
	}
	if updated.Status != "accepted" || !strings.Contains(updated.DraftText, "Acme") || len(updated.RiskFlags) != 1 {
		t.Fatalf("updated draft = %+v", updated)
	}
}

func TestBulletDraftsRespectOriginBudgets(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		RawText: "Build APIs.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	requirements, err := store.replaceJobRequirements(job, []parsedJobRequirement{{
		Category:        "responsibility",
		RequirementText: "Build APIs.",
		Keywords:        []string{"API"},
		Priority:        "high",
		SourceQuote:     "Build APIs.",
	}})
	if err != nil {
		t.Fatalf("replaceJobRequirements() error = %v", err)
	}
	facts := []factPromptContext{}
	drafts := []parsedBulletDraft{}
	for i := 1; i <= 6; i++ {
		id := int64(i)
		facts = append(facts, factPromptContext{ID: id, Status: "approved", SectionHeading: "Acme | Backend Engineer", SectionType: "experience"})
		drafts = append(drafts, parsedBulletDraft{RequirementID: requirements[0].ID, FactIDs: []int64{id}, DraftText: fmt.Sprintf("Built backend API workflow %d with reliable integration behavior.", i)})
	}
	for i := 7; i <= 9; i++ {
		id := int64(i)
		facts = append(facts, factPromptContext{ID: id, Status: "approved", SectionHeading: "CueMate", SectionType: "project"})
		drafts = append(drafts, parsedBulletDraft{RequirementID: requirements[0].ID, FactIDs: []int64{id}, DraftText: fmt.Sprintf("Built project API workflow %d with reliable integration behavior.", i)})
	}
	inserted, err := store.replaceBulletDrafts(job.ID, drafts, requirements, facts)
	if err != nil {
		t.Fatalf("replaceBulletDrafts() error = %v", err)
	}
	if len(inserted) != 7 {
		t.Fatalf("drafts len = %d, want 7: %+v", len(inserted), inserted)
	}
}

func TestCreateJobDescriptionInfersDetailsFromRawText(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		RawText: "Sitespace\nBackend Engineer\nResponsibilities\nBuild APIs and planning workflows.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	if job.Company != "Sitespace" || job.Title != "Backend Engineer" {
		t.Fatalf("job details = %q/%q, want Sitespace/Backend Engineer", job.Company, job.Title)
	}
}

func TestBuildJobMatchMapFallsBackOnEmptyLLMJSON(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		RawText: "Must have FastAPI experience.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	requirements, err := store.replaceJobRequirements(job, []parsedJobRequirement{{
		Category:        "must_have",
		RequirementText: "FastAPI experience",
		Keywords:        []string{"FastAPI"},
		Priority:        "high",
		SourceQuote:     "FastAPI experience",
	}})
	if err != nil {
		t.Fatalf("replaceJobRequirements() error = %v", err)
	}
	_, facts := createFactsForJobTests(t, store)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	matches, err := store.BuildJobMatchMap(t.Context(), job.ID, server.Client())
	if err != nil {
		t.Fatalf("BuildJobMatchMap() error = %v", err)
	}
	foundApprovedFastAPI := false
	for _, match := range matches {
		if match.RequirementID == requirements[0].ID && match.FactID == facts[0].ID {
			foundApprovedFastAPI = true
		}
	}
	if !foundApprovedFastAPI {
		t.Fatalf("matches = %+v", matches)
	}
	if !strings.Contains(matches[0].Rationale, "Local keyword overlap") {
		t.Fatalf("match rationale = %q, want local fallback", matches[0].Rationale)
	}
}

func TestParseJobDescriptionFallsBackOnEmptyLLMJSON(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		RawText: "Strong experience building scalable distributed cloud applications.\nThe role is a 12-month max term contract.\nHands-on experience with serverless and event-driven architectures.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	requirements, err := store.ParseJobDescription(t.Context(), job.ID, server.Client())
	if err != nil {
		t.Fatalf("ParseJobDescription() error = %v", err)
	}
	combined := ""
	for _, requirement := range requirements {
		combined += requirement.RequirementText + "\n"
	}
	if !strings.Contains(combined, "cloud applications") || !strings.Contains(combined, "serverless") || strings.Contains(combined, "12-month") {
		t.Fatalf("requirements = %+v", requirements)
	}
}

func TestParseJobDescriptionFallsBackOnLLMTimeout(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		RawText: "Strong experience building scalable distributed cloud applications.\nHands-on experience with serverless and event-driven architectures.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{}"}}]}`))
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	requirements, err := store.ParseJobDescription(t.Context(), job.ID, &http.Client{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("ParseJobDescription() error = %v", err)
	}
	if len(requirements) == 0 {
		t.Fatalf("requirements empty after timeout fallback")
	}
}

func TestBuildJobMatchMapFallsBackOnLLMTimeout(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		RawText: "Must have FastAPI experience.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	if _, err := store.replaceJobRequirements(job, []parsedJobRequirement{{
		Category:        "must_have",
		RequirementText: "FastAPI experience",
		Keywords:        []string{"FastAPI"},
		Priority:        "high",
		SourceQuote:     "FastAPI experience",
	}}); err != nil {
		t.Fatalf("replaceJobRequirements() error = %v", err)
	}
	_, facts := createFactsForJobTests(t, store)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{}"}}]}`))
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	matches, err := store.BuildJobMatchMap(t.Context(), job.ID, &http.Client{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("BuildJobMatchMap() error = %v", err)
	}
	if len(matches) == 0 || matches[0].FactID != facts[0].ID {
		t.Fatalf("matches = %+v", matches)
	}
}

func TestRulesAnalysisFitAndStrategyWorkflow(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	rules, err := store.ListPromptRules()
	if err != nil {
		t.Fatalf("ListPromptRules() error = %v", err)
	}
	if len(rules) == 0 {
		t.Fatalf("prompt rules not seeded")
	}
	updated, err := store.UpdatePromptRule(UpdatePromptRuleInput{
		ID:      rules[0].ID,
		Content: rules[0].Content + " Keep phrasing concrete.",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdatePromptRule() error = %v", err)
	}
	if updated.Enabled || updated.Version <= rules[0].Version {
		t.Fatalf("updated rule = %+v", updated)
	}
	sources, err := store.ListPromptResearchSources()
	if err != nil {
		t.Fatalf("ListPromptResearchSources() error = %v", err)
	}
	if len(sources) == 0 {
		t.Fatalf("prompt research sources not seeded")
	}

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Cloud Backend Engineer",
		RawText: "Strong experience building scalable distributed cloud applications.\nHands-on experience with serverless and event-driven architectures.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	requirements, err := store.replaceJobRequirements(job, []parsedJobRequirement{
		{Category: "must_have", RequirementText: "Strong experience building scalable distributed cloud applications.", Keywords: []string{"cloud", "distributed", "scalable"}, Priority: "high", SourceQuote: "Strong experience building scalable distributed cloud applications."},
		{Category: "responsibility", RequirementText: "Hands-on experience with serverless and event-driven architectures.", Keywords: []string{"serverless", "event-driven"}, Priority: "high", SourceQuote: "Hands-on experience with serverless and event-driven architectures."},
	})
	if err != nil {
		t.Fatalf("replaceJobRequirements() error = %v", err)
	}
	analysis, err := store.AnalyzeJobDescription(job.ID)
	if err != nil {
		t.Fatalf("AnalyzeJobDescription() error = %v", err)
	}
	if len(analysis.TopPainPoints) == 0 || analysis.RoleArchetype == "" {
		t.Fatalf("analysis = %+v", analysis)
	}
	_, facts := createFactsForJobTests(t, store)
	if _, err := store.replaceJobMatches(job.ID, []parsedJobMatch{{
		RequirementID:  requirements[0].ID,
		FactID:         facts[0].ID,
		Score:          0.76,
		Rationale:      "Backend API evidence transfers partially to cloud app delivery.",
		CoverageStatus: "strong",
	}}, requirements, []factPromptContext{{ID: facts[0].ID, Status: facts[0].Status}}); err != nil {
		t.Fatalf("replaceJobMatches() error = %v", err)
	}
	fit, err := store.GenerateFitAnalysis(job.ID)
	if err != nil {
		t.Fatalf("GenerateFitAnalysis() error = %v", err)
	}
	if fit.OverallScore <= 0 || len(fit.Analysis) != len(requirements) {
		t.Fatalf("fit = %+v", fit)
	}
	strategy, err := store.GenerateApplicationStrategy(job.ID)
	if err != nil {
		t.Fatalf("GenerateApplicationStrategy() error = %v", err)
	}
	if len(strategy.ApprovedFactIDs) == 0 || len(strategy.DoNotOverclaim) == 0 {
		t.Fatalf("strategy = %+v", strategy)
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

func createFactsForJobTests(t *testing.T, store *Store) (SourceSection, []EvidenceFact) {
	t.Helper()
	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:   "Resume",
		RawText: "PROJECTS\nBuilt FastAPI APIs.\nBuilt Linux automation.",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}
	facts, err := store.insertExtractedFacts(sections[0], []extractedFact{
		{
			FactText:      "Built FastAPI APIs.",
			EvidenceQuote: "Built FastAPI APIs.",
			Confidence:    "high",
		},
		{
			FactText:      "Built Linux automation.",
			EvidenceQuote: "Built Linux automation.",
			Confidence:    "medium",
			RiskFlags:     []string{"ambiguous_scope"},
		},
	})
	if err != nil {
		t.Fatalf("insertExtractedFacts() error = %v", err)
	}
	rejected, err := store.UpdateEvidenceFactReview(UpdateEvidenceFactReviewInput{
		ID:            facts[1].ID,
		FactText:      facts[1].FactText,
		EvidenceQuote: facts[1].EvidenceQuote,
		Confidence:    facts[1].Confidence,
		RiskFlags:     facts[1].RiskFlags,
		Status:        "rejected",
		ReviewNote:    "Not relevant.",
	})
	if err != nil {
		t.Fatalf("UpdateEvidenceFactReview() error = %v", err)
	}
	facts[1] = rejected
	return sections[0], facts
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
