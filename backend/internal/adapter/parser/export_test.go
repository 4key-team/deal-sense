package parser

import "github.com/daniil/deal-sense/backend/internal/usecase"

// InjectSectionsXMLForTest exposes the unexported zip-fallback XML
// injector for parser_test. The function is shaped exactly like its
// internal callsite in generativeFillZip — no behaviour change.
func (g *DocxGenerative) InjectSectionsXMLForTest(xml string, sections []usecase.ContentSection) []byte {
	return g.injectSectionsXML(xml, sections)
}
