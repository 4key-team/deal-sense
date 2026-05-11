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
