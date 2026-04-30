package pdf_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/pdf"
)

func TestLibreOfficeConverter_Convert(t *testing.T) {
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("soffice not available, skipping integration test")
	}

	// Minimal valid DOCX is hard to construct without docxgo,
	// so this test only runs where LibreOffice is present.
	c := pdf.NewLibreOfficeConverter()
	_, err := c.Convert(context.Background(), []byte("not a docx"))
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestLibreOfficeConverter_EmptyInput(t *testing.T) {
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("soffice not available")
	}

	c := pdf.NewLibreOfficeConverter()
	_, err := c.Convert(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
}
