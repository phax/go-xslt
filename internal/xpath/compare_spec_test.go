package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestCaValueComparison checks XPath 3.1 §3.7.2 (value comparisons) and the
// type pairs F&O 3.1's operator mapping (Appendix B.2) defines eq/ne/lt/… for.
func TestCaValueComparison(t *testing.T) {
	caRun(t, "<r><a>1</a><a>2</a></r>", nil, []caCase{
		{expr: "empty(() eq 1) and empty(1 lt ())", want: "true"},
		{expr: "(1, 2) eq 1", code: "XPTY0004"},
		{expr: "1 eq r/a", code: "XPTY0004"},
		// untypedAtomic is cast to xs:string, so it does not compare with numbers.
		{expr: "r/a[1] eq 1", code: "XPTY0004"},
		{expr: "r/a[1] eq '1'", want: "true"},
		{expr: "'1' eq 1", code: "XPTY0004"},
		{expr: "true() eq 1", code: "XPTY0004"},
		{expr: "true() eq 'true'", code: "XPTY0004"},
		{expr: "true() eq true() and false() lt true() and not(true() lt false())", want: "true"},
		{expr: "true() gt false() and true() ge true() and false() le false() and true() ne false()", want: "true"},
		{expr: "xs:anyURI('http://a') eq 'http://a'", want: "true"},
		{expr: "'abc' lt 'abd' and 'b' gt 'abc'", want: "true"},
		// Numeric promotion: decimal → float, then compare (B.1).
		{expr: "xs:float(1.01) eq 1.01", want: "true"},
		{expr: "1.01 eq xs:float(1.01)", want: "true"},
		{expr: "1 lt 1.5 and 2.5 gt 2 and 1e0 eq 1 and xs:float(2) gt 1", want: "true"},
		{expr: "100000000000000000001 gt 100000000000000000000", want: "true"},
		{expr: "xs:double('NaN') eq xs:double('NaN')", want: "false"},
		{expr: "xs:double('NaN') ne xs:double('NaN')", want: "true"},
		{expr: "xs:float('NaN') lt 1", want: "false"},
		// Date/time: same primitive only; gregorians are eq/ne only.
		{expr: "xs:dateTime('2000-01-01T12:00:00Z') eq xs:dateTime('2000-01-01T13:00:00+01:00')", want: "true"},
		{expr: "xs:date('2000-01-01') lt xs:date('2000-01-02') and xs:date('2000-01-02') gt xs:date('2000-01-01')", want: "true"},
		{expr: "xs:time('24:00:00') eq xs:time('00:00:00')", want: "true"},
		{expr: "xs:date('2000-01-01') eq xs:dateTime('2000-01-01T00:00:00')", code: "XPTY0004"},
		{expr: "xs:date('2000-01-01') eq 1", code: "XPTY0004"},
		{expr: "xs:date('2000-01-01') eq 'x'", code: "XPTY0004"},
		{expr: "xs:gYear('2000') eq xs:gYear('2000') and xs:gYear('2000') ne xs:gYear('2001')", want: "true"},
		{expr: "xs:gMonthDay('--12-25') eq xs:gMonthDay('--12-25')", want: "true"},
		{expr: "xs:gYear('2000') lt xs:gYear('2001')", code: "XPTY0004"},
		{expr: "xs:gDay('---01') gt xs:gDay('---02')", code: "XPTY0004"},
		{expr: "xs:dayTimeDuration('P1D') eq xs:date('2000-01-01')", code: "XPTY0004"},
		// Durations: eq across subtypes, order only within ym or dt.
		{expr: "xs:yearMonthDuration('P1Y') lt xs:yearMonthDuration('P13M')", want: "true"},
		{expr: "xs:dayTimeDuration('PT1H') gt xs:dayTimeDuration('PT59M') and xs:dayTimeDuration('PT60M') eq xs:dayTimeDuration('PT1H')", want: "true"},
		{expr: "xs:yearMonthDuration('P0M') eq xs:dayTimeDuration('PT0S')", want: "true"},
		{expr: "xs:duration('P1Y') eq xs:yearMonthDuration('P12M')", want: "true"},
		{expr: "xs:yearMonthDuration('P12M') eq xs:dayTimeDuration('P365D')", want: "false"},
		{expr: "xs:duration('P1Y') lt xs:duration('P2Y')", code: "XPTY0004"},
		{expr: "xs:yearMonthDuration('P1Y') lt xs:dayTimeDuration('P1D')", code: "XPTY0004"},
		{expr: "xs:dayTimeDuration('P1D') eq 'P1D'", code: "XPTY0004"},
		{expr: "xs:dayTimeDuration('P1D') eq true()", code: "XPTY0004"},
		{expr: "xs:dayTimeDuration('P1D') eq QName('', 'a')", code: "XPTY0004"},
		// QNames: eq/ne on expanded names only, no ordering.
		{expr: "QName('u', 'p:a') eq QName('u', 'q:a')", want: "true"},
		{expr: "QName('u', 'a') eq QName('v', 'a') or QName('u', 'a') eq QName('u', 'b')", want: "false"},
		{expr: "QName('u', 'a') ne QName('v', 'a')", want: "true"},
		{expr: "QName('u', 'a') lt QName('u', 'b')", code: "XPTY0004"},
		{expr: "QName('u', 'a') eq xs:hexBinary('01')", code: "XPTY0004"},
		// Binaries are ordered in 3.1 (F&O §12.1), same type only.
		{expr: "xs:hexBinary('01') lt xs:hexBinary('02') and xs:hexBinary('0102') gt xs:hexBinary('01')", want: "true"},
		{expr: "xs:base64Binary('AQ==') eq xs:base64Binary('AQ==')", want: "true"},
		{expr: "xs:hexBinary('01') eq xs:base64Binary('AQ==')", code: "XPTY0004"},
	})
}

