package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
)

func TestMetricsMux_GET_metrics_RendersPrometheusExposition(t *testing.T) {
	coll := metrics.NewCollector()
	coll.Inc("allowlist")
	coll.Inc("allowlist")

	mux := metricsMux(coll)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") || !strings.Contains(ct, "version=0.0.4") {
		t.Errorf("Content-Type = %q, want prometheus exposition", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `dealsense_security_decline_total{kind="allowlist"} 2`) {
		t.Errorf("body missing expected counter line; got:\n%s", body)
	}
}

func TestMetricsMux_GET_healthz_Returns200OK(t *testing.T) {
	mux := metricsMux(metrics.NewCollector())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "ok") {
		t.Errorf("body = %q, want to contain 'ok'", rec.Body.String())
	}
}

func TestMetricsMux_GET_unknown_Returns404(t *testing.T) {
	mux := metricsMux(metrics.NewCollector())
	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
