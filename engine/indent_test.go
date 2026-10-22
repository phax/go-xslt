package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// Serialization 3.1 lets indent="yes" add whitespace only where it cannot
// change the content: not next to a text node with other characters (§5.1.4),
// and for xhtml/html not next to an inline element (§6.1.4, §7.4.3).
func TestIndentLeavesMixedContentAlone(t *testing.T) {
	for _, method := range []string{"xml", "xhtml", "html"} {
		xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns="http://www.w3.org/1999/xhtml">
  <xsl:output method="` + method + `" indent="yes" omit-xml-declaration="yes"/>
  <xsl:template name="xsl:initial-template">
    <html><body><div><p>Hello <b>bold</b> world</p><p>x</p></div></body></html>
  </xsl:template>
</xsl:stylesheet>`
		res := engine.Transform(engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template"})
		if res.HasErrors() {
			t.Fatalf("%s: %+v", method, res.Diagnostics)
		}
		if !strings.Contains(res.Output, "<p>Hello <b>bold</b> world</p>") {
			t.Errorf("%s: whitespace was added inside mixed content:\n%s", method, res.Output)
		}
		if !strings.Contains(res.Output, "</p>\n") {
			t.Errorf("%s: element-only content should still be indented:\n%s", method, res.Output)
		}
	}
}

func TestIndentNotNextToInlineElement(t *testing.T) {
	xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns="http://www.w3.org/1999/xhtml">
  <xsl:output method="xhtml" indent="yes" omit-xml-declaration="yes"/>
  <xsl:template name="xsl:initial-template">
    <html><body><div><span>a</span><span>b</span></div></body></html>
  </xsl:template>
</xsl:stylesheet>`
	res := engine.Transform(engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template"})
	if !strings.Contains(res.Output, "<div><span>a</span><span>b</span></div>") {
		t.Errorf("whitespace was added next to an inline element:\n%s", res.Output)
	}
}

// The xhtml method writes XML, where an undeclared &nbsp; is not well-formed.
func TestXHTMLWritesNoNbspEntity(t *testing.T) {
	src := func(method string) string {
		return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns="http://www.w3.org/1999/xhtml">
  <xsl:output method="` + method + `" omit-xml-declaration="yes"/>
  <xsl:template name="xsl:initial-template"><html><body><p>a&#xa0;b</p></body></html></xsl:template>
</xsl:stylesheet>`
	}
	res := engine.Transform(engine.Request{Stylesheet: src("xhtml"), InitialTemplate: "xsl:initial-template"})
	if strings.Contains(res.Output, "&nbsp;") || !strings.Contains(res.Output, "a b") {
		t.Errorf("xhtml: got %q", res.Output)
	}
	res = engine.Transform(engine.Request{Stylesheet: src("html"), InitialTemplate: "xsl:initial-template"})
	if !strings.Contains(res.Output, "a&nbsp;b") {
		t.Errorf("html keeps &nbsp;: got %q", res.Output)
	}
}
