package xpath

import (
	"errors"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestPeRangeAndPredicates: §3.5 range expressions and §3.3.3 numeric
// predicates.
func TestPeRangeAndPredicates(t *testing.T) {
	peCheck(t, peDoc, nil, [][2]string{
		// Each operand is converted to xs:integer? by the function
		// conversion rules: a sequence of two, a string or a decimal
		// cannot be.
		{"(1, 2) to 3", "err:XPTY0004"},
		{"1 to (2, 3)", "err:XPTY0004"},
		{"'a' to 3", "err:XPTY0004"},
		{"1.5 to 3", "err:XPTY0004"},
		{"string-join(xs:untypedAtomic('2') to 3, ',')", "2,3"},
		{"count(3 to 1)", "0"},
		{"count(() to 3)", "0"},
		{"count(1 to ())", "0"},
		{"1 to error()", "err:FOER0000"},
		{"error() to 1", "err:FOER0000"},
		// A numeric predicate selects the item whose position equals it; a
		// non-integral or non-finite value equals no position.
		{"count((1, 2, 3)[1.5])", "0"},
		{"count((1, 2, 3)[xs:double('NaN')])", "0"},
		{"count((1, 2, 3)[xs:double('INF')])", "0"},
		{"(5, 6, 7)[2]", "6"},
		{"(5, 6, 7)[2.0]", "6"},
		{"(5, 6, 7)[xs:double(3)]", "7"},
		{"string-join((1, 2, 3)[. > 1], ',')", "2,3"},
		{"string((/r/a[1], 'x')[1])", "one"},
		{"count((/r/a, 1)[. instance of element()])", "2"},
		{"(1, 2)[error()]", "err:FOER0000"},
		{"string-join((1 to 5)[position() = (2, 4)], ',')", "2,4"},
	})
}

// TestPePaths: §3.3 path expressions over a mix of nodes and atomic values.
func TestPePaths(t *testing.T) {
	peCheck(t, peDoc, nil, [][2]string{
		{"string-join(/r/a/string(), ',')", "one,two"},
		{"string-join(/r/a/name(), ',')", "a,a"},
		// E1/E2 evaluates E2 once per node of E1; atomic results are not
		// de-duplicated or reordered.
		{"count(/r/a/(1, 2))", "4"},
		{"count(/r/a/..)", "1"},
		// §3.3.1: a last step mixing nodes and atomics is XPTY0018.
		{"/r/(a, 1)", "err:XPTY0018"},
		// §3.3.2: an axis step needs a node context item (XPTY0020).
		{"1 ! a", "err:XPTY0020"},
		{"'a' ! @x", "err:XPTY0020"},
		{"(1, 2) ! child::node()", "err:XPTY0020"},
		{"1 ! /", "err:XPTY0020"},
		// §3.1.2 / §2.1.1: the xmlns prefix is never bound, and the xmlns
		// namespace may not appear in a URIQualifiedName.
		{"/r/xmlns:a", "err:XPST0081"},
		{"/r/Q{http://www.w3.org/2000/xmlns/}a", "err:XQST0070"},
		// Unions, intersections and differences need node operands (§3.4.2).
		{"(1, 2) intersect (1)", "err:XPTY0004"},
		{"/r except 1", "err:XPTY0004"},
		{"1 | 2", "err:XPTY0004"},
		{"count(() except /r)", "0"},
		{"count(/r/a intersect /r/a[1])", "1"},
		{"count(/r/* except /r/a)", "1"},
		{"count(/r/a | ())", "2"},
		{"/r/a intersect error()", "err:FOER0000"},
		{"error() except /r", "err:FOER0000"},
		{"error() | /r", "err:FOER0000"},
		{"string-join(/r/a ! name(), ',')", "a,a"},
		{"string-join((1, 2) ! (. * 10), ',')", "10,20"},
		{"/r/a[error()]", "err:FOER0000"},
		{"/r/error()", "err:FOER0000"},
	})
}

// TestPeFocusAbsent: §2.1.2 — with no context item, '.', a relative path and
// '/' are XPDY0002.
func TestPeFocusAbsent(t *testing.T) {
	for _, expr := range []string{".", "a", "/", "/r", "@x"} {
		p, err := Parse(expr)
		if err != nil {
			t.Fatal(err)
		}
		_, err = p.Eval(&Context{NoFocus: true})
		if err == nil || !strings.Contains(err.Error(), "XPDY0002") {
			t.Errorf("%s without focus: %v, want XPDY0002", expr, err)
		}
	}
	// No context node and no context item at all: a relative path is
	// XPDY0002 too.
	p, _ := Parse("child::a")
	if _, err := p.Eval(&Context{}); err == nil || !strings.Contains(err.Error(), "XPDY0002") {
		t.Errorf("child::a with no context: %v, want XPDY0002", err)
	}
}

// TestPeStaticPrefixCheck: an unbound variable-name prefix is XPST0081
// even on a branch that is never evaluated (§2.4.1: static errors are
// detected regardless of evaluation order).
func TestPeStaticPrefixCheck(t *testing.T) {
	for _, ns := range []NamespaceResolver{nil, peNS{"p": "urn:p"}} {
		for _, expr := range []string{
			"$q:x",
			"if (true()) then 1 else $q:x",
		} {
			_, err := peRun(expr, peDoc, ns)
			if err == nil || !strings.Contains(err.Error(), "XPST0081") {
				t.Errorf("%s (ns %v): %v, want XPST0081", expr, ns, err)
			}
		}
	}
	// A bound prefix and a braced EQName are fine.
	peCheck(t, peDoc, peNS{"p": "urn:p"}, [][2]string{
		{"let $p:x := 5 return $p:x", "5"},
		{"let $Q{urn:z}x := 6 return $Q{urn:z}x", "6"},
	})
}

// TestPeOperatorsAndConstructors covers ||, treat, map/array constructors,
// dynamic calls and lookups.
func TestPeOperatorsAndConstructors(t *testing.T) {
	peCheck(t, peDoc, nil, [][2]string{
		// §3.7.1: each || operand is atomized to xs:string? — two items is
		// a type error, () is "".
		{"(1, 2) || 'a'", "err:XPTY0004"},
		{"() || 'a' || ()", "a"},
		{"'a' || error()", "err:FOER0000"},
		{"error() || 'a'", "err:FOER0000"},
		// §3.14.5 treat.
		{"1 treat as xs:string", "err:XPDY0050"},
		{"(1 treat as xs:integer) + 1", "2"},
		{"count(() treat as empty-sequence())", "0"},
		{"error() treat as item()", "err:FOER0000"},
		{"error() instance of item()", "err:FOER0000"},
		// §3.11.1.1 map constructor: a key must be one atomic value, and two
		// keys that are the same key (op:same-key) are XQDY0137.
		{"map{(1, 2): 1}", "err:XPTY0004"},
		{"map{(): 1}", "err:XPTY0004"},
		{"map{1: 1, 1: 2}", "err:XQDY0137"},
		{"map{1: 1, 1.0: 2}", "err:XQDY0137"},
		{"map{'1': 1, 1: 2}(1)", "2"},
		{"map{error(): 1}", "err:FOER0000"},
		{"map{1: error()}", "err:FOER0000"},
		{"[error()]", "err:FOER0000"},
		{"array{error()}", "err:FOER0000"},
		// §3.1.5.1 dynamic function calls.
		{"(upper-case#1, lower-case#1)('a')", "err:XPTY0004"},
		{"()('a')", "err:XPTY0004"},
		{"upper-case#1('a', 'b')", "err:XPTY0004"},
		{"1(2)", "err:XPTY0004"},
		{"[1, 2](3)", "err:FOAY0001"},
		{"[1, 2](0)", "err:FOAY0001"},
		{"[1, 2](1, 2)", "err:XPTY0004"},
		{"[1, 2](2)", "2"},
		{"map{1: 2}(1)", "2"},
		{"count(map{1: 2}(3))", "0"},
		{"map{1: 2}((1, 2))", "err:XPTY0004"},
		{"map{1: 2}(1, 2)", "err:XPTY0004"},
		{"error#0()", "err:FOER0000"},
		{"upper-case#1(error())", "err:FOER0000"},
		// Partial application (§3.1.5.1) of a dynamic call.
		{"substring#2(?, 2)('abcd')", "bcd"},
		{"substring#3(?, 2, ?)('abcd', 1)", "b"},
		{"upper-case#1(?)('x')", "X"},
		{"function-arity(substring#3(?, 2, ?))", "2"},
		{"[10, 20](?)(2)", "20"},
		{"map{1: 'a'}(?)(1)", "a"},
		{"concat#2(?, error())", "err:FOER0000"},
		// Function literals naming a function that does not exist at that
		// arity are XPST0017 (§3.1.6).
		{"upper-case#3", "err:XPST0017"},
		{"fn:no-such-function#0", "err:XPST0017"},
		{"string(math:pi#0())", "3.141592653589793"},
		// §3.11.3 lookup errors.
		{"[1]?(xs:double(1))", "err:XPTY0004"},
		{"[1, 2]?3", "err:FOAY0001"},
		{"[1]?0", "err:FOAY0001"},
		{"1?a", "err:XPTY0004"},
		{"(/r)?a", "err:XPTY0004"},
		{"count(map{'a': 1}?b)", "0"},
		{"sum((map{'a': 1}, map{'a': 2})?a)", "3"},
		{"[1]?(error())", "err:FOER0000"},
		{"error()?a", "err:FOER0000"},
	})
}

// TestPeBindingsAndFunctions: for/some/every with several clauses, inline
// function signature checks (§3.1.7, function conversion rules §3.1.5.2).
func TestPeBindingsAndFunctions(t *testing.T) {
	peCheck(t, peDoc, nil, [][2]string{
		{"string-join(for $x in (1, 2), $y in ($x, 10) return string($y), ',')", "1,10,2,10"},
		{"for $x in (1, error()) return $x", "err:FOER0000"},
		{"for $x in 1 return error()", "err:FOER0000"},
		{"for $x in 1, $y in error() return 1", "err:FOER0000"},
		{"some $x in (1, 2), $y in (2, 3) satisfies $x = $y", "true"},
		{"every $x in (1, 2), $y in (3, 4) satisfies $x < $y", "true"},
		{"every $x in (1, 2), $y in (2, 4) satisfies $x < $y", "false"},
		{"some $x in () satisfies error()", "false"},
		{"every $x in () satisfies error()", "true"},
		{"some $x in error() satisfies 1", "err:FOER0000"},
		{"some $x in 1, $y in error() satisfies 1", "err:FOER0000"},
		{"every $x in (1, 2) satisfies error()", "err:FOER0000"},
		{"let $x := error() return 1 + $x", "err:FOER0000"},
		{"if (error()) then 1 else 2", "err:FOER0000"},
		{"-error()", "err:FOER0000"},
		{"error() * 2", "err:FOER0000"},
		{"2 * error()", "err:FOER0000"},
		{"error() = 1", "err:FOER0000"},
		{"1 eq error()", "err:FOER0000"},
		{"(1, error())", "err:FOER0000"},
		{"'a' => error()", "err:FOER0000"},
		{"error() => string()", "err:FOER0000"},
		// Arguments not matching a declared parameter type, or a result not
		// matching the declared return type, are XPTY0004.
		{"function($a as xs:integer) {$a}('x')", "err:XPTY0004"},
		{"function($a as xs:integer) {$a}(1.5)", "err:XPTY0004"},
		{"function($a as xs:integer) {$a}((1, 2))", "err:XPTY0004"},
		{"function() as xs:integer {'x'}()", "err:XPTY0004"},
		{"function() as xs:integer {(1, 2)}()", "err:XPTY0004"},
		{"function() as xs:integer {error()}()", "err:FOER0000"},
		// untypedAtomic arguments are cast to the declared atomic type.
		{"function($a as xs:integer) {$a + 1}(xs:untypedAtomic('4'))", "5"},
		// Closures capture their defining variables.
		{"let $k := 10, $f := function($a) {$a + $k} return $f(1)", "11"},
	})
}

// TestPeCast: §3.14.2 cast / castable.
func TestPeCast(t *testing.T) {
	peCheck(t, peDoc, peNS{"p": "urn:p"}, [][2]string{
		// The operand is atomized; () needs the '?' occurrence indicator,
		// more than one item is a type error (castable: false).
		{"() cast as xs:integer", "err:XPTY0004"},
		{"count(() cast as xs:integer?)", "0"},
		{"(1, 2) cast as xs:integer", "err:XPTY0004"},
		{"() castable as xs:integer", "false"},
		{"() castable as xs:integer?", "true"},
		{"(1, 2) castable as xs:integer", "false"},
		{"(1, 2) castable as xs:integer?", "false"},
		{"'x' castable as xs:integer", "false"},
		{"error() cast as xs:integer", "err:FOER0000"},
		{"error() castable as xs:integer", "err:FOER0000"},
		// xs:NOTATION and xs:anyAtomicType are not valid cast targets.
		{"1 cast as xs:NOTATION", "err:XPST0080"},
		{"1 cast as xs:anyAtomicType", "err:XPST0080"},
		{"1 castable as xs:NOTATION", "err:XPST0080"},
		// Casting to a list type tokenizes the string (§3.14.2, XSD list
		// types); xs:NMTOKENS has minLength 1.
		{"count('a b c' cast as xs:NMTOKENS)", "3"},
		{"'a b' castable as xs:NMTOKENS", "true"},
		{"'' castable as xs:NMTOKENS", "false"},
		{"() castable as xs:NMTOKENS", "false"},
		{"(1, 2) castable as xs:NMTOKENS", "false"},
		// F&O 3.1 §19.2: string/untypedAtomic -> xs:QName resolves the
		// prefix against the statically known namespaces.
		{"namespace-uri-from-QName('xs:integer' cast as xs:QName)", "http://www.w3.org/2001/XMLSchema"},
		{"namespace-uri-from-QName(xs:untypedAtomic('p:a') cast as xs:QName)", "urn:p"},
		{"local-name-from-QName(xs:untypedAtomic(' p:a ') cast as xs:QName)", "a"},
		{"namespace-uri-from-QName(xs:untypedAtomic('xs:a') cast as xs:QName)", "http://www.w3.org/2001/XMLSchema"},
		{"xs:untypedAtomic('q:a') cast as xs:QName", "err:FONS0004"},
		{"xs:untypedAtomic('1a') cast as xs:QName", "err:FORG0001"},
		{"xs:untypedAtomic('a:b:c') cast as xs:QName", "err:FORG0001"},
	})
	// Without any namespace resolver, an unknown prefix cannot be bound.
	peCheck(t, peDoc, nil, [][2]string{
		{"xs:untypedAtomic('q:a') cast as xs:QName", "err:FONS0004"},
	})
	// An unprefixed name takes the default element/type namespace.
	p, err := Parse("namespace-uri-from-QName(xs:untypedAtomic('a') cast as xs:QName)")
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := xmltree.Parse("<r/>")
	v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1, DefaultElemNS: "urn:d"})
	if err != nil || ToString(v) != "urn:d" {
		t.Errorf("unprefixed QName cast: %q, %v, want urn:d", ToString(v), err)
	}
}

