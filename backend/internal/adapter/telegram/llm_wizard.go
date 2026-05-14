package telegram

import "time"

// LLMWizardStep names the question currently waiting for user input in the
// /llm edit flow. The wizard advances StepLLMProvider → StepLLMBaseURL →
// StepLLMAPIKey → StepLLMModel and then constructs an immutable
// domain.LLMSettings from the draft.
type LLMWizardStep string

const (
	StepLLMProvider LLMWizardStep = "llm_provider"
	StepLLMBaseURL  LLMWizardStep = "llm_base_url"
	StepLLMAPIKey   LLMWizardStep = "llm_api_key"
	StepLLMModel    LLMWizardStep = "llm_model"
)

// LLMSettingsDraft accumulates per-chat LLM answers as the wizard walks
// through the steps. Mutable builder — the final immutable
// domain.LLMSettings is constructed via domain.NewLLMSettings after
// StepLLMModel is filled. BaseURL is optional ("" → provider default).
type LLMSettingsDraft struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

// LLMWizardState is the in-memory FSM record for one chat's /llm edit run.
// Kept separate from WizardState so a chat can switch between the profile
// and LLM wizards without state collision.
type LLMWizardState struct {
	ChatID    int64
	Step      LLMWizardStep
	Draft     *LLMSettingsDraft
	StartedAt time.Time
}

// LLMWizardSessions stores per-chat /llm wizard state. The interface keeps
// the LLMHandler test-friendly and lets us swap in alternative stores
// (e.g. a noop for tests) without touching production wiring.
type LLMWizardSessions interface {
	Get(chatID int64) (*LLMWizardState, bool)
	Set(chatID int64, state *LLMWizardState)
	Clear(chatID int64)
}
