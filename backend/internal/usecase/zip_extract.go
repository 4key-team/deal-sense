package usecase

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
)

// ErrNoSupportedFiles is returned when a ZIP archive contains
// no files with supported types (PDF, DOCX).
var ErrNoSupportedFiles = errors.New("no supported files in archive")

// maxZipEntrySize limits the decompressed size of a single file
// inside a ZIP archive to prevent zip bomb attacks.
const maxZipEntrySize = 50 << 20 // 50 MB

// ExtractZip reads a ZIP archive and returns FileInputs for all
// supported files (PDF, DOCX) found inside. Directories, hidden
// files, macOS artifacts, and unsupported file types are skipped.
func ExtractZip(data []byte) ([]FileInput, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var inputs []FileInput
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(f.Name)
		if strings.HasPrefix(name, ".") || strings.HasPrefix(f.Name, "__") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(rc, maxZipEntrySize+1))
		rc.Close()
		if err != nil || int64(len(raw)) > maxZipEntrySize {
			continue
		}

		fi, err := NewFileInput(name, raw)
		if err != nil {
			continue
		}
		inputs = append(inputs, fi)
	}
	if len(inputs) == 0 {
		return nil, ErrNoSupportedFiles
	}
	return inputs, nil
}
