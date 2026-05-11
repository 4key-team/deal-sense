// Package dealsenseapi implements the bot's APIClient port via HTTP calls
// to the Deal Sense backend.
package dealsenseapi

import (
	"bytes"
	"context"
	"encoding/base64"
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

// GenerateProposal POSTs the template + context files + params JSON to
// /api/proposal/generate and decodes the JSON response, base64-decoding
// docx/pdf/md payloads into raw bytes.
func (c *HTTPClient) GenerateProposal(ctx context.Context, req telegram.GenerateProposalRequest) (*telegram.GenerateProposalResponse, error) {
	body := &bytes.Buffer{}
	contentType, err := writeGenerateMultipart(body, req)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/proposal/generate", body)
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
		Template string `json:"template"`
		Summary  string `json:"summary"`
		Mode     string `json:"mode"`
		Sections []struct {
			Title  string `json:"title"`
			Status string `json:"status"`
			Tokens int    `json:"tokens"`
		} `json:"sections"`
		DOCX string `json:"docx"`
		PDF  string `json:"pdf"`
		MD   string `json:"md"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	out := &telegram.GenerateProposalResponse{
		Template: raw.Template,
		Summary:  raw.Summary,
		Mode:     raw.Mode,
	}
	for _, s := range raw.Sections {
		out.Sections = append(out.Sections, telegram.GeneratedSection{
			Title: s.Title, Status: s.Status, Tokens: s.Tokens,
		})
	}
	if raw.DOCX != "" {
		out.DOCX, err = base64.StdEncoding.DecodeString(raw.DOCX)
		if err != nil {
			return nil, fmt.Errorf("decode docx base64: %w", err)
		}
	}
	if raw.PDF != "" {
		out.PDF, err = base64.StdEncoding.DecodeString(raw.PDF)
		if err != nil {
			return nil, fmt.Errorf("decode pdf base64: %w", err)
		}
	}
	if raw.MD != "" {
		// Backend returns MD as plain string (not base64); be defensive
		// and try base64 first, fall back to raw.
		if decoded, err := base64.StdEncoding.DecodeString(raw.MD); err == nil {
			out.MD = decoded
		} else {
			out.MD = []byte(raw.MD)
		}
	}
	return out, nil
}

// writeGenerateMultipart writes template + context files + params JSON to
// the multipart body. Mirrors backend handler_proposal.go field names.
func writeGenerateMultipart(out io.Writer, req telegram.GenerateProposalRequest) (string, error) {
	w := multipart.NewWriter(out)

	fw, err := w.CreateFormFile("template", req.TemplateFilename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(req.Template); err != nil {
		return "", err
	}

	for _, cf := range req.ContextFiles {
		cfw, err := w.CreateFormFile("context", cf.Filename)
		if err != nil {
			return "", err
		}
		if _, err := cfw.Write(cf.Data); err != nil {
			return "", err
		}
	}

	if len(req.Params) > 0 {
		paramsJSON, err := json.Marshal(req.Params)
		if err != nil {
			return "", fmt.Errorf("marshal params: %w", err)
		}
		if err := w.WriteField("params", string(paramsJSON)); err != nil {
			return "", err
		}
	}

	if err := w.Close(); err != nil {
		return "", err
	}
	return w.FormDataContentType(), nil
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
