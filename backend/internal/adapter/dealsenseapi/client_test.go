package dealsenseapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/dealsenseapi"
	"github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

func TestHTTPClient_AnalyzeTender_Success(t *testing.T) {
	var (
		gotMethod      string
		gotPath        string
		gotAPIKey      string
		gotContentType string
		gotProfile     string
		gotFilename    string
		gotFileBody    string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("X-API-Key")
		gotContentType = r.Header.Get("Content-Type")

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotProfile = r.FormValue("company_profile")
		files := r.MultipartForm.File["files"]
		if len(files) == 1 {
			gotFilename = files[0].Filename
			f, _ := files[0].Open()
			data, _ := io.ReadAll(f)
			gotFileBody = string(data)
			_ = f.Close()
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"verdict": "HIGH",
			"risk":    "low",
			"score":   0.82,
			"summary": "Strong fit",
			"pros":    []map[string]string{{"title": "p1", "desc": "d1"}},
			"cons":    []map[string]string{{"title": "c1", "desc": "d1"}},
			"requirements": []map[string]string{
				{"label": "Лицензия ИБ", "status": "missing"},
			},
			"effort": "2-3 weeks",
		})
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "shared-key", srv.Client())
	resp, err := client.AnalyzeTender(context.Background(), telegram.AnalyzeTenderRequest{
		File:           []byte("PDF DATA"),
		Filename:       "tender.pdf",
		CompanyProfile: "Software dev",
	})
	if err != nil {
		t.Fatalf("AnalyzeTender returned err: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/tender/analyze" {
		t.Errorf("path = %q, want /api/tender/analyze", gotPath)
	}
	if gotAPIKey != "shared-key" {
		t.Errorf("X-API-Key = %q, want shared-key", gotAPIKey)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Errorf("Content-Type = %q, want multipart/form-data prefix", gotContentType)
	}
	if gotProfile != "Software dev" {
		t.Errorf("company_profile = %q, want %q", gotProfile, "Software dev")
	}
	if gotFilename != "tender.pdf" {
		t.Errorf("filename = %q, want tender.pdf", gotFilename)
	}
	if gotFileBody != "PDF DATA" {
		t.Errorf("file body = %q, want PDF DATA", gotFileBody)
	}

	if resp.Verdict != "HIGH" {
		t.Errorf("Verdict = %q", resp.Verdict)
	}
	if resp.Score != 0.82 {
		t.Errorf("Score = %v", resp.Score)
	}
	if len(resp.Pros) != 1 || resp.Pros[0].Title != "p1" {
		t.Errorf("Pros = %+v", resp.Pros)
	}
	if len(resp.Requirements) != 1 || resp.Requirements[0].Status != "missing" {
		t.Errorf("Requirements = %+v", resp.Requirements)
	}
}

func TestHTTPClient_AnalyzeTender_OmitsAPIKeyWhenEmpty(t *testing.T) {
	var gotAPIKey string
	var headerPresent bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, headerPresent = r.Header["X-Api-Key"]
		gotAPIKey = r.Header.Get("X-API-Key")
		_ = json.NewEncoder(w).Encode(map[string]any{"verdict": "LOW", "score": 0.1})
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "", srv.Client())
	_, err := client.AnalyzeTender(context.Background(), telegram.AnalyzeTenderRequest{
		File: []byte("x"), Filename: "t.pdf", CompanyProfile: "p",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if headerPresent || gotAPIKey != "" {
		t.Errorf("X-API-Key should be absent when empty key configured (present=%v, value=%q)", headerPresent, gotAPIKey)
	}
}

func TestHTTPClient_AnalyzeTender_ErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "", srv.Client())
	_, err := client.AnalyzeTender(context.Background(), telegram.AnalyzeTenderRequest{
		File: []byte("x"), Filename: "t.pdf", CompanyProfile: "p",
	})
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want to mention status 401", err)
	}
}

