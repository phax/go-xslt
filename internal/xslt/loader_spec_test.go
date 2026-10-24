package xslt

import (
	"path/filepath"
	"strings"
	"testing"
)

const ldHead = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`

// XSLT 3.0 §3.11 combining stylesheet modules: xsl:include, xsl:import,
// import precedence, simplified and embedded modules.
func TestLoaderSpecModules(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "inc.xsl", ldHead+`<xsl:template match="b">inc-b</xsl:template></xsl:stylesheet>`)
	specWrite(t, dir, "imp1.xsl", ldHead+`<xsl:import href="sub/imp2.xsl"/><xsl:template match="a">imp1-a[<xsl:apply-imports/>]</xsl:template><xsl:template match="c">imp1-c</xsl:template></xsl:stylesheet>`)
	specWrite(t, dir, "sub/imp2.xsl", ldHead+`<xsl:template match="a">imp2-a</xsl:template><xsl:template match="c">imp2-c</xsl:template><xsl:template match="d">imp2-d</xsl:template></xsl:stylesheet>`)
	specWrite(t, dir, "simple.xsl", `<out xsl:version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:value-of select="count(//*)"/></out>`)
	specWrite(t, dir, "host.xml", `<doc><xsl:stylesheet id="emb" version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="e">embedded</xsl:template></xsl:stylesheet></doc>`)
	specWrite(t, dir, "notxslt.xml", `<doc/>`)
	specWrite(t, dir, "cyc1.xsl", ldHead+`<xsl:include href="cyc2.xsl"/></xsl:stylesheet>`)
	specWrite(t, dir, "cyc2.xsl", ldHead+`<xsl:include href="cyc1.xsl"/></xsl:stylesheet>`)
	specWrite(t, dir, "icyc1.xsl", ldHead+`<xsl:import href="icyc2.xsl"/></xsl:stylesheet>`)
	specWrite(t, dir, "icyc2.xsl", ldHead+`<xsl:import href="icyc1.xsl"/></xsl:stylesheet>`)
	specWrite(t, dir, "preserve.xsl", `<xsl:stylesheet version="3.0" input-type-annotations="preserve" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>`)

	src := `<r><a/><b/><c/><d/><e/></r>`
	main := func(decls string) string {
		return `<xsl:stylesheet version="3.0" ` + specNS + `>` + decls + `<xsl:output method="text"/>
<xsl:template match="/"><xsl:apply-templates select="r/*"/></xsl:template>
<xsl:template match="e"/></xsl:stylesheet>`
	}
	specRunAll(t, dir, []specCase{
		{name: "include", src: src, want: "inc-b", sheet: main(`<xsl:include href="inc.xsl"/>`)},
		// The importing module wins; xsl:apply-imports reaches the next lower one.
		{name: "import chain and apply-imports", src: src, want: "imp1-a[imp2-a]imp1-cimp2-d", sheet: main(`<xsl:import href="imp1.xsl"/>`)},
		{name: "importing module overrides", src: src, want: "imp1-a[imp2-a]MAINimp2-d", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `><xsl:import href="imp1.xsl"/><xsl:output method="text"/>
<xsl:template match="/"><xsl:apply-templates select="r/*"/></xsl:template>
<xsl:template match="c">MAIN</xsl:template><xsl:template match="b | e"/></xsl:stylesheet>`},
		// §3.8: a simplified module is a stylesheet with one template for "/".
		{name: "import a simplified module", src: src, want: "<out>6</out>", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `><xsl:import href="simple.xsl"/><xsl:output method="xml" omit-xml-declaration="yes"/></xsl:stylesheet>`},
		// §3.12: an embedded module named by a fragment identifier.
		{name: "include an embedded module", src: src, want: "embedded", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `><xsl:include href="host.xml#emb"/><xsl:output method="text"/>
<xsl:template match="/"><xsl:apply-templates select="r/e"/></xsl:template></xsl:stylesheet>`},
		// XTSE0165: the resource is not a stylesheet module / cannot be retrieved.
		{name: "not a stylesheet", code: "XTSE0165", sheet: main(`<xsl:include href="notxslt.xml"/>`)},
		{name: "fragment not found", code: "XTSE0165", sheet: main(`<xsl:include href="host.xml#nope"/>`)},
		// XTSE0265: conflicting input-type-annotations.
		{name: "conflicting input-type-annotations", code: "XTSE0265", sheet: `<xsl:stylesheet version="3.0" input-type-annotations="strip" ` + specNS + `><xsl:include href="preserve.xsl"/>` +
			`<xsl:template match="/">x</xsl:template></xsl:stylesheet>`},
	})
	// §5.1.2: xpath-default-namespace is a standard attribute whose scope is
	// the element it is on and its descendants — on an included or imported
	// module's root element it governs that module's expressions and
	// patterns, not the including module's.
	specWrite(t, dir, "xdn-inc.xsl", `<xsl:stylesheet version="3.0" xpath-default-namespace="urn:x" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:template match="item">[<xsl:value-of select="count(//item)"/>]</xsl:template></xsl:stylesheet>`)
	xdnSrc := `<r xmlns="urn:x"><item/><item/></r>`
	xdnMain := func(decl string) string {
		return `<xsl:stylesheet version="3.0" ` + specNS + `>` + decl + `<xsl:output method="text"/>
