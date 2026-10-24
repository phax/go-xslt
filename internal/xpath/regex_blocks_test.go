package xpath

import "testing"

// TestRxBlockEscapes checks the \p{IsX} / \P{IsX} block escapes (XSD 1.1 Part
// 2 §G.4.2.3), inside and outside a character class: current Unicode block
// names, the superseded Unicode 3.1 names §G.4.2.3 asks processors to keep
// (Greek, CombiningMarksforSymbols, and PrivateUse with its three ranges),
// and the surrogate blocks, which contain no character at all.
func TestRxBlockEscapes(t *testing.T) {
	rxRunRunes(t, []rxRuneCase{
		{`\p{IsBasicLatin}`, 'a', true},
		{`\p{IsBasicLatin}`, 'é', false},
		{`\P{IsBasicLatin}`, 'é', true},
		{`\P{IsBasicLatin}`, 'a', false},
		{`[\p{IsBasicLatin}]`, 'a', true},
		{`[\P{IsBasicLatin}]`, 'é', true},
		{`[\P{IsBasicLatin}]`, 'a', false},
		{`[\P{IsBasicLatin}a]`, 'a', true},
		{`[\p{IsLatin-1Supplement}x]`, 'é', true},
		{`[\p{IsLatin-1Supplement}x]`, 'x', true},
		{`[\p{IsLatin-1Supplement}x]`, 'y', false},
		// A block that is newer than Unicode 3.1 (Adlam: U+1E900..U+1E95F).
		{`\p{IsAdlam}`, 0x1E900, true},
		{`\p{IsAdlam}`, 0x1E95F, true},
		{`\p{IsAdlam}`, 0x1E8FF, false},
		// Superseded names keep their Unicode 3.1 ranges.
		{`\p{IsGreek}`, 'α', true},
		{`\p{IsGreek}`, 0x03FF, true},
		{`\p{IsGreek}`, 'a', false},
		{`\p{IsCombiningMarksforSymbols}`, 0x20D0, true},
		{`\p{IsCombiningMarksforSymbols}`, 0x20FF, true},
		{`\p{IsCombiningMarksforSymbols}`, 0x2100, false},
		{`\p{IsPrivateUse}`, 0xE000, true},
		{`\p{IsPrivateUse}`, 0xF8FF, true},
		{`\p{IsPrivateUse}`, 0xF0000, true},
		{`\p{IsPrivateUse}`, 0x100000, true},
		{`\p{IsPrivateUse}`, 0xF900, false},
		{`\P{IsPrivateUse}`, 0xF0000, false},
		{`\P{IsPrivateUse}`, 'a', true},
		// Surrogate blocks hold no scalar value: \p matches nothing, \P
		// everything.
		{`\p{IsHighSurrogates}`, 'a', false},
		{`\p{IsHighSurrogates}`, 0x10000, false},
		{`\P{IsHighSurrogates}`, 'a', true},
		{`\P{IsLowSurrogates}`, 0x10FFFF, true},
		{`[\p{IsHighSurrogates}a]`, 'a', true},
		{`[\p{IsHighSurrogates}a]`, 'b', false},
		{`[\P{IsHighPrivateUseSurrogates}]`, 'a', true},
	})
	// Under 'i' a block escape is unaffected (F&O 3.1 §5.6.2 'i' changes
	// only characters and ranges); BasicLatin contains both cases anyway.
	re, err := compileRegex(`^\p{IsBasicLatin}+$`, "i")
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("aZ") || re.MatchString("é") {
		t.Errorf(`^\p{IsBasicLatin}+$ (i): wrong set`)
	}
}

// TestRxXSDBlockKnown checks the exported block-name lookup the XSD 1.0
// pattern-facet validator uses to reject unknown block names.
func TestRxXSDBlockKnown(t *testing.T) {
	for _, c := range []struct {
		name string
		want bool
	}{
		{"BasicLatin", true},
		{"Latin-1Supplement", true},
		{"Adlam", true},
		{"Greek", true},
		{"PrivateUse", true},
		{"HighSurrogates", true},
		{"NoSuchBlock", false},
		{"", false},
	} {
		if got := XSDBlockKnown(c.name); got != c.want {
			t.Errorf("XSDBlockKnown(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
