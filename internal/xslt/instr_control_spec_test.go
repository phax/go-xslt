package xslt

import "testing"

const ctlCities = `<cities>
<city name="Milano" country="Italia" pop="5"/>
<city name="Paris" country="France" pop="7"/>
<city name="München" country="Deutschland" pop="4"/>
<city name="Lyon" country="France" pop="2"/>
<city name="Venezia" country="Italia" pop="1"/>
</cities>`

// XSLT 3.0 §14 grouping (the REC's own city/country example and friends).
func TestGroupSpec(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "group-by with current-group and current-grouping-key", src: ctlCities, want: "Italia:Milano,Venezia=6;France:Paris,Lyon=9;Deutschland:München=4;", sheet: `
<xsl:template match="/"><xsl:for-each-group select="cities/city" group-by="@country"><xsl:value-of select="current-grouping-key()"/>:<xsl:value-of select="current-group()/@name" separator=","/>=<xsl:value-of select="sum(current-group()/@pop)"/>;</xsl:for-each-group></xsl:template>`},
		{name: "group-by with sort by key", src: ctlCities, want: "Deutschland France Italia ", sheet: `
<xsl:template match="/"><xsl:for-each-group select="cities/city" group-by="@country"><xsl:sort select="current-grouping-key()"/><xsl:value-of select="current-grouping-key()"/><xsl:text> </xsl:text></xsl:for-each-group></xsl:template>`},
		// An item with several keys joins several groups.
		{name: "group-by with several keys per item", src: `<r><i k="a b">1</i><i k="b">2</i></r>`, want: "a:1;b:12;", sheet: `
<xsl:template match="/"><xsl:for-each-group select="r/i" group-by="tokenize(@k)"><xsl:value-of select="current-grouping-key()"/>:<xsl:value-of select="current-group()" separator=""/>;</xsl:for-each-group></xsl:template>`},
		{name: "composite group-by", src: `<r><i a="1" b="x"/><i a="1" b="y"/><i a="1" b="x"/></r>`, want: "1 x:2;1 y:1;", sheet: `
<xsl:template match="/"><xsl:for-each-group select="r/i" group-by="@a, @b" composite="yes"><xsl:value-of select="current-grouping-key()"/>:<xsl:value-of select="count(current-group())"/>;</xsl:for-each-group></xsl:template>`},
		{name: "group-by with a collation", src: `<r><i k="A"/><i k="a"/><i k="B"/></r>`, want: "2 1 ", sheet: `
<xsl:template match="/"><xsl:for-each-group select="r/i" group-by="@k" collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"><xsl:value-of select="count(current-group())"/><xsl:text> </xsl:text></xsl:for-each-group></xsl:template>`},
		{name: "group-adjacent", want: "1,1|2|1,1|", sheet: `
<xsl:template match="/"><xsl:for-each-group select="1, 1, 2, 1, 1" group-adjacent="."><xsl:value-of select="current-group()" separator=","/>|</xsl:for-each-group></xsl:template>`},
		{name: "composite group-adjacent", src: `<r><i a="1" b="x"/><i a="1" b="x"/><i a="2" b="x"/></r>`, want: "2 1 ", sheet: `
<xsl:template match="/"><xsl:for-each-group select="r/i" group-adjacent="@a, @b" composite="yes"><xsl:value-of select="count(current-group())"/><xsl:text> </xsl:text></xsl:for-each-group></xsl:template>`},
		// §14.? the REC's flat-to-hierarchy example.
		{name: "group-starting-with", src: `<body><h2>A</h2><p>1</p><p>2</p><h2>B</h2><p>3</p></body>`, want: "<s><h>A</h><p>1</p><p>2</p></s><s><h>B</h><p>3</p></s>", sheet: specXML(`
<xsl:template match="/"><xsl:for-each-group select="body/*" group-starting-with="h2"><s><h><xsl:value-of select="."/></h><xsl:copy-of select="current-group()[position() gt 1]"/></s></xsl:for-each-group></xsl:template>`)},
		{name: "group-ending-with", src: `<r><p c="y">1</p><p>2</p><p c="n">3</p><p>4</p></r>`, want: "1|23|4|", sheet: `
<xsl:template match="/"><xsl:for-each-group select="r/p" group-ending-with="p[@c]"><xsl:value-of select="current-group()" separator=""/>|</xsl:for-each-group></xsl:template>`},
		// XTDE1071: no current grouping key with group-starting-with.
		{name: "current-grouping-key with group-starting-with", src: `<r><a/></r>`, code: "XTDE1071", sheet: `
<xsl:template match="/"><xsl:for-each-group select="r/*" group-starting-with="a"><xsl:value-of select="current-grouping-key()"/></xsl:for-each-group></xsl:template>`},
		{name: "current-grouping-key inside a function", code: "XTDE1071", sheet: `
<xsl:function name="f:k"><xsl:sequence select="current-grouping-key()"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:k()"/></xsl:template>`},
		{name: "current-group inside a function", code: "XTDE1061", sheet: `
<xsl:function name="f:g"><xsl:sequence select="current-group()"/></xsl:function>
<xsl:template match="/"><xsl:value-of select="f:g()"/></xsl:template>`},
		// XTTE1100: a group-adjacent key of more than one item.
		{name: "group-adjacent key sequence", code: "XTTE1100", sheet: `
<xsl:template match="/"><xsl:for-each-group select="1, 2" group-adjacent="(., .)"><xsl:value-of select="."/></xsl:for-each-group></xsl:template>`},
		// XTSE1090: @collation without group-by / group-adjacent.
		{name: "collation with group-starting-with", code: "XTSE1090", sheet: `
<xsl:template match="/"><xsl:for-each-group select="*" group-starting-with="a" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"><x/></xsl:for-each-group></xsl:template>`},
	})
}

