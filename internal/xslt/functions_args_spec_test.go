package xslt

import "testing"

// The function conversion rules (XPath 3.1 §3.1.5.2) reach a call from a
// stylesheet: the F&O built-ins and the XSLT-defined functions of the
// standard function namespace (XSLT 3.0 §§18, 20) alike (issue #9).
func TestFunctionArgumentConversion(t *testing.T) {
	val := func(sel string) string {
		return `<xsl:key name="k" match="i" use="."/>
<xsl:template match="/"><xsl:value-of select="` + sel + `"/></xsl:template>`
	}
	const src = `<doc><i>a</i><i>b</i></doc>`
	specRunAll(t, "", []specCase{
		// F&O built-ins.
		{name: "upper-case of an integer", code: "XPTY0004", src: src, sheet: val(`upper-case(1)`)},
		{name: "empty collation", code: "XPTY0004", src: src, sheet: val(`contains-token('a b', 'a', ())`)},
		{name: "two nodes for xs:string?", code: "XPTY0004", src: src, sheet: val(`string-length(doc/i)`)},
		{name: "untyped node cast to xs:double", want: "bc", src: src, sheet: val(`substring('abc', count(doc/i))`)},
		{name: "untyped node cast, bad lexical", code: "FORG0001", src: src, sheet: val(`abs(doc/i[1])`)},
		{name: "node for xs:string", want: "A", src: src, sheet: val(`upper-case(doc/i[1])`)},
		{name: "empty for xs:string?", want: "0", src: src, sheet: val(`string-length(doc/none)`)},
		// XSLT-defined functions.
		{name: "system-property of a number", code: "XPTY0004", src: src, sheet: val(`system-property(1)`)},
		{name: "system-property of nothing", code: "XPTY0004", src: src, sheet: val(`system-property(())`)},
		{name: "system-property", want: "3.0", src: src, sheet: val(`system-property('xsl:version')`)},
		{name: "function-available of a number", code: "XPTY0004", src: src, sheet: val(`function-available(1)`)},
		{name: "function-available arity cast from untyped", want: "true", src: src,
			sheet: val(`function-available('upper-case', xs:untypedAtomic('1'))`)},
		{name: "function-available arity string", code: "XPTY0004", src: src, sheet: val(`function-available('upper-case', '1')`)},
		{name: "element-available of nothing", code: "XPTY0004", src: src, sheet: val(`element-available(())`)},
		{name: "element-available", want: "true", src: src, sheet: val(`element-available('xsl:value-of')`)},
		{name: "type-available of a number", code: "XPTY0004", src: src, sheet: val(`type-available(1)`)},
		{name: "key name of a number", code: "XPTY0004", src: src, sheet: val(`key(1, 'a')`)},
		{name: "key top node of a string", code: "XPTY0004", src: src, sheet: val(`key('k', 'a', 'x')`)},
		{name: "key value of a map", code: "FOTY0013", src: src, sheet: val(`key('k', map{})`)},
		{name: "key", want: "a", src: src, sheet: val(`key('k', 'a')`)},
		{name: "key with a top node", want: "b", src: src, sheet: val(`key('k', 'b', /)`)},
		{name: "key values from nodes", want: "ab", src: src, sheet: val(`string-join(key('k', doc/i), '')`)},
		{name: "regex-group of a string", code: "XPTY0004", src: src, sheet: val(`regex-group('1')`)},
		{name: "unparsed-entity-uri of a number", code: "XPTY0004", src: src, sheet: val(`unparsed-entity-uri(1)`)},
		{name: "unparsed-entity-uri doc argument", code: "XPTY0004", src: src, sheet: val(`unparsed-entity-uri('x', 'y')`)},
		{name: "accumulator-before of a number", code: "XPTY0004", src: src, sheet: val(`accumulator-before(1)`)},
		{name: "current-merge-group of a number", code: "XPTY0004", src: src, sheet: val(`current-merge-group(1)`)},
		{name: "stream-available of two uris", code: "XPTY0004", src: src, sheet: val(`stream-available(('a', 'b'))`)},
		{name: "document base node of a string", code: "XPTY0004", src: src, sheet: val(`document('x.xml', 'y')`)},
		{name: "generate-id of an atomic", code: "XPTY0004", src: src, sheet: val(`generate-id(1)`)},
		{name: "generate-id", want: "true", src: src, sheet: val(`string-length(generate-id(doc/i[1])) gt 0`)},
		{name: "copy-of any item", want: "a", src: src, sheet: val(`copy-of('a')`)},
		{name: "current-group takes no argument", code: "XPST0017", src: src, sheet: val(`current-group(1)`)},
		// A named reference and a dynamic call are checked too.
		{name: "system-property#1 dynamic call", code: "XPTY0004", src: src, sheet: val(`system-property#1(1)`)},
		{name: "upper-case#1 dynamic call", code: "XPTY0004", src: src, sheet: val(`let $f := upper-case#1 return $f(1)`)},
	})
}

// Under XSLT 1.0 backwards-compatible processing the strict argument check
// is not made.
func TestFunctionArgumentConversionBackwardsCompatible(t *testing.T) {
	sheet := `<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output method="text"/>
<xsl:template match="/"><xsl:value-of select="substring('abc', '2')"/>|<xsl:value-of select="contains(1, '1')"/></xsl:template>
</xsl:stylesheet>`
	ss, err := svDCompile(sheet, Features{BackwardsCompatible: true})
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := ss.Transform("<a/>", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "bc|true" {
		t.Errorf("got %q, want %q", out, "bc|true")
	}
}
