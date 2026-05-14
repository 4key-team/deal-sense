package http

import (
	"testing"
	"time"
)

// SetSSEKeepAliveForTest swaps the SSE keep-alive interval for the
// duration of a single test and restores the previous value on
// cleanup. Lets streaming tests assert progress events in
// milliseconds rather than the production 15s tick.
func SetSSEKeepAliveForTest(t *testing.T, d time.Duration) {
	t.Helper()
	sseKeepAliveMu.Lock()
	prev := sseKeepAliveInterval
	sseKeepAliveInterval = d
	sseKeepAliveMu.Unlock()
	t.Cleanup(func() {
		sseKeepAliveMu.Lock()
		sseKeepAliveInterval = prev
		sseKeepAliveMu.Unlock()
	})
}
