package xpath

import (
	"math"
	"strings"
	"testing"
)

// The function conversion rules (XPath 3.1 §3.1.5.2) applied to the arguments
// of every built-in function (issue #9): atomize, cast xs:untypedAtomic,
// promote numerics and xs:anyURI, then SequenceType matching — XPTY0004 on a
// mismatch of type or cardinality.

const argSrc = `<r><a>3</a><b>x</b><n>2.5</n><s>abc</s><!--c--></r>`

func TestArgCheckEmptyForRequired(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		// The four cases of the issue.
		{expr: `contains-token("red green blue", "RED", ())`, code: "XPTY0004"},
		{expr: `empty(math:atan2((), 1))`, code: "XPTY0004"},
		{expr: `error(QName('urn:e','E1'), (), 'v')`, code: "XPTY0004"},
		{expr: `error('a')`, code: "XPTY0004"},
		// A required (no occurrence indicator) parameter takes exactly one item.
		{expr: `math:atan2(1, ())`, code: "XPTY0004"},
		{expr: `math:pow(2, ())`, code: "XPTY0004"},
		{expr: `substring("abc", ())`, code: "XPTY0004"},
		{expr: `substring("abc", 1, ())`, code: "XPTY0004"},
		{expr: `translate("abc", "a", ())`, code: "XPTY0004"},
		{expr: `compare("a", "b", ())`, code: "XPTY0004"},
		{expr: `matches("a", "a", ())`, code: "XPTY0004"},
		{expr: `matches("a", ())`, code: "XPTY0004"},
		{expr: `replace("a", "a", ())`, code: "XPTY0004"},
		{expr: `tokenize("a b", " ", ())`, code: "XPTY0004"},
		{expr: `normalize-unicode("a", ())`, code: "XPTY0004"},
		{expr: `string-join(("a", "b"), ())`, code: "XPTY0004"},
		{expr: `resolve-uri("a", ())`, code: "XPTY0004"},
		{expr: `format-integer(1, ())`, code: "XPTY0004"},
		{expr: `round(1.5, ())`, code: "XPTY0004"},
		{expr: `QName("urn:x", ())`, code: "XPTY0004"},
		{expr: `index-of((1, 2), ())`, code: "XPTY0004"},
		{expr: `function-lookup(QName('urn:x', 'f'), ())`, code: "XPTY0004"},
		{expr: `distinct-values((1, 2), ())`, code: "XPTY0004"},
		{expr: `deep-equal(1, 2, ())`, code: "XPTY0004"},
		{expr: `sort((3, 1), (), ())`, code: "XPTY0004"},
		{expr: `map:get(map{}, ())`, code: "XPTY0004"},
		{expr: `map:contains(map{}, ())`, code: "XPTY0004"},
		{expr: `map:entry((), 1)`, code: "XPTY0004"},
		{expr: `array:get([1], ())`, code: "XPTY0004"},
		{expr: `array:size(())`, code: "XPTY0004"},
		{expr: `map:size(())`, code: "XPTY0004"},
		{expr: `environment-variable(())`, code: "XPTY0004"},
		// An optional parameter takes the empty sequence.
		{expr: `empty(codepoint-equal((), "a"))`, want: "true"},
		{expr: `string-length(())`, want: "0"},
		{expr: `upper-case(())`, want: ""},
		{expr: `empty(math:pow((), 3))`, want: "true"},
		{expr: `empty(abs(()))`, want: "true"},
		{expr: `empty(year-from-date(()))`, want: "true"},
		{expr: `local-name-from-QName(QName((), "a"))`, want: "a"},
		{expr: `concat("a", (), "b")`, want: "ab"},
		{expr: `empty(error(()))`, code: "FOER0000"},
	})
}

