package xpath

import (
	"fmt"
	"strings"
	"testing"
)

const (
	sqUCA  = "http://www.w3.org/2013/collation/UCA"
	sqHTML = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"
	sqCP   = "http://www.w3.org/2005/xpath-functions/collation/codepoint"
	// The UCA tailoring of the F&O 3.1 §5.5 substring examples.
	sqBlank = sqUCA + "?lang=en;alternate=blanked;strength=primary"
)

// sqCmp builds a fn:compare call under the given collation URI.
func sqCmp(a, b, uri string) string {
	return fmt.Sprintf("compare('%s', '%s', '%s')", a, b, uri)
}

// F&O 3.1 §5.3.3: the UCA collation family and its parameters. Only orderings
// that UCA (UTS #10) / CLDR fix unambiguously are asserted.
func TestCollationUCAParameters(t *testing.T) {
	sqRun(t, "", []sqCase{
		// strength (§5.3.3 table: primary A=a=Ä=ä; secondary A=a, Ä=ä, ä≠a;
		// tertiary A≠a≠Ä≠ä).
		{sqCmp("a", "A", sqUCA+"?strength=primary"), "0"},
		{sqCmp("a", "ä", sqUCA+"?strength=primary"), "0"},
		{sqCmp("A", "ä", sqUCA+"?strength=1"), "0"},
		{sqCmp("a", "A", sqUCA+"?strength=secondary"), "0"},
		{sqCmp("Ä", "ä", sqUCA+"?strength=2"), "0"},
		{sqCmp("a", "ä", sqUCA+"?strength=secondary"), "-1"},
		{sqCmp("a", "A", sqUCA+"?strength=tertiary"), "-1"},
		{sqCmp("a", "A", sqUCA+"?strength=3"), "-1"},
		{sqCmp("a", "A", sqUCA), "-1"},
		{sqCmp("a", "b", sqUCA), "-1"},
		{sqCmp("b", "a", sqUCA), "1"},
		{sqCmp("abc", "abc", sqUCA), "0"},
		// The REC's fn:compare example (§5.3.6).
		{`compare('Strasse', 'Straße', 'http://www.w3.org/2013/collation/UCA?lang=de;strength=primary')`, "0"},
		// If two parameters use the same keyword the last one wins.
		{sqCmp("a", "A", sqUCA+"?strength=primary;strength=tertiary"), "-1"},
		{sqCmp("a", "A", sqUCA+"?strength=tertiary;strength=primary"), "0"},

		// caseFirst=upper|lower.
		{sqCmp("A", "a", sqUCA+"?caseFirst=upper"), "-1"},
		{sqCmp("a", "A", sqUCA+"?caseFirst=upper"), "1"},
		{sqCmp("a", "A", sqUCA+"?caseFirst=lower"), "-1"},
		{sqCmp("Ab", "ab", sqUCA+"?caseFirst=upper"), "-1"},
		// case is a tertiary difference: a primary difference still wins.
		{sqCmp("a", "B", sqUCA+"?caseFirst=upper"), "-1"},
		{sqCmp("A", "a", sqUCA+"?caseFirst=upper;strength=secondary"), "0"},
		{sqCmp("Ä", "a", sqUCA+"?caseFirst=upper;strength=secondary"), "1"},
		// caseLevel=yes with primary strength: accents ignored, case kept.
		{sqCmp("a", "A", sqUCA+"?strength=primary;caseLevel=yes"), "-1"},
		{sqCmp("a", "á", sqUCA+"?strength=primary;caseLevel=yes"), "0"},
		{sqCmp("a", "a", sqUCA+"?strength=primary;caseLevel=yes"), "0"},

		// numeric=yes: chap2 sorts before chap12.
		{sqCmp("chap2", "chap12", sqUCA+"?numeric=yes"), "-1"},
		{sqCmp("chap2", "chap12", sqUCA+"?numeric=no"), "1"},
		{sqCmp("chap2", "chap12", sqUCA), "1"},
		{sqCmp("Chap2", "chap12", sqUCA+"?numeric=yes;caseFirst=upper"), "-1"},

		// backwards=yes: the last accent is the most significant (the classic
		// French ordering cote < côte < coté < côté).
		{sqCmp("côte", "coté", sqUCA+"?backwards=yes"), "-1"},
		{sqCmp("côte", "coté", sqUCA+"?backwards=no"), "1"},
		{sqCmp("côte", "coté", sqUCA), "1"},
		{sqCmp("cote", "côte", sqUCA+"?backwards=yes"), "-1"},
		{sqCmp("coté", "côté", sqUCA+"?backwards=yes"), "-1"},
		// CLDR's fr-CA tailoring orders accents backwards by default.
		{sqCmp("côte", "coté", sqUCA+"?lang=fr-CA"), "-1"},
		{sqCmp("côte", "coté", sqUCA+"?lang=fr-CA;backwards=no"), "1"},
		{sqCmp("côte", "coté", sqUCA+"?lang=fr"), "1"},

		// lang: Swedish sorts ä after z; German (and the root) before.
		{sqCmp("ä", "z", sqUCA+"?lang=sv"), "1"},
		{sqCmp("ä", "z", sqUCA+"?lang=de"), "-1"},
		{sqCmp("ä", "z", sqUCA), "-1"},

		// alternate: non-ignorable (data base sorts before datatype), shifted
		// (the variable characters only count at the quaternary level), blanked
		// (only at the identical level).
		{sqCmp("data base", "datatype", sqUCA+"?alternate=non-ignorable"), "-1"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=non-ignorable"), "-1"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=shifted"), "0"},
		{sqCmp("data base", "database", sqUCA+"?alternate=shifted"), "0"},
		{sqCmp("data-base", "Database", sqUCA+"?alternate=shifted"), "-1"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=shifted;strength=quaternary"), "-1"},
		{sqCmp("database", "data-base", sqUCA+"?alternate=shifted;strength=4"), "1"},
		{sqCmp("data base", "data-base", sqUCA+"?alternate=shifted;strength=quaternary"), "-1"},
		{sqCmp("data-base", "data-base", sqUCA+"?alternate=shifted;strength=quaternary"), "0"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=blanked;strength=quaternary"), "0"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=blanked;strength=identical"), "-1"},
		{sqCmp("data-base", "data-base", sqUCA+"?alternate=blanked;strength=5"), "0"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=blanked"), "0"},
		// maxVariable selects the variable groups: space < punct < symbol <
		// currency, each including the earlier ones.
		{sqCmp("data base", "database", sqUCA+"?alternate=shifted;maxVariable=space"), "0"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=shifted;maxVariable=space"), "-1"},
		{sqCmp("data-base", "database", sqUCA+"?alternate=shifted;maxVariable=punct"), "0"},
		{sqCmp("a+b", "ab", sqUCA+"?alternate=shifted;maxVariable=symbol"), "0"},
		{sqCmp("a b", "ab", sqUCA+"?alternate=shifted;maxVariable=symbol"), "0"},
		{sqCmp("a$b", "ab", sqUCA+"?alternate=shifted;maxVariable=currency"), "0"},
		{sqCmp("a+b", "ab", sqUCA+"?alternate=shifted;maxVariable=currency"), "0"},
		{sqCmp("a+b", "ab", sqUCA+"?alternate=shifted;maxVariable=punct"), "-1"},

		// fallback: unknown keywords/values are ignored unless fallback=no.
		{sqCmp("a", "A", sqUCA+"?fallback=yes;strength=primary"), "0"},
		{sqCmp("a", "A", sqUCA+"?foo=bar;strength=primary"), "0"},
		{sqCmp("a", "A", sqUCA+"?strength=bogus"), "-1"},
		{sqCmp("a", "b", sqUCA+"?version=6.2.0"), "-1"},
		{sqCmp("a", "b", sqUCA+"?version=96.5"), "-1"},
		{sqCmp("a", "b", sqUCA+"?fallback=no;version=6.2.0"), "-1"},
		{sqCmp("a", "b", sqUCA+"?reorder=Grek,Latn"), "-1"},
		{sqCmp("a", "b", sqUCA+"?"), "-1"},
	})
	sqRunErr(t, "", []sqErrCase{
		// §5.3.5: an unsupported collation is FOCH0002.
		{sqCmp("a", "b", "http://example.com/no-such-collation"), "FOCH0002"},
		{sqCmp("a", "b", sqUCA+"x?strength=primary"), "FOCH0002"},
		// §5.3.3: with fallback=no an unrecognized keyword or value rejects the
		// collation as unsupported (same error as an unrecognized URI).
		{sqCmp("a", "b", sqUCA+"?fallback=no;foo=bar"), "FOCH0002"},
		{sqCmp("a", "b", sqUCA+"?fallback=no;strength=bogus"), "FOCH0002"},
		{sqCmp("a", "b", sqUCA+"?fallback=no;lang=not!a!language"), "FOCH0002"},
		// fallback=no must use a conformant collation or fail; UCA 96.5 does
		// not exist.
		{sqCmp("a", "b", sqUCA+"?fallback=no;version=96.5"), "FOCH0002"},
	})
}

// F&O 3.1 §5.3.4 html-ascii-case-insensitive and the codepoint collation.
func TestCollationHTMLAndCodepoint(t *testing.T) {
	sqRun(t, "", []sqCase{
		{sqCmp("ABC", "abc", sqHTML), "0"},
		{sqCmp("aBc", "AbC", sqHTML), "0"},
		// only A-Z / a-z match each other.
		{sqCmp("Á", "á", sqHTML) + " = 0", "false"},
		{sqCmp("a", "b", sqHTML) + " = 0", "false"},
		{sqCmp("ABC", "abc", sqCP), "-1"},
		// The REC's fn:contains-token example (§5.4.?).
		{`contains-token("red green blue", "RED", "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive")`, "true"},
		{`contains("ABCDEF", "cde", '` + sqHTML + `')`, "true"},
		{`contains("ABCDEF", "xyz", '` + sqHTML + `')`, "false"},
		{`starts-with("ABCDEF", "abc", '` + sqHTML + `')`, "true"},
		{`starts-with("ABCDEF", "bcd", '` + sqHTML + `')`, "false"},
		{`ends-with("ABCDEF", "def", '` + sqHTML + `')`, "true"},
		{`ends-with("ABCDEF", "cde", '` + sqHTML + `')`, "false"},
		{`substring-before("ABCDEF", "cd", '` + sqHTML + `')`, "AB"},
		{`substring-after("ABCDEF", "cd", '` + sqHTML + `')`, "EF"},
		{`substring-after("ABCDEF", "x", '` + sqHTML + `')`, ""},
		{`contains("ABC", "B", '` + sqCP + `')`, "true"},
		{`contains("ABC", "b", '` + sqCP + `')`, "false"},
		{`starts-with("ABC", "A", '` + sqCP + `')`, "true"},
		{`ends-with("ABC", "C", '` + sqCP + `')`, "true"},
	})
	sqRunErr(t, "", []sqErrCase{
		{`contains("a", "a", "http://example.com/no-such-collation")`, "FOCH0002"},
		{`starts-with("a", "a", "http://example.com/no-such-collation")`, "FOCH0002"},
		{`contains("a", "a", ("a", "b"))`, "XPTY0004"},
		{`contains("a", "a", ())`, "XPTY0004"},
	})
}

// F&O 3.1 §5.5 examples: substring matching on collation units, where
// characters the collation ignores neither match nor obstruct.
func TestCollationSubstringExamples(t *testing.T) {
	sqRun(t, "", []sqCase{
		// §5.5.1 fn:contains.
		{`contains("abcdefghi", "-d-e-f-", '` + sqBlank + `')`, "true"},
		{`contains("a*b*c*d*e*f*g*h*i*", "d-ef-", '` + sqBlank + `')`, "true"},
		{`contains("abcd***e---f*--*ghi", "def", '` + sqBlank + `')`, "true"},
		{`contains((), "--***-*---", '` + sqBlank + `')`, "true"},
		{`contains("abcdefghi", "xyz", '` + sqBlank + `')`, "false"},
		// §5.5.2 fn:starts-with.
		{`starts-with("abcdefghi", "-a-b-c-", '` + sqBlank + `')`, "true"},
		{`starts-with("a*b*c*d*e*f*g*h*i*", "a-bc-", '` + sqBlank + `')`, "true"},
		{`starts-with("abcd***e---f*--*ghi", "abcdef", '` + sqBlank + `')`, "true"},
		{`starts-with((), "--***-*---", '` + sqBlank + `')`, "true"},
		{`starts-with("-abcdefghi", "-abc", '` + sqBlank + `')`, "true"},
		{`starts-with("abcdefghi", "bcd", '` + sqBlank + `')`, "false"},
		{`starts-with("abc", "abc", '` + sqBlank + `')`, "true"},
		// §5.5.3 fn:ends-with.
		{`ends-with("abcdefghi", "-g-h-i-", '` + sqBlank + `')`, "true"},
		{`ends-with("abcd***e---f*--*ghi", "defghi", '` + sqBlank + `')`, "true"},
		{`ends-with((), "--***-*---", '` + sqBlank + `')`, "true"},
		{`ends-with("abcdefghi", "ghi-", '` + sqBlank + `')`, "true"},
		{`ends-with("abcdefghi", "fgh", '` + sqBlank + `')`, "false"},
		// §5.5.4 fn:substring-before.
		{`substring-before("abcdefghi", "--d-e-", '` + sqBlank + `')`, "abc"},
		{`substring-before("abc--d-e-fghi", "--d-e-", '` + sqBlank + `')`, "abc--"},
		{`substring-before("a*b*c*d*e*f*g*h*i*", "***cde", '` + sqBlank + `')`, "a*b*"},
		{`substring-before("Eureka!", "--***-*---", '` + sqBlank + `')`, ""},
		{`substring-before("abc", "x", '` + sqBlank + `')`, ""},
		// §5.5.5 fn:substring-after.
		{`substring-after("abcdefghi", "--d-e-", '` + sqBlank + `')`, "fghi"},
		{`substring-after("abc--d-e-fghi", "--d-e-", '` + sqBlank + `')`, "-fghi"},
		{`substring-after("a*b*c*d*e*f*g*h*i*", "***cde***", '` + sqBlank + `')`, "*f*g*h*i*"},
		{`substring-after("Eureka!", "--***-*---", '` + sqBlank + `')`, "Eureka!"},
		// Case-insensitive matching under a primary-strength UCA collation.
		{`contains("Hello World", "WORLD", '` + sqUCA + `?strength=primary')`, "true"},
		{`starts-with("Hello", "hE", '` + sqUCA + `?strength=primary')`, "true"},
		{`ends-with("Hello", "LO", '` + sqUCA + `?strength=primary')`, "true"},
		{`ends-with("Hello", "LL", '` + sqUCA + `?strength=primary')`, "false"},
		// A whole digit run matches under numeric=yes.
		{`contains("see Chapter-10 now", "Chapter-10", '` + sqUCA + `?numeric=yes')`, "true"},
		{`starts-with("10 apples", "10", '` + sqUCA + `?numeric=yes')`, "true"},
		{`ends-with("page 12", "12", '` + sqUCA + `?numeric=yes')`, "true"},
	})
}

// F&O 3.1 §5.5 leaves a match that splits a number under numeric=yes
// undefined; an implementation may report no match or raise FOCH0004 — it must
// not report a match, since "10" and "100" are different numbers.
func TestCollationNumericPartialMatch(t *testing.T) {
	for _, expr := range []string{
		`contains("Chapter-100", "Chapter-10", '` + sqUCA + `?numeric=yes')`,
		`starts-with("100 apples", "10", '` + sqUCA + `?numeric=yes')`,
		`ends-with("page 112", "12", '` + sqUCA + `?numeric=yes')`,
	} {
		v, err := sqEval(t, expr, "", nil)
		if err != nil {
			if !strings.Contains(err.Error(), "FOCH0004") {
				t.Errorf("%s: unexpected error %v", expr, err)
			}
			continue
		}
		if ToString(v) != "false" {
			t.Errorf("%s = %s, want false (or FOCH0004)", expr, ToString(v))
		}
	}
}

// A relative collation URI is resolved against the static base URI (F&O 3.1
// §5.3.5).
func TestCollationRelativeURI(t *testing.T) {
	p, err := Parse(`substring-before("abcABC", "B", "collation/html-ascii-case-insensitive")`)
	if err != nil {
		t.Fatal(err)
	}
	v, err := p.Eval(&Context{BaseURI: "http://www.w3.org/2005/xpath-functions/", Pos: 1, Size: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := ToString(v); got != "a" {
		t.Errorf("got %q, want %q", got, "a")
	}
}

// With no $collation argument the default collation from the context applies
// (F&O 3.1 §5.3.5 rule 2).
func TestCollationDefaultFromContext(t *testing.T) {
	coll, err := ResolveCollator(sqUCA + "?strength=primary")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []sqCase{
		{`contains("ABC", "b")`, "true"},
		{`starts-with("ABC", "ab")`, "true"},
		{`ends-with("ABC", "bc")`, "true"},
		{`substring-before("ABC", "b")`, "A"},
		{`substring-after("ABC", "b")`, "C"},
		{`count(distinct-values(("a", "A")))`, "1"},
		{`count(index-of(("a", "A"), "a"))`, "2"},
		{`deep-equal("a", "A")`, "true"},
		{`max(("b", "A")) = "b"`, "true"},
		{`compare("a", "A")`, "0"},
	} {
		v, err := sqEval(t, c.expr, "", coll)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got := ToString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

// F&O 3.1 §5.3.6? fn:collation-key: collation-key($A, $C) eq
// collation-key($B, $C) iff compare($A, $B, $C) eq 0, and lt iff compare is
// -1. Checked pairwise over a set of strings under several collations.
func TestCollationKeyMirrorsCompare(t *testing.T) {
	words := []string{"a", "A", "b", "B", "ä", "Ä", "ab", "Ab", "chap2", "chap12", "data-base", "database", ""}
	colls := []string{
		"",
		sqCP,
		sqHTML,
		sqUCA,
		sqUCA + "?strength=primary",
		sqUCA + "?strength=secondary",
		sqUCA + "?lang=sv",
		sqUCA + "?numeric=yes",
		sqUCA + "?caseFirst=upper",
		sqUCA + "?lang=de;strength=1",
	}
	for _, c := range colls {
		for _, a := range words {
			for _, b := range words {
				var collArg string
				if c != "" {
					collArg = ", '" + c + "'"
				}
				expr := fmt.Sprintf(`let $ka := collation-key('%s'%s), $kb := collation-key('%s'%s), $c := compare('%s', '%s'%s)
					return ($ka eq $kb) = ($c eq 0) and ($ka lt $kb) = ($c eq -1)`, a, collArg, b, collArg, a, b, collArg)
				v, err := sqEval(t, expr, "", nil)
				if err != nil {
					t.Errorf("collation %q, %q vs %q: %v", c, a, b, err)
					continue
				}
				if ToString(v) != "true" {
					t.Errorf("collation %q: collation-key order of %q vs %q disagrees with compare", c, a, b)
				}
			}
		}
	}
	sqRun(t, "", []sqCase{
		// The REC examples, $C := UCA?strength=primary.
		{`let $C := 'http://www.w3.org/2013/collation/UCA?strength=primary' return map:merge((map{collation-key("A", $C):1}, map{collation-key("a", $C):2}), map{"duplicates":"use-last"})(collation-key("A", $C))`, "2"},
		{`let $C := 'http://www.w3.org/2013/collation/UCA?strength=primary' return let $M := map{collation-key("A", $C):1, collation-key("B", $C):2} return $M(collation-key("a", $C))`, "1"},
		{`collation-key("abc") instance of xs:base64Binary`, "true"},
	})
	sqRunErr(t, "", []sqErrCase{
		{`collation-key("a", "http://example.com/no-such-collation")`, "FOCH0002"},
		{`collation-key("a", "` + sqUCA + `?fallback=no;foo=bar")`, "FOCH0002"},
	})
}

// The exported host entry points (used by xsl:sort/xsl:key/for-each-group).
func TestCollationHostAPI(t *testing.T) {
	if _, err := ResolveCollator("http://example.com/x"); err == nil || !strings.Contains(err.Error(), "FOCH0002") {
		t.Errorf("ResolveCollator(unknown) err = %v, want FOCH0002", err)
	}
	if _, err := ResolveCollationKeyer("http://example.com/x"); err == nil || !strings.Contains(err.Error(), "FOCH0002") {
		t.Errorf("ResolveCollationKeyer(unknown) err = %v, want FOCH0002", err)
	}
	cp, err := ResolveCollator("")
	if err != nil || cp("a", "B") <= 0 {
		t.Errorf("ResolveCollator(\"\") must be codepoint order")
	}
	k, err := ResolveCollationKeyer(sqHTML)
	if err != nil || string(k("ABC")) != string(k("abc")) || string(k("ABC")) == string(k("abd")) {
		t.Errorf("html-ascii-case-insensitive keys must fold ASCII case only")
	}
	if CollatorForLang("", "") != nil || CollatorForLang("  ", "upper-first") != nil {
		t.Errorf("CollatorForLang without a language must be nil")
	}
	sv := CollatorForLang("sv", "")
	if sv == nil || sv("ä", "z") <= 0 {
		t.Errorf("CollatorForLang(sv) must sort ä after z")
	}
	for _, order := range []string{"upper-first", "lower-first"} {
		c := CollatorForLang("en", order)
		if c == nil || c("a", "b") >= 0 || c("b", "a") <= 0 || c("x", "x") != 0 {
			t.Errorf("CollatorForLang(en, %s) must keep the collation's primary order", order)
		}
	}
}
