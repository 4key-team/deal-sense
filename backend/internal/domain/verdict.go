package domain

// Verdict represents a Go/No-Go decision for a tender.
type Verdict string

const (
	VerdictGo   Verdict = "go"
	VerdictNoGo Verdict = "no-go"
)

func ParseVerdict(s string) (Verdict, error) {
	switch s {
	case "go":
		return VerdictGo, nil
	case "no-go":
		return VerdictNoGo, nil
	default:
		return "", ErrInvalidVerdict
	}
}