func TestArgCheckTooManyItems(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		{expr: `upper-case(("a", "b"))`, code: "XPTY0004"},
		{expr: `string-length(("a", "b"))`, code: "XPTY0004"},
		{expr: `string-length(r/*)`, code: "XPTY0004"},
		{expr: `substring("abc", (1, 2))`, code: "XPTY0004"},
		{expr: `abs((1, 2))`, code: "XPTY0004"},
		{expr: `round(1, (1, 2))`, code: "XPTY0004"},
		{expr: `name(r/*)`, code: "XPTY0004"},
		{expr: `concat(1, (2, 3))`, code: "XPTY0004"},
		{expr: `string((1, 2))`, code: "XPTY0004"},
		{expr: `number((1, 2))`, code: "XPTY0004"},
		{expr: `sum((1, 2), (1, 2))`, code: "XPTY0004"},
		{expr: `index-of((1, 2), (1, 2))`, code: "XPTY0004"},
		{expr: `random-number-generator((1, 2))`, code: "XPTY0004"},
		{expr: `generate-id((r, r/a))`, code: "XPTY0004"},
		{expr: `math:atan2(1, (2, 3))`, code: "XPTY0004"},
		{expr: `array:get([1, 2], (1, 2))`, code: "XPTY0004"},
		{expr: `map:get(map{}, (1, 2))`, code: "XPTY0004"},
		{expr: `map:size((map{}, map{}))`, code: "XPTY0004"},
		{expr: `function-arity((abs#1, abs#1))`, code: "XPTY0004"},
		{expr: `error(QName('urn:e', 'E1'), ('a', 'b'))`, code: "XPTY0004"},
		// A starred or plus parameter takes any number.
		{expr: `string-join(("a", "b", "c"), "-")`, want: "a-b-c"},
		{expr: `contains-token(("a", "b c"), "c")`, want: "true"},
	})
}

func TestArgCheckWrongAtomicType(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		{expr: `abs("a")`, code: "XPTY0004"},
		{expr: `abs(true())`, code: "XPTY0004"},
		{expr: `substring("abc", "2")`, code: "XPTY0004"},
		{expr: `substring("abc", true())`, code: "XPTY0004"},
		{expr: `upper-case(1)`, code: "XPTY0004"},
		{expr: `contains(true(), "a")`, code: "XPTY0004"},
		{expr: `string-length(1.5)`, code: "XPTY0004"},
		{expr: `starts-with("a", 1)`, code: "XPTY0004"},
		{expr: `math:sin("a")`, code: "XPTY0004"},
		{expr: `math:pow(2, "3")`, code: "XPTY0004"},
		{expr: `format-number(true(), "0")`, code: "XPTY0004"},
		{expr: `format-integer(1.5, "1")`, code: "XPTY0004"},
		{expr: `format-integer(5e0, "1")`, code: "XPTY0004"},
		{expr: `round(1.255, 2.0)`, code: "XPTY0004"},
		{expr: `codepoints-to-string((72, "i"))`, code: "XPTY0004"},
		{expr: `string-to-codepoints(1)`, code: "XPTY0004"},
		{expr: `trace(1, 5)`, code: "XPTY0004"},
		{expr: `doc-available(5)`, code: "XPTY0004"},
		{expr: `unparsed-text-available(5)`, code: "XPTY0004"},
		{expr: `normalize-unicode("a", 1)`, code: "XPTY0004"},
		{expr: `compare("a", 1)`, code: "XPTY0004"},
		{expr: `year-from-date("2020-01-01")`, code: "XPTY0004"},
		{expr: `year-from-date(xs:dateTime("2020-01-01T00:00:00"))`, code: "XPTY0004"},
		{expr: `hours-from-time(xs:dateTime("2020-01-01T10:00:00"))`, code: "XPTY0004"},
		{expr: `adjust-date-to-timezone(xs:date("2000-01-01"), "PT0S")`, code: "XPTY0004"},
		{expr: `format-date(xs:dateTime("2000-01-01T00:00:00"), "[Y]")`, code: "XPTY0004"},
		{expr: `years-from-duration("P1Y")`, code: "XPTY0004"},
		{expr: `prefix-from-QName("a")`, code: "XPTY0004"},
		{expr: `function-lookup("abs", 1)`, code: "XPTY0004"},
		{expr: `parse-json(5)`, code: "XPTY0004"},
		{expr: `parse-json("1", 5)`, code: "XPTY0004"},
		{expr: `parse-json("1", ())`, code: "XPTY0004"},
		{expr: `array:get([5], 1.0)`, code: "XPTY0004"},
		{expr: `array:get([5], "1")`, code: "XPTY0004"},
		{expr: `concat(map{}, 1)`, code: "FOTY0013"},
		{expr: `string-length(map{})`, code: "FOTY0013"},
		// Accepted: the declared type, a subtype, a promotion.
		{expr: `abs(1)`, want: "1"},
		{expr: `abs(-1.5)`, want: "1.5"},
		{expr: `abs(xs:float(-2))`, want: "2"},
		{expr: `abs(xs:byte(-2))`, want: "2"},
		{expr: `format-integer(xs:byte(5), "1")`, want: "5"},
		{expr: `substring("abcdef", 3)`, want: "cdef"},
		{expr: `substring("abcdef", 3.0)`, want: "cdef"},
		{expr: `substring("abcdef", xs:float(3))`, want: "cdef"},
		{expr: `substring("abcdef", 2, 2)`, want: "bc"},
		{expr: `math:sqrt(16)`, want: "4"},
		{expr: `math:pow(2, 3)`, want: "8"},
		{expr: `math:pow(2, 1.5) gt 2.8`, want: "true"},
		{expr: `string-join((1, 2), ",")`, want: "1,2"},
		{expr: `string-join((1, "a", true()), ",")`, want: "1,a,true"},
		{expr: `concat(1, true(), "a")`, want: "1truea"},
		{expr: `hours-from-duration(xs:yearMonthDuration("P1Y"))`, want: "0"},
		{expr: `years-from-duration(xs:dayTimeDuration("PT1H"))`, want: "0"},
		{expr: `year-from-dateTime(xs:dateTime("2020-01-01T00:00:00"))`, want: "2020"},
		{expr: `index-of((1, 2, 3), 2)`, want: "2"},
	})
}

