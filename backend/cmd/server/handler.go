package main

import (
	"log/slog"
	"net/http"

	apphttp "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
)

// buildHandler composes the full HTTP middleware stack:
//
//	Recover → Logger → CORS → MetricsRequests → combined
//	                                              ├─ bypass paths → mux directly
//	                                              └─ gated → APIKeyAuth → RateLimit → mux
//
// The /healthz, /readyz, /metrics paths in apphttp.BypassedPaths() skip
// APIKeyAuth + RateLimit so orchestrators and scrapers can hit them without
// holding a secret and without contributing to the per-IP bucket. Everything
// else goes through the gated branch.
//
// Extracted from run() so cmd/server/bypass_chain_test.go can verify the
// composition at the cmd layer (regression target: anyone moving APIKeyAuth
// ahead of the bypass branch will flip the test).
func buildHandler(
	mux http.Handler,
	apiKey string,
	rpsLimit float64,
	rpsBurst int,
	collector *metrics.Collector,
	logger *slog.Logger,
) http.Handler {
	gated := mux
	gated = apphttp.RateLimit(rpsLimit, rpsBurst, collector, gated)
	gated = apphttp.APIKeyAuth(apiKey, collector, gated)

	bypass := apphttp.BypassedPaths()
	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := bypass[r.URL.Path]; ok {
			mux.ServeHTTP(w, r)
			return
		}
		gated.ServeHTTP(w, r)
	})

	var handler http.Handler = combined
	handler = apphttp.MetricsRequests(collector, handler)
	handler = apphttp.CORS("*", handler)
	handler = apphttp.Logger(logger, handler)
	handler = apphttp.Recover(logger, handler)
	return handler
}
