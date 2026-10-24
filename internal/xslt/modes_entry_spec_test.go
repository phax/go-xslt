package xslt

import (
	"strings"
	"testing"
)

// XSLT 3.0 §2.3.3 initial modes named by the host, and §6.6.? mode
// attributes that do not change the result.
func TestModeSpecInitialModes(t *testing.T) {
	ss, err := Compile(specSheet(`
<xsl:mode name="declared" typed="false" warning-on-no-match="yes" warning-on-multiple-match="no"/>
<xsl:template match="/" mode="declared">declared</xsl:template>
<xsl:template match="/" mode="f:onlytemplate">only-template</xsl:template>
<xsl:template match="/" mode="a b">list</xsl:template>
<xsl:template match="/">unnamed</xsl:template>`))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		mode, want, code string
	}{
		{mode: "declared", want: "declared"},
		// A mode named only by a template's @mode is a known mode.
		{mode: "{urn:f}onlytemplate", want: "only-template"},
		{mode: "b", want: "list"},
		{mode: "#default", want: "unnamed"},
		{mode: "missing", code: "XTDE0045"},
	}
	for _, c := range cases {
		rr, err := ss.TransformEntry("<doc/>", Entry{Mode: c.mode}, "")
		if c.code != "" {
			if err == nil || !strings.Contains(err.Error(), c.code) {
				t.Errorf("%s: want %s, got %v", c.mode, c.code, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.mode, err)
		} else if rr.Output != c.want {
			t.Errorf("%s: got %q want %q", c.mode, rr.Output, c.want)
		}
	}
	specRunAll(t, "", []specCase{
		{name: "typed=yes spelled true", code: "XTTE3100", sheet: `
<xsl:mode name="t" typed="true"/>
<xsl:template match="/"><xsl:apply-templates select="*" mode="t"/></xsl:template>
<xsl:template match="*" mode="t">x</xsl:template>`},
		// typed="unspecified" (the default) accepts any node.
		{name: "typed unspecified", want: "x", sheet: `
<xsl:mode name="t" typed="unspecified"/>
<xsl:template match="/"><xsl:apply-templates select="*" mode="t"/></xsl:template>
<xsl:template match="*" mode="t">x</xsl:template>`},
		{name: "mode-specific on-no-match", src: `<r><a>t</a></r>`, want: "<R><a>t</a></R>", sheet: specXML(`
<xsl:mode name="copy" on-no-match="shallow-copy"/>
<xsl:template match="/"><R><xsl:apply-templates select="r/node()" mode="copy"/></R></xsl:template>`)},
	})
}
