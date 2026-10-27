package xmltree

import "testing"

// Untyped is what the function-argument fast path asks: a parsed node is
// untyped, and so is one that only carries bookkeeping (an xml:base, a
// namespace barrier); any type information makes it typed.
func TestNodeUntyped(t *testing.T) {
	doc, err := Parse(`<r xmlns:p="urn:p" xml:id="x"><a>1</a></r>`)
	if err != nil {
		t.Fatal(err)
	}
	r := doc.Children[0]
	if !doc.Untyped() || !r.Untyped() || !r.Children[0].Untyped() {
		t.Error("parsed nodes must be untyped")
	}
	r.SetBase("urn:base")
	if !r.Untyped() {
		t.Error("a base URI is not type information")
	}
	for name, set := range map[string]func(*Node){
		"TypeAnno":   func(n *Node) { n.SetTypeAnno(3) },
		"SchemaType": func(n *Node) { n.SetSchemaType(&SchemaTypeName{Namespace: "urn:t", Local: "T"}) },
		"ValueType":  func(n *Node) { n.SetValueType(&SchemaTypeName{Namespace: "urn:t", Local: "T"}) },
		"Nilled":     func(n *Node) { n.SetNilled(true) },
		"ListTyped":  func(n *Node) { n.SetListTyped(true) },
		"RealItem":   func(n *Node) { n.SetRealItem(42) },
	} {
		a, err := Parse(`<a/>`)
		if err != nil {
			t.Fatal(err)
		}
		n := a.Children[0]
		set(n)
		if n.Untyped() {
			t.Errorf("%s set, still untyped", name)
		}
	}
}
