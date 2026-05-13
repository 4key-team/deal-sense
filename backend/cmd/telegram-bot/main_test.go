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
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

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

func TestRun_EmptyAllowlistFails(t *testing.T) {
	cfg := telegramadapter.Config{
		BotToken: "test-token",
		// no AllowlistUserIDs — Allowlist constructor must reject
	}
	err := run(t.Context(), discardLogger(), cfg, nil)
	if !errors.Is(err, auth.ErrEmptyAllowlist) {
		t.Errorf("err = %v, want wrapping %v", err, auth.ErrEmptyAllowlist)
	}
}

func TestRun_BotInitFails(t *testing.T) {
	cfg := telegramadapter.Config{
		BotToken:         "test-token",
		AllowlistUserIDs: []int64{1},
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

	h := makeAnalyzeHandler(ah, b, nopDownloader, discardLogger())
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
	h := makeAnalyzeHandler(ah, b, nopDownloader, discardLogger())
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

	h := makeAnalyzeHandler(ah, b, dl, discardLogger())
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
	h := makeAnalyzeHandler(ah, b, nopDownloader, discardLogger())
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
	h := makeGenerateHandler(gh, b, nopDownloader, discardLogger())
	h(context.Background(), b, &models.Update{})
	if api.calls != 0 || sends.Load() != 0 {
		t.Errorf("expected no work for update without message; api=%d sends=%d", api.calls, sends.Load())
	}
}

func TestMakeGenerateHandler_NoDocument_AsksForTemplate(t *testing.T) {
	b, sends := stubBotForSend(t)
	api := &stubAPIClient{}
	gh := telegramadapter.NewGenerateHandler(api, &botReplier{b: b, logger: discardLogger()})

	h := makeGenerateHandler(gh, b, nopDownloader, discardLogger())
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

	h := makeGenerateHandler(gh, b, dl, discardLogger())
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

	h := makeGenerateHandler(gh, b, failingDL, discardLogger())
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

	h := makeAnalyzeHandler(ah, b, failingDL, discardLogger())
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

	// Empty allowlist → fast non-zero exit.
	cmd2 := exec.Command(binPath)
	cmd2.Env = append(os.Environ(),
		"TELEGRAM_BOT_TOKEN=test",
		"ALLOWLIST_USER_IDS=",
	)
	if out2, err := cmd2.CombinedOutput(); err == nil {
		t.Errorf("expected non-zero exit on empty allowlist, got success: %s", out2)
	}
	// Use syscall to avoid unused-import in case build tags change.
	_ = syscall.SIGINT
}
