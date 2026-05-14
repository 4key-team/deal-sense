package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	httpadapter "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
)

// metricsMux returns an http.Handler exposing two endpoints:
//
//	GET /metrics  → Prometheus exposition format from collector.Render
//	GET /healthz  → liveness probe, always 200
//
// Anything else returns 404. The bot has no authenticated traffic on this
// listener — operators are expected to bind it to a private interface or
// scrape from within the same compose network.
func metricsMux(coll *metrics.Collector) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", httpadapter.MetricsHandler(coll))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

// runMetricsServer hosts metricsMux on addr until ctx is canceled, then
// shuts the server down with a short grace period. Returns the first
// non-shutdown error, or nil on clean stop.
func runMetricsServer(ctx context.Context, addr string, coll *metrics.Collector, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           metricsMux(coll),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		logger.Info("metrics listener stopping", "addr", addr)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("metrics shutdown", "err", err)
		}
		return <-errCh
	case err := <-errCh:
		return err
	}
}