// XSLT 3.0 §7.? xsl:iterate.
func TestIterateSpec(t *testing.T) {
	specRunAll(t, "", []specCase{
		// The REC's running-balance example (shortened).
		{name: "running total with on-completion", want: "1 3 6 total=6", sheet: `
<xsl:template match="/"><xsl:iterate select="1 to 3"><xsl:param name="sum" select="0"/><xsl:on-completion>total=<xsl:value-of select="$sum"/></xsl:on-completion><xsl:value-of select="$sum + ."/><xsl:text> </xsl:text><xsl:next-iteration><xsl:with-param name="sum" select="$sum + ."/></xsl:next-iteration></xsl:iterate></xsl:template>`},
		{name: "break with select", want: "1 2 stop", sheet: `
<xsl:template match="/"><xsl:iterate select="1 to 10"><xsl:choose><xsl:when test=". eq 3"><xsl:break select="'stop'"/></xsl:when><xsl:otherwise><xsl:value-of select="."/><xsl:text> </xsl:text></xsl:otherwise></xsl:choose></xsl:iterate></xsl:template>`},
		{name: "on-completion with an empty sequence", want: "none", sheet: `
<xsl:template match="/"><xsl:iterate select="()"><xsl:on-completion select="'none'"/><xsl:value-of select="."/></xsl:iterate></xsl:template>`},
		{name: "next-iteration in a choose", want: "13", sheet: `
<xsl:template match="/"><xsl:iterate select="1 to 4"><xsl:param name="odd" select="true()"/><xsl:choose><xsl:when test="$odd"><xsl:value-of select="."/><xsl:next-iteration><xsl:with-param name="odd" select="false()"/></xsl:next-iteration></xsl:when><xsl:otherwise><xsl:next-iteration><xsl:with-param name="odd" select="true()"/></xsl:next-iteration></xsl:otherwise></xsl:choose></xsl:iterate></xsl:template>`},
		{name: "typed iterate parameter", want: "6", sheet: `
<xsl:template match="/"><xsl:iterate select="1 to 3"><xsl:param name="p" as="xs:integer" select="0"/><xsl:on-completion><xsl:value-of select="$p"/></xsl:on-completion><xsl:next-iteration><xsl:with-param name="p" select="$p + ."/></xsl:next-iteration></xsl:iterate></xsl:template>`},
		// XTSE3120: next-iteration / break not in tail position.
		{name: "next-iteration not in tail position", code: "XTSE3120", sheet: `
<xsl:template match="/"><xsl:iterate select="1 to 2"><xsl:next-iteration/><xsl:value-of select="."/></xsl:iterate></xsl:template>`},
		{name: "break not in tail position", code: "XTSE3120", sheet: `
<xsl:template match="/"><xsl:iterate select="1 to 2"><xsl:break/><xsl:value-of select="."/></xsl:iterate></xsl:template>`},
		// XTSE3130: next-iteration names a parameter the iterate lacks.
		{name: "undeclared iteration parameter", code: "XTSE3130", sheet: `
<xsl:template match="/"><xsl:iterate select="1 to 2"><xsl:param name="a" select="0"/><xsl:next-iteration><xsl:with-param name="b" select="1"/></xsl:next-iteration></xsl:iterate></xsl:template>`},
	})
}