func TestArgCheckUntypedAtomic(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		// Cast to the expected type.
		{expr: `abs(xs:untypedAtomic("-3"))`, want: "3"},
		{expr: `abs(xs:untypedAtomic("-3")) instance of xs:double`, want: "true"},
		{expr: `substring("abc", xs:untypedAtomic("2"))`, want: "bc"},
		{expr: `format-integer(xs:untypedAtomic("12"), "1")`, want: "12"},
		{expr: `round(xs:untypedAtomic("2.5"))`, want: "3"},
		{expr: `format-number(xs:untypedAtomic("3"), "0")`, want: "3"},
		{expr: `array:get([5], xs:untypedAtomic("1"))`, want: "5"},
		{expr: `codepoints-to-string(xs:untypedAtomic("72"))`, want: "H"},
		{expr: `year-from-date(xs:untypedAtomic("2020-01-01"))`, want: "2020"},
		{expr: `upper-case(xs:untypedAtomic("ab"))`, want: "AB"},
		// A failed cast is the cast's error.
		{expr: `substring("abc", xs:untypedAtomic("x"))`, code: "FORG0001"},
		{expr: `format-integer(xs:untypedAtomic("1.5"), "1")`, code: "FORG0001"},
		{expr: `year-from-dateTime(xs:untypedAtomic("x"))`, code: "FORG0001"},
		{expr: `math:sqrt(xs:untypedAtomic("x"))`, code: "FORG0001"},
		// xs:QName and xs:NOTATION are namespace-sensitive: no cast.
		{expr: `local-name-from-QName(xs:untypedAtomic("a"))`, code: "XPTY0117"},
		{expr: `error(xs:untypedAtomic("a"))`, code: "XPTY0117"},
		{expr: `function-lookup(xs:untypedAtomic("fn:abs"), 1)`, code: "XPTY0117"},
		// A string is not converted to a QName.
		{expr: `local-name-from-QName("a")`, code: "XPTY0004"},
	})
}

