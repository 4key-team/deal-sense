package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
	telegramadapter "github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// telegramAPIStub answers the handful of Bot API methods the runtime hits
// during a quick start-stop cycle. It signals readiness via getMeCh so
// tests can replace sleep-based waits with deterministic synchronization.
func telegramAPIStub(t *testing.T) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	getMeCh := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			select {
			case getMeCh <- struct{}{}:
			default:
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"stub","username":"stub_bot"}}`))
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	return srv, getMeCh
}

func TestRun_StartsAndStops(t *testing.T) {
	stub, ready := telegramAPIStub(t)
	defer stub.Close()

	cfg := telegramadapter.Config{
		BotToken:         "test-token",
		AllowlistUserIDs: []int64{42},
		APIBaseURL:       "http://example.invalid",
		LogLevel:         "info",
		ProfileStorePath: filepath.Join(t.TempDir(), "profiles.json"),
	}

	ctx, cancel := context.WithCancel(t.Context())

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, discardLogger(), cfg, []bot.Option{
			bot.WithServerURL(stub.URL),
		})
	}()

	// Wait for the bot to call getMe — proof that bot.New + Start are progressing.
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("bot did not reach getMe within 3s")
	}
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("run returned err: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("run did not return after context cancellation")
	}
}

func TestRun_EmptyAllowlistOpensInOpenMode(t *testing.T) {
	// Empty AllowlistUserIDs no longer fails — auth.ParseAllowlist returns
	// an open allowlist (bootstrap/dev). The run() helper must therefore
	// progress past the allowlist construction step. We exercise that by
	// pointing at an unreachable Telegram API so run() fails on bot.New
	// instead of on the allowlist — proving the allowlist step succeeded.
	cfg := telegramadapter.Config{
		BotToken:         "test-token",
		ProfileStorePath: filepath.Join(t.TempDir(), "profiles.json"),
		// no AllowlistUserIDs — must NOT fail; ParseAllowlist → open mode
	}
	err := run(t.Context(), discardLogger(), cfg, []bot.Option{
		bot.WithServerURL("http://127.0.0.1:1"),
		bot.WithCheckInitTimeout(200 * time.Millisecond),
	})
	if err == nil {
		t.Fatal("expected bot.New error (unreachable stub), got nil")
	}
	if errors.Is(err, auth.ErrEmptyAllowlist) {
		t.Errorf("err = %v, must NOT wrap ErrEmptyAllowlist (empty → open mode)", err)
	}
}

func TestRun_BotInitFails(t *testing.T) {
	cfg := telegramadapter.Config{
		BotToken:         "test-token",
		AllowlistUserIDs: []int64{1},
		ProfileStorePath: filepath.Join(t.TempDir(), "profiles.json"),
	}
	err := run(t.Context(), discardLogger(), cfg, []bot.Option{
		// Unreachable Telegram API → bot.New fails on getMe call.
		bot.WithServerURL("http://127.0.0.1:1"),
		bot.WithCheckInitTimeout(200 * time.Millisecond),
	})
	if err == nil {
		t.Fatal("expected init error, got nil")
	}
	if !strings.Contains(err.Error(), "bot.New") {
		t.Errorf("err = %v, want to wrap 'bot.New'", err)
	}
}

func TestRun_DenyMiddlewareSendsNotice(t *testing.T) {
	// Drive run() with a stub Telegram API that delivers one update from a
	// non-allowlisted user. The bot must respond with the denial message,
	// proving the closure-based middleware reaches the bot.
	var deniedSends atomic.Int32
	updateDelivered := make(chan struct{}, 1)
	getMeReady := make(chan struct{}, 1)

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			select {
			case getMeReady <- struct{}{}:
			default:
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			select {
			case <-updateDelivered:
				_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
			default:
				updateDelivered <- struct{}{}
				_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":1,"message":{"message_id":1,"date":0,"text":"/analyze","chat":{"id":5,"type":"private"},"from":{"id":999,"is_bot":false,"first_name":"intruder"}}}]}`))
			}
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			deniedSends.Add(1)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":5,"type":"private"}}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	defer stub.Close()

	cfg := telegramadapter.Config{
		BotToken:         "t",
		AllowlistUserIDs: []int64{42},
		ProfileStorePath: filepath.Join(t.TempDir(), "profiles.json"),
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, discardLogger(), cfg, []bot.Option{bot.WithServerURL(stub.URL)})
	}()

	select {
	case <-getMeReady:
	case <-time.After(3 * time.Second):
		t.Fatal("bot did not start within 3s")
	}

	// Poll for the denial sendMessage to arrive.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if deniedSends.Load() >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if deniedSends.Load() < 1 {
		t.Error("expected at least one denial sendMessage")
	}
	cancel()
	<-errCh
}

