package llm_test

import (
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
)

// TestWrappedPrompts_ConcurrentAccess pins the invariant that production
// reads (TenderAnalysisPrompt / ProposalGenerationPrompt /
// GenerativeProposalPrompt) and test-only reloads through
// InitWrappedPromptsForTest do not race.
//
// Run with `go test -race` — fails on the unsynchronised package-level
// `wrappedTender` etc. variables, passes after the atomic.Pointer
// refactor.
func TestWrappedPrompts_ConcurrentAccess(t *testing.T) {
	const readers = 50
	const reloads = 10

	var wg sync.WaitGroup

	wg.Add(readers)
	for range readers {
		go func() {
			defer wg.Done()
			for range 20 {
				_ = llm.TenderAnalysisPrompt("Russian")
				_ = llm.ProposalGenerationPrompt("Russian")
				_ = llm.GenerativeProposalPrompt("Russian")
			}
		}()
	}

	wg.Add(reloads)
	for range reloads {
		go func() {
			defer wg.Done()
			for range 5 {
				llm.InitWrappedPromptsForTest()
			}
		}()
	}

	wg.Wait()
}
