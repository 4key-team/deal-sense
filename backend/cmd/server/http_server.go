package main

import (
	"net/http"
	"time"
)

// newHTTPServer builds the production http.Server with timeouts tuned
// for the slowest expected handler. Proposal/tender generation with
// Opus-class models can take 2-3 minutes; WriteTimeout must exceed
// that or the server closes the socket before the JSON response is
// written and the browser sees net::ERR_EMPTY_RESPONSE.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 6 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}
}
