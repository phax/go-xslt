package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

func xpathStrings(t *testing.T, expr string) ([]string, string) {
	t.Helper()
	r := engine.EvalXPath(engine.XPathRequest{Expression: expr})
	var out []string
	for _, it := range r.Items {
		out = append(out, it.Value)
	}
	var diag string
	if len(r.Diagnostics) > 0 {
		diag = r.Diagnostics[0].Message
	}
	return out, diag
}

// XPath 3.1 A.1.2 (occurrence-indicators): a '*', '+' or '?' straight after a
// SequenceType is its occurrence indicator, and what follows is an operator —
// including the word operators, which the lexer used to read as names.
func TestOperatorAfterOccurrenceIndicator(t *testing.T) {
	for expr, want := range map[string]string{
		"1 instance of xs:integer or false()":                             "true",
		"(1, 2) instance of xs:integer* or false()":                       "true",
		"(1, 2) instance of xs:integer+ and true()":                       "true",
		"1 instance of xs:integer? or false()":                            "true",
		"(1, 2) instance of xs:integer* = true()":                         "true",
		"((1, 2) instance of xs:integer*) or false()":                     "true",
		"(1, 2) instance of xs:integer* and (1, 2) instance of item()+":   "true",
		"() instance of xs:integer* and not(() instance of xs:integer+)":  "true",
		"(1, 2) instance of xs:string* or (1, 2) instance of xs:integer*": "true",
		"(1,2) instance of xs:integer* eq true()":                         "true",
		"(1,2) instance of xs:integer* div 2":                             "0.5",
		"(1,2) instance of xs:integer* mod 2":                             "1",
		"(1,2) treat as xs:integer* instance of xs:integer+":              "true",
		"count((1,2) treat as xs:integer*)":                               "2",
	} {
		got, diag := xpathStrings(t, expr)
		if len(got) != 1 || got[0] != want || diag != "" {
			t.Errorf("%s -> %v %q, want %s", expr, got, diag, want)
		}
	}
}

// What the grammar does not allow after a SequenceType stays a syntax error:
// '!', '/', a predicate, '=>', and cast / castable / treat / instance of
// bound tighter than the SequenceType's own operator.
func TestSyntaxErrorsAfterSequenceTypeRemain(t *testing.T) {
	for _, expr := range []string{
		"(1, 2) treat as xs:integer* ! string(.)",
		"(1, 2) instance of xs:integer* ! 1",
		"(1, 2) instance of xs:integer* [1]",
		"(1, 2) instance of xs:integer* / 1",
		"(1, 2) instance of xs:integer* => string()",
		"(1, 2) instance of xs:integer* instance of xs:boolean",
		"(1, 2) instance of xs:integer* cast as xs:string",
		"(1, 2) instance of xs:integer * 3",
	} {
		if _, diag := xpathStrings(t, expr); !strings.Contains(diag, "XPST0003") && !strings.Contains(diag, "unexpected") {
			t.Errorf("%s: want a syntax error, got %q", expr, diag)
		}
	}
}

func TestOperatorAfterOccurrenceIndicatorInXSLT(t *testing.T) {
	const xsl = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="r" select="/r/a"/>
    <out><xsl:value-of select="if (empty($r) or $r instance of element()* or count($r) eq 1) then 'yes' else 'no'"/></out>
  </xsl:template></xsl:stylesheet>`
	r := engine.Transform(engine.Request{Stylesheet: xsl, Source: "<r><a/><a/></r>"})
	if len(r.Diagnostics) != 0 || !strings.Contains(r.Output, ">yes</out>") {
		t.Errorf("%q %v", r.Output, r.Diagnostics)
	}
}
