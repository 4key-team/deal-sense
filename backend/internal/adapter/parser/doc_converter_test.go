package parser_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
)

// withFakeSoffice replaces PATH with a temp dir containing a "soffice"
// shell script that emulates a LibreOffice conversion run. The script
// body receives the original argv (after the shebang) and can either
// write a fake output file to --outdir or fail with a non-zero exit.
func withFakeSoffice(t *testing.T, script string) {
	t.Helper()
	bin := t.TempDir()
	path := filepath.Join(bin, "soffice")
	body := "#!/bin/sh\n" + script + "\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake soffice: %v", err)
	}
	t.Setenv("PATH", bin)
}

const fakeSofficeOK = `
OUTDIR=""
INPUT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --outdir) OUTDIR="$2"; shift 2 ;;
    *.doc)    INPUT="$1"; shift ;;
    *)        shift ;;
  esac
done
STEM=$(basename "$INPUT" .doc)
# Emit a minimal "DOCX" payload (PK magic + tag) so callers can verify bytes.
printf 'PK\003\004FAKEDOCX' > "$OUTDIR/$STEM.docx"
`

func TestDocConverter_ConvertToDOCX(t *testing.T) {
	t.Run("subprocess success returns produced docx bytes", func(t *testing.T) {
		withFakeSoffice(t, fakeSofficeOK)
		conv := parser.NewDocConverter()
		out, err := conv.ConvertToDOCX(context.Background(), []byte("legacy doc bytes"))
		if err != nil {
			t.Fatalf("ConvertToDOCX: %v", err)
		}
		if len(out) < 4 || string(out[:4]) != "PK\x03\x04" {
			t.Errorf("expected PK ZIP magic in output, got %q", string(out))
		}
		if !strings.Contains(string(out), "FAKEDOCX") {
			t.Errorf("expected FAKEDOCX tag in output, got %q", string(out))
		}
	})

	t.Run("subprocess failure returns error", func(t *testing.T) {
		withFakeSoffice(t, "exit 5")
		conv := parser.NewDocConverter()
		_, err := conv.ConvertToDOCX(context.Background(), []byte("x"))
		if err == nil {
			t.Fatal("expected error from failing subprocess")
		}
	})

	t.Run("empty input returns error without invoking subprocess", func(t *testing.T) {
		conv := parser.NewDocConverter()
		_, err := conv.ConvertToDOCX(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error on empty input")
		}
	})

	t.Run("context cancellation aborts long-running subprocess", func(t *testing.T) {
		// Fake soffice that hangs longer than the ctx deadline.
		withFakeSoffice(t, "sleep 5")
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		conv := parser.NewDocConverter()
		start := time.Now()
		_, err := conv.ConvertToDOCX(ctx, []byte("x"))
		elapsed := time.Since(start)
		if err == nil {
			t.Fatal("expected error from cancelled subprocess")
		}
		if elapsed > 2*time.Second {
			t.Errorf("subprocess not aborted on ctx cancel (took %v)", elapsed)
		}
	})
}
