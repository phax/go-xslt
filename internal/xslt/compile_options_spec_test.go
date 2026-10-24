package xslt

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/fsaccess"
)

// More serialization parameters on xsl:output (Serialization 3.1) and in a
// parameter document, including inline character maps.
func TestOutputSpecMoreParameters(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "cm.xml", `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">
  <output:method value="text"/>
  <output:use-character-maps><output:character-map character="x" map-string="[X]"/></output:use-character-maps>
</output:serialization-parameters>`)
	specWrite(t, dir, "badcm.xml", `<output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">
  <output:use-character-maps><output:character-map character="xy" map-string="z"/></output:use-character-maps>
</output:serialization-parameters>`)
	specRunAll(t, dir, []specCase{
		{name: "parameter document character map", want: "a[X]b", sheet: outSheet(`parameter-document="cm.xml"`, `axb`)},
		{name: "parameter document character map of two characters", code: "XTSE0020", sheet: outSheet(`parameter-document="badcm.xml"`, `x`)},
		// The byte order mark is serialized first.
		{name: "byte-order-mark", want: "\ufeffx", sheet: outSheet(`method="text" byte-order-mark="yes"`, `x`)},
		// suppress-indentation keeps the content of the named element on one line.
		{name: "suppress-indentation", want: "~<p><b>x</b></p>", sheet: outSheet(`method="xml" omit-xml-declaration="yes" indent="yes" suppress-indentation="p"`, `<r><p><b>x</b></p></r>`)},
		// The html method %-escapes non-ASCII characters in URI attributes
		// (fn:escape-html-uri) unless escape-uri-attributes="no".
		{name: "escape-uri-attributes default", want: "~href=\"%C3%A4\"", sheet: outSheet(`method="html"`, "<html><body><a href=\"ä\">x</a></body></html>")},
		{name: "escape-uri-attributes no", want: "~href=\"ä\"", sheet: outSheet(`method="html" escape-uri-attributes="no"`, "<html><body><a href=\"ä\">x</a></body></html>")},
		{name: "media-type and include-content-type", want: "~text/html", sheet: outSheet(`method="html" media-type="text/html" include-content-type="yes" html-version="5"`, `<html><head/><body/></html>`)},
		{name: "json allow-duplicate-names", want: `{"a":1}`, sheet: outSheet(`method="json" allow-duplicate-names="no" json-node-output-method="xml"`, `<xsl:sequence select="map{'a': 1}"/>`)},
		{name: "xml version 1.1 output", want: `~version="1.1"`, sheet: outSheet(`method="xml" version="1.1"`, `<e/>`)},
		{name: "undeclare-prefixes at XML 1.1", want: "~<e", sheet: outSheet(`method="xml" version="1.1" undeclare-prefixes="yes"`, `<e/>`)},
	})
}

// XSLT 3.0 §3.13.? static parameters supplied by the host, and default
// collations (§5.3.?) used by comparisons and keys.
func TestCompileSpecStaticParamsAndCollations(t *testing.T) {
	sheet := specSheet(`<xsl:param name="s" static="yes" select="1"/>
<xsl:variable name="v" select="$s * 10"/>
<xsl:template match="/" use-when="$s gt 2"><xsl:value-of select="'big', $v"/></xsl:template>
<xsl:template match="/" use-when="$s le 2"><xsl:value-of select="'small', $v"/></xsl:template>`)
	for _, c := range []struct {
		static map[string]string
		want   string
	}{
		{nil, "small 10"},
		{map[string]string{"s": "5"}, "big 50"},
	} {
		ss, err := CompileAtWithStatic(sheet, filepath.Join(t.TempDir(), "main.xsl"), c.static)
		if err != nil {
			t.Fatal(err)
		}
		out, _, err := ss.Transform("<doc/>", nil)
		if err != nil {
			t.Fatal(err)
		} else if out != c.want {
			t.Errorf("static %v: got %q want %q", c.static, out, c.want)
		}
	}
	const ci = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"
	specRunAll(t, "", []specCase{
		{name: "default-collation on the stylesheet", want: "true", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` default-collation="` + ci + `"><xsl:output method="text"/>
<xsl:template match="/"><xsl:value-of select="'abc' = 'ABC'"/></xsl:template></xsl:stylesheet>`},
		{name: "default-collation falls back to the first supported URI", want: "true", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` default-collation="http://example.com/unknown ` + ci + `"><xsl:output method="text"/>
<xsl:template match="/"><xsl:value-of select="'abc' = 'ABC'"/></xsl:template></xsl:stylesheet>`},
		// XTSE0125: no supported collation in default-collation.
		{name: "default-collation unsupported", code: "XTSE0125", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` default-collation="http://example.com/unknown"><xsl:output method="text"/>
<xsl:template match="/">x</xsl:template></xsl:stylesheet>`},
		{name: "key with a collation", src: `<r><a k="abc">1</a></r>`, want: "1", sheet: `<xsl:key name="k" match="a" use="@k" collation="` + ci + `"/>
<xsl:template match="/"><xsl:value-of select="key('k', 'ABC')"/></xsl:template>`},
		{name: "key under the default collation", src: `<r><a k="abc">1</a></r>`, want: "1", sheet: `<xsl:stylesheet version="3.0" ` + specNS + ` default-collation="` + ci + `"><xsl:output method="text"/>
<xsl:key name="k" match="a" use="@k"/>
<xsl:template match="/"><xsl:value-of select="key('k', 'ABC')"/></xsl:template></xsl:stylesheet>`},
	})
}

// Local file access is confined by the compile's guard (CLAUDE.md: every
// read goes through fsaccess). An import inside the allowed directory works,
// one outside it fails.
func TestCompileSpecFileGuard(t *testing.T) {
	allowed := t.TempDir()
	other := t.TempDir()
	mod := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/">imported</xsl:template></xsl:stylesheet>`
	specWrite(t, allowed, "in.xsl", mod)
	specWrite(t, other, "out.xsl", mod)
	g := fsaccess.New(allowed)
	ok := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:import href="in.xsl"/><xsl:output method="text"/></xsl:stylesheet>`
	ss, err := CompileFromFS(ok, allowed, g)
	if err != nil {
		t.Fatalf("allowed import: %v", err)
	}
	if out, _, err := ss.Transform("<d/>", nil); err != nil || out != "imported" {
		t.Errorf("allowed import: %q %v", out, err)
	}
	bad := strings.Replace(ok, "in.xsl", filepath.ToSlash(filepath.Join(other, "out.xsl")), 1)
	if _, err := CompileFromFS(bad, allowed, g); err == nil {
		t.Error("import outside the allowed directory: want an error")
	}
}
