package domain

// TemplateMode represents the mode used for template processing.
type TemplateMode string

const (
	ModePlaceholder TemplateMode = "placeholder"
	ModeGenerative  TemplateMode = "generative"
	ModeClean       TemplateMode = "clean"
	// ModeMarkdown processes a `.md` template whose `##` headings drive
	// the document structure. Empty sections are filled by the LLM,
	// pre-filled sections are passed through verbatim.
	ModeMarkdown TemplateMode = "markdown"
)

func ParseTemplateMode(s string) (TemplateMode, error) {
	switch s {
	case "placeholder":
		return ModePlaceholder, nil
	case "generative":
		return ModeGenerative, nil
	case "clean":
		return ModeClean, nil
	case "markdown":
		return ModeMarkdown, nil
	default:
		return "", ErrInvalidTemplateMode
	}
}

func (m TemplateMode) String() string { return string(m) }
