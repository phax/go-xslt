package xslt

import (
	"path/filepath"
	"strings"
	"testing"
)

// F&O 3.1 §14.6 fn:collection / fn:uri-collection with host-supplied
// collections (XSLT 3.0 leaves available collections to the host), plus
// fn:doc, fn:doc-available and fn:unparsed-text* resolved against the run's
// base directory.
func TestResolverSpecCollections(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "c/one.xml", `<d n="1"/>`)
	specWrite(t, dir, "c/two.xml", `<d n="2"/>`)
	specWrite(t, dir, "c/skip.txt", `not xml`)
	specWrite(t, dir, "ids.xml", `<r><e xml:id="x1">X</e></r>`)
	specWrite(t, dir, "lines.txt", "a\nb\r\nc")
	ss, err := CompileAt(specSheet(`
<xsl:template name="default"><xsl:value-of select="sort(collection()/d/@n)"/></xsl:template>
<xsl:template name="named"><xsl:value-of select="count(collection('mine'))"/></xsl:template>
<xsl:template name="frag"><xsl:value-of select="collection('frag')"/></xsl:template>
<xsl:template name="glob"><xsl:value-of select="sort(collection('c?select=*.xml')/d/@n)"/></xsl:template>
<xsl:template name="text"><xsl:value-of select="unparsed-text('lines.txt') = concat('a', codepoints-to-string(10), 'b', codepoints-to-string(13), codepoints-to-string(10), 'c'), unparsed-text-lines('lines.txt'), unparsed-text-available('lines.txt'), unparsed-text-available('nope.txt')" separator="|"/></xsl:template>
<xsl:template name="doc"><xsl:value-of select="doc('c/one.xml')/d/@n, doc-available('c/two.xml'), doc-available('missing.xml'), doc('c/one.xml') is doc('c/one.xml')" separator="|"/></xsl:template>
<xsl:template name="docerr"><xsl:value-of select="doc('missing.xml')"/></xsl:template>
<xsl:template name="texterr"><xsl:value-of select="unparsed-text('missing.txt')"/></xsl:template>
`), filepath.Join(dir, "main.xsl"))
	if err != nil {
		t.Fatal(err)
	}
	colls := map[string][]string{
		"":     {"c/one.xml", "c/two.xml"},
		"mine": {"c/one.xml"},
		"frag": {"ids.xml#x1"},
	}
	cases := []struct {
		tmpl, want, code string
	}{
		{tmpl: "default", want: "1 2"},
		{tmpl: "named", want: "1"},
		{tmpl: "frag", want: "X"},
		{tmpl: "glob", want: "1 2"},
		{tmpl: "text", want: "true|a|b|c|true|false"},
		{tmpl: "doc", want: "1|true|false|true"},
		// F&O §14.6.1: FODC0002 when the document cannot be retrieved.
		{tmpl: "docerr", code: "FODC0002"},
		// F&O §14.8.1: FOUT1170 when the resource cannot be retrieved.
		{tmpl: "texterr", code: "FOUT1170"},
	}
	for _, c := range cases {
		rr, err := ss.TransformEntry("", Entry{Template: c.tmpl, Collections: colls}, dir)
		if c.code != "" {
			if err == nil || !strings.Contains(err.Error(), c.code) {
				t.Errorf("%s: want %s, got %v", c.tmpl, c.code, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.tmpl, err)
		} else if rr.Output != c.want {
			t.Errorf("%s: got %q want %q", c.tmpl, rr.Output, c.want)
		}
	}
}

// F&O 3.1 §14.6.1/§14.6.4: a resource that cannot be parsed as XML is
// FODC0002 for fn:doc, and doc-available() is false.
func TestResolverSpecNonXMLResource(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "skip.txt", `not xml`)
	specCheck(t, dir, specCase{name: "doc of non-XML", code: "FODC0002", sheet: `<xsl:template match="/"><xsl:value-of select="count(doc('skip.txt'))"/></xsl:template>`})
	specCheck(t, dir, specCase{name: "doc-available of non-XML", want: "false", sheet: `<xsl:template match="/"><xsl:value-of select="doc-available('skip.txt')"/></xsl:template>`})
}

// XSLT 3.0 §18.1.2 xsl:source-document, streamed and unstreamed.
func TestSourceDocSpec(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "in.xml", `<r><i>1</i><i>2</i><i>3</i></r>`)
	specRunAll(t, dir, []specCase{
		{name: "unstreamed", want: "6", sheet: `
<xsl:template match="/"><xsl:source-document href="in.xml" streamable="no"><xsl:value-of select="sum(r/i)"/></xsl:source-document></xsl:template>`},
		{name: "streamed", want: "6", sheet: `
<xsl:template match="/"><xsl:source-document href="in.xml" streamable="yes"><xsl:value-of select="sum(r/i)"/></xsl:source-document></xsl:template>`},
		{name: "streamed for-each", want: "1;2;3;", sheet: `
<xsl:template match="/"><xsl:source-document href="in.xml" streamable="yes"><xsl:for-each select="r/i"><xsl:value-of select="."/>;</xsl:for-each></xsl:source-document></xsl:template>`},
		{name: "href AVT", want: "3", sheet: `
<xsl:template match="/"><xsl:source-document href="{'in' || '.xml'}"><xsl:value-of select="count(r/i)"/></xsl:source-document></xsl:template>`},
		// XTDE1460? the resource cannot be retrieved: FODC0002 per §18.1.2.
		{name: "missing document", code: "FODC0002", sheet: `
<xsl:template match="/"><xsl:source-document href="missing.xml"><xsl:value-of select="."/></xsl:source-document></xsl:template>`},
		{name: "missing streamed document", code: "FODC0002", sheet: `
<xsl:template match="/"><xsl:source-document href="missing.xml" streamable="yes"><xsl:value-of select="count(*)"/></xsl:source-document></xsl:template>`},
	})
}
