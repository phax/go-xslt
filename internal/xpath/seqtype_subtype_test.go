package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// tyRunCtx is tyRun with a chosen source document and a hook to adjust the
// static/dynamic context (namespaces, default type namespace, …).
func tyRunCtx(expr, xml string, adjust func(*Context)) string {
	p, err := Parse(expr)
	if err != nil {
		return "!" + tyErrCode(err)
	}
	doc, err := xmltree.Parse(xml)
	if err != nil {
		return "!xml " + err.Error()
	}
	ctx := &Context{Node: doc, Pos: 1, Size: 1}
	if adjust != nil {
		adjust(ctx)
	}
	v, err := p.Eval(ctx)
	if err != nil {
		return "!" + tyErrCode(err)
	}
	return ToString(v)
}

// TestTyFunctionSubtyping checks `instance of` against TypedFunctionTests,
// which XPath 3.1 §2.5.5.7 defines by the subtype judgement of §2.5.6.2:
// function(Pa…) as Ra is a subtype of function(Pb…) as Rb iff the arities
// agree, every Pb is a subtype of the corresponding Pa (contravariance) and
// Ra is a subtype of Rb (covariance). The SequenceType judgement (§2.5.6.1)
// compares cardinality ranges as well as item types.
func TestTyFunctionSubtyping(t *testing.T) {
	tyCheck(t, []tyCase{
		// Plain atomic parameters and results.
		{`function($x as xs:integer) as xs:integer {$x} instance of function(xs:integer) as xs:integer`, "true"},
		{`function($x as xs:decimal) as xs:integer {1} instance of function(xs:integer) as xs:decimal`, "true"},
		{`function($x as xs:integer) as xs:integer {$x} instance of function(xs:decimal) as xs:integer`, "false"},
		{`function($x as xs:integer) as xs:decimal {$x} instance of function(xs:integer) as xs:integer`, "false"},
		{`function($x as xs:integer) as xs:integer {$x} instance of function(xs:integer, xs:integer) as xs:integer`, "false"},
		{`function($x as item()) as xs:integer {1} instance of function(xs:integer) as item()*`, "true"},
		{`function($x as xs:integer) as xs:integer {1} instance of function(node()) as item()*`, "false"},
		{`function($x as xs:integer) as xs:integer {1} instance of function(*)`, "true"},
		// Cardinality ranges (§2.5.6.1).
		{`function($x as xs:integer?) as xs:integer+ {1} instance of function(xs:integer) as xs:integer*`, "true"},
		{`function($x as xs:integer) as xs:integer* {1} instance of function(xs:integer) as xs:integer+`, "false"},
		{`function($x as xs:integer) as xs:integer? {1} instance of function(xs:integer?) as xs:integer?`, "false"},
		{`function($x as xs:integer+) as xs:integer {1} instance of function(xs:integer*) as xs:integer`, "false"},
		{`function() as empty-sequence() {()} instance of function() as xs:integer?`, "true"},
		{`function() as empty-sequence() {()} instance of function() as xs:integer*`, "true"},
		{`function() as empty-sequence() {()} instance of function() as xs:integer`, "false"},
		{`function() as empty-sequence() {()} instance of function() as empty-sequence()`, "true"},
		{`function() as xs:integer {1} instance of function() as empty-sequence()`, "false"},
		// xs:error has an empty value space: a subtype of every atomic type
		// (§2.5.6.2), while nothing is a subtype of xs:error but itself.
		{`function() as xs:error? {()} instance of function() as xs:string?`, "true"},
		{`function() as xs:string? {()} instance of function() as xs:error?`, "false"},
		// xs:numeric is the union of xs:double, xs:float and xs:decimal.
		{`function() as xs:numeric {1} instance of function() as xs:anyAtomicType`, "true"},
		{`function() as xs:numeric {1} instance of function() as xs:numeric`, "true"},
		{`function() as xs:numeric {1} instance of function() as xs:decimal`, "false"},
		{`function() as xs:integer {1} instance of function() as xs:numeric`, "true"},
		// Node kind tests as parameter types.
		{`function($e as element()) {1} instance of function(element(a)) as item()*`, "true"},
		{`function($e as element(a)) {1} instance of function(element()) as item()*`, "false"},
		{`function($e as element(a)) {1} instance of function(element(a)) as item()*`, "true"},
		{`function($e as element(a)) {1} instance of function(element(b)) as item()*`, "false"},
		{`function($e as element(*)) {1} instance of function(element(b)) as item()*`, "true"},
		{`function($n as node()) {1} instance of function(text()) as item()*`, "true"},
		{`function($n as text()) {1} instance of function(node()) as item()*`, "false"},
		{`function($n as text()) {1} instance of function(comment()) as item()*`, "false"},
		{`function($n as node()) {1} instance of function(xs:integer) as item()*`, "false"},
		{`function($d as document-node()) {1} instance of function(document-node(element(a))) as item()*`, "true"},
		{`function($d as document-node(element(a))) {1} instance of function(document-node()) as item()*`, "false"},
		{`function($d as document-node(element(a))) {1} instance of function(document-node(element(a))) as item()*`, "true"},
		{`function($d as document-node(element(a))) {1} instance of function(document-node(element(b))) as item()*`, "false"},
		// element(a) admits nilled elements, element(a, xs:anyType) does not:
		// element(a, xs:anyType) ⊆ element(a), but not the other way round.
		{`function($e as element(a)) {1} instance of function(element(a, xs:anyType)) as item()*`, "true"},
		{`function($e as element(a, xs:anyType)) {1} instance of function(element(a)) as item()*`, "false"},
		// Map types (§2.5.6.2: map(K1,V1) ⊆ map(K2,V2) iff K1 ⊆ K2, V1 ⊆ V2;
		// map(*) ≡ map(xs:anyAtomicType, item()*)).
		{`function($m as map(*)) {1} instance of function(map(xs:string, xs:integer)) as item()*`, "true"},
		{`function($m as map(xs:string, xs:integer)) {1} instance of function(map(*)) as item()*`, "false"},
		{`function($m as map(xs:decimal, xs:integer)) {1} instance of function(map(xs:integer, xs:integer)) as item()*`, "true"},
		{`function($m as map(xs:integer, xs:integer)) {1} instance of function(map(xs:decimal, xs:integer)) as item()*`, "false"},
		{`function($m as map(xs:string, xs:decimal)) {1} instance of function(map(xs:string, xs:integer)) as item()*`, "true"},
		{`function($m as map(xs:anyAtomicType, item()*)) {1} instance of function(map(*)) as item()*`, "true"},
		{`function($m as map(*)) {1} instance of function(array(*)) as item()*`, "false"},
		// Array types (array(*) ≡ array(item()*)).
		{`function($a as array(*)) {1} instance of function(array(xs:integer)) as item()*`, "true"},
		{`function($a as array(xs:integer)) {1} instance of function(array(*)) as item()*`, "false"},
		{`function($a as array(item()*)) {1} instance of function(array(*)) as item()*`, "true"},
		{`function($a as array(xs:decimal)) {1} instance of function(array(xs:integer)) as item()*`, "true"},
		{`function($a as array(xs:integer)) {1} instance of function(array(xs:decimal)) as item()*`, "false"},
		{`function($a as array(*)) {1} instance of function(map(*)) as item()*`, "false"},
		// Maps and arrays are functions: map(*) ⊆ function(xs:anyAtomicType)
		// as item()*, array(*) ⊆ function(xs:integer) as item()*.
		{`function($f as function(xs:anyAtomicType) as item()*) {1} instance of function(map(*)) as item()*`, "true"},
		{`function($f as function(xs:integer) as item()*) {1} instance of function(array(*)) as item()*`, "true"},
		{`function($f as function(xs:string) as item()*) {1} instance of function(array(*)) as item()*`, "false"},
		{`function($f as function(xs:anyAtomicType) as xs:integer?) {1} instance of function(map(xs:string, xs:integer)) as item()*`, "true"},
		{`function($f as function(xs:integer) as xs:integer) {1} instance of function(array(xs:integer)) as item()*`, "true"},
		{`function($f as function(*)) {1} instance of function(map(*)) as item()*`, "true"},
		{`function($f as function(xs:anyAtomicType) as item()*) {1} instance of function(xs:integer) as item()*`, "false"},
		// Function-typed parameters themselves.
		{`function($f as function(*)) {1} instance of function(function(xs:integer) as xs:integer) as item()*`, "true"},
		{`function($f as function(xs:integer) as xs:integer) {1} instance of function(function(*)) as item()*`, "false"},
		{`function($f as function(xs:decimal) as xs:integer) {1} instance of function(function(xs:integer) as xs:integer) as item()*`, "false"},
		{`function($f as function(xs:integer) as xs:integer) {1} instance of function(function(xs:decimal) as xs:integer) as item()*`, "true"},
		{`function($f as function(xs:integer) as xs:integer) {1} instance of function(function(xs:integer, xs:integer) as xs:integer) as item()*`, "false"},
	})
}

