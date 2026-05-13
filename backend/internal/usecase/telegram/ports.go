// Package telegram defines the use-case ports the Telegram bot adapter
// depends on. Interfaces live here (not in domain) so the adapter layer
// can implement them via DIP — the use case owns the contract.
package telegram

import "context"

// AnalyzeTenderRequest is the input to a tender-analysis call. Filename is
// preserved so the backend can sniff the extension; CompanyProfile feeds
// the LLM with team context.
type AnalyzeTenderRequest struct {
	File           []byte
	Filename       string
	CompanyProfile string
}

// ProConItem mirrors one pro/con bullet from the backend JSON.
type ProConItem struct {
	Title string
	Desc  string
}

// RequirementItem mirrors one requirement (label + status) from the backend.
type RequirementItem struct {
	Label  string
	Status string
}

// AnalyzeTenderResponse is the parsed result of POST /api/tender/analyze.
// It is a DTO — the bot does not own the tender domain.
type AnalyzeTenderResponse struct {
	Verdict      string
	Risk         string
	Score        float64
	Summary      string
	Pros         []ProConItem
	Cons         []ProConItem
	Requirements []RequirementItem
	Effort       string
}

// GenerateProposalRequest is the input to a proposal-generation call.
// Template is the .docx/.pdf/.md template file (Filename preserves the
// extension so the backend picks the right engine). Context is a list of
// supplementary materials parsed for the LLM. Params is forwarded as the
// `params` JSON field.
type GenerateProposalRequest struct {
	Template         []byte
	TemplateFilename string
	ContextFiles     []ContextFile
	Params           map[string]string
}

// ContextFile is a single supplementary document for proposal generation.
type ContextFile struct {
	Filename string
	Data     []byte
}

// GeneratedSection mirrors one section status entry from the backend.
type GeneratedSection struct {
	Title  string
	Status string
	Tokens int
}

// GenerateProposalResponse is the parsed result of POST /api/proposal/generate.
// DOCX/PDF/MD payloads are already base64-decoded into raw bytes; nil if the
// backend omitted that format.
type GenerateProposalResponse struct {
	Template string
	Summary  string
	Mode     string
	Sections []GeneratedSection
	DOCX     []byte
	PDF      []byte
	MD       []byte
}

// APIClient is the port the bot uses to invoke Deal Sense backend endpoints.
// Implementations live in adapter/dealsenseapi.
type APIClient interface {
	AnalyzeTender(ctx context.Context, req AnalyzeTenderRequest) (*AnalyzeTenderResponse, error)
	GenerateProposal(ctx context.Context, req GenerateProposalRequest) (*GenerateProposalResponse, error)
}
