package telegram

import "context"

// WelcomeMessage builds the /start reply, optionally prepending an
// onboarding call-to-action when the chat is operating in BYOK mode
// (requireLLM=true) and has not configured /llm yet.
//
// Lookup errors in BYOK mode fail closed: we still append the CTA so a
// transient store outage doesn't hide the onboarding cue. In legacy
// mode (requireLLM=false) the env LLM_* keys are the fallback and the
// user does not need to set up anything, so the CTA stays off.
func WelcomeMessage(ctx context.Context, chatID int64, svc LLMSettingsService, requireLLM bool) string {
	if !requireLLM {
		return MsgStart
	}
	if svc == nil {
		// No way to check — be conservative: do NOT nag a user when the
		// runtime can't tell whether they're configured. Operators wiring
		// require=true must also wire a service; this branch is purely
		// defensive against mis-wiring.
		return MsgStart
	}
	_, ok, err := svc.Get(ctx, chatID)
	if err != nil {
		return MsgStart + msgStartOnboardingCTA
	}
	if ok {
		return MsgStart
	}
	return MsgStart + msgStartOnboardingCTA
}
