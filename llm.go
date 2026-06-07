package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var openAIResponsesURL = "https://api.openai.com/v1/responses"

type ToolStatus struct {
	APIKeyConfigured bool   `json:"api_key_configured"`
	APIKeySource     string `json:"api_key_source"`
	EnvLocalPath     string `json:"env_local_path"`
	TectonicStatus   string `json:"tectonic_status"`
	TectonicPath     string `json:"tectonic_path"`
	GeneratedPath    string `json:"generated_path"`
}

type SaveAPIKeyInput struct {
	APIKey string `json:"api_key"`
}

type LLMTestResult struct {
	Success    bool   `json:"success"`
	Model      string `json:"model"`
	Text       string `json:"text"`
	LatencyMS  int64  `json:"latency_ms"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
}

type responsesRequest struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	Instructions    string `json:"instructions,omitempty"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
	Store           bool   `json:"store"`
}

type responsesResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Store) ToolStatus() ToolStatus {
	key, source := s.OpenAIAPIKey()
	tectonic := s.TectonicStatus()
	return ToolStatus{
		APIKeyConfigured: key != "",
		APIKeySource:     source,
		EnvLocalPath:     s.envLocalPath(),
		TectonicStatus:   tectonic.Status,
		TectonicPath:     tectonic.ExecutablePath,
		GeneratedPath:    s.generatedPath,
	}
}

func (s *Store) APIKeyConfigured() bool {
	key, _ := s.OpenAIAPIKey()
	return key != ""
}

func (s *Store) OpenAIAPIKey() (string, string) {
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
		return key, "environment"
	}
	values, err := parseEnvLocal(s.envLocalPath())
	if err != nil {
		return "", ""
	}
	if key := strings.TrimSpace(values["OPENAI_API_KEY"]); key != "" {
		return key, ".env.local"
	}
	return "", ""
}

func (s *Store) SaveAPIKey(input SaveAPIKeyInput) (ToolStatus, error) {
	key := strings.TrimSpace(input.APIKey)
	if key == "" {
		return ToolStatus{}, errors.New("API key is required")
	}
	if strings.ContainsAny(key, "\r\n") {
		return ToolStatus{}, errors.New("API key must be a single line")
	}
	if err := writeEnvLocal(s.envLocalPath(), key); err != nil {
		return ToolStatus{}, err
	}
	if err := s.LogEvent("info", "OpenAI API key saved to .env.local"); err != nil {
		return ToolStatus{}, err
	}
	return s.ToolStatus(), nil
}

func (s *Store) TestLLM(ctx context.Context, client *http.Client) (LLMTestResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	key, _ := s.OpenAIAPIKey()
	settings, err := s.GetSettings()
	if err != nil {
		return LLMTestResult{}, err
	}
	model := configuredModel(settings.Model)
	result := LLMTestResult{Model: model}
	if key == "" {
		result.Error = "OPENAI_API_KEY is missing"
		_ = s.LogEvent("error", "OpenAI smoke test failed: missing API key")
		return result, nil
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	body, err := buildResponsesRequest(model)
	if err != nil {
		return LLMTestResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIResponsesURL, bytes.NewReader(body))
	if err != nil {
		return LLMTestResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	result.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		_ = s.LogEvent("error", "OpenAI smoke test failed: "+err.Error())
		return result, nil
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return LLMTestResult{}, err
	}
	text, apiErr := parseResponsesText(responseBody)
	result.Text = text
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if apiErr == "" {
			apiErr = fmt.Sprintf("OpenAI returned HTTP %d", resp.StatusCode)
		}
		result.Error = apiErr
		_ = s.LogEvent("error", "OpenAI smoke test failed: "+apiErr)
		return result, nil
	}
	result.Success = true
	if result.Text == "" {
		result.Text = "ok"
	}
	_ = s.LogEvent("info", "OpenAI smoke test succeeded")
	return result, nil
}

func configuredModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return defaultModel
	}
	return model
}

func buildResponsesRequest(model string) ([]byte, error) {
	return json.Marshal(responsesRequest{
		Model:           configuredModel(model),
		Input:           "Return exactly: JD Tailor LLM check",
		Instructions:    "Return only the requested check text.",
		MaxOutputTokens: 16,
		Store:           false,
	})
}

func parseResponsesText(body []byte) (string, string) {
	var parsed responsesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err.Error()
	}
	if parsed.Error != nil {
		return "", parsed.Error.Message
	}
	if strings.TrimSpace(parsed.OutputText) != "" {
		return strings.TrimSpace(parsed.OutputText), ""
	}
	for _, output := range parsed.Output {
		for _, content := range output.Content {
			if strings.TrimSpace(content.Text) != "" {
				return strings.TrimSpace(content.Text), ""
			}
		}
	}
	return "", ""
}

func parseEnvLocal(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values, nil
}

func writeEnvLocal(path string, key string) error {
	return os.WriteFile(path, []byte("OPENAI_API_KEY="+key+"\n"), 0o600)
}

func (s *Store) envLocalPath() string {
	return filepath.Join(s.root, ".env.local")
}
