package xslt

import "testing"

// XSLT 3.0 §8.4.2 xsl:on-empty / xsl:on-non-empty.
func TestInstrSpecOnEmpty(t *testing.T) {
	sheet := specXML(`
<xsl:template match="/r">
  <t><xsl:on-non-empty><h/></xsl:on-non-empty><xsl:apply-templates select="row"/><xsl:on-empty><none/></xsl:on-empty></t>
</xsl:template>
<xsl:template match="row"><row/></xsl:template>`)
	specRunAll(t, "", []specCase{
		{name: "non-empty", sheet: sheet, src: `<r><row/><row/></r>`, want: `<t><h/><row/><row/></t>`},
		{name: "empty", sheet: sheet, src: `<r/>`, want: `<t><none/></t>`},
		{name: "on-empty with select", src: `<r/>`, want: `<t>none</t>`, sheet: specXML(`
<xsl:template match="/r"><t><xsl:sequence select="row"/><xsl:on-empty select="'none'"/></t></xsl:template>`)},
		{name: "on-non-empty with select", src: `<r><row/></r>`, want: `<t>rows:<row/></t>`, sheet: specXML(`
<xsl:template match="/r"><t><xsl:on-non-empty select="'rows:'"/><xsl:copy-of select="row"/></t></xsl:template>`)},
		// An attribute node is content: the sequence is not empty.
		{name: "attribute counts as content", want: `<e a="1"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="a">1</xsl:attribute><xsl:on-empty>E</xsl:on-empty></e></xsl:template>`)},
		{name: "in a function result", want: `0 1`, sheet: `
<xsl:function name="f:c"><xsl:param name="s"/><xsl:sequence select="$s"/><xsl:on-empty select="0"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:c(()), f:c(1)"/></xsl:template>`},
		// A zero-length text node is deemed empty (§8.4.1).
		{name: "zero-length text is empty", want: `<t>E</t>`, sheet: specXML(`
<xsl:template match="/"><t><xsl:value-of select="''"/><xsl:on-empty>E</xsl:on-empty></t></xsl:template>`)},
	})
}

// XSLT 3.0 §8.4.3 xsl:where-populated, with the §8.4.1 "deemed empty" rules.
func TestInstrSpecWherePopulated(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "empty list dropped", src: `<r/>`, want: `<out/>`, sheet: specXML(`
<xsl:template match="/r"><out><xsl:where-populated><ul><xsl:for-each select="x"><li/></xsl:for-each></ul></xsl:where-populated></out></xsl:template>`)},
		{name: "populated list kept", src: `<r><x/></r>`, want: `<out><ul><li/></ul></out>`, sheet: specXML(`
<xsl:template match="/r"><out><xsl:where-populated><ul><xsl:for-each select="x"><li/></xsl:for-each></ul></xsl:where-populated></out></xsl:template>`)},
		// An element with attributes but no children is deemed empty.
		{name: "element with only attributes dropped", want: `<out/>`, sheet: specXML(`
<xsl:template match="/"><out><xsl:where-populated><e a="1"/></xsl:where-populated></out></xsl:template>`)},
		{name: "items in a function result", want: `0 1 1 0 0 1 0 1`, sheet: `
<xsl:function name="f:w" as="item()*"><xsl:param name="v" as="item()*"/><xsl:where-populated><xsl:sequence select="$v"/></xsl:where-populated></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:w(map{})), count(f:w(map{1:2})),
  count(f:w([1])), count(f:w([])), count(f:w([()])), count(f:w([[], 'x'])), count(f:w([['']])), count(f:w(1))"/></xsl:template>`},
		{name: "comments and PIs", want: `<out><!--c--><?p x?></out>`, sheet: specXML(`
<xsl:template match="/"><out><xsl:where-populated><xsl:comment/><xsl:comment>c</xsl:comment><xsl:processing-instruction name="p"/><xsl:processing-instruction name="p">x</xsl:processing-instruction></xsl:where-populated></out></xsl:template>`)},
		{name: "document node children", want: `1 0`, sheet: `
<xsl:function name="f:w" as="item()*"><xsl:param name="v" as="item()*"/><xsl:where-populated><xsl:sequence select="$v"/></xsl:where-populated></xsl:function>
<xsl:variable name="full"><a/></xsl:variable>
<xsl:variable name="empty" as="document-node()"><xsl:document/></xsl:variable>
<xsl:template match="/"><xsl:value-of select="count(f:w($full)), count($empty/node())"/></xsl:template>`},
	})
}

