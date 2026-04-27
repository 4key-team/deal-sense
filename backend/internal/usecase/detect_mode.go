package usecase

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// DetectTemplateMode scans DOCX XML entries for {{placeholder}} markers.
// Returns ModePlaceholder if any are found, ModeGenerative otherwise.
func DetectTemplateMode(templateData []byte) (domain.TemplateMode, error) {
	if len(templateData) == 0 {
		return "", fmt.Errorf("detect template mode: %w", domain.ErrEmptyTemplate)
	}

	r, err := zip.NewReader(bytes.NewReader(templateData), int64(len(templateData)))
	if err != nil {
		return "", fmt.Errorf("detect template mode: %w", err)
	}

	for _, f := range r.File {
		if !isDocxXMLEntry(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(rc)
		rc.Close()

		if strings.Contains(string(data), "{{") {
			return domain.ModePlaceholder, nil
		}
	}

	return domain.ModeGenerative, nil
}

func isDocxXMLEntry(name string) bool {
	return name == "word/document.xml" ||
		name == "word/header1.xml" ||
		name == "word/header2.xml" ||
		name == "word/header3.xml" ||
		name == "word/footer1.xml" ||
		name == "word/footer2.xml" ||
		name == "word/footer3.xml"
}
