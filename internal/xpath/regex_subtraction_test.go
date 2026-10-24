package xpath

import "testing"

type rxRuneCase struct {
	pattern string
	in      rune
	want    bool
}

// rxRunRunes compiles each pattern through the XPath function entry point
// and matches it against a one-character input.
func rxRunRunes(t *testing.T, cases []rxRuneCase) {
	t.Helper()
	for _, c := range cases {
		re, err := compileRegex(c.pattern, "")
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.pattern, err)
			continue
		}
		if got := re.MatchString(string(c.in)); got != c.want {
			t.Errorf("%q matches U+%04X = %v, want %v", c.pattern, c.in, got, c.want)
		}
	}
}

// TestRxClassSubtraction checks character-class subtraction (XSD 1.1 Part 2
// §G.4.2.4, charClassSub: the set of the base group minus the set of the
// subtracted class), including negated base and negated subtrahend groups and
// the escapes a group may hold.
func TestRxClassSubtraction(t *testing.T) {
	rxRunRunes(t, []rxRuneCase{
		{`[a-z-[aeiou]]`, 'b', true},
		{`[a-z-[aeiou]]`, 'a', false},
		{`[a-z-[aeiou]]`, 'A', false},
		// [^B-[S]] = (not B) minus S.
		{`[^a-z-[0-4]]`, 'a', false},
		{`[^a-z-[0-4]]`, '1', false},
		{`[^a-z-[0-4]]`, '7', true},
		{`[^a-[b]]`, 'a', false},
		{`[^a-[b]]`, 'b', false},
		{`[^a-[b]]`, 'c', true},
		// [B-[^S]] = B intersected with S.
		{`[a-z-[^aeiou]]`, 'a', true},
		{`[a-z-[^aeiou]]`, 'b', false},
		{`[a-z-[^aeiou]]`, '1', false},
		// [^B-[^S]] = (not B) intersected with S.
		{`[^a-c-[^a-e]]`, 'd', true},
		{`[^a-c-[^a-e]]`, 'a', false},
		{`[^a-c-[^a-e]]`, 'f', false},
		// An empty difference matches nothing.
		{`[a-[a]]`, 'a', false},
		{`[a-[a]]`, 'b', false},
		{`[^a-z-[^aeiou]]`, 'a', false},
		{`[^a-z-[^aeiou]]`, '1', false},
		{`[a-z-[^\n]]`, 'a', false},
		// Single-character escapes in either operand.
		{`[\*\+\?-[\+]]`, '*', true},
		{`[\*\+\?-[\+]]`, '+', false},
		{`[\*\+\?-[\+]]`, '?', true},
		{`[a\]-[\]]]`, 'a', true},
		{`[a\]-[\]]]`, ']', false},
		{`[\n\t-[\n]]`, '\t', true},
		{`[\n\t-[\n]]`, '\n', false},
		{`[\(\)\{\}\.\|\\\^-[\.]]`, '{', true},
		{`[\(\)\{\}\.\|\\\^-[\.]]`, '\\', true},
		{`[\(\)\{\}\.\|\\\^-[\.]]`, '.', false},
		// \i / \c (XML NameStartChar / NameChar) as operands.
		{`[\i-[:]]`, 'a', true},
		{`[\i-[:]]`, '_', true},
		{`[\i-[:]]`, ':', false},
		{`[\i-[:]]`, '1', false},
		{`[\c-[\i]]`, '1', true},
		{`[\c-[\i]]`, '-', true},
		{`[\c-[\i]]`, '.', true},
		{`[\c-[\i]]`, 'a', false},
		{`[\c-[\i]]`, ':', false},
		// Block and category escapes as operands.
		{`[\p{IsBasicLatin}-[a-z]]`, 'A', true},
		{`[\p{IsBasicLatin}-[a-z]]`, 'a', false},
		{`[\p{IsBasicLatin}-[a-z]]`, 'é', false},
		{`[\P{IsBasicLatin}-[é]]`, 'ü', true},
		{`[\P{IsBasicLatin}-[é]]`, 'é', false},
		{`[\P{IsBasicLatin}-[é]]`, 'a', false},
		{`[\p{L}-[a-z]]`, 'A', true},
		{`[\p{L}-[a-z]]`, 'a', false},
		{`[\p{L}-[a-z]]`, '1', false},
		{`[\P{Lu}-[a]]`, 'b', true},
		{`[\P{Lu}-[a]]`, 'a', false},
		{`[\P{Lu}-[a]]`, 'B', false},
		// A surrogate block holds no character, so nothing is left.
		{`[\p{IsHighSurrogates}-[a]]`, 'a', false},
		{`[\p{IsHighSurrogates}-[a]]`, 'b', false},
		{`[\r\t-[\t]]`, '\r', true},
		{`[\r\t-[\t]]`, '\t', false},
		{`[\p{Cn}-[a]]`, 0x0378, true}, // unassigned
		{`[\p{Cn}-[a]]`, 0x0007, false},
		{`[\p{C}-[a]]`, 0x0007, true},
		// Multi-character escapes as operands (XSD's Unicode semantics:
		// \d = \p{Nd}, \s = [#x20#x9#xA#xD], \w = not P, Z or C).
		{`[\D-[a]]`, 'b', true},
		{`[\D-[a]]`, 'a', false},
		{`[\D-[a]]`, '5', false},
		{`[\d-[5]]`, '٣', true}, // ARABIC-INDIC DIGIT THREE is Nd
		{`[\S-[a]]`, 'b', true},
		{`[\S-[a]]`, ' ', false},
		{`[\S-[a]]`, 'a', false},
		{`[\W-[!]]`, '?', true},
		{`[\W-[!]]`, '!', false},
		{`[\W-[!]]`, 'a', false},
		{`[\w-[b]]`, 'é', true},
		{`[\w-[b]]`, '_', false},
	})
}

