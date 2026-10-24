package xslt

import "testing"

// XSLT 3.0 §20.2 keys: fn:key.
func TestXSLTFuncKey(t *testing.T) {
	src := `<r><p id="a" g="x">1</p><p id="b" g="y">2</p><p id="c" g="x">3</p><s><p id="d" g="x">4</p></s></r>`
	specRunAll(t, "", []specCase{
		{name: "single value", src: src, want: "1 3 4", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="key('k','x')"/></xsl:template>`},
		{name: "sequence of values, document order, no duplicates", src: src, want: "1 2 3 4", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="key('k',('y','x','x'))"/></xsl:template>`},
		{name: "node-valued search argument uses string values", src: `<r><q>y</q><q>x</q>` + src[3:], want: "1 2 3 4", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="key('k', //q)"/></xsl:template>`},
		{name: "third argument restricts to subtree", src: src, want: "4", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="key('k','x', //s)"/></xsl:template>`},
		{name: "third argument with several values", src: src, want: "4", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="key('k',('x','y'), //s)"/></xsl:template>`},
		{name: "multiple use values per node", src: `<r><p t="a b">1</p><p t="b">2</p></r>`, want: "1 2", sheet: `
<xsl:key name="k" match="p" use="tokenize(@t)"/>
<xsl:template match="/"><xsl:value-of select="key('k','b')"/></xsl:template>`},
		{name: "typed numeric search value compares by eq", src: `<r><p n="1.0">a</p><p n="2">b</p></r>`, want: "a", sheet: `
<xsl:key name="k" match="p" use="xs:decimal(@n)"/>
<xsl:template match="/"><xsl:value-of select="key('k', 1)"/></xsl:template>`},
		{name: "composite key", src: `<r><p a="1" b="2">x</p><p a="2" b="1">y</p></r>`, want: "y", sheet: `
<xsl:key name="k" match="p" use="@a, @b" composite="yes"/>
<xsl:template match="/"><xsl:value-of select="key('k', ('2','1'))"/></xsl:template>`},
		{name: "prefixed key name resolved by namespace URI", src: src, want: "2", sheet: `
<xsl:key name="f:k" match="p" use="@g"/>
<xsl:template match="/" xmlns:g="urn:f"><xsl:value-of select="key('g:k','y')"/></xsl:template>`},
		{name: "no match is empty", src: src, want: "0", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="count(key('k','zz'))"/></xsl:template>`},
		// §20.2.2: XTDE1260 if the name does not match an xsl:key declaration.
		{name: "undeclared key", src: src, code: "XTDE1260", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="key('nokey','x')"/></xsl:template>`},
		{name: "unbound prefix in key name", src: src, code: "XTDE1260", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:template match="/"><xsl:value-of select="key(string-join(('zz','k'),':'),'x')"/></xsl:template>`},
		// §20.2.2: XTDE1270 when the context item is absent (inside xsl:function).
		{name: "no context item", src: src, code: "XTDE1270", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:function name="f:k"><xsl:sequence select="key('k','x')"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:k()"/></xsl:template>`},
		// ... and when the tree searched is not rooted at a document node.
		{name: "tree not rooted in a document", src: src, code: "XTDE1270", sheet: `
<xsl:key name="k" match="p" use="@g"/>
<xsl:variable name="e" as="element()"><p g="x"/></xsl:variable>
<xsl:template match="/"><xsl:value-of select="count(key('k','x',$e))"/></xsl:template>`},
	})
}

// XSLT 3.0 §20.4 generate-id, §20.4.1 current, §17.1 regex-group.
func TestXSLTFuncContextFunctions(t *testing.T) {
	src := `<r><a>1</a><a>2</a><b>1</b></r>`
	specRunAll(t, "", []specCase{
		{name: "generate-id stable and distinct", src: src, want: "true false true",
			sheet: `<xsl:template match="/r"><xsl:value-of select="generate-id(a[1]) = generate-id(a[1]), generate-id(a[1]) = generate-id(a[2]), generate-id() = generate-id(.)"/></xsl:template>`},
		// F&O 3.1 §14.5.4: an ASCII letter followed by letters/digits.
		{name: "generate-id is an NCName of letters and digits", src: src, want: "true",
			sheet: `<xsl:template match="/r"><xsl:value-of select="matches(generate-id(b), '^[A-Za-z][A-Za-z0-9]*$')"/></xsl:template>`},
		{name: "generate-id of empty sequence", src: src, want: "",
			sheet: `<xsl:template match="/r"><xsl:value-of select="generate-id(())"/></xsl:template>`},
		{name: "current() inside a predicate", src: src, want: "1",
			sheet: `<xsl:template match="/r"><xsl:for-each select="b"><xsl:value-of select="count(../a[. = current()])"/></xsl:for-each></xsl:template>`},
		{name: "current() over atomic items", src: src, want: "b c",
			sheet: `<xsl:template match="/"><xsl:for-each select="2,3"><xsl:value-of select="('a','b','c')[current()]"/><xsl:if test="position() ne last()"><xsl:text> </xsl:text></xsl:if></xsl:for-each></xsl:template>`},
		// §20.4.1: XTDE1360 when the current item is absent (function body).
		{name: "current() with absent focus", src: src, code: "XTDE1360",
			sheet: `<xsl:function name="f:c"><xsl:sequence select="current()"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:c()"/></xsl:template>`},
		// §17.2: regex-group with no current captured substrings is "".
		{name: "regex-group outside analyze-string", src: src, want: "[]",
			sheet: `<xsl:template match="/">[<xsl:value-of select="regex-group(1)"/>]</xsl:template>`},
		{name: "regex-group inside matching-substring", src: src, want: "2026|10",
			sheet: `<xsl:template match="/"><xsl:analyze-string select="'d=2026-10'" regex="(\d+)-(\d+)"><xsl:matching-substring><xsl:value-of select="regex-group(1), regex-group(2)" separator="|"/><xsl:value-of select="regex-group(9)"/></xsl:matching-substring></xsl:analyze-string></xsl:template>`},
	})
}

// XSLT 3.0 §20.5 fn:copy-of / fn:snapshot (F&O 3.1 §16.?? via XSLT).
func TestXSLTFuncCopyOfSnapshot(t *testing.T) {
	src := `<r><a x="1"><b>t</b></a></r>`
	specRunAll(t, "", []specCase{
		{name: "copy-of makes new identity, deep-equal", src: src, want: "false true",
			sheet: `<xsl:template match="/r"><xsl:value-of select="copy-of(a) is a, deep-equal(copy-of(a), a)"/></xsl:template>`},
		{name: "copy-of of a descendant has no parent", src: src, want: "0",
			sheet: `<xsl:template match="/r"><xsl:value-of select="count(copy-of(a/b)/..)"/></xsl:template>`},
		{name: "copy-of with no argument copies the context item", src: src, want: "t",
			sheet: `<xsl:template match="/r"><xsl:value-of select="a/b/copy-of()"/></xsl:template>`},
		{name: "copy-of passes atomic values through", src: src, want: "1 x",
			sheet: `<xsl:template match="/r"><xsl:value-of select="copy-of((1,'x'))"/></xsl:template>`},
		// fn:snapshot keeps the ancestors (shallow copies) of the node.
		{name: "snapshot keeps ancestors", src: src, want: "a r 1",
			sheet: `<xsl:template match="/r"><xsl:value-of select="name(snapshot(a/b)/..), name(snapshot(a/b)/../..), snapshot(a/b)/../@x"/></xsl:template>`},
		{name: "snapshot without argument", src: src, want: "r",
			sheet: `<xsl:template match="/r"><xsl:for-each select="a/b"><xsl:value-of select="name(snapshot()/../..)"/></xsl:for-each></xsl:template>`},
		{name: "copy-of with absent focus", src: src, code: "XPDY0002",
			sheet: `<xsl:function name="f:c"><xsl:sequence select="copy-of()"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:c()"/></xsl:template>`},
	})
}

// XSLT 3.0 §20.3.? system-property / available-system-properties.
func TestXSLTFuncSystemProperty(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "version and xpath-version", want: "3.0 3.1",
			sheet: `<xsl:template match="/"><xsl:value-of select="system-property('xsl:version'), system-property('xsl:xpath-version')"/></xsl:template>`},
		{name: "vendor properties are non-empty", want: "true true true true",
			sheet: `<xsl:template match="/"><xsl:value-of select="for $p in ('vendor','vendor-url','product-name','product-version') return system-property('xsl:' || $p) ne ''"/></xsl:template>`},
		// §20.3.? : yes/no valued properties.
		{name: "yes/no properties", want: "true",
			sheet: `<xsl:template match="/"><xsl:value-of select="every $p in ('is-schema-aware','supports-serialization','supports-backwards-compatibility','supports-namespace-axis','supports-streaming','supports-dynamic-evaluation','supports-higher-order-functions') satisfies system-property('xsl:' || $p) = ('yes','no')"/></xsl:template>`},
		{name: "dynamic evaluation and HOF are supported", want: "yes yes yes",
			sheet: `<xsl:template match="/"><xsl:value-of select="system-property('xsl:supports-dynamic-evaluation'), system-property('xsl:supports-higher-order-functions'), system-property('xsl:supports-serialization')"/></xsl:template>`},
		{name: "xsd-version", want: "true",
			sheet: `<xsl:template match="/"><xsl:value-of select="system-property('xsl:xsd-version') = ('1.0','1.1')"/></xsl:template>`},
		{name: "Q{} name form", want: "3.0",
			sheet: `<xsl:template match="/"><xsl:value-of select="system-property('Q{http://www.w3.org/1999/XSL/Transform}version')"/></xsl:template>`},
		// An unknown property, or one outside the XSLT namespace, is "".
		{name: "unknown properties", want: "[][]",
			sheet: `<xsl:template match="/">[<xsl:value-of select="system-property('xsl:no-such')"/>][<xsl:value-of select="system-property('f:version')"/>]</xsl:template>`},
		// XTDE1390: not a valid QName / unbound prefix.
		{name: "invalid QName", code: "XTDE1390",
			sheet: `<xsl:template match="/"><xsl:value-of select="system-property(concat('1', 'x'))"/></xsl:template>`},
		{name: "unbound prefix", code: "XTDE1390",
			sheet: `<xsl:template match="/"><xsl:value-of select="system-property(concat('zz', ':version'))"/></xsl:template>`},
		{name: "available-system-properties lists version and every value is non-empty", want: "true true",
			sheet: `<xsl:template match="/"><xsl:variable name="p" select="available-system-properties()"/><xsl:value-of select="QName('http://www.w3.org/1999/XSL/Transform','version') = $p, every $q in $p satisfies system-property('Q{' || namespace-uri-from-QName($q) || '}' || local-name-from-QName($q)) ne ''"/></xsl:template>`},
	})
}

