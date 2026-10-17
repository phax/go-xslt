package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A schema spread over three files (xs:include + xs:import) comes back as ONE
// resolved model: shared types listed once, recursion cut by id, content
// models with their occurrence bounds.
func TestSchemaStructureResolvesFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("codes.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:c">
 <xs:simpleType name="Cur"><xs:restriction base="xs:string"><xs:enumeration value="EUR"/><xs:enumeration value="USD"/></xs:restriction></xs:simpleType></xs:schema>`)
	write("party.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:o="urn:o" targetNamespace="urn:o" elementFormDefault="qualified">
 <xs:complexType name="Party"><xs:sequence><xs:element name="Name" type="xs:string"/></xs:sequence></xs:complexType></xs:schema>`)
	main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:o="urn:o" xmlns:c="urn:c" targetNamespace="urn:o" elementFormDefault="qualified">
 <xs:include schemaLocation="party.xsd"/><xs:import namespace="urn:c" schemaLocation="codes.xsd"/>
 <xs:element name="Order"><xs:complexType><xs:sequence>
   <xs:element name="Buyer" type="o:Party"/>
   <xs:element name="Line" type="o:Line" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence><xs:attribute name="cur" type="c:Cur" use="required"/></xs:complexType></xs:element>
 <xs:complexType name="Line"><xs:sequence><xs:element name="Sub" type="o:Line" minOccurs="0"/></xs:sequence></xs:complexType>
</xs:schema>`
	m := (&XsltService{}).SchemaStructure(SchemaModelRequest{Schemas: []string{main}, BaseDir: dir, Version: "1.1"})
	if len(m.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", m.Diagnostics)
	}
	if len(m.Elements) != 1 || m.Elements[0].Name != "Order" {
		t.Fatalf("elements: %+v", m.Elements)
	}
	order := m.Types[m.Elements[0].TypeID]
	if order == nil || order.Model == nil || order.Model.Kind != "sequence" || len(order.Model.Children) != 2 {
		t.Fatalf("Order's model: %+v", order)
	}
	line := order.Model.Children[1]
	if line.Name != "Line" || line.Min != 0 || line.Max != -1 {
		t.Errorf("Line particle: %+v", line)
	}
	// the party type comes from the included file, the currency from the imported one
	if p := m.Types[order.Model.Children[0].TypeID]; p == nil || p.Model == nil || p.Model.Children[0].Name != "Name" {
		t.Errorf("Party from party.xsd: %+v", p)
	}
	cur := m.Types["{urn:c}Cur"]
	if cur == nil || len(cur.Enumeration) != 2 || order.Attributes[0].Use != "required" || order.Attributes[0].TypeID != cur.ID {
		t.Errorf("Cur from codes.xsd: %+v attrs=%+v", cur, order.Attributes)
	}
	// a type that contains itself refers to its own id (the view cuts the loop there)
	lt := m.Types["{urn:o}Line"]
	if lt == nil || lt.Model.Children[0].TypeID != lt.ID {
		t.Errorf("recursive Line: %+v", lt)
	}
	// a broken schema is a diagnostic, not a crash
	bad := (&XsltService{}).SchemaStructure(SchemaModelRequest{Schemas: []string{"<xs:schema"}})
	if len(bad.Diagnostics) == 0 || len(bad.Elements) != 0 {
		t.Errorf("bad: %+v", bad)
	}
}
