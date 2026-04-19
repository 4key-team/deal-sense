package usecase

import (
	"context"
	"fmt"
	"maps"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

type GenerateProposal struct {
	llm      LLMProvider
	template TemplateEngine
}

func NewGenerateProposal(llm LLMProvider, template TemplateEngine) *GenerateProposal {
	return &GenerateProposal{llm: llm, template: template}
}

func (uc *GenerateProposal) Execute(
	ctx context.Context,
	templateName string,
	templateData []byte,
	params map[string]string,
) (*domain.Proposal, error) {
	proposal, err := domain.NewProposal(templateName, templateData, params)
	if err != nil {
		return nil, err
	}

	systemPrompt := "You are a commercial proposal generator. Given template parameters, generate appropriate values for each placeholder in JSON format."
	userPrompt := fmt.Sprintf("Template: %s\nParameters: %v\nGenerate values for all placeholders.", templateName, params)

	llmResponse, err := uc.llm.GenerateCompletion(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	// Merge LLM-generated values with user params
	mergedParams := make(map[string]string)
	maps.Copy(mergedParams, params)
	mergedParams["_llm_response"] = llmResponse

	filled, err := uc.template.Fill(ctx, templateData, mergedParams)
	if err != nil {
		return nil, fmt.Errorf("template fill: %w", err)
	}

	proposal.SetResult(filled)
	return proposal, nil
}
