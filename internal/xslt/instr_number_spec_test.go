package xslt

import "testing"

// numSpec wraps one xsl:number usage in a template matching the document.
func numSpec(name, body, want string) specCase {
	return specCase{name: name, want: want, sheet: `<xsl:template match="/">` + body + `</xsl:template>`}
}

// XSLT 3.0 §12.3 formatting a number sequence (format tokens per F&O 3.1
// §4.6.1 fn:format-integer, which §12.4 defers to).
func TestNumberSpecFormatting(t *testing.T) {
	specRunAll(t, "", []specCase{
		numSpec("several values, several tokens", `<xsl:number value="1,2,3" format="1.a.i"/>`, "1.b.iii"),
		// More numbers than tokens: the last token and separator are reused.
		numSpec("tokens reused", `<xsl:number value="1,2,3" format="1.a"/>`, "1.b.c"),
		// One token, no separator: the default separator is ".".
		numSpec("default separator and suffix", `<xsl:number value="1,2" format="1."/>`, "1.2."),
		numSpec("prefix and suffix", `<xsl:number value="3" format="(1)"/>`, "(3)"),
		numSpec("zero padding wider than value", `<xsl:number value="1234" format="01"/>`, "1234"),
		numSpec("empty format means 1", `<xsl:number value="5" format="{''}"/>`, "5"),
		numSpec("alphabetic", `<xsl:number value="26, 28, 703" format="a"/>`, "z.ab.aaa"),
		numSpec("upper alphabetic", `<xsl:number value="28" format="A"/>`, "AB"),
		numSpec("roman", `<xsl:number value="1999" format="i"/>`, "mcmxcix"),
		numSpec("upper roman", `<xsl:number value="3888" format="I"/>`, "MMMDCCCLXXXVIII"),
		// F&O §4.6.1: "w" words, "W" upper case, "Ww" title case.
		numSpec("words", `<xsl:number value="3" format="w"/>`, "three"),
		numSpec("upper words", `<xsl:number value="3" format="W"/>`, "THREE"),
		numSpec("title words", `<xsl:number value="3" format="Ww"/>`, "Three"),
		numSpec("words for zero", `<xsl:number value="0" format="w"/>`, "zero"),
		// F&O example: format-integer(14, 'Ww;o') = "Fourteenth".
		numSpec("ordinal title words", `<xsl:number value="14" format="Ww" ordinal="yes"/>`, "Fourteenth"),
		numSpec("ordinal words", `<xsl:number value="1, 2, 3, 20" format="w" ordinal="yes"/>`, "first.second.third.twentieth"),
		numSpec("decimal digit family", `<xsl:number value="12" format="&#x661;"/>`, "١٢"),
		numSpec("padded digit family", `<xsl:number value="7" format="&#x660;&#x661;"/>`, "٠٧"),
		numSpec("fullwidth digits", `<xsl:number value="305" format="&#xFF11;"/>`, "３０５"),
		numSpec("grouping", `<xsl:number value="1234567" grouping-separator="." grouping-size="2"/>`, "1.23.45.67"),
		numSpec("grouping AVT", `<xsl:number value="1234567" grouping-separator="{' '}" grouping-size="{3}"/>`, "1 234 567"),
		numSpec("rounding of a non-integer value", `<xsl:number value="2.5, 3.4"/>`, "3.3"),
		numSpec("untypedAtomic value", `<xsl:variable name="v"><v>12</v></xsl:variable><xsl:number value="$v/v"/>`, "12"),
		// The integer is exact, not routed through xs:double.
		numSpec("big integer value", `<xsl:number value="xs:integer('123456789012345678901234')"/>`, "123456789012345678901234"),
		numSpec("big integer with grouping", `<xsl:number value="xs:integer('123456789012345678901')" grouping-separator="," grouping-size="3"/>`, "123,456,789,012,345,678,901"),
		numSpec("zero", `<xsl:number value="0"/>`, "0"),
		// §12.3 start-at.
		numSpec("start-at", `<xsl:number value="1, 2" start-at="0"/>`, "0.1"),
		numSpec("start-at per level", `<xsl:number value="1, 1, 1" start-at="10 20"/>`, "10.20.20"),
		// §12.3 lang: German words.
		numSpec("lang de", `<xsl:number value="3" format="w" lang="de"/>`, "drei"),
		// XTDE0980: not convertible to a non-negative integer.
		{name: "negative value", code: "XTDE0980", sheet: `<xsl:template match="/"><xsl:number value="-1"/></xsl:template>`},
		{name: "big negative value", code: "XTDE0980", sheet: `<xsl:template match="/"><xsl:number value="xs:integer('-123456789012345678901234')"/></xsl:template>`},
		{name: "NaN value", code: "XTDE0980", sheet: `<xsl:template match="/"><xsl:number value="number('x')"/></xsl:template>`},
		{name: "string value", code: "XTDE0980", sheet: `<xsl:template match="/"><xsl:number value="'fizz'"/></xsl:template>`},
	})
}

