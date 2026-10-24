package xpath

import "testing"

// TestMapArraySpec covers the array: functions of F&O 3.1 §17.3 and map:find
// (§17.1.9) with the REC's own examples; array results are compared with
// deep-equal against the expected array constructor.
func TestMapArraySpec(t *testing.T) {
	const abcd = `["a", "b", "c", "d"]`
	coreRun(t, "", nil, []coreCase{
		// §17.3.5 array:subarray
		{expr: `deep-equal(array:subarray(` + abcd + `, 2), ["b", "c", "d"])`, want: "true"},
		{expr: `deep-equal(array:subarray(` + abcd + `, 5), [])`, want: "true"},
		{expr: `deep-equal(array:subarray(` + abcd + `, 2, 0), [])`, want: "true"},
		{expr: `deep-equal(array:subarray(` + abcd + `, 2, 1), ["b"])`, want: "true"},
		{expr: `deep-equal(array:subarray(` + abcd + `, 2, 2), ["b", "c"])`, want: "true"},
		{expr: `deep-equal(array:subarray(` + abcd + `, 5, 0), [])`, want: "true"},
		{expr: `deep-equal(array:subarray([], 1, 0), [])`, want: "true"},
		{expr: `array:subarray(` + abcd + `, 0)`, err: "FOAY0001"},
		{expr: `array:subarray(` + abcd + `, 6)`, err: "FOAY0001"},
		{expr: `array:subarray(` + abcd + `, 2, -1)`, err: "FOAY0002"},
		{expr: `array:subarray(` + abcd + `, 2, 4)`, err: "FOAY0001"},
		{expr: `array:subarray('a', 1)`, err: "XPTY0004"},

		// §17.3.6 array:remove
		{expr: `deep-equal(array:remove(["a"], 1), [])`, want: "true"},
		{expr: `deep-equal(array:remove(` + abcd + `, 1), ["b", "c", "d"])`, want: "true"},
		{expr: `deep-equal(array:remove(["a", "b"], 2), ["a"])`, want: "true"},
		{expr: `deep-equal(array:remove(` + abcd + `, 1 to 3), ["d"])`, want: "true"},
		{expr: `deep-equal(array:remove([], ()), [])`, want: "true"},
		{expr: `array:remove(["a"], 2)`, err: "FOAY0001"},
		{expr: `array:remove(["a"], 0)`, err: "FOAY0001"},
		{expr: `array:remove(1, 1)`, err: "XPTY0004"},

		// §17.3.7 array:insert-before
		{expr: `deep-equal(array:insert-before(` + abcd + `, 3, ("x", "y")), ["a", "b", ("x", "y"), "c", "d"])`, want: "true"},
		{expr: `deep-equal(array:insert-before(` + abcd + `, 5, ("x", "y")), ["a", "b", "c", "d", ("x", "y")])`, want: "true"},
		{expr: `deep-equal(array:insert-before(` + abcd + `, 3, ["x", "y"]), ["a", "b", ["x", "y"], "c", "d"])`, want: "true"},
		{expr: `deep-equal(array:insert-before([], 1, 'x'), ['x'])`, want: "true"},
		{expr: `array:insert-before(` + abcd + `, 6, 'x')`, err: "FOAY0001"},
		{expr: `array:insert-before(` + abcd + `, 0, 'x')`, err: "FOAY0001"},
		{expr: `array:insert-before('a', 1, 'x')`, err: "XPTY0004"},

		// §17.3.4 array:put
		{expr: `deep-equal(array:put(["a", "b", "c"], 2, "d"), ["a", "d", "c"])`, want: "true"},
		{expr: `deep-equal(array:put(["a", "b", "c"], 2, ("d", "e")), ["a", ("d", "e"), "c"])`, want: "true"},
		{expr: `deep-equal(array:put(["a"], 1, ["d", "e"]), [["d", "e"]])`, want: "true"},
		// the input array is unchanged
		{expr: `let $a := ["a", "b"], $b := array:put($a, 1, "z") return ($a(1), $b(1))`, want: "a|z"},
		{expr: `array:put([], 1, 'x')`, err: "FOAY0001"},
		{expr: `array:put(["a"], 2, 'x')`, err: "FOAY0001"},
		{expr: `array:put('a', 1, 'x')`, err: "XPTY0004"},

		// §17.3.9/§17.3.10 array:head / array:tail
		{expr: `array:head([5, 6, 7, 8])`, want: "5"},
		{expr: `deep-equal(array:head([["a", "b"], ["c", "d"]]), ["a", "b"])`, want: "true"},
		{expr: `deep-equal(array:head([("a", "b"), ("c", "d")]), ("a", "b"))`, want: "true"},
		{expr: `array:head([])`, err: "FOAY0001"},
		{expr: `array:head(1)`, err: "XPTY0004"},
		{expr: `deep-equal(array:tail([5, 6, 7, 8]), [6, 7, 8])`, want: "true"},
		{expr: `deep-equal(array:tail([5]), [])`, want: "true"},
		{expr: `array:tail([])`, err: "FOAY0001"},
		{expr: `array:tail(1)`, err: "XPTY0004"},

		// §17.3.11 array:reverse
		{expr: `deep-equal(array:reverse(` + abcd + `), ["d", "c", "b", "a"])`, want: "true"},
		{expr: `deep-equal(array:reverse([("a", "b"), ("c", "d")]), [("c", "d"), ("a", "b")])`, want: "true"},
		{expr: `deep-equal(array:reverse([(1 to 5)]), [(1, 2, 3, 4, 5)])`, want: "true"},
		{expr: `deep-equal(array:reverse([]), [])`, want: "true"},
		{expr: `array:reverse(1)`, err: "XPTY0004"},

		// §17.3.14 array:filter
		{expr: `deep-equal(array:filter(["A", "B", 1, 2], function($x) {$x instance of xs:integer}), [1, 2])`, want: "true"},
		{expr: `deep-equal(array:filter(["the", "cat", "sat", "on", "the", "mat"], function($s){fn:string-length($s) gt 1}), ["the", "cat", "sat", "on", "the", "mat"])`, want: "true"},
		{expr: `deep-equal(array:filter(["A", "B", "", 0, 1], boolean#1), ["A", "B", 1])`, want: "true"},
		// the predicate must return a single xs:boolean
		{expr: `array:filter(["a"], function($x) {$x})`, err: "XPTY0004"},
		{expr: `array:filter(1, boolean#1)`, err: "XPTY0004"},
		{expr: `array:filter(["a"], abs#1)`, err: "XPTY0004"},
		{expr: `array:filter(["a"], concat#2)`, err: "XPTY0004"},

		// §17.3.15/§17.3.16 array:fold-left / array:fold-right
		{expr: `array:fold-left([true(), true(), false()], true(), function($x, $y){$x and $y})`, want: "false"},
		{expr: `array:fold-left([true(), true(), false()], false(), function($x, $y){$x or $y})`, want: "true"},
		{expr: `deep-equal(array:fold-left([1, 2, 3], [], function($x, $y){[$x, $y]}), [[[[], 1], 2], 3])`, want: "true"},
		{expr: `array:fold-left([], 7, function($x, $y){$x + $y})`, want: "7"},
		{expr: `array:fold-left([1], 0, function($x, $y){error()})`, err: "FOER0000"},
		{expr: `array:fold-left(1, 0, function($x, $y){$x})`, err: "XPTY0004"},
		{expr: `array:fold-left([1], 0, abs#1)`, err: "XPTY0004"},
		{expr: `array:fold-right([true(), true(), false()], true(), function($x, $y){$x and $y})`, want: "false"},
		{expr: `array:fold-right([true(), true(), false()], false(), function($x, $y){$x or $y})`, want: "true"},
		{expr: `deep-equal(array:fold-right([1, 2, 3], [], function($x, $y){[$x, $y]}), [1, [2, [3, []]]])`, want: "true"},
		{expr: `array:fold-right([1], 0, function($x, $y){error()})`, err: "FOER0000"},
		{expr: `array:fold-right(1, 0, function($x, $y){$x})`, err: "XPTY0004"},
		{expr: `array:fold-right([1], 0, abs#1)`, err: "XPTY0004"},

		// §17.3.17 array:for-each-pair
		{expr: `deep-equal(array:for-each-pair(["A", "B", "C"], [1, 2, 3], function($x, $y) { array {$x, $y}}), [["A", 1], ["B", 2], ["C", 3]])`, want: "true"},
		{expr: `let $A := ["A", "B", "C", "D"] return deep-equal(array:for-each-pair($A, array:tail($A), concat#2), ["AB", "BC", "CD"])`, want: "true"},
		{expr: `deep-equal(array:for-each-pair([1, 2], [10, 20, 30], function($x, $y){$x + $y}), [11, 22])`, want: "true"},
		{expr: `array:for-each-pair([1], [2], function($x, $y){error()})`, err: "FOER0000"},
		{expr: `array:for-each-pair(1, [2], concat#2)`, err: "XPTY0004"},
		{expr: `array:for-each-pair([1], 2, concat#2)`, err: "XPTY0004"},
		{expr: `array:for-each-pair([1], [2], abs#1)`, err: "XPTY0004"},

		// §17.3.19 array:sort
		{expr: `deep-equal(array:sort([1, 4, 6, 5, 3]), [1, 3, 4, 5, 6])`, want: "true"},
		{expr: `deep-equal(array:sort([1, -2, 5, 10, -10, 10, 8], (), fn:abs#1), [1, -2, 5, 8, 10, -10, 10])`, want: "true"},
		{expr: `deep-equal(array:sort([(1, 0), (1, 1), (0, 1), (0, 0)]), [(0, 0), (0, 1), (1, 0), (1, 1)])`, want: "true"},
		// a key that is a prefix of another sorts first
		{expr: `deep-equal(array:sort([(1, 2), 1, ()]), [(), 1, (1, 2)])`, want: "true"},
		// NaN sorts before every other numeric value
		{expr: `array:sort([2, xs:double('NaN'), 1])?* ! string(.)`, want: "NaN|1|2"},
		{expr: `array:sort([xs:double('NaN'), xs:double('NaN')])?* ! string(.)`, want: "NaN|NaN"},
		{expr: `array:sort(["b", "a", "C"])?* ! string(.)`, want: "C|a|b"},
		{expr: `array:sort(["b", "a", "C"], "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive")?* ! string(.)`, want: "a|b|C"},
		{expr: `array:sort(["b", "a", "C"], "http://www.w3.org/2005/xpath-functions/collation/codepoint")?* ! string(.)`, want: "C|a|b"},
		// stable for equal keys
		{expr: `array:sort(["b1", "a1", "b2", "a2"], (), substring(?, 1, 1))?* ! string(.)`, want: "a1|a2|b1|b2"},
		{expr: `array:sort([1, "a"])`, err: "XPTY0004"},
		{expr: `array:sort([xs:date('2001-01-01'), 1])`, err: "XPTY0004"},
		{expr: `array:sort([1, 2], (), function($x){error()})`, err: "FOER0000"},
		{expr: `array:sort(["a", "b"], "http://example.com/no-such-collation")`, err: "FOCH0002"},
		{expr: `array:sort(1)`, err: "XPTY0004"},

		// §17.1.9 map:find
		{expr: `let $responses := [map{0:'no', 1:'yes'}, map{0:'non', 1:'oui'}, map{0:'nein', 1:('ja', 'doch')}] return deep-equal(map:find($responses, 0), ['no', 'non', 'nein'])`, want: "true"},
		{expr: `let $responses := [map{0:'no', 1:'yes'}, map{0:'non', 1:'oui'}, map{0:'nein', 1:('ja', 'doch')}] return deep-equal(map:find($responses, 1), ['yes', 'oui', ('ja', 'doch')])`, want: "true"},
		{expr: `let $responses := [map{0:'no', 1:'yes'}, map{0:'non', 1:'oui'}, map{0:'nein', 1:('ja', 'doch')}] return deep-equal(map:find($responses, 2), [])`, want: "true"},
		{expr: `let $inventory := map{"name":"car", "id":"QZ123", "parts": [map{"name":"engine", "id":"YW678", "parts":[]}]} return array:size(map:find($inventory, "parts"))`, want: "2"},
		{expr: `deep-equal(map:find((1, "a"), "a"), [])`, want: "true"},
	})
}
