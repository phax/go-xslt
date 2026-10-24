package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// coreCase is one XPath expression with either its expected rendering (see
// coreRender) or the W3C error code it must raise.
type coreCase struct {
	expr string
	want string // rendering of the result; ignored when err is set
	err  string // expected error code (e.g. "XPTY0004"), "" for success
}

// coreRender renders a result as its items' string values joined by "|";
// the empty sequence renders as "()".
func coreRender(v Object) string {
	items := Items(v)
	if len(items) == 0 {
		return "()"
	}
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = itemString(it)
	}
	return strings.Join(parts, "|")
}

// coreRun evaluates every case against the document parsed from doc ("<r/>"
// when empty). setup, when non-nil, adjusts the evaluation context.
func coreRun(t *testing.T, doc string, setup func(*Context), cases []coreCase) {
	t.Helper()
	if doc == "" {
		doc = "<r/>"
	}
	root, err := xmltree.Parse(doc)
	if err != nil {
		t.Fatalf("parse xml: %v", err)
	}
	for _, c := range cases {
		ctx := &Context{Node: root, Pos: 1, Size: 1}
		if setup != nil {
			setup(ctx)
		}
		var v Object
		p, perr := Parse(c.expr)
		if perr == nil {
			v, err = p.Eval(ctx)
		} else {
			err = perr
		}
		if c.err != "" {
			if err == nil {
				t.Errorf("%s = %q, want error %s", c.expr, coreRender(v), c.err)
			} else if !strings.Contains(err.Error(), c.err) {
				t.Errorf("%s: error %q, want code %s", c.expr, err.Error(), c.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if got := coreRender(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

// TestCoreStringFunctions covers the F&O 3.1 §5.4/§5.5 string functions with
// the REC's own examples.
func TestCoreStringFunctions(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		// §5.4.1 fn:concat
		{expr: `concat('un', 'grateful')`, want: "ungrateful"},
		{expr: `concat('Thy ', (), 'old ', "groans", "", ' ring', ' yet', ' in', ' my', ' ancient',' ears.')`, want: "Thy old groans ring yet in my ancient ears."},
		{expr: `concat('Ciao!',())`, want: "Ciao!"},
		{expr: `concat(01, 02, 03, 04, true())`, want: "1234true"},
		{expr: `concat(10, '/', 3)`, want: "10/3"},
		// Each argument is xs:anyAtomicType?: an array atomizes to its members.
		{expr: `concat(['a'], 'b')`, want: "ab"},
		{expr: `concat([], 'b')`, want: "b"},
		{expr: `concat(('a', 'b'), 'c')`, err: "XPTY0004"},
		{expr: `concat(['a', 'b'], 'c')`, err: "XPTY0004"},
		{expr: `concat(map{}, 'c')`, err: "FOTY0013"},
		{expr: `concat(abs#1, 'c')`, err: "FOTY0013"},

		// §5.4.3 fn:substring
		{expr: `substring("motor car", 6)`, want: " car"},
		{expr: `substring("metadata", 4, 3)`, want: "ada"},
		{expr: `substring("12345", 1.5, 2.6)`, want: "234"},
		{expr: `substring("12345", 0, 3)`, want: "12"},
		{expr: `substring("12345", 5, -3)`, want: ""},
		{expr: `substring("12345", -3, 5)`, want: "1"},
		{expr: `substring("12345", 0 div 0E0, 3)`, want: ""},
		{expr: `substring("12345", 1, 0 div 0E0)`, want: ""},
		{expr: `substring((), 1, 3)`, want: ""},
		{expr: `substring("12345", -42, 1 div 0E0)`, want: "12345"},
		{expr: `substring("12345", -1 div 0E0, 1 div 0E0)`, want: ""},

		// §5.4.4 fn:string-length
		{expr: `string-length("Harp not on that string, madam; that is past.")`, want: "45"},
		{expr: `string-length(())`, want: "0"},
		{expr: `string-length(("a", "b"))`, err: "XPTY0004"},
		{expr: `string-length(xs:date('2020-01-01'))`, err: "XPTY0004"},
		{expr: `string-length(xs:untypedAtomic('abc'))`, want: "3"},

		// §5.4.5 fn:normalize-space
		{expr: `normalize-space(" The    wealthy curled darlings   of   our   nation. ")`, want: "The wealthy curled darlings of our nation."},
		{expr: `normalize-space(())`, want: ""},
		{expr: `normalize-space(12)`, err: "XPTY0004"},

		// §5.4.8 fn:translate
		{expr: `translate("bar","abc","ABC")`, want: "BAr"},
		{expr: `translate("--aaa--","abc-","ABC")`, want: "AAA"},
		{expr: `translate("abcdabc", "abc", "AB")`, want: "ABdAB"},
		{expr: `translate((), "a", "b")`, want: ""},
		// The first occurrence of a repeated map character is the one used.
		{expr: `translate("aa", "aa", "xy")`, want: "xx"},
		// A long map string (the indexed path) obeys the same rules.
		{expr: `translate("zebra!", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnop", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")`, want: "ZEBRA!"},
		{expr: `translate("zebra", "abcdefghijklmnopqrstuvwxyzABCDEFGH", "")`, want: ""},
		{expr: `translate("a", (), "b")`, err: "XPTY0004"},
		{expr: `translate("a", "a", ())`, err: "XPTY0004"},
		{expr: `translate(1, "a", "b")`, err: "XPTY0004"},

		// §5.4.7 fn:upper-case / §5.4.6 fn:lower-case (incl. one-to-many
		// unconditional SpecialCasing mappings).
		{expr: `upper-case("abCd0")`, want: "ABCD0"},
		{expr: `lower-case("ABc!D")`, want: "abc!d"},
		{expr: `upper-case("ß")`, want: "SS"},
		{expr: `string-to-codepoints(lower-case("İ"))`, want: "105|775"},
		{expr: `upper-case(())`, want: ""},
		{expr: `upper-case(12)`, err: "XPTY0004"},
		{expr: `lower-case(12)`, err: "XPTY0004"},

		// §5.5.1-§5.5.5 contains / starts-with / ends-with / substring-*.
		{expr: `contains("tattoo", "t")`, want: "true"},
		{expr: `contains("tattoo", "ttt")`, want: "false"},
		{expr: `contains("", ())`, want: "true"},
		{expr: `starts-with("tattoo", "tat")`, want: "true"},
		{expr: `starts-with("tattoo", "att")`, want: "false"},
		{expr: `starts-with((), ())`, want: "true"},
		{expr: `ends-with("tattoo", "tattoo")`, want: "true"},
		{expr: `ends-with("tattoo", "atto")`, want: "false"},
		{expr: `ends-with((), ())`, want: "true"},
		{expr: `substring-before("tattoo", "attoo")`, want: "t"},
		{expr: `substring-before("tattoo", "tatto")`, want: ""},
		{expr: `substring-before((), ())`, want: ""},
		{expr: `substring-before("abc", "x")`, want: ""},
		{expr: `substring-after("tattoo", "tat")`, want: "too"},
		{expr: `substring-after("tattoo", "tattoo")`, want: ""},
		{expr: `substring-after((), ())`, want: ""},
		{expr: `substring-after("abc", "x")`, want: ""},
		// The codepoint collation named explicitly.
		{expr: `contains("abc", "b", "http://www.w3.org/2005/xpath-functions/collation/codepoint")`, want: "true"},
		// §5.3.5 HTML ASCII case-insensitive collation.
		{expr: `contains("abc", "B", "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive")`, want: "true"},
		{expr: `starts-with("Abc", "aB", "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive")`, want: "true"},
		{expr: `ends-with("abC", "Bc", "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive")`, want: "true"},
		{expr: `substring-before("xxABcyy", "abc", "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive")`, want: "xx"},
		{expr: `substring-after("xxABcyy", "abc", "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive")`, want: "yy"},
		// Unsupported collation: FOCH0002.
		{expr: `contains("abc", "b", "http://example.com/no-such-collation")`, err: "FOCH0002"},
		{expr: `starts-with("abc", "b", "http://example.com/no-such-collation")`, err: "FOCH0002"},
		{expr: `ends-with("abc", "b", "http://example.com/no-such-collation")`, err: "FOCH0002"},
		{expr: `substring-before("abc", "b", "http://example.com/no-such-collation")`, err: "FOCH0002"},
		{expr: `substring-after("abc", "b", "http://example.com/no-such-collation")`, err: "FOCH0002"},
		// xs:string? operands: a number is a type error.
		{expr: `contains("ee", 12)`, err: "XPTY0004"},
		{expr: `starts-with(12, "1")`, err: "XPTY0004"},
		{expr: `ends-with("12", 2)`, err: "XPTY0004"},
		{expr: `substring-before('Michael Kay', 2)`, err: "XPTY0004"},
		{expr: `substring-after(1, 'a')`, err: "XPTY0004"},
		{expr: `contains(("a", "b"), "a")`, err: "XPTY0004"},

		// §5.4.2 fn:string-join
		{expr: `string-join(('Now', 'is', 'the', 'time', '...'), ' ')`, want: "Now is the time ..."},
		{expr: `string-join(1 to 9)`, want: "123456789"},
		{expr: `string-join((), 'separator')`, want: ""},
		{expr: `string-join(('a', 'b'), ())`, err: "XPTY0004"},
		{expr: `string-join(('a', 'b'), 1)`, err: "XPTY0004"},

		// §2.3 fn:string
		{expr: `string(23)`, want: "23"},
		{expr: `string(false())`, want: "false"},
		{expr: `string("Paris")`, want: "Paris"},
		{expr: `string(())`, want: ""},
		{expr: `string((1, 2))`, err: "XPTY0004"},
		{expr: `string(map{})`, err: "FOTY0014"},
		{expr: `string([1])`, err: "FOTY0014"},
		{expr: `string(abs#1)`, err: "FOTY0014"},
	})
}

// TestCoreStringContextForms covers the zero-argument forms that default to
// the context item (F&O 3.1 §5.4.4, §5.4.5, §2.3, §4.5.1).
func TestCoreStringContextForms(t *testing.T) {
	const doc = `<r>  hello   world  </r>`
	coreRun(t, doc, nil, []coreCase{
		{expr: `/r/string-length()`, want: "17"},
		{expr: `/r/normalize-space()`, want: "hello world"},
		{expr: `/r/string()`, want: "  hello   world  "},
		{expr: `(1, 22)!string-length()`, want: "1|2"},
		{expr: `(xs:date('2020-01-01'))!normalize-space()`, want: "2020-01-01"},
		{expr: `(1, 2)!string()`, want: "1|2"},
		// fn:number
		{expr: `number('12')`, want: "12"},
		{expr: `number('abc')`, want: "NaN"},
		{expr: `number(())`, want: "NaN"},
		{expr: `number(true())`, want: "1"},
		{expr: `number(xs:date('2020-01-01'))`, want: "NaN"},
		{expr: `number(' 1e2 ') instance of xs:double`, want: "true"},
		{expr: `number((1, 2))`, err: "XPTY0004"},
		{expr: `('7', '8')!number()`, want: "7|8"},
	})
	// With no focus at all the context-defaulting forms are XPDY0002.
	noFocus := func(c *Context) { c.Node = nil; c.NoFocus = true }
	coreRun(t, "", noFocus, []coreCase{
		{expr: `string-length()`, err: "XPDY0002"},
		{expr: `normalize-space()`, err: "XPDY0002"},
		{expr: `number()`, err: "XPDY0002"},
		{expr: `string()`, err: "XPDY0002"},
		{expr: `local-name()`, err: "XPDY0002"},
		{expr: `name()`, err: "XPDY0002"},
		{expr: `namespace-uri()`, err: "XPDY0002"},
		{expr: `root()`, err: "XPDY0002"},
		{expr: `lang('en')`, err: "XPDY0002"},
		{expr: `id('a')`, err: "XPDY0002"},
		{expr: `idref('a')`, err: "XPDY0002"},
		{expr: `position()`, err: "XPDY0002"},
		{expr: `last()`, err: "XPDY0002"},
	})
}

// TestCoreNodeNameFunctions covers fn:name, fn:local-name, fn:namespace-uri
// and fn:root (F&O 3.1 §2.1, §14.1-§14.3, §14.9).
func TestCoreNodeNameFunctions(t *testing.T) {
	const doc = `<p:a xmlns:p="urn:x"><b c="1"/></p:a>`
	coreRun(t, doc, nil, []coreCase{
		{expr: `name(/*)`, want: "p:a"},
		{expr: `local-name(/*)`, want: "a"},
		{expr: `namespace-uri(/*)`, want: "urn:x"},
		{expr: `namespace-uri(/*) instance of xs:anyURI`, want: "true"},
		{expr: `/*/*/name()`, want: "b"},
		{expr: `/*/*/local-name()`, want: "b"},
		{expr: `/*/*/namespace-uri()`, want: ""},
		{expr: `name(//@c)`, want: "c"},
		{expr: `name(/)`, want: ""},
		{expr: `name(())`, want: ""},
		{expr: `local-name(())`, want: ""},
		{expr: `namespace-uri(())`, want: ""},
		{expr: `name(1)`, err: "XPTY0004"},
		{expr: `local-name('a')`, err: "XPTY0004"},
		{expr: `namespace-uri(1)`, err: "XPTY0004"},
		{expr: `(1)!name()`, err: "XPTY0004"},
		{expr: `(1)!local-name()`, err: "XPTY0004"},
		{expr: `root(//b) is /`, want: "true"},
		{expr: `//@c/root() is /`, want: "true"},
		{expr: `count(root(()))`, want: "0"},
		{expr: `root(1)`, err: "XPTY0004"},
		{expr: `(1)!root()`, err: "XPTY0004"},
	})
}

// TestCoreLang covers fn:lang with the F&O 3.1 §14.5.? examples.
func TestCoreLang(t *testing.T) {
	cases := []struct{ doc, expr, want string }{
		{`<para xml:lang="en"/>`, `/para/lang("en")`, "true"},
		{`<div xml:lang="en"><para>And now, and forever!</para></div>`, `/div/para/lang("en")`, "true"},
		{`<para xml:lang="EN"/>`, `/para/lang("en")`, "true"},
		{`<para xml:lang="en-us"/>`, `/para/lang("en")`, "true"},
		{`<para xml:lang="EN"/>`, `/para/lang("fr")`, "false"},
		{`<para xml:lang="english"/>`, `/para/lang("en")`, "false"},
		{`<para/>`, `/para/lang("en")`, "false"},
		{`<div xml:lang="de"><p xml:lang="en"/></div>`, `lang("de", //p)`, "false"},
		{`<div xml:lang="de"><p xml:lang="en"/></div>`, `lang("EN", //p)`, "true"},
		{`<div xml:lang="de"><p a="1"/></div>`, `lang("de", //@a)`, "true"},
		{`<div xml:lang="de"/>`, `(/div)!lang("de")`, "true"},
	}
	for _, c := range cases {
		if got := xpStrCtx(t, c.expr, c.doc); got != c.want {
			t.Errorf("%s on %s = %q, want %q", c.expr, c.doc, got, c.want)
		}
	}
	coreRun(t, "", nil, []coreCase{
		{expr: `lang("en", 1)`, err: "XPTY0004"},
		{expr: `(1)!lang("en")`, err: "XPTY0004"},
	})
}

// TestCoreIDFunctions covers fn:id, fn:element-with-id and fn:idref (F&O
// 3.1 §14.5.1-§14.5.3) on a document whose ID/IDREF(S) attributes are typed by
// its DTD (the attribute is deliberately not called "id").
func TestCoreIDFunctions(t *testing.T) {
	const doc = `<!DOCTYPE r [
<!ATTLIST e k ID #IMPLIED ref IDREF #IMPLIED refs IDREFS #IMPLIED>
]>
<r><e k="a" n="1"/><e k="b" n="2" ref="a"/><e k="c" n="3" refs="a  b"/><f xml:id="x" n="4"/></r>`
	coreRun(t, doc, nil, []coreCase{
		{expr: `id('a')/@n/string()`, want: "1"},
		{expr: `id('c a')/@n/string()`, want: "1|3"},
		{expr: `id(('b', 'c', 'b'))/@n/string()`, want: "2|3"},
		{expr: `id('zz')`, want: "()"},
		{expr: `id(())`, want: "()"},
		// xml:id is an ID attribute (xml:id Recommendation).
		{expr: `id('x')/@n/string()`, want: "4"},
		{expr: `id('a', /r/e[3])/@n/string()`, want: "1"},
		{expr: `/r/e[2]/id('c')/@n/string()`, want: "3"},
		{expr: `element-with-id('b')/@n/string()`, want: "2"},
		{expr: `element-with-id('x a')/@n/string()`, want: "1|4"},
		// fn:idref returns the IDREF(S) attribute nodes referring to the value.
		{expr: `idref('a')/name()`, want: "ref|refs"},
		{expr: `idref('b')/../@n/string()`, want: "3"},
		{expr: `idref('c')`, want: "()"},
		// An idref candidate is not tokenized: 'a b' is not a valid IDREF.
		{expr: `idref('a b')`, want: "()"},
		{expr: `idref(('b', 'zz'), /r)/name()`, want: "refs"},
		{expr: `id('a', 1)`, err: "XPTY0004"},
		{expr: `idref('a', 'x')`, err: "XPTY0004"},
		{expr: `(1)!id('a')`, err: "XPTY0004"},
		{expr: `(1)!idref('a')`, err: "XPTY0004"},
	})
}

// TestCoreTokenize covers fn:tokenize (F&O 3.1 §5.6.5) with the REC examples.
func TestCoreTokenize(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `tokenize(" red green blue ")`, want: "red|green|blue"},
		{expr: `tokenize("The cat sat on the mat", "\s+")`, want: "The|cat|sat|on|the|mat"},
		{expr: `tokenize(" red green blue ", "\s+")`, want: "|red|green|blue|"},
		{expr: `tokenize("1, 15, 24, 50", ",\s*")`, want: "1|15|24|50"},
		{expr: `tokenize("1,15,,24,50,", ",")`, want: "1|15||24|50|"},
		{expr: `tokenize("Some unparsed <br> HTML <BR> text", "\s*<br>\s*", "i")`, want: "Some unparsed|HTML|text"},
		{expr: `tokenize("abracadabra", "(ab)|(a)")`, want: "|r|c|d|r|"},
		{expr: `tokenize("12.3.5.6", ".", "q")`, want: "12|3|5|6"},
		{expr: `tokenize("", "a")`, want: "()"},
		{expr: `tokenize("abba", ".?")`, err: "FORX0003"},
		{expr: `tokenize("abc", "b", "k")`, err: "FORX0001"},
		{expr: `tokenize("abc", "(")`, err: "FORX0002"},
		{expr: `tokenize("abc", "b", ())`, err: "XPTY0004"},
	})
}

// TestCoreMatches covers fn:matches and the regular-expression syntax and
// flags (F&O 3.1 §5.6.1, §5.6.2, §5.6.3).
func TestCoreMatches(t *testing.T) {
	poem := "let $poem := 'Kaum hat dies der Hahn gesehen,&#10;Fängt er auch schon an zu krähen:&#10;Kikeriki! Kikikerikih!!&#10;Tak, tak, tak! - da kommt er schon.' return "
	poem = strings.ReplaceAll(poem, "&#10;", "\n")
	coreRun(t, "", nil, []coreCase{
		{expr: `matches("abracadabra", "bra")`, want: "true"},
		{expr: `matches("abracadabra", "^a.*a$")`, want: "true"},
		{expr: `matches("abracadabra", "^bra")`, want: "false"},
		{expr: poem + `matches($poem, "Kaum.*krähen")`, want: "false"},
		{expr: poem + `matches($poem, "Kaum.*krähen", "s")`, want: "true"},
		{expr: poem + `matches($poem, "^Kaum.*gesehen,$", "m")`, want: "true"},
		{expr: poem + `matches($poem, "^Kaum.*gesehen,$")`, want: "false"},
		{expr: poem + `matches($poem, "kiki", "i")`, want: "true"},
		// 'x' flag examples (§5.6.2).
		{expr: `matches("helloworld", "hello world", "x")`, want: "true"},
		{expr: `matches("helloworld", "hello[ ]world", "x")`, want: "false"},
		{expr: `matches("hello world", "hello\ sworld", "x")`, want: "true"},
		{expr: `matches("hello world", "hello world", "x")`, want: "false"},
		// 'q' flag examples (§5.6.2).
		{expr: `matches("abcd", ".*", "q")`, want: "false"},
		{expr: `matches("Mr. B. Obama", "B. OBAMA", "iq")`, want: "true"},
		// 'm': neither anchor matches at the position after a final newline.
		{expr: `matches(codepoints-to-string((97, 10, 98, 10)), "^$", "m")`, want: "false"},
		{expr: `matches(codepoints-to-string((97, 10, 10, 98)), "^$", "m")`, want: "true"},
		// '.' matches neither #xA nor #xD outside dot-all mode.
		{expr: `matches(codepoints-to-string(13), "^.$")`, want: "false"},
		{expr: `matches(codepoints-to-string(10), "^.$")`, want: "false"},
		{expr: `matches(codepoints-to-string(13), "^.$", "s")`, want: "true"},
		// Back-references (§5.6.1).
		{expr: `matches("abab", "^(ab)\1$")`, want: "true"},
		{expr: `matches("abac", "^(ab)\1$")`, want: "false"},
		{expr: `matches("ABab", "^(ab)\1$", "i")`, want: "true"},
		// Multi-character escapes \i \I \c \C (XSD Part 2 App. F).
		{expr: `matches("a1", "^\i\c*$")`, want: "true"},
		{expr: `matches("1a", "^\i")`, want: "false"},
		{expr: `matches("1", "^\I$")`, want: "true"},
		{expr: `matches(" ", "^\C$")`, want: "true"},
		{expr: `matches("-", "^\C$")`, want: "false"},
		{expr: `matches("1", "^[\i]$")`, want: "false"},
		{expr: `matches("_", "^[\i]$")`, want: "true"},
		{expr: `matches("1", "^[\I]$")`, want: "true"},
		{expr: `matches(" ", "^[\C]$")`, want: "true"},
		{expr: `matches(".", "^[\c]$")`, want: "true"},
		// \d is \p{Nd}; \w excludes punctuation, separators and other.
		{expr: `matches("` + "٣" + `", "^\d$")`, want: "true"},
		{expr: `matches("a", "^\D$")`, want: "true"},
		{expr: `matches("` + "٣" + `", "^[\d]$")`, want: "true"},
		{expr: `matches("a", "^[\D]$")`, want: "true"},
		{expr: `matches("5", "^[\D]$")`, want: "false"},
		{expr: `matches("a", "^\w$")`, want: "true"},
		{expr: `matches("_", "^\w$")`, want: "false"},
		{expr: `matches("_", "^\W$")`, want: "true"},
		{expr: `matches("+", "^[\w]$")`, want: "true"},
		{expr: `matches("_", "^[\w]$")`, want: "false"},
		{expr: `matches("_", "^[\W]$")`, want: "true"},
		{expr: `matches(" ", "^\s$")`, want: "true"},
		{expr: `matches("` + " " + `", "^\s$")`, want: "false"},
		{expr: `matches("x", "^\S$")`, want: "true"},
		{expr: `matches(" ", "^[\s]$")`, want: "true"},
		{expr: `matches("x", "^[\S]$")`, want: "true"},
		{expr: `matches(" ", "^[\S]$")`, want: "false"},
		// Block escapes (XSD Part 2 App. F.1.1).
		{expr: `matches("a", "^\p{IsBasicLatin}$")`, want: "true"},
		{expr: `matches("é", "^\p{IsBasicLatin}$")`, want: "false"},
		{expr: `matches("é", "^\P{IsBasicLatin}$")`, want: "true"},
		{expr: `matches("é", "^[\P{IsBasicLatin}]$")`, want: "true"},
		{expr: `matches("a", "^[\P{IsBasicLatin}]$")`, want: "false"},
		{expr: `matches("é", "^[\p{IsLatin-1Supplement}]$")`, want: "true"},
		{expr: `matches("a", "\p{IsNoSuchBlock}")`, err: "FORX0002"},
		{expr: `matches("a", "[\P{IsNoSuchBlock}]")`, err: "FORX0002"},
		{expr: `matches("a", "\p{L")`, err: "FORX0002"},
		// Category escapes; C includes the unassigned code points.
		{expr: `matches("A", "^\p{Lu}$")`, want: "true"},
		{expr: `matches("a", "^\P{Lu}$")`, want: "true"},
		{expr: `matches(codepoints-to-string(1), "^\p{C}$")`, want: "true"},
		{expr: `matches("a", "^\P{C}$")`, want: "true"},
		{expr: `matches(codepoints-to-string(1), "^[\p{C}]$")`, want: "true"},
		{expr: `matches("a", "^\p{Cn}$")`, want: "false"},
		// The 'i' flag does not change what a category escape matches.
		{expr: `matches("a", "\p{Lu}", "i")`, want: "false"},
		{expr: `matches("A", "\p{Lu}", "i")`, want: "true"},
		// Quantifiers above RE2's own limit.
		{expr: `matches(string-join((1 to 1500)!'a'), "^a{1500}$")`, want: "true"},
		{expr: `matches(string-join((1 to 1499)!'a'), "^a{1500}$")`, want: "false"},
		{expr: `matches("aaa", "a{2147483647}")`, want: "false"},
		{expr: `matches("aaa", "a{2147483647,}")`, want: "false"},
		{expr: `matches("aaa", "a{2147483647,2147483648}")`, want: "false"},
		{expr: `matches("aaa", "^a{2,3}$")`, want: "true"},
		{expr: `matches("a{", "a\{")`, want: "true"},
		{expr: `matches("{", "[{]")`, want: "true"},
		// Grammar errors (FORX0002).
		{expr: `matches("a", "(")`, err: "FORX0002"},
		{expr: `matches("a", "a{")`, err: "FORX0002"},
		{expr: `matches("a", "a{,2}")`, err: "FORX0002"},
		{expr: `matches("a", "a}")`, err: "FORX0002"},
		{expr: `matches("a", "(?i)a")`, err: "FORX0002"},
		{expr: `matches("a", "(?=a)")`, err: "FORX0002"},
		{expr: `matches("a", "(?:a)")`, want: "true"},
		{expr: `matches("a", "\b")`, err: "FORX0002"},
		{expr: `matches("a", "\0")`, err: "FORX0002"},
		{expr: `matches("a", "(a)[\1]")`, err: "FORX0002"},
		// Flags (§5.6.2): an unknown flag is FORX0001.
		{expr: `matches("a", "a", "z")`, err: "FORX0001"},
		{expr: `matches("a", "a", ())`, err: "XPTY0004"},
		{expr: `matches("a", "a", 1)`, err: "XPTY0004"},
		{expr: `matches("a", ())`, err: "XPTY0004"},
		{expr: `matches((), "a")`, want: "false"},
		{expr: `matches((), "^$")`, want: "true"},
	})
}

// TestCoreReplace covers fn:replace (F&O 3.1 §5.6.4) with the REC examples
// and the $N rules.
func TestCoreReplace(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `replace("abracadabra", "bra", "*")`, want: "a*cada*"},
		{expr: `replace("abracadabra", "a.*a", "*")`, want: "*"},
		{expr: `replace("abracadabra", "a.*?a", "*")`, want: "*c*bra"},
		{expr: `replace("abracadabra", "a", "")`, want: "brcdbr"},
		{expr: `replace("abracadabra", "a(.)", "a$1$1")`, want: "abbraccaddabbra"},
		{expr: `replace("abracadabra", ".*?", "$1")`, err: "FORX0003"},
		{expr: `replace("AAAA", "A+", "b")`, want: "b"},
		{expr: `replace("AAAA", "A+?", "b")`, want: "bbbb"},
		{expr: `replace("darted", "^(.*?)d(.*)$", "$1c$2")`, want: "carted"},
		{expr: `replace("abcd", "(ab)|(a)", "[1=$1][2=$2]")`, want: "[1=ab][2=]cd"},
		// $0 is the whole match; S < N <= 9 gives ""; N > 9 and > S drops the
		// last digit as a literal.
		{expr: `replace("abc", "b", "[$0]")`, want: "a[b]c"},
		{expr: `replace("abc", "(b)", "[$2]")`, want: "a[]c"},
		{expr: `replace("abc", "(b)", "[$12]")`, want: "a[b2]c"},
		// \$ and \\ are literal.
		{expr: `replace("abc", "b", "\$")`, want: "a$c"},
		{expr: `replace("abc", "b", "\\")`, want: `a\c`},
		{expr: `replace("abc", "b", "$")`, err: "FORX0004"},
		{expr: `replace("abc", "b", "\x")`, err: "FORX0004"},
		{expr: `replace("abc", "b", "x\")`, err: "FORX0004"},
		// 'q' flag examples (§5.6.2).
		{expr: `replace("a\b\c", "\", "\\", "q")`, want: `a\\b\\c`},
		{expr: `replace("a/b/c", "/", "$", "q")`, want: "a$b$c"},
		{expr: `replace("A.b.a.", "a.", "$x", "iq")`, want: "$xb.$x"},
		{expr: `replace("abc", "", "x", "q")`, err: "FORX0003"},
		{expr: `replace("abc", "B", "x", "i")`, want: "axc"},
		{expr: `replace("abc", "b", "x", "z")`, err: "FORX0001"},
		{expr: `replace("abc", "(", "x")`, err: "FORX0002"},
		{expr: `replace("abc", (), "x")`, err: "XPTY0004"},
		{expr: `replace("abc", "b", ())`, err: "XPTY0004"},
		{expr: `replace((), "b", "x")`, want: ""},
	})
}

