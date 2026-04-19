package domain

import (
	"errors"
	"fmt"
)

// MatchScore represents a 0–100 compatibility score.
type MatchScore struct {
	value int
}

func NewMatchScore(v int) (MatchScore, error) {
	if v < 0 || v > 100 {
		return MatchScore{}, fmt.Errorf("%w: score must be 0–100, got %d", ErrInvalidScore, v)
	}
	return MatchScore{value: v}, nil
}

func (s MatchScore) Value() int { return s.value }

var ErrInvalidScore = errors.New("invalid match score")
