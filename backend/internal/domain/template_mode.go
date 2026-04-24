package domain

// TemplateMode represents the mode used for template processing.
type TemplateMode string

const (
	ModePlaceholder TemplateMode = "placeholder"
	ModeGenerative  TemplateMode = "generative"
	ModeAutoMarkup  TemplateMode = "automarkup"
)

func ParseTemplateMode(s string) (TemplateMode, error) {
	switch s {
	case "placeholder":
		return ModePlaceholder, nil
	case "generative":
		return ModeGenerative, nil
	case "automarkup":
		return ModeAutoMarkup, nil
	default:
		return "", ErrInvalidTemplateMode
	}
}

func (m TemplateMode) String() string { return string(m) }
