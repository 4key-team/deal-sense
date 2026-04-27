package domain

// FileType represents a supported document file type.
type FileType string

const (
	FileTypePDF  FileType = "pdf"
	FileTypeDOCX FileType = "docx"
	FileTypeMD   FileType = "md"
)

func ParseFileType(ext string) (FileType, error) {
	switch ext {
	case "pdf", ".pdf":
		return FileTypePDF, nil
	case "docx", ".docx":
		return FileTypeDOCX, nil
	case "md", ".md":
		return FileTypeMD, nil
	default:
		return "", ErrInvalidFileType
	}
}

func (ft FileType) String() string {
	return string(ft)
}
