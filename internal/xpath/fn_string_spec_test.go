package xpath

import "testing"

const strHTMLCI = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"

// TestStrContainsToken covers fn:contains-token (F&O 3.1 §5.3.9 examples).
func TestStrContainsToken(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `contains-token("red green blue ", "red")`, want: "true"},
		{expr: `contains-token(("red", "green", "blue"), " red ")`, want: "true"},
		{expr: `contains-token("red, green, blue", "red")`, want: "false"},
		{expr: `contains-token("red green blue", "RED", "` + strHTMLCI + `")`, want: "true"},
		{expr: `contains-token("red green blue", "RED")`, want: "false"},
		{expr: `contains-token("red green blue", "RED", ())`, want: "false"},
		// Only XML whitespace separates tokens; an all-blank token never matches.
		{expr: `contains-token("a` + " " + `b", "a")`, want: "false"},
		{expr: `contains-token("a b", "  ")`, want: "false"},
		{expr: `contains-token((), "a")`, want: "false"},
		{expr: `contains-token("a", "a", "http://example.com/unknown")`, err: "FOCH0002"},
	})
}

// TestStrCodepoints covers fn:codepoints-to-string (§5.2.1), the XML Char
// production check (FOCH0001) and fn:codepoint-equal (§5.3.7).
func TestStrCodepoints(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `codepoints-to-string((66, 65, 67, 72))`, want: "BACH"},
		{expr: `codepoints-to-string((2309, 2358, 2378, 2325))`, want: "अशॊक"},
		{expr: `codepoints-to-string(())`, want: ""},
		{expr: `codepoints-to-string(97)`, want: "a"},
		{expr: `codepoints-to-string(xs:untypedAtomic('98'))`, want: "b"},
		{expr: `codepoints-to-string(0)`, err: "FOCH0001"},
		{expr: `codepoints-to-string(55296)`, err: "FOCH0001"},
		{expr: `codepoints-to-string(1114112)`, err: "FOCH0001"},
		{expr: `codepoints-to-string(65534)`, err: "FOCH0001"},
		{expr: `codepoint-equal("abcd", "abcd")`, want: "true"},
		{expr: `codepoint-equal("abcd", "abcd ")`, want: "false"},
		{expr: `codepoint-equal("", "")`, want: "true"},
		{expr: `codepoint-equal("", ())`, want: "()"},
		{expr: `codepoint-equal((), ())`, want: "()"},
		{expr: `codepoint-equal(1, "1")`, err: "XPTY0004"},
		{expr: `codepoint-equal("1", 1)`, err: "XPTY0004"},
	})
	// #x8 is a Char in XML 1.1 but not in XML 1.0.
	coreRun(t, "", func(c *Context) { c.XMLVersion = "1.0" }, []coreCase{
		{expr: `codepoints-to-string(8)`, err: "FOCH0001"},
		{expr: `string-to-codepoints(codepoints-to-string((9, 10, 13, 32)))`, want: "9|10|13|32"},
	})
	coreRun(t, "", func(c *Context) { c.XMLVersion = "1.1" }, []coreCase{
		{expr: `string-to-codepoints(codepoints-to-string(8))`, want: "8"},
	})
}

// TestStrCompare covers fn:compare (§5.3.6) under the codepoint collation,
// an explicit collation and the in-scope default collation.
func TestStrCompare(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `compare('abc', 'abc')`, want: "0"},
		{expr: `compare('abc', 'abd')`, want: "-1"},
		{expr: `compare('b', 'a')`, want: "1"},
		// Codepoint order: 's' (U+0073) precedes 'ß' (U+00DF).
		{expr: `compare('Strasse', 'Straße')`, want: "-1"},
		{expr: `compare('a', 'b', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')`, want: "-1"},
		{expr: `compare('ABC', 'abc', '` + strHTMLCI + `')`, want: "0"},
		{expr: `compare('Strasse', 'Straße', 'http://www.w3.org/2013/collation/UCA?lang=de;strength=primary')`, want: "0"},
		{expr: `compare((), 'a')`, want: "()"},
		{expr: `compare('a', ())`, want: "()"},
		{expr: `compare(1, 'a')`, err: "XPTY0004"},
		{expr: `compare('a', 1)`, err: "XPTY0004"},
		{expr: `compare('a', 'b', 'http://example.com/unknown')`, err: "FOCH0002"},
	})
	ci, err := ResolveCollator(strHTMLCI)
	if err != nil {
		t.Fatal(err)
	}
	coreRun(t, "", func(c *Context) { c.DefaultCollation = ci }, []coreCase{
		{expr: `compare('ABC', 'abc')`, want: "0"},
	})
}

