package domain

// SectionStatus represents the generation status of a proposal section.
type SectionStatus string

const (
	SectionAI     SectionStatus = "ai"
	SectionFilled SectionStatus = "filled"
	SectionReview SectionStatus = "review"
)

func ParseSectionStatus(s string) (SectionStatus, error) {
	switch s {
	case "ai":
		return SectionAI, nil
	case "filled":
		return SectionFilled, nil
	case "review":
		return SectionReview, nil
	default:
		return "", ErrInvalidSectionStatus
	}
}

// ProposalSection represents a section of a generated proposal.
type ProposalSection struct {
	title  string
	status SectionStatus
	tokens int
}

func NewProposalSection(title string, status SectionStatus, tokens int) ProposalSection {
	return ProposalSection{title: title, status: status, tokens: tokens}
}

func (s ProposalSection) Title() string        { return s.title }
func (s ProposalSection) Status() SectionStatus { return s.status }
func (s ProposalSection) Tokens() int          { return s.tokens }

// Proposal represents a commercial proposal generation request and its result.
type Proposal struct {
	templateName    string
	templateContent []byte
	parameters      map[string]string
	result          []byte
	sections        []ProposalSection
	summary         string
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

func (p *Proposal) TemplateName() string         { return p.templateName }
func (p *Proposal) TemplateContent() []byte      { return p.templateContent }
func (p *Proposal) Parameters() map[string]string { return p.parameters }
func (p *Proposal) Result() []byte               { return p.result }
func (p *Proposal) Sections() []ProposalSection  { return p.sections }
func (p *Proposal) Summary() string              { return p.summary }

func (p *Proposal) SetResult(data []byte) {
	p.result = data
}

func (p *Proposal) SetSections(sections []ProposalSection, summary string) {
	p.sections = sections
	p.summary = summary
}