// TestCoreConstructors covers the xs: constructor functions dispatched from
// functions.go (XPath 3.1 §3.1.5.2 / F&O 3.1 §18, §19), including the three
// built-in list types.
func TestCoreConstructors(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `xs:integer('5') + 1`, want: "6"},
		{expr: `xs:integer(())`, want: "()"},
		{expr: `xs:integer((1, 2))`, err: "XPTY0004"},
		{expr: `xs:double('INF') eq 1 div 0e0`, want: "true"},
		{expr: `xs:float('-INF') lt 0`, want: "true"},
		{expr: `xs:double(())`, want: "()"},
		{expr: `xs:double('abc')`, err: "FORG0001"},
		{expr: `xs:gYear('2020') instance of xs:gYear`, want: "true"},
		{expr: `string(xs:gYearMonth('2020-02'))`, want: "2020-02"},
		{expr: `xs:anyURI('http://example.com/a') instance of xs:anyURI`, want: "true"},
		{expr: `xs:anyURI(())`, want: "()"},
		// xs:QName: a prefix is resolved in the static context.
		{expr: `namespace-uri-from-QName(xs:QName('xs:integer'))`, want: "http://www.w3.org/2001/XMLSchema"},
		{expr: `local-name-from-QName(xs:QName(' local '))`, want: "local"},
		{expr: `xs:QName(xs:QName('xs:integer')) eq xs:QName('xs:integer')`, want: "true"},
		{expr: `xs:QName(())`, want: "()"},
		{expr: `xs:QName('undeclared:a')`, err: "FONS0004"},
		{expr: `xs:QName(('a', 'b'))`, err: "XPTY0004"},
		// List types: whitespace-separated tokens, each of the item type.
		{expr: `count(xs:NMTOKENS(' a b  c '))`, want: "3"},
		{expr: `xs:NMTOKENS(' a b ')`, want: "a|b"},
		{expr: `xs:NMTOKENS(' a b ')[1] instance of xs:NMTOKEN`, want: "true"},
		{expr: `xs:IDREFS('x y')[2] instance of xs:IDREF`, want: "true"},
		{expr: `xs:ENTITIES('e1')[1] instance of xs:ENTITY`, want: "true"},
		{expr: `xs:NMTOKENS(())`, want: "()"},
		{expr: `xs:NMTOKENS('')`, err: "FORG0001"},
		{expr: `xs:NMTOKENS('  ')`, err: "FORG0001"},
		{expr: `xs:IDREFS('1a')`, err: "FORG0001"},
		{expr: `xs:NMTOKENS(('a', 'b'))`, err: "XPTY0004"},
	})
}

