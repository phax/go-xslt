package xslt

import "testing"

// XSLT 3.0 §12.4 format tokens beyond the Latin ones (F&O 3.1 §4.6.1: any
// token in a numbering sequence other than decimal digits, a/A, i/I and w
// generates that sequence), English numeric ordinals, and grouping with other
// digit families.
func TestNumberSpecTokens(t *testing.T) {
	specRunAll(t, "", []specCase{
		numSpec("Greek alphabetic", `<xsl:number value="1, 2, 3" format="&#x3B1;"/>`, "α.β.γ"),
		numSpec("circled digits", `<xsl:number value="3" format="&#x2460;"/>`, "③"),
		// English ordinal suffixes, including the teens.
		numSpec("numeric ordinals", `<xsl:for-each select="1, 2, 3, 4, 11, 12, 13, 21, 22, 23, 101, 111"><xsl:number value="." ordinal="yes"/><xsl:text> </xsl:text></xsl:for-each>`,
			"1st 2nd 3rd 4th 11th 12th 13th 21st 22nd 23rd 101st 111th "),
		numSpec("ordinal with zero padding", `<xsl:number value="2" format="01" ordinal="yes"/>`, "02nd"),
		numSpec("grouping in another digit family", `<xsl:number value="1234" format="&#x660;" grouping-separator="," grouping-size="3"/>`, "١,٢٣٤"),
		numSpec("roman beyond the classic range", `<xsl:number value="0, 4000" format="i"/>`, "0.4000"),
		numSpec("alphabetic zero", `<xsl:number value="0" format="a"/>`, "0"),
		numSpec("big integer with a padded token", `<xsl:number value="xs:integer('123456789012345678901')" format="0000000000000000000000001"/>`, "0000123456789012345678901"),
		numSpec("big integer with an alphabetic token", `<xsl:number value="xs:integer('123456789012345678901')" format="a"/>`, "123456789012345678901"),
		numSpec("ordinal words in German", `<xsl:number value="1" format="w" ordinal="yes" lang="de"/>`, "erste"),
		// The from pattern bounds the search for the counted ancestor.
		{name: "count beyond the from boundary", src: `<r><a><b/></a></r>`, want: "[]", sheet: `
<xsl:template match="/">[<xsl:for-each select="//b"><xsl:number count="r" from="a"/></xsl:for-each>]</xsl:template>`},
		{name: "level any with from on an ancestor", src: `<r><s><n/><n/></s><s><n/></s></r>`, want: "1 2 1 ", sheet: `
<xsl:template match="/"><xsl:for-each select="//n"><xsl:number level="any" from="s"/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`},
		{name: "level multiple default count", src: `<r><a><a/><a><a/></a></a></r>`, want: "1 1.1 1.2 1.2.1 ", sheet: `
<xsl:template match="/"><xsl:for-each select="//a"><xsl:number level="multiple"/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`},
	})
}
