package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// peNS is a namespace resolver for the tests in this file and its siblings
// (eval_spec_test.go, var_refs_test.go).
type peNS map[string]string

func (m peNS) ResolveNS(prefix string) (string, bool) { u, ok := m[prefix]; return u, ok }

// peDoc is a small source document shared by the expression tables.
const peDoc = `<r xmlns:p="urn:p"><a x="1">one</a><a>two</a><p:b/><?t data?><!--c--></r>`

// peRun parses and evaluates expr with the document node of xml as the
// context item ("" = no source document: the focus is then absent).
func peRun(expr, xml string, ns NamespaceResolver) (Object, error) {
	p, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	ctx := &Context{Pos: 1, Size: 1, NS: ns}
	if xml != "" {
		doc, err := xmltree.Parse(xml)
		if err != nil {
			return nil, err
		}
		ctx.Node = doc
	}
	return p.Eval(ctx)
}

// peCheck evaluates each case and compares ToString of the result, or, when
// want starts with "err:", requires an error carrying that code.
func peCheck(t *testing.T, xml string, ns NamespaceResolver, cases [][2]string) {
	t.Helper()
	for _, c := range cases {
		expr, want := c[0], c[1]
		v, err := peRun(expr, xml, ns)
		if strings.HasPrefix(want, "err:") {
			if err == nil {
				t.Errorf("%s: got %q, want error %s", expr, ToString(v), want)
			} else if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: error %v, want %s", expr, err, want)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", expr, err)
			continue
		}
		if got := ToString(v); got != want {
			t.Errorf("%s = %q, want %q", expr, got, want)
		}
	}
}

// TestPeGrammarErrors: each input violates the XPath 3.1 grammar (Appendix
// A.1), so Parse must reject it. Hosts report a parse error without a code
// as XPST0003, so only the failure itself is asserted here.
func TestPeGrammarErrors(t *testing.T) {
	bad := []string{
		// for / let / quantified (§3.12, §3.13)
		"for $x in (1, 2)",
		"for $x (1) return 1",
		"for $ in 1 return 1",
		"for $x in return 1",
		"for $x in 1 return",
		"let $x = 1 return $x",
		"let $x := 1",
		"let $x := return 1",
		"let $x := 1 return",
		"some $x in (1, 2)",
		"some $x in satisfies 1",
		"every $x in 1 satisfies",
		// if (§3.15)
		"if 1 then 2 else 3",
		"if (1 then 2 else 3",
		"if () then 1 else 2",
		"if (1) 2 else 3",
		"if (1) then 2",
		"if (1) then else 3",
		"if (1) then 2 else",
		// binary operators with a missing right operand
		"1 or", "1 and", "1 =", "1 >>", "'a' ||", "1 to", "2 *", "2 +", "a |",
		"a intersect", "a except", "1 ! ",
		// type expressions (§3.14)
		"1 instance xs:integer", "1 instance of", "1 treat xs:integer",
		"1 treat as", "1 castable xs:integer", "1 castable as",
		"1 cast xs:integer", "1 cast as",
		// arrow (§3.16): a specifier and an argument list are required
		"'a' => 2", "'a' => upper-case", "'a' => $", "'a' => (upper-case#1",
		"'a' => ()", "'a' => upper-case(",
		// paths and steps
		"/a/", "a//", "child::", "@", "a[1", "a[", "a[]", "a/b[1",
		"(1)[", "(1)[1",
		// function calls and primaries
		"f(1,", "f(1", "f(,)", "(1", "(1, 2", "$", "$1", ")", "concat#x",
		"concat#",
		// constructors (§3.11)
		"map{1 2}", "map{1:", "map{1:2", "map{1:2,}", "map{", "map{:1}",
		"[1,", "[1", "[,]", "array{1", "array{", "array{,}",
		"function($a $b) {1}", "function($a as) {1}", "function(a) {1}",
		"function($) {1}", "function($a) as {1}", "function($a) 1",
		"function($a) {1", "function($a {1}", "function($a) {,}",
		// kind tests
		"element(", "element(a", "attribute(a,", "1 + #",
		"document-node(", "document-node(text())", "processing-instruction(",
		"text(1)", "schema-element()",
		// lookups (§3.11.3): a key specifier is NCName | IntegerLiteral |
		// ParenthesizedExpr | "*"
		"map{}?'x'", "map{}?(1", "map{}?", "map{}?(,)",
		// lexical errors
		"'abc", `"abc`, "(: abc", "(: (: a :) b", "1 (: open",
	}
	for _, src := range bad {
		if p, err := Parse(src); err == nil {
			t.Errorf("Parse(%q) succeeded (%T), want a syntax error", src, p.root)
		}
	}
}

