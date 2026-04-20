package usecase

import "context"

type ListModels struct {
	llm LLMProvider
}

func NewListModels(llm LLMProvider) *ListModels {
	return &ListModels{llm: llm}
}

func (uc *ListModels) Execute(ctx context.Context) ([]string, error) {
	return uc.llm.ListModels(ctx)
}
