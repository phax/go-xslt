package xslt

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/tim-riep/go-xslt/internal/fsaccess"
	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// ErrNoEmbeddedStylesheet reports a source document whose prolog has no
// xml-stylesheet processing instruction naming an XSLT stylesheet.
var ErrNoEmbeddedStylesheet = errors.New("the source has no xml-stylesheet processing instruction naming an XSLT stylesheet")

// xslStylesheetTypes are the media types an xml-stylesheet processing
// instruction uses for an XSLT stylesheet: the registered one (XSLT 3.0
// §3.13) and the ones in common use.
var xslStylesheetTypes = map[string]bool{
	"application/xslt+xml": true, "text/xsl": true, "text/xml": true, "application/xml": true,
}

// EmbeddedStylesheet returns, as text, the stylesheet module a source document
// carries inside itself and names in an xml-stylesheet processing instruction
// before its document element (<?xml-stylesheet type="text/xsl"
// href="#style1"?>, XSLT 3.0 §3.13 and the xml-stylesheet Recommendation).
// The fragment identifier names the module's outermost element by an ID —
// an attribute the DTD declares ID, or xml:id (an id attribute counts only
// with the is-id property, §3.13's note) — found by elementByID. The module keeps
// the namespaces it inherits from its ancestors and nothing else of them: the
// ancestors' attributes have no effect (embedded-stylesheet-001..003). Only a
// stylesheet inside the document itself (href="#id") is supported; the
// alternate="yes" ones are passed over.
//
// The source is parsed the way a transformation parses it (baseDir and files
// govern its external DTD).
func EmbeddedStylesheet(source, baseDir string, files *fsaccess.Guard) (string, error) {
	doc, err := xmltree.ParseLenient11WithBaseFS(source, baseDir, files)
	if err != nil {
		return "", fmt.Errorf("err:XTSE0010: the source is not well-formed: %v", err)
	}
	for _, c := range doc.Children {
		if c.Kind == xmltree.KindElement {
			break // the PI belongs to the prolog
		}
		if c.Kind != xmltree.KindPI || c.Name.Local != "xml-stylesheet" {
			continue
		}
		attrs := piPseudoAttrs(c.Value)
		if !xslStylesheetTypes[strings.ToLower(strings.TrimSpace(attrs["type"]))] || attrs["alternate"] == "yes" {
			continue
		}
		href := strings.TrimSpace(attrs["href"])
		id, ok := strings.CutPrefix(href, "#")
		if !ok || id == "" {
			return "", fmt.Errorf("err:XTSE0010: the xml-stylesheet processing instruction names %q; only a stylesheet embedded in the source (href=\"#id\") is supported", href)
		}
		el := elementByID(doc, id)
		if el == nil {
			return "", fmt.Errorf("err:XTSE0010: the source has no element with the ID %q that its xml-stylesheet processing instruction names", id)
		}
		return xmltree.Serialize(el, xmltree.SerializeOptions{Method: "xml", OmitXMLDeclaration: true}), nil
	}
	return "", ErrNoEmbeddedStylesheet
}

// piPseudoAttrs reads the pseudo-attributes of an xml-stylesheet processing
// instruction: name="value" or name='value' pairs separated by whitespace.
func piPseudoAttrs(s string) map[string]string {
	out := map[string]string{}
	for {
		s = strings.TrimLeft(s, " \t\r\n")
		eq := strings.IndexByte(s, '=')
		if eq <= 0 {
			return out
		}
		name := strings.TrimSpace(s[:eq])
		rest := strings.TrimLeft(s[eq+1:], " \t\r\n")
		if rest == "" || (rest[0] != '"' && rest[0] != '\'') {
			return out
		}
		end := strings.IndexByte(rest[1:], rest[0])
		if end < 0 {
			return out
		}
		out[name] = html.UnescapeString(rest[1 : 1+end]) // character and predefined-entity references
		s = rest[end+2:]
	}
}
