package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GeminiConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type Gemini struct {
	config GeminiConfig
	client *http.Client
}

func NewGemini(cfg GeminiConfig) *Gemini {
	return &Gemini{
		config: cfg,
		client: &http.Client{},
	}
}

func (p *Gemini) Name() string { return "gemini" }

type geminiRequest struct {
	SystemInstruction geminiContent   `json:"systemInstruction"`
	Contents          []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *apiError         `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

func (p *Gemini) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := geminiRequest{
		SystemInstruction: geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		},
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: userPrompt}}},
		},
	}

	body, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.config.BaseURL, p.config.Model, p.config.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp geminiResponse
		json.Unmarshal(respBody, &errResp)
		msg := "unknown error"
		if errResp.Error != nil {
			msg = errResp.Error.Message
		}
		return "", fmt.Errorf("gemini: %s (status %d)", msg, resp.StatusCode)
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(gemResp.Candidates) == 0 {
		return "", fmt.Errorf("gemini: no candidates in response")
	}

	parts := gemResp.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return "", fmt.Errorf("gemini: no parts in response")
	}

	return parts[0].Text, nil
}

func (p *Gemini) CheckConnection(ctx context.Context) error {
	_, err := p.GenerateCompletion(ctx, "You are a test.", "Say OK.")
	return err
}