// TestCaValueComparisonCollation checks that eq/ne and general comparisons
// on strings use the in-scope default collation (XPath 3.1 §3.7.2/§3.7.3),
// while non-string pairs are untouched by it.
func TestCaValueComparisonCollation(t *testing.T) {
	fold := func(c *Context) {
		c.DefaultCollation = func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }
	}
	caRun(t, "", fold, []caCase{
		{expr: "'A' eq 'a'", want: "true"},
		{expr: "'A' = ('x', 'a')", want: "true"},
		{expr: "'B' gt 'a'", want: "true"},
		{expr: "1 eq 1.0", want: "true"},
	})
}

// TestCaGeneralComparison checks XPath 3.1 §3.7.3: existential semantics and
// the untypedAtomic conversion rules.
func TestCaGeneralComparison(t *testing.T) {
	src := "<r><a>3</a><b>three</b><d>2000-01-01</d><c>3</c><e>p:x</e></r>"
	caRun(t, src, nil, []caCase{
		{expr: "(1, 2) = (2, 3)", want: "true"},
		{expr: "(1, 2) = (3, 4)", want: "false"},
		{expr: "(1, 2) != (1, 2)", want: "true"},
		{expr: "(1, 1) != 1", want: "false"},
		{expr: "() = ()", want: "false"},
		{expr: "(1, 2) < (0, 3) and (5, 1) > 4 and 1 <= (1, 0) and 2 >= (3, 2)", want: "true"},
		// untyped vs numeric → double; vs string → string; vs other → that type.
		{expr: "r/a = 3 and r/a = 3.0 and r/a = 3e0", want: "true"},
		{expr: "3 = r/a", want: "true"},
		{expr: "r/a = '3' and not(r/a = '3.0')", want: "true"},
		{expr: "r/b = 3", code: "FORG0001"},
		{expr: "3 = r/b", code: "FORG0001"},
		{expr: "r/d = xs:date('2000-01-01')", want: "true"},
		{expr: "xs:date('2000-01-01') = r/d", want: "true"},
		{expr: "r/b = xs:date('2000-01-01')", code: "FORG0001"},
		{expr: "r/a = r/c and not(r/a = r/b)", want: "true"},
		{expr: "r/a < r/b", want: "true"}, // both untyped: compared as strings
		// Typed incomparable pairs raise XPTY0004.
		{expr: "'a' = 1", code: "XPTY0004"},
		{expr: "1 = true()", code: "XPTY0004"},
		// NaN.
		{expr: "xs:double('NaN') = (1, xs:double('NaN'))", want: "false"},
		{expr: "xs:double('NaN') != 1", want: "true"},
		{expr: "xs:double('NaN') < 1", want: "false"},
	})
	// An untyped operand facing an xs:QName is cast in the expression's
	// static namespace context (§3.7.3, cast to xs:QName needs it).
	ns := func(c *Context) { c.NS = caNS{"p": "urn:x"} }
	caRun(t, src, ns, []caCase{
		{expr: "r/e = QName('urn:x', 'q:x')", want: "true"},
		{expr: "QName('urn:x', 'x') = r/e", want: "true"},
		{expr: "r/e = QName('urn:y', 'x')", want: "false"},
	})
}

