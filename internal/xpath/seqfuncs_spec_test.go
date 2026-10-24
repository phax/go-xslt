package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// sqEval parses and evaluates expr against the document parsed from xml
// (a minimal <r/> when xml is empty) and returns the result or the error.
func sqEval(t *testing.T, expr, xml string, coll func(a, b string) int) (Object, error) {
	t.Helper()
	p, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	if xml == "" {
		xml = "<r/>"
	}
	doc, err := xmltree.Parse(xml)
	if err != nil {
		t.Fatalf("parse xml: %v", err)
	}
	return p.Eval(&Context{Node: doc, Pos: 1, Size: 1, DefaultCollation: coll})
}

type sqCase struct{ expr, want string }

// sqRun checks each expression's string value (joined by the expression
// itself where it yields a sequence).
func sqRun(t *testing.T, xml string, cases []sqCase) {
	t.Helper()
	for _, c := range cases {
		v, err := sqEval(t, c.expr, xml, nil)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if got := ToString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

type sqErrCase struct{ expr, code string }

// sqRunErr checks that each expression raises the given error code.
func sqRunErr(t *testing.T, xml string, cases []sqErrCase) {
	t.Helper()
	for _, c := range cases {
		v, err := sqEval(t, c.expr, xml, nil)
		if err == nil {
			t.Errorf("%s: want %s, got value %q", c.expr, c.code, ToString(v))
			continue
		}
		if !strings.Contains(err.Error(), c.code) {
			t.Errorf("%s: want %s, got %v", c.expr, c.code, err)
		}
	}
}

// F&O 3.1 §14.1.6 fn:reverse, §14.1.7 fn:subsequence, §14.2.1
// fn:distinct-values, §14.2.2 fn:index-of — including the REC examples.
func TestSeqSpecBasics(t *testing.T) {
	sqRun(t, "", []sqCase{
		// §14.1.6 examples.
		{`string-join(reverse(("a","b","c")), ",")`, "c,b,a"},
		{`reverse(("hello"))`, "hello"},
		{`count(reverse(()))`, "0"},
		{`count(reverse([1,2,3]))`, "1"},
		{`reverse(([1,2,3],[4,5,6]))[1](1)`, "4"},

		// §14.1.7 examples with $seq := ("item1" .. "item5").
		{`string-join(subsequence(("item1","item2","item3","item4","item5"), 4), ",")`, "item4,item5"},
		{`string-join(subsequence(("item1","item2","item3","item4","item5"), 3, 2), ",")`, "item3,item4"},
		// The rules are stated on round($start) and round($length): positions
		// p with round(start) <= p < round(start)+round(length).
		{`string-join(subsequence((1,2,3,4,5), 1.5, 2), ",")`, "2,3"},
		{`string-join(subsequence((1,2,3,4,5), 0, 3), ",")`, "1,2"},
		{`string-join(subsequence((1,2,3,4,5), -1), ",")`, "1,2,3,4,5"},
		{`count(subsequence((1,2,3), 5))`, "0"},
		{`count(subsequence((1,2,3), 2, 0))`, "0"},
		{`count(subsequence((1,2,3), 2, -1))`, "0"},
		{`count(subsequence((1,2,3), xs:double('NaN')))`, "0"},
		{`count(subsequence((1,2,3), 1, xs:double('NaN')))`, "0"},
		{`string-join(subsequence((1,2,3), 2, xs:double('INF')), ",")`, "2,3"},
		// -INF + INF is NaN: nothing selected (§14.1.7 notes).
		{`count(subsequence((1,2,3), xs:double('-INF'), xs:double('INF')))`, "0"},
		// untypedAtomic arguments are cast to xs:double (function conversion).
		{`string-join(subsequence((1,2,3), xs:untypedAtomic('2')), ",")`, "2,3"},

		// §14.2.1: numeric equality across types, NaN equal to itself.
		{`count(distinct-values((1, 2.0, 3, 2)))`, "3"},
		{`count(distinct-values((xs:double('NaN'), xs:float('NaN'), 0 div 0e0)))`, "1"},
		{`count(distinct-values((0e0, -0e0)))`, "1"},
		// a string and a number are never equal.
		{`count(distinct-values((1, '1')))`, "2"},
		// untypedAtomic compares as a string; the REC example.
		{`count(distinct-values((xs:untypedAtomic("cherry"), xs:untypedAtomic("plum"), xs:untypedAtomic("plum"))))`, "2"},
		// xs:float 1.2 and xs:decimal 1.2 are eq (comparison in xs:float).
		{`count(distinct-values((xs:float('1.2'), 1.2)))`, "1"},
		// distinct dates with the same instant collapse.
		{`count(distinct-values((xs:dateTime('2000-01-01T12:00:00Z'), xs:dateTime('2000-01-01T13:00:00+01:00'))))`, "1"},
		// a yearMonthDuration and a dayTimeDuration of zero length are eq.
		{`count(distinct-values((xs:yearMonthDuration('P0M'), xs:dayTimeDuration('PT0S'))))`, "1"},
		{`count(distinct-values((xs:yearMonthDuration('P1Y'), xs:yearMonthDuration('P12M'))))`, "1"},
		// incomparable types are simply distinct.
		{`count(distinct-values((xs:date('2000-01-01'), 1, true())))`, "3"},
		{`count(distinct-values((true(), false(), true())))`, "2"},
		{`string-join(distinct-values(('a','b','a','c','b')), ',')`, "a,b,c"},
		// atomization of nodes.
		{`count(distinct-values((xs:untypedAtomic('a'), 'a')))`, "1"},
		// an explicit codepoint collation.
		{`count(distinct-values(('a','A'), 'http://www.w3.org/2005/xpath-functions/collation/codepoint'))`, "2"},
		// html-ascii-case-insensitive makes 'a' and 'A' equal.
		{`count(distinct-values(('a','A','b'), 'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'))`, "2"},
		{`count(distinct-values(('a','A','b'), 'http://www.w3.org/2013/collation/UCA?strength=primary'))`, "2"},

		// §14.2.2 examples.
		{`count(index-of((10, 20, 30, 40), 35))`, "0"},
		{`string-join(index-of((10, 20, 30, 30, 20, 10), 20), ",")`, "2,5"},
		{`string-join(index-of(("a", "sport", "and", "a", "pastime"), "a"), ",")`, "1,4"},
		{`count(index-of(current-date(), 23))`, "0"},
		{`string-join(index-of([1, [5, 6], [6, 7]], 6), ",")`, "3,4"},
		// incomparable pairs are unequal, not errors; NaN equals nothing.
		{`count(index-of((4, '4'), '4'))`, "1"},
		{`count(index-of((xs:double('NaN')), xs:double('NaN')))`, "0"},
		{`string-join(index-of((1, 1.0, 1e0, xs:float(1)), 1), ',')`, "1,2,3,4"},
		{`string-join(index-of(('a', 'A'), 'a', 'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'), ',')`, "1,2"},
		{`string-join(index-of(('a', 'A'), 'a', 'http://www.w3.org/2005/xpath-functions/collation/codepoint'), ',')`, "1"},
		{`string-join(index-of((xs:untypedAtomic('x'), 'x'), 'x'), ',')`, "1,2"},
	})
	sqRunErr(t, "", []sqErrCase{
		// $start is xs:double: a string or the empty sequence is a type error.
		{`subsequence((1,2), 'a')`, "XPTY0004"},
		{`subsequence((1,2), ())`, "XPTY0004"},
		{`subsequence((1,2), 1, 'x')`, "XPTY0004"},
		{`subsequence((1,2), xs:untypedAtomic('z'))`, "FORG0001"},
		// $search is a single atomic value.
		{`index-of((1,2), (1,2))`, "XPTY0004"},
		{`index-of((1,2), ())`, "XPTY0004"},
		// $collation is xs:string, unknown URIs FOCH0002.
		{`index-of((1,2), 1, ())`, "XPTY0004"},
		{`index-of(('a'), 'a', 'http://example.com/no-such-collation')`, "FOCH0002"},
		{`distinct-values(('a'), 'http://example.com/no-such-collation')`, "FOCH0002"},
	})
}

// F&O 3.1 §14.4: fn:count/avg/max/min/sum, with the REC examples.
func TestSeqSpecAggregates(t *testing.T) {
	sqRun(t, "", []sqCase{
		// §14.4.2 fn:avg. $seq3 := (3, 4, 5).
		{`avg((3, 4, 5))`, "4"},
		{`avg((3, 4, 5)) instance of xs:decimal`, "true"},
		{`avg((xs:yearMonthDuration("P20Y"), xs:yearMonthDuration("P10M")))`, "P10Y5M"},
		{`avg((xs:dayTimeDuration("PT2H"), xs:dayTimeDuration("PT4H")))`, "PT3H"},
		{`count(avg(()))`, "0"},
		{`string(avg((xs:float('INF'), xs:float('-INF'))))`, "NaN"},
		{`avg((xs:float('INF'), xs:float('-INF'))) instance of xs:float`, "true"},
		{`string(avg((3, 4, 5, xs:float('NaN'))))`, "NaN"},
		{`avg((1.5, 2.5))`, "2"},
		{`avg((1.5, 2.5)) instance of xs:decimal`, "true"},
		{`avg((1, 2e0))`, "1.5"},
		{`avg((1, 2e0)) instance of xs:double`, "true"},
		{`avg((xs:float(1), xs:float(2))) instance of xs:float`, "true"},
		{`avg((xs:float(1), 2)) instance of xs:float`, "true"},
		// untypedAtomic is cast to xs:double.
		{`avg((xs:untypedAtomic('1'), xs:untypedAtomic('2'))) instance of xs:double`, "true"},
		{`avg([1, 2, 3])`, "2"},

		// §14.4.3 fn:max.
		{`max((3,4,5))`, "5"},
		{`max([3,4,5])`, "5"},
		{`max((xs:integer(5), xs:float(5.0), xs:double(0))) instance of xs:double`, "true"},
		{`max((xs:integer(5), xs:float(5.0), xs:double(0)))`, "5"},
		{`max((fn:current-date(), xs:date("2100-01-01")))`, "2100-01-01"},
		{`max(("a", "b", "c"))`, "c"},
		{`max((1, xs:float(2))) instance of xs:float`, "true"},
		{`string(max((1, xs:double('NaN'), 3)))`, "NaN"},
		{`string(max((1, xs:float('NaN')))) `, "NaN"},
		{`max((1, xs:float('NaN'))) instance of xs:float`, "true"},
		{`max((xs:dayTimeDuration('PT1H'), xs:dayTimeDuration('PT2H')))`, "PT2H"},
		{`max((xs:yearMonthDuration('P1Y'), xs:yearMonthDuration('P2M')))`, "P1Y"},
		{`max((true(), false()))`, "true"},
		{`max((xs:time('10:00:00'), xs:time('09:00:00')))`, "10:00:00"},
		{`max(('a', 'B'), 'http://www.w3.org/2005/xpath-functions/collation/codepoint')`, "a"},
		// An xs:anyURI among strings is promoted to xs:string (§14.4.3 rules).
		{`max((xs:anyURI('b'), 'a')) instance of xs:string`, "true"},
		{`max((xs:anyURI('b'), 'a'))`, "b"},
		{`max((xs:untypedAtomic('7'), 3)) instance of xs:double`, "true"},
		{`max((1.5, 2)) instance of xs:decimal`, "true"},
		{`max((1.5, 2))`, "2"},
		{`count(max(()))`, "0"},

		// §14.4.4 fn:min.
		{`min((3,4,5))`, "3"},
		{`min([3,4,5])`, "3"},
		{`min((xs:integer(5), xs:float(5), xs:double(10))) instance of xs:double`, "true"},
		{`min((xs:integer(5), xs:float(5), xs:double(10)))`, "5"},
		{`min((fn:current-date(), xs:date("1900-01-01")))`, "1900-01-01"},
		{`min(("a", "b", "c"))`, "a"},
		// equal minimal items: which one is returned is implementation-dependent.
		{`min(('b', 'A', 'a'), 'http://www.w3.org/2013/collation/UCA?strength=primary') = ('A', 'a')`, "true"},
		{`min((xs:untypedAtomic('3'), 7)) instance of xs:double`, "true"},
		{`string(min((xs:double('NaN'), 1)))`, "NaN"},

		// §14.4.5 fn:sum.
		{`sum((xs:yearMonthDuration("P20Y"), xs:yearMonthDuration("P10M")))`, "P20Y10M"},
		{`sum((xs:dayTimeDuration("PT1H"), xs:dayTimeDuration("PT30M")))`, "PT1H30M"},
		{`sum((3, 4, 5))`, "12"},
		{`sum(())`, "0"},
		{`sum(()) instance of xs:integer`, "true"},
		{`count(sum((),()))`, "0"},
		{`sum((1 to 100)[. lt 0], 0)`, "0"},
		{`sum((xs:yearMonthDuration("P20Y"), xs:yearMonthDuration("P10M")), "ein Augenblick")`, "P20Y10M"},
		{`sum([1, 2, 3])`, "6"},
		{`sum([[1, 2], [3, 4]])`, "10"},
		{`sum((1.5, 2.25))`, "3.75"},
		{`sum((1.5, 2.25)) instance of xs:decimal`, "true"},
		{`sum((1, 2.5e0)) instance of xs:double`, "true"},
		{`sum((xs:float(1), 2)) instance of xs:float`, "true"},
		{`sum((xs:untypedAtomic('1'), 2)) instance of xs:double`, "true"},
		{`sum(xs:unsignedShort(3)) instance of xs:unsignedShort`, "true"},
		{`sum((99999999999999999999, 1))`, "100000000000000000000"},
	})
	sqRunErr(t, "", []sqErrCase{
		// FORG0006: mixing non-numeric / unordered types.
		{`sum(('a', 'b'))`, "FORG0006"},
		{`avg(('a', 'b'))`, "FORG0006"},
		{`sum((xs:yearMonthDuration('P1Y'), xs:dayTimeDuration('PT1H')))`, "FORG0006"},
		{`sum((xs:duration('P1Y'), xs:duration('P1M')))`, "FORG0006"},
		{`sum((1, xs:dayTimeDuration('PT1H')))`, "FORG0006"},
		{`max((1, 'a'))`, "FORG0006"},
		{`min((xs:date('2000-01-01'), 1))`, "FORG0006"},
		{`max((xs:duration('P1Y'), xs:duration('P1M')))`, "FORG0006"},
		{`max((QName('', 'a'), QName('', 'b')))`, "FORG0006"},
		{`max((xs:gYear('2000'), xs:gYear('2001')))`, "FORG0006"},
		{`max((xs:yearMonthDuration('P1Y'), xs:dayTimeDuration('PT1H')))`, "FORG0006"},
		// untypedAtomic is cast to xs:double first.
		{`max((xs:untypedAtomic('x'), 1))`, "FORG0001"},
		{`max(('a'), 'http://example.com/no-such-collation')`, "FOCH0002"},
		{`min(('a'), 'http://example.com/no-such-collation')`, "FOCH0002"},
	})
}

// F&O 3.1 §16.2 higher-order sequence functions and §14.2.4 fn:sort, with the
// REC examples.
func TestSeqSpecHigherOrder(t *testing.T) {
	sqRun(t, "", []sqCase{
		{`string-join(for-each(1 to 5, function($a) { $a * $a }), ",")`, "1,4,9,16,25"},
		{`string-join(for-each(("john", "jane"), fn:string-to-codepoints#1), ",")`, "106,111,104,110,106,97,110,101"},
		{`string-join(for-each(("23", "29"), xs:int#1), ",")`, "23,29"},
		// a map is a function item of arity 1.
		{`string-join(for-each(("a", "b"), map{"a": 1, "b": 2}), ",")`, "1,2"},
		{`string-join(for-each((1, 2), ["x", "y"]), ",")`, "x,y"},

		{`string-join(filter(1 to 10, function($a) {$a mod 2 = 0}), ",")`, "2,4,6,8,10"},
		{`count(filter((), fn:lang("en", ?)))`, "0"},
		{`string-join(filter((1, 2), map{1: true(), 2: false()}), ",")`, "1"},

		{`fold-left(1 to 5, 0, function($a, $b) { $a + $b })`, "15"},
		{`fold-left((2,3,5,7), 1, function($a, $b) { $a * $b })`, "210"},
		{`fold-left((true(), false(), false()), false(), function($a, $b) { $a or $b })`, "true"},
		{`fold-left((true(), false(), false()), false(), function($a, $b) { $a and $b })`, "false"},
		{`string-join(fold-left(1 to 5, (), function($a, $b) {($b, $a)}), ",")`, "5,4,3,2,1"},
		{`fold-left(1 to 5, "", fn:concat(?, ".", ?))`, ".1.2.3.4.5"},
		{`fold-left(1 to 5, "$zero", fn:concat("$f(", ?, ", ", ?, ")"))`, "$f($f($f($f($f($zero, 1), 2), 3), 4), 5)"},
		{`fold-left(1 to 5, map{}, function($map, $n) {map:put($map, $n, $n*2)})(4)`, "8"},

		{`fold-right(1 to 5, 0, function($a, $b) { $a + $b })`, "15"},
		{`fold-right(1 to 5, "", fn:concat(?, ".", ?))`, "1.2.3.4.5."},
		{`fold-right(1 to 5, "$zero", concat("$f(", ?, ", ", ?, ")"))`, "$f(1, $f(2, $f(3, $f(4, $f(5, $zero)))))"},

		{`string-join(for-each-pair(("a", "b", "c"), ("x", "y", "z"), concat#2), ",")`, "ax,by,cz"},
		{`string-join(for-each-pair(1 to 5, 1 to 5, function($a, $b){10*$a + $b}), ",")`, "11,22,33,44,55"},
		// the shorter sequence bounds the result.
		{`string-join(for-each-pair(1 to 3, 1 to 2, function($a, $b){$a + $b}), ",")`, "2,4"},

		// §14.2.4? fn:sort examples (F&O 3.1 §16.2.?? fn:sort).
		{`string-join(sort((1, 4, 6, 5, 3)), ",")`, "1,3,4,5,6"},
		{`string-join(sort((1, -2, 5, 10, -10, 10, 8), (), fn:abs#1), ",")`, "1,-2,5,8,10,-10,10"},
		{`string-join(sort(("b", "A", "a")), ",")`, "A,a,b"},
		{`string-join(sort(("b", "A", "a", "B"), "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"), ",")`, "A,a,b,B"},
		// NaN sorts before every other value; a shorter key sorts first.
		{`string-join(sort((3, xs:double('NaN'), 1)), ",")`, "NaN,1,3"},
		{`string-join(for-each(sort(([2,1],[1],[1,1]), (), function($a){$a}), function($a){string-join($a?*, '')}), ",")`, "1,11,21"},
		{`string-join(sort((xs:untypedAtomic('b'), 'a')), ",")`, "a,b"},
		{`count(sort(()))`, "0"},
	})
	sqRunErr(t, "", []sqErrCase{
		// a function argument must be a single function of the declared arity.
		{`for-each((1), (abs#1, abs#1))`, "XPTY0004"},
		{`for-each((1), 1)`, "XPTY0004"},
		{`for-each((1), concat#2)`, "XPTY0004"},
		{`filter((1), abs#1)`, "XPTY0004"},
		{`filter((1), function($a){($a, $a)})`, "XPTY0004"},
		{`fold-left((1), 0, abs#1)`, "XPTY0004"},
		{`fold-right((1), 0, abs#1)`, "XPTY0004"},
		{`for-each-pair((1), (2), abs#1)`, "XPTY0004"},
		{`sort((1,2), (), concat#2)`, "XPTY0004"},
		// keys must be mutually comparable.
		{`sort((1, "a"))`, "XPTY0004"},
		{`sort(('a'), 'http://example.com/no-such-collation')`, "FOCH0002"},
		// errors raised by the callback propagate.
		{`for-each((0), function($a){1 idiv $a})`, "FOAR0001"},
		{`filter((0), function($a){1 idiv $a = 1})`, "FOAR0001"},
		{`fold-left((0), 0, function($a, $b){1 idiv $b})`, "FOAR0001"},
		{`fold-right((0), 0, function($a, $b){1 idiv $a})`, "FOAR0001"},
		{`for-each-pair((0), (0), function($a, $b){1 idiv $a})`, "FOAR0001"},
		{`sort((0, 1), (), function($a){1 idiv $a})`, "FOAR0001"},
	})
}

// F&O 3.1 §4.8 math functions (the subset registered in seqfuncs.go).
func TestSeqSpecMath(t *testing.T) {
	sqRun(t, "", []sqCase{
		{`math:pi()`, "3.141592653589793"},
		{`count(math:sqrt(()))`, "0"},
		{`math:sqrt(0.0e0)`, "0"},
		{`string(math:sqrt(-0.0e0))`, "-0"},
		{`math:sqrt(1.0e6)`, "1000"},
		{`string(math:sqrt(-2.0e0))`, "NaN"},
		{`string(math:sqrt(xs:double('INF')))`, "INF"},
		{`math:sin(0)`, "0"},
		{`math:cos(0)`, "1"},
		{`math:exp(0)`, "1"},
		{`abs(math:exp(1) - 2.718281828459045e0) lt 1e-15`, "true"},
		{`math:log(1)`, "0"},
		{`string(math:log(0))`, "-INF"},
		{`string(math:log(-1))`, "NaN"},
		{`math:log10(1.0e3)`, "3"},
		{`string(math:log10(0))`, "-INF"},
		{`count(math:pow((), 93.7))`, "0"},
		{`math:pow(2, 3)`, "8"},
		{`math:pow(-2, 3)`, "-8"},
		{`math:pow(2, -3)`, "0.125"},
		{`math:pow(0, 0)`, "1"},
		{`string(math:pow(0e0, -3))`, "INF"},
		{`math:pow(16, 0.5e0)`, "4"},
		{`string(math:pow(-1, 0.5))`, "NaN"},
		{`math:pow(xs:double('NaN'), 0)`, "1"},
		{`math:pow(-1, xs:double('INF'))`, "1"},
	})
}

// F&O 3.1 §17.1 map functions registered in seqfuncs.go.
func TestSeqSpecMapFunctions(t *testing.T) {
	sqRun(t, "", []sqCase{
		{`map:get(map{1:"yes", 2:"no"}, 1)`, "yes"},
		{`count(map:get(map{1:"yes", 2:"no"}, 3))`, "0"},
		{`count(map:get(map{1:"yes", 2:"no", 3:()}, 3))`, "0"},
		{`map:contains(map{1:"yes", 2:"no"}, 1)`, "true"},
		{`map:contains(map{1:"yes", 2:"no"}, 3)`, "false"},
		{`map:contains(map{"xyz":23}, "xyz")`, "true"},
		{`map:contains(map{"abc":23, "xyz":()}, "xyz")`, "true"},
		{`map:contains(map{}, "xyz")`, "false"},
		// keys compare with op:same-key: 1 and 1.0 are the same key.
		{`map:contains(map{1: "a"}, 1.0)`, "true"},
		{`map:size(map{})`, "0"},
		{`map:size(map{"true":1, "false":0})`, "2"},
		{`string-join(for-each(sort(map:keys(map{1:"yes", 2:"no"})), string#1), ",")`, "1,2"},
		{`map:put(map{"Su":"Sunday"}, "Mo", "Monday")("Mo")`, "Monday"},
		{`map:size(map:put(map{"Su":"Sunday"}, "Su", "Sonntag"))`, "1"},
		{`map:put(map{"Su":"Sunday"}, "Su", "Sonntag")("Su")`, "Sonntag"},
		// the original map is unchanged.
		{`let $m := map{"a":1} return (map:put($m, "b", 2), map:size($m))[2]`, "1"},
		{`map:size(map:remove(map{0:"Sun", 1:"Mon", 2:"Tue"}, 1))`, "2"},
		{`map:contains(map:remove(map{0:"Sun", 1:"Mon", 2:"Tue"}, 1), 1)`, "false"},
		{`map:size(map:remove(map{0:"Sun", 1:"Mon", 2:"Tue"}, 23))`, "3"},
		{`map:size(map:remove(map{0:"Sun", 1:"Mon", 2:"Tue"}, (0, 2)))`, "1"},
		{`map:size(map:remove(map{0:"Sun", 1:"Mon", 2:"Tue"}, ()))`, "3"},
		{`map:entry("M", "Monday")("M")`, "Monday"},
		{`map:size(map:entry("M", "Monday"))`, "1"},
		// map:merge (§17.1.1) and its duplicates option.
		{`map:size(map:merge(()))`, "0"},
		{`map:size(map:merge((map:entry(0, "no"), map:entry(1, "yes"))))`, "2"},
		{`map:merge((map{0:"a"}, map{0:"b"}))(0)`, "a"},
		{`map:merge((map{0:"a"}, map{0:"b"}), map{"duplicates":"use-first"})(0)`, "a"},
		{`map:merge((map{0:"a"}, map{0:"b"}), map{"duplicates":"use-last"})(0)`, "b"},
		{`string-join(map:merge((map{0:"a"}, map{0:"b"}), map{"duplicates":"combine"})(0), ",")`, "a,b"},
		{`map:merge((map{0:"a"}, map{0:"b"}), map{"duplicates":"use-any"})(0) = ("a", "b")`, "true"},
		{`map:merge((map{0:"a"}, map{1:"b"}), map{"duplicates":"reject"})(1)`, "b"},
		{`map:merge((map{0:"a"}), map{})(0)`, "a"},
		{`string-join(sort(map:for-each(map{1:"yes", 2:"no"}, function($k, $v){$v})), ",")`, "no,yes"},
		{`sum(map:for-each(map{1:10, 2:20}, function($k, $v){$k + $v}))`, "33"},
	})
	sqRunErr(t, "", []sqErrCase{
		{`map:get((map{}, map{}), 1)`, "XPTY0004"},
		{`map:get(1, 1)`, "XPTY0004"},
		{`map:get(map{}, ())`, "XPTY0004"},
		{`map:get(map{}, (1, 2))`, "XPTY0004"},
		{`map:contains((), 1)`, "XPTY0004"},
		{`map:contains(map{}, (1, 2))`, "XPTY0004"},
		{`map:put((), 1, 2)`, "XPTY0004"},
		{`map:put(map{}, (), 2)`, "XPTY0004"},
		{`map:remove((), 1)`, "XPTY0004"},
		{`map:entry((), 1)`, "XPTY0004"},
		{`map:merge((map{0:"a"}, map{0:"b"}), map{"duplicates":"reject"})`, "FOJS0003"},
		{`map:merge((map{0:"a"}), map{"duplicates":"bogus"})`, "FOJS0005"},
		{`map:merge((map{0:"a"}), ())`, "XPTY0004"},
		{`map:for-each(map{1:0}, function($k, $v){1 idiv $v})`, "FOAR0001"},
	})
}

// F&O 3.1 §17.3 array functions registered in seqfuncs.go.
func TestSeqSpecArrayFunctions(t *testing.T) {
	sqRun(t, "", []sqCase{
		{`array:size(["a", "b", "c"])`, "3"},
		{`array:size(["a", ["b", "c"]])`, "2"},
		{`array:size([ ])`, "0"},
		{`array:size([[ ]])`, "1"},
		{`array:get(["a", "b", "c"], 2)`, "b"},
		{`array:get(["a", ["b", "c"]], 2)(1)`, "b"},
		{`array:size(array:append(["a", "b", "c"], "d"))`, "4"},
		{`array:get(array:append(["a", "b", "c"], "d"), 4)`, "d"},
		{`array:size(array:append(["a", "b", "c"], ("d", "e")))`, "4"},
		{`array:size(array:append(["a", "b", "c"], ["d", "e"]))`, "4"},
		{`string-join(array:flatten([1, 4, 6, 5, 3]), ",")`, "1,4,6,5,3"},
		{`string-join(array:flatten(([1, 2, 5], [[10, 11], 12], [], 13)), ",")`, "1,2,5,10,11,12,13"},
		{`string-join(array:flatten([(1,0), (1,1), (0,1), (0,0)]), ",")`, "1,0,1,1,0,1,0,0"},
		{`array:size(array:join(()))`, "0"},
		{`array:size(array:join([1, 2, 3]))`, "3"},
		{`array:size(array:join((["a", "b"], ["c", "d"])))`, "4"},
		{`array:size(array:join((["a", "b"], ["c", "d"], [ ])))`, "4"},
		{`array:size(array:join((["a", "b"], ["c", "d"], [["e", "f"]])))`, "5"},
		{`string-join(array:flatten(array:for-each(["A", "B", 1, 2], function($z) {$z instance of xs:integer})), ",")`, "false,false,true,true"},
		{`array:size(array:for-each(["the cat", "sat", "on the mat"], fn:tokenize#1))`, "3"},
		{`count(array:for-each(["the cat", "sat", "on the mat"], fn:tokenize#1)(3))`, "3"},
	})
	sqRunErr(t, "", []sqErrCase{
		{`array:get(["a"], 0)`, "FOAY0001"},
		{`array:get(["a"], 2)`, "FOAY0001"},
		{`array:get([], 1)`, "FOAY0001"},
		{`array:get(["a"], (1, 1))`, "XPTY0004"},
		{`array:get(["a"], 1.0)`, "XPTY0004"},
		{`array:get(map{}, 1)`, "XPTY0004"},
		{`array:for-each([1], concat#2)`, "XPTY0004"},
		{`array:for-each([0], function($a){1 idiv $a})`, "FOAR0001"},
	})
}

// sqDoc is a small document for the node-related cases.
const sqDoc = `<r><n v="3">3</n><n v="1">1</n><n v="2">x</n></r>`

// Aggregates and sequence functions over nodes: atomization yields
// xs:untypedAtomic, which aggregates cast to xs:double (F&O 3.1 §14.4).
func TestSeqSpecNodes(t *testing.T) {
	sqRun(t, sqDoc, []sqCase{
		{`sum(//n[@v != 2]/@v)`, "4"},
		{`sum(//n/@v) instance of xs:double`, "true"},
		{`max(//n/@v)`, "3"},
		{`min(//n/@v)`, "1"},
		{`avg(//n/@v)`, "2"},
		{`count(distinct-values(//n/@v))`, "3"},
		{`string-join(index-of(//n/@v, '1'), ',')`, "2"},
		{`count(subsequence(//n, //n[2]))`, "3"},
		{`string-join(remove(//n/@v, //n[2]), ',')`, "1,2"},
	})
	sqRunErr(t, sqDoc, []sqErrCase{
		{`sum(//n)`, "FORG0001"},
		{`remove((1,2), //n[3])`, "FORG0001"},
	})
}
