package xslt

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xpath"
)

// entrySheet is shared by the entry-point tests (XSLT 3.0 §2.3).
const entrySheet = `<xsl:stylesheet version="3.0" ` + specNS + `>
<xsl:output method="text"/>
<xsl:param name="g" select="'G'"/>
<xsl:param name="typed" as="xs:integer" select="0"/>
<xsl:template name="main"><xsl:param name="p" select="'dp'"/><xsl:param name="t" tunnel="yes" select="'dt'"/>main:<xsl:value-of select="$p, $g"/>:<xsl:call-template name="inner"/></xsl:template>
<xsl:template name="inner"><xsl:param name="t" tunnel="yes" select="'none'"/><xsl:value-of select="$t"/></xsl:template>
<xsl:template name="f:main">ns-main</xsl:template>
<xsl:template name="req"><xsl:param name="r" required="yes"/><xsl:value-of select="$r"/></xsl:template>
<xsl:template name="ctx"><xsl:value-of select="name(.)"/></xsl:template>
<xsl:template match="/">root:<xsl:value-of select="$g"/></xsl:template>
<xsl:template match="*" mode="m">m:<xsl:value-of select="name()"/></xsl:template>
<xsl:template match=".[. instance of xs:integer]">i<xsl:value-of select="."/></xsl:template>
<xsl:template match="b">B</xsl:template>
<xsl:function name="f:add" visibility="public"><xsl:param name="a"/><xsl:param name="b"/><xsl:sequence select="$a + $b"/></xsl:function>
<xsl:function name="f:nodes" visibility="public"><e>1</e><e>2</e></xsl:function>
<xsl:function name="f:private"><xsl:sequence select="1"/></xsl:function>
<xsl:template name="typed"><xsl:value-of select="$typed + 1"/></xsl:template>
</xsl:stylesheet>`

