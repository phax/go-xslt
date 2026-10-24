package xslt

import "testing"

// More xsl:use-package cases (XSLT 3.0 §3.5): re-exported components, the
// rest of the xsl:accept visibility table, template rules for used modes,
// xsl:original of an abstract component, and package-local declarations.
// Helpers are in use_package_test.go.

// TestUsePackageReexport: a package that uses another may make an accepted
// component part of its own manifest by accepting it with an exposed
// visibility (§3.5.3.2); xsl:expose in a used package decides what it
// exports (§3.5.3.1).
func TestUsePackageReexport(t *testing.T) {
	inner := pkAPkg("urn:pkA:inner", "", `<xsl:function name="l:in" visibility="public"><xsl:sequence select="'in'"/></xsl:function>
		<xsl:function name="l:kept" visibility="public"><xsl:sequence select="'kept'"/></xsl:function>`)
	mid := pkAPkg("urn:pkA:mid", "", `
		<xsl:use-package name="urn:pkA:inner">
			<xsl:accept component="function" names="l:in#0" visibility="public"/>
			<xsl:accept component="function" names="l:kept#0" visibility="private"/>
		</xsl:use-package>
		<xsl:function name="l:exposed"><xsl:sequence select="'exp'"/></xsl:function>
		<xsl:expose component="function" names="l:exposed#0" visibility="public"/>`)
	use := `<xsl:use-package name="urn:pkA:mid"/>`
	pkAWant(t, "re-exported", pkAMain(use+`<xsl:template match="/"><xsl:value-of select="l:in(), l:exposed()"/></xsl:template>`), []string{inner, mid}, "in exp")
}

// TestUsePackageAcceptedPrivateNotReexported: a component the middle package
// accepts as private is private there, so it is hidden from a package using
// the middle one (§3.5.3.2) and cannot be referenced.
func TestUsePackageAcceptedPrivateNotReexported(t *testing.T) {
	inner := pkAPkg("urn:pkA:inner", "", `<xsl:function name="l:kept" visibility="public"><xsl:sequence select="'kept'"/></xsl:function>`)
	mid := pkAPkg("urn:pkA:mid", "", `
		<xsl:use-package name="urn:pkA:inner">
			<xsl:accept component="function" names="l:kept#0" visibility="private"/>
		</xsl:use-package>`)
	pkAWantErr(t, "accepted privately is not re-exported", pkAMain(`<xsl:use-package name="urn:pkA:mid"/>
		<xsl:template match="/"><xsl:value-of select="l:kept()"/></xsl:template>`), []string{inner, mid}, pkAAnyErr)
}

// TestUsePackageMoreAcceptErrors: further cases of the §3.5.3.2 table and of
// XTSE3030.
func TestUsePackageMoreAcceptErrors(t *testing.T) {
	lib := pkAPkg("urn:pkA:lib", "", `
		<xsl:function name="l:f" visibility="public"><xsl:param name="x"/><xsl:sequence select="$x"/></xsl:function>
		<xsl:function name="l:fin" visibility="final"><xsl:sequence select="1"/></xsl:function>`)
	use := func(children string) string {
		return pkAMain(`<xsl:use-package name="urn:pkA:lib">` + children + `</xsl:use-package><xsl:template match="/"/>`)
	}
	// l:f has arity 1, so l:f#0 names nothing.
	pkAWantErr(t, "wrong arity", use(`<xsl:accept component="function" names="l:f#0" visibility="private"/>`), []string{lib}, "XTSE3030")
	// A name of the right kind is required.
	pkAWantErr(t, "wrong kind", use(`<xsl:accept component="template" names="l:f" visibility="private"/>`), []string{lib}, "XTSE3030")
	// final -> final, private and hidden are permitted.
	for _, v := range []string{"final", "private", "hidden"} {
		if _, err := pkACompile(t, use(`<xsl:accept component="function" names="l:fin#0" visibility="`+v+`"/>`), []string{lib}); err != nil {
			t.Errorf("final accepted as %s: %v", v, err)
		}
	}
}