func TestArgCheckNodesAndArrays(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		// Atomized, then cast.
		{expr: `upper-case(r/s)`, want: "ABC"},
		{expr: `string-length(r/s)`, want: "3"},
		{expr: `string-length(r/nothing)`, want: "0"},
		{expr: `abs(r/a)`, want: "3"},
		{expr: `round(r/n)`, want: "3"},
		{expr: `substring("abcdef", r/a)`, want: "cdef"},
		{expr: `math:sqrt(r/a) gt 1.7`, want: "true"},
		{expr: `abs(r/b)`, code: "FORG0001"},
		{expr: `abs(r/s)`, code: "FORG0001"},
		{expr: `contains-token(r/s, "abc")`, want: "true"},
		{expr: `string-join(r/*, "-")`, want: "3-x-2.5-abc"},
		{expr: `upper-case(r/comment())`, want: "C"},
		{expr: `name(r/a)`, want: "a"},
		// A node where a node is declared; anything else is a type error.
		{expr: `name(1)`, code: "XPTY0004"},
		{expr: `root("a")`, code: "XPTY0004"},
		{expr: `innermost((r, 5))`, code: "XPTY0004"},
		{expr: `in-scope-prefixes(5)`, code: "XPTY0004"},
		{expr: `in-scope-prefixes(/)`, code: "XPTY0004"},
		{expr: `id("a", 5)`, code: "XPTY0004"},
		{expr: `lang("en", 5)`, code: "XPTY0004"},
		{expr: `has-children(5)`, code: "XPTY0004"},
		{expr: `node-name(1)`, code: "XPTY0004"},
		{expr: `path(5)`, code: "XPTY0004"},
		{expr: `generate-id(5)`, code: "XPTY0004"},
		{expr: `resolve-QName("a", 5)`, code: "XPTY0004"},
		{expr: `count(innermost((r, r/a)))`, want: "1"},
		{expr: `lang("en", r)`, want: "false"},
		// An array is atomized to its members.
		{expr: `string-join(["a", "b"], "-")`, want: "a-b"},
		{expr: `upper-case(["a"])`, want: "A"},
		{expr: `abs([3])`, want: "3"},
		{expr: `abs([1, 2])`, code: "XPTY0004"},
		{expr: `empty(abs([]))`, want: "true"},
		{expr: `abs([[3]])`, want: "3"},
	})
}

