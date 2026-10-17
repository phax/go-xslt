package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

const cardSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:en="urn:en" xmlns:msg="https://github.com/tim-riep/go-xslt/xsd/messages"
  targetNamespace="urn:en" elementFormDefault="qualified">
 <xs:element name="Doc"><xs:complexType><xs:sequence>
  <xs:element name="Number" type="xs:string">
   <xs:annotation><xs:appinfo>
    <msg:message rule="BR-2" on="missing" xml:lang="en">The number is missing.</msg:message>
    <msg:message rule="BR-2" on="missing" xml:lang="de">Die Nummer fehlt.</msg:message>
   </xs:appinfo></xs:annotation>
  </xs:element>
  <xs:element name="Date" type="xs:string"/>
  <xs:element name="Project" type="xs:string" minOccurs="0">
   <xs:annotation><xs:appinfo><msg:message rule="BT-11" on="too-many">Project only once.</msg:message></xs:appinfo></xs:annotation>
  </xs:element>
  <xs:choice>
   <xs:annotation><xs:appinfo><msg:message rule="BR-CO-3">Give a tax point date or a code.</msg:message></xs:appinfo></xs:annotation>
   <xs:element name="TaxDate" type="xs:string"/>
   <xs:element name="TaxCode" type="xs:string"/>
  </xs:choice>
  <xs:element name="Line" type="xs:string" maxOccurs="2"/>
 </xs:sequence></xs:complexType></xs:element>
</xs:schema>`

type dg struct{ kind, particle, path, msg, rule string }

func cardRun(t *testing.T, inst, lang string) []dg {
	t.Helper()
	r := engine.Validate(engine.ValidateRequest{Schemas: []string{cardSchema}, Instance: inst, Version: engine.XSD11, Language: lang})
	var out []dg
	for _, d := range r.Diagnostics {
		out = append(out, dg{d.Kind, d.Particle, d.Path, d.Message, d.Rule})
	}
	if r.Valid != (len(out) == 0) {
		t.Fatalf("valid=%v diagnostics=%v", r.Valid, out)
	}
	return out
}

func TestCardinalityDiagnostics(t *testing.T) {
	const pre = `<Doc xmlns="urn:en">`
	// a valid document stays valid
	if got := cardRun(t, pre+`<Number>1</Number><Date>d</Date><TaxDate>x</TaxDate><Line>l</Line></Doc>`, ""); got != nil {
		t.Fatalf("valid: %v", got)
	}
	// missing: Number — named, on the parent, own message + rule, language chosen
	got := cardRun(t, pre+`<Date>d</Date><TaxDate>x</TaxDate><Line>l</Line></Doc>`, "de")
	want := dg{"missing", "{urn:en}Number", "/Doc[1]", "Die Nummer fehlt.", "BR-2"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("missing: %+v", got)
	}
	got = cardRun(t, pre+`<Date>d</Date><TaxDate>x</TaxDate><Line>l</Line></Doc>`, "")
	if len(got) != 1 || got[0].msg != "The number is missing." {
		t.Errorf("missing, default language: %+v", got) // en entry has xml:lang, none without: first entry
	}
	// generic wording when no message is declared, no lists of names
	got = cardRun(t, pre+`<Number>1</Number><TaxDate>x</TaxDate><Line>l</Line></Doc>`, "")
	if len(got) != 1 || got[0].kind != "missing" || got[0].msg != "element Date is missing in Doc" {
		t.Errorf("generic missing: %+v", got)
	}
	// too many: two Project → one diagnostic at the surplus occurrence
	got = cardRun(t, pre+`<Number>1</Number><Date>d</Date><Project>a</Project><Project>b</Project><TaxDate>x</TaxDate><Line>l</Line></Doc>`, "")
	if len(got) != 1 || got[0].kind != "too-many" || got[0].path != "/Doc[1]/Project[2]" || got[0].rule != "BT-11" || got[0].msg != "Project only once." {
		t.Errorf("too-many: %+v", got)
	}
	// too many without own message: maximum is named
	got = cardRun(t, pre+`<Number>1</Number><Date>d</Date><TaxDate>x</TaxDate><Line>1</Line><Line>2</Line><Line>3</Line><Line>4</Line></Doc>`, "")
	if len(got) != 2 || got[0].path != "/Doc[1]/Line[3]" || got[1].path != "/Doc[1]/Line[4]" ||
		got[0].msg != "element Line occurs more often than allowed (maximum 2) in Doc" {
		t.Errorf("too-many x2: %+v", got)
	}
	// choice: neither alternative → the choice's message, both named
	got = cardRun(t, pre+`<Number>1</Number><Date>d</Date><Line>l</Line></Doc>`, "")
	if len(got) != 1 || got[0].kind != "missing" || got[0].rule != "BR-CO-3" || got[0].msg != "Give a tax point date or a code." {
		t.Errorf("choice: %+v", got)
	}
	// unexpected element
	got = cardRun(t, pre+`<Number>1</Number><Date>d</Date><TaxDate>x</TaxDate><Foo/><Line>l</Line></Doc>`, "")
	if len(got) != 1 || got[0].kind != "unexpected" || got[0].path != "/Doc[1]/Foo[1]" || got[0].msg != "element Foo is not allowed in Doc" {
		t.Errorf("unexpected: %+v", got)
	}
	// wrong order: Date before Number
	got = cardRun(t, pre+`<Date>d</Date><Number>1</Number><TaxDate>x</TaxDate><Line>l</Line></Doc>`, "")
	if len(got) != 1 || got[0].kind != "order" || got[0].path != "/Doc[1]/Date[1]" ||
		got[0].msg != "element Date is not allowed at this position in Doc, expected Number" {
		t.Errorf("order: %+v", got)
	}
	// several defects in one parent, all reported, in document order
	got = cardRun(t, pre+`<Project>a</Project><Project>b</Project><Foo/><Line>l</Line></Doc>`, "")
	var kinds []string
	for _, d := range got {
		kinds = append(kinds, d.kind+":"+d.particle)
	}
	if strings.Join(kinds, " ") != "missing:{urn:en}Number missing:{urn:en}Date missing:{urn:en}TaxDate too-many:{urn:en}Project unexpected:{urn:en}Foo" {
		t.Errorf("several: %v", kinds)
	}
}

// A model the explainer does not cover (xs:all) keeps the generic diagnostic:
// still invalid, still the same code, just without a Kind.
func TestCardinalityFallbackForAll(t *testing.T) {
	const sch = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="r"><xs:complexType><xs:all>
	  <xs:element name="a"/><xs:element name="b"/></xs:all></xs:complexType></xs:element></xs:schema>`
	r := engine.Validate(engine.ValidateRequest{Schemas: []string{sch}, Instance: `<r><a/></r>`, Version: engine.XSD11})
	if r.Valid || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "cvc-complex-type.2.4" || r.Diagnostics[0].Kind != "" {
		t.Errorf("%+v", r)
	}
}

