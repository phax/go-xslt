package xpath

import (
	"strconv"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestSlashSlashFusionMatchesLiteralSteps evaluates each expression twice —
// once as parsed (with the //-fusion) and once with every Step.Fuse cleared so
// the two literal steps run — and requires identical results. The cases are
// the ones the fusion must NOT get wrong: positional and numeric predicates
// (which count per parent), last(), nested ancestors producing overlapping
// descendant sets, and non-fusable shapes.
func TestSlashSlashFusionMatchesLiteralSteps(t *testing.T) {
	doc, err := xmltree.Parse(`<r>
  <a id="1"><x>a1x1</x><x>a1x2</x><b><x>a1bx1</x><x>a1bx2</x><a id="2"><x>a2x1</x></a></b></a>
  <a id="3"><y/><x>a3x1</x></a>
  <x>rx</x>
</r>`)
	if err != nil {
		t.Fatal(err)
	}
	exprs := []string{
		`//x`, `count(//x)`, `//x[1]`, `//x[2]`, `//x[last()]`, `//x[position() = 1]`,
		`//x[position() > 1]`, `//a//x`, `//a//x[1]`, `//a//x[@id]`, `//a[@id = '1']//x`,
		`//a[b]//x`, `//a[.//x = 'a2x1']/@id`, `//*[x]`, `//*`, `count(//a//a)`,
		`//a/b//x[. = 'a1bx2']`, `//x[not(../y)]`, `//x[. = ../x[2]]`, `//a[x[2]]`,
		`//x[1][. = 'a1x1']`, `.//x`, `//a//*[1]`, `//a//x[count(../x)]`, `//a//x[2.0]`,
		`//text()`, `//a//text()[1]`, `//x[exists(../b)]`, `//a//b//x[1]`,
	}
	for _, src := range exprs {
		fused, err := Parse(src)
		if err != nil {
			t.Fatalf("%s: parse: %v", src, err)
		}
		plain, _ := Parse(src)
		clearFuse(plain.root)
		ctx := func() *Context { return &Context{Node: doc, Pos: 1, Size: 1} }
		got, e1 := fused.Eval(ctx())
		want, e2 := plain.Eval(ctx())
		if (e1 == nil) != (e2 == nil) {
			t.Errorf("%s: error mismatch fused=%v plain=%v", src, e1, e2)
			continue
		}
		if e1 != nil {
			continue
		}
		g, w := render(got), render(want)
		if g != w {
			t.Errorf("%s:\n fused=%s\n plain=%s", src, g, w)
		}
	}
}

func render(o Object) string {
	var parts []string
	for _, it := range Items(o) {
		if n, ok := it.(*xmltree.Node); ok {
			parts = append(parts, n.Name.Local+"#"+strconv.Itoa(n.Order())+":"+n.StringValue())
		} else {
			parts = append(parts, ToString(FromItems([]Item{it})))
		}
	}
	return strings.Join(parts, "|")
}

func clearFuse(e Expr) {
	walkSteps(e, func(s *Step) { s.Fuse = nil })
}

// walkSteps visits every Step of every PathExpr reachable from e (including
// inside predicates and function arguments).
func walkSteps(e Expr, f func(*Step)) {
	switch v := e.(type) {
	case *PathExpr:
		walkSteps(v.Start, f)
		for _, s := range v.Steps {
			f(s)
			for _, p := range s.Preds {
				walkSteps(p, f)
			}
			walkSteps(s.Postfix, f)
		}
	case *FilterExpr:
		walkSteps(v.Primary, f)
		for _, p := range v.Preds {
			walkSteps(p, f)
		}
	case *FuncCall:
		for _, a := range v.Args {
			walkSteps(a, f)
		}
	case *CompareExpr:
		walkSteps(v.L, f)
		walkSteps(v.R, f)
	case *BinaryExpr:
		walkSteps(v.L, f)
		walkSteps(v.R, f)
	case *UnionExpr:
		walkSteps(v.L, f)
		walkSteps(v.R, f)
	}
}

// TestSlashSlashFusionActuallyFuses pins which shapes are folded, so a future
// change to the analysis can't silently turn the optimization off (or on for a
// positional predicate).
func TestSlashSlashFusionActuallyFuses(t *testing.T) {
	cases := map[string]bool{
		`//x`: true, `//x[@id]`: true, `//x[. = 'a']`: true, `//x[not(y)]`: true, `a//x`: true,
		`//x[1]`: false, `//x[last()]`: false, `//x[position() = 1]`: false, `//x[count(y)]`: false,
		`//x[y[1]]`: true, `//@id`: false, `//x/y`: true, `//x[$n]`: false, `//text()`: true,
	}
	for src, want := range cases {
		p, err := Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		fused := false
		walkSteps(p.root, func(s *Step) {
			if s.Fuse != nil {
				fused = true
			}
		})
		if fused != want {
			t.Errorf("%s: fused=%v, want %v", src, fused, want)
		}
	}
}
