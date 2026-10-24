package xslt

import "testing"

// XSLT 3.0 §6.? xsl:context-item: static constraints and dynamic checks.
func TestCompileSpecContextItem(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "as satisfied", src: `<r/>`, want: "r", sheet: `
<xsl:template match="/"><xsl:apply-templates select="r"/></xsl:template>
<xsl:template match="r"><xsl:context-item as="element()" use="required"/><xsl:value-of select="name()"/></xsl:template>`},
		{name: "optional and absent context item", want: "ok", sheet: `
<xsl:function name="f:t"><xsl:call-template name="t"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:t()"/></xsl:template>
<xsl:template name="t"><xsl:context-item use="optional"/>ok</xsl:template>`},
		// XTTE3090: required context item absent.
		{name: "required but absent", code: "XTTE3090", sheet: `
<xsl:function name="f:t"><xsl:call-template name="t"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:t()"/></xsl:template>
<xsl:template name="t"><xsl:context-item use="required"/>x</xsl:template>`},
		// XTTE0590: context item does not match @as.
		{name: "type mismatch", code: "XTTE0590", sheet: `
<xsl:template match="/"><xsl:call-template name="t"/></xsl:template>
<xsl:template name="t"><xsl:context-item as="element()"/>x</xsl:template>`},
		{name: "atomic context item against as", want: "5", sheet: `
<xsl:template match="/"><xsl:for-each select="5"><xsl:call-template name="t"/></xsl:for-each></xsl:template>
<xsl:template name="t"><xsl:context-item as="xs:integer"/><xsl:value-of select=". "/></xsl:template>`},
		// use="absent": the body has no focus, so "." is XPDY0002.
		{name: "absent focus", code: "XPDY0002", sheet: `
<xsl:template match="/"><xsl:call-template name="t"/></xsl:template>
<xsl:template name="t"><xsl:context-item use="absent"/><xsl:value-of select="name(.)"/></xsl:template>`},
		{name: "absent focus body that ignores it", want: "fine", sheet: `
<xsl:template match="/"><xsl:call-template name="t"/></xsl:template>
<xsl:template name="t"><xsl:context-item use="absent"/>fine</xsl:template>`},
		{name: "two declarations", code: "XTSE0010", sheet: `
<xsl:template name="t"><xsl:context-item/><xsl:context-item/></xsl:template>`},
		{name: "not the first child", code: "XTSE0010", sheet: `
<xsl:template name="t"><xsl:param name="p"/><xsl:context-item/></xsl:template>`},
		{name: "text before it", code: "XTSE0010", sheet: `
<xsl:template name="t">text<xsl:context-item/></xsl:template>`},
		{name: "unknown attribute", code: "XTSE0090", sheet: `
<xsl:template name="t"><xsl:context-item select="."/></xsl:template>`},
		{name: "bad use", code: "XTSE0020", sheet: `
<xsl:template name="t"><xsl:context-item use="sometimes"/></xsl:template>`},
		{name: "as with use=absent", code: "XTSE3088", sheet: `
<xsl:template name="t"><xsl:context-item use="absent" as="node()"/></xsl:template>`},
	})
}

