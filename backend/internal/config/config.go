package config

import (
	"cmp"
	"os"
)

type Config struct {
	Port        string
	LLMProvider string
	LLMBaseURL  string
	LLMAPIKey   string
	LLMModel    string
}

func Load() Config {
	return Config{
		Port:        cmp.Or(os.Getenv("PORT"), "8080"),
		LLMProvider: cmp.Or(os.Getenv("LLM_PROVIDER"), "anthropic"),
		LLMBaseURL:  os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:   os.Getenv("LLM_API_KEY"),
		LLMModel:    cmp.Or(os.Getenv("LLM_MODEL"), "claude-sonnet-4-5"),
	}
}
