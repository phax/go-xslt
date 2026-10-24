package xslt

import "testing"

// Static errors of the XSLT 3.0 REC, one case per rule (Appendix E lists the
// codes; the section is named per case).
func TestCompileSpecStaticErrors(t *testing.T) {
	tpl := `<xsl:template match="/">x</xsl:template>`
	specRunAll(t, "", []specCase{
		// §14.? / §15.?: grouping and merge functions in patterns.
		{name: "current-group in a pattern", code: "XTSE1060", sheet: `<xsl:template match="a[current-group()]">x</xsl:template>`},
		{name: "current-grouping-key in a pattern", code: "XTSE1070", sheet: `<xsl:template match="a[current-grouping-key()]">x</xsl:template>`},
		{name: "current-merge-group in a pattern", code: "XTSE3470", sheet: `<xsl:template match="a[current-merge-group()]">x</xsl:template>`},
		{name: "current-merge-key in a pattern", code: "XTSE3500", sheet: `<xsl:template match="a[current-merge-key()]">x</xsl:template>`},
		{name: "current-group in xsl:number count", code: "XTSE1060", sheet: `<xsl:template match="/"><xsl:number count="a[current-group()]"/></xsl:template>`},
		// §20.? XPST0017: an unknown function in a match pattern.
		{name: "unknown function in a pattern", code: "XPST0017", sheet: `<xsl:template match="a[f:nope()]">x</xsl:template>`},
		// §3.? XTSE0130: a top-level element in no namespace.
		{name: "top-level element in no namespace", code: "XTSE0130", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `><foo/>` + tpl + `</xsl:stylesheet>`},
		// §10.1 XTSE0660: two named templates of one name and precedence.
		{name: "duplicate named template", code: "XTSE0660", sheet: `<xsl:template name="t">1</xsl:template><xsl:template name="t">2</xsl:template>`},
		// §10.3 XTSE0770: duplicate function.
		{name: "duplicate function", code: "XTSE0770", sheet: `<xsl:function name="f:a"><xsl:sequence select="1"/></xsl:function><xsl:function name="f:a"><xsl:sequence select="2"/></xsl:function>` + tpl},
		// §9.? XTSE0630: duplicate global variable.
		{name: "duplicate global variable", code: "XTSE0630", sheet: `<xsl:variable name="v" select="1"/><xsl:variable name="v" select="2"/>` + tpl},
		// §9.? XPST0008: a global variable that refers to itself.
		{name: "self-referencing global", code: "XPST0008", sheet: `<xsl:variable name="v" select="$v"/>` + tpl},
		// §6.4 XTSE0530: invalid priority.
		{name: "invalid priority", code: "XTSE0530", sheet: `<xsl:template match="a" priority="high">x</xsl:template>`},
		// §9.2 XTSE0010: required="yes" together with a default value.
		{name: "required param with select", code: "XTSE0010", sheet: `<xsl:template name="t"><xsl:param name="p" required="yes" select="1"/></xsl:template>`},
		// §8.4.2 XTSE0010: xsl:on-empty must be last.
		{name: "on-empty not last", code: "XTSE0010", sheet: `<xsl:template match="/"><xsl:on-empty>e</xsl:on-empty><x/></xsl:template>`},
		// §6.? XTSE0010: xsl:context-item outside xsl:template.
		{name: "context-item in for-each", code: "XTSE0010", sheet: `<xsl:template match="/"><xsl:for-each select="."><xsl:context-item/></xsl:for-each></xsl:template>`},
		// §11.9.2 copy-accumulators must be a boolean.
		{name: "copy-accumulators not a boolean", code: "XTSE0020", sheet: `<xsl:template match="/"><xsl:copy-of select="." copy-accumulators="maybe"/></xsl:template>`},
		// §20.2.1 xsl:key.
		{name: "key with use and content", code: "XTSE1205", sheet: `<xsl:key name="k" match="a" use="@b"><xsl:sequence select="@c"/></xsl:key>` + tpl},
		{name: "key with neither use nor content", code: "XTSE1205", sheet: `<xsl:key name="k" match="a"/>` + tpl},
		{name: "key with an unknown collation", code: "XTSE1210", sheet: `<xsl:key name="k" match="a" use="@b" collation="http://example.com/nope"/>` + tpl},
		{name: "keys with different collations", code: "XTSE1220", sheet: `<xsl:key name="k" match="a" use="@b" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/>
<xsl:key name="k" match="b" use="@b" collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"/>` + tpl},
		{name: "keys with different composite", code: "XTSE1222", sheet: `<xsl:key name="k" match="a" use="@b" composite="yes"/><xsl:key name="k" match="b" use="@b"/>` + tpl},
		// §10.3 XTSE0080 / XTSE0760.
		{name: "function in a reserved namespace", code: "XTSE0080", sheet: `<xsl:function name="fn:mine"><xsl:sequence select="1"/></xsl:function>` + tpl},
		{name: "function param with select", code: "XTSE0760", sheet: `<xsl:function name="f:a"><xsl:param name="p" select="1"/><xsl:sequence select="$p"/></xsl:function>` + tpl},
		{name: "function param with content", code: "XTSE0760", sheet: `<xsl:function name="f:a"><xsl:param name="p">1</xsl:param><xsl:sequence select="$p"/></xsl:function>` + tpl},
		// §17.1 XTSE1130.
		{name: "analyze-string without children", code: "XTSE1130", sheet: `<xsl:template match="/"><xsl:analyze-string select="'a'" regex="a"/></xsl:template>`},
		// §13.1 xsl:sort.
		{name: "sort with an invalid lang", code: "XTSE0020", sheet: `<xsl:template match="/"><xsl:for-each select="*"><xsl:sort select="." lang="not a language"/></xsl:for-each></xsl:template>`},
		{name: "stable on a second sort", code: "XTSE1017", sheet: `<xsl:template match="/"><xsl:for-each select="*"><xsl:sort select="."/><xsl:sort select="." stable="yes"/></xsl:for-each></xsl:template>`},
		// §4.? xsl:strip-space / xsl:preserve-space.
		{name: "unbound strip-space prefix", code: "XTSE0280", sheet: `<xsl:strip-space elements="zz:a"/>` + tpl},
		{name: "unbound strip-space prefix wildcard", code: "XTSE0280", sheet: `<xsl:strip-space elements="zz:*"/>` + tpl},
		{name: "strip and preserve the same element", code: "XTSE0270", sheet: `<xsl:strip-space elements="a"/><xsl:preserve-space elements="a"/>` + tpl},
		// §4.4 xsl:decimal-format.
		{name: "zero-digit not a zero", code: "XTSE1295", sheet: `<xsl:decimal-format name="d" zero-digit="1"/>` + tpl},
		{name: "decimal-format symbols not distinct", code: "XTSE1300", sheet: `<xsl:decimal-format name="d" decimal-separator="," grouping-separator=","/>` + tpl},
		{name: "decimal-format symbol in the digit family", code: "XTSE1300", sheet: `<xsl:decimal-format name="d" percent="5"/>` + tpl},
		{name: "conflicting decimal-formats", code: "XTSE1290", sheet: `<xsl:decimal-format name="d" decimal-separator=","/><xsl:decimal-format name="d" decimal-separator=";"/>` + tpl},
		// §26.? xsl:character-map.
		{name: "character map references an unknown map", code: "XTSE1590", sheet: `<xsl:character-map name="a" use-character-maps="nope"/>` + tpl},
		// §11.1.4 xsl:namespace-alias.
		{name: "namespace-alias with an unbound prefix", code: "XTSE0812", sheet: `<xsl:namespace-alias stylesheet-prefix="zz" result-prefix="f"/>` + tpl},
		{name: "conflicting namespace aliases", code: "XTSE0810", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` xmlns:a="urn:a" xmlns:b="urn:b">
<xsl:namespace-alias stylesheet-prefix="f" result-prefix="a"/><xsl:namespace-alias stylesheet-prefix="f" result-prefix="b"/>` + tpl + `</xsl:stylesheet>`},
	})
}

