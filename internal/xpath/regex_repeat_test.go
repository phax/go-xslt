package xpath

import "testing"

// XSD/XPath regular expressions allow any {n,m} count; RE2 stops at 1000.
func TestRegexRepeatAboveRE2Limit(t *testing.T) {
	cases := map[string]string{
		`matches(string-join((1 to 2000) ! 'a'), '^a{2000}$')`:                "true",
		`matches(string-join((1 to 1999) ! 'a'), '^a{2000}$')`:                "false",
		`matches(string-join((1 to 2500) ! 'a'), '^a{1500,3000}$')`:           "true",
		`matches(string-join((1 to 1200) ! 'é'), '^\p{Ll}{1200}$')`:           "true",
		`replace(string-join((1 to 1001) ! 'b') || 'c', 'b{1001}', 'x')`:      "xc",
		`count(tokenize(string-join((1 to 1001) ! 'b') || ',z', 'b{1001},'))`: "2",
		`matches('aaa', 'a{2147483647}')`:                                     "false",
	}
	for expr, want := range cases {
		got, err := evalString(expr, true)
		if err != nil || got != want {
			t.Errorf("%s: got %q %v, want %q", expr, got, err, want)
		}
	}
}