// TestPeStaticErrorCodes covers the parse-time errors whose codes the spec
// names explicitly.
func TestPeStaticErrorCodes(t *testing.T) {
	cases := []struct{ src, code string }{
		// §3.3.2.1: the axis names are a closed list; anything else is a
		// grammar error.
		{"foo::a", "XPST0003"},
		{"a/bogus::b", "XPST0003"},
		// PITest takes an NCName or a string literal (A.1 production
		// PITest); a QName is not an NCName.
		{"processing-instruction(a:b)", "XPST0003"},
		// DocumentTest ::= "document-node" "(" (ElementTest |
		// SchemaElementTest)? ")".
		{"document-node(attribute())", "XPST0003"},
		// A.2.2 terminal delimitation: a numeric literal must be followed
		// by a delimiter (xgc: whitespace / delimiting terminal symbols).
		{"12abc", "XPST0003"},
		{"1.2.3", "XPST0003"},
		{".5a", "XPST0003"},
		{"1e5x", "XPST0003"},
		// A name must begin with a NameStartChar (A.2.1, XML 1.0 Names).
		{"µ", "XPST0003"},
		// §3.11.3.1: NCName key / IntegerLiteral key.
		{"map{}?a:b", "XPST0003"},
		{"[1]?1.5", "XPST0003"},
		{"[1]?1e0", "XPST0003"},
		// §3.1.7: an inline function may not have two parameters with the
		// same name.
		{"function($a, $a) {1}", "XQST0039"},
		{"function($a, $Q{}a) {1}", "XQST0039"},
	}
	for _, c := range cases {
		_, err := Parse(c.src)
		if err == nil {
			t.Errorf("Parse(%q) succeeded, want %s", c.src, c.code)
			continue
		}
		if !strings.Contains(err.Error(), c.code) {
			t.Errorf("Parse(%q): %v, want %s", c.src, err, c.code)
		}
	}
	// A.3 reserved function names may not be used unprefixed in a
	// NamedFunctionRef (xgc: reserved-function-names).
	for _, src := range []string{"node#0", "if#1", "item#0", "switch#1", "map#1"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q) succeeded, want a static error", src)
		}
	}
}

// TestPeLexer exercises lexical rules of A.2: comments (nested), doubled
// quote escapes, numeric literal forms and two-character operators.
func TestPeLexer(t *testing.T) {
	peCheck(t, peDoc, peNS{"p": "urn:p"}, [][2]string{
		// A.2.4: comments nest.
		{"1 (: a (: b :) c :) + 1", "2"},
		{"(: lead :) 3", "3"},
		// §3.1.1: a doubled delimiter inside a literal is one quote.
		{"'it''s'", "it's"},
		{`"say ""hi"""`, `say "hi"`},
		{`'"'`, `"`},
		// Numeric literals (§3.1.1): .5 is a decimal, exponent forms are
		// doubles.
		{".5 instance of xs:decimal", "true"},
		{".5 + .25", "0.75"},
		{"1e+2 instance of xs:double", "true"},
		{"1E2 = 100", "true"},
		{"25e-1 = 2.5", "true"},
		{"1.5e1 = 15", "true"},
		// ':' after a numeric literal is the map entry separator.
		{"map{1:'x'}(1)", "x"},
		// Two-character operators.
		{"/r/a[2] >> /r/a[1]", "true"},
		{"/r/a[1] >> /r/a[2]", "false"},
		{"/r/a[1] << /r/a[2]", "true"},
		{"2 >= 2", "true"},
		{"1 <= 0", "false"},
		{"1 != 2", "true"},
		{"let $x := 1 return $x", "1"},
		// A prefix followed by '*' is a wildcard name test, not
		// multiplication, even after an operand-ending token.
		{"count(/r/*[self::p:* except self::p:c])", "1"},
		{"count(/r/p:*)", "1"},
		{"count(/r/*:b)", "1"},
		// '*' after an operand is multiplication.
		{"2*3", "6"},
		{"count(/r/*) * 2", "6"},
	})
}

