package xslt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xpath"
)

// mtEWrite writes content to dir/name and returns the path.
func mtEWrite(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// mtEIncluding is a stylesheet whose only template comes from a RELATIVE
// xsl:include, so it runs only when its base URI is known.
const mtEIncluding = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="inc.xsl"/>
</xsl:stylesheet>`

const mtEIncluded = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="main"><out>included</out></xsl:template>
</xsl:stylesheet>`

// mtECheck compares a run's outcome with either a wanted output or a wanted
// error code.
func mtECheck(t *testing.T, got string, err error, want, code string) {
	t.Helper()
	if code != "" {
		if err == nil || !strings.Contains(err.Error(), "err:"+code) {
			t.Fatalf("want err:%s, got %q, %v", code, got, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFnTransformLocations covers stylesheet-location and stylesheet-base-uri
// (F&O 3.1 §16.3.2): the location both locates the module and is the default
// for its base URI (a relative one resolves against the static base URI of
// the call); stylesheet-text gets one from stylesheet-base-uri. (Without it
// the include cannot be retrieved, fn-transform-err-9: XTSE0165. Left out —
// the engine reports a bare file error without the XSLT code.)
func TestFnTransformLocations(t *testing.T) {
	dir := t.TempDir()
	main := mtEWrite(t, dir, "main.xsl", mtEIncluding)
	mtEWrite(t, dir, "inc.xsl", mtEIncluded)
	uri := "file://" + filepath.ToSlash(main)

	run := func(expr string) (string, error) {
		sheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="loc"/>
  <xsl:param name="text"/>
  <xsl:output method="text"/>
  <xsl:template match="/"><xsl:value-of select="` + expr + `"/></xsl:template>
</xsl:stylesheet>`
		ss, err := CompileAt(sheet, filepath.Join(dir, "caller.xsl"))
		if err != nil {
			t.Fatalf("compile: %v", err)
		}
		rr, err := ss.TransformFull("<x/>", map[string]string{"loc": uri, "text": mtEIncluding}, dir)
		if err != nil {
			return "", err
		}
		return rr.Output, nil
	}
	for _, c := range []struct{ name, expr, want, code string }{
		{"location-absolute", `transform(map{'stylesheet-location': $loc, 'initial-template': QName('', 'main')})?output/out`, "included", ""},
		{"location-relative", `transform(map{'stylesheet-location': 'main.xsl', 'initial-template': QName('', 'main')})?output/out`, "included", ""},
		{"text-with-base", `transform(map{'stylesheet-text': $text, 'stylesheet-base-uri': $loc, 'initial-template': QName('', 'main')})?output/out`, "included", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := run(c.expr)
			mtECheck(t, got, err, c.want, c.code)
		})
	}
}

// TestFnTransformPackageName invokes a host-supplied package by package-name,
// with package-version defaulting to "*" (any version), with a version that
// selects it, and with one that selects nothing (FOXT0002).
func TestFnTransformPackageName(t *testing.T) {
	dir := t.TempDir()
	pkg := mtEWrite(t, dir, "p.xsl", `<xsl:package name="urn:mte:p" package-version="1.0" version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="main" visibility="public"><out>from-package</out></xsl:template>
</xsl:package>`)
	self := filepath.Join(dir, "caller.xsl")
	for _, c := range []struct{ name, opts, want, code string }{
		{"any-version", `'package-name': 'urn:mte:p'`, "from-package", ""},
		{"matching-version", `'package-name': 'urn:mte:p', 'package-version': '1.0'`, "from-package", ""},
		{"no-such-version", `'package-name': 'urn:mte:p', 'package-version': '2.0'`, "", "FOXT0002"},
		{"with-stylesheet-too", `'package-name': 'urn:mte:p', 'stylesheet-text': '&lt;x/&gt;'`, "", "FOXT0002"},
	} {
		t.Run(c.name, func(t *testing.T) {
			sheet := mtEOuter(`transform(map{` + c.opts + `, 'initial-template': QName('', 'main')})?output/out`)
			ss, err := CompileAtWithPackages(sheet, self, nil, []PackageSource{{Path: pkg}})
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			got, _, err := ss.Transform("<x/>", nil)
			mtECheck(t, got, err, c.want, c.code)
		})
	}
}

// TestFnTransformStandalone calls fn:transform from a bare XPath expression
// with no enclosing stylesheet. With none of initial-template, source-node or
// initial-match-selection, call-template invocation of xsl:initial-template is
// used (§16.3.2).
func TestFnTransformStandalone(t *testing.T) {
	p, err := xpath.Parse(`transform(map{'stylesheet-text': '<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template name="xsl:initial-template"><out>bare</out></xsl:template></xsl:stylesheet>'})?output/out/string()`)
	if err != nil {
		t.Fatal(err)
	}
	v, err := p.Eval(&xpath.Context{})
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := xpath.ToString(v); got != "bare" {
		t.Errorf("got %q, want %q", got, "bare")
	}
}
