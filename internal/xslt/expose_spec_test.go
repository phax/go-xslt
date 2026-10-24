package xslt

import (
	"strings"
	"testing"
)

// More xsl:expose cases (XSLT 3.0 §3.5.3.1), on top of expose_test.go: the
// consistency table between an explicit xsl:expose and the declaration's own
// @visibility, implicitly declared modes, and the effect of xsl:expose on the
// entry points of a top-level package and on what a used package exports.

func TestExposeConsistencyTable(t *testing.T) {
	// Rows of the §3.5.3.1 table: the visibility an explicit xsl:expose
	// assigns, against the declared @visibility. "" = permitted, else the
	// error.
	cases := []struct{ exposed, declared, want string }{
		{"public", "public", ""},
		{"public", "final", "XTSE3010"},
		{"public", "private", "XTSE3010"},
		{"private", "public", ""},
		{"private", "private", ""},
		{"private", "final", ""},
		{"final", "public", ""},
		{"final", "final", ""},
		{"final", "private", "XTSE3010"},
		{"abstract", "abstract", ""},
		{"abstract", "public", "XTSE3010"},
		{"private", "abstract", "XTSE3010"},
	}
	for _, c := range cases {
		root := exposeRoot(t, `<xsl:template name="a" visibility="`+c.declared+`"/>
			<xsl:expose component="template" names="a" visibility="`+c.exposed+`"/>`)
		vis, _, err := packageExposure(root)
		switch {
		case c.want == "" && err != nil:
			t.Errorf("expose %s on declared %s: %v", c.exposed, c.declared, err)
		case c.want == "" && visOf(t, root, vis, "template", "a") != c.exposed:
			t.Errorf("expose %s on declared %s: got %q", c.exposed, c.declared, visOf(t, root, vis, "template", "a"))
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("expose %s on declared %s: got %v, want %s", c.exposed, c.declared, err, c.want)
		}
	}
}

func TestExposeMoreErrors(t *testing.T) {
	cases := []struct{ name, body, code string }{
		// "hidden" is not one of xsl:expose's visibilities.
		{"hidden visibility", `<xsl:template name="a"/>
			<xsl:expose component="template" names="a" visibility="hidden"/>`, "XTSE0020"},
		// An arity suffix only identifies a function.
		{"arity on a template", `<xsl:template name="a"/>
			<xsl:expose component="template" names="a#0" visibility="public"/>`, "XTSE3020"},
		{"wrong arity", `<xsl:function name="p:f"><xsl:sequence select="1"/></xsl:function>
			<xsl:expose component="function" names="p:f#2" visibility="public"/>`, "XTSE3020"},
		{"wrong kind", `<xsl:template name="a"/>
			<xsl:expose component="variable" names="a" visibility="public"/>`, "XTSE3020"},
	}
	for _, c := range cases {
		_, _, err := packageExposure(exposeRoot(t, c.body))
		if err == nil || !strings.Contains(err.Error(), c.code) {
			t.Errorf("%s: got %v, want %s", c.name, err, c.code)
		}
	}
}

// TestExposeImplicitMode: a mode named only by a template rule's @mode is a
// component too and can be matched by xsl:expose; a declared xsl:mode and the
// special mode tokens are not implicit modes.
func TestExposeImplicitMode(t *testing.T) {
	root := exposeRoot(t, `
		<xsl:mode name="p:declared"/>
		<xsl:template match="a" mode="p:implicit p:declared #default"/>
		<xsl:template match="b" mode="p:implicit"/>
		<xsl:expose component="mode" names="p:implicit" visibility="final"/>
		<xsl:expose component="mode" names="*" visibility="public"/>`)
	vis, implicit, err := packageExposure(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := implicit["{urn:p}implicit"]; got != "final" || len(implicit) != 1 {
		t.Errorf("implicit modes = %v, want only {urn:p}implicit = final", implicit)
	}
	if got := visOf(t, root, vis, "mode", "p:declared"); got != "public" {
		t.Errorf("declared mode: got %q, want public", got)
	}
}

// pkATop is a top-level package (the principal module) with one named
// template and one implicitly declared mode (declared-modes="no", since its
// default on xsl:package is "yes"), exposed as the argument says.
func pkATop(expose string) string {
	return `<xsl:package version="3.0" declared-modes="no" ` + pkANS + `>
		<xsl:output method="text"/>
		<xsl:template name="go">go</xsl:template>
		<xsl:template match="/" mode="m">m</xsl:template>` + expose + `</xsl:package>`
}

// TestExposeEntryPoints: the initial named template must be public or final
// (XTDE0040) and a named initial mode must not be private (XTDE0045); in a
// package the visibility xsl:expose assigns is what counts.
func TestExposeEntryPoints(t *testing.T) {
	run := func(expose string, e Entry) (string, error) {
		ss, err := Compile(pkATop(expose))
		if err != nil {
			return "", err
		}
		res, err := ss.TransformEntry(`<r/>`, e, "")
		if err != nil {
			return "", err
		}
		return res.Output, nil
	}
	pub := `<xsl:expose component="*" names="*" visibility="public"/>`
	priv := `<xsl:expose component="*" names="*" visibility="private"/>`
	if out, err := run(pub, Entry{Template: "go"}); err != nil || out != "go" {
		t.Errorf("public template: got %q, %v", out, err)
	}
	if _, err := run(priv, Entry{Template: "go"}); err == nil || !strings.Contains(err.Error(), "XTDE0040") {
		t.Errorf("private template: got %v, want XTDE0040", err)
	}
	if out, err := run(pub, Entry{Mode: "m"}); err != nil || out != "m" {
		t.Errorf("public mode: got %q, %v", out, err)
	}
	if _, err := run(priv, Entry{Mode: "m"}); err == nil || !strings.Contains(err.Error(), "XTDE0045") {
		t.Errorf("private mode: got %v, want XTDE0045", err)
	}
}
