package domain

// Risk represents a risk level for a tender.
type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

func ParseRisk(s string) (Risk, error) {
	switch s {
	case "low":
		return RiskLow, nil
	case "medium":
		return RiskMedium, nil
	case "high":
		return RiskHigh, nil
	default:
		return "", ErrInvalidRisk
	}
}
