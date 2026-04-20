package usecase

import (
	"context"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// LLMProvider generates text completions from a prompt.
type LLMProvider interface {
	GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	CheckConnection(ctx context.Context) error
	ListModels(ctx context.Context) ([]string, error)
	Name() string
}

// DocumentParser extracts text content from a document file.
type DocumentParser interface {
	Parse(ctx context.Context, filename string, data []byte) (string, error)
	Supports(fileType domain.FileType) bool
}

// TemplateEngine fills a DOCX template with key-value parameters.
type TemplateEngine interface {
	Fill(ctx context.Context, template []byte, params map[string]string) ([]byte, error)
}

// LLMProviderConfig holds user-selected LLM settings from the request.
type LLMProviderConfig struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

// LLMProviderFactory creates an LLMProvider from a user-supplied config.
type LLMProviderFactory interface {
	Create(cfg LLMProviderConfig) (LLMProvider, error)
}
