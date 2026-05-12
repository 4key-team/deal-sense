package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apphttp "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	apppdf "github.com/daniil/deal-sense/backend/internal/adapter/pdf"
	"github.com/daniil/deal-sense/backend/internal/config"
	"github.com/daniil/deal-sense/backend/internal/domain/security"
)

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger, cfg); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	provider, err := llm.NewLLMProvider(llm.ProviderConfig{
		Provider: cfg.LLMProvider,
		BaseURL:  cfg.LLMBaseURL,
		APIKey:   cfg.LLMAPIKey,
		Model:    cfg.LLMModel,
	}, logger)
	if err != nil {
		return fmt.Errorf("init llm provider: %w", err)
	}
	logger.Info("llm provider initialized", "provider", provider.Name(), "model", cfg.LLMModel)

	docParser := parser.NewComposite(parser.NewPDFParser(), parser.NewDocxReader(), parser.NewMDParser())
	docxTemplate := parser.NewDocxTemplate()
	docxGenerative := parser.NewDocxGenerative()
	pdfGen := apppdf.NewMarotoPDFGenerator()
	docxToPDF := apppdf.NewLibreOfficeConverter()

	providers := []apphttp.ProviderInfo{
		{ID: "anthropic", Name: "Anthropic", Models: []string{"claude-haiku-4-5", "claude-sonnet-4-5", "claude-opus-4-1"}},
		{ID: "openai", Name: "OpenAI", Models: []string{"gpt-4o", "gpt-4o-mini", "o3-mini"}},
		{ID: "gemini", Name: "Google Gemini", Models: []string{"gemini-2.5-pro", "gemini-2.5-flash"}},
		{ID: "groq", Name: "Groq", Models: []string{"llama-3.3-70b", "mixtral-8x7b"}},
		{ID: "ollama", Name: "Ollama (local)", Models: []string{"llama3.1:70b", "qwen2.5:32b"}},
		{ID: "custom", Name: "Custom", Models: []string{}},
	}
	mdRenderer := parser.NewMarkdownRenderer()
	h := apphttp.NewHandler(provider, llm.Factory{Logger: logger}, docParser, docxTemplate, llm.TenderAnalysisPrompt, llm.ProposalGenerationPrompt, providers, logger, pdfGen, docxToPDF, docxGenerative, llm.GenerativeProposalPrompt, mdRenderer)
	collector := metrics.NewCollector()
	// Pre-populate endpoint risk gauges so /metrics reflects Layer 4
	// annotations from the moment the server is reachable, not on the
	// first matching request.
	for _, path := range security.NewDefaultEndpointRegistry().Paths() {
		level, _ := security.NewDefaultEndpointRegistry().Lookup(path)
		collector.SetEndpointRisk(path, level.String())
	}
	mux := apphttp.NewRouter(h, collector)

	// Health probes bypass auth + rate limit so orchestrators can hit them
	// without holding an API key and without contributing to the per-IP
	// bucket.
	gated := http.Handler(mux)
	gated = apphttp.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, gated)
	gated = apphttp.APIKeyAuth(cfg.APIKey, gated)

	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			mux.ServeHTTP(w, r)
			return
		}
		gated.ServeHTTP(w, r)
	})

	var handler http.Handler = combined
	handler = apphttp.CORS("*", handler)
	handler = apphttp.Logger(logger, handler)
	handler = apphttp.Recover(logger, handler)
	if cfg.APIKey != "" {
		logger.Info("api key auth enabled")
	}
	if cfg.RateLimitRPS > 0 {
		logger.Info("rate limit enabled", "rps", cfg.RateLimitRPS, "burst", cfg.RateLimitBurst)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
