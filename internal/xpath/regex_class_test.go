package xpath

import "testing"

// TestClassSubtractionAndCategoryC pins the XSD regex rules this file was
// wrong about: subtraction whose operands are category escapes was left to RE2
// (which read "-[" as literals and silently matched the wrong thing), and \p{C}
// / \w / \W ignored the unassigned code points (XSD's C includes Cn).
func TestClassSubtractionAndCategoryC(t *testing.T) {
	for _, c := range []struct {
		pattern, input string
		want           bool
	}{
		{`[a-z-[\P{Ll}]]`, "a", true},
		{`[a-z-[\p{Lu}]]`, "b", true},
		{`[A-Z-[\p{Lu}]]`, "B", false},
		{`[\p{Zl}-[x]]`, " ", true},
		{`[\w-[a]]`, "a", false},
		{`[\w-[a]]`, "b", true},
		{`[\d-[8]]`, "7", true},
		{`[\d-[8]]`, "8", false},
		{`[\s-[\t]]`, " ", true},
		{`[\s-[\t]]`, "\t", false},
		{`\p{C}`, "͸", true}, // unassigned
		{`\p{Cn}`, "͸", true},
		{`\P{C}`, "͸", false},
		{`[\p{C}]`, "͸", true},
		{`\w`, "͸", false},
		{`\W`, "͸", true},
		{`\w`, "a", true},
		{`\p{C}`, "\u0007", true}, // Cc
		{`\p{C}`, "a", false},
	} {
		re, err := CompileRegex(c.pattern, "")
		if err != nil {
			t.Errorf("%s: %v", c.pattern, err)
			continue
		}
		if got := re.MatchString(c.input); got != c.want {
			t.Errorf("%q matches %q = %v, want %v", c.pattern, c.input, got, c.want)
		}
	}
}