// TestTyItemTypeMatching checks `instance of` on maps, arrays, functions and
// occurrence indicators (XPath 3.1 §2.5.5).
func TestTyItemTypeMatching(t *testing.T) {
	tyCheck(t, []tyCase{
		// §2.5.5.8 MapTest.
		{`map{'a':1} instance of map(xs:string, xs:integer)`, "true"},
		{`map{1:1} instance of map(xs:string, xs:integer)`, "false"},
		{`map{'a':'x'} instance of map(xs:string, xs:integer)`, "false"},
		{`map{'a':(1,2)} instance of map(xs:string, xs:integer+)`, "true"},
		{`map{} instance of map(xs:string, xs:integer)`, "true"},
		{`map{'a':1} instance of map(*)`, "true"},
		{`1 instance of map(*)`, "false"},
		{`[1] instance of map(*)`, "false"},
		// §2.5.5.9 ArrayTest.
		{`[1,2] instance of array(xs:integer)`, "true"},
		{`[1,'a'] instance of array(xs:integer)`, "false"},
		{`[(1,2)] instance of array(xs:integer)`, "false"},
		{`[(1,2)] instance of array(xs:integer+)`, "true"},
		{`[] instance of array(xs:string)`, "true"},
		{`map{} instance of array(*)`, "false"},
		{`'a' instance of array(*)`, "false"},
		// §2.5.5.7 FunctionTest: maps and arrays are function items.
		{`map{1:2} instance of function(*)`, "true"},
		{`[1] instance of function(*)`, "true"},
		{`abs#1 instance of function(*)`, "true"},
		{`1 instance of function(*)`, "false"},
		{`map{1:2} instance of function(xs:anyAtomicType) as item()*`, "true"},
		{`map{1:2} instance of function(xs:anyAtomicType, xs:anyAtomicType) as item()*`, "false"},
		{`[1] instance of function(xs:integer) as item()*`, "true"},
		{`[1] instance of function(xs:string) as item()*`, "false"},
		{`'x' instance of function(xs:integer) as item()*`, "false"},
		{`abs#1 instance of function(item(), item()) as item()*`, "false"},
		// Occurrence indicators (§2.5.4).
		{`(1,2) instance of xs:integer?`, "false"},
		{`() instance of xs:integer+`, "false"},
		{`() instance of xs:integer?`, "true"},
		{`(1,2) instance of xs:integer`, "false"},
		{`(1,'a') instance of xs:integer*`, "false"},
		{`1 instance of empty-sequence()`, "false"},
		// ParenthesizedItemType (XPath 3.1 grammar rule [83]).
		{`1 instance of (xs:integer)`, "true"},
		{`(1,2) instance of (xs:integer)+`, "true"},
		{`map{} instance of (map(*))`, "true"},
		{`1 instance of item()`, "true"},
		{`(1, map{}, [], abs#1) instance of item()+`, "true"},
		// Atomic type hierarchy (XSD Part 2 §3).
		{`xs:byte(1) instance of xs:short`, "true"},
		{`xs:short(1) instance of xs:byte`, "false"},
		{`xs:dayTimeDuration('PT1S') instance of xs:duration`, "true"},
		{`xs:duration('PT1S') instance of xs:dayTimeDuration`, "false"},
		{`xs:dateTimeStamp('2000-01-01T00:00:00Z') instance of xs:dateTime`, "true"},
		{`xs:ID('a') instance of xs:NCName`, "true"},
		{`xs:language('en') instance of xs:token`, "true"},
		{`xs:unsignedByte(1) instance of xs:unsignedLong`, "true"},
		{`xs:unsignedByte(1) instance of xs:long`, "false"},
		{`1.5 instance of xs:numeric`, "true"},
		{`'1' instance of xs:numeric`, "false"},
		{`1 instance of xs:error`, "false"},
		// treat as raises XPDY0050 on a mismatch (§3.16.5).
		{`1 treat as xs:string`, "!XPDY0050"},
		{`(1 treat as xs:decimal) + 1`, "2"},
		// Unknown types are static errors XPST0051 (§3.14 / §2.5.4); an
		// unprefixed built-in name is only legal under an XSD default type
		// namespace, which the default static context does not have.
		{`1 instance of xs:doesNotExist`, "!XPST0051"},
		{`1 instance of nosuchtype`, "!XPST0051"},
		{`1 instance of integer`, "!XPST0051"},
		{`map{} instance of map(integer, xs:string)`, "!XPST0051"},
		{`map{} instance of map(xs:nope, xs:string)`, "!XPST0051"},
		{`map{} instance of map(nope, xs:string)`, "!XPST0051"},
		{`1 instance of Q{http://www.w3.org/2001/XMLSchema}integer`, "true"},
		{`1 instance of Q{http://www.w3.org/2001/XMLSchema}nope`, "!XPST0051"},
	})
}

