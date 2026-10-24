package xslt

import "testing"

// XSLT 3.0 §10.4 dynamic XPath evaluation (xsl:evaluate).
func TestInstrSpecEvaluate(t *testing.T) {
	ns := `<r xmlns:p="urn:p"><p:a>v</p:a></r>`
	specRunAll(t, "", []specCase{
		{name: "with-params map", want: "10", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'$x * 2'" with-params="map{QName('', 'x'): 5}"/></xsl:template>`},
		{name: "with-param children", want: "2", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'$p + 1'"><xsl:with-param name="p" select="1"/></xsl:evaluate></xsl:template>`},
		{name: "with-params not a map", code: "XPTY0004", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'1'" with-params="1"/></xsl:template>`},
		{name: "with-params key not a QName", code: "XPTY0004", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'1'" with-params="map{'x': 1}"/></xsl:template>`},
		{name: "atomic context item", want: "6", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'. + 1'" context-item="5"/></xsl:template>`},
		{name: "node context item", src: `<r><a>x</a></r>`, want: "x", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'string(a)'" context-item="/r"/></xsl:template>`},
		// An empty context-item means the focus is absent.
		{name: "empty context item", code: "XPDY0002", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'name(.)'" context-item="()"/></xsl:template>`},
		// XTTE3210: context-item with more than one item.
		{name: "two context items", code: "XTTE3210", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'.'" context-item="1, 2"/></xsl:template>`},
		// The namespace-context node supplies the in-scope namespaces.
		{name: "namespace-context prefixes", src: ns, want: "v", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'string(p:a)'" context-item="/r" namespace-context="/r"/></xsl:template>`},
		// ... and its default namespace is the default element namespace.
		{name: "namespace-context default namespace", src: `<r xmlns="urn:d"><a>w</a></r>`, want: "w", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'string(a)'" context-item="/*" namespace-context="/*"/></xsl:template>`},
		{name: "namespace-context two nodes", src: `<r><a/><a/></r>`, code: "XTTE3170", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'1'" namespace-context="//a"/></xsl:template>`},
		{name: "base-uri", want: "http://example.com/x/", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'string(static-base-uri())'" base-uri="http://example.com/x/"/></xsl:template>`},
		{name: "as converts the result", want: "true", sheet: `
<xsl:template match="/"><xsl:variable name="v"><xsl:evaluate xpath="'3'" as="xs:double"/></xsl:variable><xsl:value-of select="$v = '3'"/></xsl:template>`},
		// XTTE3165: the result does not match @as.
		{name: "as mismatch", code: "XTTE3165", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'(1, 2)'" as="xs:integer"/></xsl:template>`},
		// XTDE3160: a static error in the evaluated expression, including the
		// XSLT functions §10.4.1 keeps out of its static context.
		{name: "current() unavailable", code: "XTDE3160", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'current()'" context-item="."/></xsl:template>`},
		{name: "system-property unavailable", code: "XTDE3160", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'system-property(&quot;xsl:version&quot;)'"/></xsl:template>`},
		{name: "attribute result into text", src: `<r a="7"/>`, want: "7", sheet: `
<xsl:template match="/"><xsl:evaluate xpath="'/r/@a'" context-item="."/></xsl:template>`},
		{name: "attribute result onto an element", src: `<r a="7"/>`, want: `<e a="7"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:evaluate xpath="'/r/@a'" context-item="."/></e></xsl:template>`)},
		{name: "public function is available", want: "4", sheet: `
<xsl:function name="f:d" visibility="public"><xsl:param name="n"/><xsl:sequence select="$n * 2"/></xsl:function>
<xsl:template match="/"><xsl:evaluate xpath="'f:d(2)'"/></xsl:template>`},
	})
}
