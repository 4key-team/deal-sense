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

type GeminiConfig struct {
	BaseURL     string
	APIKey      string
	Model       string
	SOCKS5Proxy string
}

type Gemini struct {
	config GeminiConfig
	client *http.Client
	clientErr error
	logger *slog.Logger
}

func NewGemini(cfg GeminiConfig, logger *slog.Logger) *Gemini {
	client, err := newHTTPClient(cfg.SOCKS5Proxy)
	if err != nil {
		client = &http.Client{}
	}
	return &Gemini{
		config:    cfg,
		client:    client,
		clientErr: err,
		logger: logger.With("component", "llm.gemini"),
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
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata geminiUsage       `json:"usageMetadata"`
	Error         *apiError         `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
}

func (p *Gemini) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, domain.TokenUsage, error) {
	p.logger.Debug("generating completion", "model", p.config.Model, "prompt_len", len(userPrompt))
	if p.clientErr != nil {
		return "", domain.ZeroTokenUsage(), p.clientErr
	}
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
		return "", domain.ZeroTokenUsage(), fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", domain.ZeroTokenUsage(), fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp geminiResponse
		_ = json.Unmarshal(respBody, &errResp) // best-effort parse of error body
		msg := "unknown error"
		if errResp.Error != nil {
			msg = errResp.Error.Message
		}
		return "", domain.ZeroTokenUsage(), fmt.Errorf("gemini: %s (status %d)", msg, resp.StatusCode)
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return "", domain.ZeroTokenUsage(), wrapParseErr("gemini", err)
	}

	if len(gemResp.Candidates) == 0 {
		return "", domain.ZeroTokenUsage(), fmt.Errorf("gemini: no candidates in response")
	}

	parts := gemResp.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return "", domain.ZeroTokenUsage(), fmt.Errorf("gemini: no parts in response")
	}

	usage := domain.NewTokenUsage(gemResp.UsageMetadata.PromptTokenCount, gemResp.UsageMetadata.CandidatesTokenCount)
	return parts[0].Text, usage, nil
}

func (p *Gemini) CheckConnection(ctx context.Context) error {
	_, _, err := p.GenerateCompletion(ctx, "You are a test.", "Say OK.")
	return err
}

func (p *Gemini) ListModels(_ context.Context) ([]string, error) {
	return nil, nil
}