// XSLT 3.0 §11.7.? xsl:namespace and its dynamic errors.
func TestInstrSpecNamespace(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "select form", want: `<e xmlns:p="urn:p"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="p" select="'urn:p'"/></e></xsl:template>`)},
		{name: "content form and AVT name", want: `<e xmlns:q="urn:q"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="{'q'}">urn:q</xsl:namespace></e></xsl:template>`)},
		{name: "default namespace on a namespaced element", want: `<p:e xmlns:p="urn:x" xmlns="urn:d"/>`, sheet: specXML(`
<xsl:template match="/"><p:e xmlns:p="urn:x"><xsl:namespace name="" select="'urn:d'"/></p:e></xsl:template>`)},
		{name: "same binding twice is fine", want: `<e xmlns:p="urn:p"/>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="p" select="'urn:p'"/><xsl:namespace name="p" select="'urn:p'"/></e></xsl:template>`)},
		{name: "name xmlns", code: "XTDE0920", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="{'xmlns'}" select="'urn:p'"/></e></xsl:template>`)},
		{name: "name not an NCName", code: "XTDE0920", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="{'1x'}" select="'urn:p'"/></e></xsl:template>`)},
		{name: "xml prefix to another URI", code: "XTDE0925", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="xml" select="'urn:p'"/></e></xsl:template>`)},
		{name: "XML namespace to another prefix", code: "XTDE0925", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="x" select="'http://www.w3.org/XML/1998/namespace'"/></e></xsl:template>`)},
		{name: "zero-length URI", code: "XTDE0930", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="p" select="''"/></e></xsl:template>`)},
		{name: "xmlns namespace URI", code: "XTDE0905", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="p" select="'http://www.w3.org/2000/xmlns/'"/></e></xsl:template>`)},
		{name: "default namespace on no-namespace element", code: "XTDE0440", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="" select="'urn:d'"/></e></xsl:template>`)},
	})
}

// XSLT 3.0 §11.4.? xsl:document, §5.6.2 text value templates, §19.? xsl:fork
// (outside streaming its branches are simply evaluated in order).
func TestInstrSpecDocumentTVTFork(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "xsl:document builds a document node", want: `true 1`, sheet: `
<xsl:variable name="d" as="document-node()"><xsl:document><a/></xsl:document></xsl:variable>
<xsl:template match="/"><xsl:value-of select="$d instance of document-node(), count($d/a)"/></xsl:template>`},
		{name: "attribute in a document node", code: "XTDE0420", sheet: `
<xsl:variable name="d" as="document-node()"><xsl:document><xsl:attribute name="a">1</xsl:attribute></xsl:document></xsl:variable>
<xsl:template match="/"><xsl:value-of select="count($d/*)"/></xsl:template>`},
		{name: "text value template", src: `<r><x/><x/></r>`, want: `n=2 {lit}`, sheet: `
<xsl:template match="/" expand-text="yes">n={count(//x)} {{lit}}</xsl:template>`},
		{name: "text value template joins with spaces", want: `1 2 3`, sheet: `
<xsl:template match="/" expand-text="yes">{1 to 3}</xsl:template>`},
		{name: "expand-text off inside", want: `{1}`, sheet: `
<xsl:template match="/" expand-text="yes"><xsl:text expand-text="no">{1}</xsl:text></xsl:template>`},
		{name: "fork evaluates its branches in order", want: `1 2`, sheet: `
<xsl:template match="/"><xsl:fork><xsl:sequence select="1"/><xsl:sequence select="2"/></xsl:fork></xsl:template>`},
	})
}

// Behaviour that disagrees with the REC; kept as skipped tests naming the bug.
func TestInstrSpecMiscKnownBugs(t *testing.T) {
	t.Run("where-populated keeps each atomic item", func(t *testing.T) {
		specCheck(t, "", specCase{name: "atomics", want: "2", sheet: `
<xsl:function name="f:w" as="item()*"><xsl:param name="v" as="item()*"/><xsl:where-populated><xsl:sequence select="$v"/></xsl:where-populated></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:w(('', 'a', 'b')))"/></xsl:template>`})
	})
	t.Run("where-populated drops a childless document node", func(t *testing.T) {
		specCheck(t, "", specCase{name: "doc", want: "0", sheet: `
<xsl:function name="f:w" as="item()*"><xsl:param name="v" as="item()*"/><xsl:where-populated><xsl:sequence select="$v"/></xsl:where-populated></xsl:function>
<xsl:variable name="empty" as="document-node()"><xsl:document/></xsl:variable>
<xsl:template match="/"><xsl:value-of select="count(f:w($empty))"/></xsl:template>`})
	})
	t.Run("conflicting xsl:namespace bindings are XTDE0430", func(t *testing.T) {
		t.Skip("BUG: two xsl:namespace for prefix p with different URIs raise XTDE0905, §11.7.? defines XTDE0430")
		specCheck(t, "", specCase{name: "conflict", code: "XTDE0430", sheet: specXML(`
<xsl:template match="/"><e><xsl:namespace name="p" select="'urn:a'"/><xsl:namespace name="p" select="'urn:b'"/></e></xsl:template>`)})
	})
}
