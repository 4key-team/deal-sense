package domain

import "errors"

var (
	ErrInvalidFileType = errors.New("unsupported file type")
	ErrEmptyTemplate   = errors.New("template content is empty")
	ErrEmptyContent    = errors.New("document content is empty")
	ErrEmptyCompany    = errors.New("company profile is empty")
)