// Declarations that compile and take effect (positive counterparts).
func TestCompileSpecDeclarations(t *testing.T) {
	specRunAll(t, "", []specCase{
		// §11.1.4: the REC's own use of namespace-alias, generating a stylesheet.
		{name: "namespace-alias", want: `<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"/>`, sheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:axsl="urn:alias">
<xsl:output method="xml" omit-xml-declaration="yes"/>
<xsl:namespace-alias stylesheet-prefix="axsl" result-prefix="xsl"/>
<xsl:template match="/"><axsl:stylesheet version="3.0"/></xsl:template></xsl:stylesheet>`},
		{name: "namespace-alias to the null namespace", want: `<e/>`, sheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:a="urn:alias">
<xsl:output method="xml" omit-xml-declaration="yes"/>
<xsl:namespace-alias stylesheet-prefix="a" result-prefix="#default"/>
<xsl:template match="/"><a:e/></xsl:template></xsl:stylesheet>`},
		// §4.4 + F&O 3.1 §4.7 format-number with a named decimal format.
		{name: "named decimal-format", want: "1.234,50", sheet: `<xsl:decimal-format name="de" decimal-separator="," grouping-separator="."/>
<xsl:template match="/"><xsl:value-of select="format-number(1234.5, '#.##0,00', 'de')"/></xsl:template>`},
		{name: "decimal-format special symbols", want: "50% -1 NaN! inf", sheet: `<xsl:decimal-format name="s" percent="%" minus-sign="-" NaN="NaN!" infinity="inf"/>
<xsl:template match="/"><xsl:value-of select="format-number(0.5, '0%', 's'), format-number(-1, '0', 's'), format-number(number('x'), '0', 's'), format-number(1 div 0e0, '0', 's')"/></xsl:template>`},
		{name: "repeated identical decimal-formats", want: "1,5", sheet: `<xsl:decimal-format name="d" decimal-separator="," grouping-separator="."/><xsl:decimal-format name="d" decimal-separator="," grouping-separator="."/>
<xsl:template match="/"><xsl:value-of select="format-number(1.5, '0,0', 'd')"/></xsl:template>`},
		// §4.3 whitespace stripping.
		{name: "strip-space and preserve-space", src: "<r> <p> </p> <q> </q></r>", want: "1", sheet: `<xsl:strip-space elements="*"/><xsl:preserve-space elements="p"/>
<xsl:template match="/"><xsl:value-of select="count(//text())"/></xsl:template>`},
		{name: "strip-space by namespace wildcard", src: `<r xmlns:p="urn:f"><p:a> </p:a><b> </b></r>`, want: "1", sheet: `<xsl:strip-space elements="f:*"/>
<xsl:template match="/"><xsl:value-of select="count(//text())"/></xsl:template>`},
		{name: "xml:space preserve wins over strip-space", src: `<r><a xml:space="preserve"> </a><b> </b></r>`, want: "1", sheet: `<xsl:strip-space elements="*"/>
<xsl:template match="/"><xsl:value-of select="count(//text())"/></xsl:template>`},
		{name: "key with a sequence constructor", src: `<r><a b="1">x</a></r>`, want: "x", sheet: `<xsl:key name="k" match="a"><xsl:sequence select="@b"/></xsl:key>
<xsl:template match="/"><xsl:value-of select="key('k', '1')"/></xsl:template>`},
		{name: "same function name, different arity", want: "1 2", sheet: `<xsl:function name="f:a"><xsl:sequence select="1"/></xsl:function><xsl:function name="f:a"><xsl:param name="x"/><xsl:sequence select="$x"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:a(), f:a(2)"/></xsl:template>`},
	})
}
