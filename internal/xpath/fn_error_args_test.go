package xpath

import (
	"strings"
	"testing"
)

// fn:error's parameters are checked by the function conversion rules
// (XPath 3.1 §3.1.5.2): $code is xs:QName? (a string is not converted, an
// untypedAtomic cannot be cast to the namespace-sensitive xs:QName) and
// $description is exactly one xs:string (issue #9).
func TestErrorArgumentTypes(t *testing.T) {
	for _, c := range []struct{ expr, code string }{
		{`error('a')`, "XPTY0004"},
		{`error((QName('urn:e', 'a'), QName('urn:e', 'b')))`, "XPTY0004"},
		{`error(xs:untypedAtomic('a'))`, "XPTY0117"},
		{`error(QName('urn:e', 'E1'), (), 'v')`, "XPTY0004"},
		{`error(QName('urn:e', 'E1'), ('a', 'b'))`, "XPTY0004"},
		{`error(())`, "FOER0000"},
		{`error((), 'only a description')`, "FOER0000"},
	} {
		p, err := Parse(c.expr)
		if err != nil {
			t.Fatalf("%s: %v", c.expr, err)
		}
		_, err = p.Eval(&Context{})
		if err == nil || !strings.Contains(err.Error(), c.code) {
			t.Errorf("%s: got %v, want %s", c.expr, err, c.code)
		}
	}
}
