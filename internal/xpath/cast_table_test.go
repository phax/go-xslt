package xpath

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// tyErrCode extracts the W3C error code (err:XXXX9999) from an error.
var tyErrCodeRe = regexp.MustCompile(`err:([A-Z]{4}[0-9]{4})`)

func tyErrCode(err error) string {
	if err == nil {
		return ""
	}
	if m := tyErrCodeRe.FindStringSubmatch(err.Error()); m != nil {
		return m[1]
	}
	return "?" + err.Error()
}

// tyRun parses and evaluates expr against <r/>; it returns the string value
// of the result, or "!CODE" when parsing or evaluation raised an error.
func tyRun(expr string) string {
	p, err := Parse(expr)
	if err != nil {
		return "!" + tyErrCode(err)
	}
	doc, _ := xmltree.Parse("<r/>")
	v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1})
	if err != nil {
		return "!" + tyErrCode(err)
	}
	return ToString(v)
}

type tyCase struct{ expr, want string }

func tyCheck(t *testing.T, cases []tyCase) {
	t.Helper()
	for _, c := range cases {
		got := tyRun(c.expr)
		if c.want == "!*" && strings.HasPrefix(got, "!") {
			continue
		}
		if got != c.want {
			t.Errorf("%s\n  got  %q\n  want %q", c.expr, got, c.want)
		}
	}
}

