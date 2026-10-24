package xsd

import (
	"math/rand"
	"testing"
)

// TestLexScannersMatchRegexps holds the hand-written recognizers in
// lexscan.go to the regexps they replace: every string over an alphabet that
// exercises each branch (signs, digits, points, name characters, a few
// non-ASCII and invalid bytes) up to length 4 exhaustively, then random longer
// ones.
func TestLexScannersMatchRegexps(t *testing.T) {
	alphabet := []string{"", "+", "-", "0", "7", "9", ".", "a", "Z", "_", ":", " ", "\n", "e", "·", "é", "̀", "\xff", "x1"}
	check := func(v string) {
		t.Helper()
		if got, want := isIntegerLex(v), reInteger.MatchString(v); got != want {
			t.Errorf("isIntegerLex(%q) = %v, regexp says %v", v, got, want)
		}
		if got, want := isDecimalLex(v), reDecimal.MatchString(v); got != want {
			t.Errorf("isDecimalLex(%q) = %v, regexp says %v", v, got, want)
		}
		if got, want := isNCNameLex(v), reNCName.MatchString(v); got != want {
			t.Errorf("isNCNameLex(%q) = %v, regexp says %v", v, got, want)
		}
	}
	var gen func(prefix string, n int)
	gen = func(prefix string, n int) {
		check(prefix)
		if n == 0 {
			return
		}
		for _, a := range alphabet[1:] {
			gen(prefix+a, n-1)
		}
	}
	gen("", 4)
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200000; i++ {
		n := rng.Intn(12)
		s := ""
		for j := 0; j < n; j++ {
			s += alphabet[1+rng.Intn(len(alphabet)-1)]
		}
		check(s)
	}
}
