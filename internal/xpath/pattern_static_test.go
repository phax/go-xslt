package xpath

import (
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestEvalStaticStepUnion holds the context-free evaluation of "L|R" step
// unions to Eval with a full Context, on every node of a namespaced document
// under two default element namespaces, and checks that it declines every
// other shape.
func TestEvalStaticStepUnion(t *testing.T) {
	doc, err := xmltree.Parse(`<r xmlns:p="urn:p" a="1" p:b="2"><order id="o1">t<p:order/><!--c--><?pi d?><x/></order><d:e xmlns:d="urn:d" k="v"><order/></d:e></r>`)
	if err != nil {
		t.Fatal(err)
	}
	var nodes []*xmltree.Node
	var walk func(n *xmltree.Node)
	walk = func(n *xmltree.Node) {
		nodes = append(nodes, n)
		nodes = append(nodes, n.Attrs...)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)
	handled := []string{"@*|node()", "node()|@*", "@*|*", "order|text()", "@id|comment()", "*|processing-instruction()", "@*|order"}
	declined := []string{"node()", "@*", "@*|p:order", "@*|node()[1]", "@*|element()", "@*|node()|text()", "@*|../x", "$v|node()"}
	for _, src := range declined {
		p, err := Parse(src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if _, ok := p.EvalStaticStepUnion(nodes[1], &Context{}); ok {
			t.Errorf("EvalStaticStepUnion(%q) handled it, want declined", src)
		}
	}
	for _, src := range handled {
		p, err := Parse(src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		for _, dns := range []string{"", "urn:d"} {
			for _, n := range nodes {
				scratch := &Context{DefaultElemNS: dns}
				got, ok := p.EvalStaticStepUnion(n, scratch)
				if !ok {
					t.Fatalf("EvalStaticStepUnion(%q) declined", src)
				}
				if scratch.Node != nil {
					t.Fatalf("scratch Node left set")
				}
				v, err := p.Eval(&Context{Node: n, Pos: 1, Size: 1, NS: staticNS{"p": "urn:p"}, DefaultElemNS: dns})
				if err != nil {
					t.Fatalf("Eval(%q): %v", src, err)
				}
				want, _ := ToNodeSet(v)
				if len(got) != len(want) {
					t.Fatalf("%q (default ns %q) on %q: %d nodes, Eval gives %d", src, dns, n.Name.Local, len(got), len(want))
				}
				for i := range got {
					if got[i] != want[i] {
						t.Fatalf("%q (default ns %q) on %q: node %d differs", src, dns, n.Name.Local, i)
					}
				}
			}
		}
	}
}

type staticNS map[string]string

func (m staticNS) ResolveNS(prefix string) (string, bool) { u, ok := m[prefix]; return u, ok }

// TestNeedsOnlyDefaultNS pins which patterns a host may match against a
// Context holding nothing but the default element namespace, and that for
// every one of them such a Context gives exactly the answers a full one does,
// on every node of a document with namespaces, attributes, text, comments and
// processing instructions, under two default element namespaces.
func TestNeedsOnlyDefaultNS(t *testing.T) {
	doc, err := xmltree.Parse(`<?xml version="1.0"?>
<r xmlns:p="urn:p" a="1" p:b="2"><order id="o1">t<p:order/><x:order xmlns:x="urn:x"/><!--c--><?pi d?></order><d:e xmlns:d="urn:d"><order/></d:e></r>`)
	if err != nil {
		t.Fatal(err)
	}
	var nodes []*xmltree.Node
	var walk func(n *xmltree.Node)
	walk = func(n *xmltree.Node) {
		nodes = append(nodes, n)
		nodes = append(nodes, n.Attrs...)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)

	static := []string{
		"node()", "@*|node()", "text()", "comment()", "processing-instruction()",
		"processing-instruction('pi')", "*", "@*", "order", "@id", "order/order",
		"r//order", "/", "/r", "//order", "*:order", "Q{urn:p}order", "Q{urn:p}*",
		"Q{}order", "child::order", "attribute::*", "descendant-or-self::node()/order",
		"order | text()", ".",
	}
	notStatic := []string{
		"p:order", "p:*", "@p:b", "order[1]", "order[@id]", "element()", "element(order)",
		"attribute(id)", "document-node()", "namespace-node()", ".[. = 't']",
		"$v", "key('k', 'v')", "id('o1')", "order | p:order", "(order|text())[1]",
	}
	for _, src := range notStatic {
		p, err := ParsePattern(src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if p.NeedsOnlyDefaultNS() {
			t.Errorf("NeedsOnlyDefaultNS(%q) = true, want false", src)
		}
	}
	for _, src := range static {
		p, err := ParsePattern(src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if !p.NeedsOnlyDefaultNS() {
			t.Errorf("NeedsOnlyDefaultNS(%q) = false, want true", src)
			continue
		}
		for _, dns := range []string{"", "urn:d"} {
			for _, n := range nodes {
				full := &Context{Node: n, Pos: 1, Size: 1, NS: staticNS{"p": "urn:p", "d": "urn:d"}, DefaultElemNS: dns}
				want, werr := p.Match(n, full)
				got, gerr := p.Match(n, &Context{DefaultElemNS: dns})
				if got != want || (werr == nil) != (gerr == nil) {
					t.Errorf("%q (default ns %q) on %v %q: minimal context says %v/%v, full %v/%v", src, dns, n.Kind, n.Name.Local, got, gerr, want, werr)
				}
			}
		}
	}
}
