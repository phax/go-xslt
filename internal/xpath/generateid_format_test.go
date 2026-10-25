package xpath

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestGenerateIDFormat pins generate-id()'s spelling to the Sprintf form it
// was built with before: "d", "t%p" of the root, "n", the order (and "ns" and
// the prefix for a namespace node).
func TestGenerateIDFormat(t *testing.T) {
	doc, err := xmltree.Parse(`<a xmlns:p="u" x="1"><b>t</b><!--c--></a>`)
	if err != nil {
		t.Fatal(err)
	}
	var nodes []*xmltree.Node
	var walk func(n *xmltree.Node)
	walk = func(n *xmltree.Node) {
		nodes = append(nodes, n)
		nodes = append(nodes, n.Attrs...)
		nodes = append(nodes, n.NSDecls()...)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)
	nodes = append(nodes, xmltree.NewElement(xmltree.Name{Local: "loose"}))
	for _, n := range nodes {
		want := "d" + fmt.Sprintf("t%p", n.Root()) + "n" + strconv.Itoa(n.Order())
		if n.Kind == xmltree.KindNamespace {
			want += "ns" + n.Name.Local
		}
		got, err := fnGenerateID(&Context{}, []Object{NodeSet{n}})
		if err != nil {
			t.Fatal(err)
		}
		if s := ToString(got); s != want {
			t.Errorf("generate-id = %q, want %q", s, want)
		}
	}
}