// XSLT 3.0 §12.2 numbering based on position in a document: level, count,
// from, select (the REC's own examples).
func TestNumberSpecLevels(t *testing.T) {
	book := `<book><chapter><title/><section><title/><note/></section><section><title/><note/><note/></section></chapter><chapter><title/><note/><section><title/><note/></section></chapter></book>`
	specRunAll(t, "", []specCase{
		// §12.2 example: multi-level heading numbers.
		{name: "level multiple", src: book, want: "1 1.1 1.2 2 2.1 ", sheet: `
<xsl:template match="/"><xsl:apply-templates select="//title"/></xsl:template>
<xsl:template match="title"><xsl:number level="multiple" count="chapter|section" format="1.1 "/></xsl:template>`},
		// §12.2 example: notes numbered per chapter with level="any" from="chapter".
		{name: "level any from", src: book, want: "(1)(2)(3)(1)(2)", sheet: `
<xsl:template match="/"><xsl:apply-templates select="//note"/></xsl:template>
<xsl:template match="note"><xsl:number level="any" from="chapter" format="(1)"/></xsl:template>`},
		{name: "level any without from", src: book, want: "1 2 3 4 5 ", sheet: `
<xsl:template match="/"><xsl:apply-templates select="//note"/></xsl:template>
<xsl:template match="note"><xsl:number level="any" format="1 "/></xsl:template>`},
		{name: "level any with count", src: book, want: "3 10 ", sheet: `
<xsl:template match="/"><xsl:apply-templates select="//section[note][1]/note[1] | (//note)[last()]"/></xsl:template>
<xsl:template match="note"><xsl:number level="any" count="title|note" format="1 "/></xsl:template>`},
		{name: "level single with count ancestor", src: book, want: "1.2.1.", sheet: `
<xsl:template match="/"><xsl:apply-templates select="//section/note[1]"/></xsl:template>
<xsl:template match="note"><xsl:number count="section" format="1."/></xsl:template>`},
		{name: "level single with from", src: `<r><a><x/><b/><b/></a><a><b/></a></r>`, want: "2", sheet: `
<xsl:template match="/"><xsl:apply-templates select="(//b)[2]"/></xsl:template>
<xsl:template match="b"><xsl:number from="a"/></xsl:template>`},
		// No node matches @count: the empty sequence formats as "".
		{name: "nothing counted", src: book, want: "[]", sheet: `
<xsl:template match="/">[<xsl:for-each select="/book"><xsl:number count="nonexistent"/></xsl:for-each>]</xsl:template>`},
		{name: "select attribute", src: book, want: "2", sheet: `
<xsl:template match="/"><xsl:number select="/book/chapter[2]"/></xsl:template>`},
		// XTTE1000: @select is not a single node.
		{name: "select two nodes", src: book, code: "XTTE1000", sheet: `
<xsl:template match="/"><xsl:number select="//chapter"/></xsl:template>`},
		{name: "select an atomic value", src: book, code: "XTTE1000", sheet: `
<xsl:template match="/"><xsl:number select="1"/></xsl:template>`},
		// XTTE0990: the context item is not a node.
		{name: "atomic context item", src: book, code: "XTTE0990", sheet: `
<xsl:template match="/"><xsl:for-each select="1"><xsl:number/></xsl:for-each></xsl:template>`},
		// XTSE0975: @value with @select/@level/@count/@from.
		{name: "value with level", code: "XTSE0975", sheet: `
<xsl:template match="/"><xsl:number value="1" level="any"/></xsl:template>`},
		{name: "value with count", code: "XTSE0975", sheet: `
<xsl:template match="/"><xsl:number value="1" count="x"/></xsl:template>`},
		{name: "value with select", code: "XTSE0975", sheet: `
<xsl:template match="/"><xsl:number value="1" select="."/></xsl:template>`},
	})
}

// §12.2: an omitted @count matches nodes of the same kind as the selected
// node and, if it has an expanded QName, the same expanded QName.
func TestNumberSpecDefaultCount(t *testing.T) {
	specRunAll(t, "", []specCase{
		// An attribute matches itself; it has no preceding siblings.
		{name: "attribute", src: `<r a="1" b="2"/>`, want: "1", sheet: `<xsl:template match="/"><xsl:for-each select="/r/@b"><xsl:number/></xsl:for-each></xsl:template>`},
		{name: "attribute level any", src: `<r a="1" b="2"><s b="3"/></r>`, want: "1", sheet: `<xsl:template match="/"><xsl:for-each select="//s/@b"><xsl:number level="any"/></xsl:for-each></xsl:template>`},
		// A processing instruction counts only those with its own name.
		{name: "pi", src: `<r><?p?><?q?><?p?></r>`, want: "2", sheet: `<xsl:template match="/"><xsl:for-each select="/r/processing-instruction('p')[2]"><xsl:number/></xsl:for-each></xsl:template>`},
		{name: "pi level any", src: `<r><?p?><x><?q?></x><?p?></r>`, want: "2", sheet: `<xsl:template match="/"><xsl:for-each select="/r/processing-instruction('p')[2]"><xsl:number level="any"/></xsl:for-each></xsl:template>`},
	})
}

// Behaviour that disagrees with the REC; kept as skipped tests naming the bug.
func TestNumberSpecKnownBugs(t *testing.T) {
	cases := []struct {
		bug string
		c   specCase
	}{
		{"BUG: invalid start-at AVT value has no error code, XTDE0030 expected",
			specCase{name: "start-at", code: "XTDE0030", sheet: `<xsl:template match="/"><xsl:number value="1" start-at="{'x'}"/></xsl:template>`}},
		{"BUG: an invalid AVT-valued @lang raises XTTE0990; XTDE0030 is the code for an invalid AVT value (the W3C case number-0826 expects it too)",
			specCase{name: "lang", code: "XTDE0030", sheet: `<xsl:template match="/"><xsl:number value="1" lang="{'42'}"/></xsl:template>`}},
		{"BUG: invalid static @level has no error code, XTSE0020 expected",
			specCase{name: "level", code: "XTSE0020", sheet: `<xsl:template match="/"><xsl:number level="all"/></xsl:template>`}},
	}
	for _, k := range cases {
		t.Run(k.c.name, func(t *testing.T) {
			t.Skip(k.bug)
			specCheck(t, "", k.c)
		})
	}
}