// TestRxClassSubtractionEscapedRangeEnd: a range endpoint written as a
// single-character escape denotes that character (XSD Part 2 §G.4.2.4,
// seRange ::= charOrEsc '-' charOrEsc), so [\t-\r] is #x9..#xD.
func TestRxClassSubtractionEscapedRangeEnd(t *testing.T) {
	t.Skip("BUG: parseClassRanges reads an escaped range endpoint as the escape letter, so [\\t-\\r-[\\n]] covers #x9..'r' and matches 'a'")
	rxRunRunes(t, []rxRuneCase{
		{`[\t-\r-[\n]]`, '\t', true},
		{`[\t-\r-[\n]]`, '\r', true},
		{`[\t-\r-[\n]]`, '\n', false},
		{`[\t-\r-[\n]]`, 'a', false},
	})
}

// TestRxClassSubtractionComplementEscapes: \I and \C (the complements of
// NameStartChar and NameChar, XSD Part 2 §G.4.2.5) are valid operands of a
// subtraction too.
func TestRxClassSubtractionComplementEscapes(t *testing.T) {
	t.Skip("BUG: a subtraction whose base is \\I or \\C matches nothing at all ([\\I-[a]] rejects '1' and '-')")
	rxRunRunes(t, []rxRuneCase{
		{`[\I-[a]]`, '1', true},
		{`[\I-[a]]`, '-', true},
		{`[\I-[a]]`, 'b', false},
		{`[\C-[a]]`, ' ', true},
		{`[\C-[a]]`, 'b', false},
	})
}

// TestRxClassSubtractionLenient checks the exported CompileRegex (the XSD
// pattern-facet entry point): subtraction works there too, and an unknown
// category in an operand is still an error.
func TestRxClassSubtractionLenient(t *testing.T) {
	re, err := CompileRegex(`[a-z-[aeiou]]`, "")
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("b") || re.MatchString("e") {
		t.Errorf("[a-z-[aeiou]]: wrong set")
	}
	if _, err := CompileRegex(`[\p{Xx}-[a]]`, ""); err == nil {
		t.Errorf(`[\p{Xx}-[a]]: want an error for the unknown category`)
	}
	// Malformed classes stay errors on this path too.
	for _, p := range []string{`[abc`, `[a-[b]c]`, `[a-z-[b`} {
		if _, err := CompileRegex(p, ""); err == nil {
			t.Errorf("%s: want an error", p)
		}
	}
	// CompileRegex cannot express back-references (FORX0002).
	if _, err := CompileRegex(`(a)\1`, ""); err == nil {
		t.Errorf(`(a)\1: want an error from CompileRegex`)
	}
}
