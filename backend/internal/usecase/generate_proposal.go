package usecase

import (
	"context"
	"encoding/json"
	"fmt"
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

type proposalLLMResponse struct {
	Params   map[string]string      `json:"params"`
	Meta     map[string]string      `json:"meta"`
	Sections []proposalLLMSection   `json:"sections"`
	Summary  string                 `json:"summary"`
	Log      []proposalLLMLog       `json:"log"`
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
	proposal, err := domain.NewProposal(templateName, templateData, userParams)
	if err != nil {
		return nil, noUsage, err
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
	templateText, parseErr := uc.parser.Parse(ctx, templateName, templateData)
	if parseErr != nil || templateText == "" {
		templateText = templateParseFallback
	}

	// Detect template mode: placeholder (has {{...}}) or generative (no placeholders).
	mode, detectErr := DetectTemplateMode(templateData)
	if detectErr != nil {
		// Non-DOCX or unreadable — default to placeholder mode.
		mode = domain.ModePlaceholder
	}
	// Use generative mode only if we have a generative engine.
	if mode == domain.ModeGenerative && uc.generative == nil {
		mode = domain.ModePlaceholder
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

	// Fill template based on mode.
	var filled []byte
	switch mode {
	case domain.ModeGenerative:
		genSections := make([]GenerativeSection, 0, len(resp.Sections))
		for _, s := range resp.Sections {
			genSections = append(genSections, GenerativeSection{Title: s.Title, Content: s.Content})
		}
		filled, err = uc.generative.GenerativeFill(ctx, templateData, genSections)
	default:
		// Placeholder mode: merge params and fill template.
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

	// Map sections
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

	// Best-effort PDF generation.
	if uc.pdfGen != nil {
		pdfSections := make([]PDFSection, 0, len(resp.Sections))
		for _, s := range resp.Sections {
			pdfSections = append(pdfSections, PDFSection{Title: s.Title, Content: s.Content})
		}
		pdfInput := PDFInput{
			Meta:     resp.Meta,
			Sections: pdfSections,
			Summary:  resp.Summary,
		}
		pdfBytes, pdfErr := uc.pdfGen.Generate(ctx, pdfInput)
		if pdfErr == nil {
			proposal.SetPDFResult(pdfBytes)
		}
	}

	return proposal, usage, nil
}