func TestHTTPClient_AnalyzeTender_ErrorOnBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "", srv.Client())
	_, err := client.AnalyzeTender(context.Background(), telegram.AnalyzeTenderRequest{
		File: []byte("x"), Filename: "t.pdf", CompanyProfile: "p",
	})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestHTTPClient_AnalyzeTender_ErrorOnTransport(t *testing.T) {
	// Unresolvable host triggers a Do error path without needing a server.
	client := dealsenseapi.NewHTTPClient("http://127.0.0.1:1", "", nil)
	_, err := client.AnalyzeTender(context.Background(), telegram.AnalyzeTenderRequest{
		File: []byte("x"), Filename: "t.pdf", CompanyProfile: "p",
	})
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
}

func TestHTTPClient_AnalyzeTender_ErrorOnInvalidURL(t *testing.T) {
	// Control character in URL forces http.NewRequestWithContext to fail.
	client := dealsenseapi.NewHTTPClient("http://example.com\x7f", "", http.DefaultClient)
	_, err := client.AnalyzeTender(context.Background(), telegram.AnalyzeTenderRequest{
		File: []byte("x"), Filename: "t.pdf", CompanyProfile: "p",
	})
	if err == nil {
		t.Fatal("expected request-build error, got nil")
	}
}

// failingWriter rejects every Write call so we can exercise multipart
// error branches that bytes.Buffer never triggers.
type failingWriter struct{ afterN int }

func (f *failingWriter) Write(p []byte) (int, error) {
	if f.afterN > 0 {
		f.afterN--
		return len(p), nil
	}
	return 0, io.ErrShortWrite
}

func TestWriteAnalyzeMultipart_PropagatesWriteError(t *testing.T) {
	req := telegram.AnalyzeTenderRequest{File: []byte("x"), Filename: "t.pdf", CompanyProfile: "p"}

	tests := []struct {
		name   string
		writer io.Writer
	}{
		{"fail on first write (WriteField)", &failingWriter{afterN: 0}},
		{"fail on second write (CreateFormFile)", &failingWriter{afterN: 1}},
		{"fail on file body write", &failingWriter{afterN: 2}},
		{"fail on close boundary", &failingWriter{afterN: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dealsenseapi.WriteAnalyzeMultipartForTest(tt.writer, req)
			if err == nil {
				t.Error("expected error from failing writer, got nil")
			}
		})
	}
}

// --- GenerateProposal tests ----------------------------------------------

