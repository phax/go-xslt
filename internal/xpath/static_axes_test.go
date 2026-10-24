package xpath

import (
	"strings"
	"testing"
)

// spParseST parses expr with the XPath 3.1 Static Typing Feature enabled and
// reports whether it was rejected with err:XPST0005 (XPath 3.1 §2.4.1 /
// §2.3.1: "it is a static error if the static type assigned to an expression
// other than () or data(()) is empty-sequence()").
func spParseST(t *testing.T, expr string) bool {
	t.Helper()
	_, err := ParseStaticTyping(expr)
	if err == nil {
		return false
	}
	if !strings.Contains(err.Error(), "XPST0005") {
		t.Fatalf("%s: unexpected error %v", expr, err)
	}
	return true
}

// TestSpStaticAxesEmpty lists axis steps whose static type is
// empty-sequence() by the XDM's structural rules alone (XDM 3.1 §6: document
// nodes have no parent and no attributes; attribute, text, comment, PI and
// namespace nodes have no children; a node has exactly one kind and name).
func TestSpStaticAxesEmpty(t *testing.T) {
	cases := []string{
		`/attribute::a`,         // a document node has no attributes
		`/..`,                   // a document node has no parent
		`/self::a`,              // document-node() is not element(a)
		`self::a/self::b`,       // an element named a is not named b
		`@a/child::b`,           // attributes have no children
		`@a/descendant::node()`, // ... nor descendants
		`text()/child::node()`,  // text nodes are leaves
		`comment()/b`,           // comments are leaves
		`processing-instruction()/b`,
		`text()/descendant-or-self::b`, // leaf, and text is not element(b)
		`@a/descendant-or-self::b`,     // attribute: no children; b is an element test
		`@a/@b`,                        // attributes carry no attributes
		`text()/@b`,
		`parse-xml-fragment('<a/>')/..`, // F&O §14.6.2: the result has no parent
		`/a/parse-xml-fragment('<a/>')/..`,
		`b/self::element(b)/self::c`,
		`document-node()/self::text()`,
		`attribute(a)/b`,
	}
	for _, c := range cases {
		if !spParseST(t, c) {
			t.Errorf("%s: want XPST0005", c)
		}
	}
}

// TestSpStaticAxesNotEmpty: shapes the structural analysis must NOT reject,
// since each can select something on some document.
func TestSpStaticAxesNotEmpty(t *testing.T) {
	cases := []string{
		`/a`, `//a`, `a/b/@c`, `self::node()/a`, `self::a/self::a`,
		`self::a/self::*`, `@a/..`, `text()/..`, `a/ancestor::b`,
		`@a/following-sibling::b`, `a/preceding::b`, `a/namespace::*`,
		`a/descendant-or-self::node()`, `a/descendant-or-self::a`,
		`text()/descendant-or-self::text()`, `@a/self::attribute()`,
		`self::a/descendant-or-self::b`, `//text()`, `/node()`,
		`/descendant-or-self::node()/@a`, `p:a/self::b`, `*:a/self::b`,
		`a/self::element()`, `a/element(x, xs:string)/c`, `$x/..`,
		`(a)/..`, `a/string()`, `a/@b/self::attribute(b)`,
		`a/descendant-or-self::node()/self::a`,
		`()`, `1 + 2`,
	}
	for _, c := range cases {
		if spParseST(t, c) {
			t.Errorf("%s: unexpected XPST0005", c)
		}
	}
	// Ordinary Parse never performs the check.
	if _, err := Parse(`/..`); err != nil {
		t.Errorf("Parse(/..) = %v; the static check must be opt-in", err)
	}
}

// TestSpStaticAxesNested: the check reaches every expression that can contain
// a path, so a provably-empty step nested anywhere is still found.
func TestSpStaticAxesNested(t *testing.T) {
	const e = `@a/b` // attributes have no children
	wraps := []string{
		`count(%s)`, `1 + count(%s)`, `-count(%s)`, `x | %s`, `%s | x`,
		`x[%s]`, `x/y[%s]`, `(x)[%s]`, `(%s)[1]`, `(1, %s)`, `1 to count(%s)`,
		`count(%s) to 1`, `for $v in %s return 1`, `for $v in 1 return %s`,
		`let $v := %s return 1`, `let $v := 1 return %s`,
		`some $v in %s satisfies true()`, `every $v in 1 satisfies %s`,
		`if (%s) then 1 else 2`, `if (1) then %s else 2`, `if (1) then 2 else %s`,
		`map{(%s): 1}`, `map{1: %s}`, `[%s]`, `array{%s}`, `function() {%s}`,
		`(%s)?1`, `map{}?(%s)`, `$f(%s)`, `(%s)(1)`, `%s = 1`, `1 eq count(%s)`,
		`'a' || %s`, `x intersect %s`, `%s except x`, `%s instance of node()`,
		`%s treat as node()*`, `%s cast as xs:string?`, `%s => count()`,
		`1 => f(%s)`, `x ! %s`, `x/(%s)`,
	}
	for _, w := range wraps {
		expr := strings.ReplaceAll(w, "%s", e)
		if !spParseST(t, expr) {
			t.Errorf("%s: want XPST0005 from the nested %s", expr, e)
		}
	}
}