func TestDocumentFromMessage(t *testing.T) {
	directDoc := &models.Document{FileID: "direct"}
	replyDoc := &models.Document{FileID: "reply"}

	tests := []struct {
		name    string
		msg     *models.Message
		wantNil bool
		wantID  string
	}{
		{
			name:   "direct document attached",
			msg:    &models.Message{Document: directDoc},
			wantID: "direct",
		},
		{
			name: "document on reply target",
			msg: &models.Message{
				ReplyToMessage: &models.Message{Document: replyDoc},
			},
			wantID: "reply",
		},
		{
			name:    "no document anywhere",
			msg:     &models.Message{},
			wantNil: true,
		},
		{
			name: "reply target without document",
			msg: &models.Message{
				ReplyToMessage: &models.Message{},
			},
			wantNil: true,
		},
		{
			name: "direct wins over reply",
			msg: &models.Message{
				Document:       directDoc,
				ReplyToMessage: &models.Message{Document: replyDoc},
			},
			wantID: "direct",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := documentFromMessage(tt.msg)
			if tt.wantNil {
				if got != nil {
					t.Errorf("got %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got nil document, expected one")
			}
			if got.FileID != tt.wantID {
				t.Errorf("FileID = %q, want %q", got.FileID, tt.wantID)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"GARBAGE", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseLogLevel(tt.in); got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// --- handler unit tests --------------------------------------------------

// stubAPIClient records the request it receives.
type stubAPIClient struct {
	gotReq usecase.AnalyzeTenderRequest
	resp   *usecase.AnalyzeTenderResponse
	err    error
	calls  int
}

func (s *stubAPIClient) AnalyzeTender(ctx context.Context, req usecase.AnalyzeTenderRequest) (*usecase.AnalyzeTenderResponse, error) {
	s.calls++
	s.gotReq = req
	return s.resp, s.err
}

// GenerateProposal stub — not exercised by /analyze tests.
func (s *stubAPIClient) GenerateProposal(context.Context, usecase.GenerateProposalRequest) (*usecase.GenerateProposalResponse, error) {
	return nil, nil
}

func stubBotForSend(t *testing.T) (*bot.Bot, *atomic.Int32) {
	t.Helper()
	var sends atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			sends.Add(1)
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`))
		}
	}))
	t.Cleanup(srv.Close)
	b, err := bot.New("t", bot.WithServerURL(srv.URL))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	return b, &sends
}

// failingBot constructs a bot whose SendMessage always errors via ok:false.
func failingBot(t *testing.T) *bot.Bot {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"bad"}`))
	}))
	t.Cleanup(srv.Close)
	b, err := bot.New("t", bot.WithServerURL(srv.URL))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	return b
}

func TestStartHandler_RepliesWithWelcome(t *testing.T) {
	b, sends := stubBotForSend(t)
	h := startHandler(discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/start"},
	})
	if sends.Load() != 1 {
		t.Errorf("sendMessage calls = %d, want 1", sends.Load())
	}
}