// XSLT 3.0 §8.3 xsl:try / xsl:catch.
func TestTryCatchSpec(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "error variables", want: "FOAR0001|true", sheet: `
<xsl:template match="/"><xsl:try><xsl:value-of select="1 idiv 0"/><xsl:catch><xsl:value-of select="local-name-from-QName($err:code), $err:description ne ''" separator="|" xmlns:err="http://www.w3.org/2005/xqt-errors"/></xsl:catch></xsl:try></xsl:template>`},
		{name: "user error with value", want: "e|42", sheet: `
<xsl:template match="/"><xsl:try><xsl:sequence select="error(QName('urn:x', 'e'), 'msg', 42)"/><xsl:catch errors="Q{urn:x}e" xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="local-name-from-QName($err:code), $err:value" separator="|"/></xsl:catch></xsl:try></xsl:template>`},
		{name: "namespace wildcard", want: "caught", sheet: `
<xsl:template match="/"><xsl:try><xsl:sequence select="error(QName('urn:x', 'e'))"/><xsl:catch errors="Q{urn:x}*">caught</xsl:catch></xsl:try></xsl:template>`},
		{name: "local-name wildcard", want: "caught", sheet: `
<xsl:template match="/"><xsl:try><xsl:sequence select="error(QName('urn:y', 'e'))"/><xsl:catch errors="*:e">caught</xsl:catch></xsl:try></xsl:template>`},
		{name: "first matching catch wins", want: "second", sheet: `
<xsl:template match="/"><xsl:try><xsl:sequence select="1 idiv 0"/><xsl:catch errors="err:XPTY0004" xmlns:err="http://www.w3.org/2005/xqt-errors">first</xsl:catch><xsl:catch errors="err:FOAR0001 err:FOAR0002" xmlns:err="http://www.w3.org/2005/xqt-errors">second</xsl:catch><xsl:catch>third</xsl:catch></xsl:try></xsl:template>`},
		{name: "no matching catch propagates", code: "FOAR0001", sheet: `
<xsl:template match="/"><xsl:try><xsl:sequence select="1 idiv 0"/><xsl:catch errors="err:XPTY0004" xmlns:err="http://www.w3.org/2005/xqt-errors">no</xsl:catch></xsl:try></xsl:template>`},
		{name: "nested try rethrows", want: "outer", sheet: `
<xsl:template match="/"><xsl:try><xsl:try><xsl:sequence select="1 idiv 0"/><xsl:catch><xsl:sequence select="error(QName('urn:x', 'again'))"/></xsl:catch></xsl:try><xsl:catch errors="Q{urn:x}again">outer</xsl:catch></xsl:try></xsl:template>`},
		{name: "catch with select", want: "fallback", sheet: `
<xsl:template match="/"><xsl:try select="1 idiv 0"><xsl:catch select="'fallback'"/></xsl:try></xsl:template>`},
		{name: "partial output is rolled back", want: "<r>caught</r>", sheet: specXML(`
<xsl:template match="/"><r><xsl:try><partial/><xsl:sequence select="1 idiv 0"/><xsl:catch>caught</xsl:catch></xsl:try></r></xsl:template>`)},
		{name: "dynamic error code in the catch", want: "XTDE0820", sheet: `
<xsl:template match="/"><xsl:try><xsl:element name="{'1x'}"/><xsl:catch xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="local-name-from-QName($err:code)"/></xsl:catch></xsl:try></xsl:template>`},
	})
}

