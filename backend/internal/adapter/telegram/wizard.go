package telegram

import "time"

// WizardStep names the question currently waiting for user input. The wizard
// advances from StepName through StepExtra and lands on StepConfirm, after
// which the draft is persisted and the session cleared.
type WizardStep string

const (
	StepName            WizardStep = "name"
	StepTeamSize        WizardStep = "team_size"
	StepExperience      WizardStep = "experience"
	StepTechStack       WizardStep = "tech_stack"
	StepCertifications  WizardStep = "certifications"
	StepSpecializations WizardStep = "specializations"
	StepKeyClients      WizardStep = "key_clients"
	StepExtra           WizardStep = "extra"
)

// ProfileDraft accumulates user answers as the wizard walks through the
// steps. It is intentionally a mutable builder — the final immutable
// domain.CompanyProfile is constructed via domain.NewCompanyProfile when
// StepConfirm is reached.
type ProfileDraft struct {
	Name            string
	TeamSize        string
	Experience      string
	TechStack       []string
	Certifications  []string
	Specializations []string
	KeyClients      string
	Extra           string
}

// WizardState is the in-memory FSM record for one chat's profile-wizard run.
type WizardState struct {
	ChatID    int64
	Step      WizardStep
	Draft     *ProfileDraft
	StartedAt time.Time
}

// WizardSessions stores per-chat wizard state. Implementations live alongside
// the adapter; the interface keeps the handler test-friendly.
type WizardSessions interface {
	Get(chatID int64) (*WizardState, bool)
	Set(chatID int64, state *WizardState)
	Clear(chatID int64)
}
