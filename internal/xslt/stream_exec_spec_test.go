package xslt

import (
	"os"
	"path/filepath"
	"testing"
)

// The bounded-memory executor behind xsl:source-document streamable="yes"
// (stream_exec.go, stream_fork.go, stream_group.go). Every case is compiled
// with strict streaming (per stylesheet), so the body has first passed the
// §18.1/§19 gate; the run's output is the value the XSLT semantics give for
// the body, whichever path the engine takes. The count of streamed runs says
// which path that was: a body inside the executor's documented record-path
// subset must take the streaming path, and one outside it must fall back to
// full materialization — with the same result.

// stCExecHdr / stCExecFtr wrap a body as the named template "main".
const stCExecHdr = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
`

// stCRunMain compiles sheet strictly against dir, runs its "main" template and
// returns the output and how many source-documents were streamed.
func stCRunMain(t *testing.T, sheet, dir string) (string, int64) {
	t.Helper()
	ss, err := CompileWith(sheet, dir, CompileOptions{Features: stCStrict()})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	before := strbStreamedRuns.Load()
	res, err := ss.TransformEntry("", Entry{Template: "main"}, dir)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	return res.Output, strbStreamedRuns.Load() - before
}

func stCSourceDoc(decls, body string) string {
	return stCExecHdr + decls + `
  <xsl:template name="main"><xsl:source-document href="d.xml" streamable="yes">` + body + `</xsl:source-document></xsl:template>
