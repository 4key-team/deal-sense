package telegram

import (
	"context"
	"fmt"
	"strings"

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

// Handle routes /generate. Without a template document it asks the user
// to reply with one; with a document it calls the backend, attaches the
// resulting DOCX (or falls back to MD), and posts a caption summary.
func (h *GenerateHandler) Handle(ctx context.Context, u *Update) error {
	if u.Document == nil {
		return h.replier.Reply(ctx, u.ChatID, msgAttachTemplate)
	}

	resp, err := h.api.GenerateProposal(ctx, usecase.GenerateProposalRequest{
		Template:         u.Document.Data,
		TemplateFilename: u.Document.Filename,
	})
	if err != nil {
		return h.replier.Reply(ctx, u.ChatID, fmt.Sprintf("%s %s", msgGenerationErrPrefix, err.Error()))
	}

	data, filename := pickArtifact(resp, u.Document.Filename)
	if len(data) == 0 {
		// Nothing to send back — surface mode+summary as text.
		return h.replier.Reply(ctx, u.ChatID, fmt.Sprintf(msgGenerateCaptionFmt, resp.Mode, len(resp.Sections)))
	}

	caption := fmt.Sprintf(msgGenerateCaptionFmt, resp.Mode, len(resp.Sections))
	return h.replier.ReplyDocument(ctx, u.ChatID, filename, data, caption)
}

// pickArtifact prefers DOCX, then PDF, then MD. Filename swaps the
// extension on the template name to match the chosen artifact.
func pickArtifact(resp *usecase.GenerateProposalResponse, templateName string) ([]byte, string) {
	base := stripExt(templateName)
	if base == "" {
		base = "proposal"
	}
	switch {
	case len(resp.DOCX) > 0:
		return resp.DOCX, base + ".docx"
	case len(resp.PDF) > 0:
		return resp.PDF, base + ".pdf"
	case len(resp.MD) > 0:
		return resp.MD, base + ".md"
	}
	return nil, ""
}

func stripExt(name string) string {
	if i := strings.LastIndex(name, "."); i > 0 {
		return name[:i]
	}
	return name
}
