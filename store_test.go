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

func TestMigrationAddsAtomBankAndDraftSelectionColumns(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	for _, column := range []string{"actions_json", "capabilities_json", "objects_json", "domains_json", "artifacts_json", "scope_json", "metrics_json", "outcomes_json", "profile_context_json", "evidence_strength"} {
		if !tableHasColumn(t, store, "candidate_claims", column) {
			t.Fatalf("candidate_claims missing %s", column)
		}
	}
	for _, column := range []string{"trust_tier"} {
		if !tableHasColumn(t, store, "candidate_sources", column) {
			t.Fatalf("candidate_sources missing %s", column)
		}
	}
	for _, column := range []string{"similarity_key", "similarity_score", "duplicate_of_id"} {
		if !tableHasColumn(t, store, "evidence_facts", column) {
			t.Fatalf("evidence_facts missing %s", column)
		}
	}
	for _, column := range []string{"similarity_key", "similarity_score", "duplicate_of_id"} {
		if !tableHasColumn(t, store, "candidate_claims", column) {
			t.Fatalf("candidate_claims missing %s", column)
		}
	}
	for _, column := range []string{"claim_ids_json", "origin_heading", "origin_type", "selection_score", "selected_for_resume", "value_theme", "display_order"} {
		if !tableHasColumn(t, store, "tailored_bullet_drafts", column) {
			t.Fatalf("tailored_bullet_drafts missing %s", column)
		}
	}
	for _, column := range []string{"entity_type", "entity_id", "provider", "model", "input_hash", "dimensions", "vector_json"} {
		if !tableHasColumn(t, store, "semantic_embeddings", column) {
			t.Fatalf("semantic_embeddings missing %s", column)
		}
	}
	for _, column := range []string{"source_id", "status", "started_at", "finished_at", "error", "facts_created", "claims_created"} {
		if !tableHasColumn(t, store, "context_agent_runs", column) {
			t.Fatalf("context_agent_runs missing %s", column)
		}
	}
	for _, column := range []string{"run_id", "stage", "status", "message", "created_at"} {
		if !tableHasColumn(t, store, "context_agent_steps", column) {
			t.Fatalf("context_agent_steps missing %s", column)
		}
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
			t.Logf("method = %s, want POST", r.Method)
			http.Error(w, "method", http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Logf("authorization header not set")
			http.Error(w, "auth", http.StatusInternalServerError)
			return
		}
		if r.Header.Get("X-OpenRouter-Title") != "JD Tailor" {
			t.Logf("OpenRouter title header not set")
			http.Error(w, "title", http.StatusInternalServerError)
			return
		}
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Logf("decode request: %v", err)
			http.Error(w, "decode", http.StatusInternalServerError)
			return
		}
		if request.Model != "deepseek/test" || len(request.Messages) == 0 {
			t.Logf("bad request: %+v", request)
			http.Error(w, "bad request", http.StatusInternalServerError)
			return
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
			t.Logf("decode request: %v", err)
			http.Error(w, "decode", http.StatusInternalServerError)
			return
		}
		if request.ResponseFormat == nil || request.ResponseFormat.Type != "json_object" {
			t.Logf("response format = %+v, want json_object", request.ResponseFormat)
			http.Error(w, "format", http.StatusInternalServerError)
			return
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

func TestEmbeddingClientCachesSameProviderVectors(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	if _, err := store.SaveSettings(SaveSettingsInput{Provider: "openrouter", EmbeddingModel: "openai/text-embedding-3-small"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/embeddings" {
			t.Logf("path = %s, want /api/v1/embeddings", r.URL.Path)
			http.Error(w, "path", http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Logf("authorization header missing")
			http.Error(w, "auth", http.StatusInternalServerError)
			return
		}
		var request embeddingsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Logf("decode request: %v", err)
			http.Error(w, "decode", http.StatusInternalServerError)
			return
		}
		if request.Model != "openai/text-embedding-3-small" || request.Input != "same text" {
			t.Logf("request = %+v", request)
			http.Error(w, "bad request", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"openai/text-embedding-3-small"}`))
	}))
	defer server.Close()
	restore := openRouterEmbeddingsURLForTest(server.URL + "/api/v1/embeddings")
	defer restore()

	first, err := store.embeddingForEntity(t.Context(), server.Client(), "test", 1, "same text")
	if err != nil {
		t.Fatalf("embeddingForEntity() first error = %v", err)
	}
	second, err := store.embeddingForEntity(t.Context(), server.Client(), "test", 1, "same text")
	if err != nil {
		t.Fatalf("embeddingForEntity() second error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("embedding calls = %d, want cache hit after first call", calls)
	}
	if len(first) != 3 || len(second) != 3 || first[0] != second[0] {
		t.Fatalf("vectors = %+v %+v", first, second)
	}
}

func TestAutoSelectFallsBackWhenEmbeddingsFail(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	if _, err := store.SaveSettings(SaveSettingsInput{Provider: "openrouter", EmbeddingModel: "openai/text-embedding-3-small"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"embedding unavailable"}}`, http.StatusBadGateway)
	}))
	defer server.Close()
	restore := openRouterEmbeddingsURLForTest(server.URL)
	defer restore()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{Company: "Acme", Title: "Backend Engineer", RawText: "Build APIs."})
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
	facts := []factPromptContext{
		{ID: 1, Status: "approved", SectionHeading: "Acme", SectionType: "experience"},
		{ID: 2, Status: "approved", SectionHeading: "Acme", SectionType: "experience"},
	}
	claims := testClaimsForPromptFacts(facts)
	drafts, err := store.replaceBulletDrafts(job.ID, []parsedBulletDraft{
		{RequirementID: requirements[0].ID, ClaimIDs: []int64{1}, FactIDs: []int64{1}, ValueTheme: "product_platform_delivery", DraftText: "Built backend APIs for planning workflows, turning requirements into usable platform capabilities."},
		{RequirementID: requirements[0].ID, ClaimIDs: []int64{2}, FactIDs: []int64{2}, ValueTheme: "technical_design", DraftText: "Designed backend integration flows with clear service boundaries and reliable API behavior."},
	}, requirements, facts, claims)
	if err != nil {
		t.Fatalf("replaceBulletDrafts() error = %v", err)
	}
	if len(drafts) != 2 {
		t.Fatalf("drafts = %+v", drafts)
	}
	selected, err := store.AutoSelectResumeBullets(job.ID)
	if err != nil {
		t.Fatalf("AutoSelectResumeBullets() error = %v", err)
	}
	selectedCount := 0
	for _, draft := range selected {
		if draft.SelectedForResume {
			selectedCount++
		}
	}
	if selectedCount == 0 {
		t.Fatalf("selected = %+v, want fallback selection", selected)
	}
	events, err := store.ListBulletGenerationEvents(job.ID)
	if err != nil {
		t.Fatalf("ListBulletGenerationEvents() error = %v", err)
	}
	foundFallback := false
	for _, event := range events {
		if event.Stage == "embedding_fallback" {
			foundFallback = true
		}
	}
	if !foundFallback {
		t.Fatalf("events = %+v, want embedding fallback diagnostic", events)
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
			FullName: "Jane Q. Public",
			Email:    "jane.doe@example.com",
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
	if saved.Contact.FullName != "Jane Q. Public" {
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

func TestDetectSourceSectionsPreservesClaimsWhenUnchanged(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:      "Resume",
		SourceType: "current_resume",
		RawText: `PROJECTS
- Built FastAPI APIs for planning workflows.
- Added PostgreSQL persistence for project records.`,
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() first error = %v", err)
	}
	facts, err := store.insertExtractedFacts(sections[0], []extractedFact{{
		FactText:      "Built FastAPI APIs for planning workflows.",
		EvidenceQuote: "Built FastAPI APIs for planning workflows.",
		Technologies:  []string{"FastAPI"},
		Confidence:    "high",
		Context:       []string{"planning workflows"},
	}})
	if err != nil {
		t.Fatalf("insertExtractedFacts() error = %v", err)
	}
	createdClaims, err := store.replaceCandidateClaims([]parsedCandidateClaim{{
		ClaimText:     "FastAPI APIs for planning workflows",
		ClaimType:     "project",
		SourceFactIDs: []int64{facts[0].ID},
		Actions:       []string{"built"},
		Artifacts:     []string{"FastAPI APIs"},
		Domains:       []string{"planning workflows"},
	}}, factsToPromptContext(facts))
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}
	if len(createdClaims) != 1 {
		t.Fatalf("created claims len = %d, want 1", len(createdClaims))
	}

	redetected, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() second error = %v", err)
	}
	if len(redetected) != len(sections) || redetected[0].ID != sections[0].ID {
		t.Fatalf("redetected sections = %+v, want existing sections %+v", redetected, sections)
	}
	claims, err := store.ListCandidateClaims("all")
	if err != nil {
		t.Fatalf("ListCandidateClaims() error = %v", err)
	}
	if len(claims) != 1 || claims[0].ID != createdClaims[0].ID {
		t.Fatalf("claims after redetect = %+v, want original claim", claims)
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
\textbf{\Huge \scshape John Doe} \\
\href{mailto:john.doe@example.com}{john.doe@example.com} $|$ Melbourne, VIC
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
	for _, required := range []string{"John Doe", "john.doe@example.com", "Experience", "Sitespace", "Fullstack Engineer", "Built and shipped the FastAPI/PostgreSQL backend"} {
		if !strings.Contains(cleaned, required) {
			t.Fatalf("cleaned text missing %q: %s", required, cleaned)
		}
	}
}