// TestStrNormalizeUnicode covers fn:normalize-unicode (§5.4.6): the effective
// form is upper-case(normalize-space($form)), "" means no normalization.
func TestStrNormalizeUnicode(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `string-to-codepoints(normalize-unicode(codepoints-to-string((101, 769))))`, want: "233"},
		{expr: `string-to-codepoints(normalize-unicode(codepoints-to-string((101, 769)), 'NFC'))`, want: "233"},
		{expr: `string-to-codepoints(normalize-unicode(codepoints-to-string(233), 'NFD'))`, want: "101|769"},
		{expr: `normalize-unicode(codepoints-to-string(64257), 'NFKC')`, want: "fi"},
		{expr: `normalize-unicode(codepoints-to-string(64257), ' nfkd ')`, want: "fi"},
		{expr: `string-to-codepoints(normalize-unicode(codepoints-to-string((101, 769)), ''))`, want: "101|769"},
		// FULLY-NORMALIZED: a leading composing character gets a space prefix.
		{expr: `string-to-codepoints(normalize-unicode(codepoints-to-string(769), 'FULLY-NORMALIZED'))`, want: "32|769"},
		{expr: `normalize-unicode('abc', 'fully-normalized')`, want: "abc"},
		{expr: `normalize-unicode((), 'NFC')`, want: ""},
		{expr: `normalize-unicode('a', 'NFX')`, err: "FOCH0003"},
		{expr: `normalize-unicode(12)`, err: "XPTY0004"},
		{expr: `normalize-unicode('a', ())`, err: "XPTY0004"},
	})
}

// TestStrCollationKey covers fn:collation-key (§5.3.4): two keys are equal
// exactly when the strings are equal under the collation.
func TestStrCollationKey(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `collation-key("abc") eq collation-key("abc")`, want: "true"},
		{expr: `collation-key("abc") eq collation-key("abd")`, want: "false"},
		{expr: `collation-key("abc") instance of xs:base64Binary`, want: "true"},
		{expr: `collation-key("ABC", "` + strHTMLCI + `") eq collation-key("abc", "` + strHTMLCI + `")`, want: "true"},
		{expr: `collation-key("ABC", "` + strHTMLCI + `") eq collation-key("abd", "` + strHTMLCI + `")`, want: "false"},
		{expr: `collation-key(xs:untypedAtomic("a")) eq collation-key("a")`, want: "true"},
		{expr: `collation-key(())`, err: "XPTY0004"},
		{expr: `collation-key(("a", "b"))`, err: "XPTY0004"},
		{expr: `collation-key(1)`, err: "XPTY0004"},
		{expr: `collation-key("a", "http://example.com/unknown")`, err: "FOCH0002"},
	})
}

// TestStrGenerateID covers fn:generate-id (§14.5.4): a stable identifier per
// node, distinct across nodes, consisting of ASCII alphanumerics and starting
// with a letter; () gives "".
func TestStrGenerateID(t *testing.T) {
	const doc = `<r xmlns:p="urn:p"><a/><a/></r>`
	coreRun(t, doc, nil, []coreCase{
		{expr: `generate-id(/r) eq generate-id(/r)`, want: "true"},
		{expr: `generate-id(/r/a[1]) ne generate-id(/r/a[2])`, want: "true"},
		{expr: `generate-id(/r) ne generate-id(/)`, want: "true"},
		{expr: `/r/a[1]/generate-id() eq generate-id(/r/a[1])`, want: "true"},
		{expr: `matches(generate-id(/r/a[2]), '^[A-Za-z][A-Za-z0-9]*$')`, want: "true"},
		{expr: `count(distinct-values(/r/a/namespace::*!generate-id(.)))`, want: "4"},
		{expr: `generate-id(())`, want: ""},
		{expr: `generate-id(1)`, err: "XPTY0004"},
		{expr: `generate-id(/r/a)`, err: "XPTY0004"},
	})
	coreRun(t, "", func(c *Context) { c.Node = nil; c.NoFocus = true }, []coreCase{
		{expr: `generate-id()`, err: "XPDY0002"},
	})
}