// TestTyBareAndPrefixedTypeNames checks the static context's role in type
// name resolution: an unprefixed type name denotes a built-in only when the
// default element/type namespace is the XSD namespace (XPath 3.1 §2.1.1),
// and a prefix without an in-scope binding is XPST0081.
func TestTyBareAndPrefixedTypeNames(t *testing.T) {
	xsDefault := func(c *Context) { c.DefaultElemNS = nsXS }
	nsDefault := func(c *Context) { c.NS = testNSMap{"": nsXS, "my": "urn:my"} }
	onlyMy := func(c *Context) { c.NS = testNSMap{"my": "urn:my"} }
	cases := []struct {
		expr   string
		adjust func(*Context)
		want   string
	}{
		{`1 instance of integer`, xsDefault, "true"},
		{`1 instance of integer`, nsDefault, "true"},
		{`map{1:'a'} instance of map(integer, string)`, xsDefault, "true"},
		{`function($x as integer) as string {'a'} instance of function(integer) as string`, xsDefault, "true"},
		{`1 cast as string`, xsDefault, "1"},
		{`1 instance of integer`, onlyMy, "!XPST0051"},
		{`[1] instance of array(integer)`, onlyMy, "!XPST0051"},
		{`1 instance of foo:bar`, onlyMy, "!XPST0081"},
		{`1 cast as foo:integer`, onlyMy, "!XPST0081"},
		{`function($x as foo:integer) {1}`, onlyMy, "!XPST0081"},
		{`1 instance of function(foo:integer) as item()*`, onlyMy, "!XPST0081"},
		{`1 instance of map(xs:string, foo:integer)`, onlyMy, "!XPST0081"},
		{`1 instance of array(foo:integer)`, onlyMy, "!XPST0081"},
		{`1 instance of xs:integer`, onlyMy, "true"},
	}
	for _, c := range cases {
		if got := tyRunCtx(c.expr, "<r/>", c.adjust); got != c.want {
			t.Errorf("%s\n  got  %q\n  want %q", c.expr, got, c.want)
		}
	}
}

