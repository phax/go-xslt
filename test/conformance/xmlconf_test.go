package conformance

// W3C XML Conformance Test Suite (xmlconf) harness: runs every TEST case of
// the official suite through xmltree's document parser and records the
// verdict (accepted / rejected). It is a regression gate for the parser, not
// a conformance claim — xmltree is a non-validating parser, so "valid" and
// "invalid" cases both pass when the document is accepted, "not-wf" cases
// when it is rejected, and "error" cases are recorded but not scored.
//
// Download (the test skips if absent; never commit it):
//   curl -L https://www.w3.org/XML/Test/xmlts20130923.tar.gz | tar xz -C test/conformance
// Run:  go test ./test/conformance/ -run TestXMLConf -v
//   XMLCONF_OUT=<file>  writes one line per case and entry point:
//                       "<id> <entry> <type> accept|reject<TAB><error message>"
// Diff two such files by name to compare two parser builds (cut -f1 for the
// verdicts alone).

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

type xmlconfCase struct {
	ID, Type, Path string
}

var (
	xmlconfEntityDecl = regexp.MustCompile(`<!ENTITY\s+([\w.-]+)\s+SYSTEM\s+"([^"]+)"\s*>`)
	xmlconfGroup      = regexp.MustCompile(`(?s)<TESTCASES[^>]*>(.*?)</TESTCASES>`)
	xmlconfEntityRef  = regexp.MustCompile(`&([\w.-]+);`)
)

// xmlconfCases reads the master catalog and every per-suite catalog it pulls
// in through external entities, honouring nested xml:base.
func xmlconfCases(t *testing.T, root string) []xmlconfCase {
	master, err := os.ReadFile(filepath.Join(root, "xmlconf.xml"))
	if err != nil {
		t.Fatal(err)
	}
	ents := map[string]string{}
	for _, m := range xmlconfEntityDecl.FindAllStringSubmatch(string(master), -1) {
		ents[m[1]] = m[2]
	}
	var out []xmlconfCase
	for _, g := range xmlconfGroup.FindAllStringSubmatch(string(master), -1) {
		for _, r := range xmlconfEntityRef.FindAllStringSubmatch(g[1], -1) {
			file, ok := ents[r[1]]
			if !ok {
				t.Fatalf("xmlconf: undeclared entity %s", r[1])
			}
			catPath := filepath.Join(root, filepath.FromSlash(file))
			src, err := os.ReadFile(catPath)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := xmltree.ParseWithBase(string(src), filepath.Dir(catPath))
			if err != nil {
				t.Fatalf("xmlconf: catalog %s: %v", catPath, err)
			}
			var walk func(n *xmltree.Node, dir string)
			walk = func(n *xmltree.Node, dir string) {
				for _, c := range n.Children {
					if c.Kind != xmltree.KindElement {
						continue
					}
					d := dir
					if b, ok := c.Attr("http://www.w3.org/XML/1998/namespace", "base"); ok {
						d = filepath.Join(dir, filepath.FromSlash(b))
					}
					switch c.Name.Local {
					case "TESTCASES":
						walk(c, d)
					case "TEST":
						id, _ := c.Attr("", "ID")
						typ, _ := c.Attr("", "TYPE")
						uri, _ := c.Attr("", "URI")
						out = append(out, xmlconfCase{ID: id, Type: typ, Path: filepath.Join(d, filepath.FromSlash(uri))})
					}
				}
			}
			// An external entity's base URI is its own location (XML Base
			// §4.2) — which also sidesteps the master's one wrong xml:base
			// (eduni/namespaces/misc/ for files in eduni/misc/).
			walk(doc, filepath.Dir(catPath))
		}
	}
	return out
}

func TestXMLConf(t *testing.T) {
	root := filepath.Join("xmlconf")
	if _, err := os.Stat(filepath.Join(root, "xmlconf.xml")); err != nil {
		t.Skip("W3C XML conformance suite not present (see xmlconf_test.go)")
	}
	cases := xmlconfCases(t, root)
	if len(cases) == 0 {
		t.Fatal("xmlconf: no test cases found")
	}
	entries := []struct {
		name  string
		parse func(src, dir string) (*xmltree.Node, error)
	}{
		{"Parse", func(src, _ string) (*xmltree.Node, error) { return xmltree.Parse(src) }},
		{"ParseLenient11WithBase", xmltree.ParseLenient11WithBase},
	}
	var lines []string
	type tally struct{ pass, fail, unscored int }
	totals := map[string]*tally{}
	seen := map[string]bool{}
	for _, c := range cases {
		key := c.ID
		if seen[key] { // a few IDs repeat across catalogs; keep them apart
			for i := 2; seen[key]; i++ {
				key = fmt.Sprintf("%s#%d", c.ID, i)
			}
		}
		seen[key] = true
		src, err := os.ReadFile(c.Path)
		if err != nil {
			t.Errorf("xmlconf %s: %v", key, err)
			continue
		}
		for _, e := range entries {
			verdict, msg := func() (v, msg string) {
				defer func() {
					if r := recover(); r != nil {
						v = "panic"
					}
				}()
				if _, err := e.parse(string(src), filepath.Dir(c.Path)); err != nil {
					return "reject", err.Error()
				}
				return "accept", ""
			}()
			lines = append(lines, fmt.Sprintf("%s %s %s %s\t%q", key, e.name, c.Type, verdict, msg))
			tl := totals[e.name]
			if tl == nil {
				tl = &tally{}
				totals[e.name] = tl
			}
			switch {
			case c.Type == "error":
				tl.unscored++
			case (c.Type == "not-wf") == (verdict == "reject") && verdict != "panic":
				tl.pass++
			default:
				tl.fail++
			}
			if verdict == "panic" {
				t.Errorf("xmlconf %s (%s): parser panicked", key, e.name)
			}
		}
	}
	sort.Strings(lines)
	for _, e := range entries {
		tl := totals[e.name]
		t.Logf("xmlconf %s: %d cases, %d pass, %d fail, %d unscored (error)",
			e.name, tl.pass+tl.fail+tl.unscored, tl.pass, tl.fail, tl.unscored)
	}
	if out := os.Getenv("XMLCONF_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
