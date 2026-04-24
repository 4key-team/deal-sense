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

// GenerativeSection holds a section title and its generated content.
type GenerativeSection struct {
	Title   string
	Content string
}

// GenerativeEngine fills a template in generative mode (no placeholders).
type GenerativeEngine interface {
	GenerativeFill(ctx context.Context, template []byte, sections []GenerativeSection) ([]byte, error)
}

// PDFSection holds a section for PDF generation.
type PDFSection struct {
	Title   string
	Content string
}

// PDFInput holds all data needed to generate a PDF proposal.
type PDFInput struct {
	Meta     map[string]string
	Sections []PDFSection
	Summary  string
}

// PDFGenerator creates a PDF document from proposal data.
type PDFGenerator interface {
	Generate(ctx context.Context, input PDFInput) ([]byte, error)
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
