package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

const ocHead = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning" vc:minVersion="1.1"
    targetNamespace="urn:t" xmlns="urn:t" elementFormDefault="qualified">`

const (
	ocInterleave = `<xs:openContent mode="interleave"><xs:any namespace="##other" processContents="lax"/></xs:openContent>`
	ocSuffix     = `<xs:openContent mode="suffix"><xs:any namespace="##other" processContents="lax"/></xs:openContent>`
	ocDefault    = `<xs:defaultOpenContent mode="interleave"><xs:any namespace="##other" processContents="lax"/></xs:defaultOpenContent>`
	foreignFirst = `<Doc xmlns="urn:t" xmlns:x="urn:x"><x:Foo/><A>1</A></Doc>`
	foreignLast  = `<Doc xmlns="urn:t" xmlns:x="urn:x"><A>1</A><x:Foo/></Doc>`
	ownBogus     = `<Doc xmlns="urn:t"><Bogus/></Doc>`
)

// valid reports the verdict, and the first diagnostic (a schema refusal comes
// out as one).
func ocValidate(schemaBody, inst string) (bool, string) {
	r := engine.Validate(engine.ValidateRequest{Schemas: []string{ocHead + schemaBody + `</xs:schema>`}, Instance: inst, Version: engine.XSD11})
	if len(r.Diagnostics) > 0 {
		return r.Valid, r.Diagnostics[0].Message
	}
	return r.Valid, ""
}

func restrictionOfAnyType(openContent string) string {
	return `<xs:element name="Doc" type="T"/><xs:complexType name="T"><xs:complexContent><xs:restriction base="xs:anyType">` +
		openContent + `<xs:sequence><xs:element name="A" type="xs:string" minOccurs="0"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`
}

// XSD 1.1 §3.4.6.3 clause 2.1: a restriction of xs:anyType is valid as it
// stands, so open content over it is too.
func TestOpenContentOverAnyType(t *testing.T) {
	implicit := ocDefault + `<xs:element name="Doc"><xs:complexType><xs:sequence><xs:element name="A" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	for name, tc := range map[string]struct {
		schema string
		inst   string
		valid  bool
	}{
		"implicit, interleave, foreign first": {implicit, foreignFirst, true},
		"implicit, interleave, foreign last":  {implicit, foreignLast, true},
		"implicit, own namespace refused":     {implicit, ownBogus, false},
		"explicit anyType + default OC":       {ocDefault + restrictionOfAnyType(""), foreignFirst, true},
		"explicit anyType + interleave":       {restrictionOfAnyType(ocInterleave), foreignFirst, true},
		"explicit anyType + interleave, own":  {restrictionOfAnyType(ocInterleave), ownBogus, false},
		"explicit anyType + suffix, last":     {restrictionOfAnyType(ocSuffix), foreignLast, true},
		"explicit anyType + suffix, first":    {restrictionOfAnyType(ocSuffix), foreignFirst, false},
	} {
		got, msg := ocValidate(tc.schema, tc.inst)
		if got != tc.valid {
			t.Errorf("%s: valid=%v, want %v (%s)", name, got, tc.valid, msg)
		}
		if strings.Contains(msg, "closed base") {
			t.Errorf("%s: schema refused: %s", name, msg)
		}
	}
}

// Over a genuinely closed base the introduction of open content stays refused.
func TestOpenContentOverClosedBaseRefused(t *testing.T) {
	const base = `<xs:complexType name="B"><xs:sequence><xs:element name="A" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType>`
	restrict := func(oc string) string {
		return `<xs:element name="Doc" type="T"/>` + base + `<xs:complexType name="T"><xs:complexContent><xs:restriction base="B">` + oc +
			`<xs:sequence><xs:element name="A" type="xs:string" minOccurs="0"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`
	}
	for name, oc := range map[string]string{"interleave": ocInterleave, "suffix": ocSuffix} {
		if _, msg := ocValidate(restrict(oc), foreignFirst); !strings.Contains(msg, "closed base") {
			t.Errorf("%s: want the closed-base refusal, got %q", name, msg)
		}
	}
}

// A base whose own content model already ends in (suffix) or is made of
// (interleave) unbounded wildcards covering the open-content wildcard admits it.
func TestOpenContentOverBaseWithWildcard(t *testing.T) {
	baseWith := func(any string) string {
		return `<xs:element name="Doc" type="T"/><xs:complexType name="B"><xs:sequence><xs:element name="A" type="xs:string" minOccurs="0"/>` + any + `</xs:sequence></xs:complexType>
		<xs:complexType name="T"><xs:complexContent><xs:restriction base="B">` + ocSuffix + `<xs:sequence><xs:element name="A" type="xs:string" minOccurs="0"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`
	}
	const anyOther = `<xs:any namespace="##any" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>`
	if ok, msg := ocValidate(baseWith(anyOther), foreignLast); !ok {
		t.Errorf("trailing ##any wildcard in the base: %s", msg)
	}
	const anyLocal = `<xs:any namespace="##local" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>`
	if _, msg := ocValidate(baseWith(anyLocal), foreignLast); !strings.Contains(msg, "closed base") {
		t.Errorf("a ##local wildcard does not cover ##other: %q", msg)
	}
	const anyOnce = `<xs:any namespace="##any" processContents="lax" minOccurs="0"/>`
	if _, msg := ocValidate(baseWith(anyOnce), foreignLast); !strings.Contains(msg, "closed base") {
		t.Errorf("a wildcard that may occur once is not unbounded: %q", msg)
	}
}
