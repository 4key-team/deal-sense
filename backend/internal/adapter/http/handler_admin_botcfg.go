package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
	"github.com/daniil/deal-sense/backend/internal/usecase/botconfig"
)

// AdminBotConfigPutRequest is the JSON shape accepted by PUT
// /api/admin/bot-config. The frontend types should mirror this struct
// 1-to-1.
type AdminBotConfigPutRequest struct {
	Token            string  `json:"token"`
	AllowlistUserIDs []int64 `json:"allowlist_user_ids"`
	LogLevel         string  `json:"log_level"`
}

// AdminBotConfigResponse is the JSON shape returned by GET and PUT. Note
// that Token is never returned raw — only MaskedToken — so the secret
// never escapes the backend once persisted.
type AdminBotConfigResponse struct {
	Configured       bool    `json:"configured"`
	MaskedToken      string  `json:"masked_token,omitempty"`
	AllowlistOpen    bool    `json:"allowlist_open"`
	AllowlistUserIDs []int64 `json:"allowlist_user_ids,omitempty"`
	LogLevel         string  `json:"log_level,omitempty"`
}

// adminBotConfigErrorResponse is the structured 4xx body. The Field key
// lets the frontend highlight the offending input.
type adminBotConfigErrorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

// AdminBotConfigHandler dispatches GET (read masked config) and PUT
// (validate+save) for /api/admin/bot-config. Wire it behind APIKeyAuth in
// cmd/server.
func AdminBotConfigHandler(svc *botconfig.Service, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleAdminBotConfigGet(w, r, svc, logger)
		case http.MethodPut:
			handleAdminBotConfigPut(w, r, svc, logger)
		default:
			w.Header().Set("Allow", "GET, PUT")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}

func handleAdminBotConfigGet(w http.ResponseWriter, r *http.Request, svc *botconfig.Service, logger *slog.Logger) {
	cfg, ok, err := svc.Get(r.Context())
	if err != nil {
		logger.Error("admin botcfg get", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to load bot configuration")
		return
	}
	if !ok {
		writeJSON(w, http.StatusOK, AdminBotConfigResponse{Configured: false})
		return
	}
	writeJSON(w, http.StatusOK, buildBotConfigResponse(cfg))
}

func handleAdminBotConfigPut(w http.ResponseWriter, r *http.Request, svc *botconfig.Service, logger *slog.Logger) {
	var req AdminBotConfigPutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, adminBotConfigErrorResponse{
			Error: "request body is not valid JSON",
		})
		return
	}

	cfg, err := svc.Update(r.Context(), req.Token, req.AllowlistUserIDs, req.LogLevel)
	if err != nil {
		if field, msg, ok := mapBotConfigValidationError(err); ok {
			writeJSON(w, http.StatusBadRequest, adminBotConfigErrorResponse{
				Error: msg,
				Field: field,
			})
			return
		}
		logger.Error("admin botcfg save", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to save bot configuration")
		return
	}

	writeJSON(w, http.StatusOK, buildBotConfigResponse(cfg))
}

// buildBotConfigResponse projects a *domain.BotConfig into the public
// response DTO, masking the secret token.
func buildBotConfigResponse(cfg *domain.BotConfig) AdminBotConfigResponse {
	resp := AdminBotConfigResponse{
		Configured:  true,
		MaskedToken: cfg.MaskedToken(),
		LogLevel:    cfg.LogLevel().String(),
	}
	if cfg.Allowlist().IsOpen() {
		resp.AllowlistOpen = true
	} else {
		resp.AllowlistUserIDs = cfg.Allowlist().Members()
	}
	return resp
}

// mapBotConfigValidationError translates a domain validation failure into
// (field, user-facing message). The third return is false when err is not
// a recognised validation error — caller should treat that as 500.
func mapBotConfigValidationError(err error) (field, msg string, ok bool) {
	switch {
	case errors.Is(err, domain.ErrInvalidBotToken):
		return "token", "bot token must be in the form <digits>:<secret> (Telegram format)", true
	case errors.Is(err, domain.ErrInvalidLogLevel):
		return "log_level", "log_level must be one of: debug, info, warn, error", true
	case errors.Is(err, auth.ErrInvalidUserID):
		return "allowlist_user_ids", "all allowlist user IDs must be positive integers", true
	case errors.Is(err, auth.ErrEmptyAllowlist):
		// ParseAllowlist promotes empty to open mode, so this branch is
		// reachable only if a caller invoked NewRestrictedAllowlist
		// directly with empty input — defensive.
		return "allowlist_user_ids", "allowlist must not be empty in restricted mode", true
	default:
		return "", "", false
	}
}
