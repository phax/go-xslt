package xslt

import "testing"

// Cases where the engine disagrees with the XSLT 3.0 REC. Each is skipped
// with the bug it demonstrates; remove the Skip once the engine is fixed.
func TestSpecKnownBugs(t *testing.T) {
	bugs := []struct {
		bug string
		c   specCase
	}{
		{"BUG: a static error in xsl:evaluate's expression has no error code; §10.4.? defines XTDE3160",
			specCase{name: "evaluate syntax error", code: "XTDE3160", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'1 +'"/></xsl:template>`}},
		{"BUG: an AVT-valued xsl:result-document/@escape-uri-attributes is never evaluated, so an invalid value is not SEPM0016",
			specCase{name: "result-document escape-uri-attributes AVT", code: "SEPM0016", sheet: `
<xsl:template match="/"><xsl:result-document href="a" escape-uri-attributes="{'sometimes'}">1</xsl:result-document></xsl:template>`}},
		{"BUG: an unknown XSLT instruction outside forwards-compatible mode has no error code; §3.10 / XTSE0010",
			specCase{name: "unknown instruction", code: "XTSE0010", sheet: `
<xsl:template match="/"><xsl:future-thing><xsl:fallback>fb</xsl:fallback></xsl:future-thing></xsl:template>`}},
		{"BUG: a required stylesheet parameter that is not supplied raises XTDE0700 (template parameters); XSLT 3.0 §9.? defines XTDE0050 for stylesheet parameters",
			specCase{name: "required stylesheet parameter", code: "XTDE0050", sheet: `
<xsl:param name="p" required="yes"/>
<xsl:template match="/"><xsl:value-of select="$p"/></xsl:template>`}},
		{"BUG: an attribute constructed after a child node raises XTDE0420 (document node rule); §5.7.1 defines XTDE0410",
			specCase{name: "attribute after child", code: "XTDE0410", sheet: specXML(`
<xsl:template match="/"><e><c/><xsl:attribute name="a">v</xsl:attribute></e></xsl:template>`)}},
		{"BUG: xsl:strip-space elements=\"1bad\" (not a NameTest) is accepted silently; it is an invalid attribute value, XTSE0020",
			specCase{name: "strip-space invalid name test", code: "XTSE0020", sheet: `<xsl:strip-space elements="1bad"/><xsl:template match="/">x</xsl:template>`}},
		{"BUG: a call to an undeclared function in a stylesheet expression fails without an error code; XPath 3.1 §3.1.5 requires the static error XPST0017",
			specCase{name: "unknown function", code: "XPST0017", sheet: `<xsl:template match="/"><xsl:value-of select="f:nope()"/></xsl:template>`}},
	}
	for _, b := range bugs {
		t.Run(b.c.name, func(t *testing.T) {
			t.Skip(b.bug)
			specCheck(t, "", b.c)
		})
	}
}

// Former BUG: entries, now checked against the REC.
func TestSpecFixedBugs(t *testing.T) {
	specRunAll(t, "", []specCase{
		// §5.3.4: the global context item must match
		// xsl:global-context-item/@as (XTTE0590), whichever entry point runs.
		{name: "global-context-item via TransformFull", code: "XTTE0590", sheet: `
<xsl:global-context-item as="element()"/><xsl:template match="/">x</xsl:template>`},
		{name: "global-context-item matching via TransformFull", want: "x", sheet: `
<xsl:global-context-item as="document-node()"/><xsl:template match="/">x</xsl:template>`},
		// §11.9.1: xsl:copy of a namespace node returns a copy of it.
		{name: "copy namespace node", src: `<p:a xmlns:p="urn:p" xmlns:q="urn:q"/>`, want: `<e xmlns:q="urn:q"/>`, sheet: specXML(`
<xsl:template match="/*"><e><xsl:for-each select="namespace::q"><xsl:copy/></xsl:for-each></e></xsl:template>`)},
		// §11.3: "If the attribute is in a non-null namespace and no prefix is
		// specified, then the namespace fixup process will invent a prefix."
		{name: "attribute prefix fixup", want: `true`, sheet: `
<xsl:template match="/"><xsl:variable name="e"><e><xsl:attribute name="a" namespace="urn:x">v</xsl:attribute></e></xsl:variable><xsl:value-of select="contains(name($e/e/@*), ':')"/></xsl:template>`},
		{name: "attribute prefix fixup reuses a bound prefix", want: `x:a`, sheet: `
<xsl:template match="/"><xsl:variable name="e"><e xmlns:x="urn:x"><xsl:attribute name="a" namespace="urn:x">v</xsl:attribute></e></xsl:variable><xsl:value-of select="name($e/e/@*)"/></xsl:template>`},
	})
}
