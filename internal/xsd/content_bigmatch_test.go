package xsd

import (
	"errors"
	"strings"
	"testing"
)

// Content models with more than 600 children are matched by the linear
// counting matcher (bigmatch.go). The verdicts are those of cvc-complex-type
// 2.4 / cvc-particle, the same as for small instances.
func TestLargeContentModels(t *testing.T) {
	rep := func(s string, n int) string { return strings.Repeat(s, n) }
	cases := []struct {
		name  string
		model string // complexType content
		extra string // further global declarations
		kids  string
		valid bool
	}{
		{"seq-bounded-ok", `<xs:sequence><xs:element name="a" maxOccurs="1000"/></xs:sequence>`, "", rep("<a/>", 1000), true},
		{"seq-min-met", `<xs:sequence><xs:element name="a" minOccurs="700" maxOccurs="unbounded"/><xs:element name="b"/></xs:sequence>`, "", rep("<a/>", 700) + "<b/>", true},
		{"repeated-pair", `<xs:sequence maxOccurs="unbounded"><xs:element name="k"/><xs:element name="v"/></xs:sequence>`, "", rep("<k/><v/>", 400), true},
		{"repeated-pair-broken", `<xs:sequence maxOccurs="unbounded"><xs:element name="k"/><xs:element name="v"/></xs:sequence>`, "", rep("<k/><v/>", 400) + "<k/>", false},
		{"repeated-pair-optional-v", `<xs:sequence maxOccurs="unbounded"><xs:element name="k"/><xs:element name="v" minOccurs="0"/></xs:sequence>`, "", rep("<k/>", 350) + rep("<k/><v/>", 350), true},
		{"choice-unbounded", `<xs:choice maxOccurs="unbounded"><xs:element name="p"/><xs:element name="q"/></xs:choice>`, "", rep("<p/><q/><q/>", 300), true},
		{"choice-unbounded-stranger", `<xs:choice maxOccurs="unbounded"><xs:element name="p"/><xs:element name="q"/></xs:choice>`, "", rep("<p/><q/>", 400) + "<r/>", false},
		{"choice-of-sequences", `<xs:choice maxOccurs="unbounded"><xs:sequence><xs:element name="p"/><xs:element name="q"/></xs:sequence><xs:element name="r"/></xs:choice>`, "", rep("<p/><q/><r/>", 300), true},
		{"choice-of-sequences-broken", `<xs:choice maxOccurs="unbounded"><xs:sequence><xs:element name="p"/><xs:element name="q"/></xs:sequence><xs:element name="r"/></xs:choice>`, "", rep("<p/><q/><r/>", 300) + "<p/><r/>", false},
		{"header-body-footer", `<xs:sequence><xs:element name="h" minOccurs="0"/><xs:element name="i" maxOccurs="unbounded"/><xs:element name="f" minOccurs="0"/></xs:sequence>`, "", "<h/>" + rep("<i/>", 900) + "<f/>", true},
		{"footer-before-items", `<xs:sequence><xs:element name="h" minOccurs="0"/><xs:element name="i" maxOccurs="unbounded"/><xs:element name="f" minOccurs="0"/></xs:sequence>`, "", "<f/>" + rep("<i/>", 900), false},
		{"wildcard-unbounded", `<xs:sequence><xs:any processContents="skip" maxOccurs="unbounded"/></xs:sequence>`, "", rep("<x/><y/>", 400), true},
		{"wildcard-namespace", `<xs:sequence><xs:any namespace="##other" processContents="skip" maxOccurs="unbounded"/></xs:sequence>`, "", rep(`<p:x xmlns:p="urn:p"/>`, 700) + "<local/>", false},
		{"substitution-members", `<xs:sequence><xs:element ref="h" maxOccurs="unbounded"/></xs:sequence>`,
			`<xs:element name="h"/><xs:element name="m1" substitutionGroup="h"/><xs:element name="m2" substitutionGroup="h"/>`, rep("<h/><m1/><m2/>", 300), true},
		{"substitution-abstract-head", `<xs:sequence><xs:element ref="h" maxOccurs="unbounded"/></xs:sequence>`,
			`<xs:element name="h" abstract="true"/><xs:element name="m1" substitutionGroup="h"/>`, rep("<m1/>", 700) + "<h/>", false},
		{"group-ref", `<xs:sequence><xs:group ref="g" maxOccurs="unbounded"/></xs:sequence>`,
			`<xs:group name="g"><xs:sequence><xs:element name="a"/><xs:element name="b" minOccurs="0"/></xs:sequence></xs:group>`, rep("<a/><b/>", 350) + rep("<a/>", 10), true},
		{"removed-particle", `<xs:sequence><xs:element name="a" maxOccurs="unbounded"/><xs:element name="z" minOccurs="0" maxOccurs="0"/></xs:sequence>`, "", rep("<a/>", 700) + "<z/>", false},
		{"child-content-checked", `<xs:sequence><xs:element name="n" type="xs:int" maxOccurs="unbounded"/></xs:sequence>`, "", rep("<n>1</n>", 700) + "<n>x</n>", false},
	}
	for _, tc := range cases {
		for _, v := range []Version{Version10, Version11} {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="doc"><xs:complexType>` +
					tc.model + `</xs:complexType></xs:element>` + tc.extra + `</xs:schema>`
				s, err := Compile([]string{schema}, "", v)
				if err != nil {
					t.Fatal(err)
				}
				res, err := s.Validate("<doc>" + tc.kids + "</doc>")
				if errors.Is(err, ErrUnsupported) {
					t.Fatalf("no verdict: %v", err)
				}
				if err != nil {
					t.Fatal(err)
				}
				if res.Valid != tc.valid {
					t.Fatalf("valid=%v, want %v (%d errors)", res.Valid, tc.valid, len(res.Errors))
				}
			})
		}
	}
}
