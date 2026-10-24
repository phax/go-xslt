package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// The F&O 3.1 §14.5.2 fn:path example document (the parse-xml form).
const ndPathDoc = `<?xml version="1.0"?><p xmlns="http://example.com/one" xml:lang="de" author="Friedrich von Schiller">Freude, schöner Götterfunken,<br/>Tochter aus Elysium,<br/>Wir betreten feuertrunken,<br/>Himmlische, dein Heiligtum.</p>`

// TestNodeSpecPath covers fn:path with the REC's examples plus the step
// forms for comments, processing instructions and namespace nodes.
func TestNodeSpecPath(t *testing.T) {
	coreRun(t, ndPathDoc, nil, []coreCase{
		{expr: `path(/)`, want: "/"},
		{expr: `path(/*:p)`, want: "/Q{http://example.com/one}p[1]"},
		{expr: `path(/*:p/@xml:lang)`, want: "/Q{http://example.com/one}p[1]/@Q{http://www.w3.org/XML/1998/namespace}lang"},
		{expr: `path(/*:p/@author)`, want: "/Q{http://example.com/one}p[1]/@author"},
		{expr: `path(/*:p/*:br[2])`, want: "/Q{http://example.com/one}p[1]/Q{http://example.com/one}br[2]"},
		{expr: `path(//text()[starts-with(normalize-space(), 'Tochter')])`, want: "/Q{http://example.com/one}p[1]/text()[2]"},
		{expr: `path(())`, want: "()"},
		// the context item is the default argument
		{expr: `/*:p/*:br[1]/path()`, want: "/Q{http://example.com/one}p[1]/Q{http://example.com/one}br[1]"},
	})
	coreRun(t, `<r xmlns:q="urn:q"><!--c1--><?pi a?><x/><!--c2--><?pi b?><?other?><y xmlns="urn:d"/></r>`, nil, []coreCase{
		{expr: `path(/r/comment()[2])`, want: "/Q{}r[1]/comment()[2]"},
		{expr: `path(/r/processing-instruction('pi')[2])`, want: "/Q{}r[1]/processing-instruction(pi)[2]"},
		{expr: `path(/r/processing-instruction('other'))`, want: "/Q{}r[1]/processing-instruction(other)[1]"},
		{expr: `path(/r/namespace::q)`, want: "/Q{}r[1]/namespace::q"},
		{expr: `path(/r/*:y/namespace::*[local-name() = ''])`, want: `/Q{}r[1]/Q{urn:d}y[1]/namespace::*[Q{http://www.w3.org/2005/xpath-functions}local-name()=""]`},
	})
}

// TestNodeSpecPathParentless covers fn:path of a tree whose root is not a
// document node: the path starts with a call to fn:root() (F&O 3.1 §14.5.2,
// the $emp example).
func TestNodeSpecPathParentless(t *testing.T) {
	doc, err := xmltree.Parse(`<employee xml:id="ID21256"><empnr>E21543</empnr></employee>`)
	if err != nil {
		t.Fatal(err)
	}
	emp := xmltree.RootElement(doc)
	emp.Parent = nil
	const root = "Q{http://www.w3.org/2005/xpath-functions}root()"
	for _, c := range []struct{ expr, want string }{
		{`path(.)`, root},
		{`path(@xml:id)`, root + "/@Q{http://www.w3.org/XML/1998/namespace}id"},
		{`path(empnr)`, root + "/Q{}empnr[1]"},
	} {
		p, err := Parse(c.expr)
		if err != nil {
			t.Fatal(err)
		}
		v, err := p.Eval(&Context{Node: emp, Pos: 1, Size: 1})
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got := ToString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

// TestNodeSpecInnermostOutermost covers fn:has-children, fn:innermost and
// fn:outermost (F&O 3.1 §14.5.4-§14.5.6).
func TestNodeSpecInnermostOutermost(t *testing.T) {
	const doc = `<r><a><b/></a><c/></r>`
	coreRun(t, doc, nil, []coreCase{
		// has-children($node) = exists($node/child::node())
		{expr: `has-children(/r)`, want: "true"},
		{expr: `has-children(/r/c)`, want: "false"},
		{expr: `has-children(/)`, want: "true"},
		{expr: `has-children(())`, want: "false"},
		{expr: `/r/a/has-children()`, want: "true"},
		{expr: `/r/a/b/has-children()`, want: "false"},
		{expr: `has-children(1)`, err: "XPTY0004"},
		{expr: `has-children(/r/*)`, err: "XPTY0004"},

		// innermost: drop every node that is an ancestor of another input node
		{expr: `innermost(//*) ! name()`, want: "b|c"},
		{expr: `innermost((//b, //a, //b)) ! name()`, want: "b"},
		{expr: `innermost(()) ! name()`, want: "()"},
		{expr: `innermost((/r, 1))`, err: "XPTY0004"},
		// outermost: drop every node that has an ancestor in the input
		{expr: `outermost(//*) ! name()`, want: "r"},
		{expr: `outermost((//c, //b, //a)) ! name()`, want: "a|c"},
		{expr: `outermost(()) ! name()`, want: "()"},
		{expr: `outermost(('x'))`, err: "XPTY0004"},
	})
	// no focus: XPDY0002
	p, _ := Parse(`has-children()`)
	if _, err := p.Eval(&Context{NoFocus: true}); err == nil || !strings.Contains(err.Error(), "XPDY0002") {
		t.Errorf("has-children() without focus: %v, want XPDY0002", err)
	}
}
