package xslt

import "testing"

// XSLT 3.0 §11.9.1 xsl:copy (shallow copy) for every node kind.
func TestConstructSpecCopy(t *testing.T) {
	src := `<p:a xmlns:p="urn:p" xmlns:q="urn:q" x="1"><b>t</b><!--c--><?pi v?></p:a>`
	specRunAll(t, "", []specCase{
		{name: "element without content", src: src, want: `<p:a xmlns:p="urn:p" xmlns:q="urn:q"/>`, sheet: specXML(`
<xsl:template match="/*"><xsl:copy/></xsl:template>`)},
		{name: "element with attributes and children copied by the body", src: src, want: `<p:a xmlns:p="urn:p" xmlns:q="urn:q" x="1"><b>t</b><!--c--><?pi v?></p:a>`, sheet: specXML(`
<xsl:template match="/*"><xsl:copy><xsl:for-each select="@*, node()"><xsl:copy><xsl:apply-templates select="node()"/></xsl:copy></xsl:for-each></xsl:copy></xsl:template>
<xsl:template match="text()"><xsl:copy/></xsl:template>`)},
		// copy-namespaces="no" drops namespaces the element does not use.
		{name: "copy-namespaces no", src: src, want: `<p:a xmlns:p="urn:p"/>`, sheet: specXML(`
<xsl:template match="/*"><xsl:copy copy-namespaces="no"/></xsl:template>`)},
		{name: "use-attribute-sets", src: src, want: `<p:a xmlns:p="urn:p" xmlns:q="urn:q" s="v"/>`, sheet: specXML(`
<xsl:attribute-set name="as"><xsl:attribute name="s">v</xsl:attribute></xsl:attribute-set>
<xsl:template match="/*"><xsl:copy use-attribute-sets="as"/></xsl:template>`)},
		{name: "document node", src: src, want: `<r/>`, sheet: specXML(`
<xsl:template match="/"><xsl:copy><r/></xsl:copy></xsl:template>`)},
		{name: "document node kept as a node in a sequence", src: src, want: `true 1`, sheet: `
<xsl:template match="/"><xsl:variable name="d" as="document-node()"><xsl:copy><r/></xsl:copy></xsl:variable><xsl:value-of select="$d instance of document-node(), count($d/r)"/></xsl:template>`},
		{name: "attribute in a document node", src: src, code: "XTDE0420", sheet: specXML(`
<xsl:template match="/"><xsl:copy><xsl:attribute name="a">1</xsl:attribute></xsl:copy></xsl:template>`)},
		// §11.9.1: select (XSLT 3.0).
		{name: "select", src: src, want: `<b xmlns:p="urn:p" xmlns:q="urn:q"/>`, sheet: specXML(`
<xsl:template match="/"><xsl:copy select="//b"/></xsl:template>`)},
		{name: "select empty", src: src, want: `<r/>`, sheet: specXML(`
<xsl:template match="/"><r><xsl:copy select="//nothing"/></r></xsl:template>`)},
		{name: "select an atomic value", src: src, want: `<r>5</r>`, sheet: specXML(`
<xsl:template match="/"><r><xsl:copy select="5"/></r></xsl:template>`)},
		// XTTE3180: select yields more than one item.
		{name: "select two items", src: src, code: "XTTE3180", sheet: specXML(`
<xsl:template match="/"><xsl:copy select="//b, //b"/></xsl:template>`)},
		// XPDY0002: no context item.
		{name: "absent focus", src: src, code: "XPDY0002", sheet: specXML(`
<xsl:function name="f:c"><xsl:copy/></xsl:function>
<xsl:template match="/"><xsl:sequence select="f:c()"/></xsl:template>`)},
		// XTDE0410: an attribute after a child.
		{name: "attribute copied after a child", src: src, code: "XTDE0410", sheet: specXML(`
<xsl:template match="/*"><e><c/><xsl:for-each select="@x"><xsl:copy/></xsl:for-each></e></xsl:template>`)},
		{name: "comment and PI", src: src, want: `<e><!--c--><?pi v?></e>`, sheet: specXML(`
<xsl:template match="/*"><e><xsl:for-each select="comment(), processing-instruction()"><xsl:copy/></xsl:for-each></e></xsl:template>`)},
		{name: "copy-of copy-namespaces no", src: src, want: `<b>t</b>`, sheet: specXML(`
<xsl:template match="/*"><xsl:copy-of select="b" copy-namespaces="no"/></xsl:template>`)},
	})
}

