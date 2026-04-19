package domain

// Proposal represents a commercial proposal generation request and its result.
type Proposal struct {
	templateName    string
	templateContent []byte
	parameters      map[string]string
	result          []byte
}

func NewProposal(templateName string, templateContent []byte, parameters map[string]string) (*Proposal, error) {
	if len(templateContent) == 0 {
		return nil, ErrEmptyTemplate
	}
	if templateName == "" {
		return nil, ErrEmptyTemplate
	}
	return &Proposal{
		templateName:    templateName,
		templateContent: templateContent,
		parameters:      parameters,
	}, nil
}

func (p *Proposal) TemplateName() string        { return p.templateName }
func (p *Proposal) TemplateContent() []byte      { return p.templateContent }
func (p *Proposal) Parameters() map[string]string { return p.parameters }
func (p *Proposal) Result() []byte                { return p.result }

func (p *Proposal) SetResult(data []byte) {
	p.result = data
}
