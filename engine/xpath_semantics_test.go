package engine_test

import (
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// Cases the W3C suites cover, kept here so CI (which runs without them)
// guards them too.
func TestXPathSemantics(t *testing.T) {
	src := `<r xmlns:u="urn:unused"><a>1</a><a xml:id="  k   2 "/></r>`
	for _, c := range []struct{ expr, want string }{
		// SER 3.1 §2: only adjacent atomic values are separated by a space.
		{`serialize((//a, 'x', 'y', //a[1]))`, `<a xmlns:u="urn:unused">1</a><a xmlns:u="urn:unused" xml:id="k 2"/>x y<a xmlns:u="urn:unused">1</a>`},
		{`serialize((1, 2), map{'item-separator': '|'})`, `1|2`},
		// An element serialized out of its tree keeps its in-scope namespaces.
		{`serialize(//a[1])`, `<a xmlns:u="urn:unused">1</a>`},
		// xml:id is ID-normalized when the tree is built (XDM 3.1 §6.3.3).
		{`string(//a[2]/@xml:id)`, `k 2`},
		// Group nesting follows the pattern, not the positions.
		{`serialize(analyze-string('b', '(b(x?))')//*:match)`, `<fn:match xmlns:fn="http://www.w3.org/2005/xpath-functions"><fn:group nr="1">b<fn:group nr="2"/></fn:group></fn:match>`},
		{`serialize(analyze-string('b', '(b)(x?)')//*:match)`, `<fn:match xmlns:fn="http://www.w3.org/2005/xpath-functions"><fn:group nr="1">b</fn:group><fn:group nr="2"/></fn:match>`},
		// json-to-xml keeps an empty key; a leading BOM is ignored.
		{`string(json-to-xml('{"":1}')//*:number/@key = '')`, `true`},
		{`json-to-xml(codepoints-to-string(65279) || '[1]')//*:number = 1`, `true`},
		{`parse-json(codepoints-to-string(65279) || '[1]')(1)`, `1`},
		// Ranges over integers beyond int64, compared exactly.
		{`string-join(1000000000000000000000 to 1000000000000000000002, ',')`, `1000000000000000000000,1000000000000000000001,1000000000000000000002`},
		{`1000000000000000020001 = 1000000000000000000000 to 1000000000000010000003`, `true`},
		{`1e21 = 1000000000000000000000 to 1000000000000010000003`, `true`},
		{`2.5 = 1 to 3`, `false`},
	} {
		r := engine.EvalXPath(engine.XPathRequest{Expression: c.expr, Source: src})
		if len(r.Diagnostics) > 0 || r.Value != c.want {
			t.Errorf("%s\n got %q %+v\nwant %q", c.expr, r.Value, r.Diagnostics, c.want)
		}
	}
	r := engine.EvalXPath(engine.XPathRequest{Expression: `count(1 to 3000000000)`})
	if len(r.Diagnostics) == 0 || r.Diagnostics[0].Code != "XPDY0130" {
		t.Errorf("an over-long range must be XPDY0130, got %q %+v", r.Value, r.Diagnostics)
	}
}

func TestNumberWordsGerman(t *testing.T) {
	xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template"><r><xsl:number value="134" format="W" lang="de"/>|<xsl:number value="3" format="w" lang="de" ordinal="-e"/>|<xsl:number value="2" format="w" lang="it" ordinal="%spellout-ordinal-feminine"/></r></xsl:template>
</xsl:stylesheet>`
	res := engine.Transform(engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template"})
	if want := "<r>EINHUNDERTVIERUNDDREISSIG|dritte|seconda</r>"; res.HasErrors() || !contains(res.Output, want) {
		t.Fatalf("got %q %+v, want %q", res.Output, res.Diagnostics, want)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