// XSLT 3.0 §3.10 forwards-compatible processing and §24.2 extension
// instructions with xsl:fallback.
func TestCompileSpecForwardsCompatible(t *testing.T) {
	fwd := func(body string) string {
		return `<xsl:stylesheet version="4.0" ` + specNS + `><xsl:output method="text"/>` + body + `</xsl:stylesheet>`
	}
	specRunAll(t, "", []specCase{
		{name: "unknown instruction with fallback", want: "fb1fb2", sheet: fwd(`
<xsl:template match="/"><xsl:future-thing><xsl:fallback>fb1</xsl:fallback><xsl:fallback>fb2</xsl:fallback></xsl:future-thing></xsl:template>`)},
		// Not an error unless the instruction is actually evaluated.
		{name: "unknown instruction not evaluated", want: "ok", sheet: fwd(`
<xsl:template match="/"><xsl:if test="false()"><xsl:future-thing/></xsl:if>ok</xsl:template>`)},
		{name: "unknown instruction evaluated without fallback", code: "XTDE1425", sheet: fwd(`
<xsl:template match="/"><xsl:future-thing/></xsl:template>`)},
		{name: "unknown top-level declaration ignored", want: "ok", sheet: fwd(`
<xsl:future-declaration/><xsl:template match="/">ok</xsl:template>`)},
		{name: "unknown attribute ignored", want: "ok", sheet: fwd(`
<xsl:template match="/" future-attr="1">ok</xsl:template>`)},
		// Without forwards-compatible mode these are static errors.
		{name: "unknown declaration in 3.0 mode", code: "XTSE0010", sheet: `
<xsl:future-declaration/><xsl:template match="/">ok</xsl:template>`},
		{name: "instruction at top level", code: "XTSE0010", sheet: `
<xsl:value-of select="1"/>`},
		{name: "expose outside a package", code: "XTSE0010", sheet: `
<xsl:expose component="template" names="*" visibility="public"/>`},
		{name: "local version attribute opens forwards-compatible scope", want: "fb", sheet: `
<xsl:template match="/"><xsl:future-thing version="5.0"><xsl:fallback>fb</xsl:fallback></xsl:future-thing></xsl:template>`},
		// §24.2: extension instructions.
		{name: "extension instruction falls back", want: "fb", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` xmlns:ext="urn:ext" extension-element-prefixes="ext"><xsl:output method="text"/>
<xsl:template match="/"><ext:do><xsl:fallback>fb</xsl:fallback></ext:do></xsl:template></xsl:stylesheet>`},
		{name: "extension instruction without fallback", code: "XTDE1450", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` xmlns:ext="urn:ext" extension-element-prefixes="ext"><xsl:output method="text"/>
<xsl:template match="/"><ext:do/></xsl:template></xsl:stylesheet>`},
		{name: "extension instruction never evaluated", want: "ok", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` xmlns:ext="urn:ext" extension-element-prefixes="ext"><xsl:output method="text"/>
<xsl:template match="/"><xsl:if test="false()"><ext:do/></xsl:if>ok</xsl:template></xsl:stylesheet>`},
		{name: "element-available of an unknown extension", want: "false", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` xmlns:ext="urn:ext" extension-element-prefixes="ext"><xsl:output method="text"/>
<xsl:template match="/"><xsl:value-of select="element-available('ext:do')"/></xsl:template></xsl:stylesheet>`},
	})
}

// XSLT 3.0 §3.5.1: xsl:package attributes package-version and declared-modes.
func TestCompileSpecPackageAttributes(t *testing.T) {
	pkg := func(attrs string) string {
		return `<xsl:package version="3.0" name="urn:p" ` + attrs + ` ` + specNS + `><xsl:output method="text"/><xsl:mode/><xsl:template match="/">ok</xsl:template></xsl:package>`
	}
	specRunAll(t, "", []specCase{
		{name: "valid package-version", sheet: pkg(`package-version="1.0.2-alpha"`), want: "ok"},
		{name: "numeric package-version", sheet: pkg(`package-version="2"`), want: "ok"},
		{name: "declared-modes no", sheet: pkg(`declared-modes="no"`), want: "ok"},
		{name: "package-version with a non-numeric part", sheet: pkg(`package-version="1.x"`), code: "XTSE0020"},
		{name: "package-version with an empty part", sheet: pkg(`package-version="1..2"`), code: "XTSE0020"},
		{name: "package-version with a bad suffix", sheet: pkg(`package-version="1.0-9a"`), code: "XTSE0020"},
		{name: "empty package-version", sheet: pkg(`package-version=""`), code: "XTSE0020"},
		{name: "declared-modes not a boolean", sheet: pkg(`declared-modes="maybe"`), code: "XTSE0020"},
	})
}