func TestStartHandler_AttachesReplyKeyboard(t *testing.T) {
	// Capture the outgoing SendMessage body to assert ReplyMarkup carries
	// the main keyboard. The stub records the raw JSON payload Telegram
	// receives so we can decode and inspect it.
	var lastBody []byte
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			b, _ := io.ReadAll(r.Body)
			lastBody = b
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer stub.Close()

	b, err := bot.New("t", bot.WithServerURL(stub.URL))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}

	h := startHandler(discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/start"},
	})

	if !strings.Contains(string(lastBody), "reply_markup") {
		t.Errorf("/start sendMessage missing reply_markup; body=%s", lastBody)
	}
	// Each button label must appear so the keyboard is wired end-to-end.
	for _, label := range []string{telegramadapter.ButtonAnalyze, telegramadapter.ButtonGenerate, telegramadapter.ButtonProfile, telegramadapter.ButtonHelp} {
		if !strings.Contains(string(lastBody), label) {
			t.Errorf("/start keyboard missing label %q; body=%s", label, lastBody)
		}
	}
}

func TestStartHandler_IgnoresUpdateWithoutMessage(t *testing.T) {
	b, sends := stubBotForSend(t)
	h := startHandler(discardLogger())
	h(context.Background(), b, &models.Update{})
	if sends.Load() != 0 {
		t.Errorf("sendMessage calls = %d, want 0", sends.Load())
	}
}

func TestHelpHandler_RepliesWithCommandList(t *testing.T) {
	b, sends := stubBotForSend(t)
	h := helpHandler(discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/help"},
	})
	if sends.Load() != 1 {
		t.Errorf("sendMessage calls = %d, want 1", sends.Load())
	}
}

func TestHelpHandler_IgnoresUpdateWithoutMessage(t *testing.T) {
	b, sends := stubBotForSend(t)
	h := helpHandler(discardLogger())
	h(context.Background(), b, &models.Update{})
	if sends.Load() != 0 {
		t.Errorf("sendMessage calls = %d, want 0", sends.Load())
	}
}

func TestDefaultHandler_RepliesToTextMessage(t *testing.T) {
	b, sends := stubBotForSend(t)
	h := defaultHandler(discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "hi"},
	})
	if sends.Load() != 1 {
		t.Errorf("sendMessage calls = %d, want 1", sends.Load())
	}
}

func TestDefaultHandler_IgnoresUpdateWithoutMessage(t *testing.T) {
	b, sends := stubBotForSend(t)
	h := defaultHandler(discardLogger())
	h(context.Background(), b, &models.Update{})
	if sends.Load() != 0 {
		t.Errorf("sendMessage calls = %d, want 0", sends.Load())
	}
}

// nopDownloader is a no-op docDownloader used where the test message has
// no document.
var nopDownloader docDownloader = func(context.Context, *bot.Bot, *models.Document) ([]byte, string, error) {
	return nil, "", errors.New("nopDownloader should not be called")
}

func TestMakeAnalyzeHandler_NoDocument_RepliesAttachPrompt(t *testing.T) {
	b, sends := stubBotForSend(t)
	api := &stubAPIClient{}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "profile")

	h := makeAnalyzeHandler(ah, b, nopDownloader, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/analyze"},
	})
	if api.calls != 0 {
		t.Errorf("api.AnalyzeTender called %d times, want 0", api.calls)
	}
	if sends.Load() != 1 {
		t.Errorf("sendMessage calls = %d, want 1 (attach prompt)", sends.Load())
	}
}

func TestMakeAnalyzeHandler_IgnoresUpdateWithoutMessage(t *testing.T) {
	b, sends := stubBotForSend(t)
	api := &stubAPIClient{}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "")
	h := makeAnalyzeHandler(ah, b, nopDownloader, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{})
	if api.calls != 0 || sends.Load() != 0 {
		t.Errorf("expected no work for update without message; api=%d sends=%d", api.calls, sends.Load())
	}
}

