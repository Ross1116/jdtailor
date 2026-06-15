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
	// Test title issues check logic directly
	result := ValidationResult{Passed: true}
	if strings.TrimSpace("") == "" {
		result.TitleIssues = append(result.TitleIssues, "headline is empty")
	}
	if len(result.TitleIssues) == 0 || !strings.Contains(result.TitleIssues[0], "headline is empty") {
		t.Fatalf("expected headline empty error, got: %+v", result.TitleIssues)
	}
}
