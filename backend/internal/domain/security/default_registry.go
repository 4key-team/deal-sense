package security

// NewDefaultEndpointRegistry returns an EndpointRegistry pre-populated with
// the risk level of every HTTP endpoint exposed by cmd/server. Coupling
// tests (Layer 4) consume this so a new route added to router.go but
// forgotten here will fail the suite — risk classification cannot be
// silently bypassed.
//
// Source of truth for this list: backend/internal/adapter/http/router.go.
// When adding an endpoint there, add it here too — the cross-check test
// in default_registry_test.go will fail otherwise.
func NewDefaultEndpointRegistry() *EndpointRegistry {
	r := NewEndpointRegistry()
	// SAFE_READ: pure reads, no persistence, no LLM mutation of user state.
	mustRegister(r, "/api/llm/check", RiskSafeRead)
	mustRegister(r, "/api/llm/providers", RiskSafeRead)
	mustRegister(r, "/api/llm/models", RiskSafeRead)
	mustRegister(r, "/api/tender/analyze", RiskSafeRead)
	// MODIFY: produces an artifact (DOCX/PDF/MD bytes) in the response.
	mustRegister(r, "/api/proposal/generate", RiskModify)
	return r
}

// mustRegister panics on duplicate / programmer error — Register only fails
// when a path is registered twice, which is a build-time bug in this
// curated list.
func mustRegister(r *EndpointRegistry, path string, level RiskLevel) {
	if err := r.Register(path, level); err != nil {
		panic(err)
	}
}
