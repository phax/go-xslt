package engine_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// TestBaseDirRelativeResolution pins issue #1: every lookup that resolves a
// relative URI against BaseDir must find the file on every OS. On Windows the
// base directory used to become "file://C:/…" (drive read as the URI host) and
// each of these failed; the CI runs this test on windows-latest too.
func TestBaseDirRelativeResolution(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("other.xml", `<r><v>42</v></r>`)
	write("text.txt", "hello")
	write("inc.xsl", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template name="t">included</xsl:template></xsl:stylesheet>`)

	t.Run("doc()", func(t *testing.T) {
		r := engine.EvalXPath(engine.XPathRequest{Expression: "doc('other.xml')/r/v/string()", BaseDir: dir})
		if got := strings.TrimSpace(r.Value); got != "42" {
			t.Fatalf("doc(): value %q, diagnostics %v", r.Value, r.Diagnostics)
		}
	})
	t.Run("unparsed-text()", func(t *testing.T) {
		r := engine.EvalXPath(engine.XPathRequest{Expression: "unparsed-text('text.txt')", BaseDir: dir})
		if r.Value != "hello" {
			t.Fatalf("unparsed-text(): value %q, diagnostics %v", r.Value, r.Diagnostics)
		}
	})
	t.Run("document() and xsl:include in a transform", func(t *testing.T) {
		ss := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="inc.xsl"/>
  <xsl:output method="text"/>
  <xsl:template match="/"><xsl:value-of select="document('other.xml')/r/v"/>|<xsl:call-template name="t"/></xsl:template>
</xsl:stylesheet>`
		r := engine.Transform(engine.Request{Stylesheet: ss, Source: "<x/>", BaseDir: dir})
		if got := strings.TrimSpace(r.Output); got != "42|included" {
			t.Fatalf("output %q, diagnostics %v", r.Output, r.Diagnostics)
		}
	})
}
