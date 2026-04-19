package usecase

import "context"

type ConnectionStatus struct {
	Provider string
	OK       bool
}

type CheckLLMConnection struct {
	llm LLMProvider
}

func NewCheckLLMConnection(llm LLMProvider) *CheckLLMConnection {
	return &CheckLLMConnection{llm: llm}
}

func (uc *CheckLLMConnection) Execute(ctx context.Context) (*ConnectionStatus, error) {
	if err := uc.llm.CheckConnection(ctx); err != nil {
		return nil, err
	}
	return &ConnectionStatus{
		Provider: uc.llm.Name(),
		OK:       true,
	}, nil
}