// XSLT 3.0 §21.? xsl:map / xsl:map-entry.
func TestMapSpec(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "entries and nesting", want: "2 1 x y", sheet: `
<xsl:template match="/"><xsl:variable name="m" as="map(*)"><xsl:map><xsl:map-entry key="'a'" select="1"/><xsl:map-entry key="'b'"><xsl:map><xsl:map-entry key="'c'" select="'x', 'y'"/></xsl:map></xsl:map-entry></xsl:map></xsl:variable><xsl:value-of select="map:size($m), $m?a, $m?b?c"/></xsl:template>`},
		{name: "map built from a for-each", src: `<r><i k="a" v="1"/><i k="b" v="2"/></r>`, want: "a=1 b=2", sheet: `
<xsl:template match="/"><xsl:variable name="m" as="map(*)"><xsl:map><xsl:for-each select="r/i"><xsl:map-entry key="string(@k)" select="string(@v)"/></xsl:for-each></xsl:map></xsl:variable><xsl:value-of select="for $k in sort(map:keys($m)) return $k || '=' || $m($k)"/></xsl:template>`},
		{name: "maps merged from xsl:sequence", want: "3", sheet: `
<xsl:template match="/"><xsl:variable name="m" as="map(*)"><xsl:map><xsl:sequence select="map{1: 'a'}, map{2: 'b'}"/><xsl:map-entry key="3" select="'c'"/></xsl:map></xsl:variable><xsl:value-of select="map:size($m)"/></xsl:template>`},
		{name: "duplicate keys from xsl:sequence", code: "XTDE3365", sheet: `
<xsl:template match="/"><xsl:variable name="m" as="map(*)"><xsl:map><xsl:sequence select="map{1: 'a'}"/><xsl:map-entry key="1" select="'b'"/></xsl:map></xsl:variable><xsl:value-of select="map:size($m)"/></xsl:template>`},
		// XTSE3280: map-entry with both select and content.
		{name: "map-entry select and content", code: "XTSE3280", sheet: `
<xsl:template match="/"><xsl:map><xsl:map-entry key="1" select="2">3</xsl:map-entry></xsl:map></xsl:template>`},
	})
}

// XSLT 3.0 §23.1 xsl:message and §23.2 xsl:assert.
func TestMessageSpec(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "terminate yes", code: "XTMM9000", sheet: `
<xsl:template match="/"><xsl:message terminate="yes">stop</xsl:message></xsl:template>`},
		{name: "terminate AVT", code: "XTMM9000", sheet: `
<xsl:template match="/"><xsl:message terminate="{'yes'}">stop</xsl:message></xsl:template>`},
		{name: "terminate invalid", code: "XTDE0030", sheet: `
<xsl:template match="/"><xsl:message terminate="{'maybe'}">stop</xsl:message></xsl:template>`},
		{name: "assert with error-code", code: "BAD", sheet: `
<xsl:template match="/"><xsl:assert test="1 = 2" error-code="f:BAD" select="'no'"/></xsl:template>`},
		{name: "assert failing", code: "XTMM9001", sheet: `
<xsl:template match="/"><xsl:assert test="false()"/></xsl:template>`},
		{name: "assert on an empty sequence fails", code: "XTMM9001", sheet: `
<xsl:template match="/"><xsl:assert test="()">empty</xsl:assert></xsl:template>`},
	})
	ss, err := Compile(specSheet(`<xsl:template match="/"><xsl:message select="'a', 'b'"/><xsl:message>body <xsl:value-of select="1 + 1"/></xsl:message>done</xsl:template>`))
	if err != nil {
		t.Fatal(err)
	}
	rr, err := ss.TransformFull("<doc/>", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if rr.Output != "done" || len(rr.Messages) != 2 || rr.Messages[0] != "a b" || rr.Messages[1] != "body 2" {
		t.Errorf("output %q, messages %q", rr.Output, rr.Messages)
	}
}

