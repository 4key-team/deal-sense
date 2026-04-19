package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type AnthropicConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type Anthropic struct {
	config AnthropicConfig
	client *http.Client
}

func NewAnthropic(cfg AnthropicConfig) *Anthropic {
	return &Anthropic{
		config: cfg,
		client: &http.Client{},
	}
}

func (p *Anthropic) Name() string { return "anthropic" }

type anthropicRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system"`
	Messages  []chatMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []anthropicBlock `json:"content"`
	Error   *apiError        `json:"error,omitempty"`
}

type anthropicBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (p *Anthropic) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     p.config.Model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []chatMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp anthropicResponse
		json.Unmarshal(respBody, &errResp)
		msg := "unknown error"
		if errResp.Error != nil {
			msg = errResp.Error.Message
		}
		return "", fmt.Errorf("anthropic: %s (status %d)", msg, resp.StatusCode)
	}

	var antResp anthropicResponse
	if err := json.Unmarshal(respBody, &antResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(antResp.Content) == 0 {
		return "", fmt.Errorf("anthropic: no content blocks in response")
	}

	for _, block := range antResp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("anthropic: no text block in response")
}

func (p *Anthropic) CheckConnection(ctx context.Context) error {
	_, err := p.GenerateCompletion(ctx, "You are a test.", "Say OK.")
	return err
}
