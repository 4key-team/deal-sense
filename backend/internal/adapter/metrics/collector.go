// Package metrics exposes a Prometheus-format collector for app-side
// counters/gauges. The collector is concurrency-safe; consumers (HTTP
// middleware, telegram allowlist, LLM adapters) hold it via narrow
// interfaces defined in their own packages.
package metrics

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Collector aggregates the four metrics emitted by /metrics:
//
//   - dealsense_requests_total{path,status}      — counter, HTTP requests
//   - dealsense_llm_calls_total{provider,status} — counter, LLM provider calls
//   - dealsense_endpoint_risk{path,level}        — gauge, =1 per (path,level)
//   - dealsense_security_decline_total{kind}     — counter, security declines
//
// Zero value is unusable — construct via NewCollector.
type Collector struct {
	requests        *vec
	llmCalls        *vec
	endpointRisk    *vec
	securityDecline *vec
}

// NewCollector returns an empty collector. All counters start at zero.
func NewCollector() *Collector {
	return &Collector{
		requests:        newVec([]string{"path", "status"}),
		llmCalls:        newVec([]string{"provider", "status"}),
		endpointRisk:    newVec([]string{"path", "level"}),
		securityDecline: newVec([]string{"kind"}),
	}
}

// IncRequest increments dealsense_requests_total for (path, status).
func (c *Collector) IncRequest(path, status string) {
	c.requests.inc([]string{path, status})
}

// IncLLMCall increments dealsense_llm_calls_total for (provider, status).
func (c *Collector) IncLLMCall(provider, status string) {
	c.llmCalls.inc([]string{provider, status})
}

// Inc increments dealsense_security_decline_total{kind}. Satisfies the
// DeclineCounter interface in http + telegram adapters.
func (c *Collector) Inc(kind string) {
	c.securityDecline.inc([]string{kind})
}

// SetEndpointRisk sets dealsense_endpoint_risk{path,level} to 1.
func (c *Collector) SetEndpointRisk(path, level string) {
	c.endpointRisk.set([]string{path, level}, 1)
}

// Render writes the snapshot of all metrics in Prometheus exposition format
// (text version 0.0.4) to w. Per-metric ordering is insertion order, which
// is stable across calls for the same label set.
func (c *Collector) Render(w io.Writer) (int, error) {
	var sb strings.Builder
	writeMetric(&sb, "dealsense_requests_total", "Total HTTP requests handled, labelled by path and status.", "counter", c.requests)
	writeMetric(&sb, "dealsense_llm_calls_total", "Total LLM provider calls, labelled by provider and status.", "counter", c.llmCalls)
	writeMetric(&sb, "dealsense_endpoint_risk", "Risk classification of registered HTTP endpoints (one series per path+level).", "gauge", c.endpointRisk)
	writeMetric(&sb, "dealsense_security_decline_total", "Security declines by kind (allowlist|api_key|rate_limit|llm_parse_error).", "counter", c.securityDecline)
	return io.WriteString(w, sb.String())
}

func writeMetric(sb *strings.Builder, name, help, kind string, v *vec) {
	fmt.Fprintf(sb, "# HELP %s %s\n", name, help)
	fmt.Fprintf(sb, "# TYPE %s %s\n", name, kind)
	v.render(sb, name)
}

// vec is a generic labelled metric backing both counters and gauges. Counter
// callers use inc; gauge callers use set. Render iterates in insertion order
// for stable output.
type vec struct {
	mu        sync.Mutex
	labelKeys []string
	values    map[string]float64
	labels    map[string][]string
	order     []string
}

func newVec(labelKeys []string) *vec {
	return &vec{
		labelKeys: labelKeys,
		values:    map[string]float64{},
		labels:    map[string][]string{},
	}
}

func (v *vec) key(vals []string) string {
	return strings.Join(vals, "\x00")
}

func (v *vec) inc(vals []string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	k := v.key(vals)
	if _, ok := v.values[k]; !ok {
		v.order = append(v.order, k)
		cp := make([]string, len(vals))
		copy(cp, vals)
		v.labels[k] = cp
	}
	v.values[k]++
}

func (v *vec) set(vals []string, val float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	k := v.key(vals)
	if _, ok := v.values[k]; !ok {
		v.order = append(v.order, k)
		cp := make([]string, len(vals))
		copy(cp, vals)
		v.labels[k] = cp
	}
	v.values[k] = val
}

func (v *vec) render(sb *strings.Builder, name string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, k := range v.order {
		vals := v.labels[k]
		sb.WriteString(name)
		sb.WriteByte('{')
		for i, key := range v.labelKeys {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(key)
			sb.WriteString(`="`)
			sb.WriteString(escapeLabel(vals[i]))
			sb.WriteByte('"')
		}
		sb.WriteByte('}')
		fmt.Fprintf(sb, " %s\n", formatValue(v.values[k]))
	}
}

func escapeLabel(s string) string {
	if !strings.ContainsAny(s, `\"`+"\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatValue(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%g", f)
}
