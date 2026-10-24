package xpath

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

type spVars map[string]Object

func (v spVars) ResolveVar(prefix, local string) (Object, bool) {
	o, ok := v[local]
	return o, ok
}

type spNS map[string]string

func (m spNS) ResolveNS(prefix string) (string, bool) { u, ok := m[prefix]; return u, ok }

const spPatDoc = `<r xmlns:p="urn:p"><a xml:id="a1"><b n="1"/><c/><b n="2"/><b n="3"/></a>` +
	`<a xml:id="a2"><b n="4"/>t<?pi x?><!--k--></a><p:q/></r>`

// spSelect returns the single node expr selects from doc.
func spSelect(t *testing.T, doc *xmltree.Node, expr string) *xmltree.Node {
	t.Helper()
	p, err := Parse(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1, NS: spNS{"p": "urn:p"}})
	if err != nil {
		t.Fatalf("eval %q: %v", expr, err)
	}
	items := Items(v)
	if len(items) != 1 {
		t.Fatalf("%q selects %d items, want 1", expr, len(items))
	}
	n, ok := items[0].(*xmltree.Node)
	if !ok {
		t.Fatalf("%q selects a non-node", expr)
	}
	return n
}

// TestSpPatternMatch checks XSLT 3.0 §5.5.3 "The Meaning of a Pattern": a
// node N matches pattern P if N is in the result of evaluating P (as an
// expression, with the pattern-specific axis rules) relative to some
// ancestor-or-self of N.
func TestSpPatternMatch(t *testing.T) {
	doc, err := xmltree.Parse(spPatDoc)
	if err != nil {
		t.Fatal(err)
	}
	b1 := spSelect(t, doc, `/r/a[1]/b[1]`)
	vars := spVars{
		"v":  NodeSet{b1},
		"as": NodeSet{spSelect(t, doc, `/r/a[1]`), spSelect(t, doc, `/r/a[2]`)},
	}
	ctx := &Context{Node: doc, Pos: 1, Size: 1, Vars: vars, NS: spNS{"p": "urn:p"}}
	cases := []struct {
		pat, node string
		want      bool
	}{
		{`b`, `/r/a[1]/b[1]`, true},
		{`b`, `/r/a[1]/c`, false},
		{`a/b`, `/r/a[1]/b[1]`, true},
		{`r/b`, `/r/a[1]/b[1]`, false},
		{`/r`, `/r`, true},
		{`/r`, `/r/a[1]`, false},
		{`/r/a/b`, `/r/a[2]/b`, true},
		{`/`, `/`, true},
		{`/`, `/r`, false},
		{`//b`, `/r/a[2]/b`, true},
		{`r//b`, `/r/a[2]/b`, true},
		{`c//b`, `/r/a[2]/b`, false},
		{`b[2]`, `/r/a[1]/b[@n='2']`, true},
		{`b[2]`, `/r/a[1]/b[@n='3']`, false},
		{`b[@n='3']`, `/r/a[1]/b[@n='3']`, true},
		// Second predicate: position() among the survivors of the first.
		{`b[@n!='1'][2]`, `/r/a[1]/b[@n='3']`, true},
		{`b[@n!='1'][2]`, `/r/a[1]/b[@n='2']`, false},
		{`b[position() = last()]`, `/r/a[1]/b[@n='3']`, true},
		{`b[position() = last()]`, `/r/a[1]/b[@n='2']`, false},
		{`@n`, `/r/a[1]/b[1]/@n`, true},
		{`b`, `/r/a[1]/b[1]/@n`, false},
		{`b/@n`, `/r/a[1]/b[1]/@n`, true},
		{`attribute::element()`, `/r/a[1]/b[1]`, false},
		{`attribute::node()`, `/r/a[1]/b[1]/@n`, true},
		{`node()`, `/`, false},
		{`node()`, `/r/a[1]/b[1]/@n`, false},
		{`node()`, `/r/a[2]/text()`, true},
		{`text()`, `/r/a[2]/text()`, true},
		{`comment()`, `/r/a[2]/comment()`, true},
		{`processing-instruction('pi')`, `/r/a[2]/processing-instruction()`, true},
		{`processing-instruction('zz')`, `/r/a[2]/processing-instruction()`, false},
		{`p:q`, `/r/p:q`, true},
		{`p:*`, `/r/p:q`, true},
		{`*:q`, `/r/p:q`, true},
		{`q`, `/r/p:q`, false},
		{`document-node()`, `/`, true},
		{`child::document-node()`, `/`, false},
		{`document-node(element(r))`, `/`, true},
		{`document-node(element(a))`, `/`, false},
		{`.`, `/r/a[1]/b[1]/@n`, true},
		{`.[@n='2']`, `/r/a[1]/b[@n='2']`, true},
		{`.[@n='2']`, `/r/a[1]/b[@n='3']`, false},
		{`id('a2')`, `/r/a[2]`, true},
		{`id('a2')`, `/r/a[1]`, false},
		{`id('a2')/b`, `/r/a[2]/b`, true},
		{`id('a2')/b`, `/r/a[1]/b[1]`, false},
		{`id('a1')//b`, `/r/a[1]/b[2]`, true},
		{`id('a1')//b`, `/r/a[2]/b`, false},
		{`$v`, `/r/a[1]/b[1]`, true},
		{`$v`, `/r/a[1]/b[2]`, false},
		{`$as/b`, `/r/a[2]/b`, true},
		{`$as//b`, `/r/a[1]/b[3]`, true},
		{`$v//b`, `/r/a[1]/b[3]`, false},
		// Expression patterns, anchored at each ancestor-or-self.
		{`(b|c)[1]`, `/r/a[1]/b[1]`, true},
		{`(b|c)[1]`, `/r/a[1]/c`, false},
		{`(b|c)[1]`, `/r/a[2]/b`, true},
		{`a/(b|c)`, `/r/a[1]/c`, true},
		{`r/(b|c)`, `/r/a[1]/c`, false},
		{`a/descendant::b`, `/r/a[1]/b[1]`, true},
		{`a[1]/descendant::b[1]`, `/r/a[1]/b[2]`, false},
		{`b except b[@n='2']`, `/r/a[1]/b[@n='2']`, false},
		{`b except b[@n='2']`, `/r/a[1]/b[@n='1']`, true},
		{`b intersect b[@n='2']`, `/r/a[1]/b[@n='2']`, true},
		{`b intersect b[@n='2']`, `/r/a[1]/b[@n='1']`, false},
		{`c | b[@n='4']`, `/r/a[2]/b`, true},
		{`c | b[@n='4']`, `/r/a[1]/b[1]`, false},
		{`descendant::b`, `/r/a[1]/b[1]`, true},
		{`descendant-or-self::a`, `/r/a[1]`, true},
		{`self::b`, `/r/a[1]/b[1]`, true},
		{`self::b`, `/r/a[1]/c`, false},
	}
	for _, c := range cases {
		p, err := ParsePattern(c.pat)
		if err != nil {
			t.Errorf("ParsePattern(%q): %v", c.pat, err)
			continue
		}
		if p.Src() != c.pat {
			t.Errorf("Src() = %q, want %q", p.Src(), c.pat)
		}
		n := spSelect(t, doc, c.node)
		got, err := p.Match(n, ctx)
		if err != nil {
			t.Errorf("%q on %s: %v", c.pat, c.node, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q on %s = %v, want %v", c.pat, c.node, got, c.want)
		}
	}
}

// TestSpPatternMatchError: a dynamic error inside a path pattern's predicate
// propagates (XSLT 3.0 §5.5.4 leaves recovery to the processor only for
// expression patterns; here the predicate raises FOAR0001).
func TestSpPatternMatchError(t *testing.T) {
	doc, _ := xmltree.Parse(spPatDoc)
	p, err := ParsePattern(`b[1 idiv 0]`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Match(spSelect(t, doc, `/r/a[1]/b[1]`), &Context{Node: doc})
	if err == nil || !strings.Contains(err.Error(), "FOAR0001") {
		t.Errorf("want FOAR0001, got %v", err)
	}
}

// TestSpPatternSyntaxErrors: XPath expressions outside XSLT 3.0's Pattern
// grammar (§5.5.2) are static errors XTSE0340.
func TestSpPatternSyntaxErrors(t *testing.T) {
	coded := []string{
		`b | .[1]`,    // a PredicatePattern is never a union operand
		`(.[1])`,      // ... nor parenthesized
		`.[1] | .[2]`, //
		`a/'x'`,       // a literal as a step
		`/[1]`,        // an array constructor as a step
		`a/map{}`,     //
		`a/function(){1}`,
		`parent::a`, // reverse axes are not PatternAxis
		`ancestor::a`,
		`a/following-sibling::b`,
		`a/preceding::b`,
	}
	for _, c := range coded {
		_, err := ParsePattern(c)
		if err == nil || !strings.Contains(err.Error(), "XTSE0340") {
			t.Errorf("ParsePattern(%q) = %v, want XTSE0340", c, err)
		}
	}
	// Rejected, but this package reports these without the code (the XSLT
	// compiler supplies XTSE0340 for any pattern that fails to parse); also
	// plain syntax errors.
	for _, c := range []string{`2+2`, `'lit'`, `a = 1`, `for $x in a return $x`, `a b`, `a[`, `)`, `a/`} {
		if _, err := ParsePattern(c); err == nil {
			t.Errorf("ParsePattern(%q) succeeded", c)
		}
	}
}

// TestSpPatternCheckGrammar: §5.5.2's FunctionCallP — only doc, id,
// element-with-id, key and root, only as the LEADING primary, and with
// literal or variable-reference arguments.
func TestSpPatternCheckGrammar(t *testing.T) {
	ok := []string{
		`key('k', 42)//a`, `key('k', $v)/a`, `id('x')`, `fn:id('x')/a`,
		`element-with-id('x')`, `doc('x.xml')//a`, `root($v)/a`, `a[f(1 + 2)]`,
		`a | id('x')`,
	}
	for _, c := range ok {
		p, err := ParsePattern(c)
		if err != nil {
			t.Errorf("ParsePattern(%q): %v", c, err)
			continue
		}
		if err := p.CheckGrammar(); err != nil {
			t.Errorf("CheckGrammar(%q) = %v, want nil", c, err)
		}
	}
	bad := []string{
		`copy-of(.)//a`,     // not an OuterFunctionName
		`key('k', 40+2)//a`, // argument is not a literal or variable
		`/key('k', 42)//a`,  // a function call that is not the leading primary
		`x:f()//a`,          // a prefixed, non-fn function
		`count(a)`,
	}
	for _, c := range bad {
		p, err := ParsePattern(c)
		if err != nil {
			if strings.Contains(err.Error(), "XTSE0340") {
				continue
			}
			t.Errorf("ParsePattern(%q): %v", c, err)
			continue
		}
		if err := p.CheckGrammar(); err == nil || !strings.Contains(err.Error(), "XTSE0340") {
			t.Errorf("CheckGrammar(%q) = %v, want XTSE0340", c, err)
		}
	}
	var nilPat *Pattern
	if nilPat.CheckGrammar() != nil || nilPat.UsesFunction("x") || nilPat.FunctionCalls() != nil {
		t.Error("nil pattern must report nothing")
	}
}

// TestSpPatternDefaultPriority checks the table of XSLT 3.0 §6.5.
func TestSpPatternDefaultPriority(t *testing.T) {
	cases := []struct {
		pat  string
		want float64
	}{
		{`a`, 0}, {`@a`, 0}, {`child::a`, 0}, {`attribute::a`, 0}, {`p:a`, 0},
		{`processing-instruction('x')`, 0}, {`processing-instruction(x)`, 0},
		{`*`, -0.5}, {`@*`, -0.5}, {`p:*`, -0.25}, {`*:a`, -0.25}, {`@p:*`, -0.25},
		{`node()`, -0.5}, {`text()`, -0.5}, {`comment()`, -0.5},
		{`processing-instruction()`, -0.5}, {`namespace-node()`, -0.5},
		{`element()`, -0.5}, {`element(*)`, -0.5}, {`attribute()`, -0.5}, {`attribute(*)`, -0.5},
		{`element(a)`, 0}, {`element(*, xs:integer)`, 0}, {`attribute(a)`, 0},
		{`attribute(*, xs:integer)`, 0},
		{`element(a, xs:integer)`, 0.25}, {`attribute(a, xs:integer?)`, 0.25},
		{`document-node()`, -0.5}, {`document-node(element(a))`, 0},
		{`document-node(element(a, xs:integer))`, 0.25},
		{`document-node(element(*))`, -0.5},
		{`/`, -0.5},
		{`a/b`, 0.5}, {`/a`, 0.5}, {`//a`, 0.5}, {`a//b`, 0.5}, {`a[1]`, 0.5},
		{`@a[. = 1]`, 0.5}, {`id('x')`, 0.5}, {`id('x')/a`, 0.5}, {`$v`, 0.5},
		{`(a|b)[1]`, 0.5},
		{`.[1]`, 1}, {`.[1][2]`, 1},
		// "." is a PredicatePattern with an empty predicate list: -1.
		{`.`, -1},
		// An IntersectExceptExprP takes the priority of its first operand.
		{`a except b`, 0}, {`a intersect b`, 0}, {`* except b`, -0.5},
		{`p:* except b`, -0.25}, {`a/b except c`, 0.5}, {`@a except @b`, 0},
		{`a except b except c`, 0}, {`a intersect b except c`, 0},
		{`element(a, xs:integer) except b`, 0.25}, {`(a|b) except c`, 0.5},
		{`a[1] except b`, 0.5}, {`$v except a`, 0.5},
		// A union is a set of rules; DefaultPriority reports the highest.
		{`a | *`, 0}, {`(a | b)`, 0}, {`* | a/b`, 0.5}, {`* | node()`, -0.5},
	}
	for _, c := range cases {
		p, err := ParsePattern(c.pat)
		if err != nil {
			t.Errorf("ParsePattern(%q): %v", c.pat, err)
			continue
		}
		if got := p.DefaultPriority(); got != c.want {
			t.Errorf("DefaultPriority(%q) = %v, want %v", c.pat, got, c.want)
		}
	}
}

// TestSpPatternAlternatives: §6.5 splits a top-level union (also inside outer
// parentheses) into one rule per alternative, in order.
func TestSpPatternAlternatives(t *testing.T) {
	cases := []struct {
		src  string
		want []string
	}{
		{`a`, []string{`a`}},
		{`a | b`, []string{`a`, `b`}},
		{`(a | b) | c`, []string{`a`, `b`, `c`}},
		{`((a|b))`, []string{`a`, `b`}},
		{`a[b|c] | d`, []string{`a[b|c]`, `d`}},
		{`(a)[1] | (b)`, []string{`(a)[1]`, `b`}},
		{`(a) | (b)[1]`, []string{`a`, `(b)[1]`}},
		{`(:c:) a`, []string{`(:c:) a`}},
	}
	for _, c := range cases {
		if got := PatternAlternatives(c.src); !reflect.DeepEqual(got, c.want) {
			t.Errorf("PatternAlternatives(%q) = %q, want %q", c.src, got, c.want)
		}
	}
	// A leading comment is not a parenthesis: ".[1]" stays a valid
	// PredicatePattern after one.
	if _, err := ParsePattern(`(: note :) .[1]`); err != nil {
		t.Errorf("comment before a predicate pattern: %v", err)
	}
	if _, err := ParsePattern(`(: a (: nested :) :) .[1]`); err != nil {
		t.Errorf("nested comment before a predicate pattern: %v", err)
	}
}

// TestSpPatternMayMatchNamespace: a namespace node is never a child,
// attribute or descendant (XDM 3.1 §6.4), so patterns ending on those axes
// without a namespace-node() test cannot match one.
func TestSpPatternMayMatchNamespace(t *testing.T) {
	cases := []struct {
		pat  string
		want bool
	}{
		{`a`, false}, {`@a`, false}, {`//a`, false}, {`/`, false},
		{`a/b | @c`, false}, {`node()`, false}, {`descendant::x`, false},
		{`namespace::*`, true}, {`namespace-node()`, true}, {`self::node()`, true},
		{`.`, true}, {`$v`, true}, {`id('x')/a`, true}, {`(a|b)[1]`, true},
		{`a | namespace::p`, true},
	}
	for _, c := range cases {
		p, err := ParsePattern(c.pat)
		if err != nil {
			t.Errorf("ParsePattern(%q): %v", c.pat, err)
			continue
		}
		if got := p.MayMatchNamespaceNode(); got != c.want {
			t.Errorf("MayMatchNamespaceNode(%q) = %v, want %v", c.pat, got, c.want)
		}
	}
}

// TestSpPatternFunctionUse covers the call-walk behind XTSE1060/XTSE1070
// (current-group()/current-grouping-key() in a pattern) and XPST0017
// checking: a call is found wherever it is nested.
func TestSpPatternFunctionUse(t *testing.T) {
	wraps := []string{
		`a[%s]`, `a[1 + %s]`, `a[-%s]`, `a[x | %s]`, `a[x/y[%s]]`, `a[(x)[%s]]`,
		`a[(1, %s)]`, `a[1 to %s]`, `a[for $v in %s return 1]`, `a[for $v in 1 return %s]`,
		`a[let $v := %s return 1]`, `a[let $v := 1 return %s]`,
		`a[some $v in %s satisfies 1]`, `a[every $v in 1 satisfies %s]`,
		`a[if (%s) then 1 else 2]`, `a[if (1) then 2 else %s]`,
		`a[map{(%s): 1}]`, `a[map{1: %s}]`, `a[[%s]]`, `a[array{%s}]`,
		`a[function() {%s}]`, `a[(1)?(%s)]`, `a[$f(%s)]`, `a[(%s)(1)]`,
		`a[%s = 1]`, `a['x' || %s]`, `a[x intersect %s]`, `a[%s instance of item()]`,
		`a[%s treat as item()*]`, `a[%s cast as xs:string?]`, `a[%s => count()]`,
		`a[1 => f(%s)]`, `a[x ! %s]`, `a[x/(%s)]`, `a[count(%s)]`,
		`b | (a|c)[%s]`,
	}
	for _, w := range wraps {
		src := strings.ReplaceAll(w, "%s", "current-group()")
		p, err := ParsePattern(src)
		if err != nil {
			t.Errorf("ParsePattern(%q): %v", src, err)
			continue
		}
		if !p.UsesFunction("current-group") {
			t.Errorf("%q: current-group() not found", src)
		}
		if p.UsesFunction("current-grouping-key") {
			t.Errorf("%q: current-grouping-key() falsely found", src)
		}
	}
	p, err := ParsePattern(`a[f(1)][g()] | (b|c)[h(1, 2)]`)
	if err != nil {
		t.Fatal(err)
	}
	want := []PatternCall{{Local: "f", Arity: 1}, {Local: "g", Arity: 0}, {Local: "h", Arity: 2}}
	if got := p.FunctionCalls(); !reflect.DeepEqual(got, want) {
		t.Errorf("FunctionCalls = %+v, want %+v", got, want)
	}
}
