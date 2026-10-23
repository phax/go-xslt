package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// A source that carries its own stylesheet is transformed by it when the
// request has none; the module inherits its ancestors' namespaces but none of
// their attributes.
func TestEmbeddedStylesheet(t *testing.T) {
	src := `<?xml-stylesheet type="text/xsl" href="#style1"?>
<!DOCTYPE doc [<!ATTLIST xsl:stylesheet id ID #REQUIRED>]>
<doc xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xsl:xpath-default-namespace="urn:ignored">
  <head>
    <xsl:stylesheet id="style1" version="3.0">
      <xsl:template match="/"><out><xsl:value-of select="count(//item)"/></out></xsl:template>
    </xsl:stylesheet>
  </head>
  <item/><item/>
</doc>`
	for name, run := range map[string]func() engine.Result{
		"Transform": func() engine.Result { return engine.Transform(engine.Request{Source: src}) },
		"TransformTo": func() engine.Result {
			var b strings.Builder
			res := engine.TransformTo(&b, engine.Request{Source: src})
			res.Output = b.String()
			return res
		},
	} {
		res := run()
		if res.HasErrors() || !strings.Contains(res.Output, "<out>2</out>") {
			t.Errorf("%s: got %q %+v", name, res.Output, res.Diagnostics)
		}
	}
}

func TestEmbeddedStylesheetMissing(t *testing.T) {
	for src, want := range map[string]string{
		`<doc/>`: "empty stylesheet",
		`<?xml-stylesheet type="text/xsl" href="#nope"?><doc/>`:     "no element with the ID",
		`<?xml-stylesheet type="text/xsl" href="other.xsl"?><doc/>`: "only a stylesheet embedded",
	} {
		res := engine.Transform(engine.Request{Source: src})
		if !res.HasErrors() || !strings.Contains(res.Diagnostics[0].Message, want) {
			t.Errorf("%s: want %q, got %+v", src, want, res.Diagnostics)
		}
	}
}
