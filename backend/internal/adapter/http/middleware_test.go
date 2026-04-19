package http_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