func TestDetectSectionsUsesLatexResumeSections(t *testing.T) {
	raw := `\begin{document}
\begin{center}
\textbf{\Huge \scshape John Doe}
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
\textbf{\Huge \scshape John Doe} \\
\href{mailto:john.doe@example.com}{john.doe@example.com} $|$ Melbourne, VIC
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
	if profile.Contact.FullName != "John Doe" || profile.Contact.Email != "john.doe@example.com" {
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
	_, err = store.replaceCandidateClaims([]parsedCandidateClaim{{
		Label:            "FastAPI backend APIs",
		ClaimType:        "experience",
		SourceFactIDs:    []int64{facts[0].ID},
		Actions:          []string{"built"},
		Capabilities:     []string{"API development"},
		Technologies:     []string{"FastAPI"},
		Scope:            []string{"planning workflows"},
		EvidenceStrength: "direct",
		Strength:         "strong",
		AllowedUse:       []string{"experience_bullet"},
		AllowedContexts:  []string{"backend engineering"},
	}}, []factPromptContext{{
		ID:             facts[0].ID,
		Status:         factStatusApproved,
		Confidence:     "high",
		FactText:       facts[0].FactText,
		EvidenceQuote:  facts[0].EvidenceQuote,
		Technologies:   facts[0].Technologies,
		SectionHeading: facts[0].OriginHeading,
		SectionType:    facts[0].OriginType,
	}})
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}
	if _, err := store.replaceJobMatches(job.ID, []parsedJobMatch{{
		RequirementID:  requirements[0].ID,
		FactID:         facts[0].ID,
		Score:          0.8,
		CoverageStatus: "strong",
	}}, requirements, []factPromptContext{{ID: facts[0].ID, Status: facts[0].Status}}); err != nil {
		t.Fatalf("replaceJobMatches() error = %v", err)
	}
	testClaims := testClaimsForFacts(facts)
	if _, err := store.replaceBulletDrafts(job.ID, []parsedBulletDraft{{
		RequirementID: requirements[0].ID,
		ClaimIDs:      []int64{testClaims[0].ID},
		FactIDs:       []int64{facts[0].ID},
		DraftText:     "Built API workflows with reliable backend delivery.",
	}}, requirements, []factPromptContext{{ID: facts[0].ID, SectionHeading: "Sitespace", SectionType: "experience"}}, testClaims); err != nil {
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

func TestParseExtractedFactsAcceptsFlexibleAtomLists(t *testing.T) {
	facts, err := parseExtractedFacts(`{"facts":[{"fact_text":"actions=built; artifact=APIs; tools=FastAPI","evidence_quote":"Built FastAPI APIs.","technologies":"FastAPI, PostgreSQL","confidence":"high","risk_flags":"unclear_metric","context":{"organization":"Acme","role":"Backend Engineer"}}]}`)
	if err != nil {
		t.Fatalf("parseExtractedFacts() error = %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("facts len = %d", len(facts))
	}
	if !listContains(facts[0].Technologies, "FastAPI") || !listContains(facts[0].Technologies, "PostgreSQL") {
		t.Fatalf("technologies = %+v, want split list", facts[0].Technologies)
	}
	if !listContains(facts[0].RiskFlags, "unclear_metric") ||
		!listContains(facts[0].Context, "organization=Acme") ||
		!listContains(facts[0].Context, "role=Backend Engineer") {
		t.Fatalf("flexible fields not normalized: %+v", facts[0])
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

func TestTrustedExtendedResumeAutoApprovesAndSkipsDuplicateFacts(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		SourceType: "extended_resume",
		TrustTier:  "trusted_ai_summary",
		Title:      "Extended resume",
		RawText:    "PROJECTS\nBuilt FastAPI APIs.\nBuilt FastAPI APIs.",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}
	inserted, err := store.insertExtractedFacts(sections[0], []extractedFact{
		{FactText: "actions=built; artifact=FastAPI APIs; tools=FastAPI", EvidenceQuote: "Built FastAPI APIs.", Technologies: []string{"FastAPI"}, Confidence: "medium"},
		{FactText: "actions=implemented; artifact=FastAPI APIs; tools=FastAPI", EvidenceQuote: "Built FastAPI APIs.", Technologies: []string{"FastAPI"}, Confidence: "medium"},
	})
	if err != nil {
		t.Fatalf("insertExtractedFacts() error = %v", err)
	}
	if len(inserted) != 2 {
		t.Fatalf("inserted len = %d", len(inserted))
	}
	if inserted[0].Status != factStatusApproved || !inserted[0].AutoApproved {
		t.Fatalf("first fact = %+v, want auto-approved", inserted[0])
	}
	if inserted[1].Status != factStatusRejected || inserted[1].DuplicateOfID != inserted[0].ID || !listContains(inserted[1].RiskFlags, "duplicate_fact") {
		t.Fatalf("duplicate fact = %+v, want rejected duplicate of %d", inserted[1], inserted[0].ID)
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

func TestContextAgentReusesActiveRun(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:   "Resume",
		RawText: "PROJECTS\nBuilt FastAPI APIs.",
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	first, err := store.StartContextAgent(source.ID)
	if err != nil {
		t.Fatalf("StartContextAgent() first error = %v", err)
	}
	second, err := store.StartContextAgent(source.ID)
	if err != nil {
		t.Fatalf("StartContextAgent() second error = %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("second run ID = %d, want active run %d", second.ID, first.ID)
	}
	steps, err := store.ListContextAgentSteps(first.ID)
	if err != nil {
		t.Fatalf("ListContextAgentSteps() error = %v", err)
	}
	if len(steps) != 1 || steps[0].Stage != "queued" {
		t.Fatalf("steps = %+v, want one queued step", steps)
	}
}

func TestAppRevivesQueuedContextAgentRun(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title: "Full extended resume",
		RawText: `PROJECTS
- Built and shipped FastAPI backend APIs for planning workflows.
- Added PostgreSQL persistence for project records.

TECHNICAL SKILLS
FastAPI, PostgreSQL, React`,
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	run, err := store.StartContextAgent(source.ID)
	if err != nil {
		t.Fatalf("StartContextAgent() error = %v", err)
	}

	app := NewApp()
	app.ctx = context.Background()
	app.store = store
	if _, err := app.ListContextAgentRuns(0); err != nil {
		t.Fatalf("ListContextAgentRuns() error = %v", err)
	}

	var finished ContextAgentRun
	for attempt := 0; attempt < 40; attempt++ {
		finished, err = store.GetContextAgentRun(run.ID)
		if err != nil {
			t.Fatalf("GetContextAgentRun() error = %v", err)
		}
		if finished.Status == contextAgentStatusComplete {
			break
		}
		if finished.Status != contextAgentStatusRunning {
			steps, _ := store.ListContextAgentSteps(run.ID)
			t.Fatalf("revived run status = %q, want running or complete; steps = %+v", finished.Status, steps)
		}
		time.Sleep(25 * time.Millisecond)
	}
	if finished.Status != contextAgentStatusComplete {
		steps, _ := store.ListContextAgentSteps(run.ID)
		t.Fatalf("revived run status = %q, want complete; steps = %+v", finished.Status, steps)
	}
	steps, err := store.ListContextAgentSteps(run.ID)
	if err != nil {
		t.Fatalf("ListContextAgentSteps() error = %v", err)
	}
	if len(steps) < 2 || steps[1].Stage != "source_preprocess" {
		t.Fatalf("steps = %+v, want worker to advance past queued", steps)
	}
}

func TestContextAgentRecoveryDoesNotRegenerateAfterClaimStep(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		Title:      "Resume",
		SourceType: "current_resume",
		RawText: `PROJECTS
- Built FastAPI APIs for planning workflows.
- Added PostgreSQL persistence for project records.`,
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	sections, err := store.DetectSourceSections(source.ID)
	if err != nil {
		t.Fatalf("DetectSourceSections() error = %v", err)
	}
	facts, err := store.insertExtractedFacts(sections[0], []extractedFact{{
		FactText:      "Built FastAPI APIs for planning workflows.",
		EvidenceQuote: "Built FastAPI APIs for planning workflows.",
		Technologies:  []string{"FastAPI"},
		Confidence:    "high",
		Context:       []string{"planning workflows"},
	}})
	if err != nil {
		t.Fatalf("insertExtractedFacts() error = %v", err)
	}
	createdClaims, err := store.replaceCandidateClaims([]parsedCandidateClaim{{
		ClaimText:     "FastAPI APIs for planning workflows",
		ClaimType:     "project",
		SourceFactIDs: []int64{facts[0].ID},
		Actions:       []string{"built"},
		Artifacts:     []string{"FastAPI APIs"},
		Domains:       []string{"planning workflows"},
	}}, factsToPromptContext(facts))
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}
	if len(createdClaims) != 1 {
		t.Fatalf("created claims len = %d, want 1", len(createdClaims))
	}
	run, err := store.StartContextAgent(source.ID)
	if err != nil {
		t.Fatalf("StartContextAgent() error = %v", err)
	}
	if err := store.recordContextAgentStep(run.ID, "claim_generate", "ok", "1 claim"); err != nil {
		t.Fatalf("recordContextAgentStep() error = %v", err)
	}

	finished, err := store.RunContextAgent(t.Context(), run.ID, nil)
	if err != nil {
		t.Fatalf("RunContextAgent() error = %v", err)
	}
	if finished.Status != contextAgentStatusComplete || finished.FactsCreated != 1 || finished.ClaimsCreated != 1 {
		t.Fatalf("finished = %+v, want recovered complete counts", finished)
	}
	claims, err := store.ListCandidateClaims("all")
	if err != nil {
		t.Fatalf("ListCandidateClaims() error = %v", err)
	}
	if len(claims) != 1 || claims[0].ID != createdClaims[0].ID {
		t.Fatalf("claims after recovery = %+v, want original claim", claims)
	}
	steps, err := store.ListContextAgentSteps(run.ID)
	if err != nil {
		t.Fatalf("ListContextAgentSteps() error = %v", err)
	}
	for _, step := range steps {
		if step.Stage == "fact_extract" {
			t.Fatalf("steps = %+v, recovery should not rerun fact extraction", steps)
		}
	}
}

func TestContextAgentBuildsCompactResumeContext(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		SourceType: "current_resume",
		TrustTier:  "verified",
		Title:      "Resume",
		RawText: `Jane Q. Public
jane.doe@example.com

PROJECTS
- Built and shipped FastAPI/PostgreSQL backend APIs for planning workflows.
- Added RBAC and audit logging for construction document workflows.

TECHNICAL SKILLS
FastAPI, PostgreSQL, React, TypeScript`,
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	run, err := store.StartContextAgent(source.ID)
	if err != nil {
		t.Fatalf("StartContextAgent() error = %v", err)
	}
	finished, err := store.RunContextAgent(t.Context(), run.ID, nil)
	if err != nil {
		t.Fatalf("RunContextAgent() error = %v", err)
	}
	if finished.Status != contextAgentStatusComplete {
		t.Fatalf("finished status = %q, want complete: %+v", finished.Status, finished)
	}
	if finished.FactsCreated == 0 || finished.ClaimsCreated == 0 {
		t.Fatalf("finished counts = facts %d claims %d", finished.FactsCreated, finished.ClaimsCreated)
	}

	steps, err := store.ListContextAgentSteps(run.ID)
	if err != nil {
		t.Fatalf("ListContextAgentSteps() error = %v", err)
	}
	stepStages := make(map[string]bool)
	for _, step := range steps {
		stepStages[step.Stage] = true
	}
	for _, stage := range []string{"queued", "source_preprocess", "section_detect", "fact_extract", "fact_compact", "profile_draft", "claim_generate", "claim_compact", "dedupe", "done"} {
		if !stepStages[stage] {
			t.Fatalf("steps missing %s: %+v", stage, steps)
		}
	}

	facts, err := store.ListEvidenceFacts("all")
	if err != nil {
		t.Fatalf("ListEvidenceFacts() error = %v", err)
	}
	claims, err := store.ListCandidateClaims("all")
	if err != nil {
		t.Fatalf("ListCandidateClaims() error = %v", err)
	}
	if len(facts) == 0 || len(claims) == 0 {
		t.Fatalf("facts = %d claims = %d, want generated context", len(facts), len(claims))
	}
	profile, err := store.GetCandidateProfile()
	if err != nil {
		t.Fatalf("GetCandidateProfile() error = %v", err)
	}
	if profile.Contact.Email != "jane.doe@example.com" || profile.Contact.Verified {
		t.Fatalf("profile contact = %+v, want draft unverified contact", profile.Contact)
	}

	contextBank, err := store.BuildResumeContext(source.ID)
	if err != nil {
		t.Fatalf("BuildResumeContext() error = %v", err)
	}
	if len(contextBank.Origins) == 0 || len(contextBank.Origins[0].Facts) == 0 || len(contextBank.Origins[0].Claims) == 0 {
		t.Fatalf("contextBank = %+v", contextBank)
	}
	for _, origin := range contextBank.Origins {
		for _, fact := range origin.Facts {
			if strings.Contains(fact.Atoms, ". ") {
				t.Fatalf("fact atoms look like prose, want compact atoms: %q", fact.Atoms)
			}
			if strings.TrimSpace(fact.Atoms) == "" {
				t.Fatalf("empty fact atoms: %+v", fact)
			}
		}
	}
}

func TestContextAgentSkipsEmptySectionsAndPublishesProgress(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	source, err := store.CreateCandidateSource(CreateCandidateSourceInput{
		SourceType: "current_resume",
		TrustTier:  "verified",
		Title:      "Resume",
		RawText: `SUMMARY
Hi

PROJECTS
- Built FastAPI APIs for planning workflows.`,
	})
	if err != nil {
		t.Fatalf("CreateCandidateSource() error = %v", err)
	}
	run, err := store.StartContextAgent(source.ID)
	if err != nil {
		t.Fatalf("StartContextAgent() error = %v", err)
	}
	finished, err := store.RunContextAgent(t.Context(), run.ID, nil)
	if err != nil {
		t.Fatalf("RunContextAgent() error = %v", err)
	}
	if finished.Status != contextAgentStatusComplete || finished.FactsCreated == 0 || finished.ClaimsCreated == 0 {
		t.Fatalf("finished = %+v, want complete with generated context", finished)
	}

	steps, err := store.ListContextAgentSteps(run.ID)
	if err != nil {
		t.Fatalf("ListContextAgentSteps() error = %v", err)
	}
	hasFactRunning := false
	hasSkipped := false
	for _, step := range steps {
		if step.Stage == "fact_extract" && step.Status == "running" {
			hasFactRunning = true
		}
		if step.Stage == "fact_extract" && step.Status == "skipped" {
			hasSkipped = true
		}
	}
	if !hasFactRunning || !hasSkipped {
		t.Fatalf("steps = %+v, want running progress and skipped empty section", steps)
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
	testClaims := testClaimsForFacts(facts)
	if _, err := store.replaceBulletDrafts(updated.ID, []parsedBulletDraft{{
		RequirementID: requirements[0].ID,
		ClaimIDs:      []int64{testClaims[0].ID},
		FactIDs:       []int64{facts[0].ID},
		DraftText:     "Built FastAPI APIs for production planning workflows.",
		Rationale:     "Uses exact FastAPI evidence.",
	}}, requirements, []factPromptContext{{ID: facts[0].ID}}, testClaims); err != nil {
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

func TestParseJobRequirementsFiltersProfileAndCompanyStory(t *testing.T) {
	requirements, err := parseJobRequirements(`{"requirements":[
		{"category":"responsibility","requirement_text":"Fullstack AI Software Engineer - (Agentic Systems, Fullstack Builder)","keywords":["Fullstack","AI","Software","Engineer"],"priority":"high","source_quote":"Fullstack AI Software Engineer - (Agentic Systems, Fullstack Builder)"},
		{"category":"responsibility","requirement_text":"Acting CTO | VP of AI & Engineering at Sonder | Building AI-First Healthcare Platforms | ex-Rokt","keywords":["Acting","CTO","Engineering","Sonder"],"priority":"high","source_quote":"Acting CTO | VP of AI & Engineering at Sonder | Building AI-First Healthcare Platforms | ex-Rokt"},
		{"category":"must_have","requirement_text":"At Sonder, we believe that every person deserves to feel safe, supported, and empowered to be at their best - wherever they are.","keywords":["Sonder","believe","supported"],"priority":"high","source_quote":"At Sonder, we believe that every person deserves to feel safe, supported, and empowered to be at their best - wherever they are."},
		{"category":"responsibility","requirement_text":"API Architecture: Design robust back-end services (Python/Node.js) that support data ingestion and high-concurrency reporting","keywords":["API","Python","Node.js","data ingestion"],"priority":"high","source_quote":"API Architecture: Design robust back-end services (Python/Node.js) that support data ingestion and high-concurrency reporting."},
		{"category":"responsibility","requirement_text":"RAG & Context Engineering: Build and optimize RAG pipelines, focusing on retrieval quality, chunking strategies, and metadata filtering","keywords":["RAG","retrieval","chunking","metadata filtering"],"priority":"medium","source_quote":"RAG & Context Engineering: Build and optimize RAG pipelines, focusing on retrieval quality, chunking strategies, and metadata filtering."}
	]}`)
	if err != nil {
		t.Fatalf("parseJobRequirements() error = %v", err)
	}
	if len(requirements) != 2 {
		t.Fatalf("requirements = %+v, want 2 concrete requirements", requirements)
	}
	for _, req := range requirements {
		if strings.Contains(req.RequirementText, "Acting CTO") || strings.Contains(req.RequirementText, "we believe") || strings.Contains(req.RequirementText, "Fullstack AI Software Engineer") {
			t.Fatalf("boilerplate survived: %+v", requirements)
		}
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
	claims, err := store.replaceCandidateClaims([]parsedCandidateClaim{{
		Label:            "FastAPI backend APIs",
		ClaimType:        "experience",
		SourceFactIDs:    []int64{facts[0].ID},
		Actions:          []string{"built"},
		Capabilities:     []string{"API development"},
		Technologies:     []string{"FastAPI"},
		Scope:            []string{"planning workflows"},
		EvidenceStrength: "direct",
		Strength:         "strong",
		AllowedUse:       []string{"experience_bullet"},
		AllowedContexts:  []string{"backend engineering"},
	}}, []factPromptContext{{
		ID:             facts[0].ID,
		Status:         factStatusApproved,
		Confidence:     "high",
		FactText:       facts[0].FactText,
		EvidenceQuote:  facts[0].EvidenceQuote,
		Technologies:   facts[0].Technologies,
		SectionHeading: facts[0].OriginHeading,
		SectionType:    facts[0].OriginType,
	}})
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}

	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Logf("decode request: %v", err)
			http.Error(w, "decode", http.StatusInternalServerError)
			return
		}
		content := request.Messages[len(request.Messages)-1].Content
		switch call {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"requirements\":[{\"category\":\"must_have\",\"requirement_text\":\"FastAPI experience\",\"keywords\":[\"FastAPI\"],\"priority\":\"high\",\"source_quote\":\"FastAPI experience\"}]}"}}]}`))
		case 2:
			if !strings.Contains(content, facts[0].Status) || !strings.Contains(content, facts[0].FactText) {
				t.Logf("match prompt missing relevant approved fact: %s", content)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"matches\":[{\"requirement_id\":1,\"fact_id\":1,\"score\":0.9,\"rationale\":\"Direct evidence\",\"coverage_status\":\"strong\"},{\"requirement_id\":1,\"fact_id\":1,\"score\":0,\"rationale\":\"No evidence\",\"coverage_status\":\"gap\"},{\"requirement_id\":1,\"fact_id\":999,\"score\":1,\"rationale\":\"Invalid\",\"coverage_status\":\"strong\"}]}"}}]}`))
		case 3:
			_, _ = w.Write([]byte(fmt.Sprintf(`{"choices":[{"message":{"role":"assistant","content":"{\"drafts\":[{\"requirement_id\":1,\"claim_ids\":[%d],\"fact_ids\":[1,999],\"draft_text\":\"Built FastAPI APIs for planning workflows.\",\"rationale\":\"Direct evidence\",\"risk_flags\":[]}]}"}}]}`, claims[0].ID)))
		default:
			t.Logf("unexpected LLM call %d", call)
			http.Error(w, "unexpected", http.StatusInternalServerError)
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
		drafts = append(drafts, parsedBulletDraft{RequirementID: requirements[0].ID, ClaimIDs: []int64{id}, FactIDs: []int64{id}, DraftText: fmt.Sprintf("Built backend API workflow %d with reliable integration behavior.", i)})
	}
	for i := 7; i <= 9; i++ {
		id := int64(i)
		facts = append(facts, factPromptContext{ID: id, Status: "approved", SectionHeading: "CueMate", SectionType: "project"})
		drafts = append(drafts, parsedBulletDraft{RequirementID: requirements[0].ID, ClaimIDs: []int64{id}, FactIDs: []int64{id}, DraftText: fmt.Sprintf("Built project API workflow %d with reliable integration behavior.", i)})
	}
	inserted, err := store.replaceBulletDrafts(job.ID, drafts, requirements, facts, testClaimsForPromptFacts(facts))
	if err != nil {
		t.Fatalf("replaceBulletDrafts() error = %v", err)
	}
	if len(inserted) != 2 {
		t.Fatalf("drafts len = %d, want one same-story draft per origin: %+v", len(inserted), inserted)
	}
	selected, err := store.AutoSelectResumeBullets(job.ID)
	if err != nil {
		t.Fatalf("AutoSelectResumeBullets() error = %v", err)
	}
	selectedCount := 0
	for _, draft := range selected {
		if draft.SelectedForResume {
			selectedCount++
		}
	}
	if selectedCount != 2 {
		t.Fatalf("selected drafts = %d, want 2 story-diverse representatives: %+v", selectedCount, selected)
	}
}

func TestBulletDraftsRejectNonHumanStyle(t *testing.T) {
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
	testClaims := testClaimsForPromptFacts([]factPromptContext{{ID: 1, Status: "approved", SectionHeading: "Acme", SectionType: "experience"}})
	_, err = store.replaceBulletDrafts(job.ID, []parsedBulletDraft{{
		RequirementID: requirements[0].ID,
		ClaimIDs:      []int64{1},
		FactIDs:       []int64{1},
		DraftText:     "- Leveraged cutting-edge APIs to enhance efficiency and drive growth across business outcomes with seamless dynamic execution for stakeholder value.",
	}}, requirements, []factPromptContext{{ID: 1, Status: "approved", SectionHeading: "Acme", SectionType: "experience"}}, testClaims)
	if err == nil || !strings.Contains(err.Error(), "no usable same-origin bullet drafts") {
		t.Fatalf("replaceBulletDrafts() error = %v, want rejected non-human draft", err)
	}
}

func TestBulletDraftsKeepDistinctBulletsFromRichClaim(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Acme",
		Title:   "Backend Engineer",
		RawText: "Design and deliver SaaS features.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	requirements, err := store.replaceJobRequirements(job, []parsedJobRequirement{{
		Category:        "responsibility",
		RequirementText: "Design and deliver SaaS features.",
		Keywords:        []string{"design", "deliver", "SaaS"},
		Priority:        "high",
		SourceQuote:     "Design and deliver SaaS features.",
	}})
	if err != nil {
		t.Fatalf("replaceJobRequirements() error = %v", err)
	}
	facts := []factPromptContext{
		{ID: 1, Status: "approved", FactText: "Built FastAPI/PostgreSQL backend APIs for bookings and asset scheduling.", SectionHeading: "Sitespace", SectionType: "experience"},
		{ID: 2, Status: "approved", FactText: "Added RBAC and immutable audit logging for production data changes.", SectionHeading: "Sitespace", SectionType: "experience"},
	}
	claims := []CandidateClaim{{
		ID:               1,
		ClaimText:        "FastAPI backend with scheduling APIs, RBAC, and audit logging",
		ClaimType:        "experience",
		SourceFactIDs:    []int64{1, 2},
		Actions:          []string{"built", "shipped", "added"},
		Technologies:     []string{"FastAPI", "PostgreSQL"},
		Artifacts:        []string{"backend APIs", "RBAC", "audit logging"},
		Scope:            []string{"bookings", "asset scheduling", "production data changes"},
		Outcomes:         []string{"access control", "traceability"},
		EvidenceStrength: "direct",
		Strength:         "strong",
		AllowedUse:       []string{"experience_bullet"},
		OriginHeading:    "Sitespace",
		OriginType:       "experience",
		Status:           claimStatusApproved,
	}}
	group := bulletOriginGroup{
		OriginHeading: "Sitespace",
		OriginType:    "experience",
		Requirements:  requirements,
		Facts:         facts,
		Claims:        claims,
	}
	packets := buildEvidencePackets(group)
	if len(packets) < 2 {
		t.Fatalf("packets = %+v, want multiple value themes from rich claim", packets)
	}
	inserted, err := store.replaceBulletDrafts(job.ID, []parsedBulletDraft{
		{
			OriginHeading: "Sitespace",
			OriginType:    "experience",
			ValueTheme:    "product_platform_delivery",
			RequirementID: requirements[0].ID,
			ClaimIDs:      []int64{1},
			FactIDs:       []int64{1, 2},
			DraftText:     "Built and shipped FastAPI/PostgreSQL APIs for bookings and asset scheduling, turning planning requirements into usable platform workflows.",
		},
		{
			OriginHeading: "Sitespace",
			OriginType:    "experience",
			ValueTheme:    "security_traceability",
			RequirementID: requirements[0].ID,
			ClaimIDs:      []int64{1},
			FactIDs:       []int64{1, 2},
			DraftText:     "Added RBAC and immutable audit logging to strengthen access control and traceability across production data changes.",
		},
	}, requirements, facts, claims)
	if err != nil {
		t.Fatalf("replaceBulletDrafts() error = %v", err)
	}
	if len(inserted) != 2 {
		t.Fatalf("inserted = %+v, want two distinct bullets from same rich claim", inserted)
	}
}

func TestBulletDraftsUseSameOriginClaimsBeyondDirectMatch(t *testing.T) {
	requirements := []JobRequirement{{ID: 1, RequirementText: "Build SaaS features.", Priority: "high", Category: "responsibility"}}
	facts := []factPromptContext{
		{ID: 1, Status: "approved", FactText: "Built backend APIs.", SectionHeading: "Sitespace", SectionType: "experience"},
		{ID: 2, Status: "approved", FactText: "Added audit logging.", SectionHeading: "Sitespace", SectionType: "experience"},
	}
	matches := []JobFactMatch{{RequirementID: 1, FactID: 1, Score: 0.9, CoverageStatus: "strong"}}
	claims := []CandidateClaim{
		{
			ID:            1,
			ClaimText:     "backend APIs",
			SourceFactIDs: []int64{1},
			Artifacts:     []string{"backend APIs"},
			OriginHeading: "Sitespace",
			OriginType:    "experience",
			Status:        claimStatusApproved,
		},
		{
			ID:            2,
			ClaimText:     "audit logging traceability",
			SourceFactIDs: []int64{2},
			Artifacts:     []string{"audit logging"},
			Outcomes:      []string{"traceability"},
			OriginHeading: "Sitespace",
			OriginType:    "experience",
			Status:        claimStatusApproved,
		},
	}
	groups := buildBulletOriginGroups(requirements, matches, facts, claims)
	if len(groups) != 1 {
		t.Fatalf("groups = %+v, want one Sitespace group", groups)
	}
	if len(groups[0].Claims) != 2 || len(groups[0].Facts) != 2 {
		t.Fatalf("group = %+v, want same-origin unmatched claim/fact included", groups[0])
	}
}

func TestBulletDraftsCollapseSameBackendStoryVariants(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	job, err := store.CreateJobDescription(CreateJobDescriptionInput{Company: "Acme", Title: "Backend Engineer", RawText: "Build APIs."})
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
	facts := []factPromptContext{{ID: 1, Status: "approved", SectionHeading: "Sitespace", SectionType: "experience"}}
	claims := testClaimsForPromptFacts(facts)
	inserted, err := store.replaceBulletDrafts(job.ID, []parsedBulletDraft{
		{
			OriginHeading: "Sitespace",
			OriginType:    "experience",
			ValueTheme:    "product_platform_delivery",
			RequirementID: requirements[0].ID,
			ClaimIDs:      []int64{1},
			FactIDs:       []int64{1},
			DraftText:     "Built FastAPI backend APIs for bookings and asset scheduling, turning planning requirements into usable platform workflows.",
		},
		{
			OriginHeading: "Sitespace",
			OriginType:    "experience",
			ValueTheme:    "product_platform_delivery",
			RequirementID: requirements[0].ID,
			ClaimIDs:      []int64{1},
			FactIDs:       []int64{1},
			DraftText:     "Delivered backend APIs for asset scheduling and bookings on a FastAPI stack, supporting planning workflows.",
		},
	}, requirements, facts, claims)
	if err != nil {
		t.Fatalf("replaceBulletDrafts() error = %v", err)
	}
	if len(inserted) != 1 {
		t.Fatalf("inserted = %+v, want duplicate backend story collapsed", inserted)
	}
}

func TestCandidateClaimLedgerWorkflow(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	blocked, err := store.ListBlockedClaims()
	if err != nil {
		t.Fatalf("ListBlockedClaims() error = %v", err)
	}
	if len(blocked) == 0 {
		t.Fatal("default blocked claims not seeded")
	}

	_, facts := createFactsForJobTests(t, store)
	claims, err := store.replaceCandidateClaims([]parsedCandidateClaim{
		{
			ClaimText:     "Built FastAPI APIs for planning workflows across booking and asset scheduling modules.",
			ClaimType:     "experience",
			SourceFactIDs: []int64{facts[0].ID},
			Strength:      "strong",
		},
		{
			ClaimText:     "Claim with missing fact should be discarded.",
			SourceFactIDs: []int64{999999},
		},
	}, []factPromptContext{
		{ID: facts[0].ID, Status: facts[0].Status, Confidence: facts[0].Confidence, FactText: facts[0].FactText, EvidenceQuote: facts[0].EvidenceQuote, Technologies: facts[0].Technologies, SectionHeading: "Sitespace", SectionType: "experience"},
	})
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}
	if len(claims) != 1 || claims[0].Status != claimStatusApproved || len(claims[0].SourceFactIDs) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
	if strings.HasSuffix(claims[0].ClaimText, ".") || len(strings.Fields(claims[0].ClaimText)) > 10 {
		t.Fatalf("claim label is sentence-shaped: %q", claims[0].ClaimText)
	}
	if len(claims[0].Actions) == 0 || len(claims[0].Capabilities) == 0 {
		t.Fatalf("claim atoms missing: %+v", claims[0])
	}

	updated, err := store.UpdateCandidateClaimReview(UpdateCandidateClaimReviewInput{
		ID:               claims[0].ID,
		ClaimText:        claims[0].ClaimText,
		ClaimType:        claims[0].ClaimType,
		Actions:          claims[0].Actions,
		Capabilities:     claims[0].Capabilities,
		Objects:          claims[0].Objects,
		Technologies:     claims[0].Technologies,
		Domains:          claims[0].Domains,
		Artifacts:        claims[0].Artifacts,
		Scope:            claims[0].Scope,
		Metrics:          claims[0].Metrics,
		Outcomes:         claims[0].Outcomes,
		ProfileContext:   claims[0].ProfileContext,
		EvidenceStrength: claims[0].EvidenceStrength,
		Strength:         claims[0].Strength,
		AllowedUse:       claims[0].AllowedUse,
		AllowedContexts:  claims[0].AllowedContexts,
		BlockedContexts:  claims[0].BlockedContexts,
		SafePhrasings:    claims[0].SafePhrasings,
		UnsafePhrasings:  claims[0].UnsafePhrasings,
		Status:           claimStatusApprovedRestricted,
		RiskFlags:        []string{"manual_limit"},
		ReviewNote:       "Keep narrow.",
	})
	if err != nil {
		t.Fatalf("UpdateCandidateClaimReview() error = %v", err)
	}
	if updated.Status != claimStatusApprovedRestricted || !listContains(updated.RiskFlags, "manual_limit") {
		t.Fatalf("updated = %+v", updated)
	}
}

func TestCandidateClaimsAggregateAtomicFactsForBulletUse(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	facts := []factPromptContext{
		{
			ID:             101,
			Status:         factStatusApproved,
			Confidence:     "high",
			FactText:       "actions=implemented; artifact=RBAC and audit logging; tools=FastAPI",
			EvidenceQuote:  "Implemented RBAC and audit logging with FastAPI.",
			Technologies:   []string{"FastAPI"},
			SectionHeading: "Sitespace",
			SectionType:    "experience",
		},
		{
			ID:             102,
			Status:         factStatusApproved,
			Confidence:     "high",
			FactText:       "scope=construction document workflows",
			EvidenceQuote:  "Covered construction document workflows.",
			SectionHeading: "Sitespace",
			SectionType:    "experience",
		},
		{
			ID:             103,
			Status:         factStatusApproved,
			Confidence:     "high",
			FactText:       "outcome=traceable access reviews; metric=25%",
			EvidenceQuote:  "Improved access review coverage by 25%.",
			SectionHeading: "Sitespace",
			SectionType:    "experience",
		},
	}

	claims, err := store.replaceCandidateClaims([]parsedCandidateClaim{{
		Label:         "RBAC audit workflow traceability",
		ClaimType:     "experience",
		SourceFactIDs: []int64{101, 102, 103},
	}}, facts)
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
	claim := claims[0]
	if !listContains(claim.Artifacts, "RBAC and audit logging") ||
		!listContains(claim.Scope, "construction document workflows") ||
		!listContains(claim.Outcomes, "traceable access reviews") ||
		!listContains(claim.Metrics, "25%") {
		t.Fatalf("claim atoms were not aggregated from all facts: %+v", claim)
	}

	text := fallbackBulletTextFromClaim(claim, JobRequirement{
		Category:        "responsibility",
		RequirementText: "Build secure workflow systems.",
		SourceQuote:     "Build secure workflow systems.",
	})
	if !strings.HasPrefix(text, "Implemented ") ||
		!strings.Contains(text, "RBAC and audit logging") ||
		!strings.Contains(text, "construction document workflows") ||
		!strings.Contains(text, "traceable access reviews") ||
		!strings.Contains(text, "25%") {
		t.Fatalf("fallback bullet did not synthesize atomic value: %q", text)
	}
}

func TestCandidateClaimsBlockUnsupportedScope(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	_, facts := createFactsForJobTests(t, store)
	claims, err := store.replaceCandidateClaims([]parsedCandidateClaim{{
		ClaimText:     "Led a senior Kubernetes team and improved latency by 45%.",
		SourceFactIDs: []int64{facts[0].ID},
	}}, []factPromptContext{{
		ID: factIDOrPanic(t, facts), Status: facts[0].Status, Confidence: facts[0].Confidence, FactText: facts[0].FactText, EvidenceQuote: facts[0].EvidenceQuote, Technologies: facts[0].Technologies, SectionHeading: "Sitespace", SectionType: "experience",
	}})
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}
	if len(claims) != 1 || claims[0].Status != claimStatusBlocked {
		t.Fatalf("claims = %+v, want blocked", claims)
	}
	flags := strings.Join(claims[0].RiskFlags, " ")
	for _, want := range []string{"blocked_context", "unsupported_metric"} {
		if !strings.Contains(flags, want) {
			t.Fatalf("flags = %+v, missing %s", claims[0].RiskFlags, want)
		}
	}
}

func TestCandidateClaimsMergeSimilarAtomRecords(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	claims, err := store.replaceCandidateClaims([]parsedCandidateClaim{
		{
			Label:         "FastAPI booking APIs",
			ClaimType:     "experience",
			SourceFactIDs: []int64{101},
			Actions:       []string{"built"},
			Capabilities:  []string{"API development"},
			Technologies:  []string{"FastAPI"},
			Scope:         []string{"booking workflows"},
		},
		{
			Label:         "FastAPI booking endpoints",
			ClaimType:     "experience",
			SourceFactIDs: []int64{102},
			Actions:       []string{"implemented"},
			Capabilities:  []string{"API development"},
			Technologies:  []string{"FastAPI"},
			Scope:         []string{"booking workflows"},
		},
	}, []factPromptContext{
		{ID: 101, Status: factStatusApproved, Confidence: "high", FactText: "actions=built; artifact=booking APIs; tools=FastAPI", EvidenceQuote: "Built FastAPI booking APIs.", Technologies: []string{"FastAPI"}, SectionHeading: "Sitespace", SectionType: "experience"},
		{ID: 102, Status: factStatusApproved, Confidence: "high", FactText: "actions=implemented; artifact=booking APIs; tools=FastAPI", EvidenceQuote: "Implemented FastAPI booking endpoints.", Technologies: []string{"FastAPI"}, SectionHeading: "Sitespace", SectionType: "experience"},
	})
	if err != nil {
		t.Fatalf("replaceCandidateClaims() error = %v", err)
	}
	if len(claims) != 1 || len(claims[0].SourceFactIDs) != 2 {
		t.Fatalf("claims = %+v, want one merged claim with two facts", claims)
	}
	if claims[0].SimilarityKey == "" || claims[0].SimilarityScore != 1 {
		t.Fatalf("similarity metadata missing: %+v", claims[0])
	}
}

func factIDOrPanic(t *testing.T, facts []EvidenceFact) int64 {
	t.Helper()
	if len(facts) == 0 {
		t.Fatal("no facts")
	}
	return facts[0].ID
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

func TestCreateJobDescriptionRepairsPastedLinkedInMojibake(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		RawText: "Magentus\nSoftware Engineer\nGreater Sydney Area Â· Reposted 20 hours ago\nYouâ€™ll own solutions end-to-endâ€”from design through production support.",
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	if strings.Contains(job.RawText, "Â") || strings.Contains(job.RawText, "â") {
		t.Fatalf("raw text still contains mojibake: %q", job.RawText)
	}
	for _, want := range []string{"You'll own solutions", "end-to-end - from", "Greater Sydney Area - Reposted"} {
		if !strings.Contains(job.RawText, want) {
			t.Fatalf("raw text missing %q: %q", want, job.RawText)
		}
	}
}

func TestCreateJobDescriptionIgnoresLinkedInLogoNoise(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{RawText: `TechShack logo
TechShack
Software Engineer
Melbourne, Victoria, Australia · 2 days ago · Over 100 applicants

