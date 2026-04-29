package usecase

import (
	"context"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// LLMProvider generates text completions from a prompt.
type LLMProvider interface {
	GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, domain.TokenUsage, error)
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

// ContentSection holds a section title and its generated content.
// Used by GenerativeEngine, PDFGenerator, and MDGenerator.
type ContentSection struct {
	Title   string
	Content string
}

// ContentInput holds all data needed to generate output (PDF, Markdown, etc.).
type ContentInput struct {
	Meta     map[string]string
	Sections []ContentSection
	Summary  string
}

// GenerativeEngine fills a template in generative mode (no placeholders)
// or generates a clean document from scratch.
type GenerativeEngine interface {
	GenerativeFill(ctx context.Context, template []byte, sections []ContentSection) ([]byte, error)
	GenerateClean(ctx context.Context, input ContentInput) ([]byte, error)
}

// PDFGenerator creates a PDF document from proposal data.
type PDFGenerator interface {
	Generate(ctx context.Context, input ContentInput) ([]byte, error)
}

// MDGenerator creates a Markdown document from proposal data.
type MDGenerator interface {
	Render(ctx context.Context, input ContentInput) ([]byte, error)
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
