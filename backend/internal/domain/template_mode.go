package domain

// TemplateMode represents the mode used for template processing.
type TemplateMode string

const (
	ModePlaceholder TemplateMode = "placeholder"
	ModeGenerative  TemplateMode = "generative"
	ModeClean       TemplateMode = "clean"
)

func ParseTemplateMode(s string) (TemplateMode, error) {
	switch s {
	case "placeholder":
		return ModePlaceholder, nil
	case "generative":
		return ModeGenerative, nil
	case "clean":
		return ModeClean, nil
	default:
		return "", ErrInvalidTemplateMode
	}
}

func (m TemplateMode) String() string { return string(m) }