func TestEntrySpec(t *testing.T) {
	ss, err := Compile(entrySheet)
	if err != nil {
		t.Fatal(err)
	}
	src := `<a><b/><c/></a>`
	cases := []struct {
		name string
		src  string
		e    Entry
		want string
		code string
	}{
		{name: "initial template, no source", e: Entry{Template: "main"}, want: "main:dp G:none"},
		{name: "initial template with parameters", e: Entry{Template: "main",
			TemplateParams: []EntryParam{{Name: "p", Select: "'x'"}, {Name: "t", Select: "'tun'", Tunnel: true}},
			Params:         map[string]string{"g": "host"}}, want: "main:x host:tun"},
		{name: "initial template by Clark name", e: Entry{Template: "{urn:f}main"}, want: "ns-main"},
		{name: "unknown initial template", e: Entry{Template: "nope"}, code: "XTDE0040"},
		// XTDE0700: a required parameter not supplied.
		{name: "required template parameter missing", e: Entry{Template: "req"}, code: "XTDE0700"},
		{name: "required template parameter supplied", e: Entry{Template: "req",
			TemplateParams: []EntryParam{{Name: "r", Select: "42"}}}, want: "42"},
		{name: "initial template with a context item", src: src, e: Entry{Template: "ctx", ContextItem: "/a/b"}, want: "b"},
		// §2.3.? initial mode.
		{name: "default entry", src: src, e: Entry{}, want: "root:G"},
		{name: "initial mode", src: src, e: Entry{Mode: "m"}, want: "m:a"},
		{name: "initial mode with context item", src: src, e: Entry{Mode: "m", ContextItem: "/a/c"}, want: "m:c"},
		{name: "#unnamed initial mode", src: src, e: Entry{Mode: "#unnamed"}, want: "root:G"},
		{name: "undeclared initial mode", src: src, e: Entry{Mode: "zz"}, code: "XTDE0045"},
		{name: "initial match selection of atomic values", e: Entry{MatchSelection: "(1, 2)"}, want: "i1i2"},
		{name: "initial match selection of nodes", src: src, e: Entry{MatchSelection: "//b"}, want: "B"},
		// XTDE0044: no initial match selection and no xsl:initial-template.
		{name: "no source and no initial template", e: Entry{}, code: "XTDE0044"},
		// §2.3.? initial function.
		{name: "initial function", e: Entry{Function: "{urn:f}add", FuncArgs: []string{"1", "2"}}, want: "3"},
		{name: "initial function by Q{} name", e: Entry{Function: "Q{urn:f}add", FuncArgs: []string{"1", "2"}}, want: "3"},
		{name: "initial function returning nodes", e: Entry{Function: "{urn:f}nodes"}, want: "12"},
		{name: "initial function wrong arity", e: Entry{Function: "{urn:f}add", FuncArgs: []string{"1"}}, code: "XTDE0041"},
		// Only a public or final function is an entry point.
		{name: "private initial function", e: Entry{Function: "{urn:f}private"}, code: "XTDE0041"},
		// Global parameters supplied as expressions keep their type.
		{name: "typed global parameter", e: Entry{Template: "typed", ParamSelects: map[string]string{"typed": "41"}}, want: "42"},
	}
	for _, c := range cases {
		rr, err := ss.TransformEntry(c.src, c.e, "")
		if c.code != "" {
			if err == nil || !strings.Contains(err.Error(), c.code) {
				t.Errorf("%s: want %s, got %v", c.name, c.code, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if rr.Output != c.want {
			t.Errorf("%s: got %q want %q", c.name, rr.Output, c.want)
		}
	}
}

func TestEntrySpecFunctionValue(t *testing.T) {
	ss, err := Compile(entrySheet)
	if err != nil {
		t.Fatal(err)
	}
	rr, err := ss.TransformEntry("", Entry{Function: "{urn:f}add", FuncArgs: []string{"40", "2"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	items := xpath.Items(rr.Value)
	if len(items) != 1 {
		t.Fatalf("want one item, got %d", len(items))
	}
	if a, ok := items[0].(*xpath.Atomic); !ok || xpath.ToString(xpath.FromItems(items)) != "42" || !a.IsNumeric() {
		t.Errorf("got %#v", items[0])
	}
}

// XSLT 3.0 §5.3.? xsl:global-context-item and §2.3.? xsl:initial-template,
// through the entry-point API.
func TestEntrySpecGlobalContextItem(t *testing.T) {
	cases := []struct {
		name, body, src string
		e               Entry
		want, code      string
	}{
		{name: "xsl:initial-template runs without a source", body: `<xsl:template name="xsl:initial-template">init</xsl:template>`, want: "init"},
		{name: "global context item as element()", src: "<doc/>", code: "XTTE0590", body: `
<xsl:global-context-item as="element()"/><xsl:template match="/">x</xsl:template>`},
		{name: "global context item as document-node()", src: "<doc/>", want: "x", body: `
<xsl:global-context-item as="document-node()" use="required"/><xsl:template match="/">x</xsl:template>`},
		{name: "required global context item absent", e: Entry{Template: "main"}, code: "XTDE3086", body: `
<xsl:global-context-item use="required"/><xsl:template name="main">x</xsl:template>`},
		{name: "global variables see the global context item", src: "<doc n='3'/>", e: Entry{Template: "main"}, want: "3", body: `
<xsl:global-context-item use="required"/><xsl:variable name="n" select="string(/doc/@n)"/><xsl:template name="main"><xsl:value-of select="$n"/></xsl:template>`},
	}
	for _, c := range cases {
		ss, err := Compile(specSheet(c.body))
		if err != nil {
			t.Errorf("%s: compile: %v", c.name, err)
			continue
		}
		rr, err := ss.TransformEntry(c.src, c.e, "")
		if c.code != "" {
			if err == nil || !strings.Contains(err.Error(), c.code) {
				t.Errorf("%s: want %s, got %v", c.name, c.code, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
		} else if rr.Output != c.want {
			t.Errorf("%s: got %q want %q", c.name, rr.Output, c.want)
		}
	}
}

// XSLT 3.0 §9.5 global variables and parameters: circularity and required
// parameters.
func TestEntrySpecGlobals(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "forward reference between globals", want: "3", sheet: `
<xsl:variable name="a" select="$b + 1"/><xsl:variable name="b" select="2"/>
<xsl:template match="/"><xsl:value-of select="$a"/></xsl:template>`},
		// XTDE0640: a circular definition that is actually evaluated.
		{name: "circular globals", code: "XTDE0640", sheet: `
<xsl:variable name="a" select="$b"/><xsl:variable name="b" select="$a"/>
<xsl:template match="/"><xsl:value-of select="$a"/></xsl:template>`},
		{name: "circular through a function", code: "XTDE0640", sheet: `
<xsl:variable name="a" select="f:g()"/>
<xsl:function name="f:g"><xsl:sequence select="$a"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="$a"/></xsl:template>`},
		// XTTE0570: a global's value does not match its type.
		{name: "global variable type mismatch", code: "XTTE0570", sheet: `
<xsl:variable name="v" as="xs:integer" select="'abc'"/>
<xsl:template match="/"><xsl:value-of select="$v"/></xsl:template>`},
		{name: "global variable with content and as", want: "2", sheet: `
<xsl:variable name="v" as="xs:integer"><xsl:value-of select="2"/></xsl:variable>
<xsl:template match="/"><xsl:value-of select="$v"/></xsl:template>`},
		{name: "global variable from the source document", src: `<r n="5"/>`, want: "5", sheet: `
<xsl:variable name="n" select="/r/@n"/>
<xsl:template match="/"><xsl:value-of select="$n"/></xsl:template>`},
		{name: "local variable shadows a global", want: "local", sheet: `
<xsl:variable name="v" select="'global'"/>
<xsl:template match="/"><xsl:variable name="v" select="'local'"/><xsl:value-of select="$v"/></xsl:template>`},
	})
}