func TestArgCheckMapsArraysFunctions(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		{expr: `map:get("x", 1)`, code: "XPTY0004"},
		{expr: `map:size([1])`, code: "XPTY0004"},
		{expr: `array:size(map{})`, code: "XPTY0004"},
		{expr: `array:size(1)`, code: "XPTY0004"},
		{expr: `map:merge((map{1: 2}, 5))`, code: "XPTY0004"},
		{expr: `map:merge(map{1: 2}, 5)`, code: "XPTY0004"},
		{expr: `array:join(([1], 5))`, code: "XPTY0004"},
		{expr: `map:for-each(map{}, 5)`, code: "XPTY0004"},
		{expr: `array:for-each([1], 5)`, code: "XPTY0004"},
		{expr: `array:filter([1], ())`, code: "XPTY0004"},
		{expr: `array:fold-left([1], 0, "f")`, code: "XPTY0004"},
		{expr: `for-each((1, 2), "x")`, code: "XPTY0004"},
		{expr: `filter(1, 5)`, code: "XPTY0004"},
		{expr: `fold-left(1, 0, ())`, code: "XPTY0004"},
		{expr: `sort((3, 1), (), 5)`, code: "XPTY0004"},
		{expr: `sort((3, 1), ("a", "b"))`, code: "XPTY0004"},
		{expr: `apply(abs#1, 5)`, code: "XPTY0004"},
		{expr: `apply(5, [1])`, code: "XPTY0004"},
		{expr: `function-arity(5)`, code: "XPTY0004"},
		{expr: `function-name("abs")`, code: "XPTY0004"},
		{expr: `array:subarray([1, 2], "1")`, code: "XPTY0004"},
		// Atomizing a map is FOTY0013 (XPath 3.1 §2.4.2), not a type error.
		{expr: `map:remove(map{1: 2}, map{})`, code: "FOTY0013"},
		{expr: `apply(abs#1, [-1])`, want: "1"},
		{expr: `map:keys(map{1: 2})`, want: "1"},
		{expr: `map:get(map{1: 2}, 1)`, want: "2"},
		{expr: `map:contains(map{"a": 1}, xs:untypedAtomic("a"))`, want: "true"},
		{expr: `array:size([1, 2, 3])`, want: "3"},
		{expr: `array:get([7, 8], 2)`, want: "8"},
		{expr: `count(array:join(([1], [2, 3])))`, want: "1"},
		{expr: `array:size(array:join(([1], [2, 3])))`, want: "3"},
		{expr: `map:size(map:merge((map{1: 2}, map{3: 4})))`, want: "2"},
		{expr: `map:for-each(map{1: 2}, function($k, $v) { $k + $v })`, want: "3"},
		{expr: `string-join(for-each((1, 2), function($x) { $x * 2 }) ! string(), " ")`, want: "2 4"},
		{expr: `array:sort([3, 1, 2])?1`, want: "1"},
		// A map or array is a function item.
		{expr: `for-each((1), map{1: "one"})`, want: "one"},
		{expr: `for-each((1), [9, 8])`, want: "9"},
	})
}

// A dynamic call through a function item, a named function reference,
// fn:function-lookup and a partial application are checked like a direct call.
func TestArgCheckAllCallPaths(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		{expr: `upper-case#1(1)`, code: "XPTY0004"},
		{expr: `let $f := upper-case#1 return $f(1)`, code: "XPTY0004"},
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'upper-case'), 1)(1)`, code: "XPTY0004"},
		{expr: `let $f := substring("abc", ?) return $f("x")`, code: "XPTY0004"},
		{expr: `let $f := contains-token("a", "a", ?) return $f(())`, code: "XPTY0004"},
		{expr: `for-each(("a", 1), upper-case#1)`, code: "XPTY0004"},
		{expr: `apply(upper-case#1, [1])`, code: "XPTY0004"},
		{expr: `1 => upper-case()`, code: "XPTY0004"},
		{expr: `"abc" => upper-case()`, want: "ABC"},
		{expr: `let $f := upper-case#1 return $f("abc")`, want: "ABC"},
		{expr: `let $f := substring("abc", ?) return $f(2)`, want: "bc"},
		{expr: `fn:upper-case(1)`, code: "XPTY0004"},
		{expr: `Q{http://www.w3.org/2005/xpath-functions}upper-case(1)`, code: "XPTY0004"},
		{expr: `Q{http://www.w3.org/2005/xpath-functions/math}sin("a")`, code: "XPTY0004"},
		{expr: `Q{http://www.w3.org/2005/xpath-functions/map}size(1)`, code: "XPTY0004"},
		{expr: `Q{http://www.w3.org/2005/xpath-functions/array}size(1)`, code: "XPTY0004"},
		{expr: `map:size#1(1)`, code: "XPTY0004"},
		{expr: `array:size#1(1)`, code: "XPTY0004"},
		{expr: `r/s ! upper-case(.)`, want: "ABC"},
	})
}