// XSLT 3.0 §5.6.1 attribute value templates.
func TestAVTSpec(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "doubled braces", want: `<e a="{x}"/>`, sheet: specXML(`<xsl:template match="/"><e a="{{x}}"/></xsl:template>`)},
		{name: "brace inside a string literal", want: `<e a="}{"/>`, sheet: specXML(`<xsl:template match="/"><e a="{'}' || '{'}"/></xsl:template>`)},
		{name: "XPath comment inside", want: `<e a="2"/>`, sheet: specXML(`<xsl:template match="/"><e a="{1 (: one } :) + 1}"/></xsl:template>`)},
		{name: "sequence joined by spaces", want: `<e a="1 2 3"/>`, sheet: specXML(`<xsl:template match="/"><e a="{1 to 3}"/></xsl:template>`)},
		{name: "several parts", want: `<e a="x1y2z"/>`, sheet: specXML(`<xsl:template match="/"><e a="x{1}y{2}z"/></xsl:template>`)},
		{name: "nested map constructor", want: `<e a="v"/>`, sheet: specXML(`<xsl:template match="/"><e a="{map{'k': 'v'}?k}"/></xsl:template>`)},
		// XTSE0370: an unescaped "}" outside an expression.
		{name: "lone closing brace", code: "XTSE0370", sheet: specXML(`<xsl:template match="/"><e a="x}y"/></xsl:template>`)},
	})
}

// XSLT 3.0 §10.2 attribute sets and §10.1.? call-template parameter checks.
func TestAttributeSetSpec(t *testing.T) {
	specRunAll(t, "", []specCase{
		{name: "attribute sets using others", want: `a=1 b=2 c=3`, sheet: (`
<xsl:attribute-set name="s1" use-attribute-sets="s2"><xsl:attribute name="a">1</xsl:attribute></xsl:attribute-set>
<xsl:attribute-set name="s2"><xsl:attribute name="b">2</xsl:attribute></xsl:attribute-set>
<xsl:attribute-set name="s3"><xsl:attribute name="c">3</xsl:attribute></xsl:attribute-set>
<xsl:template match="/"><xsl:variable name="e"><e xsl:use-attribute-sets="s1 s3"/></xsl:variable><xsl:value-of select="sort($e/e/@*, (), name#1) ! (name(.) || '=' || .)"/></xsl:template>`)},
		// Later attributes override earlier ones; the element's own win.
		{name: "override order", want: `<e a="own"/>`, sheet: specXML(`
<xsl:attribute-set name="s"><xsl:attribute name="a">set</xsl:attribute></xsl:attribute-set>
<xsl:template match="/"><e xsl:use-attribute-sets="s" a="own"/></xsl:template>`)},
		{name: "attribute set merged across declarations", want: `a=1 b=2`, sheet: (`
<xsl:attribute-set name="s"><xsl:attribute name="a">1</xsl:attribute></xsl:attribute-set>
<xsl:attribute-set name="s"><xsl:attribute name="b">2</xsl:attribute></xsl:attribute-set>
<xsl:template match="/"><xsl:variable name="e"><e xsl:use-attribute-sets="s"/></xsl:variable><xsl:value-of select="sort($e/e/@*, (), name#1) ! (name(.) || '=' || .)"/></xsl:template>`)},
		{name: "attribute set sees the context node", src: `<r id="7"/>`, want: `<e id="7"/>`, sheet: specXML(`
<xsl:attribute-set name="s"><xsl:attribute name="id" select="/r/@id"/></xsl:attribute-set>
<xsl:template match="/"><xsl:element name="e" use-attribute-sets="s"/></xsl:template>`)},
		// XTSE0710: an undeclared attribute set.
		{name: "undeclared attribute set", code: "XTSE0710", sheet: specXML(`
<xsl:template match="/"><e xsl:use-attribute-sets="nope"/></xsl:template>`)},
		{name: "undeclared attribute set in a declaration", code: "XTSE0710", sheet: specXML(`
<xsl:attribute-set name="s" use-attribute-sets="nope"/>
<xsl:template match="/"><e/></xsl:template>`)},
		// XTSE0720: an attribute set that uses itself.
		{name: "circular attribute sets", code: "XTSE0720", sheet: specXML(`
<xsl:attribute-set name="a" use-attribute-sets="b"/><xsl:attribute-set name="b" use-attribute-sets="a"/>
<xsl:template match="/"><e xsl:use-attribute-sets="a"/></xsl:template>`)},
		// XTSE0680: a non-tunnel with-param the called template does not declare.
		{name: "call-template with an undeclared parameter", code: "XTSE0680", sheet: `
<xsl:template name="t"><xsl:param name="a"/></xsl:template>
<xsl:template match="/"><xsl:call-template name="t"><xsl:with-param name="b" select="1"/></xsl:call-template></xsl:template>`},
		{name: "tunnel parameters pass through", want: "T", sheet: `
<xsl:template match="/"><xsl:call-template name="a"><xsl:with-param name="t" select="'T'" tunnel="yes"/></xsl:call-template></xsl:template>
<xsl:template name="a"><xsl:call-template name="b"/></xsl:template>
<xsl:template name="b"><xsl:param name="t" tunnel="yes"/><xsl:value-of select="$t"/></xsl:template>`},
		// XTTE0590: a parameter value that does not match @as.
		{name: "template parameter type", code: "XTTE0590", sheet: `
<xsl:template name="t"><xsl:param name="a" as="xs:integer"/><xsl:value-of select="$a"/></xsl:template>
<xsl:template match="/"><xsl:call-template name="t"><xsl:with-param name="a" select="'x'"/></xsl:call-template></xsl:template>`},
		// XTTE0505: a template result that does not match @as.
		{name: "template result type", code: "XTTE0505", sheet: `
<xsl:template name="t" as="xs:integer"><xsl:sequence select="'x'"/></xsl:template>
<xsl:template match="/"><xsl:call-template name="t"/></xsl:template>`},
	})
}

