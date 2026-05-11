package llm

import "github.com/daniil/deal-sense/backend/internal/domain/security"

// Test-only re-exports of the package-internal policy loader seam.
// Production code does not use these.

// PolicyLoaderForTest returns the currently active policy loader. If a
// test override has been installed via SetPolicyLoaderForTest it is
// returned; otherwise the default loader is returned.
func PolicyLoaderForTest() func() (*security.Policy, error) {
	if l := policyLoaderMu.Load(); l != nil {
		return *l
	}
	return policyLoader
}

// SetPolicyLoaderForTest installs an override loader. Pass nil to restore
// the default. Reads of the active loader are atomic so the swap is safe
// while other goroutines are constructing wrapped prompts.
func SetPolicyLoaderForTest(f func() (*security.Policy, error)) {
	if f == nil {
		policyLoaderMu.Store(nil)
		return
	}
	policyLoaderMu.Store(&f)
}

func InitWrappedPromptsForTest() {
	initWrappedPrompts()
}