// Arity errors stay XPST0017, ahead of any argument check, for every
// namespace.
func TestArgCheckArity(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		{expr: `abs()`, code: "XPST0017"},
		{expr: `abs(1, 2)`, code: "XPST0017"},
		{expr: `format-date(xs:date("2000-01-01"), "[Y]", "en")`, code: "XPST0017"},
		{expr: `math:pi(1)`, code: "XPST0017"},
		{expr: `map:size()`, code: "XPST0017"},
		{expr: `map:size(map{}, 1)`, code: "XPST0017"},
		{expr: `map:find(map{})`, code: "XPST0017"},
		{expr: `array:head()`, code: "XPST0017"},
		{expr: `array:sort([1], (), 5, 6)`, code: "XPST0017"},
		{expr: `concat("a")`, code: "XPST0017"},
		{expr: `concat("a", "b", "c", "d", "e")`, want: "abcde"},
		{expr: `concat(1, 2, 3, 4, 5, 6, 7)`, want: "1234567"},
		{expr: `concat("a", "b", (), "c", ())`, want: "abc"},
		{expr: `concat("a", "b", ("c", "d"))`, code: "XPTY0004"},
		{expr: `concat("a", "b", 1, map{})`, code: "FOTY0013"},
	})
}

// Under XSLT 1.0 backwards-compatible processing the function conversion
// rules of XPath 1.0 apply instead, and the strict checks are not made.
func TestArgCheckBackwardsCompatible(t *testing.T) {
	bc := func(c *Context) { c.BC10 = true }
	caRun(t, argSrc, bc, []caCase{
		{expr: `substring("abc", "2")`, want: "bc"},
		{expr: `contains(1, "1")`, want: "true"},
		{expr: `abs("-3")`, want: "3"},
	})
	// The same calls are errors outside it.
	caRun(t, argSrc, nil, []caCase{
		{expr: `substring("abc", "2")`, code: "XPTY0004"},
		{expr: `contains(1, "1")`, code: "XPTY0004"},
		{expr: `abs("-3")`, code: "XPTY0004"},
	})
}

// The error message names the function, the argument and the declared type.
func TestArgCheckMessage(t *testing.T) {
	_, err := caEval(t, `contains-token("a", "a", ())`, "", nil)
	if err == nil {
		t.Fatal("no error")
	}
	for _, want := range []string{"XPTY0004", "contains-token", "argument 3", "$collation", "xs:string", "empty sequence"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q lacks %q", err, want)
		}
	}
}

// An argument that already matches is not copied or converted: the same
// slice reaches the implementation.
func TestArgCheckNoCopyWhenMatching(t *testing.T) {
	ensureCatalog()
	f := sigFn["substring"].form(3)
	args := []Object{NewString("abc"), NewInteger(1), NewDouble(2)}
	out, err := f.convert("substring", args, &Context{})
	if err != nil {
		t.Fatal(err)
	}
	if &out[0] != &args[0] {
		t.Error("matching arguments were copied")
	}
	// A replaced value never writes into the caller's slice.
	args = []Object{NewString("abc"), NewUntyped("2")}
	out, err = f.convert("substring", append(args, NewDouble(1)), &Context{})
	if err != nil {
		t.Fatal(err)
	}
	if a, _ := out[1].(*Atomic); a == nil || a.T != XSdouble {
		t.Errorf("untypedAtomic not cast to xs:double: %#v", out[1])
	}
	if a, _ := args[1].(*Atomic); a.T != XSuntypedAtomic {
		t.Error("caller's argument slice was modified")
	}
}

// A function item made from a built-in carries the catalogue's declared
// signature, for every namespace.
func TestBuiltinFunctionItemSignatures(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		{expr: `abs#1 instance of function(xs:numeric?) as xs:numeric?`, want: "true"},
		{expr: `abs#1 instance of function(xs:string) as xs:string`, want: "false"},
		{expr: `math:sqrt#1 instance of function(xs:double?) as xs:double?`, want: "true"},
		{expr: `math:pi#0 instance of function() as xs:double`, want: "true"},
		{expr: `map:get#2 instance of function(map(*), xs:anyAtomicType) as item()*`, want: "true"},
		{expr: `array:size#1 instance of function(array(*)) as xs:integer`, want: "true"},
		{expr: `substring#2 instance of function(xs:string?, xs:double) as xs:string`, want: "true"},
		{expr: `substring#3 instance of function(xs:string?, xs:double) as xs:string`, want: "false"},
		{expr: `concat#3 instance of function(xs:anyAtomicType?, xs:anyAtomicType?, xs:anyAtomicType?) as xs:string`, want: "true"},
		{expr: `filter#2 instance of function(item()*, function(item()) as xs:boolean) as item()*`, want: "true"},
	})
}

