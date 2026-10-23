package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	style := write("s.xsl", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="who" select="'world'"/>
  <xsl:output method="text"/>
  <xsl:template match="/">hello <xsl:value-of select="$who"/> (<xsl:value-of select="count(//i)"/>)<xsl:result-document href="sub/side.txt">side</xsl:result-document></xsl:template>
  <xsl:template name="xsl:initial-template">no source</xsl:template>
</xsl:stylesheet>`)
	src := write("in.xml", `<r><i/><i/></r>`)
	bad := write("bad.xsl", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><xsl:value-of select="1 +"/></xsl:template>
</xsl:stylesheet>`)
	schema := write("s.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="n" type="xs:int"/></xs:schema>`)
	good := write("good.xml", `<n>5</n>`)
	invalid := write("invalid.xml", `<n>five</n>`)
	hinted := write("hinted.xml", `<n xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="s.xsd">x</n>`)
	embedded := write("emb.xml", `<?xml-stylesheet type="text/xsl" href="#s"?>
<!DOCTYPE d [<!ATTLIST xsl:stylesheet id ID #REQUIRED>]>
<d xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:stylesheet id="s" version="3.0"><xsl:output method="text"/>
<xsl:template match="/">embedded</xsl:template></xsl:stylesheet></d>`)
	out := filepath.Join(dir, "out")

	for _, c := range []struct {
		name   string
		args   []string
		code   int
		stdout string // substring
		stderr string // substring
	}{
		{"transform", []string{"transform", "-s", style, "-p", "who=you", "-d", out, src}, 0, "hello you (2)", ""},
		{"stdin", []string{"transform", "-s", style, "-"}, 0, "hello world (1)", "not written"},
		{"no source", []string{"transform", "-s", style}, 0, "no source", ""},
		{"compile error", []string{"transform", "-s", bad, src}, 1, "", "bad.xsl:2:"},
		{"embedded", []string{"transform", embedded}, 0, "embedded", ""},
		{"valid", []string{"validate", "-x", schema, good}, 0, "valid", ""},
		{"invalid", []string{"validate", "-x", schema, invalid}, 1, "invalid", "error"},
		{"hints", []string{"validate", hinted}, 1, "invalid", ""},
		{"xpath", []string{"xpath", "-s", src, "count(//i), 'x'"}, 0, "2\nx\n", ""},
		{"xpath typed", []string{"xpath", "-t", "1 + 1"}, 0, "xs:integer\t2", ""},
		{"xpath error", []string{"xpath", "1 +"}, 1, "", "XPST0003"},
		{"usage", []string{"frobnicate"}, 2, "", "unknown command"},
	} {
		var so, se bytes.Buffer
		code := run(c.args, strings.NewReader(`<r><i/></r>`), &so, &se)
		if code != c.code || !strings.Contains(so.String(), c.stdout) || !strings.Contains(se.String(), c.stderr) {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", c.name, code, so.String(), se.String())
		}
	}
	if b, err := os.ReadFile(filepath.Join(out, "sub", "side.txt")); err != nil || string(b) != "side" {
		t.Errorf("secondary output: %q %v", b, err)
	}
}