// TestTyFunctionCoercion covers the function conversion rules (XPath 3.1
// §3.1.5.2) applied to arguments of inline functions with declared parameter
// types: atomization, xs:untypedAtomic casting, numeric promotion, URI
// promotion and function coercion (§3.1.5.3).
func TestTyFunctionCoercion(t *testing.T) {
	tyCheck(t, []tyCase{
		// Atomization + casting of xs:untypedAtomic.
		{`function($x as xs:integer) {$x + 1}(xs:untypedAtomic('4'))`, "5"},
		{`function($x as xs:integer*) {sum($x)}([1, 2])`, "3"},
		{`function($x as xs:anyAtomicType) {$x instance of xs:untypedAtomic}(xs:untypedAtomic('a'))`, "true"},
		{`function($x as xs:string) {$x}(xs:untypedAtomic('a')) instance of xs:string`, "true"},
		// Numeric promotion: decimal → float/double, float → double; never
		// double → float.
		{`function($x as xs:double) {$x instance of xs:double}(1)`, "true"},
		{`function($x as xs:double) {$x instance of xs:double}(xs:float(1))`, "true"},
		{`function($x as xs:float) {$x instance of xs:float}(1.5)`, "true"},
		{`function($x as xs:float) {$x}(1e0)`, "!XPTY0004"},
		{`function($x as xs:decimal) {$x}(1e0)`, "!XPTY0004"},
		// URI promotion: xs:anyURI → xs:string.
		{`function($x as xs:string) {$x instance of xs:string}(xs:anyURI('a'))`, "true"},
		// No other conversion: a string is not an integer.
		{`function($x as xs:integer) {$x}('1')`, "!XPTY0004"},
		{`function($x as xs:integer) {$x}((1,2))`, "!XPTY0004"},
		{`function($x as xs:integer) {$x}(())`, "!XPTY0004"},
		{`function($x as xs:integer) {$x}(map{})`, "!*"},
		// Function coercion (§3.1.5.3): any function item of the right
		// arity is accepted; maps and arrays are arity-one functions.
		{`function($g as function(xs:string) as xs:integer) {$g('abc')}(string-length#1)`, "3"},
		{`function($g as function(xs:anyAtomicType) as item()*) {$g('a')}(map{'a': 5})`, "5"},
		{`function($g as function(xs:integer) as item()*) {$g(2)}([7, 8])`, "8"},
		{`function($g as function(xs:integer, xs:integer) as item()*) {1}(abs#1)`, "!XPTY0004"},
		{`function($g as function(*)) {1}((abs#1, abs#1))`, "!XPTY0004"},
		{`function($g as function(*)) {1}(())`, "!XPTY0004"},
		{`function($g as function(*)?) {1}((abs#1, abs#1))`, "!XPTY0004"},
		{`function($g as function(*)+) {1}(())`, "!XPTY0004"},
		{`function($g as function(*)) {1}(1)`, "!XPTY0004"},
		{`count(function($g as function(*)*) {$g}((abs#1, map{}, [])))`, "3"},
		// A supplied function that is a subtype is passed through unchanged.
		{`function($g as function(xs:integer) as xs:integer) {$g(2)}(function($x as xs:decimal) as xs:integer {xs:integer($x * 2)})`, "4"},
		// …otherwise it is wrapped: arguments are converted to the expected
		// parameter types and the result to the expected return type.
		{`function($g as function(xs:integer) as xs:double) {$g(2)}(function($x as xs:decimal) as xs:float {$x * 1.5})`, "3"},
		{`function($g as function(xs:integer) as xs:double) {$g(2) instance of xs:double}(function($x as xs:decimal) as xs:float {$x * 1.5})`, "true"},
		// higher-order-functions-066: an xs:float-typed function bound to
		// function(xs:double) as xs:double fails when called with a double.
		{`function($g as function(xs:double) as xs:double) {$g(2.5e0)}(function($x as xs:float) as xs:float {$x})`, "!XPTY0004"},
		{`function($g as function(xs:string) as xs:string) {$g('ab')}(string-length#1)`, "!XPTY0004"},
	})
}