func TestMakeAnalyzeHandler_WithDocument_DownloadsAndCallsAPI(t *testing.T) {
	b, _ := stubBotForSend(t)
	api := &stubAPIClient{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW", Score: 0.1}}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "profile")

	dl := docDownloader(func(_ context.Context, _ *bot.Bot, doc *models.Document) ([]byte, string, error) {
		return []byte("PDF"), doc.FileName, nil
	})

	h := makeAnalyzeHandler(ah, b, dl, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{
			Chat:     models.Chat{ID: 5},
			Text:     "/analyze",
			Document: &models.Document{FileID: "f1", FileName: "tender.pdf"},
			From:     &models.User{ID: 7},
		},
	})
	if api.calls != 1 {
		t.Errorf("api.AnalyzeTender called %d times, want 1", api.calls)
	}
	if string(api.gotReq.File) != "PDF" {
		t.Errorf("file body = %q, want PDF", string(api.gotReq.File))
	}
	if api.gotReq.Filename != "tender.pdf" {
		t.Errorf("filename = %q, want tender.pdf", api.gotReq.Filename)
	}
}

func TestMakeAnalyzeHandler_LogsHandleError(t *testing.T) {
	// Replier returns error → AnalyzeHandler.Handle returns it → handler logs.
	b := failingBot(t)
	api := &stubAPIClient{}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "")
	h := makeAnalyzeHandler(ah, b, nopDownloader, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	// No-document message → handler asks user to attach; failing bot makes
	// the Reply error which the handler then logs.
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/analyze"},
	})
}

// --- makeGenerateHandler tests ------------------------------------------

func TestMakeGenerateHandler_IgnoresUpdateWithoutMessage(t *testing.T) {
	b, sends := stubBotForSend(t)
	api := &stubAPIClient{}
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})
	h := makeGenerateHandler(gh, b, nopDownloader, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{})
	if api.calls != 0 || sends.Load() != 0 {
		t.Errorf("expected no work for update without message; api=%d sends=%d", api.calls, sends.Load())
	}
}

func TestMakeGenerateHandler_NoDocument_AsksForTemplate(t *testing.T) {
	b, sends := stubBotForSend(t)
	api := &stubAPIClient{}
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})

	h := makeGenerateHandler(gh, b, nopDownloader, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/generate"},
	})
	if sends.Load() != 1 {
		t.Errorf("sendMessage calls = %d, want 1 (template prompt)", sends.Load())
	}
}

// stubGenerateAPI satisfies APIClient with a configurable GenerateProposal.
type stubGenerateAPI struct {
	gotReq usecase.GenerateProposalRequest
	resp   *usecase.GenerateProposalResponse
	err    error
	calls  int
}

func (s *stubGenerateAPI) AnalyzeTender(context.Context, usecase.AnalyzeTenderRequest) (*usecase.AnalyzeTenderResponse, error) {
	return nil, errors.New("not used")
}
func (s *stubGenerateAPI) GenerateProposal(_ context.Context, req usecase.GenerateProposalRequest) (*usecase.GenerateProposalResponse, error) {
	s.calls++
	s.gotReq = req
	return s.resp, s.err
}