// TestPeParsedForms checks expressions that exercise the less common
// grammar productions and their meaning.
func TestPeParsedForms(t *testing.T) {
	peCheck(t, peDoc, peNS{"p": "urn:p"}, [][2]string{
		// §3.16 arrow: the specifier may be an EQName, a variable reference
		// or a parenthesized expression.
		{"'a' => Q{http://www.w3.org/2005/xpath-functions}upper-case()", "A"},
		{"let $f := upper-case#1 return 'a' => $f()", "A"},
		{"'a' => (upper-case#1)()", "A"},
		{"'abc' => substring(2)", "bc"},
		// A placeholder in the arrow's argument list makes the result a
		// partial application (§3.1.5.1): E => f(?) is f(E, ?).
		{"('abcd' => substring(?))(3)", "cd"},
		{"(('abcd' => substring(?, ?))(2, 2))", "bc"},
		{"let $f := substring#3 return ('abcd' => $f(?, 1))(2)", "b"},
		{"'a' => concat('b') => concat('c')", "abc"},
		// §3.3.5: an AttributeTest makes the default axis attribute, a
		// NamespaceNodeTest makes it namespace.
		{"string(/r/a[1]/attribute())", "1"},
		{"count(/r/a/attribute(x))", "1"},
		{"count(/r/a/attribute(*))", "1"},
		{"count(/r/a/attribute(y))", "0"},
		{"count(/r/namespace-node())", "2"},
		{"string-join(for $n in /r/namespace-node() return name($n), ',') = ('p,xml', 'xml,p')", "true"},
		// '.' as a step is self::node() and takes predicates.
		{"count(/r/a/.[@x])", "1"},
		{"count(/r/a/.)", "2"},
		// Kind tests.
		{"count(/r/processing-instruction('t'))", "1"},
		{"count(/r/processing-instruction(t))", "1"},
		{"count(/r/processing-instruction(u))", "0"},
		{"count(/r/processing-instruction())", "1"},
		{"count(/r/comment())", "1"},
		{"count(/r/element(a))", "2"},
		{"count(/r/element(Q{urn:p}b))", "1"},
		{"count(/r/element(p:b))", "1"},
		{"count(/r/element(*))", "3"},
		{"(/) instance of document-node(element(r))", "true"},
		{"(/) instance of document-node(element(a))", "false"},
		{"(/) instance of document-node()", "true"},
		{"/r instance of element(r, xs:untyped?)", "true"},
		// §3.11.3 lookups.
		{"map{'a': 1} ! ?a", "1"},
		{"string-join(map{'a': 1, 'b': 2}?('a', 'b'), ',')", "1,2"},
		{"count(map{'a': 1}?())", "0"},
		{"sum(map{'a': 1, 'b': 2}?*)", "3"},
		{"[10, 20, 30]?2", "20"},
		{"string-join([10, 20, 30]?(1 to 2), ',')", "10,20"},
		{"string-join([10, 20]?*, ',')", "10,20"},
		{"[[1, 2], [3]]?1?2", "2"},
		{"[1, 2] ! ?2", "2"},
		{"[10, 20]?(xs:untypedAtomic('1'))", "10"},
		// Braced variable names (§3.1.2: an EQName).
		{"let $Q{urn:v}x := 1 return $Q{urn:v}x + 1", "2"},
		{"let $x := 2 return $Q{}x", "2"},
		{"let $p:x := 3 return $Q{urn:p}x", "3"},
		// Function literals (§3.1.6).
		{"Q{http://www.w3.org/2005/xpath-functions}upper-case#1('a')", "A"},
		{"fn:upper-case#1('b')", "B"},
		{"Q{http://www.w3.org/2005/xpath-functions}concat('a', 'b')", "ab"},
		// Inline functions (§3.1.7).
		{"function($a as xs:integer) as xs:integer {$a + 1}(2)", "3"},
		{"count(function() {}())", "0"},
		{"function($a, $b) {$a * $b}(3, 4)", "12"},
		{"function($a as xs:decimal) {$a}(1) instance of xs:integer", "true"},
		{"function($a as xs:double) {$a}(1) instance of xs:double", "true"},
		{"function() as xs:double {1}() instance of xs:double", "true"},
		// Parenthesized and empty sequences.
		{"count(())", "0"},
		{"((1), (2, 3))[3]", "3"},
		// Square and curly arrays.
		{"array:size([])", "0"},
		{"array:size([(), (1, 2)])", "2"},
		{"array:size(array{(), (1, 2)})", "2"},
		{"array:size(array{})", "0"},
		{"map:size(map{})", "0"},
	})
}

// TestPeParseStaticTypingEntry: ParseStaticTyping accepts an ordinary
// expression exactly like Parse.
func TestPeParseStaticTypingEntry(t *testing.T) {
	p, err := ParseStaticTyping("1 + 2")
	if err != nil {
		t.Fatal(err)
	}
	v, err := p.Eval(&Context{})
	if err != nil || ToString(v) != "3" {
		t.Fatalf("got %v, %v", v, err)
	}
	if _, err := ParseStaticTyping("1 +"); err == nil {
		t.Fatal("ParseStaticTyping accepted a syntax error")
	}
}
