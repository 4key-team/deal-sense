package http_test

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/daniil/deal-sense/backend/internal/domain"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

// slowStubLLM lets a test add deterministic latency to GenerateCompletion
// so the SSE handler must emit keep-alive `progress` events before the
// final `result` event lands.
type slowStubLLM struct {
	mu       sync.Mutex
	response string
	usage    domain.TokenUsage
	err      error
	delay    time.Duration
}

func (s *slowStubLLM) GenerateCompletion(_ context.Context, _, _ string) (string, domain.TokenUsage, error) {
	s.mu.Lock()
	d := s.delay
	s.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	return s.response, s.usage, s.err
}
func (s *slowStubLLM) CheckConnection(_ context.Context) error        { return s.err }
func (s *slowStubLLM) ListModels(_ context.Context) ([]string, error) { return nil, nil }
func (s *slowStubLLM) Name() string                                   { return "slow-test" }

func TestHandleGenerateProposalStream(t *testing.T) {
	// Override SSE keep-alive cadence so the test does not need to wait
	// the production 15s tick — see export_test.go for the seam.
	handler.SetSSEKeepAliveForTest(t, 100*time.Millisecond)

	llmResp := `{"params":{"client":"Acme"},"sections":[{"title":"Intro","status":"ai","tokens":50}],"summary":"OK"}`
	llm := &slowStubLLM{response: llmResp, usage: domain.NewTokenUsage(100, 200), delay: 350 * time.Millisecond}
	tmpl := &stubTemplateEngine{result: []byte("filled-docx")}

	h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/proposal/generate-stream", h.HandleGenerateProposalStream)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("template", "offer.docx")
	fw.Write([]byte("template"))
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/proposal/generate-stream", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream*", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Errorf("Cache-Control = %q, want to contain no-cache", cc)
	}

	body, err := readAllWithTimeout(resp.Body, 3*time.Second)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}

	progressCount := strings.Count(body, "event: progress")
	if progressCount < 1 {
		t.Errorf("expected at least one `event: progress` keep-alive, got 0 (body: %q)", body)
	}

	if !strings.Contains(body, "event: result") {
		t.Errorf("expected final `event: result`, body: %q", body)
	}
	// The result payload should carry the same JSON shape as /generate.
	if !strings.Contains(body, `"summary":"OK"`) {
		t.Errorf("expected proposal summary in result event, body: %q", body)
	}
}

func TestHandleGenerateProposalStream_LLMError(t *testing.T) {
	handler.SetSSEKeepAliveForTest(t, 100*time.Millisecond)

	llm := &slowStubLLM{err: errors.New("provider quota exceeded"), delay: 50 * time.Millisecond}
	tmpl := &stubTemplateEngine{result: []byte("nope")}
	h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/proposal/generate-stream", h.HandleGenerateProposalStream)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("template", "offer.docx")
	fw.Write([]byte("template"))
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/proposal/generate-stream", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	body, err := readAllWithTimeout(resp.Body, 3*time.Second)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}

	if !strings.Contains(body, "event: error") {
		t.Errorf("expected `event: error` for LLM failure, body: %q", body)
	}
	if strings.Contains(body, "event: result") {
		t.Errorf("must NOT emit `event: result` when LLM errored, body: %q", body)
	}
}

// readAllWithTimeout reads bytes from r until EOF or until the deadline,
// whichever comes first. It returns whatever was buffered so tests can
// assert against partial streams.
func readAllWithTimeout(r interface {
	Read([]byte) (int, error)
}, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var sb strings.Builder
	buf := make([]byte, 1024)
	for time.Now().Before(deadline) {
		n, err := r.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return sb.String(), nil
}
