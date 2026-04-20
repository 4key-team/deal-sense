package domain

// RequirementStatus represents the match status of a requirement.
type RequirementStatus string

const (
	ReqMet     RequirementStatus = "met"
	ReqPartial RequirementStatus = "partial"
	ReqMiss    RequirementStatus = "miss"
)

func ParseRequirementStatus(s string) (RequirementStatus, error) {
	switch s {
	case "met":
		return ReqMet, nil
	case "partial":
		return ReqPartial, nil
	case "miss":
		return ReqMiss, nil
	default:
		return "", ErrInvalidReqStatus
	}
}

// ProCon is a strength or risk item.
type ProCon struct {
	title string
	desc  string
}

func NewProCon(title, desc string) ProCon {
	return ProCon{title: title, desc: desc}
}

func (p ProCon) Title() string { return p.title }
func (p ProCon) Desc() string  { return p.desc }

// Requirement is a tender requirement with a match status.
type Requirement struct {
	label  string
	status RequirementStatus
}

func NewRequirement(label string, status RequirementStatus) Requirement {
	return Requirement{label: label, status: status}
}

func (r Requirement) Label() string            { return r.label }
func (r Requirement) Status() RequirementStatus { return r.status }

// TenderAnalysis represents a tender document analysis request and its result.
type TenderAnalysis struct {
	documents      []Document
	companyProfile string
	verdict        Verdict
	risk           Risk
	score          MatchScore
	summary        string
	pros           []ProCon
	cons           []ProCon
	requirements   []Requirement
	effort         string
}

// Document holds extracted content from a single uploaded file.
type Document struct {
	name     string
	fileType FileType
	content  string
}

func NewDocument(name string, fileType FileType, content string) (*Document, error) {
	if content == "" {
		return nil, ErrEmptyContent
	}
	if name == "" {
		return nil, ErrEmptyContent
	}
	return &Document{
		name:     name,
		fileType: fileType,
		content:  content,
	}, nil
}

func (d *Document) Name() string       { return d.name }
func (d *Document) FileType() FileType { return d.fileType }
func (d *Document) Content() string    { return d.content }

func NewTenderAnalysis(documents []Document, companyProfile string) (*TenderAnalysis, error) {
	if len(documents) == 0 {
		return nil, ErrEmptyContent
	}
	if companyProfile == "" {
		return nil, ErrEmptyCompany
	}
	return &TenderAnalysis{
		documents:      documents,
		companyProfile: companyProfile,
	}, nil
}

func (t *TenderAnalysis) Documents() []Document { return t.documents }
func (t *TenderAnalysis) CompanyProfile() string { return t.companyProfile }
func (t *TenderAnalysis) Verdict() Verdict       { return t.verdict }
func (t *TenderAnalysis) Risk() Risk             { return t.risk }
func (t *TenderAnalysis) Score() MatchScore      { return t.score }
func (t *TenderAnalysis) Summary() string        { return t.summary }

func (t *TenderAnalysis) Pros() []ProCon          { return t.pros }
func (t *TenderAnalysis) Cons() []ProCon          { return t.cons }
func (t *TenderAnalysis) Requirements() []Requirement { return t.requirements }
func (t *TenderAnalysis) Effort() string           { return t.effort }

func (t *TenderAnalysis) SetResult(verdict Verdict, risk Risk, score MatchScore, summary string) {
	t.verdict = verdict
	t.risk = risk
	t.score = score
	t.summary = summary
}

func (t *TenderAnalysis) SetExtras(pros, cons []ProCon, reqs []Requirement, effort string) {
	t.pros = pros
	t.cons = cons
	t.requirements = reqs
	t.effort = effort
}