func TestMakeGenerateHandler_WithTemplate_DownloadsAndCallsAPI(t *testing.T) {
	// stubBotForSend used for sendDocument tracking too — both call /sendMessage
	// in the JSON RPC layer but SendDocument hits /sendDocument. We accept any
	// 2xx and just verify API was called with right body.
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`))
		}
	}))
	defer stub.Close()
	b, err := bot.New("t", bot.WithServerURL(stub.URL))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}

	api := &stubGenerateAPI{resp: &usecase.GenerateProposalResponse{Mode: "placeholder", DOCX: []byte("DOCXBYTES")}}
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})

	dl := docDownloader(func(_ context.Context, _ *bot.Bot, doc *models.Document) ([]byte, string, error) {
		return []byte("TEMPLATE"), doc.FileName, nil
	})

	h := makeGenerateHandler(gh, b, dl, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{
			Chat:     models.Chat{ID: 5},
			Text:     "/generate",
			Document: &models.Document{FileID: "f1", FileName: "tpl.docx"},
		},
	})
	if api.calls != 1 {
		t.Errorf("api.GenerateProposal calls = %d, want 1", api.calls)
	}
	if api.gotReq.TemplateFilename != "tpl.docx" {
		t.Errorf("filename = %q", api.gotReq.TemplateFilename)
	}
	if string(api.gotReq.Template) != "TEMPLATE" {
		t.Errorf("template = %q", string(api.gotReq.Template))
	}
}

func TestMakeGenerateHandler_DownloadError_RepliesToUser(t *testing.T) {
	b, sends := stubBotForSend(t)
	api := &stubGenerateAPI{}
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})

	failingDL := docDownloader(func(context.Context, *bot.Bot, *models.Document) ([]byte, string, error) {
		return nil, "", errors.New("boom")
	})

	h := makeGenerateHandler(gh, b, failingDL, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{
			Chat:     models.Chat{ID: 5},
			Document: &models.Document{FileID: "f1", FileName: "x.docx"},
		},
	})
	if api.calls != 0 {
		t.Errorf("api.GenerateProposal called despite download error")
	}
	if sends.Load() != 1 {
		t.Errorf("expected one user-facing error reply, sends=%d", sends.Load())
	}
}

func TestMakeAnalyzeHandler_DownloadError_RepliesToUser(t *testing.T) {
	b, sends := stubBotForSend(t)
	api := &stubAPIClient{}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "")

	failingDL := docDownloader(func(context.Context, *bot.Bot, *models.Document) ([]byte, string, error) {
		return nil, "", errors.New("boom")
	})

	h := makeAnalyzeHandler(ah, b, failingDL, telegramadapter.NewInMemoryPendingCommandSessions(), discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{
			Chat:     models.Chat{ID: 5},
			Document: &models.Document{FileID: "f1", FileName: "x.pdf"},
		},
	})
	if api.calls != 0 {
		t.Errorf("api.AnalyzeTender called despite download error")
	}
	if sends.Load() != 1 {
		t.Errorf("expected one user-facing error reply, sends=%d", sends.Load())
	}
}

// --- pending command (Bug-B: two-step file workflow) --------------------

func TestMakeAnalyzeHandler_NoDocument_SetsPendingAnalyze(t *testing.T) {
	b, _ := stubBotForSend(t)
	api := &stubAPIClient{}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "")
	pending := telegramadapter.NewInMemoryPendingCommandSessions()

	h := makeAnalyzeHandler(ah, b, nopDownloader, pending, discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/analyze"},
	})

	kind, ok := pending.Get(5)
	if !ok {
		t.Fatal("pending must be set after /analyze without document")
	}
	if kind != telegramadapter.PendingAnalyze {
		t.Errorf("pending kind = %q, want analyze", kind)
	}
}

func TestMakeAnalyzeHandler_WithDocument_ClearsAnyPending(t *testing.T) {
	b, _ := stubBotForSend(t)
	api := &stubAPIClient{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW", Score: 0.1}}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "")
	pending := telegramadapter.NewInMemoryPendingCommandSessions()
	pending.Set(5, telegramadapter.PendingGenerate) // stale prior state

	dl := docDownloader(func(_ context.Context, _ *bot.Bot, doc *models.Document) ([]byte, string, error) {
		return []byte("PDF"), doc.FileName, nil
	})
	h := makeAnalyzeHandler(ah, b, dl, pending, discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{
			Chat:     models.Chat{ID: 5},
			Text:     "/analyze",
			Document: &models.Document{FileID: "f", FileName: "tender.pdf"},
		},
	})

	if _, ok := pending.Get(5); ok {
		t.Error("pending must be cleared after successful processing")
	}
}

func TestMakeGenerateHandler_NoDocument_SetsPendingGenerate(t *testing.T) {
	b, _ := stubBotForSend(t)
	api := &stubAPIClient{}
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})
	pending := telegramadapter.NewInMemoryPendingCommandSessions()

	h := makeGenerateHandler(gh, b, nopDownloader, pending, discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "/generate"},
	})

	kind, ok := pending.Get(5)
	if !ok {
		t.Fatal("pending must be set after /generate without document")
	}
	if kind != telegramadapter.PendingGenerate {
		t.Errorf("pending kind = %q, want generate", kind)
	}
}

func TestPendingDocumentMatcher_Cases(t *testing.T) {
	pending := telegramadapter.NewInMemoryPendingCommandSessions()
	pending.Set(5, telegramadapter.PendingAnalyze)

	match := pendingDocumentMatcher(pending)
	cases := []struct {
		name   string
		update *models.Update
		want   bool
	}{
		{"nil update", nil, false},
		{"no message", &models.Update{}, false},
		{"text only, pending exists", &models.Update{Message: &models.Message{Chat: models.Chat{ID: 5}, Text: "hi"}}, false},
		{"document, no pending for this chat", &models.Update{Message: &models.Message{Chat: models.Chat{ID: 99}, Document: &models.Document{FileID: "f"}}}, false},
		{"document with pending for chat", &models.Update{Message: &models.Message{Chat: models.Chat{ID: 5}, Document: &models.Document{FileID: "f"}}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := match(tc.update); got != tc.want {
				t.Errorf("matcher = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMakePendingDispatcher_PendingAnalyze_DispatchesAndClears(t *testing.T) {
	b, _ := stubBotForSend(t)
	api := &stubAPIClient{resp: &usecase.AnalyzeTenderResponse{Verdict: "LOW", Score: 0.1}}
	ah := telegramadapter.NewAnalyzeHandler(api, nil, &botReplier{b: b, logger: discardLogger()}, "")
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})
	pending := telegramadapter.NewInMemoryPendingCommandSessions()
	pending.Set(5, telegramadapter.PendingAnalyze)

	dl := docDownloader(func(_ context.Context, _ *bot.Bot, doc *models.Document) ([]byte, string, error) {
		return []byte("PDF"), doc.FileName, nil
	})
	h := makePendingDispatcher(ah, gh, pending, b, dl, discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{
			Chat:     models.Chat{ID: 5},
			Document: &models.Document{FileID: "f", FileName: "tender.pdf"},
		},
	})

	if api.calls != 1 {
		t.Errorf("api.AnalyzeTender called %d times, want 1", api.calls)
	}
	if _, ok := pending.Get(5); ok {
		t.Error("dispatcher must clear pending after dispatch")
	}
}

func TestMakePendingDispatcher_PendingGenerate_DispatchesToGenerate(t *testing.T) {
	b, _ := stubBotForSend(t)
	api := &stubGenerateAPI{resp: &usecase.GenerateProposalResponse{Mode: "placeholder", DOCX: []byte("DOCX")}}
	ah := telegramadapter.NewAnalyzeHandler(nil, nil, &botReplier{b: b, logger: discardLogger()}, "")
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})
	pending := telegramadapter.NewInMemoryPendingCommandSessions()
	pending.Set(5, telegramadapter.PendingGenerate)

	dl := docDownloader(func(_ context.Context, _ *bot.Bot, doc *models.Document) ([]byte, string, error) {
		return []byte("TPL"), doc.FileName, nil
	})
	h := makePendingDispatcher(ah, gh, pending, b, dl, discardLogger())
	h(context.Background(), b, &models.Update{
		Message: &models.Message{
			Chat:     models.Chat{ID: 5},
			Document: &models.Document{FileID: "f", FileName: "tpl.docx"},
		},
	})

	if api.calls != 1 {
		t.Errorf("api.GenerateProposal called %d times, want 1", api.calls)
	}
	if _, ok := pending.Get(5); ok {
		t.Error("dispatcher must clear pending after dispatch")
	}
}

// --- defaultDocDownloader tests -----------------------------------------

func TestDefaultDocDownloader_Success(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
		case strings.HasSuffix(r.URL.Path, "/getFile"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"file_id":"f1","file_path":"documents/file_1.pdf"}}`))
		case strings.Contains(r.URL.Path, "/file/bot"):
			_, _ = w.Write([]byte("PDF DATA"))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer stub.Close()

	b, err := bot.New("token", bot.WithServerURL(stub.URL))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}

	data, filename, err := defaultDocDownloader(context.Background(), b, &models.Document{
		FileID:   "f1",
		FileName: "tender.pdf",
	})
	if err != nil {
		t.Fatalf("defaultDocDownloader: %v", err)
	}
	if string(data) != "PDF DATA" {
		t.Errorf("data = %q, want PDF DATA", string(data))
	}
	if filename != "tender.pdf" {
		t.Errorf("filename = %q", filename)
	}
}

