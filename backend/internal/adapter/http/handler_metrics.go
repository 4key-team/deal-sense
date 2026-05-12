package http

import (
	"io"
	"net/http"
)

// MetricsRenderer is the narrow interface MetricsHandler depends on. The
// HTTP adapter owns the contract; adapter/metrics.Collector implements it.
type MetricsRenderer interface {
	Render(w io.Writer) (int, error)
}

// MetricsHandler returns an http.HandlerFunc that exposes the snapshot of
// the supplied renderer in Prometheus text exposition format (version
// 0.0.4). The handler bypasses APIKeyAuth + RateLimit at the wiring layer
// (cmd/server/main.go) so scrapers can poll without holding a secret and
// without exhausting the per-IP bucket.
func MetricsHandler(r MetricsRenderer) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// RED stub: writes nothing, returns 200 with the right header so
		// the contract test for the header passes; the body assertion
		// fails until GREEN.
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}
}
