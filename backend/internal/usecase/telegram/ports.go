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

// APIClient is the port the bot uses to invoke Deal Sense backend endpoints.
// Implementations live in adapter/dealsenseapi.
type APIClient interface {
	AnalyzeTender(ctx context.Context, req AnalyzeTenderRequest) (*AnalyzeTenderResponse, error)
}
