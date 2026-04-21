package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// FileInput is a DTO for incoming file data.
type FileInput struct {
	Name string
	Data []byte
	Type domain.FileType
}

type AnalyzeTender struct {
	llm          LLMProvider
	parser       DocumentParser
	systemPrompt string
}

func NewAnalyzeTender(llm LLMProvider, parser DocumentParser, systemPrompt string) *AnalyzeTender {
	return &AnalyzeTender{llm: llm, parser: parser, systemPrompt: systemPrompt}
}

// extractJSON strips markdown code fences from LLM output.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if start := strings.Index(s, "```"); start != -1 {
		// skip ```json or ``` line
		inner := s[start+3:]
		if nl := strings.Index(inner, "\n"); nl != -1 {
			inner = inner[nl+1:]
		}
		if end := strings.LastIndex(inner, "```"); end != -1 {
			inner = inner[:end]
		}
		return strings.TrimSpace(inner)
	}
	return s
}

type analysisProCon struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

type analysisRequirement struct {
	Label  string `json:"label"`
	Status string `json:"status"` // met | partial | miss
}

type analysisResponse struct {
	Verdict      string                `json:"verdict"`
	Risk         string                `json:"risk"`
	Score        int                   `json:"score"`
	Summary      string                `json:"summary"`
	Pros         []analysisProCon      `json:"pros"`
	Cons         []analysisProCon      `json:"cons"`
	Requirements []analysisRequirement `json:"requirements"`
	Effort       string                `json:"effort"`
}

func (uc *AnalyzeTender) Execute(
	ctx context.Context,
	files []FileInput,
	companyProfile string,
) (*domain.TenderAnalysis, domain.TokenUsage, error) {
	var noUsage domain.TokenUsage
	if len(files) == 0 {
		return nil, noUsage, domain.ErrEmptyContent
	}
	if companyProfile == "" {
		return nil, noUsage, domain.ErrEmptyCompany
	}

	var docs []domain.Document
	var allText strings.Builder

	for _, f := range files {
		text, err := uc.parser.Parse(ctx, f.Name, f.Data)
		if err != nil {
			return nil, noUsage, fmt.Errorf("parse %s: %w", f.Name, err)
		}
		doc, err := domain.NewDocument(f.Name, f.Type, text)
		if err != nil {
			return nil, noUsage, fmt.Errorf("document %s: %w", f.Name, err)
		}
		docs = append(docs, *doc)
		fmt.Fprintf(&allText, "=== %s ===\n%s\n\n", f.Name, text)
	}

	analysis, _ := domain.NewTenderAnalysis(docs, companyProfile)

	userPrompt := fmt.Sprintf("Company profile:\n%s\n\nTender documents:\n%s", companyProfile, allText.String())

	llmResp, usage, err := uc.llm.GenerateCompletion(ctx, uc.systemPrompt, userPrompt)
	if err != nil {
		return nil, noUsage, fmt.Errorf("llm analysis: %w", err)
	}

	var resp analysisResponse
	cleaned := extractJSON(llmResp)
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, noUsage, fmt.Errorf("parse llm response: %w (raw: %.200s)", err, llmResp)
	}

	verdict, err := domain.ParseVerdict(resp.Verdict)
	if err != nil {
		return nil, noUsage, fmt.Errorf("parse verdict: %w", err)
	}
	risk, err := domain.ParseRisk(resp.Risk)
	if err != nil {
		return nil, noUsage, fmt.Errorf("parse risk: %w", err)
	}
	score, err := domain.NewMatchScore(resp.Score)
	if err != nil {
		return nil, noUsage, fmt.Errorf("parse score: %w", err)
	}

	analysis.SetResult(verdict, risk, score, resp.Summary)
	pros := make([]domain.ProCon, 0, len(resp.Pros))
	for _, p := range resp.Pros {
		pc, err := domain.NewProCon(p.Title, p.Desc)
		if err != nil {
			continue
		}
		pros = append(pros, pc)
	}
	cons := make([]domain.ProCon, 0, len(resp.Cons))
	for _, c := range resp.Cons {
		pc, err := domain.NewProCon(c.Title, c.Desc)
		if err != nil {
			continue
		}
		cons = append(cons, pc)
	}
	reqs := make([]domain.Requirement, 0, len(resp.Requirements))
	for _, r := range resp.Requirements {
		st, err := domain.ParseRequirementStatus(r.Status)
		if err != nil {
			continue
		}
		req, err := domain.NewRequirement(r.Label, st)
		if err != nil {
			continue
		}
		reqs = append(reqs, req)
	}
	analysis.SetExtras(pros, cons, reqs, resp.Effort)
	return analysis, usage, nil
}