Promoted by hirer · Actively reviewing applicants
How your profile and resume fit this job

Meet the hiring team
Holly Timings-Thompson
3rd
Recruitment Consultant
Job poster
About the job

Golang Engineer - Cybersecurity Platform

Melbourne, Australia (Hybrid)

We're hiring a Mid to Senior Golang Engineer to join a growing cybersecurity company building high-scale intelligence and monitoring platforms.`})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}
	if job.Company != "TechShack" || job.Title != "Golang Engineer - Cybersecurity Platform" {
		t.Fatalf("job details = %q/%q, want TechShack/Golang Engineer - Cybersecurity Platform", job.Company, job.Title)
	}
	if strings.Contains(strings.ToLower(job.Company), "logo") || strings.Contains(strings.ToLower(job.Title), "logo") {
		t.Fatalf("job details contain logo noise: %q/%q", job.Company, job.Title)
	}
}

func TestParseJobDescriptionBackfillsExplicitRequirements(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-test")
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	job, err := store.CreateJobDescription(CreateJobDescriptionInput{
		Company: "Magentus",
		Title:   "Software Engineer",
		RawText: `About You
Strong experience with TypeScript, Node.js, and React in production environments
Hands-on experience with AWS (serverless, containers, networking, databases)
Solid understanding of modern DevOps practices
Experience using AI tools in development workflows
Exposure to regulated industries (healthcare, pharma, or medical devices) is highly regarded`,
	})
	if err != nil {
		t.Fatalf("CreateJobDescription() error = %v", err)
	}

	content := `{"requirements":[{"category":"must_have","requirement_text":"Strong experience with TypeScript, Node.js, and React in production environments","keywords":["TypeScript","Node.js","React"],"priority":"high","source_quote":"Strong experience with TypeScript, Node.js, and React in production environments"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":%q}}]}`, content)
	}))
	defer server.Close()
	restoreURL := openRouterChatCompletionsURLForTest(server.URL)
	defer restoreURL()

	requirements, err := store.ParseJobDescription(t.Context(), job.ID, server.Client())
	if err != nil {
		t.Fatalf("ParseJobDescription() error = %v", err)
	}
	for _, want := range []string{"AWS", "modern DevOps practices", "regulated industries"} {
		found := false
		for _, req := range requirements {
			if strings.Contains(req.RequirementText, want) || stringListContains(req.Keywords, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("requirements missing %q: %+v", want, requirements)
		}
	}
}

