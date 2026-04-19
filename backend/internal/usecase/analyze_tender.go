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
	llm    LLMProvider
	parser DocumentParser
}

func NewAnalyzeTender(llm LLMProvider, parser DocumentParser) *AnalyzeTender {
	return &AnalyzeTender{llm: llm, parser: parser}
}

type analysisResponse struct {
	Verdict string `json:"verdict"`
	Risk    string `json:"risk"`
	Score   int    `json:"score"`
	Summary string `json:"summary"`
}

func (uc *AnalyzeTender) Execute(
	ctx context.Context,
	files []FileInput,
	companyProfile string,
) (*domain.TenderAnalysis, error) {
	if len(files) == 0 {
		return nil, domain.ErrEmptyContent
	}
	if companyProfile == "" {
		return nil, domain.ErrEmptyCompany
	}

	var docs []domain.Document
	var allText strings.Builder

	for _, f := range files {
		text, err := uc.parser.Parse(ctx, f.Name, f.Data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.Name, err)
		}
		doc, err := domain.NewDocument(f.Name, f.Type, text)
		if err != nil {
			return nil, fmt.Errorf("document %s: %w", f.Name, err)
		}
		docs = append(docs, *doc)
		fmt.Fprintf(&allText, "=== %s ===\n%s\n\n", f.Name, text)
	}

	// docs is guaranteed non-empty (we return early on parse errors above)
	// companyProfile is guaranteed non-empty (checked at L43)
	analysis, _ := domain.NewTenderAnalysis(docs, companyProfile)

	systemPrompt := `You are a tender analysis expert. Analyze tender documents against a company profile and respond in JSON:
{"verdict":"go|no-go","risk":"low|medium|high","score":0-100,"summary":"..."}`

	userPrompt := fmt.Sprintf("Company profile:\n%s\n\nTender documents:\n%s", companyProfile, allText.String())

	llmResp, err := uc.llm.GenerateCompletion(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm analysis: %w", err)
	}

	var resp analysisResponse
	if err := json.Unmarshal([]byte(llmResp), &resp); err != nil {
		return nil, fmt.Errorf("parse llm response: %w", err)
	}

	verdict, err := domain.ParseVerdict(resp.Verdict)
	if err != nil {
		return nil, fmt.Errorf("parse verdict: %w", err)
	}
	risk, err := domain.ParseRisk(resp.Risk)
	if err != nil {
		return nil, fmt.Errorf("parse risk: %w", err)
	}
	score, err := domain.NewMatchScore(resp.Score)
	if err != nil {
		return nil, fmt.Errorf("parse score: %w", err)
	}

	analysis.SetResult(verdict, risk, score, resp.Summary)
	return analysis, nil
}