// Former BUG: entries, now checked against the REC.
func TestInstrControlSpecFixed(t *testing.T) {
	specRunAll(t, "", []specCase{
		// §14.2.1: an invocation construct other than a function call (and
		// outside a declared-streamable construct) leaves the current group
		// unchanged, so a template applied from xsl:for-each-group sees it.
		{name: "current-group in an applied template", src: `<r/>`, want: "1", sheet: `
<xsl:template match="/"><xsl:for-each-group select="r" group-by="name()"><xsl:apply-templates select="."/></xsl:for-each-group></xsl:template>
<xsl:template match="r"><xsl:value-of select="count(current-group())"/></xsl:template>`},
		// §23.1: terminate="yes" raises the error code named by @error-code.
		{name: "terminate with error-code", code: "MYERR", sheet: `
<xsl:template match="/"><xsl:message terminate="yes" error-code="f:MYERR">stop</xsl:message></xsl:template>`},
		{name: "terminate can be caught", want: "Q{urn:f}MYERR", sheet: `
<xsl:template match="/"><xsl:try><xsl:message terminate="yes" error-code="f:MYERR">stop</xsl:message><xsl:catch xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="'Q{' || namespace-uri-from-QName($err:code) || '}' || local-name-from-QName($err:code)"/></xsl:catch></xsl:try></xsl:template>`},
		{name: "error-code as an AVT in Q{} form", want: "Q{urn:g}E2", sheet: `
<xsl:template match="/"><xsl:try><xsl:message terminate="yes" error-code="{'Q{urn:g}E' || 2}">stop</xsl:message><xsl:catch xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="'Q{' || namespace-uri-from-QName($err:code) || '}' || local-name-from-QName($err:code)"/></xsl:catch></xsl:try></xsl:template>`},
		// §23.1: an effective value that is not a valid EQName gives XTMM9000.
		{name: "invalid error-code falls back to XTMM9000", want: "XTMM9000", sheet: `
<xsl:template match="/"><xsl:try><xsl:message terminate="yes" error-code="not a name">stop</xsl:message><xsl:catch xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="local-name-from-QName($err:code)"/></xsl:catch></xsl:try></xsl:template>`},
		// §8.3: $err:description is the description passed to fn:error.
		{name: "err:description from fn:error", want: "msg", sheet: `
<xsl:template match="/"><xsl:try><xsl:sequence select="error(QName('urn:e', 'e:e'), 'msg', 42)"/><xsl:catch xmlns:err="http://www.w3.org/2005/xqt-errors"><xsl:value-of select="$err:description"/></xsl:catch></xsl:try></xsl:template>`},
	})
}