// TestTyCastingTable walks the F&O 3.1 §19.1 casting table: each row is a
// source type, each column a target, and the cell says whether the cast is
// always allowed (Y), never allowed (N → XPTY0004), or allowed subject to
// the value (M → FORG0001/FOCA0002 when the value does not fit). The
// expected results are the canonical lexical forms of §19.1.1–§19.1.6 and
// XSD 1.1 Part 2.
func TestTyCastingTable(t *testing.T) {
	tyCheck(t, []tyCase{
		// §19.1.2.1 casting to xs:string / xs:untypedAtomic: always allowed,
		// the canonical lexical form of the value.
		{`string(xs:double('1e3') cast as xs:untypedAtomic)`, "1000"},
		{`(3.50 cast as xs:untypedAtomic) instance of xs:untypedAtomic`, "true"},
		{`xs:hexBinary('0aff') cast as xs:string`, "0AFF"},
		{`xs:duration('P1Y13M') cast as xs:string`, "P2Y1M"},

		// §19.1.2.2 casting to numeric types.
		{`true() cast as xs:double`, "1"},
		{`false() cast as xs:float`, "0"},
		{`true() cast as xs:decimal`, "1"},
		{`false() cast as xs:decimal`, "0"},
		{`true() cast as xs:integer`, "1"},
		{`false() cast as xs:integer`, "0"},
		{`xs:untypedAtomic(' 12 ') cast as xs:integer`, "12"},
		{`'INF' cast as xs:double`, "INF"},
		{`'-INF' cast as xs:double`, "-INF"},
		{`'NaN' cast as xs:float`, "NaN"},
		{`'Infinity' cast as xs:double`, "!FORG0001"},
		{`'abc' cast as xs:double`, "!FORG0001"},
		{`'1.5e2' cast as xs:double`, "150"},
		{`xs:float('1.5') cast as xs:double`, "1.5"},
		{`1.5 cast as xs:float`, "1.5"},
		{`xs:double('-7.9') cast as xs:integer`, "-7"},
		{`xs:float('2.5') cast as xs:integer`, "2"},
		{`xs:double('NaN') cast as xs:integer`, "!FOCA0002"},
		{`xs:double('INF') cast as xs:integer`, "!FOCA0002"},
		{`xs:float('-INF') cast as xs:decimal`, "!FOCA0002"},
		{`xs:double('NaN') cast as xs:decimal`, "!FOCA0002"},
		{`xs:double('0.5') cast as xs:decimal`, "0.5"},
		{`12.75 cast as xs:integer`, "12"},
		{`-12.75 cast as xs:integer`, "-12"},
		{`7 cast as xs:decimal`, "7"},
		{`(7 cast as xs:decimal) instance of xs:integer`, "false"},
		{`'1e3' cast as xs:decimal`, "!FORG0001"},
		{`'12.5.1' cast as xs:decimal`, "!FORG0001"},
		{`' -3.25 ' cast as xs:decimal`, "-3.25"},
		{`'3.0' cast as xs:integer`, "!FORG0001"},
		{`'99999999999999999999999' cast as xs:integer`, "99999999999999999999999"},
		{`'12a' cast as xs:integer`, "!FORG0001"},
		{`xs:date('2000-01-01') cast as xs:integer`, "!XPTY0004"},
		{`xs:anyURI('1') cast as xs:double`, "!XPTY0004"},
		{`xs:hexBinary('01') cast as xs:decimal`, "!XPTY0004"},
		{`xs:duration('P1D') cast as xs:float`, "!XPTY0004"},

		// Derived integer types: value-space bounds (XSD Part 2 §3.4.x).
		{`'127' cast as xs:byte`, "127"},
		{`'128' cast as xs:byte`, "!FORG0001"},
		{`-129 cast as xs:byte`, "!FORG0001"},
		{`32768 cast as xs:short`, "!FORG0001"},
		{`2147483648 cast as xs:int`, "!FORG0001"},
		{`9223372036854775808 cast as xs:long`, "!FORG0001"},
		{`-1 cast as xs:unsignedInt`, "!FORG0001"},
		{`256 cast as xs:unsignedByte`, "!FORG0001"},
		{`65536 cast as xs:unsignedShort`, "!FORG0001"},
		{`18446744073709551615 cast as xs:unsignedLong`, "18446744073709551615"},
		{`18446744073709551616 cast as xs:unsignedLong`, "!FORG0001"},
		{`0 cast as xs:positiveInteger`, "!FORG0001"},
		{`0 cast as xs:nonNegativeInteger`, "0"},
		{`1 cast as xs:nonPositiveInteger`, "!FORG0001"},
		{`0 cast as xs:negativeInteger`, "!FORG0001"},
		{`-5 cast as xs:negativeInteger`, "-5"},
		{`(5 cast as xs:short) instance of xs:int`, "true"},
		{`(5 cast as xs:unsignedByte) instance of xs:nonNegativeInteger`, "true"},

		// §19.1.4 casting to xs:boolean.
		{`0 cast as xs:boolean`, "false"},
		{`xs:double('NaN') cast as xs:boolean`, "false"},
		{`-0.5 cast as xs:boolean`, "true"},
		{`'1' cast as xs:boolean`, "true"},
		{`' false ' cast as xs:boolean`, "false"},
		{`'TRUE' cast as xs:boolean`, "!FORG0001"},

		// §19.1.5 casting to duration and date/time types.
		{`xs:dateTime('2002-05-31T13:20:00-05:00') cast as xs:date`, "2002-05-31-05:00"},
		{`xs:dateTime('2002-05-31T13:20:00-05:00') cast as xs:time`, "13:20:00-05:00"},
		{`xs:dateTime('2002-05-31T13:20:00-05:00') cast as xs:gYear`, "2002-05:00"},
		{`xs:dateTime('2002-05-31T13:20:00-05:00') cast as xs:gYearMonth`, "2002-05-05:00"},
		{`xs:dateTime('2002-05-31T13:20:00-05:00') cast as xs:gMonth`, "--05-05:00"},
		{`xs:dateTime('2002-05-31T13:20:00-05:00') cast as xs:gMonthDay`, "--05-31-05:00"},
		{`xs:dateTime('2002-05-31T13:20:00-05:00') cast as xs:gDay`, "---31-05:00"},
		{`xs:dateTime('2002-05-31T13:20:00Z') cast as xs:dateTimeStamp`, "2002-05-31T13:20:00Z"},
		{`xs:date('2002-05-31') cast as xs:dateTime`, "2002-05-31T00:00:00"},
		{`xs:date('2002-05-31Z') cast as xs:gMonthDay`, "--05-31Z"},
		{`xs:date('2002-05-31') cast as xs:gDay`, "---31"},
		{`xs:date('2002-05-31') cast as xs:dateTimeStamp`, "!FORG0001"},
		{`xs:date('2002-05-31') cast as xs:time`, "!XPTY0004"},
		{`xs:time('13:20:00') cast as xs:date`, "!XPTY0004"},
		{`xs:gYear('2002') cast as xs:date`, "!XPTY0004"},
		{`xs:gYear('2002') cast as xs:gYear`, "2002"},
		{`xs:gMonth('--05') cast as xs:gDay`, "!XPTY0004"},
		{`'2001-01-01T00:00:00' cast as xs:dateTimeStamp`, "!FORG0001"},
		{`'2001-01-01T00:00:00+01:00' cast as xs:dateTimeStamp`, "2001-01-01T00:00:00+01:00"},
		{`'2001-13-01' cast as xs:date`, "!FORG0001"},
		{`12 cast as xs:date`, "!XPTY0004"},
		{`true() cast as xs:dateTime`, "!XPTY0004"},
		{`xs:duration('P1Y2M3DT4H') cast as xs:yearMonthDuration`, "P1Y2M"},
		{`xs:duration('P1Y2M3DT4H') cast as xs:dayTimeDuration`, "P3DT4H"},
		{`xs:yearMonthDuration('P1Y') cast as xs:dayTimeDuration`, "PT0S"},
		{`xs:dayTimeDuration('PT5M') cast as xs:yearMonthDuration`, "P0M"},
		{`xs:dayTimeDuration('PT5M') cast as xs:duration`, "PT5M"},
		{`'P1Y2D' cast as xs:yearMonthDuration`, "!FORG0001"},
		{`'P1Y2D' cast as xs:dayTimeDuration`, "!FORG0001"},
		{`'P1Q' cast as xs:duration`, "!FORG0001"},
		{`' P2M ' cast as xs:yearMonthDuration`, "P2M"},
		{`3 cast as xs:duration`, "!XPTY0004"},
		{`xs:date('2000-01-01') cast as xs:dayTimeDuration`, "!XPTY0004"},

		// §19.1.6 casting to binary types.
		{`xs:hexBinary('48656C6C6F') cast as xs:base64Binary`, "SGVsbG8="},
		{`xs:base64Binary('SGVsbG8=') cast as xs:hexBinary`, "48656C6C6F"},
		{`xs:base64Binary('SGVsbG8=') cast as xs:base64Binary`, "SGVsbG8="},
		{`xs:hexBinary('ff') cast as xs:hexBinary`, "FF"},
		{`'ABC' cast as xs:hexBinary`, "!FORG0001"},
		{`'zz' cast as xs:hexBinary`, "!FORG0001"},
		{`'@@@' cast as xs:base64Binary`, "!FORG0001"},
		{`1 cast as xs:hexBinary`, "!XPTY0004"},
		{`true() cast as xs:base64Binary`, "!XPTY0004"},

		// §19.1.7 casting to xs:anyURI and §19.1.3 to xs:QName.
		{`'  http://example.com/a  ' cast as xs:anyURI`, "http://example.com/a"},
		{`1 cast as xs:anyURI`, "!XPTY0004"},
		{`xs:anyURI('http://a/') cast as xs:string`, "http://a/"},
		{`xs:anyURI('x') cast as xs:QName`, "!XPTY0004"},
		{`12 cast as xs:QName`, "!XPTY0004"},
		{`local-name-from-QName(xs:QName('xs:integer') cast as xs:QName)`, "integer"},

		// Abstract and unknown targets are static errors (XPath 3.1 §3.14.2).
		{`1 cast as xs:NOTATION`, "!XPST0080"},
		{`1 cast as xs:anyAtomicType`, "!XPST0080"},
		{`1 cast as xs:doesNotExist`, "!XPST0051"},

		// §19.1.1 casting to (derived) string types applies the whiteSpace
		// facet, then checks the lexical space.
		{`'  a   b  ' cast as xs:token`, "a b"},
		{`string-to-codepoints(concat('a', codepoints-to-string(9), 'b') cast as xs:normalizedString)[2]`, "32"},
		{`'en-US' cast as xs:language`, "en-US"},
		{`'abcdefghi' cast as xs:language`, "!FORG0001"},
		{`'a:b' cast as xs:Name`, "a:b"},
		{`'1a' cast as xs:Name`, "!FORG0001"},
		{`'a:b' cast as xs:NCName`, "!FORG0001"},
		{`' id1 ' cast as xs:ID`, "id1"},
		{`'x y' cast as xs:IDREF`, "!FORG0001"},
		{`'e1' cast as xs:ENTITY`, "e1"},
		{`'-.x' cast as xs:NMTOKEN`, "-.x"},
		{`'a b' cast as xs:NMTOKEN`, "!FORG0001"},
		{`true() cast as xs:token`, "true"},
		{`12 cast as xs:NMTOKEN`, "12"},
		{`xs:date('2000-01-01') cast as xs:token`, "!XPTY0004"},
		{`('a' cast as xs:NCName) instance of xs:Name`, "true"},
		{`('a' cast as xs:NCName) instance of xs:NMTOKEN`, "false"},

		// xs:numeric is the union (xs:double, xs:float, xs:decimal) — XPath
		// 3.1 §2.5.5.3; a member-typed value is kept, anything else is cast
		// to the first member that accepts it.
		{`(17 cast as xs:numeric) instance of xs:integer`, "true"},
		{`('12' cast as xs:numeric) instance of xs:double`, "true"},

		// Empty operand: `cast as T?` gives (), `cast as T` is XPTY0004.
		{`count(() cast as xs:integer?)`, "0"},
		{`() cast as xs:integer`, "!XPTY0004"},
		{`(1, 2) cast as xs:integer`, "!XPTY0004"},

		// castable as never raises a dynamic error.
		{`'x' castable as xs:integer`, "false"},
		{`'P1Y' castable as xs:yearMonthDuration`, "true"},
		{`xs:double('NaN') castable as xs:integer`, "false"},
		{`() castable as xs:integer?`, "true"},
		{`() castable as xs:integer`, "false"},
	})
}

