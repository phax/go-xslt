package xslt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Instruction-level guaranteed-streamability rules (XSLT 3.0 §19.8.4), driven
// through their natural entry points: a template rule of a mode declared
// streamable="yes" (§19.8.10 + §6.6.4) and an xsl:source-document declared
// streamable="yes" (§18.1). Every case is compiled with strict enforcement on
// (Features.StrictStreaming, per stylesheet), so a construct the REC rules out
// must be XTSE3430; every accepted case is then run and its result compared
// with the value the XSLT/XPath semantics give for it.

// stCDoc is the input every case reads: three records whose v children sum to 6.
const stCDoc = `<data><rec id="r1"><v>1</v></rec><rec id="r2"><v>2</v></rec><rec id="r3"><v>3</v></rec></data>`

// stCStrict are the Features of a processor that treats streamable="yes" as
// binding (XTSE3430 for anything not guaranteed-streamable).
func stCStrict() *Features {
	return &Features{StrictStreaming: true, StreamingClaim: true}
}

// stCDir writes stCDoc as d.xml into a fresh directory.
func stCDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "d.xml"), []byte(stCDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// stCModeSheet wraps body as the "/" rule of the unnamed mode, declared
// streamable, plus any extra top-level declarations.
func stCModeSheet(body, extra string) string {
	return `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:output method="text"/>
  <xsl:mode streamable="yes"/>
  ` + extra + `
  <xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
}

// stCCheck compiles sheet strictly and either expects XTSE3430 (want == "")
// or runs it on stCDoc and expects the text output want.
func stCCheck(t *testing.T, sheet string, reject bool, want string) {
	t.Helper()
	dir := stCDir(t)
	ss, err := CompileWith(sheet, dir, CompileOptions{Features: stCStrict()})
	if reject {
		if err == nil {
			t.Fatalf("compiled; want XTSE3430 (not guaranteed-streamable)")
		}
		if !strings.Contains(err.Error(), "XTSE3430") {
			t.Fatalf("error %v; want XTSE3430", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	res, err := ss.TransformFull(stCDoc, nil, dir)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if got := strings.TrimSpace(res.Output); got != want {
		t.Fatalf("output %q, want %q", got, want)
	}
}

func TestStreamableModeInstructionRules(t *testing.T) {
	const recTpl = `<xsl:template match="rec"><xsl:value-of select="v"/></xsl:template>`
	const gRecTpl = `<xsl:template match="rec" mode="g"><xsl:value-of select="v"/></xsl:template>`
	cases := []struct {
		name   string
		body   string
		extra  string
		reject bool
		want   string
	}{
		// §19.8.4.18 xsl:for-each.
		{name: "for-each striding, consuming body",
			body: `<xsl:for-each select="data/rec"><xsl:value-of select="v"/></xsl:for-each>`, want: "123"},
		{name: "for-each over streamed nodes with xsl:sort (rule 2)",
			body: `<xsl:for-each select="data/rec"><xsl:sort select="v"/><xsl:value-of select="v"/></xsl:for-each>`, reject: true},
		{name: "for-each over a grounded population may sort (rule 1)",
			body: `<xsl:for-each select="(3, 1, 2)"><xsl:sort select="." order="descending"/><xsl:value-of select="."/></xsl:for-each>`, want: "321"},
		{name: "for-each body with two consuming operands (§19.8.1)",
			body: `<xsl:for-each select="data/rec"><xsl:value-of select="v"/><xsl:value-of select="v"/></xsl:for-each>`, reject: true},
		{name: "for-each crawling select, consuming body (rule 3)",
			body: `<xsl:for-each select="//rec"><xsl:value-of select="v"/></xsl:for-each>`, reject: true},

		// §19.8.4.19 xsl:for-each-group.
		{name: "group-adjacent motionless key over striding population",
			body: `<xsl:for-each-group select="data/rec" group-adjacent="@id"><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group>`,
			want: "r1r2r3"},
		{name: "group-by not a child of xsl:fork (rule 2)",
			body: `<xsl:for-each-group select="data/rec" group-by="@id"><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group>`, reject: true},
		{name: "group-adjacent key that is not motionless (rule 3)",
			body: `<xsl:for-each-group select="data/rec" group-adjacent="string(v)"><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group>`, reject: true},
		{name: "group-by as the child of xsl:fork (rule 1 of §19.8.4.20)",
			body: `<xsl:fork><xsl:for-each-group select="data/rec" group-by="@id"><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group></xsl:fork>`,
			want: "r1r2r3"},
		{name: "grounded population: general rules",
			body: `<xsl:for-each-group select="1 to 4" group-by=". mod 2"><xsl:sort select="current-grouping-key()"/><xsl:value-of select="current-grouping-key(), sum(current-group())"/>;</xsl:for-each-group>`,
			want: "0 6;1 4;"},
		{name: "group-starting-with motionless pattern, current-group() consumed once",
			body: `<xsl:for-each-group select="data/rec" group-starting-with="rec[@id = 'r2']"><xsl:value-of select="count(current-group())"/></xsl:for-each-group>`,
			want: "12"},
		{name: "streamed for-each-group with xsl:sort, not in fork (rule 4)",
			body: `<xsl:for-each-group select="data/rec" group-adjacent="@id"><xsl:sort select="current-grouping-key()"/><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group>`, reject: true},
		{name: "crawling population, consuming body (rule 5)",
			body: `<xsl:for-each-group select="//rec" group-adjacent="@id"><xsl:value-of select="v"/></xsl:for-each-group>`, reject: true},

		// §19.8.4.22 xsl:iterate.
		{name: "iterate running total",
			body: `<xsl:iterate select="data/rec"><xsl:param name="n" select="0"/>
			  <xsl:on-completion><xsl:value-of select="$n"/></xsl:on-completion>
			  <xsl:next-iteration><xsl:with-param name="n" select="$n + v"/></xsl:next-iteration></xsl:iterate>`,
			want: "6"},
		{name: "iterate param initializer not grounded and motionless (rule 2)",
			body: `<xsl:iterate select="data/rec"><xsl:param name="n" select="string(data)"/><xsl:value-of select="v"/></xsl:iterate>`, reject: true},
		{name: "iterate on-completion uses the context item (rule 3: roaming)",
			body: `<xsl:iterate select="data/rec"><xsl:on-completion><xsl:value-of select="name(.)"/></xsl:on-completion><xsl:value-of select="v"/></xsl:iterate>`, reject: true},
		{name: "iterate crawling select, consuming body (rule 4)",
			body: `<xsl:iterate select="//rec"><xsl:value-of select="v"/></xsl:iterate>`, reject: true},
		{name: "iterate over a grounded population (rule 1)",
			body: `<xsl:iterate select="1 to 3"><xsl:param name="s" select="0"/>
			  <xsl:on-completion><xsl:value-of select="$s"/></xsl:on-completion>
			  <xsl:next-iteration><xsl:with-param name="s" select="$s + ."/></xsl:next-iteration></xsl:iterate>`,
			want: "6"},

		// §19.8.4.20 xsl:fork.
		{name: "fork: each branch consumes independently, grounded results",
			body: `<xsl:fork><xsl:sequence select="count(data/rec)"/><xsl:sequence select="sum(data/rec/v)"/></xsl:fork>`,
			want: "3 6"},
		{name: "fork branch returning streamed nodes (rule 3)",
			body: `<xsl:fork><xsl:sequence select="data/rec"/><xsl:sequence select="data"/></xsl:fork>`, reject: true},
		{name: "empty fork is grounded and motionless (rule 2)",
			body: `<xsl:fork/>[<xsl:value-of select="count(data/rec)"/>]`, want: "[3]"},

		// §19.8.4.23/.24 xsl:map and xsl:map-entry.
		{name: "map of map-entries each consuming once",
			body: `<xsl:variable name="m" as="map(*)"><xsl:map>
			  <xsl:map-entry key="'a'" select="count(data/rec)"/>
			  <xsl:map-entry key="'b'" select="sum(data/rec/v)"/></xsl:map></xsl:variable>
			  <xsl:value-of select="$m?a, $m?b"/>`,
			want: "3 6"},
		{name: "map-entry whose value is a streamed node (navigation)",
			body:   `<xsl:variable name="m" as="map(*)"><xsl:map><xsl:map-entry key="'a'" select="data/rec"/></xsl:map></xsl:variable><xsl:value-of select="map:size($m)" xmlns:map="http://www.w3.org/2005/xpath-functions/map"/>`,
			reject: true},
		{name: "map whose content is not only map-entries",
			body: `<xsl:variable name="m" as="map(*)"><xsl:map><xsl:sequence select="map{'k': count(data/rec)}"/></xsl:map></xsl:variable><xsl:value-of select="$m?k"/>`,
			want: "3"},

		// §19.8.4.25 xsl:merge.
		{name: "merge over grounded inputs is grounded and motionless",
			extra: `<xsl:variable name="g"><a k="1"/><a k="3"/></xsl:variable><xsl:variable name="h"><a k="2"/></xsl:variable>`,
			body: `<xsl:merge>
			  <xsl:merge-source name="g" select="$g/a"><xsl:merge-key select="xs:integer(@k)"/></xsl:merge-source>
			  <xsl:merge-source name="h" select="$h/a"><xsl:merge-key select="xs:integer(@k)"/></xsl:merge-source>
			  <xsl:merge-action><xsl:value-of select="current-merge-key()"/></xsl:merge-action></xsl:merge>`,
			want: "123"},
		{name: "merge-source selecting streamed nodes without for-each-*",
			body: `<xsl:merge><xsl:merge-source select="data/rec"><xsl:merge-key select="string(@id)"/></xsl:merge-source>
			  <xsl:merge-action><xsl:value-of select="current-merge-key()"/></xsl:merge-action></xsl:merge>`,
			reject: true},

		// §19.8.4.37 xsl:source-document nested in a streamable template.
		{name: "nested source-document is grounded with the sweep of @href",
			body: `<xsl:source-document href="d.xml"><xsl:value-of select="count(//rec)"/></xsl:source-document>`, want: "3"},
		{name: "current-group() across the source-document boundary (rule 1)",
			body:   `<xsl:for-each-group select="data/rec" group-adjacent="@id"><xsl:source-document href="d.xml"><xsl:value-of select="count(current-group())"/></xsl:source-document></xsl:for-each-group>`,
			reject: true},

		// §19.8.4.5 xsl:apply-templates.
		{name: "apply-templates into the streamable unnamed mode",
			extra: recTpl, body: `<xsl:apply-templates select="data/rec"/>`, want: "123"},
		{name: "apply-templates mode=#current",
			extra: recTpl, body: `<xsl:apply-templates select="data/rec" mode="#current"/>`, want: "123"},
		{name: "apply-templates into a mode not declared streamable (rule 3)",
			extra: gRecTpl, body: `<xsl:apply-templates select="data/rec" mode="g"/>`, reject: true},
		{name: "apply-templates over streamed nodes with xsl:sort (rule 2)",
			extra: recTpl, body: `<xsl:apply-templates select="data/rec"><xsl:sort select="v"/></xsl:apply-templates>`, reject: true},
		{name: "apply-templates crawling select (rule 4)",
			extra: recTpl, body: `<xsl:apply-templates select="//rec"/>`, reject: true},
		{name: "apply-templates over a grounded copy may sort, any mode (rule 1)",
			extra: gRecTpl, body: `<xsl:apply-templates select="copy-of(data/rec)" mode="g"><xsl:sort select="v" order="descending"/></xsl:apply-templates>`,
			want: "321"},

		// Other instructions governed by the general rules.
		{name: "perform-sort of streamed nodes: navigation (§19.8.4.33)",
			body: `<xsl:perform-sort select="data/rec"><xsl:sort select="v"/></xsl:perform-sort>`, reject: true},
		{name: "xsl:number of the streamed context item (§19.8.4.30)",
			body: `<xsl:number/>`, reject: true},
		{name: "xsl:number with value is motionless",
			body: `<xsl:number value="3" format="i"/>`, want: "iii"},
		{name: "analyze-string over a grounded string",
			body: `<xsl:analyze-string select="'a1b'" regex="[0-9]"><xsl:matching-substring>[<xsl:value-of select="."/>]</xsl:matching-substring><xsl:non-matching-substring><xsl:value-of select="."/></xsl:non-matching-substring></xsl:analyze-string>`,
			want: "a[1]b"},
		{name: "try body consumes, catch is motionless (§19.8.4.39)",
			body: `<xsl:try><xsl:value-of select="sum(data/rec/v)"/><xsl:catch><xsl:value-of select="'failed'"/></xsl:catch></xsl:try>`,
			want: "6"},
		{name: "try body and catch both consume: not both (§19.8.4.39 Note)",
			body:   `<xsl:try><xsl:value-of select="sum(data/rec/v)"/><xsl:catch><xsl:value-of select="count(data/rec)"/></xsl:catch></xsl:try>`,
			reject: true},
		{name: "several catches may all consume: only one runs (§19.8.4.39 Note)",
			body: `<xsl:try><xsl:value-of select="'t'"/><xsl:catch errors="err:XTDE0000" xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="count(data/rec)"/></xsl:catch><xsl:catch><xsl:value-of select="sum(data/rec/v)"/></xsl:catch></xsl:try>`,
			want: "t"},
		{name: "copy-of at the top of the rule is grounded and consuming",
			body: `<xsl:copy-of select="data/rec/v"/>`, want: "123"},
		{name: "two consuming instructions in one sequence constructor",
			body: `<xsl:value-of select="data/rec/v"/><xsl:value-of select="count(data/rec)"/>`, reject: true},
		{name: "variable with a node type bound to a streamed node (§19.8.4.41)",
			body: `<xsl:variable name="d" as="element()" select="data"/><xsl:value-of select="name($d)"/>`, reject: true},
		{name: "variable with an atomic type atomizes",
			body: `<xsl:variable name="d" as="xs:string" select="string(data)"/><xsl:value-of select="$d"/>`, want: "123"},
		{name: "variable of map type then looked up",
			body: `<xsl:variable name="m" as="map(xs:string, xs:integer)" select="map{'n': count(data/rec)}"/><xsl:value-of select="$m('n')"/>`,
			want: "3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stCCheck(t, stCModeSheet(c.body, c.extra), c.reject, c.want)
		})
	}
}

// TestStreamableTemplateResultPosture pins §6.6.4 condition 4: the body of a
// template rule in a streamable mode, type-adjusted to its declared result
// type, must be grounded — a rule cannot hand streamed nodes back.
func TestStreamableTemplateResultPosture(t *testing.T) {
	sheet := func(as, body string) string {
		return `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:output method="text"/>
  <xsl:mode streamable="yes"/>
  <xsl:template match="/"` + as + `>` + body + `</xsl:template>
</xsl:stylesheet>`
	}
	stCCheck(t, sheet("", `<xsl:sequence select="data/rec"/>`), true, "")
	// Declared atomic: the result is atomized, hence grounded (consuming).
	stCCheck(t, sheet(` as="xs:string"`, `<xsl:sequence select="string(data)"/>`), false, "123")
	stCCheck(t, sheet(` as="xs:string*"`, `<xsl:sequence select="data/rec/@id"/>`), false, "r1 r2 r3")
}

// TestStreamableModeMatchPatterns pins §19.8.10: a template rule of a
// streamable mode needs a motionless match pattern; a predicate that reads
// children (p[b]) is the REC's own example of one that is not.
func TestStreamableModeMatchPatterns(t *testing.T) {
	sheet := func(match string) string {
		return `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:mode streamable="yes"/>
  <xsl:template match="` + match + `"><xsl:value-of select="@id"/></xsl:template>
  <xsl:template match="text()"/>
</xsl:stylesheet>`
	}
	stCCheck(t, sheet("rec"), false, "r1r2r3")
	stCCheck(t, sheet("rec[@id != 'r2']"), false, "r1r3")
	stCCheck(t, sheet("rec[v]"), true, "")
}

// TestStreamableSourceDocumentGate covers §18.1's two conditions on an
// xsl:source-document declared streamable: it must be declared so, and its
// sequence constructor must be GROUNDED. A body that classifies but returns
// streamed nodes, and a body whose verdict a REC rule states unconditionally
// (group-by outside xsl:fork, §19.8.4.19 rule 2), are XTSE3430.
func TestStreamableSourceDocumentGate(t *testing.T) {
	sheet := func(body string) string {
		return `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
	}
	sd := func(attr, body string) string {
		return `<xsl:source-document href="d.xml"` + attr + `>` + body + `</xsl:source-document>`
	}
	cases := []struct {
		name   string
		body   string
		reject bool
		want   string
	}{
		{"grounded body", sd(` streamable="yes"`, `<xsl:for-each select="data/rec"><xsl:value-of select="v"/></xsl:for-each>`), false, "123"},
		{"streamable=true is the same boolean", sd(` streamable="true"`, `<xsl:value-of select="count(data/rec)"/>`), false, "3"},
		{"striding body returns streamed nodes", sd(` streamable="yes"`, `<xsl:sequence select="data/rec/v"/>`), true, ""},
		{"group-by outside xsl:fork", sd(` streamable="yes"`, `<xsl:for-each-group select="data/rec" group-by="@id"><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group>`), true, ""},
		{"not declared streamable: not analysed", sd(``, `<xsl:sequence select="data/rec/v"/>`), false, "123"},
		{"streamable=no: not analysed", sd(` streamable="no"`, `<xsl:sequence select="data/rec/v"/>`), false, "123"},
		{"inside xsl:for-each-group the group may be reached",
			`<xsl:for-each-group select="1 to 2" group-by="."><xsl:value-of select="current-grouping-key()"/>` +
				sd(` streamable="yes"`, `<xsl:value-of select="count(data/rec) + count(current-group())"/>`) + `</xsl:for-each-group>`,
			false, "1424"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stCCheck(t, sheet(c.body), c.reject, c.want)
		})
	}
}

