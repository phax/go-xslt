package xpath

import "testing"

// TestPeUsesVariable checks Parsed.UsesVariable's contract: true exactly when
// the expression references $v outside any binder (for/let/some/every clause
// or inline-function parameter) that rebinds the same name. Per §3.12–§3.13,
// a clause's binding sequence is in the scope of the PREVIOUS clauses only,
// and the name is in scope for the later clauses and the body.
func TestPeUsesVariable(t *testing.T) {
	cases := []struct {
		expr string
		want bool
	}{
		{"$v", true},
		{"$vv", false},
		{"1", false},
		{"f($v)", true},
		{"f(1)", false},
		{"-$v", true},
		{"$v and 1", true},
		{"1 or $v", true},
		{"$v + 1", true},
		{"$v | a", true},
		{"a[$v]", true},
		{"a/b[$v]", true},
		{"a/b[1]", false},
		{"$v/a", true},
		{"a/$v", true},
		{"$v[1]", true},
		{"(1)[$v]", true},
		{"(1, $v)", true},
		{"1 to $v", true},
		{"$v to 1", true},
		{"for $v in 1 return $v", false},
		{"for $x in $v return 1", true},
		{"for $v in $v return 1", true},
		{"for $x in 1, $v in 2 return $v", false},
		{"for $x in 1, $y in $v return 1", true},
		{"for $x in 1 return $v", true},
		{"let $v := $v return 1", true},
		{"let $v := 1 return $v", false},
		{"let $x := 1, $v := $x return $v", false},
		{"let $x := 1 return $v", true},
		{"some $v in 1 satisfies $v", false},
		{"some $x in $v satisfies 1", true},
		{"every $x in 1 satisfies $v", true},
		{"every $v in 1, $x in $v satisfies 1", false},
		{"if ($v) then 1 else 2", true},
		{"if (1) then 2 else $v", true},
		{"if (1) then 2 else 3", false},
		{"map{1: $v}", true},
		{"map{$v : 1}", true},
		{"map{1: 2}", false},
		{"[$v]", true},
		{"array{$v}", true},
		{"[1]", false},
		{"function($v) {$v}", false},
		{"function($x) {$v}", true},
		{"function($x) {function($v) {$v}}", false},
		{"$m?($v)", true},
		{"$v?a", true},
		{"$m?a", false},
		{"$f($v)", true},
		{"$v(1)", true},
		{"$f(1)", false},
		{"$v eq 1", true},
		{"1 = $v", true},
		{"$v || 'a'", true},
		{"a intersect $v", true},
		{"$v except a", true},
		{"$v instance of xs:integer", true},
		{"$v treat as item()", true},
		{"$v cast as xs:string", true},
		{"$v castable as xs:string", true},
		{"$v => f()", true},
		{"1 => f($v)", true},
		{"1 => $v()", true},
		{"1 => f()", false},
		{"$v ! 1", true},
		{"1 ! $v", true},
		// A prefixed name is a different variable.
		{"$p:v", false},
	}
	for _, c := range cases {
		p, err := Parse(c.expr)
		if err != nil {
			t.Fatalf("parse %q: %v", c.expr, err)
		}
		if got := p.UsesVariable("", "v"); got != c.want {
			t.Errorf("UsesVariable(%q, v) = %v, want %v", c.expr, got, c.want)
		}
	}
	p, _ := Parse("$p:v")
	if !p.UsesVariable("p", "v") {
		t.Error("UsesVariable($p:v, p:v) = false")
	}
	var nilParsed *Parsed
	if nilParsed.UsesVariable("", "v") {
		t.Error("nil *Parsed uses a variable")
	}
}
