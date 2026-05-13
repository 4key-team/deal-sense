package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// OpenAIConfig holds configuration for OpenAI-compatible providers.
type OpenAIConfig struct {
	BaseURL     string
	APIKey      string
	Model       string
	Name        string
	SOCKS5Proxy string
}

// OpenAICompatible works with any OpenAI-compatible API (OpenAI, Groq, Ollama).
type OpenAICompatible struct {
	config OpenAIConfig
	client *http.Client
	clientErr error
	logger *slog.Logger
}

func NewOpenAICompatible(cfg OpenAIConfig, logger *slog.Logger) *OpenAICompatible {
	client, err := newHTTPClient(cfg.SOCKS5Proxy)
	if err != nil {
		client = &http.Client{}
	}
	return &OpenAICompatible{
		config:    cfg,
		client:    client,
		clientErr: err,
		logger: logger.With("component", "llm."+cfg.Name),
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
	Usage   chatUsage    `json:"usage"`
	Error   *apiError    `json:"error,omitempty"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type apiError struct {
	Message string `json:"message"`
}

func (p *OpenAICompatible) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, domain.TokenUsage, error) {
	p.logger.Debug("generating completion", "model", p.config.Model, "prompt_len", len(userPrompt))
	if p.clientErr != nil {
		return "", domain.ZeroTokenUsage(), p.clientErr
	}
	body, _ := json.Marshal(chatRequest{
		Model: p.config.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", domain.ZeroTokenUsage(), fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", domain.ZeroTokenUsage(), fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp chatResponse
		_ = json.Unmarshal(respBody, &errResp) // best-effort parse of error body
		msg := "unknown error"
		if errResp.Error != nil {
			msg = errResp.Error.Message
		}
		return "", domain.ZeroTokenUsage(), fmt.Errorf("llm %s: %s (status %d)", p.config.Name, msg, resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", domain.ZeroTokenUsage(), wrapParseErr(p.config.Name, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", domain.ZeroTokenUsage(), fmt.Errorf("llm %s: no choices in response", p.config.Name)
	}

	usage := domain.NewTokenUsage(chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens)
	p.logger.Debug("completion done",
		"prompt_tokens", usage.PromptTokens(),
		"completion_tokens", usage.CompletionTokens(),
		"response_len", len(chatResp.Choices[0].Message.Content),
	)
	return chatResp.Choices[0].Message.Content, usage, nil
}

func (p *OpenAICompatible) CheckConnection(ctx context.Context) error {
	_, _, err := p.GenerateCompletion(ctx, "You are a test.", "Say OK.")
	return err
}

type modelsResponse struct {
	Data []modelEntry `json:"data"`
}

type modelEntry struct {
	ID string `json:"id"`
}

func (p *OpenAICompatible) ListModels(ctx context.Context) ([]string, error) {
	if p.clientErr != nil {
		return nil, p.clientErr
	}

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