// TestStreamableMergeSourceGate covers §15.4's closed list of conditions on an
// xsl:merge-source declared streamable="yes": for-each-source must be present,
// sort-before-merge must be absent or "no", and @select — assessed against a
// striding document node — must be striding or grounded and motionless or
// consuming.
func TestStreamableMergeSourceGate(t *testing.T) {
	sheet := func(attrs, sel string) string {
		return `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:merge>
      <xsl:merge-source ` + attrs + ` select="` + sel + `">
        <xsl:merge-key select="string(@id)"/>
      </xsl:merge-source>
      <xsl:merge-action><xsl:value-of select="current-merge-key()"/></xsl:merge-action>
    </xsl:merge>
  </xsl:template>
</xsl:stylesheet>`
	}
	stCCheck(t, sheet(`for-each-source="'d.xml'" streamable="yes"`, "data/rec"), false, "r1r2r3")
	stCCheck(t, sheet(`streamable="yes"`, "data/rec"), true, "")
	stCCheck(t, sheet(`for-each-source="'d.xml'" streamable="yes" sort-before-merge="yes"`, "data/rec"), true, "")
	// //rec is crawling, which §15.4 does not allow.
	stCCheck(t, sheet(`for-each-source="'d.xml'" streamable="yes"`, "//rec"), true, "")
	// Not declared streamable: none of the conditions apply.
	stCCheck(t, sheet(`for-each-source="'d.xml'"`, "//rec"), false, "r1r2r3")
}

// TestStrictStreamingIsPerStylesheet checks that the XTSE3430 gate belongs to
// the stylesheet compiled with it: the same text compiled without
// StrictStreaming falls back to unstreamed processing, as XTSE3430's own
// "unless the user has indicated ..." clause permits.
func TestStrictStreamingIsPerStylesheet(t *testing.T) {
	sheet := stCModeSheet(`<xsl:value-of select="data/rec/v"/><xsl:value-of select="count(data/rec)"/>`, "")
	if _, err := CompileWith(sheet, "", CompileOptions{Features: stCStrict()}); err == nil || !strings.Contains(err.Error(), "XTSE3430") {
		t.Fatalf("strict: %v, want XTSE3430", err)
	}
	ss, err := CompileWith(sheet, "", CompileOptions{Features: &Features{StreamingClaim: true}})
	if err != nil {
		t.Fatalf("lenient compile: %v", err)
	}
	out, _, err := ss.Transform(stCDoc, nil)
	if err != nil {
		t.Fatal(err)
	}
	// xsl:value-of joins the items of its select with a single space.
	if out != "1 2 33" {
		t.Fatalf("output %q, want %q", out, "1 2 33")
	}
}
