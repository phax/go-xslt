package xpath

import "testing"

// Function arities come from the F&O 3.1 signatures; a call or reference at
// an arity no signature defines is XPST0017 (XPath 3.1 §3.1.5), and
// fn:function-lookup at such an arity returns the empty sequence (§16.1.2).
func TestArityDirectCalls(t *testing.T) {
	numRunErr(t, []numErrCase{
		{"abs()", "XPST0017"},
		{"abs(1, 2)", "XPST0017"},
		{"string(1, 2)", "XPST0017"},
		{"math:pi(1)", "XPST0017"},
		{"math:atan2(1)", "XPST0017"},
		{"concat('a')", "XPST0017"},
	}, nil)
	numRun(t, []numCase{
		{"concat('a', 'b', 'c', 'd', 'e')", "abcde"},
		{"string-join(('a', 'b'))", "ab"},
		{"math:atan2(0e0, 1e0)", "0"},
	}, nil)
}

func TestArityFunctionLookup(t *testing.T) {
	numRun(t, []numCase{
		// format-date exists only with 2 and 5 arguments.
		{"exists(function-lookup(xs:QName('fn:format-date'), 2))", "true"},
		{"exists(function-lookup(xs:QName('fn:format-date'), 3))", "false"},
		{"exists(function-lookup(xs:QName('fn:format-date'), 4))", "false"},
		{"exists(function-lookup(xs:QName('fn:format-date'), 5))", "true"},
		{"exists(function-lookup(xs:QName('fn:round'), 2))", "true"},
		{"exists(function-lookup(xs:QName('fn:round'), 3))", "false"},
		{"exists(function-lookup(xs:QName('math:atan2'), 2))", "true"},
		{"exists(function-lookup(xs:QName('math:atan2'), 1))", "false"},
		{"exists(function-lookup(xs:QName('xs:integer'), 1))", "true"},
		{"exists(function-lookup(xs:QName('xs:integer'), 2))", "false"},
		{"exists(function-lookup(xs:QName('fn:concat'), 7))", "true"},
		{"function-lookup(xs:QName('fn:concat'), 3)('a', 'b', 'c')", "abc"},
	}, nil)
}

func TestArityStdFuncArity(t *testing.T) {
	for _, c := range []struct {
		ns, local string
		lo, hi    int
		known     bool
	}{
		{nsFn, "abs", 1, 1, true},
		{nsFn, "round", 1, 2, true},
		{nsFn, "concat", 2, arityVariadic, true},
		{nsMath, "pi", 0, 0, true},
		{nsMath, "atan2", 2, 2, true},
		{nsMap, "merge", 1, 2, true},
		{nsMap, "get", 2, 2, true},
		{nsArray, "join", 1, 1, true},
		{nsArray, "for-each", 2, 2, true},
		{nsXS, "integer", 1, 1, true},
		{nsXS, "NMTOKENS", 1, 1, true},
		{nsFn, "no-such-function", 0, 0, false},
		{nsMath, "no-such-function", 0, 0, false},
		{nsMap, "no-such-function", 0, 0, false},
		{nsArray, "no-such-function", 0, 0, false},
		{nsXS, "noSuchType", 0, 0, false},
		{"http://example.com/other", "f", 0, 0, false},
	} {
		lo, hi, known := StdFuncArity(c.ns, c.local)
		if known != c.known || (known && (lo != c.lo || hi != c.hi)) {
			t.Errorf("StdFuncArity(%s, %s) = %d, %d, %v; want %d, %d, %v",
				c.ns, c.local, lo, hi, known, c.lo, c.hi, c.known)
		}
	}
}

func TestArityStdArityOK(t *testing.T) {
	for _, c := range []struct {
		ns, local string
		n         int
		want      bool
	}{
		{nsFn, "format-date", 2, true},
		{nsFn, "format-date", 3, false},
		{nsFn, "format-date", 4, false},
		{nsFn, "format-date", 5, true},
		{nsFn, "format-dateTime", 3, false},
		{nsFn, "format-time", 5, true},
		{nsFn, "concat", 1, false},
		{nsFn, "concat", 100, true},
		{nsFn, "abs", 0, false},
		{nsFn, "abs", 1, true},
		{nsMath, "pi", 0, true},
		{nsMath, "pi", 1, false},
		{nsMap, "merge", 2, true},
		{nsMap, "merge", 3, false},
		{nsArray, "size", 1, true},
		{nsXS, "integer", 1, true},
		{nsXS, "integer", 2, false},
		{nsFn, "no-such-function", 1, false},
	} {
		if got := StdArityOK(c.ns, c.local, c.n); got != c.want {
			t.Errorf("StdArityOK(%s, %s, %d) = %v, want %v", c.ns, c.local, c.n, got, c.want)
		}
	}
}

// Named function references carry the declared F&O signature, so function
// tests judge them by it (XPath 3.1 §2.5.6.3: parameter types are
// contravariant, the result type covariant).
func TestAritySignedReferences(t *testing.T) {
	numRun(t, []numCase{
		{"name#1 instance of function(node()?) as xs:string", "true"},
		{"name#1 instance of function(item()) as xs:string", "false"},
		{"abs#1 instance of function(xs:numeric?) as xs:numeric?", "true"},
		{"abs#1 instance of function(xs:integer) as xs:numeric?", "true"},
		{"abs#1 instance of function(xs:string) as xs:numeric?", "false"},
		{"count#1 instance of function(item()*) as xs:integer", "true"},
		{"count#1 instance of function(item()*) as xs:string", "false"},
		{"true#0 instance of function() as xs:boolean", "true"},
		{"concat#2 instance of function(xs:anyAtomicType?, xs:anyAtomicType?) as xs:string", "true"},
		{"substring#2 instance of function(xs:string?, xs:double) as xs:string", "true"},
		{"filter#2 instance of function(item()*, function(item()) as xs:boolean) as item()*", "true"},
		{"filter#2 instance of function(function(*), item()*) as item()*", "false"},
		{"math:pi#0 instance of function() as xs:double", "true"},
		{"function-arity(math:pi#0)", "0"},
	}, nil)
}
