package xsd_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/tim-riep/go-xslt/xsd"
)

const mainXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" xmlns:c="urn:common" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:import namespace="urn:common" schemaLocation="common.xsd"/>
  <xs:simpleType name="Money"><xs:restriction base="c:Amount"><xs:minInclusive value="0"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="Codes"><xs:list itemType="xs:NMTOKEN"/></xs:simpleType>
  <xs:simpleType name="IdOrNum"><xs:union memberTypes="xs:ID xs:integer"/></xs:simpleType>
  <xs:simpleType name="Kind"><xs:restriction base="xs:string"><xs:enumeration value="a"/><xs:enumeration value="b"/></xs:restriction></xs:simpleType>
  <xs:complexType name="Line">
    <xs:sequence>
      <xs:element name="Desc" type="xs:string"/>
      <xs:element name="Price" type="t:Money" minOccurs="0"/>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="Note" type="xs:string"/>
        <xs:element ref="t:Tag"/>
      </xs:choice>
    </xs:sequence>
    <xs:attribute name="no" type="xs:positiveInteger" use="required"/>
    <xs:attribute name="kind" type="t:Kind" default="a"/>
    <xs:attribute name="old" type="xs:string" use="prohibited"/>
  </xs:complexType>
  <xs:complexType name="SpecialLine"><xs:complexContent><xs:extension base="t:Line">
    <xs:sequence><xs:element name="Extra" type="xs:date"/></xs:sequence>
  </xs:extension></xs:complexContent></xs:complexType>
  <xs:complexType name="WithText"><xs:simpleContent><xs:extension base="t:Money"><xs:attribute name="cur" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="Mix" mixed="true"><xs:sequence><xs:any namespace="##other" processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
  <xs:element name="Invoice">
    <xs:complexType><xs:sequence>
      <xs:element name="Line" type="t:Line" maxOccurs="unbounded"/>
      <xs:element name="Total" type="t:WithText"/>
      <xs:element name="Opt" type="t:Mix" minOccurs="0" nillable="true"/>
    </xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="Tag" type="xs:string" abstract="true"/>
  <xs:element name="RedTag" type="xs:string" substitutionGroup="t:Tag"/>
  <xs:element name="DarkRedTag" type="xs:string" substitutionGroup="t:RedTag"/>
</xs:schema>`

const commonXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:common">
  <xs:simpleType name="Amount"><xs:restriction base="xs:decimal"><xs:fractionDigits value="2"/></xs:restriction></xs:simpleType>
</xs:schema>`

func compile(t *testing.T) *xsd.Schema {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "common.xsd"), []byte(commonXSD), 0o644); err != nil {
		t.Fatal(err)
	}
	s, diags := xsd.Compile(xsd.Request{Schemas: []string{mainXSD}, BaseDir: dir, Version: xsd.V10})
	if s == nil {
		t.Fatalf("compile: %+v", diags)
	}
	return s
}

func name(local string) xml.Name { return xml.Name{Space: "urn:t", Local: local} }