// TestCaRangeComparison checks general comparisons against a range
// expression, whose meaning is that of comparing against every integer in
// the range (XPath 3.1 §3.3.1 + §3.7.3).
func TestCaRangeComparison(t *testing.T) {
	caRun(t, "<r><a>3</a><b>x</b></r>", nil, []caCase{
		{expr: "5 = (1 to 10)", want: "true"},
		{expr: "11 = (1 to 10)", want: "false"},
		{expr: "5.5 = (1 to 10)", want: "false"},
		{expr: "5.0 = (1 to 10)", want: "true"},
		{expr: "(1 to 10) = 5", want: "true"},
		{expr: "5 != (5 to 5)", want: "false"},
		{expr: "5 != (1 to 2)", want: "true"},
		{expr: "5 != (5 to 6)", want: "true"},
		{expr: "0 < (1 to 3)", want: "true"},
		{expr: "3 < (1 to 3)", want: "false"},
		{expr: "3 <= (1 to 3)", want: "true"},
		{expr: "4 <= (1 to 3)", want: "false"},
		{expr: "1 > (1 to 3)", want: "false"},
		{expr: "2 > (1 to 3)", want: "true"},
		{expr: "1 >= (1 to 3)", want: "true"},
		{expr: "0 >= (1 to 3)", want: "false"},
		{expr: "(1 to 3) < 0", want: "false"},
		{expr: "(1 to 3) < 2", want: "true"},
		{expr: "(1 to 3) <= 1", want: "true"},
		{expr: "(1 to 3) > 2", want: "true"},
		{expr: "(1 to 3) > 3", want: "false"},
		{expr: "(1 to 3) >= 3", want: "true"},
		{expr: "(1 to 3) >= 4", want: "false"},
		{expr: "1 = (3 to 1)", want: "false"},
		{expr: "1 != (3 to 1)", want: "false"},
		{expr: "() = (1 to 3)", want: "false"},
		{expr: "1 = (1 to ())", want: "false"},
		{expr: "(7, 2) = (1 to 3)", want: "true"},
		{expr: "1000000000000000000000 = (999999999999999999999 to 1000000000000000000001)", want: "true"},
		{expr: "1000000000000000000000.5 = (999999999999999999999 to 1000000000000000000001)", want: "false"},
		// Double/float operands: range members are promoted to their type.
		{expr: "xs:double(2) = (1 to 3)", want: "true"},
		{expr: "2.5e0 = (1 to 3)", want: "false"},
		{expr: "4e0 = (1 to 3)", want: "false"},
		{expr: "xs:double('NaN') = (1 to 3)", want: "false"},
		{expr: "xs:double('NaN') != (1 to 3)", want: "true"},
		{expr: "xs:double('NaN') < (1 to 3)", want: "false"},
		{expr: "1e0 != (1 to 1)", want: "false"},
		{expr: "1e0 != (1 to 2)", want: "true"},
		{expr: "xs:float(2) <= (1 to 3) and xs:float(2) = (1 to 3)", want: "true"},
		{expr: "xs:double(5) > (1 to 3) and xs:double(1) >= (1 to 3)", want: "true"},
		{expr: "xs:double(0) > (1 to 3) or xs:double(0) >= (1 to 3)", want: "false"},
		{expr: "xs:double(3) < (1 to 3)", want: "false"},
		{expr: "xs:double(3) <= (1 to 3)", want: "true"},
		// untyped → double; a non-numeric typed operand compares as usual.
		{expr: "r/a = (1 to 5)", want: "true"},
		{expr: "r/b = (1 to 5)", code: "FORG0001"},
		{expr: "'a' = (1 to 3)", code: "XPTY0004"},
	})
}