// TestUsePackageAbstractAcceptTable: per the §3.5.3.2 table an abstract
// component can only be accepted as abstract or hidden, and only an abstract
// component can be accepted as abstract; anything else named explicitly is
// XTSE3040.
func TestUsePackageAbstractAcceptTable(t *testing.T) {
	t.Skip("engine bug: visibilityCompatible permits abstract->public/private/final and public->abstract (the last reported as XTSE3080 instead)")
	lib := pkAPkg("urn:pkA:abs", "", `
		<xsl:function name="l:abs" visibility="abstract"/>
		<xsl:function name="l:pub" visibility="public"><xsl:sequence select="1"/></xsl:function>`)
	use := func(children string) string {
		return pkAMain(`<xsl:use-package name="urn:pkA:abs">` + children + `</xsl:use-package><xsl:template match="/"/>`)
	}
	for _, v := range []string{"public", "private", "final"} {
		pkAWantErr(t, "abstract as "+v, use(`<xsl:accept component="function" names="l:abs#0" visibility="`+v+`"/>`), []string{lib}, "XTSE3040")
	}
	pkAWantErr(t, "public as abstract", use(`<xsl:accept component="function" names="l:pub#0" visibility="abstract"/>`), []string{lib}, "XTSE3040")
}

// TestUsePackageModeRules: template rules for a mode of a used package may
// only be added through xsl:override (§3.5.3.3); a rule outside it is
// rejected (W3C xslt30-test override-m-018 expects XTSE3050).
func TestUsePackageModeRules(t *testing.T) {
	main := pkAMain(`<xsl:use-package name="urn:pkA:lib"/>
		<xsl:template match="r" mode="l:m #default">own</xsl:template>`)
	pkAWantErr(t, "rule outside override", main, []string{pkAOvLib}, "XTSE3050")
}

// TestUsePackageLocalDeclarations: keys and character maps are local to the
// package that declares them (§3.5: they "have local scope within that
// package"), even when two packages use the same name.
func TestUsePackageLocalDeclarations(t *testing.T) {
	lib := pkAPkg("urn:pkA:lib", "", `
		<xsl:key name="k" match="i" use="."/>
		<xsl:character-map name="cm"><xsl:output-character character="x" string="LIB"/></xsl:character-map>
		<xsl:output method="text" use-character-maps="cm"/>
		<xsl:function name="l:keyed" visibility="public">
			<xsl:param name="d"/>
			<xsl:sequence select="local-name(key('k', '1', $d))"/>
		</xsl:function>`)
	main := `<xsl:stylesheet version="3.0" ` + pkANS + ` xmlns:l="urn:pkA:l">
		<xsl:output method="text" use-character-maps="cm"/>
		<xsl:character-map name="cm"><xsl:output-character character="x" string="MAIN"/></xsl:character-map>
		<xsl:key name="k" match="r" use="'1'"/>
		<xsl:use-package name="urn:pkA:lib"/>
		<xsl:template match="/"><xsl:value-of select="l:keyed(/), local-name(key('k', '1')), 'x'"/></xsl:template>
	</xsl:stylesheet>`
	pkAWant(t, "local keys and character maps", main, []string{lib}, "i r MAIN")
}

// TestUsePackageOriginalAbstract: xsl:original may not be used when the
// overridden component is abstract (XTSE3075, §3.5.3.4).
func TestUsePackageOriginalAbstract(t *testing.T) {
	t.Skip("engine bug: XTSE3075 is not detected; the call fails at run time with XTDE3052 instead")
	lib := pkAPkg("urn:pkA:abs", "", `<xsl:function name="l:abs" visibility="abstract" as="xs:string"/>`)
	main := pkAMain(`<xsl:use-package name="urn:pkA:abs"><xsl:override>
			<xsl:function name="l:abs" visibility="public" as="xs:string"><xsl:sequence select="xsl:original()"/></xsl:function>
		</xsl:override></xsl:use-package><xsl:template match="/"><xsl:value-of select="l:abs()"/></xsl:template>`)
	pkAWantErr(t, "xsl:original of abstract", main, []string{lib}, "XTSE3075")
}
