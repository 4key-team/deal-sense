package telegram_test

import (
	"context"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain"
	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// --- ShouldRoutePending ---------------------------------------------------

func TestShouldRoutePending(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(7, telegram.PendingGenerate)

	tests := []struct {
		name   string
		text   string
		hasDoc bool
		chatID int64
		want   bool
	}{
		{"document during pending", "", true, 7, true},
		{"document on absent chat", "", true, 42, false},
		{"text during pending", "шаблона может не быть", false, 7, true},
		{"text on absent chat", "any", false, 42, false},
		{"empty text during pending", "", false, 7, false},
		{"empty text on absent chat", "", false, 42, false},
		{"/go during pending", "/go", false, 7, true},
		{"/cancel during pending", "/cancel", false, 7, true},
		{"other slash command during pending", "/profile", false, 7, false},
		{"/analyze during pending", "/analyze", false, 7, false},
		{"/help during pending", "/help", false, 7, false},
		{"document with caption", "caption", true, 7, true},
		{"text with leading whitespace", "  abc", false, 7, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := telegram.ShouldRoutePending(tt.text, tt.hasDoc, tt.chatID, sessions)
			if got != tt.want {
				t.Errorf("ShouldRoutePending(%q, hasDoc=%v, chat=%d) = %v, want %v",
					tt.text, tt.hasDoc, tt.chatID, got, tt.want)
			}
		})
	}
}

// --- RoutePending ---------------------------------------------------------

func newGenerateRoutingDeps(t *testing.T) (*telegram.GenerateHandler, *generateFakeAPI, *recordingReplier) {
	t.Helper()
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{
		Mode:     "placeholder",
		Sections: []usecase.GeneratedSection{{Title: "S1"}},
		DOCX:     []byte("DOCX BYTES"),
	}}
	llm := newFakeLLMService()
	cfg, _ := domain.NewLLMSettings("openai", "", "sk-test", "gpt-4o")
	llm.data[100] = cfg
	llm.data[200] = cfg
	h := telegram.NewGenerateHandler(api, rep, telegram.WithGenerateLLMService(llm))
	return h, api, rep
}

