package xsd

import (
	"strings"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// Documentation is one xs:documentation of an xs:annotation.
type Documentation struct {
	// Lang is the xml:lang of the element ("" when absent), Source its source
	// attribute (a URI, "" when absent).
	Lang, Source string
	// Text is the element's text content (mixed content flattened to plain
	// text), with leading and trailing white space removed.
	Text string
}

// docsOf reads the xs:annotation/xs:documentation children of a schema
// component's element. Nil unless the schema was compiled with CompileWithDocs
// (a validator has no use for them).
func (c *compiler) docsOf(node *xmltree.Node) []Documentation {
	if !c.keepDocs || node == nil {
		return nil
	}
	var out []Documentation
	for _, ann := range node.Children {
		if !isXS(ann, "annotation") {
			continue
		}
		for _, d := range ann.Children {
			if !isXS(d, "documentation") {
				continue
			}
			doc := Documentation{Text: strings.TrimSpace(d.StringValue())}
			doc.Source, _ = d.AttrLocal("source")
			for _, a := range d.Attrs {
				if a.Name.Space == xmlNS && a.Name.Local == "lang" {
					doc.Lang = strings.TrimSpace(a.Value)
				}
			}
			out = append(out, doc)
		}
	}
	return out
}
