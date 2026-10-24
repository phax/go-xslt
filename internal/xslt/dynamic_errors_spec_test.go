package xslt

import (
	"strings"
	"testing"
)

// Dynamic and serialization errors of XSLT 3.0 / XPath 3.1 / Serialization
// 3.1, one case per rule.
func TestSpecDynamicErrors(t *testing.T) {
	specRunAll(t, "", []specCase{
		// Serialization 3.1 §5: SEPM0009 omit-xml-declaration with a doctype under XML 1.1.
		{name: "omitted declaration with doctype at XML 1.1", code: "SEPM0009", sheet: outSheet(`method="xml" version="1.1" omit-xml-declaration="yes" doctype-system="e.dtd"`, `<e/>`)},
		// doctype-public must be a PubidLiteral.
		{name: "invalid doctype-public", code: "SEPM0016", sheet: outSheet(`method="xml" doctype-public="a~b" doctype-system="e.dtd"`, `<e/>`)},
		{name: "invalid html-version", code: "XTSE0020", sheet: outSheet(`method="html" html-version="five"`, `<html/>`)},
		// SENR0001: a map in a sequence serialized with the xml method.
		{name: "map serialized as xml", code: "SENR0001", sheet: outSheet(`method="xml"`, `<xsl:sequence select="map{}"/>`)},
		// XTDE1490: principal output written after an href-less xsl:result-document produced it.
		{name: "principal written after result-document", code: "XTDE1490", sheet: `
<xsl:template match="/"><xsl:result-document>a</xsl:result-document>b</xsl:template>`},
		// The engine's recursion limits raise XPDY0130 (documented in CLAUDE.md).
		{name: "unbounded template recursion", code: "XPDY0130", sheet: `
<xsl:template match="/"><xsl:call-template name="t"/></xsl:template>
<xsl:template name="t"><x/><xsl:call-template name="t"/></xsl:template>`},
		{name: "unbounded function recursion", code: "XPDY0130", sheet: `
<xsl:function name="f:r"><xsl:param name="n"/><xsl:sequence select="f:r($n + 1)"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:r(0)"/></xsl:template>`},
		// XTTE0570: a local variable whose value does not match @as.
		{name: "local variable type", code: "XTTE0570", sheet: `
<xsl:template match="/"><xsl:variable name="v" as="xs:integer" select="'a'"/><xsl:value-of select="$v"/></xsl:template>`},
		{name: "local variable content type", code: "XTTE0570", sheet: `
<xsl:template match="/"><xsl:variable name="v" as="xs:integer"><xsl:sequence select="'a'"/></xsl:variable><xsl:value-of select="$v"/></xsl:template>`},
		{name: "local variable cardinality", code: "XTTE0570", sheet: `
<xsl:template match="/"><xsl:variable name="v" as="xs:integer" select="1, 2"/><xsl:value-of select="$v"/></xsl:template>`},
		// XTDE0420: an attribute in a temporary tree's document node.
		{name: "attribute in a temporary tree", code: "XTDE0420", sheet: `
<xsl:template match="/"><xsl:variable name="v"><xsl:attribute name="a">1</xsl:attribute></xsl:variable><xsl:value-of select="$v"/></xsl:template>`},
		// FOTY0013: xsl:value-of / xsl:attribute of a map.
		{name: "value-of a map", code: "FOTY0013", sheet: `
<xsl:template match="/"><xsl:value-of select="map{1: 2}"/></xsl:template>`},
		{name: "attribute of a function item", code: "FOTY0013", sheet: specXML(`
<xsl:template match="/"><e><xsl:attribute name="a" select="true#0"/></e></xsl:template>`)},
		// XTTE0510: apply-templates without select when the context item is not a node.
		{name: "apply-templates on an atomic context", code: "XTTE0510", sheet: `
<xsl:template match="/"><xsl:for-each select="1"><xsl:apply-templates/></xsl:for-each></xsl:template>`},
		// XTTE3100: a typed="yes" mode applied to an untyped node.
		{name: "typed mode on an untyped node", code: "XTTE3100", sheet: `
<xsl:mode name="t" typed="yes"/>
<xsl:template match="/"><xsl:apply-templates select="*" mode="t"/></xsl:template>
<xsl:template match="*" mode="t">x</xsl:template>`},
		{name: "typed=no mode on an untyped node", want: "x", sheet: `
<xsl:mode name="t" typed="no"/>
<xsl:template match="/"><xsl:apply-templates select="*" mode="t"/></xsl:template>
<xsl:template match="*" mode="t">x</xsl:template>`},
		// XTDE0410: an attribute node in content after a child.
		{name: "attribute node after a child", code: "XTDE0410", sheet: specXML(`
<xsl:variable name="a" as="attribute()"><xsl:attribute name="a">1</xsl:attribute></xsl:variable>
<xsl:template match="/"><e><c/><xsl:sequence select="$a"/></e></xsl:template>`)},
		{name: "attribute node before a child", want: `<e a="1"><c/></e>`, sheet: specXML(`
<xsl:variable name="a" as="attribute()"><xsl:attribute name="a">1</xsl:attribute></xsl:variable>
<xsl:template match="/"><e><xsl:sequence select="$a"/><c/></e></xsl:template>`)},
		// XTDE0450: a function item in element content.
		{name: "function item in element content", code: "XTDE0450", sheet: specXML(`
<xsl:template match="/"><e><xsl:sequence select="true#0"/></e></xsl:template>`)},
		// XPTY0004: analyze-string select is not a single string.
		{name: "analyze-string over a sequence", code: "XPTY0004", sheet: `
<xsl:template match="/"><xsl:analyze-string select="('a', 'b')" regex="a"><xsl:matching-substring>m</xsl:matching-substring></xsl:analyze-string></xsl:template>`},
		// XSLT 3.0 (unlike 2.0) allows a regex that matches the empty string.
		{name: "regex matching the empty string", want: "mm", sheet: `
<xsl:template match="/"><xsl:analyze-string select="'a'" regex="{'x*'}"><xsl:matching-substring>m</xsl:matching-substring></xsl:analyze-string></xsl:template>`},
		{name: "analyze-string with flags", want: "[A]b[A]", sheet: `
<xsl:template match="/"><xsl:analyze-string select="'aba'" regex="A" flags="i"><xsl:matching-substring>[A]</xsl:matching-substring><xsl:non-matching-substring><xsl:value-of select="."/></xsl:non-matching-substring></xsl:analyze-string></xsl:template>`},
		// XTDE0640: a key whose use expression needs the key itself.
		{name: "circular key", src: `<r><a/></r>`, code: "XTDE0640", sheet: `
<xsl:key name="k" match="a" use="count(key('k', 'x'))"/>
<xsl:template match="/"><xsl:value-of select="count(key('k', '1'))"/></xsl:template>`},
		// XTDE1030: durations have no total order.
		{name: "sorting durations", code: "XTDE1030", sheet: `
<xsl:template match="/"><xsl:perform-sort select="(xs:duration('P1M'), xs:duration('P30D'))"><xsl:sort select="."/></xsl:perform-sort></xsl:template>`},
		{name: "sorting dates with strings", code: "XTDE1030", sheet: `
<xsl:template match="/"><xsl:perform-sort select="(xs:date('2020-01-01'), 'x')"><xsl:sort select="."/></xsl:perform-sort></xsl:template>`},
		{name: "sorting dayTimeDurations", want: "PT1M PT1H", sheet: `
<xsl:template match="/"><xsl:perform-sort select="(xs:dayTimeDuration('PT1H'), xs:dayTimeDuration('PT1M'))"><xsl:sort select="."/></xsl:perform-sort></xsl:template>`},
	})
}