// What the engine's own functions and operators return is accepted by the
// parameters they feed: integers are real xs:integer values, not doubles.
func TestArgCheckResultsFeedParameters(t *testing.T) {
	caRun(t, argSrc, nil, []caCase{
		{expr: `array:get([7, 8], position())`, want: "7"},
		{expr: `array:get([7, 8], last() + 1)`, want: "8"},
		{expr: `array:get([7, 8], count(r/a))`, want: "7"},
		{expr: `array:get([7, 8], string-length('xy'))`, want: "8"},
		{expr: `array:get([7, 8], array:size([1]))`, want: "7"},
		{expr: `array:get([7, 8], map:size(map{1: 1}))`, want: "7"},
		{expr: `array:get([7, 8], index-of((5, 6), 6))`, want: "8"},
		{expr: `array:get([7, 8], 3 idiv 2)`, want: "7"},
		{expr: `array:get([7, 8], 5 mod 3)`, want: "8"},
		{expr: `array:get([7, 8], xs:integer('2'))`, want: "8"},
		{expr: `array:get([7, 8], xs:int(2))`, want: "8"},
		{expr: `array:get([7, 8], sum((1, 1)))`, want: "8"},
		{expr: `array:get([7, 8], max((1, 2)))`, want: "8"},
		{expr: `array:get([7, 8], (1 to 3)[2])`, want: "8"},
		{expr: `array:get([7, 8], abs(-2))`, want: "8"},
		{expr: `array:get([7, 8], round(1.5))`, code: "XPTY0004"},
		{expr: `array:get([7, 8], 2 * 1)`, want: "8"},
		{expr: `array:get([7, 8, 9], r/a)`, want: "9"},
		{expr: `format-integer(position(), '1')`, want: "1"},
		{expr: `format-integer(count(r/*), '1')`, want: "4"},
		{expr: `substring('abcdef', string-length('ab'), count(r/*))`, want: "bcde"},
		{expr: `round(2.567, count(r/a) + 1)`, want: "2.57"},
	})
}

// simpleDouble is a shortcut for the general untypedAtomic → xs:double cast:
// where it answers, the answer is the cast's.
func TestSimpleDoubleAgreesWithCast(t *testing.T) {
	for _, s := range []string{
		"0", "1", "-1", "+1", "007", "1.", ".5", "-.5", "+.5", "-0", "0.0", "3.14159", "123456789012345678901234567890",
		"0.000000000000000000000000000001", " 7 ", "\t8\n", "12345678901234567890.12345678901234567890",
		"", ".", "+", "-", "1.2.3", "1e3", "1E-2", "INF", "-INF", "NaN", "abc", "1 2", "٣", "0x10", "1_0", "--1",
	} {
		want, werr := castInContext(NewUntyped(s), XSdouble, &Context{})
		got, ok := simpleDouble(s)
		if !ok {
			continue // handled by the general cast
		}
		if werr != nil {
			t.Errorf("%q: simpleDouble accepts it, the cast says %v", s, werr)
			continue
		}
		wa := want.(*Atomic)
		if g := NewDouble(got); g.Lexical() != wa.Lexical() || math.Signbit(got) != math.Signbit(wa.Float()) {
			t.Errorf("%q: simpleDouble %s, cast %s", s, g.Lexical(), wa.Lexical())
		}
	}
}
