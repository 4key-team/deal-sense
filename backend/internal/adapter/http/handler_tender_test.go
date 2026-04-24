package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

type stubParser struct {
	content string
	err     error
}

func (s *stubParser) Parse(_ context.Context, _ string, _ []byte) (string, error) {
	return s.content, s.err
}
func (s *stubParser) Supports(_ domain.FileType) bool { return true }

func makeMultipartRequest(t *testing.T, files map[string][]byte, fields map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for name, data := range files {
		fw, err := w.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(data)
	}
	for k, v := range fields {
		w.WriteField(k, v)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/tender/analyze", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestHandleAnalyzeTender(t *testing.T) {
	tests := []struct {
		name       string
		llm        *stubLLM
		parser     *stubParser
		files      map[string][]byte
		fields     map[string]string
		useRawBody bool   // when true, send raw body instead of multipart
		rawBody    string // raw body content
		wantStatus int
	}{
		{
			name:       "successful analysis",
			llm:        &stubLLM{response: `{"verdict":"go","risk":"low","score":82,"summary":"Good fit"}`, name: "test"},
			parser:     &stubParser{content: "tender requirements"},
			files:      map[string][]byte{"spec.pdf": []byte("pdf data")},
			fields:     map[string]string{"company_profile": "We build software"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing files",
			llm:        &stubLLM{name: "test"},
			parser:     &stubParser{},
			files:      nil,
			fields:     map[string]string{"company_profile": "We build software"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty company profile uses default",
			llm:        &stubLLM{name: "test", response: `{"verdict":"go","risk":"low","score":75,"summary":"ok","pros":[],"cons":[],"requirements":[],"effort":"~10h"}`},
			parser:     &stubParser{content: "text"},
			files:      map[string][]byte{"spec.pdf": []byte("pdf data")},
			fields:     nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unsupported file type",
			llm:        &stubLLM{name: "test"},
			parser:     &stubParser{content: "text"},
			files:      map[string][]byte{"spec.txt": []byte("data")},
			fields:     map[string]string{"company_profile": "Acme"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "llm error returns 500",
			llm:        &stubLLM{response: "not json", name: "test"},
			parser:     &stubParser{content: "requirements"},
			files:      map[string][]byte{"spec.pdf": []byte("data")},
			fields:     map[string]string{"company_profile": "Acme"},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid multipart",
			llm:        &stubLLM{name: "test"},
			parser:     &stubParser{},
			useRawBody: true,
			rawBody:    "not multipart",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewHandler(tt.llm, nil, tt.parser, nil, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

			var req *http.Request
			if tt.useRawBody {
				req = httptest.NewRequest(http.MethodPost, "/api/tender/analyze", strings.NewReader(tt.rawBody))
				req.Header.Set("Content-Type", "multipart/form-data; boundary=bad")
			} else if tt.files == nil {
				var buf bytes.Buffer
				w := multipart.NewWriter(&buf)
				for k, v := range tt.fields {
					w.WriteField(k, v)
				}
				w.Close()
				req = httptest.NewRequest(http.MethodPost, "/api/tender/analyze", &buf)
				req.Header.Set("Content-Type", w.FormDataContentType())
			} else {
				req = makeMultipartRequest(t, tt.files, tt.fields)
			}

			rec := httptest.NewRecorder()
			h.HandleAnalyzeTender(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandleAnalyzeTender_FullResponse(t *testing.T) {
	llmResp := `{
		"verdict":"go","risk":"medium","score":72,"summary":"Decent fit",
		"pros":[{"title":"Strong team","desc":"10 engineers"}],
		"cons":[{"title":"No ISO","desc":"Missing cert"}],
		"requirements":[{"label":"Go experience","status":"met"},{"label":"ISO 27001","status":"partial"}],
		"effort":"~40 hours"
	}`
	llm := &stubLLM{response: llmResp, name: "test"}
	p := &stubParser{content: "requirements text"}
	h := handler.NewHandler(llm, nil, p, nil, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

	req := makeMultipartRequest(t, map[string][]byte{"spec.pdf": []byte("data")}, map[string]string{"company_profile": "Acme"})
	rec := httptest.NewRecorder()
	h.HandleAnalyzeTender(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Check all fields present
	if resp["verdict"] != "go" {
		t.Errorf("verdict = %v, want go", resp["verdict"])
	}
	if resp["risk"] != "medium" {
		t.Errorf("risk = %v, want medium", resp["risk"])
	}
	if resp["effort"] != "~40 hours" {
		t.Errorf("effort = %v, want ~40 hours", resp["effort"])
	}
	pros := resp["pros"].([]any)
	if len(pros) != 1 {
		t.Errorf("pros len = %d, want 1", len(pros))
	}
	cons := resp["cons"].([]any)
	if len(cons) != 1 {
		t.Errorf("cons len = %d, want 1", len(cons))
	}
	reqs := resp["requirements"].([]any)
	if len(reqs) != 2 {
		t.Errorf("requirements len = %d, want 2", len(reqs))
	}

	usage, ok := resp["usage"].(map[string]any)
	if !ok {
		t.Error("expected usage object")
	} else if usage["total_tokens"] == nil {
		t.Error("expected total_tokens in usage")
	}
}

func TestHandleAnalyzeTender_WithDocx(t *testing.T) {
	llmResp := `{"verdict":"no-go","risk":"high","score":30,"summary":"Bad fit"}`
	llm := &stubLLM{response: llmResp, name: "test"}
	p := &stubParser{content: "text"}
	h := handler.NewHandler(llm, nil, p, nil, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("files", "spec.docx")
	fw.Write([]byte("docx data"))
	w.WriteField("company_profile", "Acme Corp")
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/tender/analyze", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	h.HandleAnalyzeTender(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
