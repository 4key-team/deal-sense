package telegram

import "context"

// WelcomeMessage builds the /start reply, optionally prepending an
// onboarding call-to-action when the chat is operating in BYOK mode
// (requireLLM=true) and has not configured /llm yet.
//
// Stub for RED: always returns MsgStart unchanged. GREEN reads the
// LLMSettingsService state and conditionally appends the CTA.
func WelcomeMessage(ctx context.Context, chatID int64, svc LLMSettingsService, requireLLM bool) string {
	_ = ctx
	_ = chatID
	_ = svc
	_ = requireLLM
	return MsgStart
}