func TestGlobalsAndTypes(t *testing.T) {
	s := compile(t)
	var got []string
	for _, e := range s.GlobalElements() {
		got = append(got, e.Name().Local)
	}
	if want := []string{"DarkRedTag", "Invoice", "RedTag", "Tag"}; len(got) != len(want) {
		t.Fatalf("global elements %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("global elements %v, want %v", got, want)
			}
		}
	}
	// BuiltinBase follows restriction chains, across the imported file.
	money := s.Type(name("Money"))
	if money == nil || !money.IsSimple() || money.BuiltinBase() != (xml.Name{Space: "http://www.w3.org/2001/XMLSchema", Local: "decimal"}) {
		t.Fatalf("Money: %+v base %v", money, money.BuiltinBase())
	}
	if money.Name() != name("Money") {
		t.Errorf("Money name %v", money.Name())
	}
	// A reference to a built-in type reports the built-in's own name.
	line := s.Type(name("Line")).(*xsd.ComplexType)
	desc := line.ContentParticles()[0].Term.(*xsd.Element)
	if desc.Name().Local != "Desc" || desc.Type().Name().Local != "string" || desc.Type().Name().Space != "http://www.w3.org/2001/XMLSchema" {
		t.Errorf("Desc type %v", desc.Type().Name())
	}
	// list and union
	codes := s.Type(name("Codes")).(*xsd.SimpleType)
	if codes.Variety() != xsd.VarietyList || codes.ItemType() == nil || codes.BuiltinBase().Local != "NMTOKEN" {
		t.Errorf("Codes: %v item %v base %v", codes.Variety(), codes.ItemType(), codes.BuiltinBase())
	}
	u := s.Type(name("IdOrNum")).(*xsd.SimpleType)
	if u.Variety() != xsd.VarietyUnion || len(u.MemberTypes()) != 2 || u.MemberTypes()[1].BuiltinBase().Local != "integer" {
		t.Errorf("IdOrNum: %v %v", u.Variety(), u.MemberTypes())
	}
	if en := s.Type(name("Kind")).(*xsd.SimpleType).Enumeration(); len(en) != 2 || en[0] != "a" || en[1] != "b" {
		t.Errorf("Kind enumeration %v", en)
	}
}

func TestContentModelCardinalityAndAttributes(t *testing.T) {
	s := compile(t)
	inv := s.Element(name("Invoice")).Type().(*xsd.ComplexType)
	if inv.Name() != (xml.Name{}) {
		t.Errorf("Invoice's type should be anonymous, got %v", inv.Name())
	}
	ps := inv.ContentParticles()
	if len(ps) != 3 {
		t.Fatalf("Invoice particles: %d", len(ps))
	}
	line := ps[0]
	if line.Term.(*xsd.Element).Name().Local != "Line" || line.MinOccurs != 1 || line.MaxOccurs != xsd.Unbounded {
		t.Errorf("Line particle %+v", line)
	}
	opt := ps[2]
	if opt.MinOccurs != 0 || opt.MaxOccurs != 1 || !opt.Term.(*xsd.Element).Nillable() {
		t.Errorf("Opt particle %+v", opt)
	}
	// Line: sequence(Desc, Price?, choice*(Note|Tag))
	lt := line.Term.(*xsd.Element).Type().(*xsd.ComplexType)
	lp := lt.ContentParticles()
	if len(lp) != 3 || lp[1].MinOccurs != 0 {
		t.Fatalf("Line content: %+v", lp)
	}
	choice := lp[2].Term.(*xsd.ModelGroup)
	if choice.Compositor != xsd.CompositorChoice || lp[2].MaxOccurs != xsd.Unbounded || len(choice.Particles) != 2 {
		t.Errorf("choice: %+v", lp[2])
	}
	if choice.Particles[1].Term.(*xsd.Element) != s.Element(name("Tag")) {
		t.Error("a ref'd element should be the very same *Element as the global declaration")
	}
	// attributes
	attrs := map[string]*xsd.AttributeUse{}
	for _, a := range lt.Attributes() {
		attrs[a.Name.Local] = a
	}
	if a := attrs["no"]; a == nil || a.Use != xsd.UseRequired || a.Type.BuiltinBase().Local != "positiveInteger" {
		t.Errorf("no: %+v", a)
	}
	if a := attrs["kind"]; a == nil || a.Use != xsd.UseOptional || !a.HasDefault || a.Default != "a" {
		t.Errorf("kind: %+v", a)
	}
	// A prohibited use is reported where the type itself declares it.
	if a := attrs["old"]; a == nil || a.Use != xsd.UseProhibited {
		t.Errorf("old: %+v", a)
	}
	// Extension: base content then own content, effective after derivation.
	sp := s.Type(name("SpecialLine")).(*xsd.ComplexType)
	if sp.Derivation() != "extension" || sp.Base().Name() != name("Line") {
		t.Errorf("SpecialLine derivation %q base %v", sp.Derivation(), sp.Base())
	}
	if pp := sp.ContentParticles(); len(pp) != 2 {
		t.Errorf("SpecialLine content should be (Line's model, Extra): %d particles", len(pp))
	}
	if len(sp.Attributes()) != len(lt.Attributes()) {
		t.Error("an extension inherits the base's attributes")
	}
}

