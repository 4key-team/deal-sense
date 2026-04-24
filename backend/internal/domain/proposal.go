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

func NewProposalSection(title string, status SectionStatus, tokens int) (ProposalSection, error) {
	if title == "" {
		return ProposalSection{}, ErrEmptyTitle
	}
	return ProposalSection{title: title, status: status, tokens: tokens}, nil
}

func (s ProposalSection) Title() string        { return s.title }
func (s ProposalSection) Status() SectionStatus { return s.status }
func (s ProposalSection) Tokens() int          { return s.tokens }

// LogEntry represents a step in the generation process.
type LogEntry struct {
	time string
	msg  string
}

func NewLogEntry(time, msg string) (LogEntry, error) {
	if msg == "" {
		return LogEntry{}, ErrEmptyMsg
	}
	return LogEntry{time: time, msg: msg}, nil
}
func (l LogEntry) Time() string             { return l.time }
func (l LogEntry) Msg() string              { return l.msg }

// Proposal represents a commercial proposal generation request and its result.
type Proposal struct {
	templateName    string
	templateContent []byte
	parameters      map[string]string
	result          []byte
	sections        []ProposalSection
	meta            map[string]string
	log             []LogEntry
	summary         string
	mode            TemplateMode
	pdfResult       []byte
}

func NewProposal(templateName string, templateContent []byte, parameters map[string]string) (*Proposal, error) {
	if len(templateContent) == 0 {
		return nil, ErrEmptyTemplate
	}
	if templateName == "" {
		return nil, ErrEmptyName
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
func (p *Proposal) Sections() []ProposalSection   { return p.sections }
func (p *Proposal) Meta() map[string]string       { return p.meta }
func (p *Proposal) Log() []LogEntry               { return p.log }
func (p *Proposal) Summary() string               { return p.summary }

func (p *Proposal) Mode() TemplateMode     { return p.mode }
func (p *Proposal) PDFResult() []byte       { return p.pdfResult }

func (p *Proposal) SetMode(m TemplateMode)      { p.mode = m }
func (p *Proposal) SetPDFResult(data []byte)     { p.pdfResult = data }

func (p *Proposal) SetResult(data []byte) {
	p.result = data
}

func (p *Proposal) SetSections(sections []ProposalSection, summary string) {
	p.sections = sections
	p.summary = summary
}

func (p *Proposal) SetMeta(meta map[string]string) {
	if meta == nil {
		meta = map[string]string{}
	}
	p.meta = meta
}

func (p *Proposal) SetLog(log []LogEntry) {
	if log == nil {
		log = []LogEntry{}
	}
	p.log = log
}
