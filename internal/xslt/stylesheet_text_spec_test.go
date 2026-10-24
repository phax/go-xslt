package xslt

import "testing"

// XSLT 3.0 §4.3 stripping whitespace and comments from the stylesheet, §5.6.2
// text value templates, and local variable scoping (§9.?).
func TestCompileSpecStylesheetText(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "xml:space preserve keeps whitespace text", want: " 1 ", sheet: `
<xsl:template match="/" xml:space="preserve"> <xsl:value-of select="1"/> </xsl:template>`},
		// Whitespace before xsl:sort is removed even under xml:space="preserve".
		{name: "whitespace before xsl:sort", want: "|1|2", sheet: `
<xsl:template match="/"><xsl:for-each select="2, 1" xml:space="preserve">
  <xsl:sort select="."/>|<xsl:value-of select="."/></xsl:for-each></xsl:template>`},
		// ... and whitespace children of xsl:choose.
		{name: "whitespace inside xsl:choose", want: "y", sheet: `
<xsl:template match="/" xml:space="preserve"><xsl:choose>
  <xsl:when test="true()">y</xsl:when>
</xsl:choose></xsl:template>`},
		{name: "whitespace before xsl:param", want: "p", sheet: `
<xsl:template match="/"><xsl:call-template name="t"/></xsl:template>
<xsl:template name="t" xml:space="preserve">
  <xsl:param name="p" select="'p'"/><xsl:value-of select="$p"/></xsl:template>`},
		// Comments are removed first, so the texts around one are adjacent.
		{name: "text split by a comment", want: "abcd", sheet: `
<xsl:template match="/">ab<!--c-->cd</xsl:template>`},
		{name: "text value template split by a comment", want: "a2b", sheet: `
<xsl:template match="/" expand-text="yes">a{1<!--c-->+1}b</xsl:template>`},
		{name: "whitespace after a comment joins the text", want: "x ", sheet: `
<xsl:template match="/">x<!--c--> </xsl:template>`},
		{name: "processing instruction in the stylesheet", want: "xy", sheet: `
<xsl:template match="/">x<?pi data?>y</xsl:template>`},
		// Local variables may shadow earlier ones; a reference sees the binding in scope.
		{name: "shadowing local variables", want: "1 2", sheet: `
<xsl:template match="/"><xsl:variable name="x" select="1"/><xsl:variable name="y" select="$x"/><xsl:variable name="x" select="2"/><xsl:value-of select="$y, $x"/></xsl:template>`},
		{name: "variable aliasing a sequence", want: "2", sheet: `
<xsl:template match="/"><xsl:variable name="b" select="(1, 2)"/><xsl:variable name="a" select="$b"/><xsl:value-of select="count($a)"/></xsl:template>`},
		{name: "alias used inside a nested instruction", want: "x x", sheet: `
<xsl:template match="/"><xsl:variable name="b" select="'x'"/><xsl:variable name="a" select="$b"/><xsl:for-each select="1 to 2"><xsl:value-of select="$a"/><xsl:if test="position() lt last()"><xsl:text> </xsl:text></xsl:if></xsl:for-each></xsl:template>`},
		{name: "variable with an as and a sequence constructor", want: "3", sheet: `
<xsl:template match="/"><xsl:variable name="n" as="xs:integer*"><xsl:sequence select="1, 2"/><xsl:sequence select="3"/></xsl:variable><xsl:value-of select="count($n)"/></xsl:template>`},
		// A variable with content and no @as is a document node.
		{name: "temporary tree", want: "true 2", sheet: `
<xsl:template match="/"><xsl:variable name="t"><a/><b/></xsl:variable><xsl:value-of select="$t instance of document-node(), count($t/*)"/></xsl:template>`},
		{name: "empty variable is a zero-length string", want: "[]", sheet: `
<xsl:template match="/"><xsl:variable name="e"/>[<xsl:value-of select="$e"/>]</xsl:template>`},
	})
}