func TestJobAnalysisTopPainPointsSkipCredentialsAndSoftSkills(t *testing.T) {
	analysis := buildJobAnalysis(JobDescription{ID: 1, Company: "Magentus", Title: "Software Engineer"}, []JobRequirement{
		{RequirementText: "Collaborate with cross-functional teams to deliver shared outcomes across the Magentus group", Category: "responsibility", Priority: "medium", Keywords: []string{"collaborate", "cross-functional"}},
		{RequirementText: "Degree in Computer Science/IT or equivalent practical experience", Category: "must_have", Priority: "medium", Keywords: []string{"Computer Science", "IT"}},
		{RequirementText: "Design, build, and deliver new features for our health-tech SaaS platform", Category: "responsibility", Priority: "high", Keywords: []string{"design", "build", "SaaS"}},
		{RequirementText: "Hands-on experience with AWS (serverless, containers, networking, databases)", Category: "must_have", Priority: "high", Keywords: []string{"AWS", "serverless"}},
		{RequirementText: "Contribute to continuous improvement across software design, product capability, and system architecture", Category: "responsibility", Priority: "medium", Keywords: []string{"system architecture"}},
	})
	if len(analysis.TopPainPoints) == 0 {
		t.Fatal("top pain points empty")
	}
	for _, bad := range []string{"Degree", "Collaborate"} {
		for _, painPoint := range analysis.TopPainPoints {
			if strings.Contains(painPoint, bad) {
				t.Fatalf("top pain points include low-signal item %q: %+v", bad, analysis.TopPainPoints)
			}
		}
	}
	if analysis.TopPainPoints[0] != "Design, build, and deliver new features for our health-tech SaaS platform" {
		t.Fatalf("top pain points = %+v, want parser order with strongest responsibility first", analysis.TopPainPoints)
	}
}

