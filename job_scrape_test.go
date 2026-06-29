package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var allowPrivateJobFetchMu sync.Mutex

func allowPrivateJobFetch(t *testing.T) {
	t.Helper()
	allowPrivateJobFetchMu.Lock()
	old := allowPrivateJobFetchForTests
	allowPrivateJobFetchForTests = true
	t.Cleanup(func() {
		allowPrivateJobFetchForTests = old
		allowPrivateJobFetchMu.Unlock()
	})
}

func TestFetchJobDescriptionExtractsStaticHTML(t *testing.T) {
	allowPrivateJobFetch(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Senior Go Engineer at Acme Labs</title><meta property="og:site_name" content="Acme Labs"></head><body><nav>Home Jobs Login</nav><main class="job-description"><h1>Senior Go Engineer</h1><p>Company: Acme Labs</p><p>We are hiring a backend engineer to build resilient APIs and distributed services.</p><h2>Responsibilities</h2><ul><li>Design Go services for high-volume customer workflows.</li><li>Collaborate with product teams to ship secure, observable platforms.</li></ul><h2>Requirements</h2><ul><li>5+ years building production backend systems.</li><li>Strong SQL, cloud, and API design experience.</li></ul></main><script>secret tracking noise</script></body></html>`))
	}))
	defer server.Close()

	result, err := (&Store{}).FetchJobDescription(context.Background(), FetchJobDescriptionInput{URL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("FetchJobDescription() error = %v", err)
	}
	if result.Company != "Acme Labs" {
		t.Fatalf("Company = %q, want Acme Labs", result.Company)
	}
	if result.Title == "" || !strings.Contains(result.Title, "Senior Go Engineer") {
		t.Fatalf("Title = %q, want Senior Go Engineer", result.Title)
	}
	if !strings.Contains(result.RawText, "Design Go services") || !strings.Contains(result.RawText, "Strong SQL") {
		t.Fatalf("RawText missing job content: %q", result.RawText)
	}
	if strings.Contains(result.RawText, "secret tracking noise") || strings.Contains(result.RawText, "Home Jobs Login") {
		t.Fatalf("RawText included boilerplate/script: %q", result.RawText)
	}
}

func TestFetchJobDescriptionAcceptsPlainText(t *testing.T) {
	allowPrivateJobFetch(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`Company: Plain Co
Title: Platform Engineer

We need a platform engineer to build internal developer tooling, operate cloud infrastructure, improve CI/CD systems, maintain observability, partner with security, write clear documentation, and support reliable production services for multiple product teams.

Requirements include Go, SQL, Kubernetes, API design, incident response, and pragmatic communication across engineering teams.`))
	}))
	defer server.Close()

	result, err := (&Store{}).FetchJobDescription(context.Background(), FetchJobDescriptionInput{URL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("FetchJobDescription() error = %v", err)
	}
	if result.Company != "Plain Co" || result.Title != "Platform Engineer" {
		t.Fatalf("details = %q/%q", result.Company, result.Title)
	}
	if result.Source != "plain_text" {
		t.Fatalf("Source = %q, want plain_text", result.Source)
	}
}

func TestFetchJobDescriptionRejectsNon2xx(t *testing.T) {
	allowPrivateJobFetch(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer server.Close()

	_, err := (&Store{}).FetchJobDescription(context.Background(), FetchJobDescriptionInput{URL: server.URL}, server.Client())
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("error = %v, want HTTP 403", err)
	}
}

func TestFetchJobDescriptionRejectsOversizedBody(t *testing.T) {
	allowPrivateJobFetch(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Repeat("a", jobFetchBodyLimit+2)))
	}))
	defer server.Close()

	_, err := (&Store{}).FetchJobDescription(context.Background(), FetchJobDescriptionInput{URL: server.URL}, server.Client())
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("error = %v, want too large", err)
	}
}

func TestFetchJobDescriptionRejectsInvalidURL(t *testing.T) {
	_, err := (&Store{}).FetchJobDescription(context.Background(), FetchJobDescriptionInput{URL: "ftp://example.com/job"}, nil)
	if err == nil || !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("error = %v, want scheme error", err)
	}
}

func TestFetchJobDescriptionRejectsLoopbackDestination(t *testing.T) {
	_, err := (&Store{}).FetchJobDescription(context.Background(), FetchJobDescriptionInput{URL: "http://127.0.0.1/job"}, nil)
	if err == nil || !strings.Contains(err.Error(), "blocked address") {
		t.Fatalf("error = %v, want blocked address", err)
	}
}

func TestNormalizeJobFetchURLRewritesLinkedInSearchCurrentJobID(t *testing.T) {
	got, err := normalizeJobFetchURL("https://www.linkedin.com/jobs/search/?alertAction=viewjobs&currentJobId=4429612922&distance=25&keywords=software%20developer&originToLandingJobPostings=4429612922%2C4430781758")
	if err != nil {
		t.Fatalf("normalizeJobFetchURL() error = %v", err)
	}
	want := "https://www.linkedin.com/jobs/view/4429612922"
	if got != want {
		t.Fatalf("normalizeJobFetchURL() = %q, want %q", got, want)
	}
}

func TestFetchJobDescriptionRejectsLinkedInLoginGate(t *testing.T) {
	allowPrivateJobFetch(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>LinkedIn Login, Sign in | LinkedIn</title></head><body><main>LinkedIn Login, Sign in | LinkedIn 0 notifications User Agreement Privacy Policy Community Guidelines Cookie Policy Copyright Policy Send Feedback Language LinkedIn Corporation © 2026</main></body></html>`))
	}))
	defer server.Close()

	_, err := (&Store{}).FetchJobDescription(context.Background(), FetchJobDescriptionInput{URL: server.URL}, server.Client())
	if err == nil || !strings.Contains(err.Error(), "login or JavaScript") {
		t.Fatalf("error = %v, want login/JavaScript error", err)
	}
}