// XSLT 3.0 §2.3: entry points of a package must be public or final.
func TestEntrySpecPackageVisibility(t *testing.T) {
	pkg := `<xsl:package version="3.0" name="urn:p" ` + specNS + `><xsl:output method="text"/>
<xsl:mode/><xsl:mode name="pub" visibility="public"/><xsl:mode name="priv"/>
<xsl:template match="/">u</xsl:template>
<xsl:template match="/" mode="pub">p</xsl:template>
<xsl:template match="/" mode="priv">q</xsl:template>
<xsl:template name="xsl:initial-template">init</xsl:template>
</xsl:package>`
	ss, err := Compile(pkg)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, src string
		e         Entry
		want      string
		code      string
	}{
		{name: "public mode", src: "<d/>", e: Entry{Mode: "pub"}, want: "p"},
		{name: "private mode", src: "<d/>", e: Entry{Mode: "priv"}, code: "XTDE0045"},
		// A private xsl:initial-template is no entry point (XTDE0040).
		{name: "private initial template", e: Entry{}, code: "XTDE0040"},
	}
	for _, c := range cases {
		rr, err := ss.TransformEntry(c.src, c.e, "")
		if c.code != "" {
			if err == nil || !strings.Contains(err.Error(), c.code) {
				t.Errorf("%s: want %s, got %v", c.name, c.code, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
		} else if rr.Output != c.want {
			t.Errorf("%s: got %q want %q", c.name, rr.Output, c.want)
		}
	}
}

func TestSpecDynamicErrorsKnownBugs(t *testing.T) {
	bugs := []struct {
		bug string
		c   specCase
	}{
		{"BUG: invalid flags in an AVT-valued xsl:analyze-string/@flags raise FORX0001; §17.1 defines XTDE1145",
			specCase{name: "invalid regex flags", code: "XTDE1145", sheet: `
<xsl:template match="/"><xsl:analyze-string select="'a'" regex="a" flags="{'z'}"><xsl:matching-substring>m</xsl:matching-substring></xsl:analyze-string></xsl:template>`}},
		{"BUG: an invalid AVT-valued xsl:analyze-string/@regex raises FORX0002; §17.1 defines XTDE1140",
			specCase{name: "invalid regex", code: "XTDE1140", sheet: `
<xsl:template match="/"><xsl:analyze-string select="'a'" regex="{'('}"><xsl:matching-substring>m</xsl:matching-substring></xsl:analyze-string></xsl:template>`}},
	}
	for _, b := range bugs {
		t.Run(b.c.name, func(t *testing.T) {
			t.Skip(b.bug)
			specCheck(t, "", b.c)
		})
	}
}
