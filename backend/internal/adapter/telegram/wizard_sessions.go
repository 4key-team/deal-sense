package telegram

import "sync"

// InMemoryWizardSessions is the production WizardSessions implementation:
// a sync.Map keyed by chatID. State is lost on process restart — acceptable
// because the wizard only takes seconds and a fresh /profile edit restarts it.
type InMemoryWizardSessions struct {
	m sync.Map // chatID (int64) -> *WizardState
}

// NewInMemoryWizardSessions constructs an empty session store.
func NewInMemoryWizardSessions() *InMemoryWizardSessions {
	return &InMemoryWizardSessions{}
}

func (s *InMemoryWizardSessions) Get(chatID int64) (*WizardState, bool) {
	v, ok := s.m.Load(chatID)
	if !ok {
		return nil, false
	}
	state, ok := v.(*WizardState)
	return state, ok
}

func (s *InMemoryWizardSessions) Set(chatID int64, state *WizardState) {
	s.m.Store(chatID, state)
}

func (s *InMemoryWizardSessions) Clear(chatID int64) {
	s.m.Delete(chatID)
}
