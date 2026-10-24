package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// analyzeRender renders an fn:analyze-string result compactly: an element as
// its local name (plus @nr for fn:group) with its children in brackets, a
// text node as its text.
func analyzeRender(n *xmltree.Node) string {
	if n.Kind == xmltree.KindText {
		return n.Value
	}
	var b strings.Builder
	b.WriteString(n.Name.Local)
	if v, ok := n.Attr("", "nr"); ok {
		b.WriteString(v)
	}
	b.WriteByte('[')
	for _, ch := range n.Children {
		b.WriteString(analyzeRender(ch))
	}
	b.WriteByte(']')
	return b.String()
}

// TestAnalyzeString covers fn:analyze-string (F&O 3.1 §5.6.6): the REC's
// three examples plus nested, non-capturing, non-participating and repeated
// groups.
func TestAnalyzeString(t *testing.T) {
	cases := []struct{ expr, want string }{
		{`analyze-string("The cat sat on the mat.", "\w+")`,
			"analyze-string-result[match[The]non-match[ ]match[cat]non-match[ ]match[sat]non-match[ ]match[on]non-match[ ]match[the]non-match[ ]match[mat]non-match[.]]"},
		{`analyze-string("2008-12-03", "^(\d+)\-(\d+)\-(\d+)$")`,
			"analyze-string-result[match[group1[2008]-group2[12]-group3[03]]]"},
		{`analyze-string("A1,C15,,D24, X50,", "([A-Z])([0-9]+)")`,
			"analyze-string-result[match[group1[A]group2[1]]non-match[,]match[group1[C]group2[15]]non-match[,,]match[group1[D]group2[24]]non-match[, ]match[group1[X]group2[50]]non-match[,]]"},
		// Nested groups nest in the result, numbered by opening parenthesis.
		{`analyze-string("abc", "(a(b))c")`, "analyze-string-result[match[group1[agroup2[b]]c]]"},
		// (?:...) does not capture; parentheses in a class or escaped do not either.
		{`analyze-string("ab", "(?:a)(b)")`, "analyze-string-result[match[agroup1[b]]]"},
		{`analyze-string("(a", "[(](a)")`, "analyze-string-result[match[(group1[a]]]"},
		{`analyze-string("(a", "\((a)")`, "analyze-string-result[match[(group1[a]]]"},
		// A group that does not participate is omitted.
		{`analyze-string("b", "(a)|(b)")`, "analyze-string-result[match[group2[b]]]"},
		// A repeated group reports its last iteration only.
		{`analyze-string("aab", "(a)+b")`, "analyze-string-result[match[agroup1[a]b]]"},
		// q flag: the pattern is literal, so nothing is a group.
		{`analyze-string("x(a)", "(a)", "q")`, "analyze-string-result[non-match[x]match[(a)]]"},
		{`analyze-string("xAx", "a", "i")`, "analyze-string-result[non-match[x]match[A]non-match[x]]"},
		{`analyze-string("xyz", "a")`, "analyze-string-result[non-match[xyz]]"},
		{`analyze-string((), "a")`, "analyze-string-result[]"},
		{`analyze-string("", "a")`, "analyze-string-result[]"},
	}
	doc, _ := xmltree.Parse("<r/>")
	for _, c := range cases {
		p, err := Parse(c.expr)
		if err != nil {
			t.Fatalf("parse %s: %v", c.expr, err)
		}
		v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1})
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		items := Items(v)
		if len(items) != 1 {
			t.Errorf("%s: %d items, want 1", c.expr, len(items))
			continue
		}
		n, ok := items[0].(*xmltree.Node)
		if !ok {
			t.Errorf("%s: result is %T, not a node", c.expr, items[0])
			continue
		}
		if got := analyzeRender(n); got != c.want {
			t.Errorf("%s =\n %s\nwant\n %s", c.expr, got, c.want)
		}
	}
	coreRun(t, "", nil, []coreCase{
		{expr: `namespace-uri(analyze-string("a", "a"))`, want: nsFn},
		{expr: `namespace-uri(analyze-string("a", "a")/*)`, want: nsFn},
		{expr: `analyze-string("a", "(a)")//@nr/string()`, want: "1"},
		{expr: `analyze-string("abc", "a?")`, err: "FORX0003"},
		{expr: `analyze-string("abc", "(")`, err: "FORX0002"},
		{expr: `analyze-string("abc", "a", "k")`, err: "FORX0001"},
	})
}
