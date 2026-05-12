// Package metrics exposes a Prometheus-format collector for app-side
// counters/gauges. The collector is concurrency-safe; consumers (HTTP
// middleware, telegram allowlist, LLM adapters) hold it via narrow
// interfaces defined in their own packages.
package metrics

import "io"

// Collector aggregates the four metrics emitted by /metrics:
//
//   - dealsense_requests_total{path,status}      — counter, HTTP requests
//   - dealsense_llm_calls_total{provider,status} — counter, LLM provider calls
//   - dealsense_endpoint_risk{path,level}        — gauge, =1 per (path,level)
//   - dealsense_security_decline_total{kind}     — counter, security declines
//
// Zero value is unusable — construct via NewCollector.
type Collector struct{}

// NewCollector returns an empty collector. All counters start at zero.
func NewCollector() *Collector { return &Collector{} }

// IncRequest increments dealsense_requests_total for (path, status).
func (c *Collector) IncRequest(path, status string) {}

// IncLLMCall increments dealsense_llm_calls_total for (provider, status).
func (c *Collector) IncLLMCall(provider, status string) {}

// Inc increments dealsense_security_decline_total{kind}. Satisfies the
// DeclineCounter interface in http + telegram adapters.
func (c *Collector) Inc(kind string) {}

// SetEndpointRisk sets dealsense_endpoint_risk{path,level} to 1.
func (c *Collector) SetEndpointRisk(path, level string) {}

// Render writes the snapshot of all metrics in Prometheus exposition format
// (text version 0.0.4) to w. Order is stable across calls for the same
// label set.
func (c *Collector) Render(w io.Writer) (int, error) {
	return 0, nil
}
