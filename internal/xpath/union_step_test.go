package xpath

import (
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestStepUnionMatchesGeneralUnion pins evalStepUnion (the one-slice fast
// path for unions of two simple child::/attribute:: steps, "@*|node()") to the
// general UnionExpr evaluation it replaces: both operands through evalExpr,
// concatenated, normalized. Node IDENTITY and order must agree, on a parsed
// tree and on constructed trees whose nodes carry no document order (all 0)
// or one copied in from elsewhere.
func TestStepUnionMatchesGeneralUnion(t *testing.T) {
	parsed, err := xmltree.Parse(`<r a="1" b="2"><x/>t<y c="3"/><!--c--><?p q?></r>`)
	if err != nil {
		t.Fatal(err)
	}
	built := xmltree.NewElement(xmltree.Name{Local: "r"})
	built.SetAttr(xmltree.Name{Local: "a"}, "1")
	built.SetAttr(xmltree.Name{Local: "b"}, "2")
	built.Append(xmltree.NewElement(xmltree.Name{Local: "x"}))
	built.Append(xmltree.NewText("t"))
	built.Append(xmltree.NewElement(xmltree.Name{Local: "y"}))
	// A constructed parent holding a node that still carries a parsed tree's
	// numbering: document order inside it is not the child-list order.
	mixed := xmltree.NewElement(xmltree.Name{Local: "m"})
	mixed.SetAttr(xmltree.Name{Local: "k"}, "v")
	mixed.Append(xmltree.NewText("first"))
	mixed.Append(parsed.Children[0].Children[1]) // "t", re-parented, keeps its order

	ctxNodes := []*xmltree.Node{parsed.Children[0], parsed, built, mixed, parsed.Children[0].Attrs[0]}
	exprs := []string{
		`@*|node()`, `node()|@*`, `@*|*`, `*|@*`, `@a|@b`, `@b|@a`, `node()|node()`,
		`@*|@*`, `text()|*`, `x|y`, `y|x`, `comment()|processing-instruction()`,
		`@*|text()`, `@nope|node()`,
	}
	for _, src := range exprs {
		p, err := Parse(src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		u, ok := p.root.(*UnionExpr)
		if !ok || unionStep(u.L) == nil || unionStep(u.R) == nil {
			t.Fatalf("%s: not a step union", src)
		}
		for i, cn := range ctxNodes {
			got, err := p.Eval(&Context{Node: cn, Pos: 1, Size: 1})
			if err != nil {
				t.Fatalf("%s on %d: %v", src, i, err)
			}
			want := generalUnion(t, u, &Context{Node: cn, Pos: 1, Size: 1})
			g, _ := ToNodeSet(got)
			if len(g) != len(want) {
				t.Errorf("%s on %d: got %d nodes, want %d", src, i, len(g), len(want))
				continue
			}
			for k := range g {
				if g[k] != want[k] {
					t.Errorf("%s on %d: node %d differs", src, i, k)
				}
			}
		}
	}
}

// generalUnion is the UnionExpr evaluation without the step fast path.
func generalUnion(t *testing.T, u *UnionExpr, ctx *Context) NodeSet {
	l, err := evalExpr(u.L, ctx)
	if err != nil {
		t.Fatal(err)
	}
	r, err := evalExpr(u.R, ctx)
	if err != nil {
		t.Fatal(err)
	}
	ls, _ := nodeSetOperand(l, "union")
	rs, _ := nodeSetOperand(r, "union")
	out := make(NodeSet, 0, len(ls)+len(rs))
	out = append(out, ls...)
	out = append(out, rs...)
	return out.normalize()
}
