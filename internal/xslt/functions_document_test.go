package xslt

import "testing"

// XSLT 3.0 §20.1 fn:document, and fn:stream-available (§18.1.3).
func TestXSLTFuncDocument(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "a.xml", `<a xml:id="top"><b xml:id="bee">B</b></a>`)
	specWrite(t, dir, "sub/c.xml", `<c>C</c>`)
	specWrite(t, dir, "sub/list.xml", `<l><h>c.xml</h></l>`)
	specWrite(t, dir, "prolog.xml", "\ufeff<?xml version=\"1.0\"?>\n<!-- c --><?pi x?>\n<!DOCTYPE p [<!ELEMENT p ANY>]>\n<p/>")
	specWrite(t, dir, "dtdonly.xml", `<!DOCTYPE p [<!ELEMENT p ANY>]>`)
	specWrite(t, dir, "text.txt", `just text`)
	specWrite(t, dir, "trunc.xml", `<p><q>`)
	specRunAll(t, dir, []specCase{
		{name: "relative URI against the stylesheet base", want: "B",
			sheet: `<xsl:template match="/"><xsl:value-of select="document('a.xml')/a/b"/></xsl:template>`},
		{name: "same URI twice gives one document", want: "1",
			sheet: `<xsl:template match="/"><xsl:value-of select="count(document(('a.xml','a.xml')))"/></xsl:template>`},
		{name: "two documents", want: "2",
			sheet: `<xsl:template match="/"><xsl:value-of select="count(document(('a.xml','sub/c.xml')))"/></xsl:template>`},
		{name: "fragment identifier selects an element by ID", want: "b",
			sheet: `<xsl:template match="/"><xsl:value-of select="name(document('a.xml#bee'))"/></xsl:template>`},
		{name: "unknown fragment selects nothing", want: "0",
			sheet: `<xsl:template match="/"><xsl:value-of select="count(document('a.xml#none'))"/></xsl:template>`},
		// A node in the first argument resolves against that node's base URI.
		{name: "node argument resolves against its own base URI", want: "C",
			sheet: `<xsl:template match="/"><xsl:value-of select="document(document('sub/list.xml')/l/h)"/></xsl:template>`},
		// The second argument supplies the base URI for every item.
		{name: "base-node argument", want: "C",
			sheet: `<xsl:template match="/"><xsl:value-of select="document('c.xml', document('sub/list.xml'))"/></xsl:template>`},
		{name: "empty sequence argument", want: "0",
			sheet: `<xsl:template match="/"><xsl:value-of select="count(document(()))"/></xsl:template>`},
		// document('') is the stylesheet module itself.
		{name: "document('') is the stylesheet", want: "f:marker",
			sheet: `<xsl:variable name="m" select="'unused'"/><f:marker/>
<xsl:template match="/"><xsl:value-of select="name(document('')/*/f:marker)"/></xsl:template>`},
		// The fragment of an empty URI with a base node names an element of
		// that node's document.
		{name: "document('#id', node)", src: `<r xml:id="r1"><x xml:id="x1">X</x></r>`, want: "X",
			sheet: `<xsl:template match="/"><xsl:value-of select="document('#x1', /)"/></xsl:template>`},
		{name: "stream-available", want: "true true false false false true false",
			sheet: `<xsl:template match="/"><xsl:value-of select="stream-available('a.xml'), stream-available('prolog.xml'), stream-available('missing.xml'), stream-available('text.txt'), stream-available('dtdonly.xml'), stream-available('trunc.xml'), stream-available('')"/></xsl:template>`},
	})
}

