package pdf

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/johnfercher/maroto/v2/pkg/repository"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

//go:embed fonts/Roboto-Regular.ttf fonts/Roboto-Bold.ttf
var fontsFS embed.FS

const fontFamily = "Roboto"

// MarotoPDFGenerator creates PDF proposals using maroto v2.
type MarotoPDFGenerator struct{}

func NewMarotoPDFGenerator() *MarotoPDFGenerator {
	return &MarotoPDFGenerator{}
}

func (g *MarotoPDFGenerator) Generate(_ context.Context, input usecase.ContentInput) ([]byte, error) {
	regularBytes, err := fontsFS.ReadFile("fonts/Roboto-Regular.ttf")
	if err != nil {
		return nil, fmt.Errorf("pdf: read regular font: %w", err)
	}
	boldBytes, err := fontsFS.ReadFile("fonts/Roboto-Bold.ttf")
	if err != nil {
		return nil, fmt.Errorf("pdf: read bold font: %w", err)
	}

	customFonts, err := repository.New().
		AddUTF8FontFromBytes(fontFamily, fontstyle.Normal, regularBytes).
		AddUTF8FontFromBytes(fontFamily, fontstyle.Bold, boldBytes).
		AddUTF8FontFromBytes(fontFamily, fontstyle.Italic, regularBytes).
		AddUTF8FontFromBytes(fontFamily, fontstyle.BoldItalic, boldBytes).
		Load()
	if err != nil {
		return nil, fmt.Errorf("pdf: load fonts: %w", err)
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithCustomFonts(customFonts).
		WithDefaultFont(&props.Font{Family: fontFamily, Size: 10}).
		Build()

	m := maroto.New(cfg)

	// Header
	g.addHeader(m, input)

	// Summary
	if input.Summary != "" {
		m.AddRows(text.NewRow(8, input.Summary, props.Text{
			Size:  10,
			Style: fontstyle.Italic,
			Color: &props.Color{Red: 80, Green: 80, Blue: 80},
			Top:   2,
		}))
		m.AddRows(row.New(4)) // spacer
	}

	// Sections
	for _, sec := range input.Sections {
		m.AddRows(text.NewRow(8, sec.Title, props.Text{
			Size:  12,
			Style: fontstyle.Bold,
			Top:   4,
		}))
		g.addContentLines(m, sec.Content)
		m.AddRows(row.New(3))
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("pdf: generate: %w", err)
	}

	return doc.GetBytes(), nil
}

const (
	lineHeight   = 5.0
	bulletIndent = 8.0
	charsPerLine = 90.0
)

func (g *MarotoPDFGenerator) addContentLines(m core.Maroto, content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* "):
			itemText := "•  " + strings.TrimSpace(trimmed[2:])
			h := estimateHeight(itemText)
			m.AddRows(text.NewRow(h, itemText, props.Text{
				Size: 10,
				Left: bulletIndent,
				Top:  1,
			}))
		default:
			h := estimateHeight(trimmed)
			m.AddRows(text.NewRow(h, trimmed, props.Text{
				Size: 10,
				Top:  1,
			}))
		}
	}
}

func estimateHeight(text string) float64 {
	wrappedLines := float64(len(text))/charsPerLine + 1
	return max(lineHeight, wrappedLines*lineHeight)
}

func (g *MarotoPDFGenerator) addHeader(m core.Maroto, input usecase.ContentInput) {
	meta := input.Meta
	if meta == nil {
		meta = map[string]string{}
	}

	client := meta["client"]
	project := meta["project"]
	date := meta["date"]

	title := "Коммерческое предложение"
	if project != "" {
		title = project
	}

	_ = m.RegisterHeader(
		row.New(16).Add(
			col.New(8).Add(
				text.New(title, props.Text{
					Size:  14,
					Style: fontstyle.Bold,
				}),
				text.New(client, props.Text{
					Top:   8,
					Size:  10,
					Color: &props.Color{Red: 80, Green: 80, Blue: 80},
				}),
			),
			col.New(4).Add(
				text.New(date, props.Text{
					Size:  10,
					Align: align.Right,
				}),
			),
		),
	)
}

// Ensure MarotoPDFGenerator implements PDFGenerator at compile time.
var _ usecase.PDFGenerator = (*MarotoPDFGenerator)(nil)