func TestJobAnalysisFiltersCatapultDisplayNoise(t *testing.T) {
	analysis := buildJobAnalysis(JobDescription{ID: 1, Company: "Catapult", Title: "Senior Software Engineer", RawText: "Melbourne, Victoria, Australia · Reposted 1 day ago · Over 100 people clicked apply\nRetry Premium for A$0"}, []JobRequirement{
		{RequirementText: "A customer and product mindset - Understanding of user needs and a focus on delivering impactful solutions", Category: "responsibility", Priority: "high", Keywords: []string{"customer", "product", "impactful"}},
		{RequirementText: "Contribute to technical design discussions, proposing solutions, and collaborating with technical leaders and colleagues globally to define system architecture and design patterns", Category: "responsibility", Priority: "high", Keywords: []string{"architecture", "design patterns", "Define"}},
		{RequirementText: "Participate in code reviews to ensure code quality, adherence to coding standards, and best practices. Conducting unit testing and integration testing to verify functionality and reliability", Category: "responsibility", Priority: "high", Keywords: []string{"Participate", "code reviews", "unit testing", "integration testing"}},
		{RequirementText: "Real-time data processing using technologies like Kafka, Kinesis, AWS, and edge devices. We're processing live athlete data streams like never before, fueling data-driven decisions that impact the performance of athletes", Category: "responsibility", Priority: "high", Keywords: []string{"Kafka", "Kinesis", "AWS", "edge devices", "real-time"}},
		{RequirementText: "Experience with AWS infrastructure, including IoT, IaC, Go, Rust, C# .Net, C++", Category: "nice_to_have", Priority: "low", Keywords: []string{"AWS", "IoT", "IaC", "Go", "Rust", "C#", ".Net", "C++"}},
		{RequirementText: "Proficiency in microservice architectures within AWS, applying domain-driven design principles", Category: "must_have", Priority: "high", Keywords: []string{"AWS", "microservice", "domain-driven design"}},
		{RequirementText: "An understanding of NoSQL & relational database architecture, querying and performance", Category: "must_have", Priority: "high", Keywords: []string{"NoSQL", "relational", "database", "querying", "performance"}},
	})

	if analysis.Location != "Melbourne, Victoria, Australia" {
		t.Fatalf("location = %q, want clean LinkedIn location", analysis.Location)
	}
	if analysis.Salary != "" {
		t.Fatalf("salary = %q, want Premium upsell ignored", analysis.Salary)
	}
	for _, value := range analysis.Responsibilities {
		if strings.Contains(value, "customer and product mindset") {
			t.Fatalf("responsibilities include customer mindset noise: %+v", analysis.Responsibilities)
		}
		if strings.Contains(value, "We're processing live athlete data") || strings.Contains(value, "Conducting unit testing") {
			t.Fatalf("responsibilities include explanatory follow-up sentence: %+v", analysis.Responsibilities)
		}
	}
	for _, value := range append(append([]string{}, analysis.RequiredSkills...), analysis.Keywords...) {
		for _, bad := range []string{"Define", "Participate", "customer", "product", "adherence"} {
			if value == bad {
				t.Fatalf("analysis skills include generic token %q: required=%+v keywords=%+v", bad, analysis.RequiredSkills, analysis.Keywords)
			}
		}
	}
	for _, want := range []string{"AWS", "Microservices", "Domain-driven design", "NoSQL", "Relational databases"} {
		if !stringListContains(analysis.RequiredSkills, want) {
			t.Fatalf("required skills missing %q: %+v", want, analysis.RequiredSkills)
		}
	}
	for _, want := range []string{"IoT", "IaC", "Go", "Rust", "C#", ".Net", "C++"} {
		if !stringListContains(analysis.PreferredSkills, want) {
			t.Fatalf("preferred skills missing %q: %+v", want, analysis.PreferredSkills)
		}
	}
	if len(analysis.TopPainPoints) == 0 || strings.Contains(analysis.TopPainPoints[0], "customer and product mindset") {
		t.Fatalf("top pain points prioritize noise: %+v", analysis.TopPainPoints)
	}
	for _, value := range analysis.TopPainPoints {
		if strings.Contains(value, "We're processing live athlete data") || strings.Contains(value, "Conducting unit testing") {
			t.Fatalf("top pain points include explanatory follow-up sentence: %+v", analysis.TopPainPoints)
		}
	}
}

