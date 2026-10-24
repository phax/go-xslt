package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// caVars is a VarResolver over a fixed map of unprefixed variable names.
type caVars map[string]Object

func (v caVars) ResolveVar(prefix, local string) (Object, bool) {
	o, ok := v[local]
	return o, ok && prefix == ""
}

// caNS is a NamespaceResolver over a fixed prefix map.
type caNS map[string]string

func (m caNS) ResolveNS(prefix string) (string, bool) { u, ok := m[prefix]; return u, ok }

// caCase is one expression and what the spec says it yields: want is the
// string value of the result, or code is the error code it must raise.
type caCase struct {
	expr string
	want string
	code string
}

// caEval parses expr and evaluates it with the document parsed from src as
// context node; setup (may be nil) adjusts the context first.
func caEval(t *testing.T, expr, src string, setup func(*Context)) (Object, error) {
	t.Helper()
	if src == "" {
		src = "<r/>"
	}
	doc, err := xmltree.Parse(src)
	if err != nil {
		t.Fatalf("parse xml %q: %v", src, err)
	}
	p, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	ctx := &Context{Node: doc, Pos: 1, Size: 1}
	if setup != nil {
		setup(ctx)
	}
	return p.Eval(ctx)
}

// caRun runs a table of caCase against src.
func caRun(t *testing.T, src string, setup func(*Context), cases []caCase) {
	t.Helper()
	for _, c := range cases {
		v, err := caEval(t, c.expr, src, setup)
		if c.code != "" {
			if err == nil {
				t.Errorf("%s: got %q, want error %s", c.expr, ToString(v), c.code)
			} else if !strings.Contains(err.Error(), c.code) {
				t.Errorf("%s: error %v, want %s", c.expr, err, c.code)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if got := ToString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

// TestCaNumericArithmetic checks the numeric operators against F&O 3.1 §4.2
// (op:numeric-add … op:numeric-mod) and XPath 3.1 §3.5 (operand conversion,
// type promotion B.1, cardinality).
func TestCaNumericArithmetic(t *testing.T) {
	caRun(t, "<r><a>3</a><b>x</b></r>", nil, []caCase{
		// Unary operators keep the operand's type (F&O §4.2.7/4.2.8).
		{expr: "-(3) instance of xs:integer and -(3) eq -3", want: "true"},
		{expr: "-(1.5) instance of xs:decimal and -(1.5) eq -1.5", want: "true"},
		{expr: "-xs:float(2) instance of xs:float and -xs:float(2) eq xs:float(-2)", want: "true"},
		{expr: "-xs:double(2) instance of xs:double and -xs:double(2) eq -2e0", want: "true"},
		{expr: "+(1.5) instance of xs:decimal", want: "true"},
		{expr: "empty(-())", want: "true"},
		// An untypedAtomic operand is cast to xs:double (§3.5).
		{expr: "-r/a instance of xs:double and -r/a eq -3e0", want: "true"},
		{expr: "-'a'", code: "XPTY0004"},
		{expr: "+xs:date('2000-01-01')", code: "XPTY0004"},
		{expr: "-r/b", code: "FORG0001"},
		// Cardinality: more than one item is a type error.
		{expr: "(1, 2) + 1", code: "XPTY0004"},
		{expr: "1 + (1, 2)", code: "XPTY0004"},
		{expr: "empty(() + 1) and empty(1 * ())", want: "true"},
		// Typed non-numeric operands are XPTY0004 (no implicit cast).
		{expr: "'3' + 1", code: "XPTY0004"},
		{expr: "xs:anyURI('1') + 1", code: "XPTY0004"},
		{expr: "xs:hexBinary('01') + 1", code: "XPTY0004"},
		{expr: "1 + xs:base64Binary('AQ==')", code: "XPTY0004"},
		{expr: "QName('', 'a') + 1", code: "XPTY0004"},
		{expr: "r/a + 1 instance of xs:double and r/a + 1 eq 4", want: "true"},
		// xs:integer × xs:integer.
		{expr: "(1 + 2) instance of xs:integer and 1 + 2 eq 3", want: "true"},
		{expr: "(xs:short(2) + xs:byte(3)) instance of xs:integer and not((xs:short(2) + xs:byte(3)) instance of xs:short)", want: "true"},
		{expr: "5 - 7 eq -2 and 6 * 7 eq 42", want: "true"},
		{expr: "(5 div 2) instance of xs:decimal and 5 div 2 eq 2.5", want: "true"},
		{expr: "-7 idiv 2 eq -3 and 7 idiv -2 eq -3", want: "true"},
		{expr: "-7 mod 3 eq -1 and 7 mod -3 eq 1", want: "true"},
		{expr: "1 div 0", code: "FOAR0001"},
		{expr: "1 idiv 0", code: "FOAR0001"},
		{expr: "1 mod 0", code: "FOAR0001"},
		{expr: "99999999999999999999 * 10 eq 999999999999999999990", want: "true"},
		// xs:decimal (and decimal × integer promoting the integer).
		{expr: "(1.5 + 1) instance of xs:decimal and 1.5 + 1 eq 2.5", want: "true"},
		{expr: "1 + 1.5 eq 2.5 and 2.5 - 1 eq 1.5 and 1.5 * 2 eq 3.0", want: "true"},
		{expr: "(1.5 div 0.5) instance of xs:decimal and 1.5 div 0.5 eq 3", want: "true"},
		{expr: "(7.5 idiv 2) instance of xs:integer and 7.5 idiv 2 eq 3 and -7.5 idiv 2 eq -3", want: "true"},
		{expr: "4.5 mod 1.2 eq 0.9 and -5.5 mod 2 eq -1.5", want: "true"},
		{expr: "1.5 div 0.0", code: "FOAR0001"},
		{expr: "1.5 idiv 0.0", code: "FOAR0001"},
		{expr: "1.5 mod 0.0", code: "FOAR0001"},
		// xs:float: decimal/integer promote to float (B.1), idiv stays integer.
		{expr: "(1.5 + xs:float(1)) instance of xs:float and 1.5 + xs:float(1) eq xs:float(2.5)", want: "true"},
		{expr: "(xs:float(7) idiv 2) instance of xs:integer and xs:float(7) idiv 2 eq 3", want: "true"},
		{expr: "xs:float(1) div 0 eq xs:float('INF')", want: "true"},
		{expr: "(xs:float(5) mod 3) instance of xs:float", want: "true"},
		// xs:double, IEEE 754 semantics.
		{expr: "(1 + 1e0) instance of xs:double", want: "true"},
		{expr: "(xs:float(1) + 1e0) instance of xs:double", want: "true"},
		{expr: "string(1e0 div 0)", want: "INF"},
		{expr: "string(-1e0 div 0)", want: "-INF"},
		{expr: "string(0e0 div 0)", want: "NaN"},
		{expr: "-7e0 mod 2 eq -1 and 7e0 mod -2 eq 1", want: "true"},
		{expr: "string(xs:double('INF') mod 2)", want: "NaN"},
		{expr: "(7e0 idiv 2) instance of xs:integer and 7e0 idiv 2 eq 3", want: "true"},
		{expr: "1e0 idiv xs:double('INF') eq 0", want: "true"},
		{expr: "(1e38 idiv 1e-37) instance of xs:integer and 1e38 idiv 1e-37 gt 1000000000000000000000000000000000000000000000000000000000000000000000000", want: "true"},
		{expr: "xs:double('NaN') idiv 1", code: "FOAR0002"},
		{expr: "1 idiv xs:double('NaN')", code: "FOAR0002"},
		{expr: "xs:double('-INF') idiv 1", code: "FOAR0002"},
		{expr: "1e0 idiv 0e0", code: "FOAR0001"},
		{expr: "1.0e0 - 0.5e0 eq 0.5e0 and 2e0 * 3e0 eq 6e0", want: "true"},
	})
}

// TestCaBackwardsCompatibleArithmetic checks XPath 1.0 compatibility mode
// (XPath 2.0 §3.4 / XSLT 3.0 §3.9.1): the operand is the FIRST item converted
// with fn:number, empty is NaN, and the result is xs:double.
func TestCaBackwardsCompatibleArithmetic(t *testing.T) {
	bc := func(c *Context) { c.BC10 = true }
	caRun(t, "<r><a>4</a><a>5</a></r>", bc, []caCase{
		{expr: "1 + (6 to 10)", want: "7"},
		{expr: "1 + r/a", want: "5"},
		{expr: "1 + ()", want: "NaN"},
		{expr: "'abc' + 1", want: "NaN"},
		{expr: "true() + 1", want: "2"},
		{expr: "'2' * '3'", want: "6"},
	})
}

// TestCaDateTimeDurationArithmetic checks F&O 3.1 §9.6/§10.6/§10.8 (duration
// and date/time arithmetic) and the operator mapping of XPath 3.1 Appendix B.2.
func TestCaDateTimeDurationArithmetic(t *testing.T) {
	caRun(t, "", nil, []caCase{
		// date/dateTime ± yearMonthDuration: day clamped (XSD 1.1 Part 2 E.3).
		{expr: "xs:date('2000-02-29') - xs:yearMonthDuration('P1Y') eq xs:date('1999-02-28')", want: "true"},
		{expr: "xs:date('2000-01-31') + xs:yearMonthDuration('P1M') eq xs:date('2000-02-29')", want: "true"},
		{expr: "xs:date('2000-03-31') - xs:yearMonthDuration('P13M') eq xs:date('1999-02-28')", want: "true"},
		{expr: "xs:date('2000-11-15') + xs:yearMonthDuration('P3M') eq xs:date('2001-02-15')", want: "true"},
		{expr: "xs:yearMonthDuration('P1M') + xs:date('2000-01-15') eq xs:date('2000-02-15')", want: "true"},
		{expr: "xs:dateTime('2000-01-01T00:00:00Z') + xs:dayTimeDuration('PT36H') eq xs:dateTime('2000-01-02T12:00:00Z')", want: "true"},
		{expr: "xs:dayTimeDuration('P1D') + xs:dateTime('2000-12-31T10:00:00') eq xs:dateTime('2001-01-01T10:00:00')", want: "true"},
		{expr: "xs:date('2000-01-01') - xs:dayTimeDuration('PT36H') eq xs:date('1999-12-30')", want: "true"},
		{expr: "(xs:date('2000-01-01') + xs:dayTimeDuration('P1D')) instance of xs:date", want: "true"},
		{expr: "xs:time('23:30:00') + xs:dayTimeDuration('PT1H') eq xs:time('00:30:00')", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') + xs:time('23:30:00') eq xs:time('00:30:00')", want: "true"},
		{expr: "xs:time('00:30:00') - xs:dayTimeDuration('PT1H') eq xs:time('23:30:00')", want: "true"},
		{expr: "string(xs:dateTime('2000-01-01T00:00:00+02:00') + xs:dayTimeDuration('PT1H'))", want: "2000-01-01T01:00:00+02:00"},
		{expr: "xs:time('10:00:00') + xs:yearMonthDuration('P1M')", code: "XPTY0004"},
		{expr: "xs:gYear('2000') + xs:yearMonthDuration('P1Y')", code: "XPTY0004"},
		{expr: "xs:yearMonthDuration('P1Y') + xs:gYear('2000')", code: "XPTY0004"},
		{expr: "xs:date('2000-01-01') * xs:dayTimeDuration('P1D')", code: "XPTY0004"},
		{expr: "xs:date('2000-01-01') + xs:duration('P1D')", code: "XPTY0004"},
		{expr: "xs:dayTimeDuration('P1D') - xs:date('2000-01-01')", code: "XPTY0004"},
		// Differences of date/time values are dayTimeDurations (§10.8.1–3).
		{expr: "xs:date('2000-03-01') - xs:date('2000-02-01') eq xs:dayTimeDuration('P29D')", want: "true"},
		{expr: "xs:time('12:00:00') - xs:time('10:30:00') eq xs:dayTimeDuration('PT1H30M')", want: "true"},
		{expr: "xs:dateTime('2000-01-02T00:00:00Z') - xs:dateTime('2000-01-01T12:00:00Z') eq xs:dayTimeDuration('PT12H')", want: "true"},
		{expr: "xs:dateTime('2000-01-01T00:00:00.5Z') - xs:dateTime('2000-01-01T00:00:00Z') eq xs:dayTimeDuration('PT0.5S')", want: "true"},
		{expr: "(xs:date('0001-01-01') - xs:date('2001-01-01')) lt xs:dayTimeDuration('-P730000D')", want: "true"},
		{expr: "xs:date('2000-01-01') - xs:dateTime('2000-01-01T00:00:00')", code: "XPTY0004"},
		{expr: "xs:gYear('2000') - xs:gYear('1999')", code: "XPTY0004"},
		{expr: "xs:date('2000-01-01') + xs:date('2000-01-01')", code: "XPTY0004"},
		// Duration ± duration of the same subtype (§8.4.1–4).
		{expr: "xs:yearMonthDuration('P1Y') + xs:yearMonthDuration('P2M') eq xs:yearMonthDuration('P1Y2M')", want: "true"},
		{expr: "xs:yearMonthDuration('P1Y') - xs:yearMonthDuration('P2M') eq xs:yearMonthDuration('P10M')", want: "true"},
		{expr: "xs:dayTimeDuration('P1D') + xs:dayTimeDuration('PT1H') eq xs:dayTimeDuration('P1DT1H')", want: "true"},
		{expr: "xs:dayTimeDuration('P1D') - xs:dayTimeDuration('P2D') eq xs:dayTimeDuration('-P1D')", want: "true"},
		{expr: "xs:yearMonthDuration('P1Y') + xs:dayTimeDuration('P1D')", code: "XPTY0004"},
		{expr: "xs:duration('P1Y') + xs:duration('P1Y')", code: "XPTY0004"},
		{expr: "xs:dayTimeDuration('PT1H') * xs:dayTimeDuration('PT1H')", code: "XPTY0004"},
		// Duration div duration is an xs:decimal (§8.4.7, §8.4.10).
		{expr: "(xs:yearMonthDuration('P3Y') div xs:yearMonthDuration('P1Y6M')) instance of xs:decimal", want: "true"},
		{expr: "xs:yearMonthDuration('P3Y') div xs:yearMonthDuration('P1Y6M') eq 2", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') div xs:dayTimeDuration('PT15M') eq 4", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') div xs:dayTimeDuration('PT0S')", code: "FOAR0001"},
		{expr: "xs:yearMonthDuration('P1Y') div xs:yearMonthDuration('P0M')", code: "FOAR0001"},
		// Duration × / div number (§8.4.5–6, §8.4.8–9).
		{expr: "xs:dayTimeDuration('PT1H') * 2 eq xs:dayTimeDuration('PT2H')", want: "true"},
		{expr: "2 * xs:yearMonthDuration('P1M') eq xs:yearMonthDuration('P2M')", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') * 0 eq xs:dayTimeDuration('PT0S')", want: "true"},
		{expr: "xs:yearMonthDuration('P5M') div 2 eq xs:yearMonthDuration('P3M')", want: "true"},
		{expr: "xs:yearMonthDuration('P1M') * -3.5 eq xs:yearMonthDuration('-P3M')", want: "true"},
		{expr: "xs:yearMonthDuration('P2Y') * 1.5 eq xs:yearMonthDuration('P3Y')", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') div 4 eq xs:dayTimeDuration('PT15M')", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') div xs:double('INF') eq xs:dayTimeDuration('PT0S')", want: "true"},
		{expr: "xs:yearMonthDuration('P1Y') div xs:double('-INF') eq xs:yearMonthDuration('P0M')", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') div 0", code: "FODT0002"},
		{expr: "xs:dayTimeDuration('PT1H') div xs:double('NaN')", code: "FOCA0005"},
		{expr: "xs:dayTimeDuration('PT1H') * xs:double('INF')", code: "FODT0002"},
		{expr: "xs:double('NaN') * xs:yearMonthDuration('P1M')", code: "FOCA0005"},
		{expr: "xs:double('INF') * xs:yearMonthDuration('P1M')", code: "FODT0002"},
		{expr: "xs:yearMonthDuration('P1M') * 1e16", code: "FODT0002"},
		{expr: "xs:duration('P1Y') * 2", code: "XPTY0004"},
		{expr: "xs:dayTimeDuration('PT1H') + 1", code: "XPTY0004"},
		{expr: "2 div xs:dayTimeDuration('PT1H')", code: "XPTY0004"},
	})
}
