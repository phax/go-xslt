package xslt

import "testing"

// TestShallowCopyRuleCopiesNamespaces pins that the built-in shallow-copy rule
// (xsl:mode on-no-match="shallow-copy") produces exactly what the explicit
// identity template does: both are xsl:copy plus apply-templates, so both
// carry the source element's namespace nodes — including a binding declared
// on the root that no element in the document happens to use.
func TestShallowCopyRuleCopiesNamespaces(t *testing.T) {
	const src = `<a:r xmlns:a="urn:a" xmlns:b="urn:b" xmlns:unused="urn:u"><a:x><b:y>t</b:y><b:y>u</b:y></a:x></a:r>`
	run := func(sheet string) string {
		ss, err := CompileFrom(sheet, "")
		if err != nil {
			t.Fatal(err)
		}
		res, err := ss.TransformEntry(src, Entry{}, "")
		if err != nil {
			t.Fatal(err)
		}
		return res.Output
	}
	identity := run(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="@*|node()"><xsl:copy><xsl:apply-templates select="@*|node()"/></xsl:copy></xsl:template>
</xsl:stylesheet>`)
	shallow := run(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode on-no-match="shallow-copy"/>
</xsl:stylesheet>`)
	if identity != shallow {
		t.Errorf("shallow-copy rule differs from the explicit identity template\n identity: %s\n shallow:  %s", identity, shallow)
	}
}
