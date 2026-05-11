package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

const templateParseFallback = "(template could not be parsed as text — generate content based on context documents and typical proposal structure)"

type GenerateProposal struct {
	llm              LLMProvider
	parser           DocumentParser
	template         TemplateEngine
	systemPrompt     string
	generative       GenerativeEngine
	generativePrompt string
	pdfGen           PDFGenerator
	docxToPDF        DOCXToPDFConverter
	mdGen            MDGenerator
	logger           *slog.Logger
}

func NewGenerateProposal(llm LLMProvider, parser DocumentParser, template TemplateEngine, systemPrompt string) *GenerateProposal {
	return &GenerateProposal{llm: llm, parser: parser, template: template, systemPrompt: systemPrompt}
}

func (uc *GenerateProposal) SetGenerativeEngine(g GenerativeEngine, prompt string) {
	uc.generative = g
	uc.generativePrompt = prompt
}

func (uc *GenerateProposal) SetPDFGenerator(g PDFGenerator) {
	uc.pdfGen = g
}

func (uc *GenerateProposal) SetDOCXToPDFConverter(c DOCXToPDFConverter) {
	uc.docxToPDF = c
}

func (uc *GenerateProposal) SetMDGenerator(g MDGenerator) {
	uc.mdGen = g
}

func (uc *GenerateProposal) SetLogger(l *slog.Logger) {
	uc.logger = l
}

func (uc *GenerateProposal) log() *slog.Logger {
	if uc.logger != nil {
		return uc.logger
	}
	return slog.Default()
}

type proposalLLMResponse struct {
	Params   map[string]string    `json:"params"`
	Meta     map[string]string    `json:"meta"`
	Sections []proposalLLMSection `json:"sections"`
	Summary  string               `json:"summary"`
	Log      []proposalLLMLog     `json:"log"`
}

type proposalLLMSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Status  string `json:"status"`
	Tokens  int    `json:"tokens"`
}

type proposalLLMLog struct {
	Time string `json:"time"`
	Msg  string `json:"msg"`
}

