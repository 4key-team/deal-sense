package dealsenseapi

import (
	"io"

	"github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// WriteAnalyzeMultipartForTest re-exports writeAnalyzeMultipart so tests can
// exercise its error paths by passing failing writers.
func WriteAnalyzeMultipartForTest(out io.Writer, req telegram.AnalyzeTenderRequest) (string, error) {
	return writeAnalyzeMultipart(out, req)
}
