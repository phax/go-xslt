package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// Limits stop a run that exceeds them, and fn:transform's nested runs spend
// from the caller's budget instead of getting their own.
func TestLimits(t *testing.T) {
	loop := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template">
    <xsl:iterate select="1 to 5000"><xsl:value-of select="."/></xsl:iterate>
  </xsl:template>
</xsl:stylesheet>`
	run := func(xsl string, l engine.Limits) engine.Result {
		return engine.Transform(engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template", Limits: l})
	}
	if res := run(loop, engine.Limits{}); res.HasErrors() {
		t.Fatalf("default limits: %+v", res.Diagnostics)
	}
	if res := run(loop, engine.Limits{MaxSteps: 1000}); !res.HasErrors() || res.Diagnostics[0].Code != "XPDY0130" {
		t.Fatalf("MaxSteps 1000: want XPDY0130, got %+v", res.Diagnostics)
	}

	deep := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:f="urn:f">
  <xsl:function name="f:down"><xsl:param name="n"/><xsl:sequence select="if ($n = 0) then 0 else f:down($n - 1)"/></xsl:function>
  <xsl:template name="xsl:initial-template"><r><xsl:value-of select="f:down(500)"/></r></xsl:template>
</xsl:stylesheet>`
	if res := run(deep, engine.Limits{}); res.HasErrors() {
		t.Fatalf("depth 500, default limit: %+v", res.Diagnostics)
	}
	if res := run(deep, engine.Limits{MaxDepth: 100}); !res.HasErrors() || res.Diagnostics[0].Code != "XPDY0130" {
		t.Fatalf("MaxDepth 100: want XPDY0130, got %+v", res.Diagnostics)
	}

	// The fewest steps the loop needs on its own, found by bisection; five
	// nested runs of it cannot fit in twice that.
	lo, hi := int64(1), int64(10_000_000)
	for lo < hi {
		mid := (lo + hi) / 2
		if run(loop, engine.Limits{MaxSteps: mid}).HasErrors() {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	nested := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="inner"/>
  <xsl:template name="xsl:initial-template">
    <xsl:for-each select="1 to 5"><xsl:sequence select="transform(map{'stylesheet-text': $inner, 'initial-template': QName('http://www.w3.org/1999/XSL/Transform', 'initial-template')})?output"/></xsl:for-each>
  </xsl:template>
</xsl:stylesheet>`
	runNested := func(steps int64) engine.Result {
		return engine.Transform(engine.Request{Stylesheet: nested, InitialTemplate: "xsl:initial-template",
			Params: map[string]string{"inner": loop}, Limits: engine.Limits{MaxSteps: steps}})
	}
	if res := runNested(10 * lo); res.HasErrors() {
		t.Fatalf("nested, enough steps: %+v", res.Diagnostics)
	}
	if res := runNested(2 * lo); !res.HasErrors() || !strings.Contains(res.Diagnostics[0].Message, "step budget") {
		t.Fatalf("nested runs must share the budget, got %+v", res.Diagnostics)
	}
}

// Input that would otherwise overflow the Go stack — a fatal error no
// recover() catches, ending the whole process — is refused with an error:
// endless inline-function recursion, absurdly nested expressions and
// absurdly nested documents.
func TestNoStackOverflow(t *testing.T) {
	for _, c := range []struct{ name, expr, src, want string }{
		{"inline recursion", `let $f := function($f, $n) { $f($f, $n + 1) } return $f($f, 0)`, "", "XPDY0130"},
		{"nested parentheses", strings.Repeat("(", 2_000_000) + "1" + strings.Repeat(")", 2_000_000), "", "XPDY0130"},
		{"nested elements", "count(//a)", strings.Repeat("<a>", 1_000_000) + strings.Repeat("</a>", 1_000_000), "nested more than"},
	} {
		r := engine.EvalXPath(engine.XPathRequest{Expression: c.expr, Source: c.src})
		if len(r.Diagnostics) == 0 || !strings.Contains(r.Diagnostics[0].Code+" "+r.Diagnostics[0].Message, c.want) {
			t.Errorf("%s: want %s, got %q %+v", c.name, c.want, r.Value, r.Diagnostics)
		}
	}
	deep := strings.Repeat("<a>", 1_000_000) + strings.Repeat("</a>", 1_000_000)
	copyAll := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"><xsl:copy-of select="."/></xsl:template></xsl:stylesheet>`
	if res := engine.Transform(engine.Request{Stylesheet: copyAll, Source: deep}); !res.HasErrors() {
		t.Error("a document nested a million levels deep must be refused")
	}
	if r := engine.Validate(engine.ValidateRequest{Schemas: []string{`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a"/></xs:schema>`}, Instance: deep}); r.Valid {
		t.Error("validate: a document nested a million levels deep must be refused")
	}
	// Just under the limit still works.
	ok := strings.Repeat("<a>", 9999) + strings.Repeat("</a>", 9999)
	if res := engine.Transform(engine.Request{Stylesheet: copyAll, Source: ok}); res.HasErrors() {
		t.Errorf("9,999 levels: %+v", res.Diagnostics)
	}
}