// TestCaNodeComparison checks XPath 3.1 §3.7.4 (is, <<, >>).
func TestCaNodeComparison(t *testing.T) {
	caRun(t, "<r><a/><b/></r>", nil, []caCase{
		{expr: "r/a is r/a", want: "true"},
		{expr: "r/a is r/b", want: "false"},
		{expr: "r/a << r/b and r/b >> r/a", want: "true"},
		{expr: "r/b << r/a or r/a >> r/b", want: "false"},
		{expr: "r << r/a and r/a/.. is r", want: "true"},
		{expr: "empty(() is r/a) and empty(r/a << ())", want: "true"},
		{expr: "1 is r/a", code: "XPTY0004"},
		{expr: "r/a is 'a'", code: "XPTY0004"},
		{expr: "r/* is r/a", code: "XPTY0004"},
		{expr: "r/a << r/*", code: "XPTY0004"},
	})
	// Nodes of different trees: the order is implementation-dependent but
	// must be stable and consistent (XDM 3.1 §2.4): exactly one of x << y
	// and y << x holds, and it is the same for every pair across the trees.
	d1, _ := xmltree.Parse("<x><p/><q/></x>")
	d2, _ := xmltree.Parse("<y><p/><q/></y>")
	vars := func(c *Context) { c.Vars = caVars{"d1": NodeSet{d1}, "d2": NodeSet{d2}} }
	caRun(t, "", vars, []caCase{
		{expr: "($d1 << $d2) ne ($d2 << $d1)", want: "true"},
		{expr: "($d1 << $d2) eq ($d1//p << $d2//q) and ($d1 << $d2) eq ($d1//q << $d2//p)", want: "true"},
		{expr: "($d1 >> $d2) eq ($d2 << $d1)", want: "true"},
	})
}

// TestCaBackwardsCompatibleComparison checks XPath 1.0 §3.4's comparison
// rules, which apply under XPath 1.0 compatibility mode.
func TestCaBackwardsCompatibleComparison(t *testing.T) {
	bc := func(c *Context) { c.BC10 = true }
	caRun(t, "<r><a>3</a><b>10</b><c>3</c><s>abc</s></r>", bc, []caCase{
		// Scalars: boolean beats number beats string for = and !=.
		{expr: "'false' = true()", want: "true"},
		{expr: "true() != 'x'", want: "false"},
		{expr: "1 = '1.0'", want: "true"},
		{expr: "'1.0' != 1", want: "false"},
		{expr: "'a' = 'a' and 'a' != 'b'", want: "true"},
		{expr: "'a' = 'b'", want: "false"},
		// Relational operators always compare numbers.
		{expr: "'10' > '2'", want: "true"},
		{expr: "'x' < 1 or 'x' >= 1", want: "false"},
		{expr: "true() > false() and 1 <= 1 and 2 >= 1 and 0 < 1", want: "true"},
		// Node-set vs boolean: boolean(node-set), also for an empty one.
		{expr: "r/missing = false()", want: "true"},
		{expr: "r/a = true()", want: "true"},
		{expr: "false() = r/missing", want: "true"},
		{expr: "true() != r/a", want: "false"},
		// Node-set vs number / string: per-node conversion.
		{expr: "r/a = 3 and 3 = r/a", want: "true"},
		{expr: "2 < r/a and r/a > 2 and r/b >= 10 and 10 <= r/b", want: "true"},
		{expr: "r/a = '3' and '3' = r/a and r/s != 'x'", want: "true"},
		{expr: "r/s < 'x'", want: "false"},
		// Node-set vs node-set: string-values; relational numerically.
		{expr: "r/a = r/c and not(r/a != r/c)", want: "true"},
		{expr: "r/a < r/b", want: "true"}, // 3 < 10 numerically, not as strings
		{expr: "r/* = r/c", want: "true"},
		// The range fast path is not taken in this mode; results agree.
		{expr: "2 = (1 to 3)", want: "true"},
		{expr: "(1 to 3) > 3", want: "false"},
	})
}