// XSLT 3.0 §20.3.1 element-available, §20.3.2 function-available, §20.3.3 type-available.
func TestXSLTFuncAvailability(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "element-available", want: "true true true false false",
			sheet: `<xsl:template match="/"><xsl:value-of select="element-available('xsl:value-of'), element-available('xsl:for-each-group'), element-available('Q{http://www.w3.org/1999/XSL/Transform}iterate'), element-available('xsl:no-such-instruction'), element-available('f:value-of')"/></xsl:template>`},
		{name: "element-available invalid QName", code: "XTDE1440",
			sheet: `<xsl:template match="/"><xsl:value-of select="element-available(concat('a b',''))"/></xsl:template>`},
		{name: "function-available standard", want: "true true true false true false",
			sheet: `<xsl:template match="/"><xsl:value-of select="function-available('concat'), function-available('fn:substring', 3), function-available('Q{http://www.w3.org/2005/xpath-functions}abs', 1), function-available('abs', 2), function-available('math:pi', 0), function-available('no-such-function')"/></xsl:template>`},
		{name: "function-available XSLT functions", want: "true true false",
			sheet: `<xsl:template match="/"><xsl:value-of select="function-available('key', 2), function-available('document'), function-available('key', 5)"/></xsl:template>`},
		{name: "function-available constructor functions", want: "true false",
			sheet: `<xsl:template match="/"><xsl:value-of select="function-available('xs:integer', 1), function-available('xs:anyAtomicType', 1)"/></xsl:template>`},
		{name: "function-available user function by arity", want: "true true false",
			sheet: `<xsl:function name="f:u"><xsl:param name="a"/><xsl:sequence select="$a"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="function-available('f:u'), function-available('f:u', 1), function-available('f:u', 2)"/></xsl:template>`},
		{name: "function-available invalid QName", code: "XTDE1400",
			sheet: `<xsl:template match="/"><xsl:value-of select="function-available(concat(':x',''))"/></xsl:template>`},
		{name: "type-available", want: "true true true false",
			sheet: `<xsl:template match="/"><xsl:value-of select="type-available('xs:integer'), type-available('xs:untypedAtomic'), type-available('xs:anyType'), type-available('xs:no-such-type')"/></xsl:template>`},
		{name: "type-available invalid QName", code: "XTDE1450",
			sheet: `<xsl:template match="/"><xsl:value-of select="type-available(concat('1',''))"/></xsl:template>`},
	})
}

