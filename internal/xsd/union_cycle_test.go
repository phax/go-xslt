package xsd

import (
	"strings"
	"testing"
)

// TestSimpleTypeCyclesTerminate pins issue #12: a simple type that reaches
// itself used to send every walk over union members into unbounded recursion
// (a fatal stack overflow, not a recoverable panic). A pure-union cycle is
// tolerated in XSD 1.0 — its value space is the least fixed point, the
// members that do not lead back (closeUnion) — and rejected in 1.1; a cycle
// through a restriction base or a list item type is a circular derivation in
// both versions, also when an inline type names the enclosing one.
func TestSimpleTypeCyclesTerminate(t *testing.T) {
	const circular = "circular"
	cases := []struct {
		name, schema string
		want10       string // circular, or the verdicts on "x" then "7" as "valid/invalid"
		want11       string
	}{
		{"self as element type",
			`<xs:simpleType name="u"><xs:union memberTypes="u xs:string"/></xs:simpleType><xs:element name="e" type="u"/>`,
			"valid/valid", circular},
		{"self as attribute type",
			`<xs:simpleType name="u"><xs:union memberTypes="u xs:int"/></xs:simpleType><xs:element name="e"><xs:complexType><xs:attribute name="a" type="u"/></xs:complexType></xs:element>`,
			"invalid/valid", circular},
		{"self as restriction base with enumeration",
			`<xs:simpleType name="u"><xs:union memberTypes="u xs:string"/></xs:simpleType><xs:simpleType name="v"><xs:restriction base="u"><xs:enumeration value="x"/></xs:restriction></xs:simpleType><xs:element name="e" type="v"/>`,
			"valid/invalid", circular},
		{"self as list item type",
			`<xs:simpleType name="u"><xs:union memberTypes="u xs:string"/></xs:simpleType><xs:simpleType name="l"><xs:list itemType="u"/></xs:simpleType><xs:element name="e" type="l"/>`,
			"valid/valid", circular},
		{"self as only member", // the least fixed point is the empty value space
			`<xs:simpleType name="u"><xs:union memberTypes="u"/></xs:simpleType><xs:element name="e" type="u"/>`,
			"invalid/invalid", circular},
		{"two unions (stE110)",
			`<xs:simpleType name="st"><xs:union memberTypes="xs:int xs:string st2"/></xs:simpleType><xs:simpleType name="st2"><xs:union memberTypes="st"/></xs:simpleType><xs:element name="e" type="st2"/>`,
			"valid/valid", circular},
		{"three unions",
			`<xs:simpleType name="a"><xs:union memberTypes="b xs:int"/></xs:simpleType><xs:simpleType name="b"><xs:union memberTypes="c"/></xs:simpleType><xs:simpleType name="c"><xs:union memberTypes="a xs:string"/></xs:simpleType><xs:element name="e" type="b"/>`,
			"valid/valid", circular},
		{"inline union naming the enclosing union",
			`<xs:simpleType name="u"><xs:union><xs:simpleType><xs:union memberTypes="u xs:string"/></xs:simpleType></xs:union></xs:simpleType><xs:element name="e" type="u"/>`,
			"valid/valid", circular},
		{"inline restriction of the enclosing union",
			`<xs:simpleType name="u"><xs:union memberTypes="xs:int"><xs:simpleType><xs:restriction base="u"/></xs:simpleType></xs:union></xs:simpleType><xs:element name="e" type="u"/>`,
			circular, circular},
		{"inline list of the enclosing union",
			`<xs:simpleType name="u"><xs:union memberTypes="xs:int"><xs:simpleType><xs:list itemType="u"/></xs:simpleType></xs:union></xs:simpleType><xs:element name="e" type="u"/>`,
			circular, circular},
		{"restriction of an inline union naming it",
			`<xs:simpleType name="u"><xs:restriction><xs:simpleType><xs:union memberTypes="u xs:string"/></xs:simpleType></xs:restriction></xs:simpleType><xs:element name="e" type="u"/>`,
			circular, circular},
		{"list of an inline union naming it",
			`<xs:simpleType name="u"><xs:list><xs:simpleType><xs:union memberTypes="u xs:string"/></xs:simpleType></xs:list></xs:simpleType><xs:element name="e" type="u"/>`,
			circular, circular},
	}
	for _, c := range cases {
		for _, v := range []struct {
			ver  Version
			want string
		}{{Version10, c.want10}, {Version11, c.want11}} {
			src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + c.schema + `</xs:schema>`
			sch, err := Compile([]string{src}, "", v.ver)
			if v.want == circular {
				if err == nil || !strings.Contains(err.Error(), "circular simpleType derivation") {
					t.Errorf("%s, XSD %v: want a circular-derivation error, got %v", c.name, v.ver, err)
				}
				continue
			}
			if err != nil {
				t.Errorf("%s, XSD %v: compile: %v", c.name, v.ver, err)
				continue
			}
			var got []string
			for _, value := range []string{"x", "7"} {
				inst := "<e>" + value + "</e>"
				if strings.Contains(c.name, "attribute") {
					inst = `<e a="` + value + `"/>`
				}
				r, verr := sch.Validate(inst)
				switch {
				case verr != nil:
					got = append(got, "error:"+verr.Error())
				case r.Valid:
					got = append(got, "valid")
				default:
					got = append(got, "invalid")
				}
			}
			if g := strings.Join(got, "/"); g != v.want {
				t.Errorf("%s, XSD %v: want %s, got %s", c.name, v.ver, v.want, g)
			}
		}
	}
}