func TestParseJobRequirementsCleansCatapultNoise(t *testing.T) {
	response := `{"requirements":[
		{"category":"responsibility","requirement_text":"Catapult is a sports technology company that empowers professional teams to make data-driven decisions. We deliver health, performance, video, and AI insights from the locker room to competitive environments, ensuring every decision is an opportunity to gain an advantage, sharpen performance, and build lasting success","keywords":["Catapult","sports","company"],"priority":"high","source_quote":"Catapult is a sports technology company that empowers professional teams to make data-driven decisions. We deliver health, performance, video, and AI insights from the locker room to competitive environments, ensuring every decision is an opportunity to gain an advantage, sharpen performance, and build lasting success"},
		{"category":"responsibility","requirement_text":"Live Data Stream Processing: Dive into the world of real-time data processing using technologies like Kafka, Kinesis, AWS, and edge devices","keywords":["Live","Data","Stream","Processing","AWS"],"priority":"high","source_quote":"Live Data Stream Processing: Dive into the world of real-time data processing using technologies like Kafka, Kinesis, AWS, and edge devices"},
		{"category":"must_have","requirement_text":"Experience with the specific technologies in our stack is advantageous but not mandatory. (AWS infrastructure, including IoT, IaC, Go, Rust, C# .Net, C++)","keywords":["AWS","Rust"],"priority":"medium","source_quote":"Experience with the specific technologies in our stack is advantageous but not mandatory. (AWS infrastructure, including IoT, IaC, Go, Rust, C# .Net, C++)"},
		{"category":"must_have","requirement_text":"A collaborative team player with strong verbal and written communication skills","keywords":["collaborative","communication"],"priority":"high","source_quote":"A collaborative team player with strong verbal and written communication skills"},
		{"category":"must_have","requirement_text":"Proficiency in microservice architectures within AWS, applying domain-driven design principles","keywords":["AWS","microservice"],"priority":"high","source_quote":"Proficiency in microservice architectures within AWS, applying domain-driven design principles"}
	]}`
	requirements, err := parseJobRequirements(response)
	if err != nil {
		t.Fatalf("parseJobRequirements() error = %v", err)
	}
	if len(requirements) != 3 {
		t.Fatalf("requirements = %+v, want company blurb and soft skill removed", requirements)
	}
	if requirements[0].RequirementText != "Real-time data processing using technologies like Kafka, Kinesis, AWS, and edge devices" {
		t.Fatalf("stream requirement = %q", requirements[0].RequirementText)
	}
	optional := requirements[1]
	if optional.Category != "nice_to_have" || optional.Priority != "low" {
		t.Fatalf("optional stack requirement = %+v, want nice_to_have/low", optional)
	}
	if !stringListContains(optional.Keywords, "IoT") || !stringListContains(optional.Keywords, "IaC") || !stringListContains(optional.Keywords, "Go") {
		t.Fatalf("optional keywords = %+v, want stack terms preserved", optional.Keywords)
	}
}

