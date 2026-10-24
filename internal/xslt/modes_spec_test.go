package xslt

import "testing"

// XSLT 3.0 §6.7 built-in template rules selected by xsl:mode/@on-no-match,
// and §6.6.? on-multiple-match.
func TestModeSpecOnNoMatch(t *testing.T) {
	src := `<a x="1"><b/>t<!--c--><?p d?><c><b/></c></a>`
	specRunAll(t, "", []specCase{
		{name: "shallow-copy", src: src, want: `<a x="1"><B/>t<!--c--><?p d?><c><B/></c></a>`, sheet: specXML(`
<xsl:mode on-no-match="shallow-copy"/>
<xsl:template match="b"><B/></xsl:template>`)},
		{name: "shallow-copy reaches attributes", src: src, want: `<a y="1"><b/>t<!--c--><?p d?><c><b/></c></a>`, sheet: specXML(`
<xsl:mode on-no-match="shallow-copy"/>
<xsl:template match="@x"><xsl:attribute name="y" select="."/></xsl:template>`)},
		{name: "shallow-copy keeps namespaces", src: `<p:a xmlns:p="urn:p"><p:b/></p:a>`, want: `<p:a xmlns:p="urn:p"><p:b/></p:a>`, sheet: specXML(`
<xsl:mode on-no-match="shallow-copy"/>`)},
		{name: "deep-copy", src: src, want: `<A><a x="1"><b/>t<!--c--><?p d?><c><b/></c></a></A>`, sheet: specXML(`
<xsl:mode on-no-match="deep-copy"/>
<xsl:template match="/"><A><xsl:apply-templates/></A></xsl:template>
<xsl:template match="b"><B/></xsl:template>`)},
		{name: "shallow-skip", src: src, want: `<B/><B/>`, sheet: specXML(`
<xsl:mode on-no-match="shallow-skip"/>
<xsl:template match="b"><B/></xsl:template>`)},
		{name: "shallow-skip visits attributes", src: src, want: `x`, sheet: `
<xsl:mode on-no-match="shallow-skip"/>
<xsl:template match="@*"><xsl:value-of select="name()"/></xsl:template>`},
		// The document node's children are still processed under deep-skip.
		{name: "deep-skip", src: src, want: `<A><B/></A>`, sheet: specXML(`
<xsl:mode on-no-match="deep-skip"/>
<xsl:template match="a"><A><xsl:apply-templates/></A></xsl:template>
<xsl:template match="b"><B/></xsl:template>`)},
		{name: "text-only-copy", src: src, want: `t1`, sheet: `
<xsl:template match="a"><xsl:apply-templates select="node(), @x"/></xsl:template>`},
		{name: "fail", src: src, code: "XTDE0555", sheet: `
<xsl:mode on-no-match="fail"/>
<xsl:template match="/"><xsl:apply-templates/></xsl:template>`},
		{name: "fail can be caught", src: src, want: `caught`, sheet: `
<xsl:mode on-no-match="fail"/>
<xsl:template match="/"><xsl:try><xsl:apply-templates/><xsl:catch errors="err:XTDE0555" xmlns:err="http://www.w3.org/2005/xqt-errors">caught</xsl:catch></xsl:try></xsl:template>`},
		// Arrays: the built-in rule applies templates to the members.
		{name: "array members under text-only-copy", want: `1|2|3|`, sheet: `
<xsl:template match="/"><xsl:apply-templates select="[1, [2, 3]]"/></xsl:template>
<xsl:template match=".[. instance of xs:integer]"><xsl:value-of select="."/>|</xsl:template>`},
		{name: "array under deep-skip", want: `[]`, sheet: `
<xsl:mode on-no-match="deep-skip"/>
<xsl:template match="/">[<xsl:apply-templates select="[1, 2]"/>]</xsl:template>
<xsl:template match=".[. instance of xs:integer]"><xsl:value-of select="."/></xsl:template>`},
		{name: "array under fail", code: "XTDE0555", sheet: `
<xsl:mode on-no-match="fail"/>
<xsl:template match="/"><xsl:apply-templates select="[1, 2]"/></xsl:template>`},
		{name: "atomic values under text-only-copy", want: `ab`, sheet: `
<xsl:template match="/"><xsl:apply-templates select="'a', 'b'"/></xsl:template>`},
		// §6.6: on-multiple-match="fail" — two rules, same precedence and priority.
		{name: "on-multiple-match fail", src: src, code: "XTDE0540", sheet: `
<xsl:mode on-multiple-match="fail"/>
<xsl:template match="a">1</xsl:template>
<xsl:template match="*" priority="0">2</xsl:template>
<xsl:template match="/"><xsl:apply-templates select="a"/></xsl:template>`},
		// Otherwise the last in declaration order is chosen.
		{name: "on-multiple-match use-last", src: src, want: `2`, sheet: `
<xsl:template match="/"><xsl:apply-templates select="a"/></xsl:template>
<xsl:template match="a">1</xsl:template>
<xsl:template match="a">2</xsl:template>`},
		// §6.? XTSE0545: conflicting xsl:mode declarations at the same precedence.
		{name: "conflicting mode declarations", code: "XTSE0545", sheet: `
<xsl:mode name="m" on-no-match="deep-skip"/>
<xsl:mode name="m" on-no-match="shallow-copy"/>`},
		{name: "repeated identical mode declarations", want: `ok`, sheet: `
<xsl:mode name="m" on-no-match="deep-skip"/>
<xsl:mode name="m" on-no-match="deep-skip"/>
<xsl:template match="/">ok</xsl:template>`},
	})
}

