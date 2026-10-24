package xslt

import "testing"

// XSLT 3.0 §23.1 xsl:message content (constructed like the content of a
// document node), §8.4.3 xsl:where-populated inside element content, and
// §6.8 xsl:next-match / xsl:apply-imports with parameters.
func TestInstrSpecMessageContent(t *testing.T) {
	cases := []struct {
		name, body, src string
		want            []string
	}{
		{"element content is serialized", `<xsl:message><a x="1"/></xsl:message>`, "", []string{`<a x="1"/>`}},
		{"attribute node is atomized", `<xsl:message select="/r/@a"/>`, `<r a="v"/>`, []string{"v"}},
		{"select and body together", `<xsl:message select="'s'">b</xsl:message>`, "", []string{"sb"}},
	}
	for _, c := range cases {
		src := c.src
		if src == "" {
			src = "<doc/>"
		}
		rr, err := specRunIn("", `<xsl:template match="/">`+c.body+`</xsl:template>`, src)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if len(rr.Messages) != len(c.want) || (len(c.want) > 0 && rr.Messages[0] != c.want[0]) {
			t.Errorf("%s: messages %q, want %q", c.name, rr.Messages, c.want)
		}
	}
	specRunAll(t, "", []specCase{
		{name: "terminating message value in the catch", want: "x", sheet: `
<xsl:template match="/"><xsl:try><xsl:message terminate="yes"><m>x</m></xsl:message><xsl:catch xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="string-join($err:value ! string())"/></xsl:catch></xsl:try></xsl:template>`},
		{name: "where-populated keeps a populated document", want: `<e><n/></e>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:where-populated><xsl:document><n/></xsl:document></xsl:where-populated></e></xsl:template>`)},
		{name: "where-populated joins surviving atomic values", want: `<e>a b</e>`, sheet: specXML(`
<xsl:template match="/"><e><xsl:where-populated><xsl:sequence select="'a', '', 'b'"/></xsl:where-populated></e></xsl:template>`)},
	})
}

func TestApplyImportsSpecParams(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "base.xsl", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:template match="a"><xsl:param name="p" select="'none'"/><xsl:param name="t" tunnel="yes" select="'none'"/>base(<xsl:value-of select="$p, $t"/>)</xsl:template>
</xsl:stylesheet>`)
	specRunAll(t, dir, []specCase{
		{name: "apply-imports with parameters", src: `<r><a/></r>`, want: "main[base(P T)]", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `><xsl:import href="base.xsl"/><xsl:output method="text"/>
<xsl:template match="/"><xsl:apply-templates select="r/a"/></xsl:template>
<xsl:template match="a">main[<xsl:apply-imports><xsl:with-param name="p" select="'P'"/><xsl:with-param name="t" select="'T'" tunnel="yes"/></xsl:apply-imports>]</xsl:template></xsl:stylesheet>`},
		{name: "next-match with parameters", src: `<r><a/></r>`, want: "1[2(P)]", sheet: `
<xsl:template match="/"><xsl:apply-templates select="r/a"/></xsl:template>
<xsl:template match="a" priority="2">1[<xsl:next-match><xsl:with-param name="p" select="'P'"/></xsl:next-match>]</xsl:template>
<xsl:template match="a" priority="1"><xsl:param name="p"/>2(<xsl:value-of select="$p"/>)</xsl:template>`},
		{name: "next-match falls back to the built-in rule", src: `<r><a>text</a></r>`, want: "[text]", sheet: `
<xsl:template match="/"><xsl:apply-templates select="r/a"/></xsl:template>
<xsl:template match="a">[<xsl:next-match/>]</xsl:template>`},
		{name: "apply-imports without imports uses the built-in rule", src: `<r><a>t</a></r>`, want: "(t)", sheet: `
<xsl:template match="/"><xsl:apply-templates select="r/a"/></xsl:template>
<xsl:template match="a">(<xsl:apply-imports/>)</xsl:template>`},
		// XTDE0560: apply-imports / next-match with no current template rule.
		{name: "apply-imports inside for-each", src: `<r><a/></r>`, code: "XTDE0560", sheet: `
<xsl:template match="/"><xsl:for-each select="r"><xsl:apply-imports/></xsl:for-each></xsl:template>`},
	})
}

func TestApplyImportsSpecKnownBugs(t *testing.T) {
	t.Skip("BUG: xsl:next-match inside a stylesheet function keeps the caller's current template rule; it is absent in a function (§6.8), so XTDE0560")
	specCheck(t, "", specCase{name: "next-match in a function", code: "XTDE0560", sheet: `
<xsl:function name="f:n"><xsl:next-match/></xsl:function>
<xsl:template match="/"><xsl:sequence select="f:n()"/></xsl:template>`})
}
