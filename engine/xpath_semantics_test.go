package engine_test

import (
	"strings"
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
		// substring() with NaN, infinite or huge bounds (F&O 3.1 §5.4.3), the
		// same on every platform: Go's float-to-int conversion is not.
		{`substring("12345", 1, 0 div 0E0)`, ``},
		{`substring("12345", 0 div 0E0, 3)`, ``},
		{`substring("12345", -1 div 0E0, 1 div 0E0)`, ``},
		{`substring("12345", -42, 1 div 0E0)`, `12345`},
		{`substring("12345", 1e300)`, ``},
		{`substring("12345", 1, 1e300)`, `12345`},
		{`substring("12345", 1.5, 2.6)`, `234`},
		// math:exp is bit-exact everywhere (math-exp-006): amd64's math.Exp
		// is one ulp off on CPUs without FMA.
		{`math:exp(math:pi()) eq 23.140692632779267e0`, `true`},
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
	if want := "<r>EINHUNDERTVIERUNDDREISSIG|dritte|seconda</r>"; res.HasErrors() || !strings.Contains(res.Output, want) {
		t.Fatalf("got %q %+v, want %q", res.Output, res.Diagnostics, want)
	}
}

// A standalone expression sees the prefixes of the document element, the
// standard prefixes and the request's own bindings; an unbound prefix is an
// error rather than a silent "no namespace".
func TestXPathNamespacesAndVariables(t *testing.T) {
	const inv = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	ubl := `<Invoice xmlns="` + inv + `" xmlns:cbc="urn:cbc"><cbc:ID>R-1</cbc:ID><cbc:Note>a</cbc:Note><cbc:Note>b</cbc:Note></Invoice>`
	for _, c := range []struct {
		req  engine.XPathRequest
		want string
	}{
		{engine.XPathRequest{Expression: "string(/*/cbc:ID)", Source: ubl}, "R-1"},
		{engine.XPathRequest{Expression: "count(/i:Invoice/cbc:Note)", Source: ubl, Namespaces: map[string]string{"i": inv}}, "2"},
		{engine.XPathRequest{Expression: "count(/Invoice/cbc:Note)", Source: ubl, DefaultElementNamespace: inv}, "2"},
		{engine.XPathRequest{Expression: "count(/Q{" + inv + "}Invoice)", Source: ubl}, "1"},
		{engine.XPathRequest{Expression: "$n + count($notes)", Source: ubl, VariableExprs: map[string]string{"n": "10", "notes": "//cbc:Note"}}, "12"},
		{engine.XPathRequest{Expression: "concat($a, '!')", Variables: map[string]string{"a": "hi"}}, "hi!"},
		{engine.XPathRequest{Expression: "math:pi() > 3 and map:size(map{1: 2}) = 1 and xs:integer('1') = 1"}, "true"},
	} {
		r := engine.EvalXPath(c.req)
		if r.HasErrors() || r.Value != c.want {
			t.Errorf("%s: got %q %+v, want %q", c.req.Expression, r.Value, r.Diagnostics, c.want)
		}
	}
	if r := engine.EvalXPath(engine.XPathRequest{Expression: "/nope:x", Source: ubl}); !r.HasErrors() || r.Diagnostics[0].Code != "XPST0081" {
		t.Errorf("unbound prefix: want XPST0081, got %+v", r.Diagnostics)
	}
	if r := engine.EvalXPath(engine.XPathRequest{Expression: "$x", VariableExprs: map[string]string{"x": "1 +"}}); !r.HasErrors() || r.Diagnostics[0].Code != "XPST0003" {
		t.Errorf("bad variable expression: want XPST0003, got %+v", r.Diagnostics)
	}
}

// ParamExprs hands a stylesheet typed values; a broken expression is
// reported instead of silently becoming the string in Params.
func TestParamExprs(t *testing.T) {
	xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="n"/><xsl:param name="d"/><xsl:param name="m"/><xsl:param name="doc"/>
  <xsl:template name="xsl:initial-template"><r><xsl:value-of select="$n + 1, $d instance of xs:date, $m?a, count($doc//x)" xmlns:xs="http://www.w3.org/2001/XMLSchema"/></r></xsl:template>
</xsl:stylesheet>`
	req := engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template", ParamExprs: map[string]string{
		"n": "41", "d": "xs:date('2026-01-31')", "m": "map{'a': 'x'}", "doc": "parse-xml('<r><x/><x/></r>')",
	}}
	if res := engine.Transform(req); res.HasErrors() || !strings.Contains(res.Output, "<r>42 true x 2</r>") {
		t.Errorf("got %q %+v", res.Output, res.Diagnostics)
	}
	req.ParamExprs = map[string]string{"n": "41 +"}
	if res := engine.Transform(req); !res.HasErrors() || !strings.Contains(res.Diagnostics[0].Message, "parameter $n") {
		t.Errorf("broken expression: %+v", res.Diagnostics)
	}
}
