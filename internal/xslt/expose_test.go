package xslt

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

func exposeRoot(t *testing.T, body string) *xmltree.Node {
	t.Helper()
	doc, err := xmltree.Parse(`<xsl:package version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:p="urn:p">` + body + `</xsl:package>`)
	if err != nil {
		t.Fatal(err)
	}
	return xmltree.RootElement(doc)
}

// visOf finds the effective visibility of the declaration with this name.
func visOf(t *testing.T, root *xmltree.Node, vis map[*xmltree.Node]string, kind, name string) string {
	t.Helper()
	for _, ch := range elementChildren(root) {
		if ch.Name.Local == kind {
			if n, _ := ch.AttrLocal("name"); n == name {
				return vis[ch]
			}
		}
	}
	t.Fatalf("no %s %q", kind, name)
	return ""
}

// TestPackageExposureOrder pins the §3.5.3.1 precedence: explicit name, then
// the declaration's own @visibility, then partial wildcard, then "*", then
// private; the last matching xsl:expose of a class wins.
func TestPackageExposureOrder(t *testing.T) {
	root := exposeRoot(t, `
		<xsl:template name="a"/>
		<xsl:template name="b" visibility="final"/>
		<xsl:template name="p:c"/>
		<xsl:template name="d"/>
		<xsl:function name="p:f"><xsl:sequence select="1"/></xsl:function>
		<xsl:expose component="template" names="*" visibility="private"/>
		<xsl:expose component="template" names="p:*" visibility="public"/>
		<xsl:expose component="template" names="a" visibility="final"/>
		<xsl:expose component="template" names="a" visibility="private"/>
		<xsl:expose component="function" names="p:f#0" visibility="public"/>`)
	vis, _, err := packageExposure(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ kind, name, want string }{
		{"template", "a", "private"},  // the last explicit expose wins
		{"template", "b", "final"},    // declared beats the "*" wildcard
		{"template", "p:c", "public"}, // partial wildcard beats "*"
		{"template", "d", "private"},  // only "*" matched
		{"function", "p:f", "public"},
	} {
		if got := visOf(t, root, vis, c.kind, c.name); got != c.want {
			t.Errorf("%s %s: got %q, want %q", c.kind, c.name, got, c.want)
		}
	}
}

func TestPackageExposureErrors(t *testing.T) {
	cases := []struct{ name, body, code string }{
		{"raise private to public", `<xsl:template name="a" visibility="private"/>
			<xsl:expose component="template" names="a" visibility="public"/>`, "XTSE3010"},
		{"unknown name", `<xsl:template name="a"/>
			<xsl:expose component="template" names="b" visibility="public"/>`, "XTSE3020"},
		{"function without arity", `<xsl:function name="p:f"><xsl:sequence select="1"/></xsl:function>
			<xsl:expose component="function" names="p:f" visibility="public"/>`, "XTSE3020"},
		{"star kind needs wildcard", `<xsl:template name="a"/>
			<xsl:expose component="*" names="a" visibility="public"/>`, "XTSE3022"},
		{"cannot make abstract", `<xsl:template name="a"/>
			<xsl:expose component="template" names="a" visibility="abstract"/>`, "XTSE3025"},
		{"unnamed mode", `<xsl:mode/>
			<xsl:expose component="mode" names="#unnamed" visibility="public"/>`, "XTSE0020"},
		{"bad component kind", `<xsl:template name="a"/>
			<xsl:expose component="widget" names="a" visibility="public"/>`, "XTSE0020"},
		{"undeclared prefix", `<xsl:template name="a"/>
			<xsl:expose component="template" names="zz:*" visibility="public"/>`, "XTSE0020"},
		{"param is never matched", `<xsl:param name="v"/>
			<xsl:expose component="variable" names="v" visibility="private"/>`, "XTSE3020"},
	}
	for _, c := range cases {
		_, _, err := packageExposure(exposeRoot(t, c.body))
		if err == nil || !strings.Contains(err.Error(), c.code) {
			t.Errorf("%s: got %v, want %s", c.name, err, c.code)
		}
	}
}
