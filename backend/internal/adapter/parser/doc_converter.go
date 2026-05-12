package parser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

const docConverterTimeout = 60 * time.Second

// ErrEmptyDocInput is returned when ConvertToDOCX is called with no bytes.
var ErrEmptyDocInput = errors.New("doc converter: empty input")

// DocConverter shells out to LibreOffice headless to convert legacy
// Word 97-2003 (.doc) binaries into .docx. LibreOffice is already
// available in the backend image for DOCX→PDF rendering.
type DocConverter struct{}

func NewDocConverter() *DocConverter {
	return &DocConverter{}
}

func (c *DocConverter) ConvertToDOCX(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmptyDocInput
	}

	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > docConverterTimeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, docConverterTimeout)
		defer cancel()
	}

	tmpDir, err := os.MkdirTemp("", "doc2docx-*")
	if err != nil {
		return nil, fmt.Errorf("doc2docx: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // best-effort cleanup

	inputPath := filepath.Join(tmpDir, "input.doc")
	if err := os.WriteFile(inputPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("doc2docx: write input: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx,
		"soffice", "--headless", "--norestore", "--convert-to", "docx",
		"--outdir", tmpDir, inputPath,
	)
	cmd.Stderr = &stderr
	// Force-close IO and reap the process if children keep stderr open
	// after the context is cancelled (Go 1.20+ semantics).
	cmd.WaitDelay = 1 * time.Second

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("doc2docx: soffice: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	outputPath := filepath.Join(tmpDir, "input.docx")
	out, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("doc2docx: read output: %w", err)
	}
	return out, nil
}

var _ usecase.DocConverter = (*DocConverter)(nil)
