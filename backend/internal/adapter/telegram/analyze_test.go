package telegram_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain"
	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// --- test doubles --------------------------------------------------------

type fakeReplier struct {
	mu      sync.Mutex
	chatIDs []int64
	texts   []string
	err     error
}

func (f *fakeReplier) Reply(ctx context.Context, chatID int64, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chatIDs = append(f.chatIDs, chatID)
	f.texts = append(f.texts, text)
	return f.err
}

// ReplyDocument is a no-op stub for /analyze tests — /generate tests in
// generate_test.go use a richer fake.
func (f *fakeReplier) ReplyDocument(context.Context, int64, string, []byte, string) error {
	return nil
}

type fakeAPI struct {
	gotReq usecase.AnalyzeTenderRequest
	resp   *usecase.AnalyzeTenderResponse
	err    error
	calls  int
}

func (f *fakeAPI) AnalyzeTender(ctx context.Context, req usecase.AnalyzeTenderRequest) (*usecase.AnalyzeTenderResponse, error) {
	f.calls++
	f.gotReq = req
	return f.resp, f.err
}

// GenerateProposal stub — not exercised by these tests.
func (f *fakeAPI) GenerateProposal(context.Context, usecase.GenerateProposalRequest) (*usecase.GenerateProposalResponse, error) {
	return nil, nil
}

// --- tests ---------------------------------------------------------------

func TestAnalyzeHandler_NoDocument_AsksForFile(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{}
	h := telegram.NewAnalyzeHandler(api, nil, rep, "Software dev")

	err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, UserID: 7, Text: "/analyze"})
	if err != nil {
		t.Fatalf("Handle returned err: %v", err)
	}
	if api.calls != 0 {
		t.Errorf("api.AnalyzeTender called %d times, want 0", api.calls)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if rep.chatIDs[0] != 42 {
		t.Errorf("chatID = %d, want 42", rep.chatIDs[0])
	}
	// Russian hint about replying with a file.
	if !strings.Contains(rep.texts[0], "файл") {
		t.Errorf("reply = %q, want to mention 'файл'", rep.texts[0])
	}
}

func TestAnalyzeHandler_Success_CallsAPIAndReplies(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{
		resp: &usecase.AnalyzeTenderResponse{
			Verdict: "HIGH",
			Score:   0.82,
			Summary: "Strong fit",
			Pros:    []usecase.ProConItem{{Title: "p1", Desc: "d1"}},
			Cons:    []usecase.ProConItem{{Title: "c1", Desc: "d1"}},
		},
	}
	h := telegram.NewAnalyzeHandler(api, nil, rep, "Software dev")

	doc := &telegram.Document{FileID: "f1", Filename: "tender.pdf", Data: []byte("PDF")}
	err := h.Handle(context.Background(), &telegram.Update{
		ChatID: 42, UserID: 7, Text: "/analyze", Document: doc,
	})
	if err != nil {
		t.Fatalf("Handle returned err: %v", err)
	}
	if api.calls != 1 {
		t.Fatalf("api.AnalyzeTender called %d times, want 1", api.calls)
	}
	if api.gotReq.Filename != "tender.pdf" {
		t.Errorf("filename = %q, want tender.pdf", api.gotReq.Filename)
	}
	if string(api.gotReq.File) != "PDF" {
		t.Errorf("file body = %q, want PDF", string(api.gotReq.File))
	}
	if api.gotReq.CompanyProfile != "Software dev" {
		t.Errorf("company profile = %q", api.gotReq.CompanyProfile)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if !strings.Contains(rep.texts[0], "HIGH") {
		t.Errorf("reply missing verdict, got %q", rep.texts[0])
	}
	if !strings.Contains(rep.texts[0], "0.82") {
		t.Errorf("reply missing score, got %q", rep.texts[0])
	}
}

func TestAnalyzeHandler_APIError_ReportsToUser(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{err: errors.New("backend exploded")}
	h := telegram.NewAnalyzeHandler(api, nil, rep, "Software dev")

	doc := &telegram.Document{Filename: "t.pdf", Data: []byte("x")}
	err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc, Text: "/analyze"})
	if err != nil {
		t.Fatalf("Handle returned err: %v", err)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times", len(rep.texts))
	}
	if !strings.Contains(strings.ToLower(rep.texts[0]), "ошибка") {
		t.Errorf("reply should mention error, got %q", rep.texts[0])
	}
}

