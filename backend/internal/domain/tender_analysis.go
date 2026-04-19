package domain

// TenderAnalysis represents a tender document analysis request and its result.
type TenderAnalysis struct {
	documents      []Document
	companyProfile string
	verdict        Verdict
	risk           Risk
	score          MatchScore
	summary        string
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

func (t *TenderAnalysis) SetResult(verdict Verdict, risk Risk, score MatchScore, summary string) {
	t.verdict = verdict
	t.risk = risk
	t.score = score
	t.summary = summary
}
