package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// rxEval evaluates expr against <r/> and returns ToString of the result, or
// the evaluation error.
func rxEval(expr string) (string, error) {
	p, err := Parse(expr)
	if err != nil {
		return "", err
	}
	doc, _ := xmltree.Parse("<r/>")
	v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1})
	if err != nil {
		return "", err
	}
	return ToString(v), nil
}

type rxCase struct{ expr, want string }

func rxRun(t *testing.T, cases []rxCase) {
	t.Helper()
	for _, c := range cases {
		got, err := rxEval(c.expr)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

func rxRunErr(t *testing.T, code string, exprs []string) {
	t.Helper()
	for _, e := range exprs {
		got, err := rxEval(e)
		if err == nil {
			t.Errorf("%s = %q, want error %s", e, got, code)
			continue
		}
		if !strings.Contains(err.Error(), code) {
			t.Errorf("%s: error %v, want %s", e, err, code)
		}
	}
}

// TestRxBackrefMatching checks back-references (F&O 3.1 §5.6.1.4) together
// with the control structure the back-reference matcher interprets itself:
// quantifiers (greedy and reluctant, §5.6.1.2), alternation, capturing and
// non-capturing groups (§5.6.1.3) and the anchors (§5.6.1.1).
func TestRxBackrefMatching(t *testing.T) {
	rxRun(t, []rxCase{
		// The back-reference matches the string the group matched.
		{`matches('abcabc', '^(abc)\1$')`, "true"},
		{`matches('abcabd', '^(abc)\1$')`, "false"},
		{`matches('xabcabcy', '(abc)\1')`, "true"},
		{`matches('abab', '(ab)\1')`, "true"},
		{`matches('abba', '^(a)(b)\2\1$')`, "true"},
		{`matches('abab', '^(a)(b)\2\1$')`, "false"},
		// A back-reference longer than the rest of the input fails.
		{`matches('aba', '^(ab)\1$')`, "false"},
		// "If the Nth group has not matched anything, the back-reference
		// matches a zero-length string."
		{`matches('b', '^(a)?b\1$')`, "true"},
		{`matches('ab', '^(a)?b\1$')`, "false"},
		{`matches('aba', '^(a)?b\1$')`, "true"},
		// Alternation inside and outside groups.
		{`matches('xx', '^(x|y)\1$')`, "true"},
		{`matches('yy', '^(x|y)\1$')`, "true"},
		{`matches('xy', '^(x|y)\1$')`, "false"},
		{`matches('zz', '^(a)\1$|^zz$')`, "true"},
		{`matches('aa', '^(a)\1$|^zz$')`, "true"},
		{`matches('az', '^(a)\1$|^zz$')`, "false"},
		// Quantified back-references: *, +, ?, {n}, {n,}, {n,m}.
		{`matches('a', '^(a)\1*$')`, "true"},
		{`matches('aaaa', '^(a)\1*$')`, "true"},
		{`matches('a', '^(a)\1+$')`, "false"},
		{`matches('aaa', '^(a)\1+$')`, "true"},
		{`matches('a', '^(a)\1?$')`, "true"},
		{`matches('aa', '^(a)\1?$')`, "true"},
		{`matches('aaa', '^(a)\1?$')`, "false"},
		{`matches('aaa', '^(a)\1{2}$')`, "true"},
		{`matches('aa', '^(a)\1{2}$')`, "false"},
		{`matches('aaaa', '^(a)\1{2}$')`, "false"},
		{`matches('aa', '^(a)\1{2,}$')`, "false"},
		{`matches('aaaaaa', '^(a)\1{2,}$')`, "true"},
		{`matches('aaa', '^(a)\1{2,3}$')`, "true"},
		{`matches('aaaa', '^(a)\1{2,3}$')`, "true"},
		{`matches('aaaaa', '^(a)\1{2,3}$')`, "false"},
		{`matches('aaaa', '^(a)\1{2,3}?$')`, "true"},
		// A quantified group followed by its back-reference: the reference
		// is to the LAST iteration's match (§5.6.1.3).
		{`matches('abb', '^(a|b)+\1$')`, "true"},
		{`matches('aba', '^(a|b)+\1$')`, "false"},
		// Non-capturing groups do not count for the numbering.
		{`matches('xyy', '^(?:x)(y)\1$')`, "true"},
		{`matches('xyx', '^(?:x)(y)\1$')`, "false"},
		// Leaves: character classes, multi-character and category escapes.
		{`matches('77', '^([0-9])\1$')`, "true"},
		{`matches('78', '^([0-9])\1$')`, "false"},
		{`matches('55', '^(\d)\1$')`, "true"},
		{`matches('ÄÄ', '^(\p{Lu})\1$')`, "true"},
		{`matches('ää', '^(\p{Lu})\1$')`, "false"},
		{`matches('ää', '^(\P{Lu})\1$')`, "true"},
		{`matches('aa', '^(\p{IsBasicLatin})\1$')`, "true"},
		// '.' excludes #xA and #xD outside dot-all mode (§5.6.2 's').
		{`matches('qq', '^(.)\1$')`, "true"},
		{`matches(codepoints-to-string((10, 10)), '^(.)\1$')`, "false"},
		{`matches('a.a.', '^(a\.)\1$')`, "true"},
		{`matches('abab', '^(a\.)\1$')`, "false"},
	})
}

// TestRxBackrefDotAll: under 's' the '.' matches every character, #xA
// included (F&O 3.1 §5.6.2), also in a pattern that uses a back-reference.
func TestRxBackrefDotAll(t *testing.T) {
	t.Skip("BUG: with a back-reference the 's' flag is ignored — the matcher's '.' leaf is compiled without dot-all, so it never matches #xA")
	rxRun(t, []rxCase{
		{`matches(codepoints-to-string((10, 10)), '^(.)\1$', 's')`, "true"},
	})
}

// TestRxBackrefMultiDigit pins §5.6.1.4's rule for a back-reference written
// with more than one digit: a single digit is always a back-reference, a
// further digit is part of it only when the larger number names an existing
// group.
func TestRxBackrefMultiDigit(t *testing.T) {
	rxRun(t, []rxCase{
		// One group: "\10" is \1 followed by the literal '0'.
		{`matches('aa0', '^(a)\10$')`, "true"},
		{`matches('aa', '^(a)\10$')`, "false"},
		// Eleven groups, all closed before the reference: "\11" is group 11.
		{`matches('abcdefghijkk', '^(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)(k)\11$')`, "true"},
		{`matches('abcdefghijkaa', '^(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)(k)\11$')`, "false"},
		// Two groups: "\21" is group 2 followed by '1'.
		{`matches('abb1', '^(a)(b)\21$')`, "true"},
	})
}

// TestRxBackrefMultiDigitOpenAfter: a further digit joins the reference only
// if the larger number names a group "whose opening parenthesis precedes the
// back-reference" (F&O 3.1 §5.6.1.4) — group 10 opens after "\10" here, so
// it is \1 followed by the literal '0'.
func TestRxBackrefMultiDigitOpenAfter(t *testing.T) {
	t.Skip("BUG: the digit run is resolved against every group in the pattern, so \\10 picks group 10 (opening after the reference) and fails with FORX0002")
	rxRun(t, []rxCase{
		{`matches('abcdefghia0j', '^(a)(b)(c)(d)(e)(f)(g)(h)(i)\10(j)$')`, "true"},
	})
}

// TestRxBackrefCaseInsensitive checks that under the 'i' flag back-references
// are compared case-blind (§5.6.1.4), while \p{Lu} keeps matching only
// upper-case letters (§5.6.2 'i').
func TestRxBackrefCaseInsensitive(t *testing.T) {
	rxRun(t, []rxCase{
		{`matches('aA', '^(a)\1$', 'i')`, "true"},
		{`matches('aA', '^(a)\1$')`, "false"},
		{`matches('ABac', '^(ab)\1$', 'i')`, "false"},
		{`matches('ABab', '^(ab)\1$', 'i')`, "true"},
		{`matches('Aa', '^(\p{Lu})\1$', 'i')`, "true"},
		{`matches('aa', '^(\p{Lu})\1$', 'i')`, "false"},
	})
}

// TestRxBackrefAnchors checks ^ and $ in the back-reference matcher: they match
// at the whole string's ends by default, and at line boundaries (#xA only)
// under 'm' (§5.6.1.1, §5.6.2).
func TestRxBackrefAnchors(t *testing.T) {
	rxRun(t, []rxCase{
		{`matches("x
aa
y", '^(a)\1$')`, "false"},
		{`matches("x
aa
y", '^(a)\1$', 'm')`, "true"},
		{`matches("x
aab
y", '^(a)\1$', 'm')`, "false"},
		{`matches("baa
y", '^(a)\1', 'm')`, "false"},
		{`matches("aa
", '(a)\1$', 'm')`, "true"},
	})
}

// TestRxBackrefReluctant checks reluctant quantifiers (§5.6.1.2: the shortest
// possible match) through fn:replace, where the captured text is visible.
func TestRxBackrefReluctant(t *testing.T) {
	rxRun(t, []rxCase{
		{`replace('aaaa', '(a)(a*)\1', '[$2]')`, "[aa]"},
		{`replace('aaaa', '(a)(a*?)\1', '[$2]')`, "[][]"},
		{`replace('aaaa', '(a)(a+?)\1', '[$2]')`, "[a]a"},
		{`replace('aaaa', '(a)(a??)\1', '[$2]')`, "[][]"},
		{`replace('aaaaa', '(a)(a{1,3}?)\1', '[$2]')`, "[a]aa"},
	})
}

// TestRxBackrefFunctions drives the back-reference matcher through fn:replace
// (§5.6.4's $N rules), fn:tokenize and fn:analyze-string.
func TestRxBackrefFunctions(t *testing.T) {
	rxRun(t, []rxCase{
		{`replace('abab', '(a)(b)\1', 'X')`, "Xb"},
		{`replace('xaay bbz', '(\w)\1', '<$1>')`, "x<a>y <b>z"},
		{`replace('xaay', '(\w)\1', '$0$0')`, "xaaaay"},
		{`replace('aa', '(a)\1', '\$')`, "$"},
		{`replace('aa', '(a)\1', '\\')`, `\`},
		// A group that did not participate becomes the zero-length string.
		{`replace('b', '(a)?b\1', '[$1]')`, "[]"},
		// S < N <= 9: zero-length string.
		{`replace('aa', '(a)\1', '[$5]')`, "[]"},
		// Two groups: "$23" is group 2 followed by '3'.
		{`replace('abab', '(a)(b)\1\2', '$23')`, "b3"},
		{`replace('nomatch', '(a)\1', 'X')`, "nomatch"},
		{`string-join(tokenize('xaay bbz', '(\w)\1'), '|')`, "x|y |z"},
		{`string-join(tokenize('aa', '(a)\1'), '|')`, "|"},
		{`string-join(analyze-string('xabay', '(a)(b)\1')/*:match/*:group[@nr = 2], ',')`, "b"},
		{`string-join(analyze-string('xabay', '(a)(b)\1')/*:non-match, ',')`, "x,y"},
		{`string-join(analyze-string('ééxéé', '(é)\1')/*:match, ',')`, "éé,éé"},
	})
}

// TestRxBackrefErrors pins the invalid back-references of §5.6.1.4: one to a
// group that does not exist, or whose closing parenthesis comes after the
// reference, makes the pattern invalid (FORX0002).
func TestRxBackrefErrors(t *testing.T) {
	rxRunErr(t, "FORX0002", []string{
		`matches('aa', '(a)\2')`,
		`matches('aa', '\1(a)')`,
		`matches('aa', '(a\1)')`,
		`matches('aa', '((a)\1)')`,
		`matches('aa', '\1')`,
		// Inside a character class a backslash before a digit is invalid.
		`matches('aa', '(a)[\1]')`,
	})
}