// XSLT 3.0 §20.? unparsed-entity-uri / unparsed-entity-public-id.
func TestXSLTFuncUnparsedEntity(t *testing.T) {
	src := `<?xml version="1.0"?>
<!DOCTYPE r [
<!NOTATION gif PUBLIC "-//GIF//EN" "image/gif">
<!ENTITY pic PUBLIC "-//PIC//EN" "http://example.com/pic.gif" NDATA gif>
<!ENTITY sys SYSTEM "http://example.com/sys.gif" NDATA gif>
]>
<r/>`
	specRunAll(t, "", []specCase{
		{name: "uri and public id", src: src, want: "http://example.com/pic.gif|-//PIC//EN|http://example.com/sys.gif|",
			sheet: `<xsl:template match="/"><xsl:value-of select="unparsed-entity-uri('pic'), unparsed-entity-public-id('pic'), unparsed-entity-uri('sys'), unparsed-entity-public-id('sys')" separator="|"/></xsl:template>`},
		{name: "uri is xs:anyURI", src: src, want: "true",
			sheet: `<xsl:template match="/"><xsl:value-of select="unparsed-entity-uri('pic') instance of xs:anyURI"/></xsl:template>`},
		{name: "unknown entity gives zero-length strings", src: src, want: "[][]",
			sheet: `<xsl:template match="/">[<xsl:value-of select="unparsed-entity-uri('none')"/>][<xsl:value-of select="unparsed-entity-public-id('none')"/>]</xsl:template>`},
		{name: "second argument selects the document", src: src, want: "http://example.com/pic.gif",
			sheet: `<xsl:template match="/"><xsl:value-of select="unparsed-entity-uri('pic', /)"/></xsl:template>`},
		// XTDE1370: no context node.
		{name: "absent focus", src: src, code: "XTDE1370",
			sheet: `<xsl:function name="f:u"><xsl:sequence select="unparsed-entity-uri('pic')"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:u()"/></xsl:template>`},
		{name: "tree not rooted in a document", src: src, code: "XTDE1370",
			sheet: `<xsl:variable name="e" as="element()"><e/></xsl:variable>
<xsl:template match="/"><xsl:value-of select="unparsed-entity-uri('pic', $e)"/></xsl:template>`},
	})
}
