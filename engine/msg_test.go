package engine_test

import (
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
