package http_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apphttp "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
)

func TestMetricsHandler_ContentType(t *testing.T) {
	c := metrics.NewCollector()
	h := apphttp.MetricsHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := rec.Header().Get("Content-Type")
	want := "text/plain; version=0.0.4; charset=utf-8"
	if got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestMetricsHandler_BodyIsPrometheusFormat(t *testing.T) {
	c := metrics.NewCollector()
	c.IncRequest("/api/tender/analyze", "200")
	c.SetEndpointRisk("/api/proposal/generate", "MODIFY")

	h := apphttp.MetricsHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	out := string(body)

	for _, marker := range []string{
		"# TYPE dealsense_requests_total counter",
		`dealsense_requests_total{path="/api/tender/analyze",status="200"} 1`,
		"# TYPE dealsense_endpoint_risk gauge",
		`dealsense_endpoint_risk{path="/api/proposal/generate",level="MODIFY"} 1`,
	} {
		if !strings.Contains(out, marker) {
			t.Errorf("body missing %q\n--- got ---\n%s", marker, out)
		}
	}
}
