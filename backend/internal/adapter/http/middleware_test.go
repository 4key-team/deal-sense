package http_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

func TestCORS(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("sets CORS headers", func(t *testing.T) {
		h := handler.CORS("http://localhost:5173", inner)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
			t.Errorf("Allow-Origin = %q, want http://localhost:5173", got)
		}
	})

	t.Run("handles preflight OPTIONS", func(t *testing.T) {
		h := handler.CORS("*", inner)
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	h := handler.Logger(logger, inner)
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "POST") {
		t.Errorf("log missing method, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "/api/test") {
		t.Errorf("log missing path, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "201") {
		t.Errorf("log missing status 201, got: %s", logOutput)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	tests := []struct {
		name       string
		expected   string
		sent       string
		sendHeader bool
		wantStatus int
		wantBody   string
	}{
		{
			name:       "empty expected key acts as passthrough",
			expected:   "",
			sendHeader: false,
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:       "missing header when key configured returns 401",
			expected:   "secret-key",
			sendHeader: false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong header value returns 401",
			expected:   "secret-key",
			sent:       "wrong-key",
			sendHeader: true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "matching header passes through",
			expected:   "secret-key",
			sent:       "secret-key",
			sendHeader: true,
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.APIKeyAuth(tt.expected, nil, inner)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.sendHeader {
				req.Header.Set("X-API-Key", tt.sent)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want to contain %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRateLimit_AllowsUpToBurst(t *testing.T) {
	called := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	})
	h := handler.RateLimit(1, 5, nil, inner) // burst 5, rps 1

	for i := range 5 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("burst call %d: status = %d, want 200", i, rec.Code)
		}
	}
	if called != 5 {
		t.Errorf("inner called %d times, want 5", called)
	}
}

func TestRateLimit_BlocksAfterBurst(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := handler.RateLimit(1, 2, nil, inner) // burst 2

	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}

	// 3rd call should be blocked.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Errorf("missing Retry-After header")
	}
}

func TestRateLimit_IsolatesByRemoteAddr(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := handler.RateLimit(1, 1, nil, inner) // burst 1 per key

	// Exhaust client A.
	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.RemoteAddr = "1.1.1.1:1000"
	recA := httptest.NewRecorder()
	h.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("A first call status = %d", recA.Code)
	}

	// Client A's second call blocked.
	recA2 := httptest.NewRecorder()
	h.ServeHTTP(recA2, reqA)
	if recA2.Code != http.StatusTooManyRequests {
		t.Errorf("A second call status = %d, want 429", recA2.Code)
	}

	// Client B fresh — still gets a token.
	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.RemoteAddr = "2.2.2.2:1000"
	recB := httptest.NewRecorder()
	h.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Errorf("B first call status = %d, want 200", recB.Code)
	}
}

func TestRateLimit_DisabledWhenRPSZero(t *testing.T) {
	called := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	})
	h := handler.RateLimit(0, 0, nil, inner) // disabled

	for range 100 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.1.1.1:1000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 when disabled", rec.Code)
		}
	}
	if called != 100 {
		t.Errorf("inner called %d times, want 100", called)
	}
}

// --- security_decline_total counter wiring ---

type fakeDeclineCounter struct {
	mu   sync.Mutex
	kind []string
}

func (f *fakeDeclineCounter) Inc(kind string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kind = append(f.kind, kind)
}

func (f *fakeDeclineCounter) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.kind))
	copy(out, f.kind)
	return out
}

func TestAPIKeyAuth_IncrementsCounterOn401(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	counter := &fakeDeclineCounter{}
	h := handler.APIKeyAuth("secret-key", counter, inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// no header → 401
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	got := counter.snapshot()
	if len(got) != 1 || got[0] != "api_key" {
		t.Errorf("counter calls = %v, want [api_key]", got)
	}
}

func TestAPIKeyAuth_NoIncrementOnSuccess(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	counter := &fakeDeclineCounter{}
	h := handler.APIKeyAuth("secret-key", counter, inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "secret-key")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := counter.snapshot(); len(got) != 0 {
		t.Errorf("counter calls = %v, want [] (no decline on success)", got)
	}
}

func TestRateLimit_IncrementsCounterOn429(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	counter := &fakeDeclineCounter{}
	h := handler.RateLimit(1, 1, counter, inner)

	// First call consumes the bucket.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Second call blocked.
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec2.Code)
	}
	got := counter.snapshot()
	if len(got) != 1 || got[0] != "rate_limit" {
		t.Errorf("counter calls = %v, want [rate_limit]", got)
	}
}

func TestRateLimit_NoIncrementOnAllowed(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	counter := &fakeDeclineCounter{}
	h := handler.RateLimit(10, 5, counter, inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := counter.snapshot(); len(got) != 0 {
		t.Errorf("counter calls = %v, want []", got)
	}
}

func TestRecover(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something broke")
	})

	h := handler.Recover(logger, inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Errorf("log missing panic message, got: %s", buf.String())
	}
}
