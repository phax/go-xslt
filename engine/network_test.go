package engine_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// netServer serves a Latin-1 text resource, an XML document, and an XML
// document whose DOCTYPE points at a local file.
func netServer(t *testing.T, secret string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latin1.txt":
			w.Header().Set("Content-Type", "text/plain; charset=iso-8859-1")
			w.Write([]byte("H\xe5kon"))
		case "/doc.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<r><v>remote</v></r>`))
		case "/xxe.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<!DOCTYPE r SYSTEM "` + secret + `"><r>&leak;</r>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

const netXSL = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="base"/>
  <xsl:template name="xsl:initial-template">
    <out t="{unparsed-text-available($base || '/latin1.txt')}" d="{doc-available($base || '/doc.xml')}">
      <xsl:if test="unparsed-text-available($base || '/latin1.txt')">
        <text><xsl:value-of select="unparsed-text($base || '/latin1.txt')"/></text>
        <doc><xsl:value-of select="doc($base || '/doc.xml')/r/v"/></doc>
        <same><xsl:value-of select="doc($base || '/doc.xml') is document($base || '/doc.xml')"/></same>
      </xsl:if>
    </out>
  </xsl:template>
</xsl:stylesheet>`

func TestNetworkOffByDefault(t *testing.T) {
	srv := netServer(t, "")
	res := engine.Transform(engine.Request{
		Stylesheet: netXSL, InitialTemplate: "xsl:initial-template",
		Params: map[string]string{"base": srv.URL},
	})
	if res.HasErrors() {
		t.Fatalf("diagnostics: %+v", res.Diagnostics)
	}
	if !strings.Contains(res.Output, `t="false" d="false"`) || strings.Contains(res.Output, "<text>") {
		t.Fatalf("network must be off by default, got %s", res.Output)
	}
	// And fn:unparsed-text reports the resource as unretrievable.
	res = engine.Transform(engine.Request{
		Stylesheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template"><xsl:value-of select="unparsed-text('` + srv.URL + `/latin1.txt')"/></xsl:template>
</xsl:stylesheet>`, InitialTemplate: "xsl:initial-template",
	})
	if !res.HasErrors() || res.Diagnostics[0].Code != "FOUT1170" {
		t.Fatalf("want FOUT1170, got %+v", res.Diagnostics)
	}
}

func TestNetworkAllowed(t *testing.T) {
	srv := netServer(t, "")
	res := engine.Transform(engine.Request{
		Stylesheet: netXSL, InitialTemplate: "xsl:initial-template",
		Params:   map[string]string{"base": srv.URL},
		Security: engine.Security{AllowNetwork: true},
	})
	if res.HasErrors() {
		t.Fatalf("diagnostics: %+v", res.Diagnostics)
	}
	for _, want := range []string{`t="true" d="true"`, "<text>Håkon</text>", "<doc>remote</doc>", "<same>true</same>"} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("missing %q in %s", want, res.Output)
		}
	}
}

func TestNetworkAllowedHosts(t *testing.T) {
	srv := netServer(t, "")
	res := engine.Transform(engine.Request{
		Stylesheet: netXSL, InitialTemplate: "xsl:initial-template",
		Params:   map[string]string{"base": srv.URL},
		Security: engine.Security{AllowNetwork: true, AllowedHosts: []string{"example.com"}},
	})
	if res.HasErrors() || !strings.Contains(res.Output, `t="false" d="false"`) {
		t.Fatalf("a host outside AllowedHosts must be unavailable, got %s %+v", res.Output, res.Diagnostics)
	}
}

// A remote document must not be able to read local files through its DOCTYPE.
func TestNetworkDocumentReadsNoLocalDTD(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.dtd")
	if err := os.WriteFile(secret, []byte(`<!ENTITY leak "TOPSECRET">`), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := netServer(t, secret)
	res := engine.Transform(engine.Request{
		Stylesheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template"><out><xsl:value-of select="doc-available('` + srv.URL + `/xxe.xml') and contains(string(doc('` + srv.URL + `/xxe.xml')), 'TOPSECRET')"/></out></xsl:template>
</xsl:stylesheet>`, InitialTemplate: "xsl:initial-template",
		BaseDir:  dir,
		Security: engine.Security{AllowNetwork: true},
	})
	if strings.Contains(res.Output, "true") {
		t.Fatalf("a remote DOCTYPE read a local file: %s", res.Output)
	}
	// Control: the same document read from disk does expand the entity, so
	// the check above is not passing for some unrelated reason.
	local := filepath.Join(dir, "xxe.xml")
	if err := os.WriteFile(local, []byte(`<!DOCTYPE r SYSTEM "`+secret+`"><r>&leak;</r>`), 0o600); err != nil {
		t.Fatal(err)
	}
	res = engine.Transform(engine.Request{
		Stylesheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template"><out><xsl:value-of select="contains(string(doc('xxe.xml')), 'TOPSECRET')"/></out></xsl:template>
</xsl:stylesheet>`, InitialTemplate: "xsl:initial-template",
		BaseDir: dir,
	})
	if !strings.Contains(res.Output, "<out>true</out>") {
		t.Fatalf("control: a local DOCTYPE should expand the entity, got %s %+v", res.Output, res.Diagnostics)
	}
}

func TestXPathNetwork(t *testing.T) {
	srv := netServer(t, "")
	expr := `unparsed-text-available('` + srv.URL + `/latin1.txt')`
	if r := engine.EvalXPath(engine.XPathRequest{Expression: expr}); r.Value != "false" {
		t.Fatalf("XPath without Security: got %q %+v", r.Value, r.Diagnostics)
	}
	r := engine.EvalXPath(engine.XPathRequest{
		Expression: `unparsed-text('` + srv.URL + `/latin1.txt') || ' ' || doc('` + srv.URL + `/doc.xml')/r/v`,
		Security:   engine.Security{AllowNetwork: true},
	})
	if r.Value != "Håkon remote" {
		t.Fatalf("XPath with network allowed: got %q %+v", r.Value, r.Diagnostics)
	}
}