// Behaviour that disagrees with the REC, skipped with the bug it shows.
func TestInstrControlKnownBugs(t *testing.T) {
	bugs := []struct {
		bug string
		c   specCase
	}{
		{"BUG: xsl:for-each-group with two grouping attributes raises XTSE1300 (a decimal-format code); §14.1 defines XTSE1080",
			specCase{name: "two grouping attributes", code: "XTSE1080", sheet: `
<xsl:template match="/"><xsl:for-each-group select="*" group-by="." group-adjacent="."><x/></xsl:for-each-group></xsl:template>`}},
		{"BUG: xsl:for-each-group with no grouping attribute raises XTSE1300; §14.1 defines XTSE1080",
			specCase{name: "no grouping attribute", code: "XTSE1080", sheet: `
<xsl:template match="/"><xsl:for-each-group select="*"><x/></xsl:for-each-group></xsl:template>`}},
		{"BUG: xsl:try without xsl:catch fails without an error code; the content model violation is XTSE0010",
			specCase{name: "try without catch", code: "XTSE0010", sheet: `
<xsl:template match="/"><xsl:try><x/></xsl:try></xsl:template>`}},
		{"BUG: an xsl:map-entry key of two items fails without an error code; §21.? requires XPTY0004",
			specCase{name: "map-entry key sequence", code: "XPTY0004", sheet: `
<xsl:template match="/"><xsl:variable name="m" as="map(*)"><xsl:map><xsl:map-entry key="1, 2" select="3"/></xsl:map></xsl:variable><xsl:value-of select="map:size($m)"/></xsl:template>`}},
		{"BUG: xsl:map content that is not a map is reported only by the variable's @as (XTTE0570); §21.? defines XTTE3375 for xsl:map itself",
			specCase{name: "non-map content", code: "XTTE3375", sheet: `
<xsl:template match="/"><xsl:variable name="m" as="map(*)"><xsl:map><xsl:sequence select="1"/></xsl:map></xsl:variable><xsl:value-of select="map:size($m)"/></xsl:template>`}},
		{"BUG: an unmatched { in an AVT fails without an error code; §5.6.1 defines XTSE0350",
			specCase{name: "unclosed brace", code: "XTSE0350", sheet: specXML(`<xsl:template match="/"><e a="x{1"/></xsl:template>`)}},
		{"BUG: an unterminated comment in an AVT expression fails without an error code (XTSE0350, the expression is not closed)",
			specCase{name: "unclosed comment", code: "XTSE0350", sheet: specXML(`<xsl:template match="/"><e a="{1 (: c }"/></xsl:template>`)}},
		{"BUG: xsl:call-template omitting a required non-tunnel parameter raises the dynamic XTDE0700; §10.1.? makes it the static XTSE0690",
			specCase{name: "call-template missing a required parameter", code: "XTSE0690", sheet: `
<xsl:template name="t"><xsl:param name="a" required="yes"/></xsl:template>
<xsl:template match="/"><xsl:call-template name="t"/></xsl:template>`}},
		{"BUG: xsl:call-template naming no template fails without an error code; §10.1 defines XTSE0650",
			specCase{name: "call-template to an unknown template", code: "XTSE0650", sheet: `
<xsl:template match="/"><xsl:call-template name="nope"/></xsl:template>`}},
	}
	for _, b := range bugs {
		t.Run(b.c.name, func(t *testing.T) {
			t.Skip(b.bug)
			specCheck(t, "", b.c)
		})
	}
}
