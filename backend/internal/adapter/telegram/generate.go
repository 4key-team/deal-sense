package telegram

import (
	"context"

	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// GenerateHandler implements the /generate command flow.
type GenerateHandler struct {
	api     usecase.APIClient
	replier Replier
}

// NewGenerateHandler wires the dependencies for /generate.
func NewGenerateHandler(api usecase.APIClient, replier Replier) *GenerateHandler {
	return &GenerateHandler{api: api, replier: replier}
}

// Handle stub for RED step — returns nil without contacting any dependency
// so every behavioural test fails on the missing side-effects.
func (h *GenerateHandler) Handle(ctx context.Context, u *Update) error {
	return nil
}
