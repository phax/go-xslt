package xmltree

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// parseEager is parseRaw as it was before the attribute whitespace pre-pass
// became lazy: normalizeAttrWhitespace over the whole source first.
func parseEager(src string) (*Node, error) {
	return parseDecoded(decodeBOM(src), "", nil, nil, true)
}

// dumpTree spells out everything a parse produces that a caller can see:
// every node's kind, name, value, position and document order, and an error's
// text and position.
func dumpTree(doc *Node, err error) string {
	if err != nil {
		if pe, ok := err.(*ParseError); ok {
			return fmt.Sprintf("error %d:%d %s", pe.Line, pe.Col, pe.Msg)
		}
		return "error " + err.Error()
	}
	var b strings.Builder
	var walk func(n *Node, depth int)
	walk = func(n *Node, depth int) {
		fmt.Fprintf(&b, "%*s%d %q %q %q %q %d:%d #%d id%d\n", depth, "", n.Kind, n.Name.Space, n.Name.Prefix, n.Name.Local, n.Value, n.Line, n.Col, n.order, n.IDKind())
		for _, ns := range n.NSDecls() {
			fmt.Fprintf(&b, "%*s ns %q %q #%d\n", depth, "", ns.Name.Local, ns.Value, ns.order)
		}
		for _, a := range n.Attrs {
			walk(a, depth+1)
		}
		for _, c := range n.Children {
			if c.Parent != n {
				fmt.Fprintf(&b, "BAD PARENT\n")
			}
			walk(c, depth+1)
		}
	}
	walk(doc, 0)
	return b.String()
}

func compareLazyEager(s string) error {
	if strings.Contains(s, "<!DOCTYPE") {
		// Parsed eagerly either way. (And DTD-defaulted attributes are added
		// in map order, so two parses of one such document may differ.)
		return nil
	}
	lazy := dumpTree(parseRaw(s, "", nil, nil))
	eager := dumpTree(parseEager(s))
	if lazy != eager {
		return fmt.Errorf("lazy and eager parses differ:\nlazy:\n%s\neager:\n%s", lazy, eager)
	}
	return nil
}

var lazyNormSeeds = []string{
	"<a b='x\ny'/>",
	"<a>\n<b c=\"1\"/>\n<b c=\"2\n3\" d='\t'/>\n<b c='4\r\n5'/></a>",
	"<a><b c='&#10;'/><b c='x\ny'/></a>",
	"<a c='&#9;&#13;'>text\n</a>",
	"<?xml version=\"1.1\"?>\n<a>\n<b c='x\ny'/></a>",
	"\xef\xbb\xbf<a>\n<b c='x\ny'/></a>",
	"<a><!x <b c='\n'>><c d='\n'/></a>",
	"<a><!x <b> <c d='\n'>>\n<e/></a>",
	"<a><!x [<b c='\n'>]>\n<c d='e\nf'/></a>",
	"<a>\n<!-- '\n' -->\n<?p q='\n'?>\n<![CDATA[<b c='\n'>]]><b c='\n'/></a>",
	"<a b='x\ny'>\n<c d=\"e\ny\n\"/></x>",
	"<a b='x\ny'>\n<c d='e\ny'></a>",
	"<a>\n<b c='x\ny\n'>\n<c d='e\n",
	"<a xmlns:p='u\nv'><p:b p:c='\n'/></a>",
	"<a b='1'>\n<c d='2'>\n</c>\n<e f='3\n4' g='>'>\n</e></a>\n<!-- after -->",
	"<a>\n<b c='x\ny'/>\n</a>\n<a2/>",
	"<a b='x\ny'",
	"<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<a b='x\ny'>\xe9</a>",
	"<?xml encoding=\"EUC-JP\"?><a>\xa4\x90\x93\xa2<!0>",
	"<a b='x\n<y'/>",
}

func TestLazyAttrNormMatchesEager(t *testing.T) {
	for _, s := range append(append([]string{}, lazyNormSeeds...), tokenizerSeeds...) {
		if err := compareLazyEager(s); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
}

// checkParseOrder reports whether the document order parseRaw stamped on the
// nodes as it made them is the numbering assignOrder's walk gives the tree.
func checkParseOrder(s string) error {
	doc, err := parseRaw(s, "", nil, nil)
	if err != nil {
		return nil
	}
	type mark struct {
		order int32
		tree  uint64
	}
	var got []mark
	var nodes []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		nodes = append(nodes, n)
		nodes = append(nodes, n.NSDecls()...)
		nodes = append(nodes, n.Attrs...)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)
	for _, n := range nodes {
		got = append(got, mark{n.order, n.tree})
	}
	assignOrder(doc)
	for i, n := range nodes {
		if got[i].order != n.order || got[i].tree == 0 || got[i].tree != got[0].tree {
			return fmt.Errorf("node %d (%v %q): stamped order %d tree %d, walk gives %d", i, n.Kind, n.Name.Local, got[i].order, got[i].tree, n.order)
		}
	}
	return nil
}

func TestParseOrderMatchesAssignOrder(t *testing.T) {
	for _, s := range append(append([]string{}, lazyNormSeeds...), tokenizerSeeds...) {
		if err := checkParseOrder(s); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
	if testing.Short() {
		return
	}
	n := 0
	for _, p := range suiteFiles(1 << 62) {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := checkParseOrder(string(b)); err != nil {
			if n++; n <= 10 {
				t.Errorf("%s: %v", p, err)
			}
		}
	}
}

// TestLazyAttrNormMatchesEagerOnSuites parses every W3C suite file both ways.
func TestLazyAttrNormMatchesEagerOnSuites(t *testing.T) {
	if testing.Short() {
		t.Skip("walks the W3C suites")
	}
	files := suiteFiles(1 << 62)
	if len(files) == 0 {
		t.Skip("W3C suites not present")
	}
	bad := 0
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := compareLazyEager(string(b)); err != nil {
			bad++
			if bad <= 10 {
				t.Errorf("%s: %v", p, err)
			}
		}
	}
	t.Logf("%d files compared, %d differ", len(files), bad)
}

// FuzzLazyAttrNorm compares the lazy attribute whitespace normalization with
// the eager one on arbitrary input.
//
//	go test ./internal/xmltree/ -run '^$' -fuzz FuzzLazyAttrNorm -fuzztime 10m
func FuzzLazyAttrNorm(f *testing.F) {
	for _, s := range lazyNormSeeds {
		f.Add(s)
	}
	for _, s := range tokenizerSeeds {
		if len(s) <= 1<<16 {
			f.Add(s)
		}
	}
	if os.Getenv("XMLTREE_FUZZ_SUITES") != "" {
		for _, p := range suiteFiles(256 << 10) {
			if b, err := os.ReadFile(p); err == nil {
				f.Add(string(b))
			}
		}
	}
	f.Fuzz(func(t *testing.T, s string) {
		if err := compareLazyEager(s); err != nil {
			t.Fatal(err)
		}
	})
}
