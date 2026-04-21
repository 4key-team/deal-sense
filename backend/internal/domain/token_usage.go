package domain

// TokenUsage holds real token consumption from an LLM API call.
type TokenUsage struct {
	promptTokens     int
	completionTokens int
}

func ZeroTokenUsage() TokenUsage { return TokenUsage{} }

func NewTokenUsage(prompt, completion int) TokenUsage {
	return TokenUsage{promptTokens: prompt, completionTokens: completion}
}

func (u TokenUsage) PromptTokens() int     { return u.promptTokens }
func (u TokenUsage) CompletionTokens() int { return u.completionTokens }
func (u TokenUsage) TotalTokens() int      { return u.promptTokens + u.completionTokens }

func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{
		promptTokens:     u.promptTokens + other.promptTokens,
		completionTokens: u.completionTokens + other.completionTokens,
	}
}
