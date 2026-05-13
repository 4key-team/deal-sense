package telegram_test

import (
	"context"
	"errors"
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
