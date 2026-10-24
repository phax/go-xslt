package xslt

import "testing"

// XSLT 3.0 §13 sorting: xsl:sort in xsl:for-each / xsl:apply-templates /
// xsl:perform-sort, with data-type, order, case-order, collation, lang and
// typed sort keys.
func TestSortSpec(t *testing.T) {
	src := `<r><i n="10" s="b">x</i><i n="9" s="A">y</i><i n="100" s="a">z</i><i n="9" s="B">w</i></r>`
	each := func(sorts string) string {
		return `<xsl:template match="/"><xsl:for-each select="r/i">` + sorts + `<xsl:value-of select="@n"/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`
	}
	specRunAll(t, "", []specCase{
		{name: "text order by default", src: src, want: "10 100 9 9 ", sheet: each(`<xsl:sort select="@n"/>`)},
		{name: "number order", src: src, want: "9 9 10 100 ", sheet: each(`<xsl:sort select="@n" data-type="number"/>`)},
		{name: "descending", src: src, want: "100 10 9 9 ", sheet: each(`<xsl:sort select="@n" data-type="number" order="descending"/>`)},
		{name: "AVT data-type and order", src: src, want: "100 10 9 9 ", sheet: each(`<xsl:sort select="@n" data-type="{'number'}" order="{'descending'}"/>`)},
		// Stable: equal keys stay in document order (y before w).
		{name: "stable for equal keys", src: src, want: "y w x z ", sheet: `<xsl:template match="/"><xsl:for-each select="r/i"><xsl:sort select="@n" data-type="number"/><xsl:sort select="0" data-type="number"/><xsl:value-of select="."/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`},
		{name: "second key breaks ties", src: src, want: "w y x z ", sheet: `<xsl:template match="/"><xsl:for-each select="r/i"><xsl:sort select="@n" data-type="number"/><xsl:sort select="."/><xsl:value-of select="."/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`},
		// Codepoint collation: upper-case letters precede lower-case.
		{name: "codepoint collation", src: src, want: "A B a b ", sheet: `<xsl:template match="/"><xsl:for-each select="r/i"><xsl:sort select="@s" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/><xsl:value-of select="@s"/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`},
		// HTML ASCII case-insensitive collation treats A and a as equal: document order decides.
		{name: "case-insensitive collation", src: src, want: "A a b B ", sheet: `<xsl:template match="/"><xsl:for-each select="r/i"><xsl:sort select="@s" collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"/><xsl:value-of select="@s"/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`},
		// XTDE1035: an unknown collation.
		{name: "unknown collation", src: src, code: "XTDE1035", sheet: each(`<xsl:sort select="@s" collation="http://example.com/no-such-collation"/>`)},
		{name: "invalid lang", src: src, code: "XTDE0030", sheet: each(`<xsl:sort select="@s" lang="{'not a language'}"/>`)},
		{name: "sort body instead of select", src: src, want: "9 9 10 100 ", sheet: each(`<xsl:sort data-type="number"><xsl:value-of select="@n"/></xsl:sort>`)},
		{name: "sort body xsl:sequence", src: src, want: "9 9 10 100 ", sheet: each(`<xsl:sort><xsl:sequence select="xs:integer(@n)"/></xsl:sort>`)},
		{name: "sort without select uses the string value", src: src, want: "w x y z ", sheet: `<xsl:template match="/"><xsl:for-each select="r/i"><xsl:sort/><xsl:value-of select="."/><xsl:text> </xsl:text></xsl:for-each></xsl:template>`},
		// Typed sort keys compare by their type, not as strings.
		{name: "typed numeric keys", want: "1.5 2 3 10", sheet: `<xsl:template match="/"><xsl:perform-sort select="(3, 10, 1.5, 2)"><xsl:sort select="."/></xsl:perform-sort></xsl:template>`},
		{name: "perform-sort of dates", want: "2001-01-01 2020-05-01 2026-10-24", sheet: `<xsl:template match="/"><xsl:perform-sort select="(xs:date('2026-10-24'), xs:date('2001-01-01'), xs:date('2020-05-01'))"><xsl:sort select="."/></xsl:perform-sort></xsl:template>`},
		{name: "empty sort key sorts first", src: `<r><i k="b"/><i/><i k="a"/></r>`, want: "-ab", sheet: `<xsl:template match="/"><xsl:for-each select="r/i"><xsl:sort select="@k"/><xsl:value-of select="if (@k) then @k else '-'"/></xsl:for-each></xsl:template>`},
		{name: "NaN sorts first with number", src: `<r><i n="2"/><i n="x"/><i n="1"/></r>`, want: "x12", sheet: `<xsl:template match="/"><xsl:for-each select="r/i"><xsl:sort select="@n" data-type="number"/><xsl:value-of select="@n"/></xsl:for-each></xsl:template>`},
		// XTDE1030: sort key values that are not comparable.
		{name: "incomparable typed keys", code: "XTDE1030", sheet: `<xsl:template match="/"><xsl:perform-sort select="(1, 'a')"><xsl:sort select="."/></xsl:perform-sort></xsl:template>`},
		// XTTE1020: a sort key of more than one item.
		{name: "sort key sequence", src: src, code: "XTTE1020", sheet: each(`<xsl:sort select="(@n, @s)"/>`)},
		{name: "apply-templates with sort", src: src, want: "zxyw", sheet: `<xsl:template match="/"><xsl:apply-templates select="r/i"><xsl:sort select="@n" data-type="number" order="descending"/></xsl:apply-templates></xsl:template>
<xsl:template match="i"><xsl:value-of select="."/></xsl:template>`},
		// XTSE1015: a sort key with both select and content.
		{name: "select and content", src: src, code: "XTSE1015", sheet: each(`<xsl:sort select="@n">x</xsl:sort>`)},
		{name: "bad stable", src: src, code: "XTSE0020", sheet: each(`<xsl:sort select="@n" stable="maybe"/>`)},
	})
}