// XSLT 3.0 §20.? fn:current-output-uri: the base output URI in the principal
// output, the resolved href inside xsl:result-document, and the empty
// sequence in temporary output state (a variable or function body).
func TestXSLTFuncCurrentOutputURI(t *testing.T) {
	dir := t.TempDir()
	sheet := specSheet(`
<xsl:function name="f:u"><xsl:sequence select="current-output-uri()"/></xsl:function>
<xsl:template match="/">
  <xsl:variable name="v"><xsl:value-of select="count(current-output-uri())"/></xsl:variable>
  <xsl:value-of select="current-output-uri(), $v, count(f:u())" separator="|"/>
  <xsl:result-document href="sub.txt" method="text"><xsl:value-of select="current-output-uri()"/></xsl:result-document>
</xsl:template>`)
	ss, err := Compile(sheet)
	if err != nil {
		t.Fatal(err)
	}
	base := "file:///tmp/out/main.txt"
	rr, err := ss.TransformEntry("<doc/>", Entry{BaseOutputURI: base}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := base + "|0|0"; rr.Output != want {
		t.Errorf("principal: got %q want %q", rr.Output, want)
	}
	if len(rr.Secondary) != 1 {
		t.Fatalf("want 1 secondary document, got %d", len(rr.Secondary))
	}
	if got := rr.Secondary[0].Content; got != "file:///tmp/out/sub.txt" {
		t.Errorf("secondary: got %q", got)
	}
	// Without a base output URI the function returns the empty sequence.
	rr, err = ss.TransformEntry("<doc/>", Entry{}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if rr.Output != "0|0" {
		t.Errorf("no base output URI: got %q", rr.Output)
	}
}

// XSLT 3.0 §10.3 stylesheet functions: the result of the body is converted
// to the declared type by the function conversion rules.
func TestXSLTFuncFunctionBodies(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "value-of body joins with separator", want: "1-2-3",
			sheet: `<xsl:function name="f:j"><xsl:value-of select="1 to 3" separator="-"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:j()"/></xsl:template>`},
		{name: "value-of body default separator", want: "a b",
			sheet: `<xsl:function name="f:j"><xsl:value-of select="'a','b'"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:j()"/></xsl:template>`},
		// A text node is atomized to xs:untypedAtomic, then cast to xs:integer.
		{name: "value-of body cast to declared atomic type", want: "42 true",
			sheet: `<xsl:function name="f:n" as="xs:integer"><xsl:value-of select="40 + 2"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:n(), f:n() instance of xs:integer"/></xsl:template>`},
		{name: "value-of body as a text node", want: "true x",
			sheet: `<xsl:function name="f:t" as="text()"><xsl:value-of select="'x'"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:t() instance of text(), f:t()"/></xsl:template>`},
		// Atomizing a map is FOTY0013.
		{name: "value-of a map in a function body", code: "FOTY0013",
			sheet: `<xsl:function name="f:m"><xsl:value-of select="map{1:2}"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:m()"/></xsl:template>`},
		{name: "sequence body keeps items", want: "3",
			sheet: `<xsl:function name="f:s"><xsl:sequence select="1, 'a', 2.5"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:s())"/></xsl:template>`},
		{name: "perform-sort body returns a sequence", want: "31",
			sheet: `<xsl:function name="f:p"><xsl:perform-sort select="(3,1,2)"><xsl:sort select="."/></xsl:perform-sort></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:p()), f:p()[1]" separator=""/></xsl:template>`},
		{name: "evaluate body returns the sequence", want: "3",
			sheet: `<xsl:function name="f:e"><xsl:evaluate xpath="'(1,2,3)'"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:e())"/></xsl:template>`},
		{name: "where-populated body drops empty element", want: "0 1",
			sheet: `<xsl:function name="f:w" as="element()*"><xsl:param name="x"/><xsl:where-populated><e><xsl:value-of select="$x"/></e></xsl:where-populated></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:w('')), count(f:w('a'))"/></xsl:template>`},
		// XTTE0780: the result does not match the declared type.
		{name: "result type mismatch", code: "XTTE0780",
			sheet: `<xsl:function name="f:w" as="element()"><xsl:where-populated><e/></xsl:where-populated></xsl:function>
<xsl:template match="/"><xsl:value-of select="count(f:w())"/></xsl:template>`},
		{name: "constructed content body", want: "e 1",
			sheet: `<xsl:function name="f:c"><xsl:param name="n"/><e><xsl:value-of select="$n"/></e></xsl:function>
<xsl:template match="/"><xsl:value-of select="name(f:c(1)), f:c(1)"/></xsl:template>`},
		{name: "recursion", want: "120",
			sheet: `<xsl:function name="f:fact" as="xs:integer"><xsl:param name="n" as="xs:integer"/><xsl:sequence select="if ($n le 1) then 1 else $n * f:fact($n - 1)"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:fact(5)"/></xsl:template>`},
		// XTTE0790: an argument that does not match the declared parameter type.
		{name: "argument type mismatch", code: "XTTE0790",
			sheet: `<xsl:function name="f:i" as="xs:integer"><xsl:param name="n" as="xs:integer"/><xsl:sequence select="$n"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:i('abc')"/></xsl:template>`},
		{name: "function item and dynamic call", want: "6",
			sheet: `<xsl:function name="f:d"><xsl:param name="n"/><xsl:sequence select="$n * 2"/></xsl:function>
<xsl:template match="/"><xsl:variable name="g" select="f:d#1"/><xsl:value-of select="$g(3)"/></xsl:template>`},
		{name: "fn:-prefixed XSLT function", src: `<r><a/></r>`, want: "true",
			sheet: `<xsl:template match="/r"><xsl:value-of select="fn:generate-id(a) = generate-id(a)"/></xsl:template>`},
	})
}