// XSLT 3.0 §11.1.3 / §11.9: namespace inheritance and copy-namespaces.
func TestConstructSpecNamespaces(t *testing.T) {
	src := `<p:a xmlns:p="urn:p" xmlns:q="urn:q"><q:b/><c/></p:a>`
	specRunAll(t, "", []specCase{
		{name: "copy-namespaces no on xsl:copy", src: src, want: `<p:a xmlns:p="urn:p"><e/></p:a>`, sheet: specXML(`
<xsl:template match="/*"><xsl:copy copy-namespaces="no"><e/></xsl:copy></xsl:template>`)},
		{name: "copy-of copy-namespaces no", src: src, want: `<q:b xmlns:q="urn:q"/><c/>`, sheet: specXML(`
<xsl:template match="/*"><xsl:copy-of select="*" copy-namespaces="no"/></xsl:template>`)},
		// inherit-namespaces="no": the child does not inherit the parent's namespaces.
		{name: "xsl:element inherit-namespaces no", want: `0`, sheet: `
<xsl:template match="/"><xsl:variable name="v"><xsl:element name="r" inherit-namespaces="no"><xsl:namespace name="x" select="'urn:x'"/><c/></xsl:element></xsl:variable><xsl:value-of select="count(in-scope-prefixes($v/r/c)[. = 'x'])"/></xsl:template>`},
		{name: "xsl:copy inherit-namespaces no", src: src, want: `0`, sheet: `
<xsl:template match="/*"><xsl:variable name="v"><xsl:copy inherit-namespaces="no"><c/></xsl:copy></xsl:variable><xsl:value-of select="count(in-scope-prefixes($v/*/c)[. = 'q'])"/></xsl:template>`},
		{name: "exclude-result-prefixes", want: `<r/>`, sheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:x="urn:x" exclude-result-prefixes="x">
<xsl:output method="xml" omit-xml-declaration="yes"/><xsl:template match="/"><r/></xsl:template></xsl:stylesheet>`},
		{name: "exclude-result-prefixes #all", want: `<r/>`, sheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:x="urn:x" xmlns:y="urn:y">
<xsl:output method="xml" omit-xml-declaration="yes"/><xsl:template match="/"><r xsl:exclude-result-prefixes="#all"/></xsl:template></xsl:stylesheet>`},
		{name: "used namespace is not excluded", want: `<x:r xmlns:x="urn:x"/>`, sheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:x="urn:x" exclude-result-prefixes="x">
<xsl:output method="xml" omit-xml-declaration="yes"/><xsl:template match="/"><x:r/></xsl:template></xsl:stylesheet>`},
		// XTSE0808: exclude-result-prefixes names an undeclared prefix.
		{name: "exclude an undeclared prefix", code: "XTSE0808", sheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="nope">
<xsl:template match="/"><r/></xsl:template></xsl:stylesheet>`},
	})
}

// Former BUG: entries (§11.1.3, §11.9.1), now checked against the REC.
func TestConstructSpecNamespacesFixed(t *testing.T) {
	t.Run("literal result element inherit-namespaces", func(t *testing.T) {
		specCheck(t, "", specCase{name: "literal result element inherit-namespaces no", want: `x|`, sheet: `
<xsl:template match="/"><xsl:variable name="v"><r xmlns:x="urn:x" xsl:inherit-namespaces="no"><xsl:element name="c"/></r></xsl:variable><xsl:value-of select="in-scope-prefixes($v/r)[. = 'x'], in-scope-prefixes($v/r/c)[. = 'x']" separator="|"/>|</xsl:template>`})
	})
	specCheck(t, "", specCase{name: "copy-namespaces scope", src: `<p:a xmlns:p="urn:p" xmlns:q="urn:q"><q:b/><c/></p:a>`,
		want: `<p:a xmlns:p="urn:p"><q:b xmlns:q="urn:q"/><c xmlns:q="urn:q"/></p:a>`, sheet: specXML(`
<xsl:template match="/*"><xsl:copy copy-namespaces="no"><xsl:copy-of select="*"/></xsl:copy></xsl:template>`)})
}
