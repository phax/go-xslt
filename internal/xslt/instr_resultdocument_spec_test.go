package xslt

import (
	"strings"
	"testing"
)

// rdSpecRun runs a stylesheet body and returns the principal output and the
// secondary result documents by href.
func rdSpecRun(t *testing.T, dir, body string) (string, map[string]string, error) {
	t.Helper()
	rr, err := specRunIn(dir, body, "<doc/>")
	if err != nil {
		return "", nil, err
	}
	sec := map[string]string{}
	for _, s := range rr.Secondary {
		sec[s.Href] = s.Content
	}
	return rr.Output, sec, nil
}

// XSLT 3.0 §25.1 xsl:result-document and its serialization attributes
// (§26 / Serialization 3.1).
func TestInstrSpecResultDocumentSerialization(t *testing.T) {
	cases := []struct {
		name, body, href, want string
	}{
		{"named output definition", `<xsl:output name="f:t" method="text"/>
<xsl:template match="/"><xsl:result-document href="a.txt" format="f:t"><e>x</e></xsl:result-document></xsl:template>`, "a.txt", "x"},
		{"format as AVT", `<xsl:output name="t" method="text"/>
<xsl:template match="/"><xsl:result-document href="a.txt" format="{'t'}"><e>y</e></xsl:result-document></xsl:template>`, "a.txt", "y"},
		{"method attribute", `<xsl:template match="/"><xsl:result-document href="a" method="text"><e>z</e></xsl:result-document></xsl:template>`, "a", "z"},
		{"method AVT", `<xsl:template match="/"><xsl:result-document href="a" method="{'text'}"><e>z</e></xsl:result-document></xsl:template>`, "a", "z"},
		{"xml with omitted declaration", `<xsl:template match="/"><xsl:result-document href="a" method="xml" omit-xml-declaration="yes"><e/></xsl:result-document></xsl:template>`, "a", "<e/>"},
		{"xml with declaration and standalone", `<xsl:template match="/"><xsl:result-document href="a" method="xml" standalone="yes"><e/></xsl:result-document></xsl:template>`, "a", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><e/>`},
		{"doctype", `<xsl:template match="/"><xsl:result-document href="a" method="xml" omit-xml-declaration="yes" doctype-system="e.dtd"><e/></xsl:result-document></xsl:template>`, "a", `<!DOCTYPE e SYSTEM "e.dtd"><e/>`},
		{"cdata-section-elements", `<xsl:template match="/"><xsl:result-document href="a" method="xml" omit-xml-declaration="yes" cdata-section-elements="e"><e>a&lt;b</e></xsl:result-document></xsl:template>`, "a", `<e><![CDATA[a<b]]></e>`},
		{"item-separator", `<xsl:template match="/"><xsl:result-document href="a" method="text" item-separator="|"><xsl:sequence select="1, 2, 3"/></xsl:result-document></xsl:template>`, "a", "1|2|3"},
		{"json method", `<xsl:template match="/"><xsl:result-document href="a.json" method="json"><xsl:sequence select="map{'a': 1}"/></xsl:result-document></xsl:template>`, "a.json", `{"a":1}`},
		{"adaptive method", `<xsl:template match="/"><xsl:result-document href="a" method="adaptive"><xsl:sequence select="'s'"/></xsl:result-document></xsl:template>`, "a", `"s"`},
		{"character map", `<xsl:character-map name="m"><xsl:output-character character="x" string="[X]"/></xsl:character-map>
<xsl:template match="/"><xsl:result-document href="a" method="text" use-character-maps="m">axb</xsl:result-document></xsl:template>`, "a", "a[X]b"},
		{"html method", `<xsl:template match="/"><xsl:result-document href="a.html" method="html" include-content-type="no"><html><body><br/></body></html></xsl:result-document></xsl:template>`, "a.html", "<html><body><br></body></html>"},
	}
	for _, c := range cases {
		_, sec, err := rdSpecRun(t, "", c.body)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		got, ok := sec[c.href]
		if !ok {
			t.Errorf("%s: no secondary result %q (have %v)", c.name, c.href, sec)
			continue
		}
		if strings.TrimSpace(got) != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestInstrSpecResultDocumentPrincipal(t *testing.T) {
	// An absent href targets the principal result, with the instruction's
	// own serialization attributes.
	out, sec, err := rdSpecRun(t, "", `<xsl:template match="/"><xsl:result-document method="text" item-separator="-"><xsl:sequence select="1, 2"/></xsl:result-document></xsl:template>`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "1-2" || len(sec) != 0 {
		t.Errorf("got %q, secondary %v", out, sec)
	}
	// Secondary documents leave the principal output alone.
	out, sec, err = rdSpecRun(t, "", `<xsl:template match="/">main<xsl:result-document href="one.txt" method="text">1</xsl:result-document><xsl:result-document href="two.txt" method="text">2</xsl:result-document></xsl:template>`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "main" || sec["one.txt"] != "1" || sec["two.txt"] != "2" {
		t.Errorf("got %q, secondary %v", out, sec)
	}
}

func TestInstrSpecResultDocumentErrors(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "in.xml", "<in/>")
	specRunAll(t, dir, []specCase{
		// XTDE1490: two final result trees with the same URI.
		{name: "same href twice", code: "XTDE1490", sheet: `
<xsl:template match="/"><xsl:result-document href="a.txt">1</xsl:result-document><xsl:result-document href="a.txt">2</xsl:result-document></xsl:template>`},
		{name: "empty href after principal content", code: "XTDE1490", sheet: `
<xsl:template match="/">x<xsl:result-document>1</xsl:result-document></xsl:template>`},
		// XTDE1480: evaluated in temporary output state.
		{name: "inside a variable", code: "XTDE1480", sheet: `
<xsl:template match="/"><xsl:variable name="v"><xsl:result-document href="a.txt">1</xsl:result-document></xsl:variable><xsl:value-of select="$v"/></xsl:template>`},
		// XTDE1500: writing a resource read in the same transformation.
		{name: "write after read", code: "XTDE1500", sheet: `
<xsl:template match="/"><xsl:value-of select="count(doc('in.xml'))"/><xsl:result-document href="in.xml">1</xsl:result-document></xsl:template>`},
		// XTDE1460: @format does not name an output definition.
		{name: "unknown format", code: "XTDE1460", sheet: `
<xsl:template match="/"><xsl:result-document href="a" format="{'nope'}">1</xsl:result-document></xsl:template>`},
		{name: "format with unbound prefix", code: "XTDE1460", sheet: `
<xsl:template match="/"><xsl:result-document href="a" format="{'zz:o'}">1</xsl:result-document></xsl:template>`},
		// SEPM0016: invalid serialization parameter values.
		{name: "bad boolean", code: "SEPM0016", sheet: `
<xsl:template match="/"><xsl:result-document href="a" indent="{'maybe'}">1</xsl:result-document></xsl:template>`},
		{name: "bad standalone", code: "SEPM0016", sheet: `
<xsl:template match="/"><xsl:result-document href="a" standalone="{'perhaps'}">1</xsl:result-document></xsl:template>`},
		{name: "bad escape-uri-attributes", code: "SEPM0016", sheet: `
<xsl:template match="/"><xsl:result-document href="a" escape-uri-attributes="y e s">1</xsl:result-document></xsl:template>`},
		{name: "bad include-content-type", code: "SEPM0016", sheet: `
<xsl:template match="/"><xsl:result-document href="a" include-content-type="maybe">1</xsl:result-document></xsl:template>`},
		{name: "standalone omit is valid", want: "", sheet: `
<xsl:template match="/"><xsl:result-document href="a" standalone="omit">1</xsl:result-document></xsl:template>`},
	})
}
