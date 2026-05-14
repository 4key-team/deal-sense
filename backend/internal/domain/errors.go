package domain

import "errors"

var (
	ErrInvalidFileType      = errors.New("unsupported file type")
	ErrInvalidVerdict       = errors.New("invalid verdict")
	ErrInvalidRisk          = errors.New("invalid risk")
	ErrInvalidReqStatus     = errors.New("invalid requirement status")
	ErrInvalidSectionStatus = errors.New("invalid section status")
	ErrEmptyTemplate        = errors.New("template content is empty")
	ErrEmptyContent         = errors.New("document content is empty")
	ErrEmptyTitle           = errors.New("title is empty")
	ErrEmptyLabel           = errors.New("label is empty")
	ErrEmptyName            = errors.New("name is empty")
	ErrEmptyMsg             = errors.New("message is empty")
	ErrEmptyCompany         = errors.New("company profile is empty")
	ErrInvalidScore         = errors.New("invalid match score")
	ErrInvalidTemplateMode  = errors.New("invalid template mode")
	ErrInvalidBotToken      = errors.New("bot token format invalid (expected <digits>:<secret>)")
	ErrInvalidLogLevel      = errors.New("invalid log level (expected: debug, info, warn, error)")
	ErrEmptyLLMProvider     = errors.New("LLM provider is empty")
	ErrEmptyLLMAPIKey       = errors.New("LLM API key is empty")
	ErrEmptyLLMModel        = errors.New("LLM model is empty")
	ErrInvalidLLMBaseURL    = errors.New("LLM base URL is not a valid absolute URL")
)
