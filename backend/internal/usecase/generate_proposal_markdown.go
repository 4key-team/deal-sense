package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// executeMarkdown drives the .md-template flow. The structure (heading
// order, pre-filled bodies, meta) comes from the user-supplied
// markdown; the LLM is only asked to fill the empty sections.
// Pre-filled sections are passed through verbatim — this is the
// "hybrid Smart+Plain" mode (ADR-013).
//
// Output is a clean DOCX assembled from the merged content via the
// existing GenerativeEngine.GenerateClean — no DOCX template byte
// input is needed.
func (uc *GenerateProposal) executeMarkdown(
	ctx context.Context,
	templateName string,
	templateData []byte,
	contextFiles []FileInput,
	userParams map[string]string,
) (*domain.Proposal, domain.TokenUsage, error) {
	noUsage := domain.ZeroTokenUsage()

	if uc.generative == nil {
		return nil, noUsage, fmt.Errorf("markdown template requires a generative engine")
	}

	mdTmpl, err := ParseMarkdownTemplate(templateData)
	if err != nil {
		return nil, noUsage, err
	}

	proposal, err := domain.NewProposal(templateName, templateData, userParams)
	if err != nil {
		return nil, noUsage, err
	}
	proposal.SetMode(domain.ModeMarkdown)

	// Parse context files (best-effort: skip ones the parser cannot read).
	var contextText strings.Builder
	for _, f := range contextFiles {
		text, perr := uc.parser.Parse(ctx, f.Name, f.Data)
		if perr != nil {
			continue
		}
		fmt.Fprintf(&contextText, "=== %s ===\n%s\n\n", f.Name, text)
	}

	// Build a description of the template skeleton for the LLM. Filled
	// sections are shown so the LLM can keep tone/voice consistent, but
	// the instruction explicitly tells it to ONLY generate content for
	// empty headings — anything it returns for already-filled titles
	// will be overwritten on merge.
	var skeleton strings.Builder
	if mdTmpl.Title != "" {
		fmt.Fprintf(&skeleton, "# %s\n", mdTmpl.Title)
	}
	for k, v := range mdTmpl.Meta {
		fmt.Fprintf(&skeleton, "- **%s:** %s\n", k, v)
	}
	skeleton.WriteByte('\n')
	for _, sec := range mdTmpl.Sections {
		fmt.Fprintf(&skeleton, "## %s\n", sec.Title)
		if sec.RawBody != "" {
			fmt.Fprintf(&skeleton, "(already filled — do not regenerate)\n%s\n\n", sec.RawBody)
		} else {
			skeleton.WriteString("(empty — generate content here)\n\n")
		}
	}

	userPrompt := fmt.Sprintf(
		"Markdown template (file %s):\n%s\nContext documents:\n%s\nUser parameters: %v\n\nGenerate content for every section that is marked empty. Use the same titles as in the template.",
		templateName, skeleton.String(), contextText.String(), userParams,
	)

	llmResp, usage, err := uc.llm.GenerateCompletion(ctx, uc.generativePrompt, userPrompt)
	if err != nil {
		return nil, noUsage, fmt.Errorf("llm completion: %w", err)
	}

	cleaned := extractJSON(llmResp)
	var resp proposalLLMResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, noUsage, fmt.Errorf("parse llm response: %w (raw: %.200s)", err, llmResp)
	}

	llmByTitle := make(map[string]string, len(resp.Sections))
	for _, s := range resp.Sections {
		llmByTitle[strings.TrimSpace(s.Title)] = s.Content
	}

	// Merge: template order wins, RawBody wins over LLM for filled
	// sections, LLM content fills the rest. Missing LLM content for an
	// empty section becomes an empty string (downstream renderers skip
	// it cleanly).
	contentSections := make([]ContentSection, 0, len(mdTmpl.Sections))
	for _, sec := range mdTmpl.Sections {
		content := sec.RawBody
		if content == "" {
			content = llmByTitle[strings.TrimSpace(sec.Title)]
		}
		contentSections = append(contentSections, ContentSection{Title: sec.Title, Content: content})
	}

	mergedMeta := map[string]string{}
	maps.Copy(mergedMeta, mdTmpl.Meta)
	maps.Copy(mergedMeta, resp.Meta)
	maps.Copy(mergedMeta, userParams)

	contentInput := ContentInput{Meta: mergedMeta, Sections: contentSections, Summary: resp.Summary}

	filled, err := uc.generative.GenerateClean(ctx, contentInput)
	if err != nil {
		return nil, noUsage, fmt.Errorf("md clean fill: %w", err)
	}
	proposal.SetResult(filled)

	// Map sections for the response. Status: ai for LLM-filled, filled
	// for pre-existing raw bodies — keeps the existing UI badges meaningful.
	domainSections := make([]domain.ProposalSection, 0, len(mdTmpl.Sections))
	for _, sec := range mdTmpl.Sections {
		status := domain.SectionAI
		tokens := 0
		if sec.RawBody != "" {
			status = domain.SectionFilled
		}
		// Borrow token count from LLM response when present.
		for _, ls := range resp.Sections {
			if strings.TrimSpace(ls.Title) == strings.TrimSpace(sec.Title) {
				tokens = ls.Tokens
				break
			}
		}
		ps, perr := domain.NewProposalSection(sec.Title, status, tokens)
		if perr != nil {
			continue
		}
		domainSections = append(domainSections, ps)
	}
	proposal.SetSections(domainSections, resp.Summary)
	proposal.SetMeta(mergedMeta)

	logEntries := make([]domain.LogEntry, 0, len(resp.Log))
	for _, l := range resp.Log {
		entry, err := domain.NewLogEntry(l.Time, l.Msg)
		if err != nil {
			continue
		}
		logEntries = append(logEntries, entry)
	}
	proposal.SetLog(logEntries)

	// Best-effort PDF / MD outputs reuse the same content input.
	if uc.docxToPDF != nil && len(filled) > 0 {
		if pdfBytes, convErr := uc.docxToPDF.Convert(ctx, filled); convErr == nil {
			proposal.SetPDFResult(pdfBytes)
		} else {
			uc.log().Warn("docx-to-pdf conversion failed (md mode)", "err", convErr)
		}
	}
	if proposal.PDFResult() == nil && uc.pdfGen != nil {
		if pdfBytes, pdfErr := uc.pdfGen.Generate(ctx, contentInput); pdfErr == nil {
			proposal.SetPDFResult(pdfBytes)
		} else {
			uc.log().Warn("maroto pdf generation failed (md mode)", "err", pdfErr)
		}
	}
	if uc.mdGen != nil {
		if mdBytes, mdErr := uc.mdGen.Render(ctx, contentInput); mdErr == nil {
			proposal.SetMDResult(mdBytes)
		} else {
			uc.log().Warn("markdown generation failed (md mode)", "err", mdErr)
		}
	}

	return proposal, usage, nil
}
