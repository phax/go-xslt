package xpath

import (
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// BenchmarkBuiltinCall measures a call of a built-in function with arguments
// that already match its declared types — the case the function conversion
// rules (fn_argcheck.go) must not slow down — and the calls that need a
// conversion (an untyped node for a number, an integer for a double).
func BenchmarkBuiltinCall(b *testing.B) {
	doc, err := xmltree.Parse(`<r><a x="3" y="hello">hello world</a><n>2.5</n></r>`)
	if err != nil {
		b.Fatal(err)
	}
	ctxP, err := Parse("/r/a")
	if err != nil {
		b.Fatal(err)
	}
	v, err := ctxP.Eval(&Context{Node: doc, Pos: 1, Size: 1})
	if err != nil {
		b.Fatal(err)
	}
	ns, _ := v.(NodeSet)
	if len(ns) != 1 {
		b.Fatalf("context node: %v", v)
	}
	for _, c := range []struct{ name, expr string }{
		{"baseline-arith", `1 + 2`},
		{"string-length-node", `string-length(.)`},
		{"string-length-string", `string-length('hello')`},
		{"upper-case-attr", `upper-case(@y)`},
		{"contains", `contains(., 'wor')`},
		{"starts-with-attr", `starts-with(@y, 'he')`},
		{"substring-int-args", `substring(., 2, 3)`},
		{"concat-4", `concat(., '-', @x, 'z')`},
		{"normalize-space", `normalize-space(.)`},
		{"string-join-3", `string-join(('a', 'b', 'c'), ',')`},
		{"round-attr-untyped", `round(@x)`},
		{"abs-int", `abs(-3)`},
		{"abs-attr-untyped", `abs(@x)`},
		{"sqrt-int-for-double", `math:sqrt(16)`},
		{"count-any", `count(/r/*)`},
		{"exists-any", `exists(@x)`},
		{"not-any", `not(@x)`},
		{"name-node", `name(.)`},
		{"map-get", `map:get(map{'k': 1}, 'k')`},
		{"array-size", `array:size([1, 2, 3])`},
		{"format-number", `format-number(@x, '0.00')`},
		{"nested", `string-length(normalize-space(upper-case(concat(., 'x'))))`},
	} {
		p, err := Parse(c.expr)
		if err != nil {
			b.Fatalf("%s: %v", c.expr, err)
		}
		b.Run(c.name, func(b *testing.B) {
			ctx := &Context{Node: ns[0], Pos: 1, Size: 1}
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := p.Eval(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDispatch calls dispatchFunc directly with matching arguments, so
// the cost of looking a built-in up and checking its arguments is not buried
// in parsing and evaluation.
func BenchmarkDispatch(b *testing.B) {
	doc, err := xmltree.Parse(`<r><a x="3">42</a></r>`)
	if err != nil {
		b.Fatal(err)
	}
	a := NodeSet{doc.Children[0].Children[0]}
	str := NewString("hello world")
	ctx := &Context{Node: doc, Pos: 1, Size: 1}
	arr := NewArray([]Object{NewInteger(1), NewInteger(2)})
	m := NewMap()
	for _, c := range []struct {
		name, prefix, local string
		args                []Object
	}{
		{"string-length-string", "", "string-length", []Object{str}},
		{"string-length-node", "", "string-length", []Object{a}},
		{"substring", "", "substring", []Object{str, NewInteger(2), NewInteger(3)}},
		{"contains", "", "contains", []Object{str, NewString("wor")}},
		{"concat-4", "", "concat", []Object{str, str, str, str}},
		{"abs-int", "", "abs", []Object{NewInteger(-3)}},
		{"exists-any", "", "exists", []Object{a}},
		{"count-any", "", "count", []Object{a}},
		{"sqrt-int", "math", "sqrt", []Object{NewInteger(16)}},
		{"array-size", "array", "size", []Object{arr}},
		{"map-size", "map", "size", []Object{m}},
		{"round-untyped-node", "", "round", []Object{a}},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := dispatchFunc(c.prefix, c.local, c.args, ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