func TestAnalyzeHandler_ReplierError_Propagates(t *testing.T) {
	repErr := errors.New("network blip")
	rep := &fakeReplier{err: repErr}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	h := telegram.NewAnalyzeHandler(api, nil, rep, "Software dev")

	err := h.Handle(context.Background(), &telegram.Update{
		ChatID: 1, Document: &telegram.Document{Filename: "x.pdf", Data: []byte("x")},
	})
	if !errors.Is(err, repErr) {
		t.Errorf("Handle err = %v, want to wrap %v", err, repErr)
	}
}

// --- per-chat profile tests ---------------------------------------------

func TestAnalyzeHandler_NoProfileForChat_UsesFallback(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	store := newFakeProfileStore()
	h := telegram.NewAnalyzeHandler(api, store, rep, "Fallback Co")

	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.gotReq.CompanyProfile != "Fallback Co" {
		t.Errorf("CompanyProfile = %q, want fallback when no profile saved", api.gotReq.CompanyProfile)
	}
}

func TestAnalyzeHandler_ProfileExists_UsesRenderedProfile(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "HIGH"}}
	prof, err := domain.NewCompanyProfile("Acme", "15", "", []string{"Go", "React"}, nil, nil, "", "")
	if err != nil {
		t.Fatalf("NewCompanyProfile: %v", err)
	}
	store := newFakeProfileStore()
	store.data[42] = prof
	h := telegram.NewAnalyzeHandler(api, store, rep, "Fallback Co")

	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(api.gotReq.CompanyProfile, "Company: Acme") {
		t.Errorf("CompanyProfile = %q, want rendered per-chat profile", api.gotReq.CompanyProfile)
	}
	if !strings.Contains(api.gotReq.CompanyProfile, "Tech stack: Go, React") {
		t.Errorf("CompanyProfile missing tech stack, got %q", api.gotReq.CompanyProfile)
	}
}

func TestAnalyzeHandler_ProfileForDifferentChat_FallbackUsed(t *testing.T) {
	// Confirms per-chat isolation: profile saved for chat 7 must NOT leak
	// into chat 42's analyze call.
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	prof, _ := domain.NewCompanyProfile("Leak", "", "", nil, nil, nil, "", "")
	store := newFakeProfileStore()
	store.data[7] = prof
	h := telegram.NewAnalyzeHandler(api, store, rep, "Fallback Co")

	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if strings.Contains(api.gotReq.CompanyProfile, "Leak") {
		t.Errorf("chat 7 profile leaked into chat 42 analyze: %q", api.gotReq.CompanyProfile)
	}
	if api.gotReq.CompanyProfile != "Fallback Co" {
		t.Errorf("CompanyProfile = %q, want fallback", api.gotReq.CompanyProfile)
	}
}

func TestAnalyzeHandler_StoreReadError_FallsBackInsteadOfFailing(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	store := newFakeProfileStore()
	store.getErr = errors.New("disk gone")
	h := telegram.NewAnalyzeHandler(api, store, rep, "Fallback Co")

	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.calls != 1 {
		t.Fatalf("api.calls = %d, want 1 (analyze should proceed despite profile read error)", api.calls)
	}
	if api.gotReq.CompanyProfile != "Fallback Co" {
		t.Errorf("CompanyProfile = %q, want fallback after store read error", api.gotReq.CompanyProfile)
	}
}

func TestAnalyzeHandler_NilProfileStore_UsesFallback(t *testing.T) {
	// Defensive: until wiring is complete a nil ProfileStore is acceptable.
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	h := telegram.NewAnalyzeHandler(api, nil, rep, "Fallback Co")
	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.gotReq.CompanyProfile != "Fallback Co" {
		t.Errorf("CompanyProfile = %q, want fallback when ProfileStore nil", api.gotReq.CompanyProfile)
	}
}