func choiceSchema(withMsg bool) string {
	msg := ""
	if withMsg {
		msg = `<xs:annotation><xs:appinfo>
          <msg:message rule="BR-CO-3" xml:lang="en">Tax point date and its code must not both be given.</msg:message>
          <msg:message rule="BR-CO-3" xml:lang="de">Datum und Code dürfen nicht beide angegeben werden.</msg:message>
        </xs:appinfo></xs:annotation>`
	}
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:msg="https://github.com/tim-riep/go-xslt/xsd/messages">
  <xs:element name="Doc"><xs:complexType><xs:sequence>
    <xs:element name="A" type="xs:string"/>
    <xs:choice minOccurs="0">` + msg + `
      <xs:element name="TaxPointDate" type="xs:string"/>
      <xs:element name="TaxPointDateCode" type="xs:string"/>
    </xs:choice>
    <xs:element name="B" type="xs:string" minOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element></xs:schema>`
}

func validateWith(t *testing.T, schema, inst, lang string) []engine.Diagnostic {
	t.Helper()
	r := engine.Validate(engine.ValidateRequest{Schemas: []string{schema}, Instance: inst, Version: engine.XSD11, Language: lang})
	if r.Valid != (len(r.Diagnostics) == 0) {
		t.Fatalf("valid=%v, %d diagnostics", r.Valid, len(r.Diagnostics))
	}
	return r.Diagnostics
}

// A second alternative of a choice that may occur once is "too many" of the
// CHOICE: its message and rule apply, not "out of order".
func TestExhaustedChoiceIsTooMany(t *testing.T) {
	const inst = `<Doc><A/><TaxPointDate/><TaxPointDateCode/></Doc>`
	d := validateWith(t, choiceSchema(true), inst, "de")
	if len(d) != 1 || d[0].Kind != "too-many" || d[0].Rule != "BR-CO-3" || d[0].Particle != "TaxPointDateCode" ||
		d[0].Path != "/Doc[1]/TaxPointDateCode[1]" || d[0].Message != "Datum und Code dürfen nicht beide angegeben werden." {
		t.Errorf("with message: %+v", d)
	}
	// without a message the generic text names the excess
	d = validateWith(t, choiceSchema(false), inst, "")
	want := "element TaxPointDateCode: the choice of TaxPointDate, TaxPointDateCode occurs more often than allowed (maximum 1) in Doc"
	if len(d) != 1 || d[0].Kind != "too-many" || d[0].Message != want || d[0].Rule != "" {
		t.Errorf("generic: %+v", d)
	}
	// the same alternative twice is the choice used up as well
	d = validateWith(t, choiceSchema(true), `<Doc><A/><TaxPointDate/><TaxPointDate/></Doc>`, "en")
	if len(d) != 1 || d[0].Kind != "too-many" || d[0].Rule != "BR-CO-3" {
		t.Errorf("same alternative twice: %+v", d)
	}
	// an element that belongs to an EARLIER particle stays "order"
	d = validateWith(t, choiceSchema(true), `<Doc><A/><B/><TaxPointDate/></Doc>`, "en")
	if len(d) != 1 || d[0].Kind != "order" || d[0].Rule != "" {
		t.Errorf("order: %+v", d)
	}
	// valid use of the choice stays valid
	if d = validateWith(t, choiceSchema(true), `<Doc><A/><TaxPointDateCode/><B/></Doc>`, ""); len(d) != 0 {
		t.Errorf("valid: %+v", d)
	}
}

// A repeated group that has run out of repetitions is named as such.
func TestExhaustedSequenceGroupIsTooMany(t *testing.T) {
	const sch = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="r"><xs:complexType><xs:sequence>
	  <xs:sequence maxOccurs="2"><xs:element name="a"/><xs:element name="b"/></xs:sequence></xs:sequence></xs:complexType></xs:element></xs:schema>`
	d := validateWith(t, sch, `<r><a/><b/><a/><b/><a/><b/></r>`, "")
	if len(d) == 0 || d[0].Kind != "too-many" || d[0].Path != "/r[1]/a[3]" ||
		d[0].Message != "element a: the sequence of a, b occurs more often than allowed (maximum 2) in r" {
		t.Errorf("%+v", d)
	}
}

