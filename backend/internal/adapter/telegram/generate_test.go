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

// --- richer fake replier (records both text + document replies) ---------

type sentDoc struct {
	chatID   int64
	filename string
	data     []byte
	caption  string
}

type recordingReplier struct {
	mu        sync.Mutex
	textCalls []string
	docCalls  []sentDoc
	textErr   error
	docErr    error
}

func (r *recordingReplier) Reply(ctx context.Context, chatID int64, text string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.textCalls = append(r.textCalls, text)
	return r.textErr
}

func (r *recordingReplier) ReplyDocument(ctx context.Context, chatID int64, filename string, data []byte, caption string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docCalls = append(r.docCalls, sentDoc{chatID: chatID, filename: filename, data: data, caption: caption})
	return r.docErr
}

// --- richer fake API client that also records GenerateProposal -----------

type generateFakeAPI struct {
	gotReq usecase.GenerateProposalRequest
	resp   *usecase.GenerateProposalResponse
	err    error
	calls  int
}

func (f *generateFakeAPI) AnalyzeTender(context.Context, usecase.AnalyzeTenderRequest) (*usecase.AnalyzeTenderResponse, error) {
	return nil, errors.New("not used")
}

func (f *generateFakeAPI) GenerateProposal(ctx context.Context, req usecase.GenerateProposalRequest) (*usecase.GenerateProposalResponse, error) {
	f.calls++
	f.gotReq = req
	return f.resp, f.err
}

// --- tests ---------------------------------------------------------------

func TestGenerateHandler_NoDocument_AsksForTemplate(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{}
	h := telegram.NewGenerateHandler(api, rep)

	err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Text: "/generate"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.calls != 0 {
		t.Errorf("api.GenerateProposal called %d times, want 0", api.calls)
	}
	if len(rep.textCalls) != 1 {
		t.Fatalf("text replies = %d, want 1", len(rep.textCalls))
	}
	if !strings.Contains(rep.textCalls[0], "шаблон") {
		t.Errorf("reply = %q, want to mention 'шаблон'", rep.textCalls[0])
	}
}

func TestGenerateHandler_Success_SendsDocxDocument(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{
		resp: &usecase.GenerateProposalResponse{
			Template: "tpl.docx",
			Mode:     "placeholder",
			Sections: []usecase.GeneratedSection{
				{Title: "Intro", Status: "ok", Tokens: 120},
				{Title: "Body", Status: "ok", Tokens: 240},
			},
			DOCX: []byte("DOCX BYTES"),
		},
	}
	h := telegram.NewGenerateHandler(api, rep)

	doc := &telegram.Document{FileID: "f1", Filename: "tpl.docx", Data: []byte("TEMPLATE")}
	err := h.Handle(context.Background(), &telegram.Update{ChatID: 7, Document: doc})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.calls != 1 {
		t.Fatalf("api.GenerateProposal called %d times, want 1", api.calls)
	}
	if api.gotReq.TemplateFilename != "tpl.docx" {
		t.Errorf("template filename = %q", api.gotReq.TemplateFilename)
	}
	if string(api.gotReq.Template) != "TEMPLATE" {
		t.Errorf("template body = %q", string(api.gotReq.Template))
	}
	if len(rep.docCalls) != 1 {
		t.Fatalf("doc replies = %d, want 1", len(rep.docCalls))
	}
	if rep.docCalls[0].chatID != 7 {
		t.Errorf("chatID = %d, want 7", rep.docCalls[0].chatID)
	}
	if rep.docCalls[0].filename != "tpl.docx" {
		t.Errorf("filename = %q, want tpl.docx", rep.docCalls[0].filename)
	}
	if string(rep.docCalls[0].data) != "DOCX BYTES" {
		t.Errorf("data = %q, want DOCX BYTES", string(rep.docCalls[0].data))
	}
	if !strings.Contains(rep.docCalls[0].caption, "placeholder") || !strings.Contains(rep.docCalls[0].caption, "2") {
		t.Errorf("caption = %q, want to mention mode 'placeholder' and section count 2", rep.docCalls[0].caption)
	}
}

func TestGenerateHandler_APIError_RepliesText(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{err: errors.New("backend broke")}
	h := telegram.NewGenerateHandler(api, rep)

	doc := &telegram.Document{Filename: "t.docx", Data: []byte("T")}
	err := h.Handle(context.Background(), &telegram.Update{ChatID: 1, Document: doc})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(rep.docCalls) != 0 {
		t.Errorf("document should not be sent on API error")
	}
	if len(rep.textCalls) != 1 {
		t.Fatalf("text replies = %d, want 1", len(rep.textCalls))
	}
	if !strings.Contains(strings.ToLower(rep.textCalls[0]), "ошибка") {
		t.Errorf("reply = %q, want to mention 'ошибка'", rep.textCalls[0])
	}
}