func TestAnalyzeHandler_ProfileFor_LogsLookupOutcome(t *testing.T) {
	// Confirms profileFor emits one log record per lookup branch so operators
	// can tell "per-chat profile applied" from "fallback".
	prof, _ := domain.NewCompanyProfile("Acme", "", "", nil, nil, nil, "", "")

	tests := []struct {
		name        string
		store       usecase.ProfileStore
		wantMsg     string
		wantLevel   slog.Level
		wantProfile string
	}{
		{
			name:        "profile present → debug + per-chat profile applied",
			store:       func() *fakeProfileStore { s := newFakeProfileStore(); s.data[42] = prof; return s }(),
			wantMsg:     "per-chat profile applied",
			wantLevel:   slog.LevelDebug,
			wantProfile: prof.Render(),
		},
		{
			name:        "no profile → info + fallback",
			store:       newFakeProfileStore(),
			wantMsg:     "no per-chat profile; using fallback",
			wantLevel:   slog.LevelInfo,
			wantProfile: "Fallback Co",
		},
		{
			name:        "store error → error + fallback",
			store:       func() *fakeProfileStore { s := newFakeProfileStore(); s.getErr = errors.New("disk gone"); return s }(),
			wantMsg:     "profile lookup failed; using fallback",
			wantLevel:   slog.LevelError,
			wantProfile: "Fallback Co",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rh := &recordingHandler{}
			logger := slog.New(rh)
			rep := &fakeReplier{}
			api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
			h := telegram.NewAnalyzeHandler(api, tt.store, rep, "Fallback Co", telegram.WithAnalyzeLogger(logger))

			doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}
			if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
				t.Fatalf("Handle: %v", err)
			}
			if api.gotReq.CompanyProfile != tt.wantProfile {
				t.Errorf("CompanyProfile = %q, want %q", api.gotReq.CompanyProfile, tt.wantProfile)
			}
			rec := recordOfMessage(rh, tt.wantMsg)
			if rec == nil {
				seen := make([]string, 0, len(rh.records))
				for _, r := range rh.records {
					seen = append(seen, r.Message)
				}
				t.Fatalf("no %q log record; saw %v", tt.wantMsg, seen)
			}
			if rec.Level != tt.wantLevel {
				t.Errorf("level = %s, want %s", rec.Level, tt.wantLevel)
			}
			chatID, ok := attrValue(rec, "chat_id")
			if !ok || chatID.Int64() != 42 {
				t.Errorf("chat_id attr missing or wrong: ok=%v val=%v", ok, chatID)
			}
		})
	}
}

func TestAnalyzeHandler_NilLogger_DoesNotPanic(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	h := telegram.NewAnalyzeHandler(api, newFakeProfileStore(), rep, "Fallback Co")
	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
}

func TestFormatAnalyzeReply(t *testing.T) {
	resp := &usecase.AnalyzeTenderResponse{
		Verdict: "HIGH",
		Risk:    "low",
		Score:   0.75,
		Summary: "Good match",
		Pros: []usecase.ProConItem{
			{Title: "Established stack", Desc: "Familiar tech"},
		},
		Cons: []usecase.ProConItem{
			{Title: "Tight deadline", Desc: "2 weeks"},
		},
		Requirements: []usecase.RequirementItem{
			{Label: "Лицензия ИБ", Status: "missing"},
		},
		Effort: "2-3 weeks",
	}
	out := telegram.FormatAnalyzeReply(resp)
	for _, want := range []string{"HIGH", "0.75", "Good match", "Established stack", "Tight deadline", "Лицензия ИБ", "missing", "2-3 weeks"} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatAnalyzeReply missing %q\nout: %s", want, out)
		}
	}
}

// --- per-chat LLM override forwarding -----------------------------------

func TestAnalyzeHandler_LLMService_NotConfigured_NoOverride(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	// No WithAnalyzeLLMService — handler must not invent an override.
	h := telegram.NewAnalyzeHandler(api, newFakeProfileStore(), rep, "Fallback Co")
	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}

	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.gotReq.LLM != (usecase.LLMOverride{}) {
		t.Errorf("LLM = %+v, want zero (no override) when service not wired", api.gotReq.LLM)
	}
}

func TestAnalyzeHandler_LLMService_NoSettingsForChat_NoOverride(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	llm := newFakeLLMService() // empty store
	h := telegram.NewAnalyzeHandler(api, newFakeProfileStore(), rep, "Fallback Co",
		telegram.WithAnalyzeLLMService(llm))
	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}

	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.gotReq.LLM != (usecase.LLMOverride{}) {
		t.Errorf("LLM = %+v, want zero when chat has no settings", api.gotReq.LLM)
	}
}

func TestAnalyzeHandler_LLMService_ChatHasSettings_PopulatesOverride(t *testing.T) {
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	llm := newFakeLLMService()
	cfg, err := domain.NewLLMSettings("openai", "https://openrouter.ai/api/v1", "sk-test1234", "anthropic/claude-sonnet-4")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	llm.data[42] = cfg
	h := telegram.NewAnalyzeHandler(api, newFakeProfileStore(), rep, "Fallback Co",
		telegram.WithAnalyzeLLMService(llm))
	doc := &telegram.Document{Filename: "x.pdf", Data: []byte("x")}

	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	want := usecase.LLMOverride{
		Provider: "openai",
		BaseURL:  "https://openrouter.ai/api/v1",
		APIKey:   "sk-test1234",
		Model:    "anthropic/claude-sonnet-4",
	}
	if api.gotReq.LLM != want {
		t.Errorf("LLM = %+v, want %+v", api.gotReq.LLM, want)
	}
}
