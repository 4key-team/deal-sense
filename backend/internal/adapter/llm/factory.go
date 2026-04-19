package llm

import (
	"fmt"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// ProviderConfig holds the user-selected LLM provider settings.
type ProviderConfig struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

// NewLLMProvider creates an LLMProvider based on the provider name.
func NewLLMProvider(cfg ProviderConfig) (usecase.LLMProvider, error) {
	switch cfg.Provider {
	case "openai", "groq", "ollama", "custom":
		return NewOpenAICompatible(OpenAIConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			Name:    cfg.Provider,
		}), nil

	case "anthropic":
		return NewAnthropic(AnthropicConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
		}), nil

	case "gemini":
		return NewGemini(GeminiConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
		}), nil

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