// TestCaEqIndex checks general "=" against long sequences of integers or
// strings (the hash-index path) against the §3.7.3 meaning, and that the
// index gives the same answer as the item-by-item comparison.
func TestCaEqIndex(t *testing.T) {
	ints := "(1 to 100, 200)"
	strs := "((for $i in 1 to 100 return string($i)), 'z')"
	cases := []caCase{
		{expr: "7 = " + ints, want: "true"},
		{expr: "300 = " + ints, want: "false"},
		{expr: "100000000000000000000 = " + ints, want: "false"},
		{expr: "5.0 = " + ints, want: "true"},
		{expr: "5.5 = " + ints, want: "false"},
		{expr: "100000000000000000000.0 = " + ints, want: "false"},
		{expr: "xs:untypedAtomic('7') = " + ints, want: "true"},
		{expr: "xs:untypedAtomic('7.5') = " + ints, want: "false"},
		{expr: "xs:untypedAtomic('x') = " + ints, code: "FORG0001"},
		{expr: "7e0 = " + ints, want: "true"},
		{expr: "7.5e0 = " + ints, want: "false"},
		{expr: "xs:double('INF') = " + ints, want: "false"},
		{expr: "xs:double('NaN') = " + ints, want: "false"},
		{expr: "1e300 = " + ints, want: "false"},
		{expr: "'7' = " + ints, code: "XPTY0004"},
		{expr: "(-9223372036854775808, 2 to 100) = -9223372036854775808", want: "true"},
		{expr: "(1 to 100, 'x') = 'x'", code: "XPTY0004"},
		{expr: "'z' = " + strs, want: "true"},
		{expr: "'q' = " + strs, want: "false"},
		{expr: "xs:anyURI('z') = " + strs, want: "true"},
		{expr: "xs:untypedAtomic('50') = " + strs, want: "true"},
		{expr: "1 = " + strs, code: "XPTY0004"},
		{expr: "1 = ((for $i in 1 to 100 return string($i)), 1)", code: "XPTY0004"},
		{expr: "'a' = ((for $i in 1 to 100 return xs:string($i)), 'a')", want: "true"},
		{expr: "true() = (for $i in 1 to 100 return $i mod 2 = 0)", want: "true"},
		{expr: "'r' = ((for $i in 1 to 100 return string($i)), /r)", want: "false"},
		{expr: "count((1 to 5)[. = (for $i in 1 to 100 return $i * 2)])", want: "2"},
		{expr: "let $s := (for $i in 1 to 100 return $i * 2) return count((1 to 9)[. = $s])", want: "4"},
	}
	for _, indexed := range []bool{true, false} {
		disableEqIndex = !indexed
		caRun(t, "", nil, cases)
	}
	disableEqIndex = false
}

// TestCaComparisonErrors checks that a dynamic error in an operand
// propagates out of every comparison form (XPath 3.1 §2.3.1).
func TestCaComparisonErrors(t *testing.T) {
	caRun(t, "<r><a>3</a></r>", nil, []caCase{
		{expr: "(1 div 0) eq 1", code: "FOAR0001"},
		{expr: "1 eq (1 div 0)", code: "FOAR0001"},
		{expr: "(1 div 0) = (1 to 3)", code: "FOAR0001"},
		{expr: "1 = ((1 div 0) to 3)", code: "FOAR0001"},
		{expr: "1 = (1 to (1 div 0))", code: "FOAR0001"},
		{expr: "1 = ('a' to 3)", code: "XPTY0004"},
		{expr: "1 = (1 to 'a')", code: "XPTY0004"},
		{expr: "(1 div 0) is r/a", code: "FOAR0001"},
		{expr: "(1 div 0) = 1", code: "FOAR0001"},
	})
	// An untyped value whose prefix is not bound does not cast to xs:QName.
	ns := func(c *Context) { c.NS = caNS{"p": "urn:x"} }
	caRun(t, "<r><e>zz:x</e></r>", ns, []caCase{
		{expr: "r/e = QName('urn:x', 'x')", code: "FONS0004"},
	})
	bc := func(c *Context) { c.BC10 = true }
	caRun(t, "<r><a>3</a></r>", bc, []caCase{
		{expr: "r/a = ()", want: "false"},
		{expr: "r/a = (1, true())", want: "true"},
		{expr: "4 > r/a and 3 >= r/a and not(3 > r/a)", want: "true"},
		{expr: "xs:boolean('true') = 'false'", want: "true"},
		{expr: "xs:boolean('false') = 0", want: "true"},
	})
}
