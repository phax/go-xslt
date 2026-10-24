package xslt

import "testing"

// XSLT 3.0 §10.3 stylesheet functions: the result is the sequence the body
// constructs (no tree is built), converted to @as by the function conversion
// rules.
func TestXSLTFuncConstructedResults(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "empty body result is the empty sequence", want: "0", sheet: `
<xsl:function name="f:e"><xsl:copy-of select="()"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:e())"/></xsl:template>`},
		{name: "literal text constructs a text node", want: "true", sheet: `
<xsl:function name="f:t">abc</xsl:function>
<xsl:template match="/"><xsl:value-of select="f:t() instance of text()"/></xsl:template>`},
		{name: "two atomic items stay two items", want: "2", sheet: `
<xsl:function name="f:s"><xsl:sequence select="1"/><xsl:sequence select="2"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:s())"/></xsl:template>`},
		{name: "typed atomic value keeps its type", want: "true true", sheet: `
<xsl:function name="f:b"><xsl:try select="false()"><xsl:catch select="true()"/></xsl:try></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:b() instance of xs:boolean, not(f:b())"/></xsl:template>`},
		{name: "choose body cast to the declared type", want: "5 true", sheet: `
<xsl:function name="f:c" as="xs:integer"><xsl:param name="b"/><xsl:choose><xsl:when test="$b"><xsl:sequence select="5"/></xsl:when><xsl:otherwise><xsl:value-of select="6"/></xsl:otherwise></xsl:choose></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:c(true()), f:c(false()) instance of xs:integer"/></xsl:template>`},
		{name: "text content cast to the declared type", want: "7", sheet: `
<xsl:function name="f:c" as="xs:integer"><xsl:if test="true()"><xsl:text>7</xsl:text></xsl:if></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:c()"/></xsl:template>`},
		{name: "elements are returned as nodes", want: "2 a b", sheet: `
<xsl:function name="f:n"><a/><b/></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:n()), f:n() ! name()"/></xsl:template>`},
		{name: "attribute nodes are returned as nodes", want: "x=1", sheet: `
<xsl:function name="f:a"><xsl:attribute name="x">1</xsl:attribute></xsl:function>
<xsl:template match="/"><xsl:value-of select="name(f:a()) || '=' || f:a()"/></xsl:template>`},
		{name: "source nodes are returned by identity", src: `<r><a/></r>`, want: "true", sheet: `
<xsl:function name="f:id"><xsl:param name="n"/><xsl:if test="true()"><xsl:sequence select="$n"/></xsl:if></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:id(/r/a) is /r/a"/></xsl:template>`},
		{name: "cached function", want: "6 6", sheet: `
<xsl:function name="f:m" cache="yes"><xsl:param name="n"/><xsl:sequence select="$n * 2"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:m(3), f:m(3)"/></xsl:template>`},
		{name: "cached function with node arguments", src: `<r><a>1</a><a>2</a></r>`, want: "1 2", sheet: `
<xsl:function name="f:m" new-each-time="no"><xsl:param name="n"/><xsl:sequence select="string($n)"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:m(/r/a[1]), f:m(/r/a[2])"/></xsl:template>`},
		{name: "braced EQName call", want: "4", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` xmlns:app="http://app.example/">
<xsl:output method="text"/>
<xsl:function name="app:f"><xsl:param name="n"/><xsl:sequence select="$n + 1"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="Q{http://app.example/}f(3)"/></xsl:template></xsl:stylesheet>`},
	})
}

// fn:copy-of / fn:snapshot and namespaces (XSLT 3.0 §20.?).
func TestXSLTFuncCopyNamespaces(t *testing.T) {
	src := `<p:a xmlns:p="urn:p" xmlns:q="urn:q" x="1"><c/></p:a>`
	specRunAll(t, "", []specCase{
		{name: "copy-of keeps inherited namespaces", src: src, want: "p q xml", sheet: `
<xsl:template match="/"><xsl:value-of select="sort(in-scope-prefixes(copy-of(/*/c)))"/></xsl:template>`},
		{name: "snapshot of an attribute", src: src, want: "p:a 1", sheet: `
<xsl:template match="/"><xsl:variable name="s" select="snapshot(/*/@x)"/><xsl:value-of select="name($s/..), string($s)"/></xsl:template>`},
		{name: "snapshot of a namespace node", src: src, want: "p:a urn:q", sheet: `
<xsl:template match="/"><xsl:variable name="s" select="snapshot(/*/namespace::q)"/><xsl:value-of select="name($s/..), string($s)"/></xsl:template>`},
		{name: "snapshot of a text node", src: `<r><a>t</a></r>`, want: "a r", sheet: `
<xsl:template match="/"><xsl:variable name="s" select="snapshot(/r/a/text())"/><xsl:value-of select="name($s/..), name($s/../..)"/></xsl:template>`},
		{name: "snapshot nodes have distinct ids", src: `<r><a/></r>`, want: "true", sheet: `
<xsl:template match="/"><xsl:variable name="s" select="snapshot(/r/a)"/><xsl:value-of select="generate-id($s) ne generate-id($s/..)"/></xsl:template>`},
	})
}
