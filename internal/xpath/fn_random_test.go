package xpath

import "testing"

// fn:random-number-generator (F&O 3.1 §4.9.1). The numbers themselves are
// implementation-dependent, so these tests check the rules that ARE specified:
// the map's entries and their types, number in [0,1), determinism for equal
// arguments within one execution scope, next() yielding another generator,
// permute returning a permutation of its argument.
func TestRandomNumberGeneratorSpec(t *testing.T) {
	numRun(t, []numCase{
		{"random-number-generator() instance of map(xs:string, item())", "true"},
		{"random-number-generator(42) instance of map(xs:string, item())", "true"},
		{"every $k in ('number', 'next', 'permute') satisfies map:contains(random-number-generator(1), $k)", "true"},
		{"random-number-generator(1)?number instance of xs:double", "true"},
		{"random-number-generator(1)?next instance of function() as map(*)", "true"},
		{"function-arity(random-number-generator(1)?next)", "0"},
		{"function-arity(random-number-generator(1)?permute)", "1"},
		// number is in [0, 1), for many successive generators and seed types.
		{`every $s in (0, 1, 42, -7, 'seed', xs:date('2020-01-01'), 3.5e0, true())
		    satisfies (let $g := random-number-generator($s) return $g?number ge 0 and $g?number lt 1)`, "true"},
		{`let $nums := fold-left(1 to 200, [random-number-generator(9)],
		      function($acc, $i) { array:append($acc, $acc(array:size($acc))?next()) })
		  return every $g in $nums?* satisfies ($g?number ge 0e0 and $g?number lt 1e0)`, "true"},
		// Deterministic: same arguments give the same results.
		{"random-number-generator(42)?number eq random-number-generator(42)?number", "true"},
		{"random-number-generator()?number eq random-number-generator()?number", "true"},
		{"random-number-generator(())?number eq random-number-generator(())?number", "true"},
		{"random-number-generator(42)?next()?next()?number eq random-number-generator(42)?next()?next()?number", "true"},
		{"deep-equal(random-number-generator('x')?permute(1 to 50), random-number-generator('x')?permute(1 to 50))", "true"},
		// next() returns another generator.
		{"random-number-generator(5)?next() instance of map(xs:string, item())", "true"},
		{"let $n := random-number-generator(5)?next()?number return $n ge 0 and $n lt 1", "true"},
		// permute returns a permutation of its argument (same items, any order).
		{"let $p := random-number-generator(3)?permute(1 to 100) return count($p) eq 100 and deep-equal(sort($p), 1 to 100)", "true"},
		{"let $p := random-number-generator(3)?next()?permute(('a', 'b', 'c', 'd')) return deep-equal(sort($p), ('a', 'b', 'c', 'd'))", "true"},
		{"empty(random-number-generator(3)?permute(()))", "true"},
		{"random-number-generator(3)?permute('only')", "only"},
		// Spec example: a random permutation of 1 to 100.
		{"count(distinct-values(fn:random-number-generator()?permute(1 to 100)))", "100"},
	}, nil)
}
