package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

const msgSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:msg="https://github.com/tim-riep/go-xslt/xsd/messages" xmlns:o="urn:other" elementFormDefault="unqualified">
 <xs:complexType name="Base"><xs:sequence><xs:element name="Reason" minOccurs="0"/><xs:element name="Code" minOccurs="0"/></xs:sequence>
  <xs:assert test="exists(Reason) or exists(Code)"><xs:annotation><xs:appinfo>
   <o:message>ignored</o:message>
   <msg:message rule="BR-33">Reason missing.</msg:message>
   <msg:message rule="BR-33" xml:lang="de">Grund fehlt.</msg:message>
   <msg:message rule="BR-33-fr" xml:lang="fr-CA">Raison manquante.</msg:message>
  </xs:appinfo></xs:annotation></xs:assert></xs:complexType>
 <xs:complexType name="Ext"><xs:complexContent><xs:extension base="Base"><xs:sequence><xs:element name="X" minOccurs="0"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
 <xs:simpleType name="Pos"><xs:restriction base="xs:int"><xs:assertion test="$value gt 0"><xs:annotation><xs:appinfo><msg:message rule="POS">Must be positive.</msg:message></xs:appinfo></xs:annotation></xs:assertion></xs:restriction></xs:simpleType>
 <xs:element name="Doc"><xs:complexType><xs:sequence>
  <xs:element name="Allow" type="Ext" maxOccurs="unbounded"/><xs:element name="N" type="Pos" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`

func val(inst, lang string) engine.Diagnostic {
	r := engine.Validate(engine.ValidateRequest{Schemas: []string{msgSchema}, Instance: inst, Version: engine.XSD11, Language: lang})
	if r.Valid || len(r.Diagnostics) == 0 {
		return engine.Diagnostic{}
	}
	return r.Diagnostics[0]
}

func TestAssertionMessages(t *testing.T) {
	inst := "<Doc>\n<Allow><Reason/></Allow>\n  <Allow/></Doc>"
	for lang, want := range map[string]string{"": "Reason missing.", "de": "Grund fehlt.", "DE-at": "Grund fehlt.", "fr": "Raison manquante.", "it": "Reason missing."} {
		d := val(inst, lang)
		if d.Message != want {
			t.Errorf("lang %q: %q, want %q", lang, d.Message, want)
		}
		if d.Path != "/Doc[1]/Allow[2]" || d.Line != 3 || d.Col != 3 || d.Code != "cvc-assertion" {
			t.Errorf("lang %q: %+v", lang, d)
		}
	}
	if d := val(inst, "fr"); d.Rule != "BR-33-fr" {
		t.Errorf("rule %q", d.Rule)
	}
	d := val("<Doc><Allow><Code/></Allow><N>-1</N></Doc>", "")
	if d.Rule != "POS" || d.Message != "Must be positive." {
		t.Errorf("simple: %+v", d)
	}
}

const multiSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:msg="https://github.com/tim-riep/go-xslt/xsd/messages">
 <xs:complexType name="Item"><xs:sequence><xs:element name="Qty" type="xs:int"/><xs:element name="Note" minOccurs="0"/></xs:sequence>
  <xs:assert test="Qty gt 0"><xs:annotation><xs:appinfo><msg:message rule="R-QTY">Qty must be positive.</msg:message></xs:appinfo></xs:annotation></xs:assert>
  <xs:assert test="not(Note) or string-length(Note) lt 5"><xs:annotation><xs:appinfo><msg:message rule="R-NOTE">Note too long.</msg:message></xs:appinfo></xs:annotation></xs:assert>
 </xs:complexType>
 <xs:element name="Doc"><xs:complexType><xs:sequence>
  <xs:element name="Head"/><xs:element name="Item" type="Item" maxOccurs="unbounded"/></xs:sequence>
  <xs:assert test="count(Item) lt 3"><xs:annotation><xs:appinfo><msg:message rule="R-COUNT">Too many items.</msg:message></xs:appinfo></xs:annotation></xs:assert>
 </xs:complexType></xs:element>
</xs:schema>`

func rules(t *testing.T, inst string, max int) []string {
	r := engine.Validate(engine.ValidateRequest{Schemas: []string{multiSchema}, Instance: inst, Version: engine.XSD11, MaxErrors: max})
	var out []string
	for _, d := range r.Diagnostics {
		x := d.Rule
		if x == "" {
			x = d.Code
		}
		out = append(out, x+"@"+d.Path)
	}
	if r.Valid != (len(out) == 0) {
		t.Fatalf("Valid=%v with %d diagnostics", r.Valid, len(out))
	}
	return out
}

func TestAllErrorsReported(t *testing.T) {
	inst := "<Doc>\n<Head/>\n<Item><Qty>0</Qty><Note>toolong</Note></Item>\n<Item><Qty>x</Qty></Item>\n<Item><Qty>-1</Qty></Item></Doc>"
	got := rules(t, inst, 0)
	// Item[2]'s Qty is not an int: its own assertion cannot hold either.
	want := []string{"R-COUNT@/Doc[1]", "R-QTY@/Doc[1]/Item[1]", "R-NOTE@/Doc[1]/Item[1]", "R-QTY@/Doc[1]/Item[2]", "cvc-datatype-valid@/Doc[1]/Item[2]/Qty[1]", "R-QTY@/Doc[1]/Item[3]"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("got %v", got)
	}
	// missing mandatory element: content-model error + the assertions
	got = rules(t, "<Doc><Item><Qty>0</Qty></Item></Doc>", 0)
	t.Log(got)
	if len(got) != 2 || got[0] != "cvc-complex-type.2.4@/Doc[1]" && got[1] != "cvc-complex-type.2.4@/Doc[1]" {
		t.Fatalf("got %v", got)
	}
	if got = rules(t, inst, 2); len(got) != 2 {
		t.Fatalf("max: %v", got)
	}
	if got = rules(t, "<Doc><Head/><Item><Qty>1</Qty></Item></Doc>", 0); got != nil {
		t.Fatalf("valid: %v", got)
	}
}
