package pdf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

const sofficeTimeout = 60 * time.Second

type LibreOfficeConverter struct{}

func NewLibreOfficeConverter() *LibreOfficeConverter {
	return &LibreOfficeConverter{}
}

func (c *LibreOfficeConverter) Convert(ctx context.Context, docxData []byte) ([]byte, error) {
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > sofficeTimeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sofficeTimeout)
		defer cancel()
	}
	tmpDir, err := os.MkdirTemp("", "docx2pdf-*")
	if err != nil {
		return nil, fmt.Errorf("docx2pdf: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.docx")
	if err := os.WriteFile(inputPath, docxData, 0o600); err != nil {
		return nil, fmt.Errorf("docx2pdf: write input: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx,
		"soffice", "--headless", "--norestore", "--convert-to", "pdf",
		"--outdir", tmpDir, inputPath,
	)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docx2pdf: soffice: %w (stderr: %s)", err, stderr.String())
	}

	outputPath := filepath.Join(tmpDir, "input.pdf")
	pdfData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("docx2pdf: read output: %w", err)
	}

	return pdfData, nil
}

var _ usecase.DOCXToPDFConverter = (*LibreOfficeConverter)(nil)
