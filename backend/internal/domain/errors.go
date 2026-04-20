package domain

import "errors"

var (
	ErrInvalidFileType = errors.New("unsupported file type")
	ErrInvalidVerdict  = errors.New("invalid verdict")
	ErrInvalidRisk     = errors.New("invalid risk")
	ErrInvalidReqStatus = errors.New("invalid requirement status")
	ErrInvalidSectionStatus = errors.New("invalid section status")
	ErrEmptyTemplate   = errors.New("template content is empty")
	ErrEmptyContent    = errors.New("document content is empty")
	ErrEmptyCompany    = errors.New("company profile is empty")
)
