// Package dealsenseapi implements the bot's APIClient port via HTTP calls
// to the Deal Sense backend.
package dealsenseapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// HTTPClient calls the Deal Sense HTTP API with optional X-API-Key auth.
type HTTPClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewHTTPClient constructs an HTTPClient. Pass an empty apiKey when the
// backend is running in open-access mode.
func NewHTTPClient(baseURL, apiKey string, c *http.Client) *HTTPClient {
	if c == nil {
		c = http.DefaultClient
	}
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    c,
	}
}

// AnalyzeTender POSTs the file to /api/tender/analyze as multipart form and
// decodes the JSON response into AnalyzeTenderResponse. The multipart body
// is built over a bytes.Buffer whose Write never fails — so the writer
// errors below are dead code in production, exercised only by
// writeAnalyzeMultipart's own tests.
func (c *HTTPClient) AnalyzeTender(ctx context.Context, req telegram.AnalyzeTenderRequest) (*telegram.AnalyzeTenderResponse, error) {
	body := &bytes.Buffer{}
	contentType, err := writeAnalyzeMultipart(body, req)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/tender/analyze", body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", contentType)
	if c.apiKey != "" {
		httpReq.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("backend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	var raw struct {
		Verdict      string  `json:"verdict"`
		Risk         string  `json:"risk"`
		Score        float64 `json:"score"`
		Summary      string  `json:"summary"`
		Pros         []struct{ Title, Desc string } `json:"pros"`
		Cons         []struct{ Title, Desc string } `json:"cons"`
		Requirements []struct{ Label, Status string } `json:"requirements"`
		Effort       string `json:"effort"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	out := &telegram.AnalyzeTenderResponse{
		Verdict: raw.Verdict,
		Risk:    raw.Risk,
		Score:   raw.Score,
		Summary: raw.Summary,
		Effort:  raw.Effort,
	}
	for _, p := range raw.Pros {
		out.Pros = append(out.Pros, telegram.ProConItem{Title: p.Title, Desc: p.Desc})
	}
	for _, c := range raw.Cons {
		out.Cons = append(out.Cons, telegram.ProConItem{Title: c.Title, Desc: c.Desc})
	}
	for _, r := range raw.Requirements {
		out.Requirements = append(out.Requirements, telegram.RequirementItem{Label: r.Label, Status: r.Status})
	}
	return out, nil
}

// GenerateProposal stub for the RED step. The real impl lands in GREEN.
func (c *HTTPClient) GenerateProposal(ctx context.Context, req telegram.GenerateProposalRequest) (*telegram.GenerateProposalResponse, error) {
	return nil, nil
}

func writeAnalyzeMultipart(out io.Writer, req telegram.AnalyzeTenderRequest) (string, error) {
	w := multipart.NewWriter(out)

	if err := w.WriteField("company_profile", req.CompanyProfile); err != nil {
		return "", err
	}
	fw, err := w.CreateFormFile("files", req.Filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(req.File); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return w.FormDataContentType(), nil
}