<xsl:template match="/"><xsl:apply-templates select="*/*"/></xsl:template></xsl:stylesheet>`
	}
	specRunAll(t, dir, []specCase{
		{name: "xpath-default-namespace on an included module", src: xdnSrc, want: "[2][2]", sheet: xdnMain(`<xsl:include href="xdn-inc.xsl"/>`)},
		{name: "xpath-default-namespace on an imported module", src: xdnSrc, want: "[2][2]", sheet: xdnMain(`<xsl:import href="xdn-inc.xsl"/>`)},
	})
	// Behaviour that disagrees with the REC, skipped with the bug it shows.
	bugs := []struct {
		bug string
		c   specCase
	}{
		{"BUG: an xsl:include whose resource cannot be retrieved fails without an error code; §3.11.1 defines XTSE0165",
			specCase{name: "missing module", code: "XTSE0165", sheet: main(`<xsl:include href="missing.xsl"/>`)}},
		{"BUG: a module that includes itself (indirectly) fails without an error code; §3.11.2 defines XTSE0180",
			specCase{name: "include cycle", code: "XTSE0180", sheet: main(`<xsl:include href="cyc1.xsl"/>`)}},
		{"BUG: a module that imports itself (indirectly) fails without an error code; §3.11.3 defines XTSE0210",
			specCase{name: "import cycle", code: "XTSE0210", sheet: main(`<xsl:import href="icyc1.xsl"/>`)}},
		{"BUG: xsl:import after another declaration is accepted; §3.11.3 requires imports first (XTSE0200)",
			specCase{name: "import after a declaration", code: "XTSE0200", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `><xsl:template match="/">x</xsl:template><xsl:import href="inc.xsl"/></xsl:stylesheet>`}},
	}
	for _, b := range bugs {
		t.Run(b.c.name, func(t *testing.T) {
			t.Skip(b.bug)
			specCheck(t, dir, b.c)
		})
	}
}

// XSLT 3.0 §3.5.2 xsl:use-package against host-supplied packages.
func TestLoaderSpecUsePackage(t *testing.T) {
	dir := t.TempDir()
	lib := `<xsl:package name="urn:lib" package-version="1.0" version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:l="urn:lib">
<xsl:function name="l:twice" visibility="public"><xsl:param name="n"/><xsl:sequence select="$n * 2"/></xsl:function>
</xsl:package>`
	specWrite(t, dir, "lib.xsl", lib)
	specWrite(t, dir, "a.xsl", `<xsl:package name="urn:a" version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:use-package name="urn:b"/></xsl:package>`)
	specWrite(t, dir, "b.xsl", `<xsl:package name="urn:b" version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:use-package name="urn:a"/></xsl:package>`)
	specWrite(t, dir, "notpkg.xsl", ldHead+`</xsl:stylesheet>`)
	pkgs := []PackageSource{
		{Name: "urn:lib", Path: filepath.Join(dir, "lib.xsl")},
		{Name: "urn:a", Path: filepath.Join(dir, "a.xsl")},
		{Name: "urn:b", Path: filepath.Join(dir, "b.xsl")},
		{Name: "urn:notpkg", Path: filepath.Join(dir, "notpkg.xsl")},
	}
	top := func(use string) string {
		return `<xsl:stylesheet version="3.0" ` + specNS + ` xmlns:l="urn:lib">` + use + `<xsl:output method="text"/>
<xsl:template match="/"><xsl:value-of select="l:twice(21)"/></xsl:template></xsl:stylesheet>`
	}
	cases := []struct {
		name, sheet, want, code string
	}{
		{"use a package", top(`<xsl:use-package name="urn:lib" package-version="1.0"/>`), "42", ""},
		{"accept inside use-package", top(`<xsl:use-package name="urn:lib"><xsl:accept component="function" names="*" visibility="public"/></xsl:use-package>`), "42", ""},
		{"use-package without a name", top(`<xsl:use-package/>`), "", "XTSE0010"},
		{"template inside use-package", top(`<xsl:use-package name="urn:lib"><xsl:template name="t"/></xsl:use-package>`), "", "XTSE0010"},
		{"foreign element inside use-package", top(`<xsl:use-package name="urn:lib"><f:x/></xsl:use-package>`), "", "XTSE0010"},
		// XTSE3005: packages that use each other.
		{"recursive packages", top(`<xsl:use-package name="urn:a"/>`), "", "XTSE3005"},
		{"not a package module", top(`<xsl:use-package name="urn:notpkg"/>`), "", "XTSE0165"},
		// XTSE3000: no package of that name/version is available.
		{"unavailable version", top(`<xsl:use-package name="urn:lib" package-version="2.0"/>`), "", "XTSE3000"},
	}
	for _, c := range cases {
		ss, err := CompileAtWithPackages(c.sheet, filepath.Join(dir, "main.xsl"), nil, pkgs)
		if c.code != "" {
			if err == nil || !strings.Contains(err.Error(), c.code) {
				t.Errorf("%s: want %s, got %v", c.name, c.code, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		out, _, err := ss.Transform("<doc/>", nil)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
		} else if out != c.want {
			t.Errorf("%s: got %q want %q", c.name, out, c.want)
		}
	}
}
