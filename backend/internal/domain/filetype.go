package domain

// FileType represents a supported document file type.
type FileType string

const (
	FileTypePDF  FileType = "pdf"
	FileTypeDOCX FileType = "docx"
	FileTypeMD   FileType = "md"
	// FileTypeDOC is the legacy Word 97-2003 binary format (Composite
	// Document File / OLE2). Parsing requires conversion to DOCX —
	// pure-Go support is not mature, so adapter/parser uses LibreOffice.
	FileTypeDOC FileType = "doc"
)

func ParseFileType(ext string) (FileType, error) {
	switch ext {
	case "pdf", ".pdf":
		return FileTypePDF, nil
	case "docx", ".docx":
		return FileTypeDOCX, nil
	case "md", ".md":
		return FileTypeMD, nil
	case "doc", ".doc":
		return FileTypeDOC, nil
	default:
		return "", ErrInvalidFileType
	}
}

func (ft FileType) String() string {
	return string(ft)
}