// TestPeMathExpEdges: math:exp's special values (F&O 3.1 §4.8.4): NaN stays
// NaN, +INF gives +INF, -INF gives 0; a result beyond the double range
// overflows to INF or underflows to 0.
func TestPeMathExpEdges(t *testing.T) {
	peCheck(t, "<r/>", nil, [][2]string{
		{"string(math:exp(xs:double('NaN')))", "NaN"},
		{"string(math:exp(xs:double('INF')))", "INF"},
		{"string(math:exp(xs:double('-INF')))", "0"},
		{"string(math:exp(1000))", "INF"},
		{"string(math:exp(-1000))", "0"},
		{"math:exp(0) = 1", "true"},
		// For |x| < 2^-28 the correctly rounded result is 1 + x.
		{"math:exp(1e-10) = 1 + 1e-10", "true"},
		{"math:exp(-1e-10) = 1 - 1e-10", "true"},
		// exp(-1) * exp(1) = 1 within a couple of ulps.
		{"abs(math:exp(-1) * math:exp(1) - 1) < 1e-15", "true"},
		{"abs(math:exp(-2.5) - 0.0820849986238988) < 1e-15", "true"},
	})
}

// TestPeTypeNamesAndArity: an unprefixed atomic type name is not in the XSD
// namespace when the default type namespace is none, so it names no known
// type (XPST0051, §2.5.4); a partial application must supply exactly the
// function's arity (XPTY0004, §3.1.5.1).
func TestPeTypeNamesAndArity(t *testing.T) {
	peCheck(t, peDoc, nil, [][2]string{
		{"1 instance of integer", "err:XPST0051"},
		{"1 treat as integer", "err:XPST0051"},
		{"function($a as integer) {$a}", "err:XPST0051"},
		{"function() as integer {1}", "err:XPST0051"},
		{"upper-case#1(?, ?)", "err:XPTY0004"},
		{"substring#2(?)", "err:XPTY0004"},
	})
	// With XSD as the default namespace (an NS resolver binding the empty
	// prefix to it) the same names resolve.
	peCheck(t, peDoc, peNS{"": "http://www.w3.org/2001/XMLSchema"}, [][2]string{
		{"1 instance of integer", "true"},
	})
}

// TestPeInterrupt: a host's Interrupt hook stops a long evaluation with the
// hook's error.
func TestPeInterrupt(t *testing.T) {
	stop := errors.New("stopped")
	for _, expr := range []string{
		"for $x in 1 to 100 return $x",
		"some $x in 1 to 100 satisfies $x = 0",
		"(1 to 100)[. = 0]",
		"(1 to 100) ! (. + 1)",
		"let $f := function($n) {$n} return $f(1)",
		"/r/a[. = 'x']",
	} {
		p, err := Parse(expr)
		if err != nil {
			t.Fatal(err)
		}
		doc, _ := xmltree.Parse(peDoc)
		_, err = p.Eval(&Context{Node: doc, Pos: 1, Size: 1, Interrupt: func() error { return stop }})
		if !errors.Is(err, stop) {
			t.Errorf("%s: %v, want the interrupt error", expr, err)
		}
	}
}