// XSLT 3.0 §6.6 modes: named modes, #all, #default, #current, #unnamed and
// default-mode.
func TestModeSpecSelection(t *testing.T) {
	src := `<r><a/><b/></r>`
	specRunAll(t, "", []specCase{
		{name: "named mode and #current", src: src, want: `m:a[m:b]`, sheet: `
<xsl:template match="/"><xsl:apply-templates select="r/a" mode="m"/></xsl:template>
<xsl:template match="a" mode="m">m:a[<xsl:apply-templates select="../b" mode="#current"/>]</xsl:template>
<xsl:template match="b" mode="m">m:b</xsl:template>
<xsl:template match="b">default:b</xsl:template>`},
		{name: "#all", src: src, want: `all all`, sheet: `
<xsl:template match="/"><xsl:apply-templates select="r/a" mode="x"/><xsl:text> </xsl:text><xsl:apply-templates select="r/a"/></xsl:template>
<xsl:template match="a" mode="#all">all</xsl:template>`},
		{name: "#default and named in one list", src: src, want: `ab`, sheet: `
<xsl:template match="/"><xsl:apply-templates select="r/a" mode="x"/><xsl:apply-templates select="r/b"/></xsl:template>
<xsl:template match="a | b" mode="x #default"><xsl:value-of select="name()"/></xsl:template>`},
		{name: "#unnamed", src: src, want: `a`, sheet: `
<xsl:template match="/"><xsl:apply-templates select="r/a" mode="#unnamed"/></xsl:template>
<xsl:template match="a" mode="#unnamed">a</xsl:template>`},
		// §3.7.? default-mode on an element makes it the default for the subtree.
		{name: "default-mode", src: src, want: `M`, sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` default-mode="m"><xsl:output method="text"/>
<xsl:template match="/" mode="#unnamed"><xsl:apply-templates select="r/a"/></xsl:template>
<xsl:template match="a">M</xsl:template>
<xsl:template match="a" mode="#unnamed">U</xsl:template></xsl:stylesheet>`},
		{name: "prefixed mode names compare by namespace", src: src, want: `ok`, sheet: `
<xsl:template match="/" xmlns:g="urn:f"><xsl:apply-templates select="r/a" mode="g:m"/></xsl:template>
<xsl:template match="a" mode="f:m">ok</xsl:template>`},
		{name: "mode-specific built-in falls back to text", src: `<r>t<a>u</a></r>`, want: `tu`, sheet: `
<xsl:template match="/"><xsl:apply-templates mode="q"/></xsl:template>`},
		// §3.5.? XTSE3085: with declared-modes (the package default) every mode
		// used must be declared.
		{name: "undeclared mode in a package", code: "XTSE3085", sheet: `<xsl:package version="3.0" name="urn:p" ` + specNS + `>
<xsl:mode/><xsl:template match="/"><xsl:apply-templates mode="undeclared"/></xsl:template></xsl:package>`},
		{name: "declared mode in a package", want: `ok`, sheet: `<xsl:package version="3.0" name="urn:p" ` + specNS + `><xsl:output method="text"/>
<xsl:mode/><xsl:mode name="m"/><xsl:template match="/"><xsl:apply-templates select="*" mode="m"/></xsl:template><xsl:template match="*" mode="m">ok</xsl:template></xsl:package>`},
		{name: "declared-modes=no", want: `ok`, sheet: `<xsl:package version="3.0" name="urn:p" declared-modes="no" ` + specNS + `><xsl:output method="text"/>
<xsl:template match="/"><xsl:apply-templates select="*" mode="m"/></xsl:template><xsl:template match="*" mode="m">ok</xsl:template></xsl:package>`},
	})
}

// §2.3.3: with no initial mode supplied, the initial mode is the
// default-mode of the top-level package.
func TestModeSpecInitialModeFromDefaultMode(t *testing.T) {
	specCheck(t, "", specCase{name: "initial mode from default-mode", src: `<r><a/></r>`, want: "M", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` default-mode="m"><xsl:output method="text"/>
<xsl:template match="/"><xsl:apply-templates select="r/a"/></xsl:template>
<xsl:template match="a">M</xsl:template></xsl:stylesheet>`})
}
