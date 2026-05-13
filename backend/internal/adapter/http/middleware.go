package http

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DeclineCounter is the narrow port the HTTP middleware uses to record
// security declines (401, 429). Implementations live in adapter/metrics.
// nil is tolerated and treated as a no-op — production wires a collector,
// unit tests typically pass nil.
type DeclineCounter interface {
	Inc(kind string)
}

// DeclineKindAPIKey / DeclineKindRateLimit are the canonical decline-kind
// labels emitted by this package. They duplicate the constants in
// adapter/metrics to keep the http package importable in isolation
// (avoiding the metrics → http back-reference cycle).
const (
	DeclineKindAPIKey    = "api_key"
	DeclineKindRateLimit = "rate_limit"
)

// RequestCounter is the narrow port the request-counting middleware uses
// to record HTTP requests by path + status.
type RequestCounter interface {
	IncRequest(path, status string)
}

// MetricsRequests wraps next with per-request counting via counter.IncRequest.
// nil counter ⇒ passthrough (no-op for deployments without metrics).
//
// The status reported to the counter mirrors what the client sees: when
// the inner handler returns without explicitly writing a header,
// net/http emits 200, and the counter records "200" too.
func MetricsRequests(counter RequestCounter, next http.Handler) http.Handler {
	if counter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		counter.IncRequest(r.URL.Path, strconv.Itoa(sw.status))
	})
}

// CORS wraps a handler with CORS headers for the frontend dev server.
func CORS(allowOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-LLM-Provider, X-LLM-Key, X-LLM-URL, X-LLM-Model")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying writer so SSE handlers wrapped by
// this middleware can still stream. Without this, http.ResponseController
// also unwraps to find the Flusher, but exposing it directly keeps the
// older `w.(http.Flusher)` type-assertion idiom working too.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logger logs every request with method, path, status, duration.
func Logger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

// Recover catches panics and returns 500.
func Recover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered", "error", err, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RateLimit caps the request rate per client (key: RemoteAddr IP) via a
// token bucket. When the bucket for a key is empty the middleware
// responds 429 + Retry-After and short-circuits — the wrapped handler is
// not invoked. rps == 0 disables the middleware (passthrough).
//
// Defence-in-depth only — the primary rate-limit is the API gateway.
// Per-key state is kept in-memory; for multi-instance deployments use the
// gateway's distributed limiter as the source of truth.
func RateLimit(rps float64, burst int, counter DeclineCounter, next http.Handler) http.Handler {
	if rps <= 0 {
		return next
	}
	limit := rate.Limit(rps)
	limiters := &perKeyLimiters{limit: limit, burst: burst}
	retryAfter := strconv.Itoa(int(math.Ceil(1 / rps)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := remoteIP(r)
		if !limiters.get(key).Allow() {
			if counter != nil {
				counter.Inc(DeclineKindRateLimit)
			}
			w.Header().Set("Retry-After", retryAfter)
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// perKeyLimiters stores one token-bucket per key. No eviction — fine for
// the local defence-in-depth role; high cardinality is handled by the
// gateway anyway.
type perKeyLimiters struct {
	mu       sync.Mutex
	limit    rate.Limit
	burst    int
	limiters map[string]*rate.Limiter
}

func (p *perKeyLimiters) get(key string) *rate.Limiter {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.limiters == nil {
		p.limiters = map[string]*rate.Limiter{}
	}
	lim, ok := p.limiters[key]
	if !ok {
		lim = rate.NewLimiter(p.limit, p.burst)
		p.limiters[key] = lim
	}
	return lim
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// APIKeyAuth gates requests behind the X-API-Key header. When expectedKey is
// empty the middleware is a passthrough — this preserves open-access local
// dev while production deployments inject a real key via env var. CORS must
// be wrapped outside this middleware so preflight requests succeed without
// the header.
func APIKeyAuth(expectedKey string, counter DeclineCounter, next http.Handler) http.Handler {
	if expectedKey == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != expectedKey {
			if counter != nil {
				counter.Inc(DeclineKindAPIKey)
			}
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
