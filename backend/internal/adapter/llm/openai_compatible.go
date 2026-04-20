package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIConfig holds configuration for OpenAI-compatible providers.
type OpenAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Name    string
}

// OpenAICompatible works with any OpenAI-compatible API (OpenAI, Groq, Ollama).
type OpenAICompatible struct {
	config OpenAIConfig
	client *http.Client
}

func NewOpenAICompatible(cfg OpenAIConfig) *OpenAICompatible {
	return &OpenAICompatible{
		config: cfg,
		client: &http.Client{},
	}
}

func (p *OpenAICompatible) Name() string {
	return p.config.Name
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type apiError struct {
	Message string `json:"message"`
}

func (p *OpenAICompatible) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	body, _ := json.Marshal(chatRequest{
		Model: p.config.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp chatResponse
		json.Unmarshal(respBody, &errResp)
		msg := "unknown error"
		if errResp.Error != nil {
			msg = errResp.Error.Message
		}
		return "", fmt.Errorf("llm %s: %s (status %d)", p.config.Name, msg, resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("llm %s: no choices in response", p.config.Name)
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (p *OpenAICompatible) CheckConnection(ctx context.Context) error {
	_, err := p.GenerateCompletion(ctx, "You are a test.", "Say OK.")
	return err
}

type modelsResponse struct {
	Data []modelEntry `json:"data"`
}

type modelEntry struct {
	ID string `json:"id"`
}

func (p *OpenAICompatible) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config.BaseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models: status %d", resp.StatusCode)
	}

	var result modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse models: %w", err)
	}

	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}
	return models, nil
}