func (uc *GenerateProposal) Execute(
	ctx context.Context,
	templateName string,
	templateData []byte,
	contextFiles []FileInput,
	userParams map[string]string,
) (*domain.Proposal, domain.TokenUsage, error) {
	noUsage := domain.ZeroTokenUsage()

	// Determine mode: clean (no template) or template-based.
	isClean := len(templateData) == 0 && uc.generative != nil

	var proposal *domain.Proposal
	if isClean {
		proposal = domain.NewCleanProposal(userParams)
	} else {
		var err error
		proposal, err = domain.NewProposal(templateName, templateData, userParams)
		if err != nil {
			return nil, noUsage, err
		}
	}

	// Parse context files
	var contextText strings.Builder
	for _, f := range contextFiles {
		text, err := uc.parser.Parse(ctx, f.Name, f.Data)
		if err != nil {
			continue // skip unparseable context files
		}
		fmt.Fprintf(&contextText, "=== %s ===\n%s\n\n", f.Name, text)
	}

	// Read template text for LLM (best-effort — complex templates may fail to parse as text)
	var templateText string
	if isClean {
		templateText = templateParseFallback
	} else {
		var parseErr error
		templateText, parseErr = uc.parser.Parse(ctx, templateName, templateData)
		if parseErr != nil || templateText == "" {
			templateText = templateParseFallback
		}
	}

	// Detect template mode.
	var mode domain.TemplateMode
	if isClean {
		mode = domain.ModeClean
	} else {
		var detectErr error
		mode, detectErr = DetectTemplateMode(templateData)
		if detectErr != nil {
			if uc.generative != nil {
				mode = domain.ModeGenerative
			} else {
				mode = domain.ModePlaceholder
			}
		}
		if mode == domain.ModeGenerative && uc.generative == nil {
			mode = domain.ModePlaceholder
		}
	}
	proposal.SetMode(mode)

	// Select system prompt based on mode.
	systemPrompt := uc.systemPrompt
	if mode == domain.ModeGenerative {
		systemPrompt = uc.generativePrompt
	}

	userPrompt := fmt.Sprintf(
		"Template from file %s:\n%s\n\nContext documents:\n%s\n\nUser parameters: %v\n\nGenerate content based on the context.",
		templateName, templateText, contextText.String(), userParams,
	)

	llmResp, usage, err := uc.llm.GenerateCompletion(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, noUsage, fmt.Errorf("llm completion: %w", err)
	}

	cleaned := extractJSON(llmResp)
	var resp proposalLLMResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, noUsage, fmt.Errorf("parse llm response: %w (raw: %.200s)", err, llmResp)
	}

	// Build content sections once — reused by generative fill, PDF, and MD.
	contentSections := make([]ContentSection, 0, len(resp.Sections))
	for _, s := range resp.Sections {
		contentSections = append(contentSections, ContentSection{Title: s.Title, Content: s.Content})
	}
	contentInput := ContentInput{Meta: resp.Meta, Sections: contentSections, Summary: resp.Summary}

	// Fill template based on mode.
	var filled []byte
	switch mode {
	case domain.ModeClean:
		filled, err = uc.generative.GenerateClean(ctx, contentInput)
	case domain.ModeGenerative:
		filled, err = uc.generative.GenerativeFill(ctx, templateData, contentSections)
	default:
		mergedParams := make(map[string]string)
		if resp.Meta != nil {
			maps.Copy(mergedParams, resp.Meta)
			if v, ok := resp.Meta["client"]; ok {
				mergedParams["client_name"] = v
			}
			if v, ok := resp.Meta["project"]; ok {
				mergedParams["project_name"] = v
			}
		}
		maps.Copy(mergedParams, resp.Params)
		maps.Copy(mergedParams, userParams)
		filled, err = uc.template.Fill(ctx, templateData, mergedParams)
	}
	if err != nil {
		return nil, noUsage, fmt.Errorf("template fill: %w", err)
	}

	proposal.SetResult(filled)

	// Map domain sections.
	sections := make([]domain.ProposalSection, 0, len(resp.Sections))
	for _, s := range resp.Sections {
		st, err := domain.ParseSectionStatus(s.Status)
		if err != nil {
			st = domain.SectionAI
		}
		sec, err := domain.NewProposalSection(s.Title, st, s.Tokens)
		if err != nil {
			continue
		}
		sections = append(sections, sec)
	}
	proposal.SetSections(sections, resp.Summary)
	proposal.SetMeta(resp.Meta)

	logEntries := make([]domain.LogEntry, 0, len(resp.Log))
	for _, l := range resp.Log {
		entry, err := domain.NewLogEntry(l.Time, l.Msg)
		if err != nil {
			continue
		}
		logEntries = append(logEntries, entry)
	}
	proposal.SetLog(logEntries)

	// Best-effort PDF generation: prefer DOCX→PDF converter, fallback to Maroto.
	var pdfDone bool
	if uc.docxToPDF != nil && len(filled) > 0 {
		pdfBytes, convErr := uc.docxToPDF.Convert(ctx, filled)
		if convErr != nil {
			uc.log().Warn("docx-to-pdf conversion failed, falling back to maroto", "err", convErr)
		} else {
			proposal.SetPDFResult(pdfBytes)
			pdfDone = true
		}
	}
	if !pdfDone && uc.pdfGen != nil {
		pdfBytes, pdfErr := uc.pdfGen.Generate(ctx, contentInput)
		if pdfErr != nil {
			uc.log().Warn("maroto pdf generation failed", "err", pdfErr)
		} else {
			proposal.SetPDFResult(pdfBytes)
		}
	}

	// Best-effort Markdown generation.
	if uc.mdGen != nil {
		mdBytes, mdErr := uc.mdGen.Render(ctx, contentInput)
		if mdErr != nil {
			uc.log().Warn("markdown generation failed", "err", mdErr)
		} else {
			proposal.SetMDResult(mdBytes)
		}
	}

	return proposal, usage, nil
}
