package metrics_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
	"github.com/daniil/deal-sense/backend/internal/domain/security"
)

// TestEndpointRiskPopulation_FromRegistry pins the contract between the
// default EndpointRegistry (domain Layer 4) and the metrics adapter:
// every registered path should be visible as a dealsense_endpoint_risk
// gauge once the server's startup loop has run. The loop itself lives
// in cmd/server/main.go — this test reproduces it locally so a future
// change to the registry shape is caught here, not at deploy time.
func TestEndpointRiskPopulation_FromRegistry(t *testing.T) {
	c := metrics.NewCollector()
	registry := security.NewDefaultEndpointRegistry()
	for _, path := range registry.Paths() {
		level, err := registry.Lookup(path)
		if err != nil {
			t.Fatalf("registry.Lookup(%q) failed for path returned by Paths(): %v", path, err)
		}
		c.SetEndpointRisk(path, level.String())
	}

	var buf bytes.Buffer
	_, _ = c.Render(&buf)
	out := buf.String()

	for _, path := range registry.Paths() {
		level, _ := registry.Lookup(path)
		needle := `dealsense_endpoint_risk{path="` + path + `",level="` + level.String() + `"} 1`
		if !strings.Contains(out, needle) {
			t.Errorf("missing gauge for %q (level %s)\n--- got ---\n%s", path, level, out)
		}
	}
}
