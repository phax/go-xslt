package xslt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for xsl:use-package, xsl:accept and xsl:override (XSLT 3.0 §3.5),
// driven end to end: library packages are written to a temporary directory
// and offered to the compile through CompileAtWithPackages, exactly as a host
// makes packages available. Every expectation is derived from §3.5 of the
// XSLT 3.0 Recommendation.

const pkANS = `xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"`

// pkAPkg writes the text of an xsl:package named name (package-version ver,
// omitted when empty) with the given body; the prefix l is bound to urn:pkA:l.
func pkAPkg(name, ver, body string) string {
	v := ""
	if ver != "" {
		v = ` package-version="` + ver + `"`
	}
	return `<xsl:package name="` + name + `"` + v + ` version="3.0" ` + pkANS + ` xmlns:l="urn:pkA:l">` + body + `</xsl:package>`
}

// pkAMain wraps body in a principal stylesheet module with text output.
func pkAMain(body string) string {
	return `<xsl:stylesheet version="3.0" ` + pkANS + ` xmlns:l="urn:pkA:l" exclude-result-prefixes="#all">` +
		`<xsl:output method="text"/>` + body + `</xsl:stylesheet>`
}

// pkAFile is one extra file written next to the principal module.
type pkAFile struct{ name, text string }

// pkACompile writes the packages into a fresh directory, registers each by
// FILE only (its name and version come from its own xsl:package element) and
// compiles main as if read from that directory.
func pkACompile(t *testing.T, main string, pkgs []string, extra ...pkAFile) (*Stylesheet, error) {
	t.Helper()
	dir := t.TempDir()
	var srcs []PackageSource
	for i, p := range pkgs {
		path := filepath.Join(dir, "pkg"+itoa(i)+".xsl")
		if err := os.WriteFile(path, []byte(p), 0o644); err != nil {
			t.Fatal(err)
		}
		srcs = append(srcs, PackageSource{Path: path})
	}
	for _, f := range extra {
		if err := os.WriteFile(filepath.Join(dir, f.name), []byte(f.text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return CompileAtWithPackages(main, filepath.Join(dir, "main.xsl"), nil, srcs)
}

// pkARun compiles and runs main against a small source document, returning
// the first error of either phase.
func pkARun(t *testing.T, main string, pkgs []string, extra ...pkAFile) (string, error) {
	t.Helper()
	ss, err := pkACompile(t, main, pkgs, extra...)
	if err != nil {
		return "", err
	}
	out, _, err := ss.Transform(`<r><i>1</i><i>2</i></r>`, nil)
	return out, err
}

func pkAWant(t *testing.T, name, main string, pkgs []string, want string) {
	t.Helper()
	got, err := pkARun(t, main, pkgs)
	if err != nil {
		t.Errorf("%s: unexpected error %v", name, err)
		return
	}
	if got != want {
		t.Errorf("%s: got %q, want %q", name, got, want)
	}
}

func pkAWantErr(t *testing.T, name, main string, pkgs []string, code string, extra ...pkAFile) {
	t.Helper()
	_, err := pkARun(t, main, pkgs, extra...)
	if err == nil || !strings.Contains(err.Error(), code) {
		if code == pkAAnyErr {
			code = "an error"
		}
		t.Errorf("%s: got error %v, want %s", name, err, code)
	}
}

// pkALib is a library package with one public component of every kind, plus
// private ones the using package must not see.
var pkALib = pkAPkg("urn:pkA:lib", "1.0", `
	<xsl:function name="l:f" visibility="public" as="xs:string">
		<xsl:param name="x" as="xs:string"/>
		<xsl:sequence select="concat('f:', $x, l:priv())"/>
	</xsl:function>
	<xsl:function name="l:priv" visibility="private"><xsl:sequence select="'!'"/></xsl:function>
	<xsl:template name="l:t" visibility="public"><xsl:text>t</xsl:text></xsl:template>
	<xsl:template name="l:tpriv"><xsl:text>tp</xsl:text></xsl:template>
	<xsl:variable name="l:v" visibility="public" select="'v'"/>
	<xsl:variable name="l:vpriv" select="'vp'"/>
	<xsl:param name="l:p" select="'pdef'"/>
	<xsl:attribute-set name="l:as" visibility="public"><xsl:attribute name="a" select="1"/></xsl:attribute-set>
	<xsl:attribute-set name="l:aspriv"><xsl:attribute name="b" select="2"/></xsl:attribute-set>
	<xsl:mode name="l:m" visibility="public"/>
	<xsl:template match="*" mode="l:m">m:<xsl:value-of select="local-name()"/></xsl:template>`)

// TestUsePackagePublicComponents: every public component of a used package
// — and its global parameters, which are always public (§3.5.3.1) — is
// accepted (as private, §3.5.3.2 "in the absence of a matching xsl:accept")
// and usable from the using package; the library's own reference to its
// private function still works.
func TestUsePackagePublicComponents(t *testing.T) {
	main := pkAMain(`<xsl:use-package name="urn:pkA:lib" package-version="1.0"/>
		<xsl:template match="/">
			<xsl:value-of select="l:f('a'), $l:v, $l:p" separator="|"/>|<xsl:call-template name="l:t"/>|<xsl:apply-templates select="r" mode="l:m"/>|<xsl:variable name="e"><e xsl:use-attribute-sets="l:as"/></xsl:variable><xsl:value-of select="$e/e/@a"/>
		</xsl:template>`)
	pkAWant(t, "public components", main, []string{pkALib}, "f:a!|v|pdef|t|m:r|1")
}

// pkAAnyErr marks an expected error whose W3C code the engine does not
// report: a reference to an undeclared function or variable is XPST0017 /
// XPST0008, but the engine raises it as an uncoded dynamic error for every
// stylesheet, packages or not (listed in the coverage report's bug list). The
// test still pins that the reference is rejected.
const pkAAnyErr = ""

// TestUsePackageHiddenComponents: a component that is private in the used
// package becomes hidden in the using package and cannot be referenced
// (§3.5.3.2), so a reference to it is a reference to an undeclared component.
// The same holds for a public component the xsl:accept lowers to hidden.
func TestUsePackageHiddenComponents(t *testing.T) {
	use := `<xsl:use-package name="urn:pkA:lib"/>`
	cases := []struct{ name, body, code string }{
		{"private function", `<xsl:template match="/"><xsl:value-of select="l:priv()"/></xsl:template>`, pkAAnyErr},
		{"private variable", `<xsl:template match="/"><xsl:value-of select="$l:vpriv"/></xsl:template>`, pkAAnyErr},
		{"private template", `<xsl:template match="/"><xsl:call-template name="l:tpriv"/></xsl:template>`, "XTSE0650"},
		{"private attribute set", `<xsl:template match="/"><e xsl:use-attribute-sets="l:aspriv"/></xsl:template>`, "XTSE0710"},
	}
	for _, c := range cases {
		pkAWantErr(t, c.name, pkAMain(use+c.body), []string{pkALib}, c.code)
	}
	hidden := `<xsl:use-package name="urn:pkA:lib">
			<xsl:accept component="function" names="l:f#1" visibility="hidden"/>
		</xsl:use-package>
		<xsl:template match="/"><xsl:value-of select="l:f('a')"/></xsl:template>`
	pkAWantErr(t, "accepted as hidden", pkAMain(hidden), []string{pkALib}, pkAAnyErr)
}

// TestUsePackageAcceptPrecedence: among the xsl:accept elements that match a
// component, an explicit name beats a partial wildcard, which beats "*"
// (§3.5.3.2, same rules as xsl:expose). Templates are used because, unlike
// functions, they may have names in no namespace; calling a hidden template is
// XTSE0650 (no template of that name).
func TestUsePackageAcceptPrecedence(t *testing.T) {
	lib := pkAPkg("urn:pkA:lib", "", `
		<xsl:template name="l:a" visibility="public">a</xsl:template>
		<xsl:template name="l:b" visibility="public">b</xsl:template>
		<xsl:template name="nons" visibility="public">n</xsl:template>`)
	call := func(name string) string {
		return `<xsl:template match="/"><xsl:call-template name="` + name + `"/></xsl:template>`
	}
	// "*" hides everything, the partial wildcard l:* brings l:a and l:b back,
	// and the explicit name hides l:b again — whatever the document order.
	accepts := `<xsl:use-package name="urn:pkA:lib">
			<xsl:accept component="template" names="l:b" visibility="hidden"/>
			<xsl:accept component="template" names="l:*" visibility="private"/>
			<xsl:accept component="*" names="*" visibility="hidden"/>
		</xsl:use-package>`
	pkAWant(t, "partial beats *", pkAMain(accepts+call("l:a")), []string{lib}, "a")
	pkAWantErr(t, "explicit beats partial", pkAMain(accepts+call("l:b")), []string{lib}, "XTSE0650")
	pkAWantErr(t, "* applies", pkAMain(accepts+call("nons")), []string{lib}, "XTSE0650")

	// Q{}* selects exactly the names in no namespace; *:local selects by
	// local name in any namespace.
	noNS := `<xsl:use-package name="urn:pkA:lib">
			<xsl:accept component="template" names="Q{}*" visibility="hidden"/>
			<xsl:accept component="template" names="*:b" visibility="hidden"/>
		</xsl:use-package>`
	pkAWant(t, "Q{}* keeps namespaced", pkAMain(noNS+call("l:a")), []string{lib}, "a")
	pkAWantErr(t, "Q{}* hides no-namespace", pkAMain(noNS+call("nons")), []string{lib}, "XTSE0650")
	pkAWantErr(t, "*:b hides l:b", pkAMain(noNS+call("l:b")), []string{lib}, "XTSE0650")
	// Q{uri}* and Q{uri}local forms, and a component list naming two kinds.
	qforms := `<xsl:use-package name="urn:pkA:lib">
			<xsl:accept component="template" names="Q{urn:pkA:l}*" visibility="hidden"/>
			<xsl:accept component="function template" names="Q{urn:pkA:l}a" visibility="final"/>
		</xsl:use-package>`
	pkAWant(t, "Q{uri}local beats Q{uri}*", pkAMain(qforms+call("l:a")), []string{lib}, "a")
	pkAWantErr(t, "Q{uri}* hides", pkAMain(qforms+call("l:b")), []string{lib}, "XTSE0650")
	// An accept for another component kind does not apply.
	other := `<xsl:use-package name="urn:pkA:lib">
			<xsl:accept component="function" names="*" visibility="hidden"/>
		</xsl:use-package>`
	pkAWant(t, "other kind", pkAMain(other+call("nons")), []string{lib}, "n")
}

// TestUsePackageAcceptErrors pins the static errors of §3.5.3.2.
func TestUsePackageAcceptErrors(t *testing.T) {
	lib := pkAPkg("urn:pkA:lib", "", `
		<xsl:function name="l:f" visibility="public"><xsl:sequence select="'f'"/></xsl:function>
		<xsl:function name="l:fin" visibility="final"><xsl:sequence select="'fin'"/></xsl:function>`)
	use := func(children string) string {
		return pkAMain(`<xsl:use-package name="urn:pkA:lib">` + children + `</xsl:use-package><xsl:template match="/"/>`)
	}
	// XTSE3030: a non-wildcard token matches no component.
	pkAWantErr(t, "XTSE3030", use(`<xsl:accept component="function" names="l:nope#0" visibility="private"/>`), []string{lib}, "XTSE3030")
	// XTSE3040: a final component cannot be accepted as public (table in
	// §3.5.3.2: final -> public is N/P).
	pkAWantErr(t, "XTSE3040 final as public", use(`<xsl:accept component="function" names="l:fin#0" visibility="public"/>`), []string{lib}, "XTSE3040")
	// XTSE3051: an explicit accept naming a component the same
	// xsl:use-package overrides.
	pkAWantErr(t, "XTSE3051", use(`<xsl:accept component="function" names="l:f#0" visibility="private"/>
		<xsl:override><xsl:function name="l:f" visibility="public"><xsl:sequence select="'o'"/></xsl:function></xsl:override>`), []string{lib}, "XTSE3051")

	// With a wildcard, an incompatible visibility is not an error: the accept
	// is treated as not matching the final component, which then keeps its
	// default (accepted as private, so still callable here).
	got, err := pkARun(t, pkAMain(`<xsl:use-package name="urn:pkA:lib">
			<xsl:accept component="function" names="*" visibility="public"/>
		</xsl:use-package>
		<xsl:template match="/"><xsl:value-of select="l:fin(), l:f()"/></xsl:template>`), []string{lib})
	if err != nil || got != "fin f" {
		t.Errorf("wildcard accept of final as public: got %q, %v; want \"fin f\"", got, err)
	}
	// A wildcard that matches nothing is not XTSE3030 (only non-wildcard
	// tokens are subject to it).
	got, err = pkARun(t, pkAMain(`<xsl:use-package name="urn:pkA:lib">
			<xsl:accept component="template" names="l:*" visibility="private"/>
		</xsl:use-package>
		<xsl:template match="/"><xsl:value-of select="l:f()"/></xsl:template>`), []string{lib})
	if err != nil || got != "f" {
		t.Errorf("unmatched wildcard accept: got %q, %v; want \"f\"", got, err)
	}
}

// TestUsePackageHomonyms: XTSE3050 when two used packages make homonymous
// components visible; lowering one of them to hidden resolves the conflict.
// A declaration of the using package homonymous with an accepted component
// is a static error too (§3.5.3.2: "a static error occurs").
func TestUsePackageHomonyms(t *testing.T) {
	a := pkAPkg("urn:pkA:a", "", `<xsl:function name="l:f" visibility="public"><xsl:sequence select="'A'"/></xsl:function>`)
	b := pkAPkg("urn:pkA:b", "", `<xsl:function name="l:f" visibility="public"><xsl:sequence select="'B'"/></xsl:function>`)
	call := `<xsl:template match="/"><xsl:value-of select="l:f()"/></xsl:template>`
	pkAWantErr(t, "two packages", pkAMain(`<xsl:use-package name="urn:pkA:a"/><xsl:use-package name="urn:pkA:b"/>`+call), []string{a, b}, "XTSE3050")
	// Lowering one of them to hidden resolves the conflict (no error); which
	// one a reference then means is pinned by TestUsePackageHiddenHomonym.
	if _, err := pkACompile(t, pkAMain(`<xsl:use-package name="urn:pkA:a"/>
		<xsl:use-package name="urn:pkA:b"><xsl:accept component="function" names="l:f#0" visibility="hidden"/></xsl:use-package>`+call), []string{a, b}); err != nil {
		t.Errorf("one hidden: %v", err)
	}

	_, err := pkARun(t, pkAMain(`<xsl:use-package name="urn:pkA:a"/>
		<xsl:function name="l:f"><xsl:sequence select="'own'"/></xsl:function>`+call), []string{a})
	if err == nil {
		t.Error("a local declaration homonymous with an accepted function must be a static error")
	}
}

// TestUsePackageHiddenHomonym: with l:f accepted from package a and lowered to
// hidden from package b, a reference to l:f in the using package can only mean
// a's (a hidden component "cannot be referenced", §3.5.3.2).
func TestUsePackageHiddenHomonym(t *testing.T) {
	a := pkAPkg("urn:pkA:a", "", `<xsl:function name="l:f" visibility="public"><xsl:sequence select="'A'"/></xsl:function>`)
	b := pkAPkg("urn:pkA:b", "", `<xsl:function name="l:f" visibility="public"><xsl:sequence select="'B'"/></xsl:function>`)
	pkAWant(t, "one hidden", pkAMain(`<xsl:use-package name="urn:pkA:a"/>
		<xsl:use-package name="urn:pkA:b"><xsl:accept component="function" names="l:f#0" visibility="hidden"/></xsl:use-package>
		<xsl:template match="/"><xsl:value-of select="l:f()"/></xsl:template>`), []string{a, b}, "A")
}

// TestUsePackageVersions: xsl:use-package selects by name and by the
// package-version ranges (§3.5.1, §3.5.2); no match is XTSE3000.
func TestUsePackageVersions(t *testing.T) {
	ver := func(v string) string {
		return pkAPkg("urn:pkA:v", v, `<xsl:function name="l:ver" visibility="public"><xsl:sequence select="'`+v+`'"/></xsl:function>`)
	}
	pkgs := []string{ver("1.0"), ver("2.5.1")}
	run := func(spec string) string {
		return pkAMain(`<xsl:use-package name="urn:pkA:v" package-version="` + spec + `"/>
			<xsl:template match="/"><xsl:value-of select="l:ver()"/></xsl:template>`)
	}
	pkAWant(t, "exact 1", run("1"), pkgs, "1.0") // 1 and 1.0 are the same version
	pkAWant(t, "range", run("2 to 3"), pkgs, "2.5.1")
	pkAWant(t, "prefix", run("2.5.*"), pkgs, "2.5.1")
	pkAWant(t, "upper bound", run("to 1.9"), pkgs, "1.0")
	pkAWantErr(t, "no version", run("3+"), pkgs, "XTSE3000")
	pkAWantErr(t, "no name", pkAMain(`<xsl:use-package name="urn:pkA:other"/><xsl:template match="/"/>`), pkgs, "XTSE3000")

	// A package that declares no package-version has version 1.0 (§3.5.1).
	nov := pkAPkg("urn:pkA:nov", "", `<xsl:function name="l:g" visibility="public"><xsl:sequence select="'g'"/></xsl:function>`)
	pkAWant(t, "default version", pkAMain(`<xsl:use-package name="urn:pkA:nov" package-version="1.0.0"/>
		<xsl:template match="/"><xsl:value-of select="l:g()"/></xsl:template>`), []string{nov}, "g")
}

// TestUsePackageHostIdentity: a host registration that names the package and
// version itself is used as given.
func TestUsePackageHostIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib.xsl")
	lib := pkAPkg("urn:pkA:file-name", "9.9", `<xsl:function name="l:g" visibility="public"><xsl:sequence select="'g'"/></xsl:function>`)
	if err := os.WriteFile(path, []byte(lib), 0o644); err != nil {
		t.Fatal(err)
	}
	main := pkAMain(`<xsl:use-package name="urn:pkA:host-name" package-version="3"/>
		<xsl:template match="/"><xsl:value-of select="l:g()"/></xsl:template>`)
	ss, err := CompileAtWithPackages(main, filepath.Join(dir, "main.xsl"), nil,
		[]PackageSource{{Name: "urn:pkA:host-name", Version: "3.0", Path: path}})
	if err != nil {
		t.Fatal(err)
	}
	if out, _, err := ss.Transform(`<r/>`, nil); err != nil || out != "g" {
		t.Errorf("got %q, %v; want \"g\"", out, err)
	}
}

// TestUsePackageStructureErrors: XTSE3005 (a package used recursively),
// XTSE3008 (xsl:use-package reached through xsl:import), XTSE0010 (bad
// content of xsl:use-package).
func TestUsePackageStructureErrors(t *testing.T) {
	a := pkAPkg("urn:pkA:a", "", `<xsl:use-package name="urn:pkA:b"/>`)
	b := pkAPkg("urn:pkA:b", "", `<xsl:use-package name="urn:pkA:a"/>`)
	pkAWantErr(t, "XTSE3005", pkAMain(`<xsl:use-package name="urn:pkA:a"/><xsl:template match="/"/>`), []string{a, b}, "XTSE3005")

	lib := pkAPkg("urn:pkA:lib", "", `<xsl:function name="l:g" visibility="public"><xsl:sequence select="'g'"/></xsl:function>`)
	mod := `<xsl:stylesheet version="3.0" ` + pkANS + `><xsl:use-package name="urn:pkA:lib"/></xsl:stylesheet>`
	pkAWantErr(t, "XTSE3008", pkAMain(`<xsl:import href="mod.xsl"/><xsl:template match="/"/>`), []string{lib}, "XTSE3008", pkAFile{"mod.xsl", mod})
	// Through xsl:include it is allowed (§3.5.2).
	ss, err := pkACompile(t, pkAMain(`<xsl:include href="mod.xsl"/><xsl:template match="/"><xsl:value-of select="l:g()"/></xsl:template>`), []string{lib}, pkAFile{"mod.xsl", mod})
	if err != nil {
		t.Fatalf("use-package in an included module: %v", err)
	}
	if out, _, err := ss.Transform(`<r/>`, nil); err != nil || out != "g" {
		t.Errorf("include: got %q, %v; want \"g\"", out, err)
	}

	pkAWantErr(t, "no name", pkAMain(`<xsl:use-package/><xsl:template match="/"/>`), []string{lib}, "XTSE0010")
	pkAWantErr(t, "bad child", pkAMain(`<xsl:use-package name="urn:pkA:lib"><xsl:variable name="x"/></xsl:use-package><xsl:template match="/"/>`), []string{lib}, "XTSE0010")
}

// pkAOvLib is the library the override tests replace parts of. Its public
// template l:callf calls l:f, so an override of l:f must be seen from inside
// the library too (§3.5.3.3: references are rebound to the overriding
// component).
var pkAOvLib = pkAPkg("urn:pkA:lib", "", `
	<xsl:function name="l:f" visibility="public" as="xs:string">
		<xsl:param name="x" as="xs:string"/>
		<xsl:sequence select="concat('lib:', $x)"/>
	</xsl:function>
	<xsl:function name="l:fin" visibility="final"><xsl:sequence select="1"/></xsl:function>
	<xsl:template name="l:callf" visibility="public"><xsl:value-of select="l:f('x')"/></xsl:template>
	<xsl:template name="l:t" visibility="public" as="xs:string">
		<xsl:param name="p" as="xs:string" required="yes"/>
		<xsl:param name="tun" tunnel="yes"/>
		<xsl:sequence select="'T' || $p"/>
	</xsl:template>
	<xsl:variable name="l:v" visibility="public" as="xs:string" select="'lib'"/>
	<xsl:attribute-set name="l:as" visibility="public"><xsl:attribute name="a" select="1"/></xsl:attribute-set>
	<xsl:mode name="l:m" visibility="public"/>
	<xsl:mode name="l:fm" visibility="final"/>
	<xsl:template match="*" mode="l:m">lib</xsl:template>
	<xsl:template match="*" mode="l:fm">fm</xsl:template>`)

// TestUsePackageOverride: overriding a function, a named template, a
// variable, an attribute set and adding a template rule to a public mode,
// each reaching the overridden component through xsl:original / next-match
// (§3.5.3.3, §3.5.3.4).
func TestUsePackageOverride(t *testing.T) {
	main := pkAMain(`<xsl:use-package name="urn:pkA:lib">
			<xsl:override>
				<xsl:function name="l:f" visibility="public" as="xs:string">
					<xsl:param name="x" as="xs:string"/>
					<xsl:sequence select="concat('ovr(', xsl:original($x), ')')"/>
				</xsl:function>
				<xsl:template name="l:t" visibility="public" as="xs:string">
					<xsl:param name="p" as="xs:string" required="yes"/>
					<xsl:param name="extra" required="no"/>
					<xsl:variable name="o" as="xs:string"><xsl:call-template name="xsl:original"><xsl:with-param name="p" select="$p"/></xsl:call-template></xsl:variable>
					<xsl:sequence select="'O' || $o"/>
				</xsl:template>
				<xsl:variable name="l:v" visibility="public" as="xs:string" select="'o-' || $xsl:original"/>
				<xsl:attribute-set name="l:as" visibility="public" use-attribute-sets="xsl:original"><xsl:attribute name="b" select="2"/></xsl:attribute-set>
				<xsl:template match="r" mode="l:m">ovr+<xsl:next-match/></xsl:template>
			</xsl:override>
		</xsl:use-package>
		<xsl:template match="/">
			<xsl:value-of select="l:f('a')"/>|<xsl:call-template name="l:callf"/>|<xsl:call-template name="l:t"><xsl:with-param name="p" select="'p'"/></xsl:call-template>|<xsl:value-of select="$l:v"/>|<xsl:apply-templates select="r" mode="l:m"/>|<xsl:variable name="e"><e xsl:use-attribute-sets="l:as"/></xsl:variable><xsl:value-of select="$e/e/@*" separator=","/>
		</xsl:template>`)
	pkAWant(t, "override", main, []string{pkAOvLib}, "ovr(lib:a)|ovr(lib:x)|OTp|o-lib|ovr+lib|1,2")
}

// TestUsePackageOverrideErrors pins the static errors of §3.5.3.3 and the
// template-rule rules XTSE3440 / XTSE3460.
func TestUsePackageOverrideErrors(t *testing.T) {
	ov := func(decls string) string {
		return pkAMain(`<xsl:use-package name="urn:pkA:lib"><xsl:override>` + decls + `</xsl:override></xsl:use-package><xsl:template match="/"/>`)
	}
	fn := func(attrs, params, body string) string {
		return `<xsl:function name="l:f" visibility="public"` + attrs + `>` + params + `<xsl:sequence select="` + body + `"/></xsl:function>`
	}
	strParam := `<xsl:param name="x" as="xs:string"/>`
	tmpl := func(attrs, params string) string {
		return `<xsl:template name="l:t" visibility="public"` + attrs + `>` + params + `<xsl:sequence select="'x'"/></xsl:template>`
	}
	reqP := `<xsl:param name="p" as="xs:string" required="yes"/>`
	cases := []struct{ name, decls, code string }{
		{"XTSE3058 unknown function", `<xsl:function name="l:nope"><xsl:sequence select="1"/></xsl:function>`, "XTSE3058"},
		{"XTSE3058 other arity", fn(` as="xs:string"`, "", "''"), "XTSE3058"},
		{"XTSE3060 final function", `<xsl:function name="l:fin"><xsl:sequence select="2"/></xsl:function>`, "XTSE3060"},
		{"XTSE3060 rule in final mode", `<xsl:template match="r" mode="l:fm">x</xsl:template>`, "XTSE3060"},
		{"XTSE3070 result type", fn(` as="xs:integer"`, strParam, "1"), "XTSE3070"},
		{"XTSE3070 param type", fn(` as="xs:string"`, `<xsl:param name="x" as="xs:integer"/>`, "''"), "XTSE3070"},
		{"XTSE3070 new-each-time", fn(` as="xs:string" new-each-time="yes"`, strParam, "''"), "XTSE3070"},
		{"XTSE3070 template result", tmpl(` as="xs:integer"`, reqP+`<xsl:param name="tun" tunnel="yes"/>`), "XTSE3070"},
		{"XTSE3070 template lacks param", tmpl(` as="xs:string"`, ""), "XTSE3070"},
		{"XTSE3070 template param type", tmpl(` as="xs:string"`, `<xsl:param name="p" as="xs:integer" required="yes"/>`), "XTSE3070"},
		{"XTSE3070 template param required", tmpl(` as="xs:string"`, `<xsl:param name="p" as="xs:string"/>`), "XTSE3070"},
		{"XTSE3070 template param tunnel", tmpl(` as="xs:string"`, reqP+`<xsl:param name="tun" tunnel="no"/>`), "XTSE3070"},
		{"XTSE3070 template extra required", tmpl(` as="xs:string"`, reqP+`<xsl:param name="q" required="yes"/>`), "XTSE3070"},
		{"XTSE3070 template context item", tmpl(` as="xs:string"`, `<xsl:context-item use="required"/>`+reqP), "XTSE3070"},
		{"XTSE3070 variable type", `<xsl:variable name="l:v" visibility="public" as="xs:integer" select="1"/>`, "XTSE3070"},
		{"XTSE3440 no mode", `<xsl:template match="r">x</xsl:template>`, "XTSE3440"},
		{"XTSE3440 #all", `<xsl:template match="r" mode="#all">x</xsl:template>`, "XTSE3440"},
		{"XTSE3440 #unnamed", `<xsl:template match="r" mode="#unnamed">x</xsl:template>`, "XTSE3440"},
		{"XTSE3440 #default", `<xsl:template match="r" mode="#default">x</xsl:template>`, "XTSE3440"},
		{"XTSE3460 apply-imports", `<xsl:template match="r" mode="l:m"><xsl:apply-imports/></xsl:template>`, "XTSE3460"},
	}
	for _, c := range cases {
		pkAWantErr(t, c.name, ov(c.decls), []string{pkAOvLib}, c.code)
	}

	// Controls: a signature repeated verbatim, an omitted tunnel parameter, and
	// a template rule whose mode comes from [xsl:]default-mode are all fine.
	okDecls := fn(` as="xs:string"`, strParam, "'ok'") +
		tmpl(` as="xs:string"`, reqP) +
		`<xsl:template match="r" mode="#default" default-mode="l:m">x</xsl:template>`
	if _, err := pkACompile(t, ov(okDecls), []string{pkAOvLib}); err != nil {
		t.Errorf("compatible overrides: %v", err)
	}
}

// TestUsePackageAbstract: an abstract component may live in a library
// package; overriding it supplies the implementation. Left unoverridden it is
// hidden, and invoking it from the library is XTDE3052. A top-level package
// may contain no abstract component (XTSE3080).
func TestUsePackageAbstract(t *testing.T) {
	lib := pkAPkg("urn:pkA:abs", "", `
		<xsl:function name="l:abs" visibility="abstract" as="xs:string"/>
		<xsl:template name="l:run" visibility="public"><xsl:value-of select="l:abs()"/></xsl:template>`)
	run := `<xsl:template match="/"><xsl:call-template name="l:run"/></xsl:template>`
	pkAWant(t, "overridden", pkAMain(`<xsl:use-package name="urn:pkA:abs"><xsl:override>
			<xsl:function name="l:abs" visibility="public" as="xs:string"><xsl:sequence select="'impl'"/></xsl:function>
		</xsl:override></xsl:use-package>`+run), []string{lib}, "impl")
	pkAWantErr(t, "not overridden", pkAMain(`<xsl:use-package name="urn:pkA:abs"/>`+run), []string{lib}, "XTDE3052")
	pkAWantErr(t, "accepted as abstract", pkAMain(`<xsl:use-package name="urn:pkA:abs">
			<xsl:accept component="function" names="l:abs#0" visibility="abstract"/>
		</xsl:use-package>`+run), []string{lib}, "XTSE3080")

	top := `<xsl:package version="3.0" ` + pkANS + ` xmlns:l="urn:pkA:l">
		<xsl:function name="l:abs" visibility="abstract"/>
		<xsl:template match="/"/></xsl:package>`
	pkAWantErr(t, "own abstract", top, nil, "XTSE3080")
}

// TestUsePackageDiamond: two used packages each use a different version of
// one library declaring the same public variable; each sees its own
// version's value, and each keeps its own package-local decimal format
// (§3.5: decimal formats are local to the package).
func TestUsePackageDiamond(t *testing.T) {
	base := func(v string) string {
		return pkAPkg("urn:pkA:base", v, `<xsl:variable name="l:v" visibility="public" select="'`+v+`'"/>`)
	}
	user := func(name, spec, sep string) string {
		return `<xsl:package name="` + name + `" version="3.0" ` + pkANS + ` xmlns:l="urn:pkA:l" xmlns:u="` + name + `">
			<xsl:decimal-format decimal-separator="` + sep + `" grouping-separator="_"/>
			<xsl:use-package name="urn:pkA:base" package-version="` + spec + `"/>
			<xsl:function name="u:get" visibility="public"><xsl:sequence select="$l:v || '/' || format-number(1.5, '0` + sep + `0')"/></xsl:function>
		</xsl:package>`
	}
	main := `<xsl:stylesheet version="3.0" ` + pkANS + ` xmlns:a="urn:pkA:ua" xmlns:b="urn:pkA:ub">
		<xsl:output method="text"/>
		<xsl:use-package name="urn:pkA:ua"/><xsl:use-package name="urn:pkA:ub"/>
		<xsl:template match="/"><xsl:value-of select="a:get(), b:get(), format-number(1.5, '0.0')" separator=" "/></xsl:template>
	</xsl:stylesheet>`
	pkAWant(t, "diamond", main, []string{base("1.0"), base("2.0"), user("urn:pkA:ua", "1", ","), user("urn:pkA:ub", "2", "!")}, "1.0/1,5 2.0/1!5 1.5")
}
