package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/daniil/deal-sense/backend/internal/config"
)

func TestRun_StartsAndStops(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		Port:        "18923",
		LLMProvider: "anthropic",
		LLMBaseURL:  "http://localhost:1",
		LLMAPIKey:   "test",
		LLMModel:    "test-model",
	}

	ctx, cancel := context.WithCancel(t.Context())

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, logger, cfg)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:18923/api/llm/providers")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	resp, err := http.Get("http://localhost:18923/api/llm/providers")
	if err != nil {
		t.Fatalf("GET /api/llm/providers: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["providers"] == nil {
		t.Error("expected providers in response")
	}

	cancel()

	if err := <-errCh; err != nil {
		t.Errorf("run() returned error: %v", err)
	}
}

func TestRun_PortInUse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ln, err := net.Listen("tcp", ":18925")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	cfg := config.Config{
		Port:        "18925",
		LLMProvider: "anthropic",
		LLMBaseURL:  "http://localhost:1",
		LLMAPIKey:   "test",
		LLMModel:    "test",
	}

	err = run(t.Context(), logger, cfg)
	if err == nil {
		t.Error("expected error for port in use")
	}
}

func TestRun_InvalidProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		Port:        "18926",
		LLMProvider: "nonexistent",
	}

	err := run(t.Context(), logger, cfg)
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}

func TestMain_Integration(t *testing.T) {
	binPath := t.TempDir() + "/server"
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"PORT=18930",
		"LLM_PROVIDER=anthropic",
		"LLM_BASE_URL=http://localhost:1",
		"LLM_API_KEY=test",
		"LLM_MODEL=test",
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:18930/api/llm/providers")
		if err == nil {
			resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !ready {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatal("server did not start in time")
	}

	resp, err := http.Get("http://localhost:18930/api/llm/providers")
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	cmd.Process.Signal(os.Interrupt)
	cmd.Wait()
}