func TestGenerateHandler_ReplierError_Propagates(t *testing.T) {
	docErr := errors.New("doc send failed")
	rep := &recordingReplier{docErr: docErr}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{DOCX: []byte("X")}}
	h := telegram.NewGenerateHandler(api, rep)

	err := h.Handle(context.Background(), &telegram.Update{
		ChatID:   1,
		Document: &telegram.Document{Filename: "t.docx", Data: []byte("T")},
	})
	if !errors.Is(err, docErr) {
		t.Errorf("err = %v, want to wrap %v", err, docErr)
	}
}

func TestGenerateHandler_NoDocx_FallsBackToPDF(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{
		resp: &usecase.GenerateProposalResponse{
			Template: "tpl.docx",
			Mode:     "placeholder",
			PDF:      []byte("PDF BYTES"),
		},
	}
	h := telegram.NewGenerateHandler(api, rep)

	err := h.Handle(context.Background(), &telegram.Update{
		ChatID:   1,
		Document: &telegram.Document{Filename: "tpl.docx", Data: []byte("T")},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(rep.docCalls) != 1 {
		t.Fatalf("expected 1 doc reply (PDF fallback), got %d", len(rep.docCalls))
	}
	if !strings.HasSuffix(rep.docCalls[0].filename, ".pdf") {
		t.Errorf("filename = %q, want to end with .pdf", rep.docCalls[0].filename)
	}
	if string(rep.docCalls[0].data) != "PDF BYTES" {
		t.Errorf("data = %q, want PDF BYTES", string(rep.docCalls[0].data))
	}
}

func TestGenerateHandler_NoDocxNoPDF_FallsBackToMD(t *testing.T) {
	// Generative templates may produce only MD; bot uploads it as .md.
	rep := &recordingReplier{}
	api := &generateFakeAPI{
		resp: &usecase.GenerateProposalResponse{
			Template: "tpl.md",
			Mode:     "generative",
			MD:       []byte("# Proposal\n\nBody"),
		},
	}
	h := telegram.NewGenerateHandler(api, rep)

	err := h.Handle(context.Background(), &telegram.Update{
		ChatID:   1,
		Document: &telegram.Document{Filename: "tpl.md", Data: []byte("T")},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(rep.docCalls) != 1 {
		t.Fatalf("expected 1 doc reply (MD fallback), got %d", len(rep.docCalls))
	}
	if !strings.HasSuffix(rep.docCalls[0].filename, ".md") {
		t.Errorf("filename = %q, want to end with .md", rep.docCalls[0].filename)
	}
}

func TestGenerateHandler_NoArtifacts_RepliesTextSummary(t *testing.T) {
	// When the backend returns no DOCX/PDF/MD payload at all the handler
	// must still acknowledge with a text summary so the user isn't left
	// guessing.
	rep := &recordingReplier{}
	api := &generateFakeAPI{
		resp: &usecase.GenerateProposalResponse{
			Mode:     "placeholder",
			Sections: []usecase.GeneratedSection{{Title: "Intro"}, {Title: "Body"}},
			// no DOCX/PDF/MD
		},
	}
	h := telegram.NewGenerateHandler(api, rep)

	err := h.Handle(context.Background(), &telegram.Update{
		ChatID:   1,
		Document: &telegram.Document{Filename: "tpl.docx", Data: []byte("T")},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(rep.docCalls) != 0 {
		t.Errorf("expected no document reply when artifact empty, got %d", len(rep.docCalls))
	}
	if len(rep.textCalls) != 1 {
		t.Fatalf("text replies = %d, want 1", len(rep.textCalls))
	}
	// Caption format mentions mode + section count.
	if !strings.Contains(rep.textCalls[0], "placeholder") || !strings.Contains(rep.textCalls[0], "2") {
		t.Errorf("summary = %q, want mode 'placeholder' and section count 2", rep.textCalls[0])
	}
}

func TestGenerateHandler_FilenameFallbackWhenTemplateHasNoExt(t *testing.T) {
	// stripExt fallback path: template with no extension → filename swap
	// produces "proposal" base + artifact extension. Covers the
	// no-dot branch.
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{DOCX: []byte("X")}}
	h := telegram.NewGenerateHandler(api, rep)

	err := h.Handle(context.Background(), &telegram.Update{
		ChatID:   1,
		Document: &telegram.Document{Filename: "noextension", Data: []byte("T")},
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(rep.docCalls) != 1 {
		t.Fatalf("doc replies = %d", len(rep.docCalls))
	}
	// "noextension" has no dot → stripExt returns whole name → base+".docx"
	if rep.docCalls[0].filename != "noextension.docx" {
		t.Errorf("filename = %q, want noextension.docx", rep.docCalls[0].filename)
	}
}

// --- per-chat LLM override forwarding -----------------------------------

func TestGenerateHandler_LLMService_NotConfigured_NoOverride(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{Mode: "placeholder"}}
	h := telegram.NewGenerateHandler(api, rep) // no LLM service

	doc := &telegram.Document{Filename: "t.docx", Data: []byte("DOCX")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.gotReq.LLM != (usecase.LLMOverride{}) {
		t.Errorf("LLM = %+v, want zero when service not wired", api.gotReq.LLM)
	}
}

func TestGenerateHandler_LLMService_NoSettingsForChat_NoOverride(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{Mode: "placeholder"}}
	llm := newFakeLLMService()
	h := telegram.NewGenerateHandler(api, rep, telegram.WithGenerateLLMService(llm))

	doc := &telegram.Document{Filename: "t.docx", Data: []byte("DOCX")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.gotReq.LLM != (usecase.LLMOverride{}) {
		t.Errorf("LLM = %+v, want zero when chat has no settings", api.gotReq.LLM)
	}
}

// --- BYOK enforce mode -----------------------------------------------------

func TestGenerateHandler_RequireLLM_NoOverride_BlocksAndPromptsLLMEdit(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{}
	llm := newFakeLLMService() // empty
	h := telegram.NewGenerateHandler(api, rep,
		telegram.WithGenerateLLMService(llm),
		telegram.WithGenerateRequirePerChatLLM(true),
	)
	doc := &telegram.Document{Filename: "t.docx", Data: []byte("DOCX")}

	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.calls != 0 {
		t.Errorf("backend GenerateProposal called %d times, want 0", api.calls)
	}
	if len(rep.docCalls) != 0 {
		t.Errorf("doc replies = %d, want 0 (must short-circuit before sending)", len(rep.docCalls))
	}
	if len(rep.textCalls) != 1 {
		t.Fatalf("text replies = %d, want 1", len(rep.textCalls))
	}
	if !strings.Contains(rep.textCalls[0], "/llm edit") {
		t.Errorf("reply must mention /llm edit, got %q", rep.textCalls[0])
	}
}

func TestGenerateHandler_RequireLLM_HasOverride_ProceedsToBackend(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{Mode: "placeholder", DOCX: []byte("X")}}
	llm := newFakeLLMService()
	cfg, _ := domain.NewLLMSettings("openai", "", "sk-test1234", "gpt-4o")
	llm.data[42] = cfg
	h := telegram.NewGenerateHandler(api, rep,
		telegram.WithGenerateLLMService(llm),
		telegram.WithGenerateRequirePerChatLLM(true),
	)
	doc := &telegram.Document{Filename: "t.docx", Data: []byte("DOCX")}

	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.calls != 1 {
		t.Errorf("backend GenerateProposal called %d times, want 1", api.calls)
	}
}

func TestGenerateHandler_RequireLLM_Disabled_LegacyFallbackWorks(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{Mode: "placeholder", DOCX: []byte("X")}}
	llm := newFakeLLMService()
	h := telegram.NewGenerateHandler(api, rep,
		telegram.WithGenerateLLMService(llm),
		telegram.WithGenerateRequirePerChatLLM(false),
	)
	doc := &telegram.Document{Filename: "t.docx", Data: []byte("DOCX")}

	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if api.calls != 1 {
		t.Errorf("backend GenerateProposal called %d times, want 1 (legacy mode)", api.calls)
	}
}

func TestGenerateHandler_LLMService_ChatHasSettings_PopulatesOverride(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{Mode: "placeholder"}}
	llm := newFakeLLMService()
	cfg, _ := domain.NewLLMSettings("anthropic", "", "sk-ant-test", "claude-3-5-sonnet")
	llm.data[42] = cfg
	h := telegram.NewGenerateHandler(api, rep, telegram.WithGenerateLLMService(llm))

	doc := &telegram.Document{Filename: "t.docx", Data: []byte("DOCX")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 42, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	want := usecase.LLMOverride{
		Provider: "anthropic",
		BaseURL:  "",
		APIKey:   "sk-ant-test",
		Model:    "claude-3-5-sonnet",
	}
	if api.gotReq.LLM != want {
		t.Errorf("LLM = %+v, want %+v", api.gotReq.LLM, want)
	}
}
