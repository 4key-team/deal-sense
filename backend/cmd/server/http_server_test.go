package main

import (
	"net/http"
	"testing"
	"time"
)

// TestNewHTTPServer_Timeouts pins the HTTP server timeouts so that
// long-running endpoints (proposal/tender generation can exceed two
// minutes on Opus-class models) do not get their connection closed by
// the server mid-flight, which appears in the browser as
// `net::ERR_EMPTY_RESPONSE`.
func TestNewHTTPServer_Timeouts(t *testing.T) {
	srv := newHTTPServer(":0", http.NewServeMux())

	if got, min := srv.WriteTimeout, 5*time.Minute; got < min {
		t.Errorf("WriteTimeout = %v, want >= %v (longest LLM call + buffer)", got, min)
	}
	if got, want := srv.ReadTimeout, 30*time.Second; got != want {
		t.Errorf("ReadTimeout = %v, want %v", got, want)
	}
	if got, min := srv.IdleTimeout, 60*time.Second; got < min {
		t.Errorf("IdleTimeout = %v, want >= %v", got, min)
	}
}