func TestDefaultDocDownloader_GetFileError(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"file gone"}`))
		}
	}))
	defer stub.Close()

	b, err := bot.New("token", bot.WithServerURL(stub.URL))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	_, _, err = defaultDocDownloader(context.Background(), b, &models.Document{FileID: "missing"})
	if err == nil {
		t.Fatal("expected getFile error")
	}
}

func TestDefaultDocDownloader_FallbackFilename(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`))
		case strings.HasSuffix(r.URL.Path, "/getFile"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"file_id":"f1","file_path":"x.pdf"}}`))
		case strings.Contains(r.URL.Path, "/file/bot"):
			_, _ = w.Write([]byte("X"))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer stub.Close()

	b, _ := bot.New("token", bot.WithServerURL(stub.URL))
	_, filename, err := defaultDocDownloader(context.Background(), b, &models.Document{FileID: "f1"})
	if err != nil {
		t.Fatalf("defaultDocDownloader: %v", err)
	}
	if filename != "document" {
		t.Errorf("filename = %q, want 'document' fallback", filename)
	}
}

func TestBotReplier_PropagatesError(t *testing.T) {
	r := &botReplier{b: failingBot(t), logger: discardLogger()}
	if err := r.Reply(context.Background(), 1, "hello"); err == nil {
		t.Error("expected error on failed SendMessage")
	}
}

// --- runWizardSweeper observability tests --------------------------------

// bufferHandler is a goroutine-safe slog.Handler that captures records.
type bufferHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *bufferHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *bufferHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}
func (h *bufferHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *bufferHandler) WithGroup(string) slog.Handler      { return h }

func (h *bufferHandler) find(msg string) *slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.records {
		if h.records[i].Message == msg {
			return &h.records[i]
		}
	}
	return nil
}

func TestRunWizardSweeper_LogsNonZeroEvictions(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	sessions := telegramadapter.NewInMemoryWizardSessions(
		telegramadapter.WithSessionTTL(1*time.Minute),
		telegramadapter.WithSessionClock(func() time.Time { return base }),
	)
	sessions.Set(42, &telegramadapter.WizardState{ChatID: 42, StartedAt: base.Add(-10 * time.Minute)})

	bh := &bufferHandler{}
	logger := slog.New(bh)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runWizardSweeper(ctx, sessions, 5*time.Millisecond, logger, nil)
		close(done)
	}()

	// Poll for the eviction log.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if bh.find("wizard sessions swept") != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done

	rec := bh.find("wizard sessions swept")
	if rec == nil {
		t.Fatal("expected 'wizard sessions swept' log record within 500ms")
	}
	if rec.Level != slog.LevelInfo {
		t.Errorf("level = %s, want Info", rec.Level)
	}
	var removed int64
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == "removed" {
			removed = a.Value.Int64()
			return false
		}
		return true
	})
	if removed < 1 {
		t.Errorf("removed attr = %d, want >= 1", removed)
	}
}

func TestRunWizardSweeper_SilentWhenNothingToEvict(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	sessions := telegramadapter.NewInMemoryWizardSessions(
		telegramadapter.WithSessionTTL(1*time.Minute),
		telegramadapter.WithSessionClock(func() time.Time { return base }),
	)
	sessions.Set(7, &telegramadapter.WizardState{ChatID: 7, StartedAt: base})

	bh := &bufferHandler{}
	logger := slog.New(bh)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	go func() {
		runWizardSweeper(ctx, sessions, 5*time.Millisecond, logger, nil)
		close(done)
	}()

	// Let the sweeper tick a few times.
	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done

	if rec := bh.find("wizard sessions swept"); rec != nil {
		t.Errorf("unexpected sweep log on no-op runs: %+v", rec)
	}
}

func TestRunWizardSweeper_IncrementsEvictedCounter(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	sessions := telegramadapter.NewInMemoryWizardSessions(
		telegramadapter.WithSessionTTL(1*time.Minute),
		telegramadapter.WithSessionClock(func() time.Time { return base }),
	)
	// Three stale sessions — one tick must increment wizard_evicted by 3.
	for i := int64(1); i <= 3; i++ {
		sessions.Set(i, &telegramadapter.WizardState{ChatID: i, StartedAt: base.Add(-10 * time.Minute)})
	}

	coll := metrics.NewCollector()
	logger := slog.New(&bufferHandler{})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runWizardSweeper(ctx, sessions, 5*time.Millisecond, logger, coll)
		close(done)
	}()

	// Wait for evictions to land in the collector.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		var buf strings.Builder
		_, _ = coll.Render(&buf)
		if strings.Contains(buf.String(), `dealsense_bot_events_total{event="wizard_evicted"} 3`) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done

	var buf strings.Builder
	if _, err := coll.Render(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), `dealsense_bot_events_total{event="wizard_evicted"} 3`) {
		t.Errorf("collector missing wizard_evicted=3 after sweep; got:\n%s", buf.String())
	}
}

func TestRunWizardSweeper_NilCounter_DoesNotPanic(t *testing.T) {
	// Defensive — wiring may omit the counter before the metrics listener
	// is enabled.
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	sessions := telegramadapter.NewInMemoryWizardSessions(
		telegramadapter.WithSessionTTL(1*time.Minute),
		telegramadapter.WithSessionClock(func() time.Time { return base }),
	)
	sessions.Set(1, &telegramadapter.WizardState{ChatID: 1, StartedAt: base.Add(-10 * time.Minute)})

	logger := slog.New(&bufferHandler{})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		runWizardSweeper(ctx, sessions, 5*time.Millisecond, logger, nil)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done
}

func TestRunWizardSweeper_ReturnsImmediatelyIfCtxAlreadyDone(t *testing.T) {
	sessions := telegramadapter.NewInMemoryWizardSessions()
	logger := slog.New(&bufferHandler{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan struct{})
	go func() {
		runWizardSweeper(ctx, sessions, 5*time.Millisecond, logger, nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("runWizardSweeper did not return on already-canceled ctx")
	}
}

// TestMain_BinaryFastExit runs the compiled binary and checks two
// configuration-failure exits. The graceful-SIGINT path is covered by
// TestRun_StartsAndStops without depending on external subprocess timing.
func TestMain_BinaryFastExit(t *testing.T) {
	if testing.Short() {
		t.Skip("skip subprocess build in -short")
	}
	binPath := t.TempDir() + "/bot"
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Missing token → fast non-zero exit with message.
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(), "TELEGRAM_BOT_TOKEN=")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("expected non-zero exit on missing token, got success: %s", out)
	}
	if !strings.Contains(string(out), "TELEGRAM_BOT_TOKEN") {
		t.Errorf("expected token-missing message, got: %s", out)
	}

	// Empty allowlist is no longer a fatal-config error — ParseAllowlist
	// returns an open allowlist. The "fast non-zero exit" check was removed
	// alongside the open-mode change (was a closed-system invariant the
	// product now relaxes for bootstrap UX).
	_ = syscall.SIGINT // keep import live in case build tags change
}
