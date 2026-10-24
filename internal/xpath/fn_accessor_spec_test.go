package xpath

import (
	"errors"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestAccessorSpec covers the accessor functions of F&O 3.1 §2 (node-name,
// nilled, data, base-uri, document-uri) and fn:static-base-uri (the
// context functions).
func TestAccessorSpec(t *testing.T) {
	const doc = `<r xmlns:p="urn:p" xmlns="urn:d"><a p:x="1">t<!--c--><?pi d?></a></r>`
	coreRun(t, doc, nil, []coreCase{
		// §2.1 fn:node-name
		{expr: `local-name-from-QName(node-name(/*))`, want: "r"},
		{expr: `namespace-uri-from-QName(node-name(/*))`, want: "urn:d"},
		{expr: `namespace-uri-from-QName(node-name(//@*:x))`, want: "urn:p"},
		{expr: `local-name-from-QName(node-name(//processing-instruction()))`, want: "pi"},
		{expr: `local-name-from-QName(node-name(/*/namespace::p))`, want: "p"},
		{expr: `namespace-uri-from-QName(node-name(/*/namespace::p))`, want: ""},
		// document, text, comment and default-namespace nodes have no name
		{expr: `node-name(/)`, want: "()"},
		{expr: `node-name(//text())`, want: "()"},
		{expr: `node-name(//comment())`, want: "()"},
		{expr: `node-name(/*/namespace::*[local-name() = ''])`, want: "()"},
		{expr: `node-name(())`, want: "()"},
		{expr: `/*/*/node-name() ! local-name-from-QName(.)`, want: "a"},
		{expr: `node-name("string")`, err: "XPTY0004"},
		{expr: `(1)!node-name()`, err: "XPTY0004"},

		// §2.2 fn:nilled: an unvalidated element is never nilled; other kinds ()
		{expr: `nilled(/*)`, want: "false"},
		{expr: `nilled(//text())`, want: "()"},
		{expr: `nilled(//@*:x)`, want: "()"},
		{expr: `nilled(/)`, want: "()"},
		{expr: `nilled(())`, want: "()"},
		{expr: `/*/nilled()`, want: "false"},
		{expr: `nilled(1)`, err: "XPTY0004"},
		{expr: `nilled(/*/*/node())`, err: "XPTY0004"},
		{expr: `(1)!nilled()`, err: "XPTY0004"},

		// §2.4 fn:data
		{expr: `data(/*)`, want: "t"},
		{expr: `data((1, 'a', //@*:x))`, want: "1|a|1"},
		{expr: `data(//@*:x) instance of xs:untypedAtomic`, want: "true"},
		{expr: `data([1, [2, 3]])`, want: "1|2|3"},
		{expr: `/*/*/@*/data()`, want: "1"},

		// §2.5 fn:base-uri / §2.6 fn:document-uri with no base anywhere
		{expr: `base-uri(())`, want: "()"},
		{expr: `base-uri(/*/namespace::p)`, want: "()"},
		{expr: `(1)!base-uri()`, err: "XPTY0004"},
		{expr: `document-uri(())`, want: "()"},
		{expr: `document-uri(/*)`, want: "()"},
		{expr: `document-uri(/)`, want: "()"},

		// static-base-uri: absent here
		{expr: `static-base-uri()`, want: "()"},
	})

	// With a static base URI and a retrieved document.
	d, err := xmltree.Parse(`<r><e/></r>`)
	if err != nil {
		t.Fatal(err)
	}
	d.SetBase("http://example.com/docs/r.xml")
	res := &rsResolver{docs: map[string]*xmltree.Node{"http://example.com/docs/r.xml": d}}
	coreRun(t, "", func(c *Context) {
		c.BaseURI = "http://example.com/q/main.xq"
		c.Resolver = res
	}, []coreCase{
		{expr: `static-base-uri()`, want: "http://example.com/q/main.xq"},
		{expr: `static-base-uri() instance of xs:anyURI`, want: "true"},
		{expr: `document-uri(doc('http://example.com/docs/r.xml'))`, want: "http://example.com/docs/r.xml"},
		{expr: `document-uri(doc('http://example.com/docs/r.xml')) instance of xs:anyURI`, want: "true"},
		{expr: `doc('http://example.com/docs/r.xml')/document-uri()`, want: "http://example.com/docs/r.xml"},
		{expr: `document-uri(doc('http://example.com/docs/r.xml')/r)`, want: "()"},
		{expr: `base-uri(doc('http://example.com/docs/r.xml')//e)`, want: "http://example.com/docs/r.xml"},
		{expr: `doc('http://example.com/docs/r.xml')//e/base-uri()`, want: "http://example.com/docs/r.xml"},
	})
}

// TestErrorTraceSpec covers fn:error and fn:trace (F&O 3.1 §3.1.1, §3.2.1)
// and math:exp10 (§4.8.3).
func TestErrorTraceSpec(t *testing.T) {
	ev := func(expr string) (Object, error) {
		p, err := Parse(expr)
		if err != nil {
			t.Fatalf("parse %q: %v", expr, err)
		}
		doc, _ := xmltree.Parse("<r/>")
		return p.Eval(&Context{Node: doc, Pos: 1, Size: 1})
	}
	for _, c := range []struct {
		expr, ns, local, desc string
		value                 string // coreRender of the error object, "-" for none
	}{
		// fn:error() raises err:FOER0000
		{`error()`, nsErr, "FOER0000", "", "-"},
		{`error(())`, nsErr, "FOER0000", "", "-"},
		{`error((), 'only a description')`, nsErr, "FOER0000", "only a description", "-"},
		// the REC example
		{`error(QName('http://www.example.com/HR', 'myerr:toohighsal'), 'Does not apply because salary is too high')`,
			"http://www.example.com/HR", "toohighsal", "Does not apply because salary is too high", "-"},
		{`error(QName('urn:e', 'E1'), 'd', (1, 2))`, "urn:e", "E1", "d", "1|2"},
		{`error(QName('urn:e', 'E1'), (), 'v')`, "urn:e", "E1", "", "v"},
	} {
		_, err := ev(c.expr)
		var xe *Error
		if !errors.As(err, &xe) {
			t.Errorf("%s: error %v is not a structured *Error", c.expr, err)
			continue
		}
		if xe.Code.Space != c.ns || xe.Code.Local != c.local {
			t.Errorf("%s: code Q{%s}%s, want Q{%s}%s", c.expr, xe.Code.Space, xe.Code.Local, c.ns, c.local)
		}
		if xe.Desc != c.desc {
			t.Errorf("%s: description %q, want %q", c.expr, xe.Desc, c.desc)
		}
		if c.value == "-" {
			if xe.Value != nil {
				t.Errorf("%s: unexpected error object %v", c.expr, xe.Value)
			}
		} else if xe.Value == nil || coreRender(xe.Value) != c.value {
			t.Errorf("%s: error object %v, want %q", c.expr, xe.Value, c.value)
		}
		if !strings.Contains(xe.Error(), c.local) {
			t.Errorf("%s: message %q lacks the code %s", c.expr, xe.Error(), c.local)
		}
	}

	// NewError builds the same structure a host can raise.
	ne := NewError(xmltree.Name{Space: nsErr, Local: "XTMM9000", Prefix: "err"}, "terminated", NewString("x"))
	if ne.Error() != "err:XTMM9000: terminated" || ne.Desc != "terminated" || ToString(ne.Value) != "x" {
		t.Errorf("NewError = %q / %q / %v", ne.Error(), ne.Desc, ne.Value)
	}
	if got := NewError(xmltree.Name{Local: "E"}, "", nil).Error(); got != "err:E" {
		t.Errorf("NewError without description = %q, want err:E", got)
	}

	// fn:trace returns its first argument unchanged (its output goes to a
	// processor log).
	coreRun(t, "", nil, []coreCase{
		{expr: `trace((1, 2, 3), 'label')`, want: "1|2|3"},
		{expr: `trace('x')`, want: "x"},
		{expr: `trace(())`, want: "()"},
		// §4.8.3 math:exp10
		{expr: `math:exp10(())`, want: "()"},
		{expr: `math:exp10(0)`, want: "1"},
		{expr: `math:exp10(1)`, want: "10"},
		{expr: `math:exp10(0.5)`, want: "3.1622776601683795"},
		{expr: `math:exp10(-1)`, want: "0.1"},
		{expr: `math:exp10(0) instance of xs:double`, want: "true"},
	})
}

// TestTransformUnavailable covers fn:transform's argument check and its
// "no XSLT processor is available" outcome (F&O 3.1 §16.3.2, FOXT0001)
// when neither a host function resolver nor TransformFunc serves it, and the
// TransformFunc hook contract when one is installed.
func TestTransformUnavailable(t *testing.T) {
	saved := TransformFunc
	defer func() { TransformFunc = saved }()

	TransformFunc = nil
	coreRun(t, "", nil, []coreCase{
		{expr: `transform(map{})`, err: "FOXT0001"},
		{expr: `transform(1)`, err: "XPTY0004"},
		{expr: `transform((map{}, map{}))`, err: "XPTY0004"},
		{expr: `transform(())`, err: "XPTY0004"},
	})

	var gotOpts *Map
	TransformFunc = func(ctx *Context, opts *Map) (Object, error) {
		gotOpts = opts
		return NewMap(), nil
	}
	coreRun(t, "", nil, []coreCase{
		{expr: `map:size(transform(map{'stylesheet-text': 'x'}))`, want: "0"},
	})
	if gotOpts == nil || ToString(gotOpts.Get(NewString("stylesheet-text"))) != "x" {
		t.Errorf("TransformFunc did not receive the options map")
	}
}