// TestTyCoerceAndParseAPI covers the exported SequenceType helpers that hosts
// use with @as attribute text.
func TestTyCoerceAndParseAPI(t *testing.T) {
	// CoerceToDeclaredType applies the function conversion rules.
	if v, ok := CoerceToDeclaredType("xs:double", NewInteger(3)); !ok || Items(v)[0].(*Atomic).T != XSdouble {
		t.Errorf("integer → xs:double promotion failed: %v %v", v, ok)
	}
	if v, ok := CoerceToDeclaredType("xs:integer*", Sequence{NewInteger(1), NewInteger(2)}); !ok || len(Items(v)) != 2 {
		t.Errorf("integer* match failed: %v %v", v, ok)
	}
	if _, ok := CoerceToDeclaredType("xs:integer", NewString("x")); ok {
		t.Errorf("xs:string must not coerce to xs:integer")
	}
	if _, ok := CoerceToDeclaredType("xs:integer )", NewInteger(1)); ok {
		t.Errorf("unparsable type text must report !ok")
	}
	if _, ok := CoerceToDeclaredType("'unterminated", NewInteger(1)); ok {
		t.Errorf("unlexable type text must report !ok")
	}
	if v, ok := CoerceToDeclaredType("xs:integer", NewUntyped("12")); !ok || ToString(v) != "12" {
		t.Errorf("untypedAtomic → xs:integer cast failed: %v %v", v, ok)
	}

	// SeqTypeWantsNodes: only a node kind test qualifies.
	for s, want := range map[string]bool{
		"text()*": true, "node()": true, "element(a)+": true,
		"item()": false, "xs:integer": false, "map(*)": false,
		"empty-sequence()": false, "text() junk": false, "'x": false, "": false,
	} {
		if got := SeqTypeWantsNodes(s); got != want {
			t.Errorf("SeqTypeWantsNodes(%q) = %v, want %v", s, got, want)
		}
	}

	// ParseSeqTypeString: "" is item()* (nil).
	if st, err := ParseSeqTypeString("  "); st != nil || err != nil {
		t.Errorf("ParseSeqTypeString(blank) = %v %v, want nil nil", st, err)
	}
	if st, err := ParseSeqTypeString("xs:integer+"); err != nil || st.Occur != '+' {
		t.Errorf("ParseSeqTypeString(xs:integer+) = %+v %v", st, err)
	}
	for _, bad := range []string{"xs:integer xs:string", "'x", "function(xs:integer)", "map(xs:string)", "array(", "element(", "1"} {
		if _, err := ParseSeqTypeString(bad); err == nil {
			t.Errorf("ParseSeqTypeString(%q) should fail", bad)
		}
	}

	// ParseItemTypeStrict: no occurrence indicator, no trailing junk, no
	// unbound prefix, no prefixed non-built-in type (XPST0051).
	if st, err := ParseItemTypeStrict("xs:integer", nil); err != nil || st.Item.Atom != XSinteger {
		t.Errorf("ParseItemTypeStrict(xs:integer) = %v %v", st, err)
	}
	ctx := &Context{NS: testNSMap{"my": "urn:my"}}
	for _, c := range []struct{ in, code string }{
		{"xs:integer*", ""}, {"xs:integer x", ""}, {"empty-sequence()", ""}, {"'x", ""},
		{"my:thing", "XPST0051"}, {"foo:thing", "XPST0081"}, {"(", ""},
	} {
		_, err := ParseItemTypeStrict(c.in, ctx)
		if err == nil {
			t.Errorf("ParseItemTypeStrict(%q) should fail", c.in)
			continue
		}
		if c.code != "" && tyErrCode(err) != c.code {
			t.Errorf("ParseItemTypeStrict(%q) = %v, want %s", c.in, err, c.code)
		}
	}

	// MatchesItemTypeOf: plain instance-of, no conversion.
	st, _ := ParseItemTypeStrict("xs:string", nil)
	doc, _ := xmltree.Parse("<r>x</r>")
	if MatchesItemTypeOf(st, doc.Children[0], nil) {
		t.Errorf("an element is not an instance of xs:string")
	}
	if !MatchesItemTypeOf(st, NewString("a"), nil) || !MatchesItemTypeOf(nil, doc, nil) {
		t.Errorf("MatchesItemTypeOf rejected a matching item")
	}
	if MatchesItemTypeOf(st, NewUntyped("a"), nil) {
		t.Errorf("xs:untypedAtomic is not an instance of xs:string")
	}
	// MatchesSeqType without a context.
	nt, _ := ParseSeqTypeString("element(r)")
	if !MatchesSeqType(nt, doc.Children[0]) || MatchesSeqType(nt, doc) {
		t.Errorf("MatchesSeqType element(r) is wrong")
	}
	at, _ := ParseSeqTypeString("attribute(a)")
	d2, _ := xmltree.Parse(`<r a="1" b="2"/>`)
	if !MatchesSeqType(at, d2.Children[0].Attrs[0]) || MatchesSeqType(at, d2.Children[0].Attrs[1]) {
		t.Errorf("MatchesSeqType attribute(a) is wrong")
	}

	// ContextItemValue / TypedTextItem: plain nodes stand for themselves; a
	// text node stamped with a TypeAnno stands for that typed value.
	if ContextItemValue(nil) != nil {
		t.Errorf("ContextItemValue(nil) must be nil")
	}
	if ContextItemValue(doc) != Item(doc) {
		t.Errorf("an ordinary node must stand for itself")
	}
	carrier := xmltree.NewText("")
	m := NewMap()
	carrier.SetRealItem(m)
	if ContextItemValue(carrier) != Item(m) {
		t.Errorf("a RealItem carrier must stand for its item")
	}
	stamped := xmltree.NewText("42")
	stamped.SetTypeAnno(int32(XSinteger))
	stamped.SetSynthCtx(true)
	if a, ok := ContextItemValue(stamped).(*Atomic); !ok || a.T != XSinteger || a.Lexical() != "42" {
		t.Errorf("a stamped synthetic text node must stand for its xs:integer, got %v", ContextItemValue(stamped))
	}
	if a, ok := TypedTextItem(stamped); !ok || a.(*Atomic).T != XSinteger {
		t.Errorf("TypedTextItem(stamped) = %v %v", a, ok)
	}
	if _, ok := TypedTextItem(xmltree.NewText("42")); ok {
		t.Errorf("an unstamped text node has no typed item")
	}
	if _, ok := TypedTextItem(doc.Children[0]); ok {
		t.Errorf("an element is not a text node")
	}
	if _, ok := TypedTextItem(nil); ok {
		t.Errorf("TypedTextItem(nil) must be !ok")
	}

	if !AtomDerivesFrom(XSbyte, XSinteger) || AtomDerivesFrom(XSinteger, XSbyte) || !AtomDerivesFrom(XSfloat, XSnumeric) {
		t.Errorf("AtomDerivesFrom is wrong")
	}
	if !strings.Contains(XSinteger.String(), "integer") {
		t.Errorf("AtomType.String() = %q", XSinteger.String())
	}
}