// TestCoreDispatch covers how a call is resolved: any prefix bound to a
// standard function namespace (XPath 3.1 §3.1.5: a function is identified by
// its expanded QName), partial application (§3.1.5.1), and the unknown-
// function and fn-prefix errors (XPST0017, XPST0081).
func TestCoreDispatch(t *testing.T) {
	ns := testNSMap{
		"xfn": nsFn,
		"m":   nsMath,
		"mp":  nsMap,
		"ar":  nsArray,
		"xsd": "http://www.w3.org/2001/XMLSchema",
	}
	coreRun(t, "", func(c *Context) { c.NS = ns }, []coreCase{
		{expr: `xfn:concat('a', 'b')`, want: "ab"},
		{expr: `xsd:integer('5') + 1`, want: "6"},
		{expr: `m:sqrt(16)`, want: "4"},
		{expr: `mp:size(map{1: 2})`, want: "1"},
		{expr: `ar:size([1, 2])`, want: "2"},
		{expr: `let $f := substring(?, 2) return $f('abc')`, want: "bc"},
		{expr: `concat('a', ?, ?)('b', 'c')`, want: "abc"},
		{expr: `let $f := substring((1 div 0) idiv 1, ?) return 1`, err: "FOAR0001"},
	})
	// A host that predeclares no fn prefix leaves a lexical fn: unbound.
	coreRun(t, "", func(c *Context) { c.NS = testNSMap{}; c.HostPrefixesOnly = true }, []coreCase{
		{expr: `fn:concat('a', 'b')`, err: "XPST0081"},
		{expr: `concat('a', 'b')`, want: "ab"},
	})
}