func newAnalyzeRoutingDeps(t *testing.T) (*telegram.AnalyzeHandler, *fakeAPI, *fakeReplier) {
	t.Helper()
	rep := &fakeReplier{}
	api := &fakeAPI{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW"}}
	llm := newFakeLLMService()
	cfg, _ := domain.NewLLMSettings("openai", "", "sk-test", "gpt-4o")
	llm.data[100] = cfg
	h := telegram.NewAnalyzeHandler(api, newFakeProfileStore(), rep, "Fallback Co",
		telegram.WithAnalyzeLLMService(llm))
	return h, api, rep
}

func TestRoutePending_NoSession_NotHandled(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	gh, _, rep := newGenerateRoutingDeps(t)
	ah, _, _ := newAnalyzeRoutingDeps(t)

	handled, err := telegram.RoutePending(
		context.Background(),
		&telegram.Update{ChatID: 999, Text: "anything"},
		sessions, ah, gh, rep,
	)
	if err != nil {
		t.Fatalf("RoutePending: %v", err)
	}
	if handled {
		t.Error("RoutePending without session must return handled=false")
	}
}

func TestRoutePending_Document_AppendsAndReplies(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(100, telegram.PendingGenerate)
	gh, api, rep := newGenerateRoutingDeps(t)
	ah, _, _ := newAnalyzeRoutingDeps(t)

	handled, err := telegram.RoutePending(
		context.Background(),
		&telegram.Update{ChatID: 100, Document: &telegram.Document{
			Filename: "template.docx", Data: []byte("T"),
		}},
		sessions, ah, gh, rep,
	)
	if err != nil {
		t.Fatalf("RoutePending: %v", err)
	}
	if !handled {
		t.Fatal("document during pending must be handled")
	}
	files := sessions.Files(100)
	if len(files) != 1 {
		t.Errorf("expected 1 collected file, got %d", len(files))
	}
	if files[0].Filename != "template.docx" {
		t.Errorf("collected filename = %q, want template.docx", files[0].Filename)
	}
	if api.calls != 0 {
		t.Errorf("backend GenerateProposal must NOT be called on file collect, got %d calls", api.calls)
	}
	if len(rep.textCalls) != 1 {
		t.Fatalf("expected one reply, got %d", len(rep.textCalls))
	}
	if !strings.Contains(rep.textCalls[0], "template.docx") {
		t.Errorf("reply must mention filename, got %q", rep.textCalls[0])
	}
	if !strings.Contains(rep.textCalls[0], "/go") {
		t.Errorf("reply must mention /go, got %q", rep.textCalls[0])
	}
}

func TestRoutePending_MultipleDocuments_AllAccumulated(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(100, telegram.PendingGenerate)
	gh, _, rep := newGenerateRoutingDeps(t)
	ah, _, _ := newAnalyzeRoutingDeps(t)

	for i, name := range []string{"template.docx", "brief.zip", "prices.md"} {
		_, err := telegram.RoutePending(
			context.Background(),
			&telegram.Update{ChatID: 100, Document: &telegram.Document{
				Filename: name, Data: []byte{byte(i)},
			}},
			sessions, ah, gh, rep,
		)
		if err != nil {
			t.Fatalf("RoutePending #%d: %v", i, err)
		}
	}
	files := sessions.Files(100)
	if len(files) != 3 {
		t.Fatalf("expected 3 collected files, got %d", len(files))
	}
	if files[0].Filename != "template.docx" {
		t.Errorf("Files[0] = %q, want template.docx", files[0].Filename)
	}
	if files[2].Filename != "prices.md" {
		t.Errorf("Files[2] = %q, want prices.md", files[2].Filename)
	}
}

func TestRoutePending_FreeText_RepliesHint(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(100, telegram.PendingGenerate)
	gh, api, rep := newGenerateRoutingDeps(t)
	ah, _, _ := newAnalyzeRoutingDeps(t)

	handled, err := telegram.RoutePending(
		context.Background(),
		&telegram.Update{ChatID: 100, Text: "шаблона может не быть"},
		sessions, ah, gh, rep,
	)
	if err != nil {
		t.Fatalf("RoutePending: %v", err)
	}
	if !handled {
		t.Fatal("free text during pending must be handled")
	}
	if api.calls != 0 {
		t.Errorf("backend must not be called, got %d calls", api.calls)
	}
	if len(rep.textCalls) != 1 || !strings.Contains(rep.textCalls[0], "Жду файл") {
		t.Errorf("expected hint reply, got %v", rep.textCalls)
	}
}

func TestRoutePending_Cancel_ClearsSessionAndReplies(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(100, telegram.PendingGenerate)
	sessions.AppendFile(100, telegram.CollectedFile{Filename: "x.docx", Data: []byte("X")})
	gh, _, rep := newGenerateRoutingDeps(t)
	ah, _, _ := newAnalyzeRoutingDeps(t)

	handled, err := telegram.RoutePending(
		context.Background(),
		&telegram.Update{ChatID: 100, Text: "/cancel"},
		sessions, ah, gh, rep,
	)
	if err != nil {
		t.Fatalf("RoutePending: %v", err)
	}
	if !handled {
		t.Fatal("/cancel during pending must be handled")
	}
	if _, ok := sessions.Get(100); ok {
		t.Error("session must be cleared after /cancel")
	}
	if len(rep.textCalls) != 1 || !strings.Contains(strings.ToLower(rep.textCalls[0]), "отмен") {
		t.Errorf("expected cancellation reply, got %v", rep.textCalls)
	}
}

func TestRoutePending_Go_NoFiles_RepliesWarning(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(100, telegram.PendingGenerate)
	gh, api, rep := newGenerateRoutingDeps(t)
	ah, _, _ := newAnalyzeRoutingDeps(t)

	handled, err := telegram.RoutePending(
		context.Background(),
		&telegram.Update{ChatID: 100, Text: "/go"},
		sessions, ah, gh, rep,
	)
	if err != nil {
		t.Fatalf("RoutePending: %v", err)
	}
	if !handled {
		t.Fatal("/go during pending must be handled")
	}
	if api.calls != 0 {
		t.Errorf("backend must not be called without files, got %d calls", api.calls)
	}
	if _, ok := sessions.Get(100); !ok {
		t.Error("session should remain so the user can still attach files")
	}
	if len(rep.textCalls) != 1 || !strings.Contains(strings.ToLower(rep.textCalls[0]), "нет файлов") {
		t.Errorf("expected 'no files' reply, got %v", rep.textCalls)
	}
}

func TestRoutePending_Go_Generate_DispatchesWithTemplateAndContext(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(100, telegram.PendingGenerate)
	sessions.AppendFile(100, telegram.CollectedFile{Filename: "template.docx", Data: []byte("TPL")})
	sessions.AppendFile(100, telegram.CollectedFile{Filename: "brief.md", Data: []byte("BRIEF")})
	sessions.AppendFile(100, telegram.CollectedFile{Filename: "prices.zip", Data: []byte("PRICES")})
	gh, api, rep := newGenerateRoutingDeps(t)
	ah, _, _ := newAnalyzeRoutingDeps(t)

	handled, err := telegram.RoutePending(
		context.Background(),
		&telegram.Update{ChatID: 100, Text: "/go"},
		sessions, ah, gh, rep,
	)
	if err != nil {
		t.Fatalf("RoutePending: %v", err)
	}
	if !handled {
		t.Fatal("/go must be handled")
	}
	if _, ok := sessions.Get(100); ok {
		t.Error("session must be cleared after successful /go")
	}
	if api.calls != 1 {
		t.Fatalf("backend GenerateProposal calls = %d, want 1", api.calls)
	}
	if api.gotReq.TemplateFilename != "template.docx" {
		t.Errorf("template = %q, want template.docx", api.gotReq.TemplateFilename)
	}
	if string(api.gotReq.Template) != "TPL" {
		t.Errorf("template body = %q, want TPL", string(api.gotReq.Template))
	}
	if len(api.gotReq.ContextFiles) != 2 {
		t.Fatalf("context files = %d, want 2", len(api.gotReq.ContextFiles))
	}
	if api.gotReq.ContextFiles[0].Filename != "brief.md" {
		t.Errorf("ContextFiles[0] = %q, want brief.md", api.gotReq.ContextFiles[0].Filename)
	}
	if api.gotReq.ContextFiles[1].Filename != "prices.zip" {
		t.Errorf("ContextFiles[1] = %q, want prices.zip", api.gotReq.ContextFiles[1].Filename)
	}
	if len(rep.docCalls) != 1 {
		t.Fatalf("expected one document reply, got %d", len(rep.docCalls))
	}
}

func TestRoutePending_Go_Analyze_UsesFirstFile(t *testing.T) {
	sessions := telegram.NewInMemoryPendingCommandSessions()
	sessions.Set(100, telegram.PendingAnalyze)
	sessions.AppendFile(100, telegram.CollectedFile{Filename: "tender.pdf", Data: []byte("TENDER")})
	sessions.AppendFile(100, telegram.CollectedFile{Filename: "extra.md", Data: []byte("EXTRA")})
	gh, _, _ := newGenerateRoutingDeps(t)
	ah, analyzeAPI, rep := newAnalyzeRoutingDeps(t)
	_ = rep

	handled, err := telegram.RoutePending(
		context.Background(),
		&telegram.Update{ChatID: 100, Text: "/go"},
		sessions, ah, gh, rep,
	)
	if err != nil {
		t.Fatalf("RoutePending: %v", err)
	}
	if !handled {
		t.Fatal("/go for analyze must be handled")
	}
	if analyzeAPI.calls != 1 {
		t.Fatalf("backend AnalyzeTender calls = %d, want 1", analyzeAPI.calls)
	}
	if analyzeAPI.gotReq.Filename != "tender.pdf" {
		t.Errorf("Analyze used filename = %q, want tender.pdf (first file)", analyzeAPI.gotReq.Filename)
	}
	if string(analyzeAPI.gotReq.File) != "TENDER" {
		t.Errorf("Analyze body = %q, want TENDER", string(analyzeAPI.gotReq.File))
	}
}

func TestGenerateHandler_NoContext_AppendsCaptionWarning(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{
		Mode: "generative", Sections: []usecase.GeneratedSection{{Title: "S"}},
		DOCX: []byte("X"),
	}}
	h := telegram.NewGenerateHandler(api, rep)

	doc := &telegram.Document{Filename: "t.docx", Data: []byte("TPL")}
	if err := h.Handle(context.Background(), &telegram.Update{ChatID: 1, Document: doc}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(rep.docCalls) != 1 {
		t.Fatalf("doc replies = %d", len(rep.docCalls))
	}
	if !strings.Contains(rep.docCalls[0].caption, "Generic") {
		t.Errorf("caption missing no-context warning, got %q", rep.docCalls[0].caption)
	}
}

func TestGenerateHandler_WithContext_NoCaptionWarning(t *testing.T) {
	rep := &recordingReplier{}
	api := &generateFakeAPI{resp: &usecase.GenerateProposalResponse{
		Mode: "generative", Sections: []usecase.GeneratedSection{{Title: "S"}},
		DOCX: []byte("X"),
	}}
	h := telegram.NewGenerateHandler(api, rep)

	template := telegram.CollectedFile{Filename: "t.docx", Data: []byte("TPL")}
	ctxFiles := []usecase.ContextFile{
		{Filename: "brief.md", Data: []byte("BRIEF")},
	}
	if err := h.HandleCollected(context.Background(), 1, template, ctxFiles); err != nil {
		t.Fatalf("HandleCollected: %v", err)
	}
	if len(rep.docCalls) != 1 {
		t.Fatalf("doc replies = %d", len(rep.docCalls))
	}
	if strings.Contains(rep.docCalls[0].caption, "Generic") {
		t.Errorf("caption should NOT include warning when context is present, got %q", rep.docCalls[0].caption)
	}
}