// XSLT 3.0 §11.2 xsl:element and §11.3 xsl:attribute, with their dynamic
// errors.
func TestConstructSpecElementAttribute(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "computed element in a namespace", want: `<p:e xmlns:p="urn:p"/>`, sheet: specXML(`
<xsl:template match="/"><xsl:element name="p:e" namespace="urn:p"/></xsl:template>`)},
		{name: "computed element, prefix from scope", want: `<f:e xmlns:f="urn:f"/>`, sheet: specXML(`
<xsl:template match="/"><xsl:element name="f:e"/></xsl:template>`)},
		{name: "computed element, no prefix, namespace", want: `<e xmlns="urn:d"/>`, sheet: specXML(`
<xsl:template match="/"><xsl:element name="e" namespace="urn:d"/></xsl:template>`)},
		{name: "element use-attribute-sets", want: `<e a="1"/>`, sheet: specXML(`
<xsl:attribute-set name="s"><xsl:attribute name="a">1</xsl:attribute></xsl:attribute-set>
<xsl:template match="/"><xsl:element name="e" use-attribute-sets="s"/></xsl:template>`)},
		// XTDE0820: name is not a lexical QName.
		{name: "invalid element name", code: "XTDE0820", sheet: specXML(`
<xsl:template match="/"><xsl:element name="{'1e'}"/></xsl:template>`)},
		// XTDE0830: prefix with no namespace binding.
		{name: "unbound element prefix", code: "XTDE0830", sheet: specXML(`
<xsl:template match="/"><xsl:element name="{'zz:e'}"/></xsl:template>`)},
		// XTDE0835: namespace attribute not a valid URI.
		{name: "invalid element namespace", code: "XTDE0835", sheet: specXML(`
<xsl:template match="/"><xsl:element name="e" namespace="{'http://www.w3.org/2000/xmlns/'}"/></xsl:template>`)},
		{name: "attribute with select and separator", want: `<e a="1,2,3"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="a" select="1 to 3" separator=","/></e></xsl:template>`)},
		{name: "attribute select default separator", want: `<e a="1 2 3"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="a" select="1 to 3"/></e></xsl:template>`)},
		{name: "attribute in a namespace with a prefix", want: `<e xmlns:p="urn:p" p:a="v"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="p:a" namespace="urn:p">v</xsl:attribute></e></xsl:template>`)},
		{name: "attribute replaces one of the same name", want: `<e a="2"/>`, sheet: specXML(`
<xsl:template match="/"><e a="1"><xsl:attribute name="a">2</xsl:attribute></e></xsl:template>`)},
		// XTDE0850: not a lexical QName.
		{name: "invalid attribute name", code: "XTDE0850", sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="{'a b'}">v</xsl:attribute></e></xsl:template>`)},
		// XTDE0855: the name xmlns.
		{name: "attribute named xmlns", code: "XTDE0855", sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="{'xmlns'}">v</xsl:attribute></e></xsl:template>`)},
		// XTDE0860: unbound prefix.
		{name: "unbound attribute prefix", code: "XTDE0860", sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="{'zz:a'}">v</xsl:attribute></e></xsl:template>`)},
		// XTDE0865: invalid namespace URI.
		{name: "invalid attribute namespace", code: "XTDE0865", sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="a" namespace="{'http://www.w3.org/2000/xmlns/'}">v</xsl:attribute></e></xsl:template>`)},
		// XTDE0420: attribute in a document node.
		{name: "attribute in a document", code: "XTDE0420", sheet: specXML(`
<xsl:variable name="d" as="document-node()"><xsl:document><xsl:attribute name="a">v</xsl:attribute></xsl:document></xsl:variable>
<xsl:template match="/"><xsl:copy-of select="$d"/></xsl:template>`)},
		{name: "standalone attribute node", want: `a=v`, sheet: `
<xsl:template match="/"><xsl:variable name="a" as="attribute()"><xsl:attribute name="a">v</xsl:attribute></xsl:variable><xsl:value-of select="name($a), '=', string($a)" separator=""/></xsl:template>`},
		{name: "processing instruction", want: `<?target data?>`, sheet: specXML(`
<xsl:template match="/"><xsl:processing-instruction name="target" select="'data'"/></xsl:template>`)},
		// XTDE0890: PI name not an NCName or "xml".
		{name: "PI named xml", code: "XTDE0890", sheet: specXML(`
<xsl:template match="/"><xsl:processing-instruction name="{'xml'}"/></xsl:template>`)},
		{name: "comment with select", want: `<!--a b-->`, sheet: specXML(`
<xsl:template match="/"><xsl:comment select="'a', 'b'"/></xsl:template>`)},
	})
}
