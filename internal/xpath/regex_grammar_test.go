package xpath

import "testing"

// TestRxValidateRegexGrammar checks ValidateRegexGrammar against the three
// grammars it implements: XSD 1.0 (Part 2 Appendix F), XSD 1.1 (Appendix G)
// and the XPath grammar of F&O 3.1 §5.6.1 (XSD's plus anchors, reluctant
// quantifiers, back-references and "(?:…)").
func TestRxValidateRegexGrammar(t *testing.T) {
	const (
		y = true
		n = false
	)
	cases := []struct {
		pat                 string
		xsd10, xsd11, xpath bool
	}{
		// Plain atoms, branches and groups.
		{`abc`, y, y, y},
		{`a|`, y, y, y},
		{`|`, y, y, y},
		{`(a|)`, y, y, y},
		{`a()b`, y, y, y},
		{`(a`, n, n, n},
		{`a)`, n, n, n},
		{`a]`, n, n, n},
		// Quantifiers: a quantifier needs an atom and a piece has at most one.
		{`a*`, y, y, y},
		{`a{2}`, y, y, y},
		{`a{2,}`, y, y, y},
		{`a{1,2}`, y, y, y},
		{`a{09,10}`, y, y, y},
		{`a{2,1}`, n, n, n},
		{`a{0010,9}`, n, n, n},
		{`a{,2}`, n, n, n},
		{`a{x}`, n, n, n},
		{`a{1`, n, n, n},
		{`a**`, n, n, n},
		{`a{2}*`, n, n, n},
		{`a*{2}`, n, n, n},
		{`a{1,2}{3}`, n, n, n},
		{`a?+`, n, n, n},
		{`*a`, n, n, n},
		{`+`, n, n, n},
		{`?`, n, n, n},
		{`{2}`, n, n, n},
		{`a|*`, n, n, n},
		{`(*)`, n, n, n},
		// Reluctant quantifiers are F&O-only (§5.6.1.2).
		{`a*?`, n, n, y},
		{`a+?`, n, n, y},
		{`a??`, n, n, y},
		{`a{2}?`, n, n, y},
		{`a{2,3}?`, n, n, y},
		{`a*?*`, n, n, n},
		// Back-references (§5.6.1.4) and "(?:…)" (§5.6.1.3) are F&O-only;
		// no other "(?…)" form exists in any of the grammars.
		{`(a)\1`, n, n, y},
		{`(a)\12`, n, n, y},
		{`(?:a)`, n, n, y},
		{`(?=a)`, n, n, n},
		{`(?i)a`, n, n, n},
		// Escapes: the repertoire is closed.
		{`\n\r\t\\\|\.\?\*\+\(\)\{\}\-\[\]\^`, y, y, y},
		{`\s\S\i\I\c\C\d\D\w\W`, y, y, y},
		{`\$`, n, n, y},
		{`\`, n, n, n},
		{`a\`, n, n, n},
		{`\b`, n, n, n},
		{`\x41`, n, n, n},
		{`\0`, n, n, n},
		{`\p`, n, n, n},
		{`\pL`, n, n, n},
		{`\p{L`, n, n, n},
		{`\p{Lu}`, y, y, y},
		{`\P{Nd}`, y, y, y},
		{`\p{Cn}`, y, y, y},
		{`\p{Xx}`, n, n, n},
		{`\p{L&}`, n, n, n},
		// Block escapes: IsX with a grammatical name; an unknown block is an
		// error in XSD 1.0 only at the grammar level.
		{`\p{IsBasicLatin}`, y, y, y},
		{`\p{IsLatin-1Supplement}`, y, y, y},
		{`\p{Is}`, n, n, n},
		{`\p{Is_A}`, n, n, n},
		{`\p{IsFooBar}`, n, y, y},
		// Character classes.
		{`[abc]`, y, y, y},
		{`[^abc]`, y, y, y},
		{`[a-z]`, y, y, y},
		{`[\t-\r]`, y, y, y},
		{`[\--\[]`, y, y, y},
		{`[a-]`, y, y, y},
		{`[-a]`, y, y, y},
		{`[a-z-]`, y, y, y},
		{`[]`, n, n, n},
		{`[^]`, n, n, n},
		{`a[]]b`, n, n, n},
		{`[[a]`, n, n, n},
		{`[a`, n, n, n},
		{`[a-`, n, n, n},
		{`[\`, n, n, n},
		{`[z-a]`, n, n, n},
		{`[a-\n]`, n, n, n},
		{`[--z]`, n, n, n},
		{`[!--]`, n, n, n},
		{`[a--b]`, n, n, n},
		{`[a-\d]`, n, n, n},
		{`[\d-z]`, n, n, n},
		{`[a-\p{L}]`, n, n, n},
		{`[\p{L]`, n, n, n},
		{`[\p{Lu}\d]`, y, y, y},
		{`[\$-a]`, n, n, y},
		{`[\1]`, n, n, n},
		// Class subtraction: a non-empty base, and the subtraction last.
		{`[a-z-[aeiou]]`, y, y, y},
		{`[a-z-[^b]]`, y, y, y},
		{`[\p{L}-[a]]`, y, y, y},
		{`[-[a]]`, n, n, n},
		{`[a-z-[b]c]`, n, n, n},
		{`[a-z-[b]`, n, n, n},
		{`[a-z-[[b]]]`, n, n, n},
		// XSD 1.0's charClass production rejects a chained range and a bare
		// '-' after a range unless it closes the class or precedes '-[';
		// XSD 1.1's charGroupPart (G.2) reads it as a literal.
		{`[a-z--[b-z]]`, y, y, y},
		{`[a-c-1-4]`, n, y, y},
		{`[0-9-.]`, n, y, y},
	}
	for _, c := range cases {
		for _, g := range []struct {
			name string
			gr   RegexGrammar
			want bool
		}{{"XSD10", GrammarXSD10, c.xsd10}, {"XSD11", GrammarXSD11, c.xsd11}, {"XPath", GrammarXPath, c.xpath}} {
			err := ValidateRegexGrammar(c.pat, g.gr)
			if (err == nil) != g.want {
				t.Errorf("%s %q: err=%v, want valid=%v", g.name, c.pat, err, g.want)
			}
		}
	}
}

// TestRxFlags checks the flags argument (F&O 3.1 §5.6.2), using the spec's own
// examples for 'x' and 'q'.
func TestRxFlags(t *testing.T) {
	rxRun(t, []rxCase{
		// 'x': whitespace is removed from the pattern, except inside a class.
		{`matches('helloworld', 'hello world', 'x')`, "true"},
		{`matches('helloworld', 'hello[ ]world', 'x')`, "false"},
		{`matches('hello world', 'hello\ sworld', 'x')`, "true"},
		{`matches('hello world', 'hello world', 'x')`, "false"},
		{"matches('ab', ' a\tb\n\r ', 'x')", "true"},
		{`matches('a b', 'a[ ]b', 'x')`, "true"},
		{`matches('[a', '\[ a', 'x')`, "true"},
		{`matches('a]', 'a \]', 'x')`, "true"},
		// 'q': every character is literal.
		{`matches('a.b', '.', 'q')`, "true"},
		{`matches('ab', '.', 'q')`, "false"},
		{`matches('a(b', '(', 'q')`, "true"},
		{`replace('a/b/c', '/', '$', 'q')`, "a$b$c"},
		// 's' and 'm'.
		{`matches(codepoints-to-string(10), '.')`, "false"},
		{`matches(codepoints-to-string(13), '.')`, "false"},
		{`matches(codepoints-to-string(10), '.', 's')`, "true"},
		{`matches(concat('a', codepoints-to-string(10), 'b'), '^b$')`, "false"},
		{`matches(concat('a', codepoints-to-string(10), 'b'), '^b$', 'm')`, "true"},
		// 'i': case-variants; \p{Lu} keeps matching upper-case letters only.
		{`matches('ABC', 'abc', 'i')`, "true"},
		{`matches('a', '\p{Lu}', 'i')`, "false"},
		{`matches('A', '\p{Lu}', 'i')`, "true"},
		{`matches('ab', 'AB', 'ims')`, "true"},
	})
	rxRunErr(t, "FORX0001", []string{
		`matches('a', 'a', 'z')`,
		`matches('a', 'a', 'I')`,
		`replace('a', 'a', 'b', 'g')`,
		`tokenize('a', 'a', 'X')`,
	})
}

// TestRxGrammarErrorsThroughFunctions checks that a pattern outside the F&O
// grammar is FORX0002 at the function level (§5.6.1).
func TestRxGrammarErrorsThroughFunctions(t *testing.T) {
	rxRunErr(t, "FORX0002", []string{
		`matches('a', 'a**')`,
		`matches('a', '[a-\d]')`,
		`matches('a', '(?=a)')`,
		`matches('a', '\b')`,
		`matches('a', '[a-z-[b]c]')`,
		`replace('a', '[]', 'x')`,
		`tokenize('a', 'a{2,1}')`,
		`matches('a', '\p{IsBadBlockName}')`,
		`matches('a', '\p{Unknown}')`,
	})
}