// TestTyCastToAPI pins the exported Go entry points CastTo, Castable,
// CallConstructor and AtomicFromItem on the same casting rules.
func TestTyCastToAPI(t *testing.T) {
	if a, err := CastTo("42", XSinteger); err != nil || a.T != XSinteger || a.Lexical() != "42" {
		t.Errorf("CastTo(\"42\", xs:integer) = %v, %v", a, err)
	}
	if a, err := CastTo(true, XSdouble); err != nil || a.Float() != 1 {
		t.Errorf("CastTo(true, xs:double) = %v, %v", a, err)
	}
	if a, err := CastTo(2.5, XSstring); err != nil || a.T != XSstring || a.Lexical() != "2.5" {
		t.Errorf("CastTo(2.5, xs:string) = %v, %v", a, err)
	}
	if _, err := CastTo(NewMap(), XSstring); err == nil {
		t.Errorf("CastTo(map, xs:string) should fail: a map is not atomizable")
	}
	if Castable("abc", XSinteger) || !Castable("12", XSbyte) {
		t.Errorf("Castable gives wrong answers")
	}
	v, ok, err := CallConstructor("integer", NewString("7"))
	if !ok || err != nil || ToString(v) != "7" {
		t.Errorf("CallConstructor(integer, '7') = %v %v %v", v, ok, err)
	}
	v, ok, err = CallConstructor("integer", Sequence{})
	if !ok || err != nil || len(Items(v)) != 0 {
		t.Errorf("CallConstructor(integer, ()) must be the empty sequence, got %v %v %v", v, ok, err)
	}
	if _, ok, _ = CallConstructor("noSuchType", NewString("7")); ok {
		t.Errorf("CallConstructor for an unknown type must report !ok")
	}
	if _, ok, err = CallConstructor("integer", NewString("x")); !ok || tyErrCode(err) != "FORG0001" {
		t.Errorf("CallConstructor(integer, 'x') error = %v", err)
	}
	// Comment and PI nodes atomize to xs:string (XDM 3.1 §6.6.5/§6.7.5);
	// element nodes to xs:untypedAtomic (§6.2.5).
	doc, _ := xmltree.Parse("<r><!--c--><?p data?><e>v</e></r>")
	r := doc.Children[0]
	for _, c := range []struct {
		n    *xmltree.Node
		want AtomType
	}{{r.Children[0], XSstring}, {r.Children[1], XSstring}, {r.Children[2], XSuntypedAtomic}} {
		a, err := AtomicFromItem(c.n)
		if err != nil || a.T != c.want {
			t.Errorf("AtomicFromItem(%v) = %v %v, want type %v", c.n.Kind, a, err, c.want)
		}
	}
	if tag, ok := ItemAtomTypeTag(NewDouble(1)); !ok || AtomType(tag) != XSdouble {
		t.Errorf("ItemAtomTypeTag(double) = %v %v", tag, ok)
	}
	if tag, ok := ItemAtomTypeTag(true); !ok || AtomType(tag) != XSboolean {
		t.Errorf("ItemAtomTypeTag(bool) = %v %v", tag, ok)
	}
	if _, ok := ItemAtomTypeTag(r); ok {
		t.Errorf("ItemAtomTypeTag(node) must be !ok")
	}
}

// TestTyAtomize covers fn:data's atomization of arrays (XPath 3.1 §2.4.2):
// an array atomizes to the atomization of its members, recursively.
func TestTyAtomize(t *testing.T) {
	tyCheck(t, []tyCase{
		{`string-join(data([1, [2, 3], ()]), ',')`, "1,2,3"},
		{`count(data([]))`, "0"},
	})
}
