package xpath

import (
	"strings"
	"testing"
)

// The nillable "?" after the type name exists in ElementTest only; in an
// AttributeTest it is a syntax error (XPath 3.1 A.1, issue #11). As an
// occurrence indicator after the closing parenthesis it stays valid.
func TestAttributeTestHasNoNillable(t *testing.T) {
	for _, src := range []string{
		`//@attribute(a, xs:untypedAtomic?)`,
		`//@attribute(*, xs:integer?)`,
		`. instance of attribute(a, xs:string?)`,
	} {
		if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), "XPST0003") {
			t.Errorf("%s: want XPST0003, got %v", src, err)
		}
	}
	for _, src := range []string{
		`//element(a, xs:integer?)`,
		`//@attribute(a, xs:untypedAtomic)`,
		`. instance of attribute(a, xs:string)?`,
		`. instance of element(*, xs:anyType?)?`,
	} {
		if _, err := Parse(src); err != nil {
			t.Errorf("%s: %v", src, err)
		}
	}
}
