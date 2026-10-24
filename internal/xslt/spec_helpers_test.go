package xslt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specCase is one stylesheet-level test whose expected result is taken from
// the XSLT 3.0 / XPath 3.1 / F&O 3.1 specifications (the section is named in
// the case or next to the table).
//
// sheet is either a complete stylesheet (it starts with "<") or the body of
// one, which specSheet wraps in an XSLT 3.0 xsl:stylesheet with text output.
// Exactly one of want / code is checked: code names the W3C error code the
// compile or the run must fail with.
type specCase struct {
	name  string
	sheet string
	src   string
	want  string
	code  string
}

const specNS = `xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:array="http://www.w3.org/2005/xpath-functions/array" xmlns:fn="http://www.w3.org/2005/xpath-functions" xmlns:math="http://www.w3.org/2005/xpath-functions/math" xmlns:f="urn:f" exclude-result-prefixes="xs map array fn math f"`

// specSheet wraps a stylesheet body in an XSLT 3.0 stylesheet with text output.
func specSheet(body string) string {
	b := strings.TrimSpace(body)
	isBody := strings.HasPrefix(b, "<xsl:") &&
		!strings.HasPrefix(b, "<xsl:stylesheet") && !strings.HasPrefix(b, "<xsl:transform") && !strings.HasPrefix(b, "<xsl:package")
	if !isBody {
		return body
	}
	return `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output method="text"/>
` + body + `
</xsl:stylesheet>`
}

// specRunIn compiles sheet as the file dir/main.xsl (so relative URIs resolve
// against dir) and runs it against src, returning the principal output.
// specXML wraps a stylesheet body in an XSLT 3.0 stylesheet with XML output
// and no XML declaration.
func specXML(body string) string {
	return `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output method="xml" omit-xml-declaration="yes"/>
` + body + `
</xsl:stylesheet>`
}

func specRunIn(dir, sheet, src string) (*RunResult, error) {
	if src == "" {
		src = "<doc/>"
	}
	sheet = specSheet(sheet)
	var ss *Stylesheet
	var err error
	if dir == "" {
		ss, err = Compile(sheet)
	} else {
		ss, err = CompileAt(sheet, filepath.Join(dir, "main.xsl"))
	}
	if err != nil {
		return nil, err
	}
	return ss.TransformFull(src, nil, dir)
}

func specRun(sheet, src string) (string, error) {
	rr, err := specRunIn("", sheet, src)
	if err != nil {
		return "", err
	}
	return rr.Output, nil
}

// specCheck runs one case and reports a mismatch.
func specCheck(t *testing.T, dir string, c specCase) {
	t.Helper()
	rr, err := specRunIn(dir, c.sheet, c.src)
	if c.code != "" {
		if err == nil {
			t.Errorf("%s: want error %s, got output %q", c.name, c.code, rr.Output)
			return
		}
		if !strings.Contains(err.Error(), c.code) {
			t.Errorf("%s: want error %s, got %v", c.name, c.code, err)
		}
		return
	}
	if err != nil {
		t.Errorf("%s: unexpected error: %v", c.name, err)
		return
	}
	if sub, ok := strings.CutPrefix(c.want, "~"); ok {
		// "~text": the output must contain text (for serializations whose
		// surrounding details the specification leaves to the implementation).
		if !strings.Contains(rr.Output, sub) {
			t.Errorf("%s:\n got %q\nwant it to contain %q", c.name, rr.Output, sub)
		}
		return
	}
	if rr.Output != c.want {
		t.Errorf("%s:\n got %q\nwant %q", c.name, rr.Output, c.want)
	}
}

func specRunAll(t *testing.T, dir string, cases []specCase) {
	t.Helper()
	for _, c := range cases {
		specCheck(t, dir, c)
	}
}

// specWrite writes name (slash-separated, relative to dir) with content.
func specWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