</xsl:stylesheet>`
}

func TestStreamedSourceDocumentBodies(t *testing.T) {
	dir := stCDir(t)
	cases := []struct {
		name   string
		decls  string
		body   string
		want   string
		stream bool
	}{
		{name: "focus-free wrapper attribute",
			body: `<out n="{1 + 1}"><xsl:for-each select="data/rec"><xsl:value-of select="v"/></xsl:for-each></out>`,
			want: `<out n="2">123</out>`, stream: true},
		{name: "wrapper attribute reading the document falls back",
			body: `<out n="{name(*)}"><xsl:for-each select="data/rec"><xsl:value-of select="v"/></xsl:for-each></out>`,
			want: `<out n="data">123</out>`},
		{name: "boolean record filter on the last step",
			body: `<xsl:for-each select="data/rec[@id != 'r2']"><xsl:value-of select="v"/></xsl:for-each>`,
			want: "13", stream: true},
		{name: "existence filter on the last step",
			body: `<xsl:for-each select="/data/rec[@id]"><xsl:value-of select="v"/></xsl:for-each>`,
			want: "123", stream: true},
		{name: "positional filter falls back",
			body: `<xsl:for-each select="data/rec[2]"><xsl:value-of select="v"/></xsl:for-each>`,
			want: "2"},
		{name: "wildcard step",
			body: `<xsl:for-each select="*/rec"><xsl:value-of select="v"/></xsl:for-each>`,
			want: "123", stream: true},
		{name: "comments and string literals are not expression syntax",
			body: `<xsl:for-each select="data/rec"><xsl:value-of select="concat(v (: a comment: last() :), '/(..)')"/></xsl:for-each>`,
			want: "1/(..)2/(..)3/(..)", stream: true},
		{name: "constructors inside a record",
			body: `<xsl:for-each select="data/rec"><e><xsl:attribute name="id" select="@id"/><xsl:comment select="@id"/><xsl:processing-instruction name="p" select="@id"/><xsl:value-of select="v"/></e></xsl:for-each>`,
			want: `<e id="r1"><!--r1--><?p r1?>1</e><e id="r2"><!--r2--><?p r2?>2</e><e id="r3"><!--r3--><?p r3?>3</e>`, stream: true},
		{name: "xsl:element and xsl:copy inside a record",
			body: `<xsl:for-each select="data/rec"><xsl:element name="{'x' || @id}"><xsl:copy><xsl:value-of select="v"/></xsl:copy></xsl:element></xsl:for-each>`,
			want: `<xr1><rec>1</rec></xr1><xr2><rec>2</rec></xr2><xr3><rec>3</rec></xr3>`, stream: true},
		{name: "choose, variable and sequence inside a record",
			body: `<xsl:for-each select="data/rec"><xsl:variable name="id" select="string(@id)"/><xsl:choose><xsl:when test="$id = 'r2'"><xsl:sequence select="string(v)"/></xsl:when><xsl:otherwise>-</xsl:otherwise></xsl:choose></xsl:for-each>`,
			want: "-2-", stream: true},
		{name: "analyze-string and a sorted grounded for-each inside a record",
			body: `<xsl:for-each select="data/rec"><xsl:analyze-string select="string(@id)" regex="[0-9]"><xsl:matching-substring>[<xsl:value-of select="."/>]</xsl:matching-substring><xsl:non-matching-substring><xsl:value-of select="."/></xsl:non-matching-substring></xsl:analyze-string><xsl:for-each select="(2, 1)"><xsl:sort select="." data-type="number"/><xsl:value-of select="."/></xsl:for-each>;</xsl:for-each>`,
			want: "r[1]12;r[2]12;r[3]12;", stream: true},
		{name: "apply-templates dispatches each record",
			decls: `<xsl:template match="rec"><xsl:param name="sep" select="'.'"/><xsl:value-of select="v, $sep" separator=""/></xsl:template>`,
			body:  `<xsl:apply-templates select="data/rec"/>`,
			want:  "1.2.3.", stream: true},
		{name: "apply-templates with a parameter falls back",
			decls: `<xsl:template match="rec"><xsl:param name="sep"/><xsl:value-of select="v, $sep" separator=""/></xsl:template>`,
			body:  `<xsl:apply-templates select="data/rec"><xsl:with-param name="sep" select="'!'"/></xsl:apply-templates>`,
			want:  "1!2!3!"},
		{name: "top-level copy-of",
			body: `<xsl:copy-of select="data/rec/v"/>`,
			want: "<v>1</v><v>2</v><v>3</v>", stream: true},
		{name: "iterate with typed parameter and on-completion",
			body: `<xsl:iterate select="data/rec"><xsl:param name="n" as="xs:integer" select="0"/>
			  <xsl:on-completion><sum><xsl:value-of select="$n"/></sum></xsl:on-completion>
			  <xsl:next-iteration><xsl:with-param name="n" select="$n + xs:integer(v)"/></xsl:next-iteration></xsl:iterate>`,
			want: "<sum>6</sum>", stream: true},
		{name: "iterate stopped by xsl:break skips on-completion",
			body: `<xsl:iterate select="data/rec"><xsl:on-completion>done</xsl:on-completion><xsl:choose><xsl:when test="@id = 'r3'"><xsl:break select="'stop'"/></xsl:when><xsl:otherwise><xsl:value-of select="v"/></xsl:otherwise></xsl:choose></xsl:iterate>`,
			want: "12stop", stream: true},
		{name: "group-starting-with",
			body: `<xsl:for-each-group select="data/rec" group-starting-with="rec[@id = 'r2']"><g n="{position()}"><xsl:value-of select="current-group()/@id"/></g></xsl:for-each-group>`,
			want: `<g n="1">r1</g><g n="2">r2 r3</g>`, stream: true},
		{name: "group-ending-with",
			body: `<xsl:for-each-group select="data/rec" group-ending-with="rec[@id = 'r1']"><g><xsl:value-of select="count(current-group())"/></g></xsl:for-each-group>`,
			want: `<g>1</g><g>2</g>`, stream: true},
		{name: "group-adjacent falls back",
			body: `<xsl:for-each-group select="data/rec" group-adjacent="@id"><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group>`,
			want: "r1r2r3"},
		{name: "fork: branches over one record path, results concatenated in branch order",
			body: `<xsl:fork>
			  <xsl:sequence><xsl:for-each select="data/rec"><xsl:value-of select="v"/></xsl:for-each></xsl:sequence>
			  <xsl:sequence><xsl:for-each select="/data/rec"><xsl:value-of select="@id"/></xsl:for-each></xsl:sequence>
			</xsl:fork>`,
			want: "123r1r2r3", stream: true},
		{name: "fork: iterate and grouping branches",
			body: `<xsl:fork>
			  <xsl:sequence><xsl:iterate select="data/rec"><xsl:param name="n" select="0"/><xsl:on-completion>[<xsl:value-of select="$n"/>]</xsl:on-completion><xsl:next-iteration><xsl:with-param name="n" select="$n + 1"/></xsl:next-iteration></xsl:iterate></xsl:sequence>
			  <xsl:sequence><xsl:for-each-group select="data/rec" group-ending-with="rec[@id = 'r2']"><xsl:value-of select="count(current-group())"/></xsl:for-each-group></xsl:sequence>
			</xsl:fork>`,
			want: "[3]21", stream: true},
		{name: "fork: one branch breaks, the other runs to the end",
			body: `<xsl:fork>
			  <xsl:sequence><xsl:iterate select="data/rec"><xsl:choose><xsl:when test="@id = 'r2'"><xsl:break/></xsl:when><xsl:otherwise><xsl:value-of select="v"/></xsl:otherwise></xsl:choose></xsl:iterate></xsl:sequence>
			  <xsl:sequence><xsl:for-each select="data/rec"><xsl:value-of select="@id"/></xsl:for-each></xsl:sequence>
			</xsl:fork>`,
			want: "1r1r2r3", stream: true},
		{name: "fork: branches over different paths fall back",
			body: `<xsl:fork>
			  <xsl:sequence><xsl:for-each select="data/rec"><xsl:value-of select="v"/></xsl:for-each></xsl:sequence>
			  <xsl:sequence><xsl:for-each select="data"><xsl:value-of select="name()"/></xsl:for-each></xsl:sequence>
			</xsl:fork>`,
			want: "123data"},
		{name: "fork: for-each-group child",
			body: `<xsl:fork><xsl:for-each-group select="data/rec" group-starting-with="rec[@id = 'r3']"><xsl:value-of select="count(current-group())"/></xsl:for-each-group></xsl:fork>`,
			want: "21", stream: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, n := stCRunMain(t, stCSourceDoc(c.decls, c.body), dir)
			if got != c.want {
				t.Errorf("output %q, want %q", got, c.want)
			}
			if (n > 0) != c.stream {
				t.Errorf("streamed runs = %d, want streamed=%v", n, c.stream)
			}
		})
	}
}

// TestStreamedRecordsAreStripped checks that xsl:strip-space (§4.3) applies to
// a streamed record exactly as to a materialized document, including an
// xml:space="preserve" inherited from an ancestor outside the record.
func TestStreamedRecordsAreStripped(t *testing.T) {
	dir := t.TempDir()
	write := func(name, doc string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("s.xml", "<data>\n <rec> <v>1</v> </rec>\n <rec> <v>2</v> </rec>\n</data>")
	write("p.xml", "<data xml:space=\"preserve\">\n <rec> <v>1</v> </rec>\n <rec> <v>2</v> </rec>\n</data>")
	sheet := func(href string) string {
		return stCExecHdr + `  <xsl:strip-space elements="*"/>
  <xsl:template name="main"><xsl:source-document href="` + href + `" streamable="yes"><xsl:for-each select="data/rec"><xsl:value-of select="count(node())"/></xsl:for-each></xsl:source-document></xsl:template>
</xsl:stylesheet>`
	}
	if got, n := stCRunMain(t, sheet("s.xml"), dir); got != "11" || n == 0 {
		t.Errorf("stripped: output %q (streamed %d), want \"11\" streamed", got, n)
	}
	if got, n := stCRunMain(t, sheet("p.xml"), dir); got != "33" || n == 0 {
		t.Errorf("xml:space=preserve: output %q (streamed %d), want \"33\" streamed", got, n)
	}
}
