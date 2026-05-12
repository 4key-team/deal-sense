package metrics_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
)

func TestCollector_Empty_RendersHelpAndType(t *testing.T) {
	c := metrics.NewCollector()
	var buf bytes.Buffer
	if _, err := c.Render(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, marker := range []string{
		"# HELP dealsense_requests_total",
		"# TYPE dealsense_requests_total counter",
		"# HELP dealsense_llm_calls_total",
		"# TYPE dealsense_llm_calls_total counter",
		"# HELP dealsense_endpoint_risk",
		"# TYPE dealsense_endpoint_risk gauge",
		"# HELP dealsense_security_decline_total",
		"# TYPE dealsense_security_decline_total counter",
	} {
		if !strings.Contains(out, marker) {
			t.Errorf("Render missing %q\n--- got ---\n%s", marker, out)
		}
	}
}

func TestCollector_IncRequest_RendersCounter(t *testing.T) {
	c := metrics.NewCollector()
	c.IncRequest("/api/tender/analyze", "200")
	c.IncRequest("/api/tender/analyze", "200")
	c.IncRequest("/api/tender/analyze", "500")

	var buf bytes.Buffer
	if _, err := c.Render(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `dealsense_requests_total{path="/api/tender/analyze",status="200"} 2`) {
		t.Errorf("Render missing analyze 200 = 2 counter\n--- got ---\n%s", out)
	}
	if !strings.Contains(out, `dealsense_requests_total{path="/api/tender/analyze",status="500"} 1`) {
		t.Errorf("Render missing analyze 500 = 1 counter\n--- got ---\n%s", out)
	}
}

func TestCollector_IncLLMCall_RendersCounter(t *testing.T) {
	c := metrics.NewCollector()
	c.IncLLMCall("anthropic", "ok")
	c.IncLLMCall("anthropic", "ok")
	c.IncLLMCall("openai", "error")

	var buf bytes.Buffer
	_, _ = c.Render(&buf)
	out := buf.String()
	if !strings.Contains(out, `dealsense_llm_calls_total{provider="anthropic",status="ok"} 2`) {
		t.Errorf("Render missing anthropic ok 2\n--- got ---\n%s", out)
	}
	if !strings.Contains(out, `dealsense_llm_calls_total{provider="openai",status="error"} 1`) {
		t.Errorf("Render missing openai error 1\n--- got ---\n%s", out)
	}
}

func TestCollector_Inc_SecurityDecline(t *testing.T) {
	c := metrics.NewCollector()
	c.Inc("api_key")
	c.Inc("api_key")
	c.Inc("rate_limit")
	c.Inc("allowlist")
	c.Inc("llm_parse_error")

	var buf bytes.Buffer
	_, _ = c.Render(&buf)
	out := buf.String()
	for _, want := range []string{
		`dealsense_security_decline_total{kind="api_key"} 2`,
		`dealsense_security_decline_total{kind="rate_limit"} 1`,
		`dealsense_security_decline_total{kind="allowlist"} 1`,
		`dealsense_security_decline_total{kind="llm_parse_error"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Render missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestCollector_SetEndpointRisk_RendersGauge(t *testing.T) {
	c := metrics.NewCollector()
	c.SetEndpointRisk("/api/tender/analyze", "SAFE_READ")
	c.SetEndpointRisk("/api/proposal/generate", "MODIFY")

	var buf bytes.Buffer
	_, _ = c.Render(&buf)
	out := buf.String()
	if !strings.Contains(out, `dealsense_endpoint_risk{path="/api/tender/analyze",level="SAFE_READ"} 1`) {
		t.Errorf("Render missing safe_read gauge\n--- got ---\n%s", out)
	}
	if !strings.Contains(out, `dealsense_endpoint_risk{path="/api/proposal/generate",level="MODIFY"} 1`) {
		t.Errorf("Render missing modify gauge\n--- got ---\n%s", out)
	}
}

func TestCollector_StableOrder_AcrossRenders(t *testing.T) {
	c := metrics.NewCollector()
	c.IncRequest("/a", "200")
	c.IncRequest("/b", "200")
	c.IncRequest("/c", "200")

	var first, second bytes.Buffer
	_, _ = c.Render(&first)
	_, _ = c.Render(&second)
	if first.String() != second.String() {
		t.Errorf("Render output not stable across calls\nfirst:\n%s\nsecond:\n%s", first.String(), second.String())
	}
}

func TestCollector_RaceFree_ConcurrentInc(t *testing.T) {
	c := metrics.NewCollector()
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			for range 100 {
				c.IncRequest("/api/tender/analyze", "200")
			}
		})
	}
	wg.Wait()

	var buf bytes.Buffer
	_, _ = c.Render(&buf)
	out := buf.String()
	if !strings.Contains(out, `dealsense_requests_total{path="/api/tender/analyze",status="200"} 5000`) {
		t.Errorf("Render missing total = 5000 after 50*100 concurrent inc\n--- got ---\n%s", out)
	}
}

func TestCollector_RendersExpositionContentTypeFriendly(t *testing.T) {
	// Prometheus text format requires LF line endings, no CR.
	c := metrics.NewCollector()
	c.IncRequest("/healthz", "200")

	var buf bytes.Buffer
	_, _ = c.Render(&buf)
	if strings.Contains(buf.String(), "\r") {
		t.Errorf("Render must not contain CR — got %q", buf.String())
	}
	// Output must end with newline per spec.
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Errorf("Render output must end with newline; got %q", buf.String())
	}
}
