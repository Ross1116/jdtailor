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
		Provider: "openai",
		Model:    "gpt-test",
	})
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if saved.Model != "gpt-test" {
		t.Fatalf("saved Model = %q, want gpt-test", saved.Model)
	}

	loaded, err := store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() after save error = %v", err)
	}
	if loaded.Model != "gpt-test" {
		t.Fatalf("loaded Model = %q, want gpt-test", loaded.Model)
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
	if string(content) != "OPENAI_API_KEY="+secret+"\n" {
		t.Fatalf(".env.local content = %q", string(content))
	}
}

func TestOpenAIEnvVarPrecedence(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	if err := writeEnvLocal(store.envLocalPath(), "sk-from-file"); err != nil {
		t.Fatalf("writeEnvLocal() error = %v", err)
	}
	t.Setenv("OPENAI_API_KEY", "sk-from-env")

	key, source := store.OpenAIAPIKey()
	if key != "sk-from-env" {
		t.Fatalf("key = %q, want env key", key)
	}
	if source != "environment" {
		t.Fatalf("source = %q, want environment", source)
	}
}

func TestBuildResponsesRequestShape(t *testing.T) {
	body, err := buildResponsesRequest("")
	if err != nil {
		t.Fatalf("buildResponsesRequest() error = %v", err)
	}
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if request["model"] != defaultModel {
		t.Fatalf("model = %v, want %s", request["model"], defaultModel)
	}
	if request["input"] == "" {
		t.Fatal("input is empty")
	}
	if request["store"] != false {
		t.Fatalf("store = %v, want false", request["store"])
	}
}

func TestLLMMissingKeyFailsWithoutNetwork(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENAI_API_KEY", "")
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

func TestLLMUsesResponsesAPIShape(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	t.Setenv("OPENAI_API_KEY", "sk-test")
	if _, err := store.SaveSettings(SaveSettingsInput{Provider: "openai", Model: "gpt-test"}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("authorization header not set")
		}
		var request responsesRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Model != "gpt-test" || request.Input == "" || request.Store {
			t.Fatalf("bad request: %+v", request)
		}
		_, _ = w.Write([]byte(`{"output_text":"JD Tailor LLM check"}`))
	}))
	defer server.Close()

	originalURL := openAIResponsesURLForTest(server.URL)
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
