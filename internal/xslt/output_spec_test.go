package xslt

import "testing"

// outSheet is a stylesheet with the given xsl:output attributes and template body.
func outSheet(output, body string) string {
	return `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output ` + output + `/>
<xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
}

// XSLT 3.0 §26 xsl:output and Serialization 3.1: serialization parameters
// and the serialization errors (SEPM*, SESU*, SERE*).
func TestOutputSpecSerialization(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "xml declaration", want: `<?xml version="1.0" encoding="UTF-8"?><e/>`, sheet: outSheet(`method="xml"`, `<e/>`)},
		{name: "standalone", want: `<?xml version="1.0" encoding="UTF-8" standalone="no"?><e/>`, sheet: outSheet(`method="xml" standalone="no"`, `<e/>`)},
		{name: "doctype", want: `<!DOCTYPE e PUBLIC "-//X//EN" "e.dtd"><e/>`, sheet: outSheet(`method="xml" omit-xml-declaration="yes" doctype-public="-//X//EN" doctype-system="e.dtd"`, `<e/>`)},
		{name: "cdata-section-elements", want: `<e><![CDATA[<x>]]></e>`, sheet: outSheet(`method="xml" omit-xml-declaration="yes" cdata-section-elements="e"`, `<e>&lt;x&gt;</e>`)},
		{name: "item-separator", want: `1;2;3`, sheet: outSheet(`method="text" item-separator=";"`, `<xsl:sequence select="1 to 3"/>`)},
		{name: "json output", want: `[1,"a",true]`, sheet: outSheet(`method="json"`, `<xsl:sequence select="[1, 'a', true()]"/>`)},
		{name: "adaptive output", want: `"x"`, sheet: outSheet(`method="adaptive"`, `<xsl:sequence select="'x'"/>`)},
		{name: "html void element", want: `~<br>`, sheet: outSheet(`method="html"`, `<html><body><br/></body></html>`)},
		{name: "html5 doctype", want: `~<!DOCTYPE html>`, sheet: outSheet(`method="html" html-version="5"`, `<html><body/></html>`)},
		{name: "xhtml method", want: `~<br />`, sheet: outSheet(`method="xhtml" omit-xml-declaration="yes"`, `<html xmlns="http://www.w3.org/1999/xhtml"><body><br/></body></html>`)},
		{name: "text method", want: `a<b`, sheet: outSheet(`method="text"`, `<e>a&lt;b</e>`)},
		{name: "indent", want: "~<e>\n", sheet: outSheet(`method="xml" omit-xml-declaration="yes" indent="yes"`, `<e><f/></e>`)},
		{name: "character map", want: `a[X]b`, sheet: `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output method="text" use-character-maps="m"/>
<xsl:character-map name="m" use-character-maps="m2"><xsl:output-character character="x" string="[X]"/></xsl:character-map>
<xsl:character-map name="m2"><xsl:output-character character="y" string="Y"/></xsl:character-map>
<xsl:template match="/">axb</xsl:template></xsl:stylesheet>`},
		{name: "character map from a referenced map", want: `aYb`, sheet: `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output method="text" use-character-maps="m"/>
<xsl:character-map name="m" use-character-maps="m2"><xsl:output-character character="x" string="[X]"/></xsl:character-map>
<xsl:character-map name="m2"><xsl:output-character character="y" string="Y"/></xsl:character-map>
<xsl:template match="/">ayb</xsl:template></xsl:stylesheet>`},
		// XTSE1590: use-character-maps names an undeclared map.
		{name: "undeclared character map", code: "XTSE1590", sheet: outSheet(`method="text" use-character-maps="nope"`, `x`)},
		// XTSE1600: a character map that references itself.
		{name: "circular character maps", code: "XTSE1600", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:character-map name="a" use-character-maps="b"/><xsl:character-map name="b" use-character-maps="a"/>
<xsl:template match="/">x</xsl:template></xsl:stylesheet>`},
		// XTSE1580: two character maps of the same name and precedence.
		{name: "duplicate character map", code: "XTSE1580", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:character-map name="a"/><xsl:character-map name="a"/>
<xsl:template match="/">x</xsl:template></xsl:stylesheet>`},
		// SEPM0004: doctype-system with more than one top-level element.
		{name: "doctype with two top-level elements", code: "SEPM0004", sheet: outSheet(`method="xml" doctype-system="e.dtd"`, `<a/><b/>`)},
		// SEPM0009: omit-xml-declaration with standalone.
		{name: "omitted declaration with standalone", code: "SEPM0009", sheet: outSheet(`method="xml" omit-xml-declaration="yes" standalone="yes"`, `<e/>`)},
		// SEPM0010: undeclare-prefixes with XML 1.0.
		{name: "undeclare-prefixes with XML 1.0", code: "SEPM0010", sheet: outSheet(`method="xml" undeclare-prefixes="yes"`, `<e/>`)},
		// SESU0007: unsupported encoding.
		{name: "unsupported encoding", code: "SESU0007", sheet: outSheet(`method="text" encoding="x-no-such-encoding"`, `x`)},
		// SESU0011: unsupported normalization form.
		{name: "unsupported normalization-form", code: "SESU0011", sheet: outSheet(`method="xml" normalization-form="NFQ"`, `<e/>`)},
		{name: "supported normalization-form", want: "é", sheet: outSheet(`method="text" normalization-form="NFC"`, "é")},
		// SERE0015: ">" in a processing instruction with the html method.
		{name: "html PI with >", code: "SERE0015", sheet: outSheet(`method="html"`, `<html><xsl:processing-instruction name="p">a>b</xsl:processing-instruction></html>`)},
		// SERE0014: a C1 control character with HTML before version 5.
		{name: "html4 C1 character", code: "SERE0014", sheet: outSheet(`method="html" version="4.01"`, `<html><xsl:value-of select="codepoints-to-string(150)"/></html>`)},
		// SESU0013: an unsupported html version.
		{name: "unsupported html version", code: "SESU0013", sheet: outSheet(`method="html" version="0.5"`, `<html/>`)},
		// Static validation of xsl:output attributes: XTSE0020.
		{name: "bad boolean on xsl:output", code: "XTSE0020", sheet: outSheet(`method="xml" indent="sometimes"`, `<e/>`)},
		{name: "bad method", code: "XTSE1570", sheet: outSheet(`method="pdf"`, `<e/>`)},
		// XTSE1560: conflicting xsl:output values at the same precedence.
		{name: "conflicting outputs", code: "XTSE1560", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output method="xml"/><xsl:output method="text"/>
<xsl:template match="/">x</xsl:template></xsl:stylesheet>`},
		{name: "outputs merge by attribute", want: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><e/>`, sheet: `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output method="xml"/><xsl:output standalone="yes"/>
<xsl:template match="/"><e/></xsl:template></xsl:stylesheet>`},
		{name: "build-tree no", want: `1 2`, sheet: outSheet(`method="text" build-tree="no"`, `<xsl:sequence select="1, 2"/>`)},
	})
}

// XSLT 3.0 §26.? xsl:output/@parameter-document and
// xsl:result-document/@parameter-document (Serialization 3.1 §3.1).
func TestOutputSpecParameterDocument(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "params.xml", `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">
  <output:method value="text"/>
  <output:item-separator value="|"/>
</output:serialization-parameters>`)
	specRunAll(t, dir, []specCase{
		{name: "xsl:output parameter document", want: "1|2", sheet: outSheet(`parameter-document="params.xml"`, `<xsl:sequence select="1, 2"/>`)},
		{name: "result-document parameter document", want: "", sheet: `<xsl:stylesheet version="3.0" ` + specNS + `><xsl:output method="text"/>
<xsl:template match="/"><xsl:result-document href="out.txt" parameter-document="params.xml"><xsl:sequence select="1, 2"/></xsl:result-document></xsl:template></xsl:stylesheet>`},
		{name: "attribute overrides the parameter document", want: "1;2", sheet: outSheet(`parameter-document="params.xml" item-separator=";"`, `<xsl:sequence select="1, 2"/>`)},
	})
}