func TestMissingRequiredAttributeMessages(t *testing.T) {
	const sch = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:msg="https://github.com/tim-riep/go-xslt/xsd/messages">
	  <xs:attribute name="lang" type="xs:string"><xs:annotation><xs:appinfo>
	    <msg:message rule="R-LANG" on="missing">The language is always stated.</msg:message></xs:appinfo></xs:annotation></xs:attribute>
	  <xs:element name="r"><xs:complexType>
	   <xs:attribute name="scheme" type="xs:string" use="required"><xs:annotation><xs:appinfo>
	     <msg:message rule="BR-62" on="missing" xml:lang="en">The scheme is always stated.</msg:message>
	     <msg:message rule="BR-62" on="missing" xml:lang="de">Das Schema wird immer angegeben.</msg:message>
	     <msg:message rule="IGNORED" on="too-many">not for attributes</msg:message></xs:appinfo></xs:annotation></xs:attribute>
	   <xs:attribute name="plain" type="xs:string" use="required"/>
	   <xs:attribute ref="lang" use="required"/>
	  </xs:complexType></xs:element></xs:schema>`
	d := validateWith(t, sch, `<r/>`, "de")
	var got []string
	for _, x := range d {
		got = append(got, x.Rule+"|"+x.Message+"|"+x.Path)
	}
	want := []string{
		"BR-62|Das Schema wird immer angegeben.|/r[1]",
		"|required attribute plain is missing|/r[1]",
		"R-LANG|The language is always stated.|/r[1]",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got %q", got)
	}
	// present attributes: valid
	if d = validateWith(t, sch, `<r scheme="s" plain="p" lang="en"/>`, ""); len(d) != 0 {
		t.Errorf("valid: %+v", d)
	}
}
