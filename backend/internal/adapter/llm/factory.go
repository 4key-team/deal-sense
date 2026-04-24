package llm

import (
	"fmt"
	"log/slog"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// ProviderConfig holds the user-selected LLM provider settings.
type ProviderConfig struct {
	Provider    string
	BaseURL     string
	APIKey      string
	Model       string
	SOCKS5Proxy string
}

// Factory implements usecase.LLMProviderFactory.
type Factory struct {
	SOCKS5Proxy string
	Logger *slog.Logger
}

func (f Factory) Create(cfg usecase.LLMProviderConfig) (usecase.LLMProvider, error) {
	return NewLLMProvider(ProviderConfig{
		Provider:    cfg.Provider,
		BaseURL:     cfg.BaseURL,
		APIKey:      cfg.APIKey,
		Model:       cfg.Model,
		SOCKS5Proxy: f.SOCKS5Proxy,
	}, f.Logger)
}

// NewLLMProvider creates an LLMProvider based on the provider name.
func NewLLMProvider(cfg ProviderConfig, logger *slog.Logger) (usecase.LLMProvider, error) {
	if err := validateSOCKS5ProxyURL(cfg.SOCKS5Proxy); err != nil {
		return nil, err
	}
	switch cfg.Provider {
	case "openai", "groq", "ollama", "custom":
		return NewOpenAICompatible(OpenAIConfig{
			BaseURL:     cfg.BaseURL,
			APIKey:      cfg.APIKey,
			Model:       cfg.Model,
			Name:        cfg.Provider,
			SOCKS5Proxy: cfg.SOCKS5Proxy,
		}, logger), nil

	case "anthropic":
		return NewAnthropic(AnthropicConfig{
			BaseURL:     cfg.BaseURL,
			APIKey:      cfg.APIKey,
			Model:       cfg.Model,
			SOCKS5Proxy: cfg.SOCKS5Proxy,
		}, logger), nil

	case "gemini":
		return NewGemini(GeminiConfig{
			BaseURL:     cfg.BaseURL,
			APIKey:      cfg.APIKey,
			Model:       cfg.Model,
			SOCKS5Proxy: cfg.SOCKS5Proxy,
		}, logger), nil

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

func validateSOCKS5ProxyURL(raw string) error {
	if raw == "" {
		return nil
	}
	_, err := parseSOCKS5ProxyURL(raw)
	return err
}
