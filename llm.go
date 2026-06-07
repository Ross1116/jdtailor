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
var openRouterChatCompletionsURL = "https://openrouter.ai/api/v1/chat/completions"

type ToolStatus struct {
	APIKeyConfigured bool   `json:"api_key_configured"`
	APIKeySource     string `json:"api_key_source"`
	EnvLocalPath     string `json:"env_local_path"`
	TectonicStatus   string `json:"tectonic_status"`
	TectonicPath     string `json:"tectonic_path"`
	GeneratedPath    string `json:"generated_path"`
}

type SaveAPIKeyInput struct {
	APIKey   string `json:"api_key"`
	Provider string `json:"provider"`
}

type LLMTestResult struct {
	Success    bool   `json:"success"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Text       string `json:"text"`
	LatencyMS  int64  `json:"latency_ms"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
}

type openAIResponsesRequest struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	Instructions    string `json:"instructions,omitempty"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
	Store           bool   `json:"store"`
}

type openAIResponsesResponse struct {
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

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Store) ToolStatus() ToolStatus {
	settings, err := s.GetSettings()
	provider := defaultProvider
	if err == nil {
		provider = settings.Provider
	}
	key, source := s.APIKeyForProvider(provider)
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

func (s *Store) APIKeyConfigured(provider string) bool {
	key, _ := s.APIKeyForProvider(provider)
	return key != ""
}

func (s *Store) APIKeyForProvider(provider string) (string, string) {
	envName := apiKeyEnvName(provider)
	if key := strings.TrimSpace(os.Getenv(envName)); key != "" {
		return key, envName + " environment"
	}
	values, err := parseEnvLocal(s.envLocalPath())
	if err != nil {
		return "", ""
	}
	if key := strings.TrimSpace(values[envName]); key != "" {
		return key, envName + " .env.local"
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
	provider := configuredProvider(input.Provider)
	if strings.TrimSpace(input.Provider) == "" {
		settings, err := s.GetSettings()
		if err != nil {
			return ToolStatus{}, err
		}
		provider = settings.Provider
	}
	envName := apiKeyEnvName(provider)
	if err := writeEnvLocalValue(s.envLocalPath(), envName, key); err != nil {
		return ToolStatus{}, err
	}
	if err := s.LogEvent("info", provider+" API key saved to .env.local"); err != nil {
		return ToolStatus{}, err
	}
	return s.ToolStatus(), nil
}

func (s *Store) TestLLM(ctx context.Context, client *http.Client) (LLMTestResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	settings, err := s.GetSettings()
	if err != nil {
		return LLMTestResult{}, err
	}
	provider := configuredProvider(settings.Provider)
	model := configuredModel(provider, settings.Model)
	result := LLMTestResult{Provider: provider, Model: model}
	key, _ := s.APIKeyForProvider(provider)
	if key == "" {
		result.Error = apiKeyEnvName(provider) + " is missing"
		_ = s.LogEvent("error", provider+" smoke test failed: missing API key")
		return result, nil
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	switch provider {
	case "openai":
		return s.testOpenAIResponses(ctx, client, key, result)
	default:
		return s.testOpenRouterChat(ctx, client, key, result)
	}
}

func (s *Store) testOpenRouterChat(ctx context.Context, client *http.Client, key string, result LLMTestResult) (LLMTestResult, error) {
	body, err := buildChatCompletionRequest(result.Model)
	if err != nil {
		return LLMTestResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterChatCompletionsURL, bytes.NewReader(body))
	if err != nil {
		return LLMTestResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpenRouter-Title", "JD Tailor")

	start := time.Now()
	resp, err := client.Do(req)
	result.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		_ = s.LogEvent("error", "openrouter smoke test failed: "+err.Error())
		return result, nil
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return LLMTestResult{}, err
	}
	text, apiErr := parseChatCompletionText(responseBody)
	result.Text = text
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if apiErr == "" {
			apiErr = fmt.Sprintf("OpenRouter returned HTTP %d", resp.StatusCode)
		}
		result.Error = apiErr
		_ = s.LogEvent("error", "openrouter smoke test failed: "+apiErr)
		return result, nil
	}
	result.Success = true
	if result.Text == "" {
		result.Text = "ok"
	}
	_ = s.LogEvent("info", "openrouter smoke test succeeded")
	return result, nil
}

func (s *Store) testOpenAIResponses(ctx context.Context, client *http.Client, key string, result LLMTestResult) (LLMTestResult, error) {
	body, err := buildResponsesRequest(result.Model)
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
		_ = s.LogEvent("error", "openai smoke test failed: "+err.Error())
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
		_ = s.LogEvent("error", "openai smoke test failed: "+apiErr)
		return result, nil
	}
	result.Success = true
	if result.Text == "" {
		result.Text = "ok"
	}
	_ = s.LogEvent("info", "openai smoke test succeeded")
	return result, nil
}

func configuredProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return defaultProvider
	}
	switch provider {
	case "openrouter", "openai":
		return provider
	default:
		return defaultProvider
	}
}

func configuredModel(provider string, model string) string {
	model = strings.TrimSpace(model)
	if model != "" {
		return model
	}
	if configuredProvider(provider) == "openai" {
		return "gpt-5.4-mini"
	}
	return defaultModel
}

func apiKeyEnvName(provider string) string {
	if configuredProvider(provider) == "openai" {
		return "OPENAI_API_KEY"
	}
	return "OPENROUTER_API_KEY"
}

func buildChatCompletionRequest(model string) ([]byte, error) {
	return json.Marshal(chatCompletionRequest{
		Model: configuredModel(defaultProvider, model),
		Messages: []chatMessage{
			{Role: "system", Content: "Return only the requested check text."},
			{Role: "user", Content: "Return exactly: JD Tailor LLM check"},
		},
		MaxTokens:   16,
		Temperature: 0,
	})
}

func buildResponsesRequest(model string) ([]byte, error) {
	return json.Marshal(openAIResponsesRequest{
		Model:           configuredModel("openai", model),
		Input:           "Return exactly: JD Tailor LLM check",
		Instructions:    "Return only the requested check text.",
		MaxOutputTokens: 16,
		Store:           false,
	})
}

func parseChatCompletionText(body []byte) (string, string) {
	var parsed chatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err.Error()
	}
	if parsed.Error != nil {
		return "", parsed.Error.Message
	}
	if len(parsed.Choices) == 0 {
		return "", ""
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), ""
}

func parseResponsesText(body []byte) (string, string) {
	var parsed openAIResponsesResponse
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

func writeEnvLocalValue(path string, key string, value string) error {
	values, err := parseEnvLocal(path)
	if err != nil {
		return err
	}
	values[key] = value
	order := []string{"OPENROUTER_API_KEY", "OPENAI_API_KEY"}
	var builder strings.Builder
	wrote := map[string]bool{}
	for _, name := range order {
		if val, ok := values[name]; ok && strings.TrimSpace(val) != "" {
			builder.WriteString(name)
			builder.WriteString("=")
			builder.WriteString(strings.TrimSpace(val))
			builder.WriteString("\n")
			wrote[name] = true
		}
	}
	for name, val := range values {
		if wrote[name] || strings.TrimSpace(val) == "" {
			continue
		}
		builder.WriteString(name)
		builder.WriteString("=")
		builder.WriteString(strings.TrimSpace(val))
		builder.WriteString("\n")
	}
	return os.WriteFile(path, []byte(builder.String()), 0o600)
}

func (s *Store) envLocalPath() string {
	return filepath.Join(s.root, ".env.local")
}
