package llm

import "github.com/daniil/deal-sense/backend/internal/domain/security"

// Test-only re-exports of the package-internal policy loader seam.
// Production code does not use these.

func PolicyLoaderForTest() func() (*security.Policy, error) {
	return policyLoader
}

func SetPolicyLoaderForTest(f func() (*security.Policy, error)) {
	policyLoader = f
}

func InitWrappedPromptsForTest() {
	initWrappedPrompts()
}
