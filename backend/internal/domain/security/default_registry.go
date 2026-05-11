package security

// NewDefaultEndpointRegistry returns an EndpointRegistry pre-populated with
// the risk level of every HTTP endpoint exposed by cmd/server. Coupling
// tests (Layer 4) consume this so a new route added to router.go but
// forgotten here will fail the suite — risk classification cannot be
// silently bypassed.
//
// Stub for the RED step: returns an empty registry, so the coverage test
// fails for every known path.
func NewDefaultEndpointRegistry() *EndpointRegistry {
	return NewEndpointRegistry()
}
