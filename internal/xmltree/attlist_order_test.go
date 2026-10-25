package xmltree

import (
	"strings"
	"testing"
)

// DTD-defaulted attributes are added in declaration order, the same on every
// parse (they used to follow Go's map order, so one document could serialize
// differently from run to run), and the first declaration of an attribute is
// binding (XML 1.0 §3.3).
func TestAttlistDefaultsOrderAndFirstBinding(t *testing.T) {
	src := `<!DOCTYPE r [
<!ATTLIST e zeta CDATA "1" alpha CDATA "2" mid CDATA "3">
<!ATTLIST e omega CDATA "4" alpha CDATA "ignored">
]><r><e/><e alpha="own"/></r>`
	for i := 0; i < 50; i++ {
		doc, err := Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		r := doc.Children[len(doc.Children)-1]
		var got []string
		for _, e := range r.Children {
			var parts []string
			for _, a := range e.Attrs {
				parts = append(parts, a.Name.Local+"="+a.Value)
			}
			got = append(got, strings.Join(parts, " "))
		}
		want := []string{"zeta=1 alpha=2 mid=3 omega=4", "alpha=own zeta=1 mid=3 omega=4"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("parse %d: got %q, want %q", i, got, want)
		}
	}
}
