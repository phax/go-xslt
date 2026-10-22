package xpath

import (
	"fmt"
	"testing"
)

// The index must give exactly what the item-by-item comparison gives, for
// every kind of left operand, including the ones it hands back (errors).
func TestFastGeneralEqMatchesOrdinary(t *testing.T) {
	longInts := "(1 to 200, 9007199254740993)"
	longStrs := "(for $i in 1 to 200 return string($i), 'a', 'b')"
	cases := []string{
		"7 = %s", "500 = %s", "7.0 = %s", "7.5 = %s", "xs:double(7) = %s", "xs:double('NaN') = %s",
		"xs:float(7) = %s", "xs:untypedAtomic('7') = %s", "xs:untypedAtomic('7.0') = %s",
		"xs:untypedAtomic('x') = %s", "'7' = %s", "(1000, 7) = %s", "() = %s", "%s = 9",
		"xs:short(9) = %s", "9007199254740993 = %s", "9007199254740992 = %s",
		"xs:double(9007199254740992) = %s",
	}
	for _, long := range []string{longInts, longStrs} {
		for _, c := range cases {
			expr := fmt.Sprintf(c, long)
			fast, fastErr := evalString(expr, true)
			slow, slowErr := evalString(expr, false)
			if fast != slow || (fastErr == nil) != (slowErr == nil) {
				t.Errorf("%s: index %q/%v, ordinary %q/%v", expr, fast, fastErr, slow, slowErr)
			}
		}
	}
}

func evalString(expr string, indexed bool) (string, error) {
	p, err := Parse(expr)
	if err != nil {
		return "", err
	}
	if !indexed {
		disableEqIndex = true
		defer func() { disableEqIndex = false }()
	}
	v, err := p.Eval(&Context{Pos: 1, Size: 1})
	if err != nil {
		return "", err
	}
	return ToString(v), nil
}

// The equivalence test above only means something if the index is used.
func TestFastGeneralEqIsTaken(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want bool
	}{{"7 = (1 to 200)", true}, {"'b' = (for $i in 1 to 200 return string($i), 'b')", true}, {"xs:float(7) = (1 to 200)", false}} {
		p, err := Parse(tc.expr)
		if err != nil {
			t.Fatal(err)
		}
		ce := p.root.(*CompareExpr)
		ctx := &Context{Pos: 1, Size: 1}
		l, _ := evalExpr(ce.L, ctx)
		r, _ := evalExpr(ce.R, ctx)
		if _, ok := ce.fastGeneralEq(l, r, ctx); ok != tc.want {
			t.Errorf("%s: index used = %v, want %v", tc.expr, ok, tc.want)
		}
	}
}

// translate with a long map string (indexed) must agree with the short form,
// including the rule that the first occurrence of a repeated character wins.
func TestTranslateLongMap(t *testing.T) {
	long := "abcabcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	v, err := evalString(`translate('cab-zZ9', '`+long+`', 'XYZ')`, true)
	if err != nil || v != "ZXY-" {
		t.Fatalf("got %q %v, want %q", v, err, "ZXY-")
	}
}
