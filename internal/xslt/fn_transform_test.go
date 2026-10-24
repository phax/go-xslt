package xslt

import (
	"strings"
	"testing"
)

// fn:transform (F&O 3.1 §16.3.2), called from inside a stylesheet. The SOURCE
// document of each run is the INNER stylesheet, so the outer stylesheet hands
// it on as "stylesheet-node": / (or "stylesheet-text": serialize(/)) and no
// file is ever read.

// mtEOuter builds the calling stylesheet: its root template writes the string
// value of expr as text.
func mtEOuter(expr string) string {
	return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:output method="text"/>
  <xsl:template match="/"><xsl:value-of select="` + expr + `"/></xsl:template>
</xsl:stylesheet>`
}

// mtERunOuter runs mtEOuter(expr) over the inner stylesheet text.
func mtERunOuter(t *testing.T, inner, expr string) (string, error) {
	t.Helper()
	ss, err := Compile(mtEOuter(expr))
	if err != nil {
		t.Fatalf("compile outer: %v", err)
	}
	out, _, err := ss.Transform(inner, nil)
	return out, err
}

const mtEInnerHead = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="urn:my" exclude-result-prefixes="xs my">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
`

// mtEInner is the inner stylesheet: a root template copying doc's text into
// <out>, plus the named template, mode, function, and parameters the cases
// below select.
const mtEInner = mtEInnerHead + `
  <xsl:param name="g" select="'default-g'"/>
  <xsl:param name="st" static="yes" select="'static-default'"/>
  <xsl:template match="/"><out><xsl:value-of select="doc"/></out></xsl:template>
  <xsl:template match="/" mode="m"><out>mode-m:<xsl:value-of select="doc"/></out></xsl:template>
  <xsl:template match="item"><i><xsl:value-of select="."/></i></xsl:template>
  <xsl:template name="main">
    <xsl:param name="p" select="'default-p'"/>
    <out>main:<xsl:value-of select="$p"/></out>
  </xsl:template>
  <xsl:template name="tun">
    <xsl:call-template name="tun-inner"/>
  </xsl:template>
  <xsl:template name="tun-inner">
    <xsl:param name="tp" tunnel="yes" select="'no-tunnel'"/>
    <out>tunnel:<xsl:value-of select="$tp"/></out>
  </xsl:template>
  <xsl:template name="global"><out><xsl:value-of select="$g"/></out></xsl:template>
  <xsl:template name="static"><out><xsl:value-of select="$st"/></out></xsl:template>
  <xsl:template name="req">
    <xsl:param name="r" required="yes"/>
    <out><xsl:value-of select="$r"/></out>
  </xsl:template>
  <xsl:template name="merge"><out><xsl:value-of select="'a'"/><xsl:value-of select="'b'"/><xsl:value-of select="'c'"/></out></xsl:template>
  <xsl:template name="fail"><xsl:message terminate="yes">stop</xsl:message></xsl:template>
  <xsl:template name="secondary">
    <xsl:result-document href="sec.xml"><s>second</s></xsl:result-document>
    <p>primary</p>
  </xsl:template>
  <xsl:template name="only-secondary">
    <xsl:result-document href="a.xml"><a/></xsl:result-document>
  </xsl:template>
  <xsl:function name="my:add" as="xs:integer" visibility="public">
    <xsl:param name="a" as="xs:integer"/>
    <xsl:param name="b" as="xs:integer"/>
    <xsl:sequence select="$a + $b"/>
  </xsl:function>
  <xsl:function name="my:seq" as="xs:integer*" visibility="public">
    <xsl:sequence select="1 to 3"/>
  </xsl:function>
</xsl:stylesheet>`

// mtEInnerCtx reads the global context item in a global variable; it is a
// separate stylesheet because a global variable whose context item is absent
// may be reported as XPDY0002 even when nothing references it.
const mtEInnerCtx = mtEInnerHead + `
  <xsl:variable name="ctx" select="if (. instance of document-node()) then 'document' else if (. instance of node()) then name(.) else string(.)"/>
  <xsl:template name="ctx"><out><xsl:value-of select="$ctx"/></out></xsl:template>
</xsl:stylesheet>`

const mtESrc = `parse-xml('&lt;doc&gt;this&lt;/doc&gt;')`

func TestFnTransformOptions(t *testing.T) {
	cases := []struct {
		name, expr, want string
	}{
		// stylesheet-node / stylesheet-text with source-node; the default
		// delivery-format is "document", so ?output is a document node.
		{"node", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `})?output/out`, "this"},
		{"text", `transform(map{'stylesheet-text': serialize(/), 'source-node': ` + mtESrc + `})?output/out`, "this"},
		{"document-is-node", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `})?output instance of document-node()`, "true"},
		// delivery-format "serialized": the output serialized per the
		// stylesheet's own xsl:output.
		{"serialized", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'delivery-format': 'serialized'})?output`, "<out>this</out>"},
		// serialization-params override the unnamed xsl:output.
		{"serialization-params-text", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'delivery-format': 'serialized', 'serialization-params': map{'method': 'text'}})?output`, "this"},
		{"serialization-params-decl", `starts-with(transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'delivery-format': 'serialized', 'serialization-params': map{'omit-xml-declaration': false()}})?output, '&lt;?xml')`, "true"},
		// initial-mode selects the mode the initial apply-templates uses.
		{"initial-mode", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'initial-mode': QName('', 'm')})?output/out`, "mode-m:this"},
		// initial-template, with and without template-params.
		{"initial-template", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'main')})?output/out`, "main:default-p"},
		{"template-params", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'main'), 'template-params': map{QName('', 'p'): 'given'}})?output/out`, "main:given"},
		{"tunnel-params", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'tun'), 'tunnel-params': map{QName('', 'tp'): 'carried'}})?output/out`, "tunnel:carried"},
		// stylesheet-params set global parameters; static-params static ones.
		{"stylesheet-params", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'global'), 'stylesheet-params': map{QName('', 'g'): 'set-g'}})?output/out`, "set-g"},
		{"stylesheet-params-default", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'global')})?output/out`, "default-g"},
		{"static-params", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'static'), 'static-params': map{QName('', 'st'): 'static-set'}})?output/out`, "static-set"},
		// initial-function with function-params; "raw" delivers the
		// function's own result sequence, not a document.
		{"initial-function-raw", `transform(map{'stylesheet-node': /, 'initial-function': QName('urn:my', 'add'), 'function-params': [2, 3], 'delivery-format': 'raw'})?output`, "5"},
		{"raw-integer", `transform(map{'stylesheet-node': /, 'initial-function': QName('urn:my', 'add'), 'function-params': [2, 3], 'delivery-format': 'raw'})?output instance of xs:integer`, "true"},
		{"raw-sequence", `string-join(transform(map{'stylesheet-node': /, 'initial-function': QName('urn:my', 'seq'), 'function-params': [], 'delivery-format': 'raw'})?output, ',')`, "1,2,3"},
		// initial-match-selection: apply-templates to each item in turn.
		{"initial-match-selection", `string-join(transform(map{'stylesheet-node': /, 'initial-match-selection': parse-xml('&lt;r&gt;&lt;item&gt;a&lt;/item&gt;&lt;item&gt;b&lt;/item&gt;&lt;/r&gt;')//item})?output/i, ',')`, "a,b"},
		// post-process is applied to every result, with its key.
		{"post-process", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'post-process': function($k, $v) { $k || '=' || $v/out }})?output`, "output=this"},
		// base-output-uri keys the principal result; a secondary result
		// document is keyed by its absolute URI (resolved against it).
		{"base-output-uri", `string-join(sort(map:keys(transform(map{'stylesheet-node': /, 'initial-template': QName('', 'secondary'), 'base-output-uri': 'http://example.com/out/main.xml'}))), ' ')`, "http://example.com/out/main.xml http://example.com/out/sec.xml"},
		{"secondary-value", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'secondary'), 'base-output-uri': 'http://example.com/out/main.xml'})('http://example.com/out/sec.xml')/s`, "second"},
		{"secondary-serialized", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'secondary'), 'base-output-uri': 'http://example.com/out/main.xml', 'delivery-format': 'serialized'})('http://example.com/out/sec.xml')`, "<s>second</s>"},
		// An absent principal result has no "output" entry (bug 30209,
		// fn-transform-13a).
		{"no-principal", `map:contains(transform(map{'stylesheet-node': /, 'initial-template': QName('', 'only-secondary'), 'base-output-uri': 'http://example.com/o.xml'}), 'http://example.com/o.xml')`, "false"},
		// xslt-version is accepted when it is a single number.
		{"xslt-version", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'xslt-version': 3.0})?output/out`, "this"},
		// "raw" for a template delivers its result without a document node.
		{"raw-template", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'main'), 'delivery-format': 'raw'})?output instance of element(out)`, "true"},
		// Adjacent text nodes in constructed content are merged into one
		// (XSLT 3.0 §5.7.1).
		{"merged-text", `count(transform(map{'stylesheet-node': /, 'initial-template': QName('', 'merge')})?output/out/text())`, "1"},
		// Serialization 3.1: character maps apply to text content,
		// cdata-section-elements wrap the listed element's text, and the
		// declaration carries standalone when asked to.
		{"ser-character-map", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'delivery-format': 'serialized', 'serialization-params': map{'use-character-maps': map{'t': 'T'}}})?output`, "<out>This</out>"},
		{"ser-cdata", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'delivery-format': 'serialized', 'serialization-params': map{'cdata-section-elements': QName('', 'out')}})?output`, "<out><![CDATA[this]]></out>"},
		{"ser-standalone", `contains(transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'delivery-format': 'serialized', 'serialization-params': map{'omit-xml-declaration': false(), 'standalone': 'yes'}})?output, 'standalone=&quot;yes&quot;')`, "true"},
		// requested-properties that this processor satisfies run normally.
		{"requested-satisfied", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'requested-properties': map{QName('http://www.w3.org/1999/XSL/Transform', 'supports-namespace-axis'): true()}})?output/out`, "this"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mtERunOuter(t, mtEInner, c.expr)
			if err != nil {
				t.Fatalf("transform: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestFnTransformGlobalContextItem checks the global-context-item option and
// its default under source-node.
func TestFnTransformGlobalContextItem(t *testing.T) {
	cases := []struct {
		name, expr, want string
	}{
		// global-context-item: an explicit item for global variables.
		{"global-context-item", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'ctx'), 'global-context-item': 'atomic'})?output/out`, "atomic"},
		// With source-node and no global-context-item, the global context
		// item is the ROOT of the tree containing source-node
		// (fn-transform-82b), not source-node itself.
		{"source-node-root-is-global", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `/doc, 'initial-template': QName('', 'ctx')})?output/out`, "document"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mtERunOuter(t, mtEInnerCtx, c.expr)
			if err != nil {
				t.Fatalf("transform: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestFnTransformErrors(t *testing.T) {
	cases := []struct {
		name, expr, code string
	}{
		// FOXT0002: invalid or inconsistent options (§16.3.2).
		{"two-stylesheets", `transform(map{'stylesheet-node': /, 'stylesheet-text': serialize(/)})`, "FOXT0002"},
		{"no-stylesheet", `transform(map{'source-node': ` + mtESrc + `})`, "FOXT0002"},
		{"template-and-mode", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'main'), 'initial-mode': QName('', 'm')})`, "FOXT0002"},
		{"source-and-selection", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'initial-match-selection': ` + mtESrc + `})`, "FOXT0002"},
		{"bad-delivery-format", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'delivery-format': 'doc'})`, "FOXT0002"},         // fn-transform-err-13
		{"param-key-not-qname", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'stylesheet-params': map{'g': 1}})`, "FOXT0002"}, // fn-transform-err-18
		{"empty-stylesheet-node", `transform(map{'stylesheet-node': ()})`, "FOXT0002"},
		{"unknown-package", `transform(map{'package-name': 'urn:no-such-package'})`, "FOXT0002"},
		// xslt-version is xs:decimal: a string is a type error (fn-transform-err-4).
		{"xslt-version-string", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'xslt-version': '3.0'})`, "XPTY0004"},
		// The option values must match their declared map types.
		{"stylesheet-params-not-map", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'stylesheet-params': 'x'})`, "XPTY0004"},
		{"static-params-not-map", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'static-params': 'x'})`, "XPTY0004"},
		// $options is map(*): anything else is a type error.
		{"arg-not-map", `transform('x')`, "XPTY0004"},
		{"arg-two-maps", `transform((map{}, map{}))`, "XPTY0004"},
		// An error raised by the invoked transformation keeps its own code
		// (fn-transform-err-15: missing required template parameter).
		// function-params defaults to the empty array and the arity of the
		// initial function is inferred from its length, so omitting it asks
		// for my:add#0, which does not exist: XSLT's XTDE0041, whose code
		// fn:transform retains.
		{"function-without-params", `transform(map{'stylesheet-node': /, 'initial-function': QName('urn:my', 'add')})`, "XTDE0041"},
		{"inner-required-param", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'req')})`, "XTDE0700"},
		{"inner-terminate", `transform(map{'stylesheet-node': /, 'initial-template': QName('', 'fail')})`, "XTMM9000"},
		// A static error in the supplied stylesheet text.
		{"inner-static-error", `transform(map{'stylesheet-text': '&lt;xsl:stylesheet version=&quot;3.0&quot; xmlns:xsl=&quot;http://www.w3.org/1999/XSL/Transform&quot;&gt;&lt;xsl:bogus/&gt;&lt;/xsl:stylesheet&gt;'})`, "XTSE0010"},
		// requested-properties this processor cannot honor: F&O 3.1 lets a
		// processor either ignore the request or raise FOXT0001; this one
		// documents the error (fn_transform.go, xfCheckRequestedProperties).
		{"requested-unavailable", `transform(map{'stylesheet-node': /, 'source-node': ` + mtESrc + `, 'requested-properties': map{QName('http://www.w3.org/1999/XSL/Transform', 'supports-namespace-axis'): false()}})`, "FOXT0001"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := mtERunOuter(t, mtEInner, c.expr)
			if err == nil {
				t.Fatalf("want err:%s, got success", c.code)
			}
			if !strings.Contains(err.Error(), "err:"+c.code) {
				t.Errorf("want err:%s, got %v", c.code, err)
			}
		})
	}
}