func TestJobMatchSanitizerBalancesCatapultTransferableEvidence(t *testing.T) {
	streamReq := JobRequirement{
		ID:              1,
		Category:        "must_have",
		Priority:        "high",
		RequirementText: "Real-time data processing using technologies like Kafka, Kinesis, AWS, and edge devices",
		Keywords:        []string{"Kafka", "Kinesis", "AWS", "real-time", "edge devices"},
	}
	sftpFact := factPromptContext{
		ID:           10,
		FactText:     "actions=refactored; artifact=SFTP/FTP ingestion pipelines; scope=automated processing flows; outcome=improved reliability and reduced manual intervention for data transfers",
		Technologies: []string{"SFTP", "FTP"},
	}
	match := sanitizeParsedJobMatch(parsedJobMatch{RequirementID: 1, FactID: 10, Score: 0.88, CoverageStatus: "strong"}, streamReq, sftpFact)
	if match.Score < 0.45 || match.Score >= 0.75 || match.CoverageStatus != "partial" {
		t.Fatalf("stream/SFTP match = %+v, want partial transferable evidence", match)
	}
	awsOnlyFact := factPromptContext{
		ID:           12,
		FactText:     "experience_years=3+; technologies=AWS, Java, Python",
		Technologies: []string{"AWS", "Java", "Python"},
	}
	match = sanitizeParsedJobMatch(parsedJobMatch{RequirementID: 1, FactID: 12, Score: 0.54, CoverageStatus: "partial"}, streamReq, awsOnlyFact)
	if match.Score <= 0 || match.Score >= 0.45 || match.CoverageStatus != "weak" {
		t.Fatalf("stream/AWS-only match = %+v, want weak adjacent evidence", match)
	}

	iotReq := JobRequirement{
		ID:              2,
		Category:        "must_have",
		Priority:        "high",
		RequirementText: "Enhance athlete support and customer experiences through IoT capabilities that provide real-time information about device health and configuration",
		Keywords:        []string{"IoT", "real-time", "device", "configuration"},
	}
	linuxFact := factPromptContext{
		ID:           11,
		FactText:     "actions=built, supported; artifact=automation scripts and supported Linux-based internal systems; tools=Linux",
		Technologies: []string{"Linux"},
	}
	match = sanitizeParsedJobMatch(parsedJobMatch{RequirementID: 2, FactID: 11, Score: 0.75, CoverageStatus: "strong"}, iotReq, linuxFact)
	if match.Score <= 0 || match.Score >= 0.45 || match.CoverageStatus != "weak" {
		t.Fatalf("iot/linux match = %+v, want weak adjacent evidence", match)
	}

	codeQualityReq := JobRequirement{
		ID:              3,
		Category:        "must_have",
		Priority:        "high",
		RequirementText: "Participate in code reviews to ensure code quality, adherence to coding standards, and best practices. Conducting unit testing and integration testing to verify functionality and reliability",
		Keywords:        []string{"code reviews", "code quality", "coding standards", "unit testing", "integration testing", "reliability"},
	}
	reliabilityFact := factPromptContext{
		ID:       13,
		FactText: "actions=refactored; artifact=SFTP/FTP ingestion pipelines; outcome=improved reliability and reduced manual intervention for data transfers",
	}
	match = sanitizeParsedJobMatch(parsedJobMatch{RequirementID: 3, FactID: 13, Score: 0.68, CoverageStatus: "partial"}, codeQualityReq, reliabilityFact)
	if match.Score < 0.45 || match.Score >= 0.75 || match.CoverageStatus != "partial" {
		t.Fatalf("code-quality/reliability-only match = %+v, want partial transferable evidence", match)
	}

	productReq := JobRequirement{
		ID:              4,
		Category:        "responsibility",
		Priority:        "medium",
		RequirementText: "A customer and product mindset - Understanding of user needs and a focus on delivering impactful solutions",
		Keywords:        []string{"customer", "product", "user", "delivering", "impactful"},
	}
	userFlowFact := factPromptContext{
		ID:           14,
		FactText:     "actions=built; artifact=login, registration, forgot password, reset password, and invited-user password setup flows",
		Technologies: []string{"Go", "Vite"},
	}
	match = sanitizeParsedJobMatch(parsedJobMatch{RequirementID: 4, FactID: 14, Score: 0.68, CoverageStatus: "partial"}, productReq, userFlowFact)
	if match.Score < 0.45 || match.Score >= 0.75 || match.CoverageStatus != "partial" {
		t.Fatalf("product-mindset/user-flow match = %+v, want partial transferable evidence", match)
	}
}

