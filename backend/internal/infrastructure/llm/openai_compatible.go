package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	applicationmodel "kubeclaw/backend/internal/application/model"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatInput struct {
	Model    applicationmodel.ResolvedRecord
	Messages []Message
}

type ChatResult struct {
	Model   string `json:"model"`
	Content string `json:"content"`
}

type OpenAICompatibleClient struct {
	httpClient *http.Client
}

func NewOpenAICompatibleClient() *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *OpenAICompatibleClient) TestConnection(ctx context.Context, model applicationmodel.ResolvedRecord) (*applicationmodel.TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	_, err := c.Chat(testCtx, ChatInput{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: "Return OK only."},
			{Role: "user", Content: "health check"},
		},
	})
	if err != nil {
		return nil, err
	}

	return &applicationmodel.TestResult{
		Reachable: true,
		Model:     model.Model,
		Provider:  model.Provider,
		Message:   "model endpoint is reachable",
		CheckedAt: time.Now(),
	}, nil
}

func (c *OpenAICompatibleClient) Chat(ctx context.Context, input ChatInput) (*ChatResult, error) {
	if strings.TrimSpace(input.Model.Model) == "" {
		return nil, fmt.Errorf("model name is empty")
	}
	if len(input.Messages) == 0 {
		return nil, fmt.Errorf("chat messages are empty")
	}

	requestBody := map[string]any{
		"model":       input.Model.Model,
		"messages":    input.Messages,
		"temperature": input.Model.Temperature,
		"top_p":       input.Model.TopP,
		"stream":      false,
	}
	if input.Model.MaxTokens > 0 {
		requestBody["max_tokens"] = input.Model.MaxTokens
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal llm request: %w", err)
	}

	url := strings.TrimRight(defaultBaseURL(input.Model.BaseURL), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(input.Model.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+input.Model.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request llm endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read llm response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Model   string `json:"model"`
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode llm response: %w", err)
	}
	if len(envelope.Choices) == 0 {
		return nil, fmt.Errorf("llm response has no choices")
	}

	return &ChatResult{
		Model:   envelope.Model,
		Content: strings.TrimSpace(envelope.Choices[0].Message.Content),
	}, nil
}

func defaultBaseURL(value string) string {
	if strings.TrimSpace(value) == "" {
		return "https://api.openai.com/v1"
	}
	return value
}
