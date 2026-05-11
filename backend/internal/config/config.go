package config

import (
	"cmp"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	LLMProvider string
	LLMBaseURL  string
	LLMAPIKey   string
	LLMModel    string
	LogLevel    string
	APIKey      string
	// RateLimitRPS is the per-IP requests-per-second cap. Zero disables.
	RateLimitRPS float64
	// RateLimitBurst is the bucket size for the per-IP limiter.
	RateLimitBurst int
}

func Load() Config {
	return Config{
		Port:           cmp.Or(os.Getenv("PORT"), "8080"),
		LLMProvider:    cmp.Or(os.Getenv("LLM_PROVIDER"), "anthropic"),
		LLMBaseURL:     os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:      os.Getenv("LLM_API_KEY"),
		LLMModel:       cmp.Or(os.Getenv("LLM_MODEL"), "claude-sonnet-4-5"),
		LogLevel:       strings.ToLower(cmp.Or(os.Getenv("LOG_LEVEL"), "info")),
		APIKey:         os.Getenv("DEAL_SENSE_API_KEY"),
		RateLimitRPS:   parseFloat(os.Getenv("RATE_LIMIT_RPS"), 0),
		RateLimitBurst: parseInt(os.Getenv("RATE_LIMIT_BURST"), 30),
	}
}

func parseFloat(s string, fallback float64) float64 {
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}

func parseInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