func TestHTTPClient_GenerateProposal_Success(t *testing.T) {
	var (
		gotMethod        string
		gotPath          string
		gotAPIKey        string
		gotContentType   string
		gotParams        string
		gotTemplateName  string
		gotTemplateBody  string
		gotContextCount  int
		gotContextName   string
		gotContextBody   string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("X-API-Key")
		gotContentType = r.Header.Get("Content-Type")

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotParams = r.FormValue("params")
		if files := r.MultipartForm.File["template"]; len(files) == 1 {
			gotTemplateName = files[0].Filename
			f, _ := files[0].Open()
			data, _ := io.ReadAll(f)
			gotTemplateBody = string(data)
			_ = f.Close()
		}
		ctxFiles := r.MultipartForm.File["context"]
		gotContextCount = len(ctxFiles)
		if gotContextCount == 1 {
			gotContextName = ctxFiles[0].Filename
			f, _ := ctxFiles[0].Open()
			data, _ := io.ReadAll(f)
			gotContextBody = string(data)
			_ = f.Close()
		}

		// Backend returns base64-encoded artifacts + section log.
		_, _ = w.Write([]byte(`{
			"template": "tpl.docx",
			"summary": "ok",
			"mode": "placeholder",
			"sections": [{"title":"Intro","status":"ok","tokens":120}],
			"docx": "UEsDB0RPQ1g=",
			"pdf":  "UEsDB1BERg==",
			"md":   "IyBQcm9wb3NhbA=="
		}`))
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "shared-key", srv.Client())
	resp, err := client.GenerateProposal(context.Background(), telegram.GenerateProposalRequest{
		Template:         []byte("TEMPLATE BODY"),
		TemplateFilename: "tpl.docx",
		ContextFiles: []telegram.ContextFile{
			{Filename: "ctx.pdf", Data: []byte("CTX")},
		},
		Params: map[string]string{"company": "Acme"},
	})
	if err != nil {
		t.Fatalf("GenerateProposal: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/proposal/generate" {
		t.Errorf("path = %q, want /api/proposal/generate", gotPath)
	}
	if gotAPIKey != "shared-key" {
		t.Errorf("X-API-Key = %q, want shared-key", gotAPIKey)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Errorf("Content-Type = %q, want multipart/form-data prefix", gotContentType)
	}
	if gotTemplateName != "tpl.docx" {
		t.Errorf("template filename = %q", gotTemplateName)
	}
	if gotTemplateBody != "TEMPLATE BODY" {
		t.Errorf("template body = %q", gotTemplateBody)
	}
	if gotContextCount != 1 {
		t.Errorf("context files = %d, want 1", gotContextCount)
	}
	if gotContextName != "ctx.pdf" {
		t.Errorf("context filename = %q", gotContextName)
	}
	if gotContextBody != "CTX" {
		t.Errorf("context body = %q", gotContextBody)
	}
	if !strings.Contains(gotParams, `"company":"Acme"`) {
		t.Errorf("params = %q, want JSON with company=Acme", gotParams)
	}

	if resp.Template != "tpl.docx" {
		t.Errorf("Template = %q", resp.Template)
	}
	if resp.Mode != "placeholder" {
		t.Errorf("Mode = %q", resp.Mode)
	}
	if len(resp.Sections) != 1 || resp.Sections[0].Title != "Intro" || resp.Sections[0].Tokens != 120 {
		t.Errorf("Sections = %+v", resp.Sections)
	}
	if string(resp.DOCX) != "PK\x03\x07DOCX" {
		t.Errorf("DOCX decoded = %q, want PK\\x03\\x07DOCX", string(resp.DOCX))
	}
	if string(resp.PDF) != "PK\x03\x07PDF" {
		t.Errorf("PDF decoded = %q", string(resp.PDF))
	}
	if string(resp.MD) != "# Proposal" {
		t.Errorf("MD decoded = %q, want '# Proposal'", string(resp.MD))
	}
}

func TestHTTPClient_GenerateProposal_NoContextFiles(t *testing.T) {
	var gotContextCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(10 << 20)
		gotContextCount = len(r.MultipartForm.File["context"])
		_, _ = w.Write([]byte(`{"template":"x","mode":"clean"}`))
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "", srv.Client())
	resp, err := client.GenerateProposal(context.Background(), telegram.GenerateProposalRequest{
		Template:         []byte("T"),
		TemplateFilename: "t.docx",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if gotContextCount != 0 {
		t.Errorf("context files = %d, want 0", gotContextCount)
	}
	if resp.Mode != "clean" {
		t.Errorf("Mode = %q", resp.Mode)
	}
}

func TestHTTPClient_GenerateProposal_ErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "", srv.Client())
	_, err := client.GenerateProposal(context.Background(), telegram.GenerateProposalRequest{
		Template: []byte("x"), TemplateFilename: "t.docx",
	})
	if err == nil {
		t.Fatal("expected error on 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("err = %v, want to mention 403", err)
	}
}

func TestHTTPClient_GenerateProposal_ErrorOnBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "", srv.Client())
	_, err := client.GenerateProposal(context.Background(), telegram.GenerateProposalRequest{
		Template: []byte("x"), TemplateFilename: "t.docx",
	})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestHTTPClient_GenerateProposal_ErrorOnBadBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"docx":"%%%not-base64%%%"}`))
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL, "", srv.Client())
	_, err := client.GenerateProposal(context.Background(), telegram.GenerateProposalRequest{
		Template: []byte("x"), TemplateFilename: "t.docx",
	})
	if err == nil {
		t.Fatal("expected base64 decode error")
	}
}

func TestNewHTTPClient_NilUsesDefault(t *testing.T) {
	// Pass nil — constructor must substitute http.DefaultClient so AnalyzeTender
	// still works against a real server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"verdict":"LOW"}`))
	}))
	defer srv.Close()

	client := dealsenseapi.NewHTTPClient(srv.URL+"/", "", nil)
	resp, err := client.AnalyzeTender(context.Background(), telegram.AnalyzeTenderRequest{
		File: []byte("x"), Filename: "t.pdf", CompanyProfile: "p",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Verdict != "LOW" {
		t.Errorf("Verdict = %q", resp.Verdict)
	}
}
