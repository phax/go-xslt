package xpath

import "testing"

// hofHost is a FuncResolver that serves one function in the standard
// function namespace beyond the package catalog, and declares it through
// HostFuncCatalog so it is reachable as a function item.
type hofHost struct{}

func (hofHost) ResolveFunc(prefix, local string, args []Object, ctx *Context) (Object, bool, error) {
	if local == "host-answer" {
		return NewInteger(42), true, nil
	}
	return nil, false, nil
}

func (hofHost) HostFuncArity(uri, local string) (int, int, bool) {
	switch local {
	case "host-answer":
		return 0, 0, true
	case "host-declared-only":
		// declared, but ResolveFunc does not serve it
		return 1, 1, true
	}
	return 0, 0, false
}

// TestHOFSpec covers fn:function-name, fn:function-arity,
// fn:function-lookup and fn:apply (F&O 3.1 §16.1/§16.2) with the REC's
// examples, function items from every built-in namespace, and the
// documented error codes.
func TestHOFSpec(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		// §16.1.2 fn:function-name
		{expr: `local-name-from-QName(function-name(fn:substring#2))`, want: "substring"},
		{expr: `namespace-uri-from-QName(function-name(fn:substring#2))`, want: "http://www.w3.org/2005/xpath-functions"},
		{expr: `namespace-uri-from-QName(function-name(math:pi#0))`, want: "http://www.w3.org/2005/xpath-functions/math"},
		{expr: `namespace-uri-from-QName(function-name(map:get#2))`, want: "http://www.w3.org/2005/xpath-functions/map"},
		{expr: `namespace-uri-from-QName(function-name(array:size#1))`, want: "http://www.w3.org/2005/xpath-functions/array"},
		{expr: `namespace-uri-from-QName(function-name(xs:integer#1))`, want: "http://www.w3.org/2001/XMLSchema"},
		{expr: `local-name-from-QName(function-name(xs:integer#1))`, want: "integer"},
		{expr: `function-name(function($node){count($node/*)})`, want: "()"},
		{expr: `function-name(substring(?, 1, 1))`, want: "()"},
		{expr: `function-name(map{})`, want: "()"},
		{expr: `function-name((abs#1, abs#1))`, err: "XPTY0004"},
		{expr: `function-name(1)`, err: "XPTY0004"},

		// §16.1.3 fn:function-arity
		{expr: `function-arity(fn:substring#2)`, want: "2"},
		{expr: `function-arity(function($node){name($node)})`, want: "1"},
		{expr: `let $initial := fn:substring(?, 1, 1) return function-arity($initial)`, want: "1"},
		{expr: `function-arity(map{})`, want: "1"},
		{expr: `function-arity([1, 2])`, want: "1"},
		{expr: `function-arity(())`, err: "XPTY0004"},
		{expr: `function-arity('abs')`, err: "XPTY0004"},

		// §16.1.1 fn:function-lookup
		{expr: `function-lookup(xs:QName('fn:substring'), 2)('abcd', 2)`, want: "bcd"},
		{expr: `function-lookup(xs:QName('fn:abs'), 1)(-3)`, want: "3"},
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions/math', 'sqrt'), 1)(4)`, want: "2"},
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions/map', 'size'), 1)(map{1:2})`, want: "1"},
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions/array', 'size'), 1)([1, 2, 3])`, want: "3"},
		{expr: `function-lookup(QName('http://www.w3.org/2001/XMLSchema', 'integer'), 1)('12') + 1`, want: "13"},
		{expr: `function-arity(function-lookup(xs:QName('fn:concat'), 4))`, want: "4"},
		// no such function, wrong arity, no namespace: the empty sequence
		{expr: `function-lookup(xs:QName('fn:abs'), 3)`, want: "()"},
		{expr: `function-lookup(QName('', 'abs'), 1)`, want: "()"},
		{expr: `function-lookup(QName('http://example.com/none', 'f'), 0)`, want: "()"},
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'no-such-function'), 1)`, want: "()"},
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions/math', 'no-such'), 1)`, want: "()"},
		// $name as xs:QName and $arity as xs:integer are required singletons
		{expr: `function-lookup((), 1)`, err: "XPTY0004"},
		{expr: `function-lookup('fn:abs', 1)`, err: "XPTY0004"},
		{expr: `function-lookup(xs:QName('fn:abs'), ())`, err: "XPTY0004"},
		{expr: `function-lookup((xs:QName('fn:abs'), xs:QName('fn:abs')), 1)`, err: "XPTY0004"},

		// §16.2.1 fn:apply
		{expr: `apply(fn:concat#3, ["a", "b", "c"])`, want: "abc"},
		{expr: `apply(function($a, $b){$a * $b}, [6, 7])`, want: "42"},
		{expr: `apply(map{'k': 'v'}, ['k'])`, want: "v"},
		{expr: `apply(abs#1, [1, 2])`, err: "FOAP0001"},
		{expr: `apply(abs#1, [])`, err: "FOAP0001"},
		{expr: `apply(abs#1, 1)`, err: "XPTY0004"},
		{expr: `apply(1, [1])`, err: "XPTY0004"},
	})
}

// TestHOFHostCatalog checks the HostFuncCatalog contract: a host function in
// the standard namespace is reachable through fn:function-lookup only when
// the host declares it.
func TestHOFHostCatalog(t *testing.T) {
	coreRun(t, "", func(c *Context) { c.Funcs = hofHost{} }, []coreCase{
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'host-answer'), 0)()`, want: "42"},
		{expr: `local-name-from-QName(function-name(function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'host-answer'), 0)))`, want: "host-answer"},
		// outside the declared arity range: no function
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'host-answer'), 1)`, want: "()"},
		// not declared at all
		{expr: `function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'host-unknown'), 0)`, want: "()"},
	})
	// A declared function the host then fails to serve is an error when called.
	p, err := Parse(`function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'host-declared-only'), 1)(1)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Eval(&Context{Funcs: hofHost{}}); err == nil {
		t.Errorf("calling an unserved host function item succeeded, want an error")
	}
}
