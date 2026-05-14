package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apphttp "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase/botconfig"
)

const fixtureValidToken = "8829614348:AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bts"

// stubRepo mirrors the in-package tests; we redeclare here because the
// service-layer stub lives in usecase/botconfig and is not exported.
type stubRepo struct {
	loaded    *domain.BotConfig
	hasLoaded bool
	loadErr   error
	saved     *domain.BotConfig
	saveErr   error
}

func (s *stubRepo) Load(context.Context) (*domain.BotConfig, bool, error) {
	return s.loaded, s.hasLoaded, s.loadErr
}
func (s *stubRepo) Save(_ context.Context, cfg *domain.BotConfig) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = cfg
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newHandler(repo botconfig.Repository) stdhttp.Handler {
	return apphttp.AdminBotConfigHandler(botconfig.NewService(repo), discardLogger())
}

// --- GET -------------------------------------------------------------------

func TestAdminBotConfig_Get_NoConfig_ReturnsConfiguredFalse(t *testing.T) {
	h := newHandler(&stubRepo{hasLoaded: false})

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/admin/bot-config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp apphttp.AdminBotConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rec.Body.String())
	}
	if resp.Configured {
		t.Error("Configured must be false when no config saved")
	}
	if resp.MaskedToken != "" {
		t.Errorf("MaskedToken = %q, want empty when not configured", resp.MaskedToken)
	}
}

func TestAdminBotConfig_Get_WithConfig_ReturnsMaskedTokenAndMembers(t *testing.T) {
	cfg, err := domain.NewBotConfig(fixtureValidToken, []int64{42, 100}, "warn")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	h := newHandler(&stubRepo{loaded: cfg, hasLoaded: true})

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/admin/bot-config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apphttp.AdminBotConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rec.Body.String())
	}
	if !resp.Configured {
		t.Error("Configured must be true after load")
	}
	if !strings.HasPrefix(resp.MaskedToken, "8829614348:") {
		t.Errorf("MaskedToken = %q, want bot ID prefix", resp.MaskedToken)
	}
	if strings.Contains(resp.MaskedToken, "AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q") {
		t.Errorf("MaskedToken must not leak raw secret middle, got %q", resp.MaskedToken)
	}
	if resp.AllowlistOpen {
		t.Error("AllowlistOpen must be false for restricted config")
	}
	if len(resp.AllowlistUserIDs) != 2 {
		t.Errorf("AllowlistUserIDs len = %d, want 2 (got %v)", len(resp.AllowlistUserIDs), resp.AllowlistUserIDs)
	}
	if resp.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn", resp.LogLevel)
	}
}

func TestAdminBotConfig_Get_OpenAllowlist_FlagsOpenTrue(t *testing.T) {
	cfg, err := domain.NewBotConfig(fixtureValidToken, nil, "info")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	h := newHandler(&stubRepo{loaded: cfg, hasLoaded: true})

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/admin/bot-config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp apphttp.AdminBotConfigResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !resp.AllowlistOpen {
		t.Error("AllowlistOpen must be true for open-mode config")
	}
	if len(resp.AllowlistUserIDs) != 0 {
		t.Errorf("AllowlistUserIDs must be empty for open mode, got %v", resp.AllowlistUserIDs)
	}
}

func TestAdminBotConfig_Get_RepositoryError_Returns500(t *testing.T) {
	h := newHandler(&stubRepo{loadErr: errors.New("disk on fire")})

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/admin/bot-config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// --- PUT -------------------------------------------------------------------

func putRequest(t *testing.T, body any) *stdhttp.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	r := httptest.NewRequest(stdhttp.MethodPut, "/api/admin/bot-config", bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestAdminBotConfig_Put_ValidRestricted_PersistsAndReturns200(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo)

	req := putRequest(t, apphttp.AdminBotConfigPutRequest{
		Token:            fixtureValidToken,
		AllowlistUserIDs: []int64{42, 100},
		LogLevel:         "warn",
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.saved == nil {
		t.Fatal("PUT must persist the new config")
	}
	if repo.saved.Token() != fixtureValidToken {
		t.Errorf("saved token wrong: %q", repo.saved.Token())
	}
	var resp apphttp.AdminBotConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Configured {
		t.Error("response Configured must be true after PUT")
	}
}

func TestAdminBotConfig_Put_EmptyAllowlist_AcceptedAsOpen(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo)

	req := putRequest(t, apphttp.AdminBotConfigPutRequest{
		Token:            fixtureValidToken,
		AllowlistUserIDs: nil,
		LogLevel:         "info",
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.saved == nil {
		t.Fatal("PUT with empty allowlist must still persist (open mode)")
	}
	if !repo.saved.Allowlist().IsOpen() {
		t.Error("saved allowlist must be open mode")
	}
}

func TestAdminBotConfig_Put_InvalidToken_Returns400AndDoesNotPersist(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo)

	req := putRequest(t, apphttp.AdminBotConfigPutRequest{
		Token:            "not-a-token",
		AllowlistUserIDs: []int64{42},
		LogLevel:         "info",
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if repo.saved != nil {
		t.Error("invalid input must not persist")
	}
	// Response must identify the offending field.
	if !strings.Contains(rec.Body.String(), "token") {
		t.Errorf("error body must mention 'token', got %s", rec.Body.String())
	}
}

func TestAdminBotConfig_Put_InvalidLogLevel_Returns400(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo)

	req := putRequest(t, apphttp.AdminBotConfigPutRequest{
		Token:            fixtureValidToken,
		AllowlistUserIDs: []int64{42},
		LogLevel:         "verbose",
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "log_level") {
		t.Errorf("error body must mention 'log_level', got %s", rec.Body.String())
	}
}

func TestAdminBotConfig_Put_InvalidUserID_Returns400(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo)

	req := putRequest(t, apphttp.AdminBotConfigPutRequest{
		Token:            fixtureValidToken,
		AllowlistUserIDs: []int64{0, -1},
		LogLevel:         "info",
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminBotConfig_Put_MalformedJSON_Returns400(t *testing.T) {
	repo := &stubRepo{}
	h := newHandler(repo)

	r := httptest.NewRequest(stdhttp.MethodPut, "/api/admin/bot-config", strings.NewReader("{not json"))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if rec.Code != stdhttp.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestAdminBotConfig_Put_RepositorySaveError_Returns500(t *testing.T) {
	repo := &stubRepo{saveErr: errors.New("disk full")}
	h := newHandler(repo)

	req := putRequest(t, apphttp.AdminBotConfigPutRequest{
		Token:            fixtureValidToken,
		AllowlistUserIDs: []int64{42},
		LogLevel:         "info",
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// --- Method routing --------------------------------------------------------

func TestAdminBotConfig_UnsupportedMethod_Returns405(t *testing.T) {
	h := newHandler(&stubRepo{})

	for _, m := range []string{stdhttp.MethodPost, stdhttp.MethodDelete, stdhttp.MethodPatch} {
		t.Run(m, func(t *testing.T) {
			req := httptest.NewRequest(m, "/api/admin/bot-config", nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != stdhttp.StatusMethodNotAllowed {
				t.Errorf("%s status = %d, want 405", m, rec.Code)
			}
		})
	}
}