func TestSimpleContentMixedWildcardAndSubstitution(t *testing.T) {
	s := compile(t)
	w := s.Type(name("WithText")).(*xsd.ComplexType)
	if w.ContentKind() != xsd.ContentSimple || w.SimpleContent() == nil || w.BuiltinBase().Local != "decimal" {
		t.Errorf("WithText: %v %v %v", w.ContentKind(), w.SimpleContent(), w.BuiltinBase())
	}
	if attrs := w.Attributes(); len(attrs) != 1 || attrs[0].Name.Local != "cur" {
		t.Errorf("WithText attributes %+v", w.Attributes())
	}
	m := s.Type(name("Mix")).(*xsd.ComplexType)
	if !m.Mixed() || m.ContentKind() != xsd.ContentMixed {
		t.Error("Mix should be mixed")
	}
	wc := m.ContentParticles()[0].Term.(*xsd.Wildcard)
	if wc.Mode != "other" || wc.ProcessContents != "lax" {
		t.Errorf("wildcard %+v", wc)
	}
	// substitution groups: transitive, head excluded
	var members []string
	for _, e := range s.SubstitutionMembers(name("Tag")) {
		members = append(members, e.Name().Local)
	}
	if len(members) != 2 || members[0] != "DarkRedTag" || members[1] != "RedTag" {
		t.Errorf("members of Tag: %v", members)
	}
	if g := s.Element(name("DarkRedTag")).SubstitutionGroups(); len(g) != 1 || g[0] != name("RedTag") {
		t.Errorf("DarkRedTag groups %v", g)
	}
	if !s.Element(name("Tag")).Abstract() {
		t.Error("Tag is abstract")
	}
}

func TestCompileErrors(t *testing.T) {
	if s, d := xsd.Compile(xsd.Request{Schemas: []string{"<not-a-schema/>"}}); s != nil || len(d) == 0 {
		t.Errorf("non-schema: %v %v", s, d)
	}
	if s, d := xsd.Compile(xsd.Request{}); s != nil || len(d) == 0 {
		t.Errorf("no documents: %v %v", s, d)
	}
	bad := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a" type="xs:nope"/></xs:schema>`
	if s, d := xsd.Compile(xsd.Request{Schemas: []string{bad}}); s != nil || len(d) == 0 || d[0].Message == "" {
		t.Errorf("unknown built-in type: %v %v", s, d)
	}
	// A reference to a type the schema does not define: a usable Schema plus a warning.
	dangling := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a" type="nope"/></xs:schema>`
	s, d := xsd.Compile(xsd.Request{Schemas: []string{dangling}})
	if s == nil || len(d) != 1 || d[0].Severity != "warning" || d[0].Code != "src-resolve" {
		t.Errorf("dangling type: schema=%v diags=%+v", s != nil, d)
	}
}

// TestConcurrentWalks: one Schema, many goroutines walking it (run with -race).
func TestConcurrentWalks(t *testing.T) {
	s := compile(t)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 50; n++ {
				inv := s.Element(name("Invoice")).Type().(*xsd.ComplexType)
				for _, p := range inv.ContentParticles() {
					if e, ok := p.Term.(*xsd.Element); ok {
						_ = e.Type().BuiltinBase()
						if ct, ok := e.Type().(*xsd.ComplexType); ok {
							_ = ct.Attributes()
							_ = ct.ContentParticles()
						}
					}
				}
				_ = s.SubstitutionMembers(name("Tag"))
				_ = s.GlobalTypes()
			}
		}()
	}
	wg.Wait()
}