func TestJobMatchSanitizerAllowsOptionalGoEvidenceButCapsScore(t *testing.T) {
	req := JobRequirement{
		ID:              1,
		Category:        "nice_to_have",
		Priority:        "low",
		RequirementText: "Experience with the specific technologies in our stack is advantageous but not mandatory. (AWS infrastructure, including IoT, IaC, Go, Rust, C# .Net, C++)",
		Keywords:        []string{"AWS", "IoT", "IaC", "Go", "Rust", "C#", ".Net", "C++"},
	}
	fact := factPromptContext{
		ID:           12,
		FactText:     "actions=designed; artifact=custom peer-to-peer transfer protocol; technologies=Go, goroutines, mutexes",
		Technologies: []string{"Go"},
	}
	match := sanitizeParsedJobMatch(parsedJobMatch{RequirementID: 1, FactID: 12, Score: 0.96, CoverageStatus: "strong"}, req, fact)
	if match.Score <= 0 || match.CoverageStatus == "gap" {
		t.Fatalf("go optional match rejected: %+v", match)
	}
	if match.Score > 0.72 || match.CoverageStatus == "strong" {
		t.Fatalf("go optional match = %+v, want capped non-strong score", match)
	}
}

func stringListContains(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
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
	var matched JobFactMatch
	for _, match := range matches {
		if match.RequirementID == requirements[0].ID && match.FactID == facts[0].ID {
			foundApprovedFastAPI = true
			matched = match
		}
	}
	if !foundApprovedFastAPI {
		t.Fatalf("matches = %+v", matches)
	}
	if !strings.Contains(matched.Rationale, "Local keyword overlap") {
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
			"tag_name": "v0.16.9",
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

	url, _, err := tectonicDownloadURL(t.Context(), server.Client())
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

func testClaimsForFacts(facts []EvidenceFact) []CandidateClaim {
	claims := []CandidateClaim{}
	for index, fact := range facts {
		claims = append(claims, CandidateClaim{
			ID:               int64(index + 1),
			ClaimText:        "FastAPI backend APIs",
			ClaimType:        firstNonEmpty(fact.OriginType, "experience"),
			SourceFactIDs:    []int64{fact.ID},
			Technologies:     fact.Technologies,
			Actions:          []string{"built"},
			Capabilities:     []string{"API development"},
			Scope:            []string{"planning workflows"},
			EvidenceStrength: "direct",
			Strength:         "strong",
			AllowedUse:       []string{"experience_bullet"},
			AllowedContexts:  []string{"backend engineering"},
			OriginHeading:    fact.OriginHeading,
			OriginType:       firstNonEmpty(fact.OriginType, "experience"),
			Status:           claimStatusApproved,
		})
	}
	return claims
}

func factsToPromptContext(facts []EvidenceFact) []factPromptContext {
	contextFacts := []factPromptContext{}
	for _, fact := range facts {
		contextFacts = append(contextFacts, factPromptContext{
			ID:             fact.ID,
			Status:         fact.Status,
			Confidence:     fact.Confidence,
			RiskFlags:      fact.RiskFlags,
			FactText:       fact.FactText,
			EvidenceQuote:  fact.EvidenceQuote,
			Technologies:   fact.Technologies,
			SectionHeading: fact.OriginHeading,
			SectionType:    fact.OriginType,
			Context:        fact.Context,
		})
	}
	return contextFacts
}

func testClaimsForPromptFacts(facts []factPromptContext) []CandidateClaim {
	claims := []CandidateClaim{}
	for _, fact := range facts {
		claims = append(claims, CandidateClaim{
			ID:               fact.ID,
			ClaimText:        "API workflow",
			ClaimType:        firstNonEmpty(fact.SectionType, "experience"),
			SourceFactIDs:    []int64{fact.ID},
			Actions:          []string{"built"},
			Capabilities:     []string{"API development"},
			Scope:            []string{"workflow"},
			EvidenceStrength: "direct",
			Strength:         "strong",
			AllowedUse:       []string{"experience_bullet"},
			AllowedContexts:  []string{"backend engineering"},
			OriginHeading:    fact.SectionHeading,
			OriginType:       firstNonEmpty(fact.SectionType, "experience"),
			Status:           claimStatusApproved,
		})
	}
	return claims
}

func tableHasColumn(t *testing.T, store *Store, table string, column string) bool {
	t.Helper()
	rows, err := store.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("table_info %s: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table_info %s: %v", table, err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table_info rows %s: %v", table, err)
	}
	return false
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

func openRouterEmbeddingsURLForTest(url string) func() {
	previous := openRouterEmbeddingsURL
	openRouterEmbeddingsURL = url
	return func() {
		openRouterEmbeddingsURL = previous
	}
}
