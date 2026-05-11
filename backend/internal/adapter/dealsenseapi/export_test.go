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

// MultipartBuilderForTest reads/restores the multipartBuilder seam so tests
// can simulate the (otherwise unreachable) build-failure branch.
func MultipartBuilderForTest() func(telegram.AnalyzeTenderRequest) (io.Reader, string, error) {
	return multipartBuilder
}

// SetMultipartBuilderForTest replaces the multipartBuilder seam. Caller must
// restore the original before returning.
func SetMultipartBuilderForTest(fn func(telegram.AnalyzeTenderRequest) (io.Reader, string, error)) {
	multipartBuilder = fn
}
