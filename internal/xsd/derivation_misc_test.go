package xsd

import "testing"

// Assorted derivation rules: open content in derivations (1.1 §3.4.6.2 /
// §3.4.6.3), attribute wildcard intersection and union (§3.10.6.3/4 and
// the 1.0 expressibility clause), block on elements/types against xsi:type
// (cvc-elt.4.3, Type Derivation OK), year zero (1.0 has none; 1.1 does),
// and typed values of list attributes inside assertions (1.1 §3.13.4.1).
func TestDerivationMisc(t *testing.T) {
	cases := []struct {
		name   string
		ver    int
		schema string
		valid  bool
		docs   []docCheck
	}{
		{"open-restriction-narrows", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:openContent mode="interleave"><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:restriction base="B"><xs:openContent mode="suffix"><xs:any namespace="urn:x" processContents="strict"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
		</xs:schema>`, true, nil},
		{"open-restriction-loosens-mode", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:openContent mode="suffix"><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:restriction base="B"><xs:openContent mode="interleave"><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
		</xs:schema>`, false, nil},
		{"open-restriction-widens-wildcard", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:openContent><xs:any namespace="urn:x" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:restriction base="B"><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
		</xs:schema>`, false, nil},
		{"open-restriction-over-closed-base", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:restriction base="B"><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
		</xs:schema>`, false, nil},
		{"open-restriction-removed", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:restriction base="B"><xs:openContent mode="none"/><xs:sequence><xs:element name="a"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
		  <xs:element name="d" type="D"/><xs:element name="b" type="B"/>
		</xs:schema>`, true, []docCheck{
			{`<d><a/></d>`, true},
			{`<d xmlns:p="urn:p"><a/><p:x/></d>`, false},
			{`<b xmlns:p="urn:p"><p:x/><a/></b>`, true},
		}},
		{"open-extension-narrows-mode", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:openContent mode="interleave"><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:extension base="B"><xs:openContent mode="suffix"><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="b"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
		</xs:schema>`, false, nil},
		{"open-extension-inherits", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:openContent mode="interleave"><xs:any namespace="##other" processContents="skip"/></xs:openContent><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:extension base="B"><xs:sequence><xs:element name="b"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
		  <xs:element name="d" type="D"/>
		</xs:schema>`, true, []docCheck{
			{`<d xmlns:p="urn:p"><a/><p:x/><b/></d>`, true},
			{`<d><a/><c/><b/></d>`, false},
		}},
		{"open-extension-over-closed", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:extension base="B"><xs:openContent mode="suffix"><xs:any namespace="##other" processContents="skip"/></xs:openContent><xs:sequence><xs:element name="b"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
		  <xs:element name="d" type="D"/>
		</xs:schema>`, true, []docCheck{
			{`<d xmlns:p="urn:p"><a/><b/><p:x/></d>`, true},
		}},

		// Attribute wildcards: the complete wildcard of a type is the
		// intersection of its own and its attribute groups' wildcards; an
		// extension's is the union with the base's.
		{"attr-wildcard-intersection", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:attributeGroup name="g"><xs:anyAttribute namespace="urn:a urn:b ##local" processContents="skip"/></xs:attributeGroup>
		  <xs:element name="e"><xs:complexType><xs:attributeGroup ref="g"/><xs:anyAttribute namespace="##other" processContents="skip"/></xs:complexType></xs:element>
		</xs:schema>`, true, []docCheck{
			{`<e xmlns:a="urn:a" a:x="1"/>`, true},
			{`<e x="1"/>`, false},
			{`<e xmlns:c="urn:c" c:x="1"/>`, false},
		}},
		{"attr-wildcard-intersection-groups", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:attributeGroup name="g1"><xs:anyAttribute namespace="urn:a urn:b" processContents="skip"/></xs:attributeGroup>
		  <xs:attributeGroup name="g2"><xs:anyAttribute namespace="urn:b urn:c" processContents="skip"/></xs:attributeGroup>
		  <xs:element name="e"><xs:complexType><xs:attributeGroup ref="g1"/><xs:attributeGroup ref="g2"/></xs:complexType></xs:element>
		</xs:schema>`, true, []docCheck{
			{`<e xmlns:b="urn:b" b:x="1"/>`, true},
			{`<e xmlns:a="urn:a" a:x="1"/>`, false},
		}},
		{"attr-wildcard-intersection-any", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:attributeGroup name="g"><xs:anyAttribute processContents="skip"/></xs:attributeGroup>
		  <xs:element name="e"><xs:complexType><xs:attributeGroup ref="g"/><xs:anyAttribute namespace="urn:a" processContents="skip"/></xs:complexType></xs:element>
		</xs:schema>`, true, []docCheck{
			{`<e xmlns:a="urn:a" a:x="1"/>`, true},
			{`<e xmlns:b="urn:b" b:x="1"/>`, false},
		}},
		{"attr-wildcard-union-extension", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:anyAttribute namespace="urn:a" processContents="skip"/></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:extension base="B"><xs:anyAttribute namespace="urn:b" processContents="skip"/></xs:extension></xs:complexContent></xs:complexType>
		  <xs:element name="d" type="D"/>
		</xs:schema>`, true, []docCheck{
			{`<d xmlns:a="urn:a" a:x="1"/>`, true},
			{`<d xmlns:b="urn:b" b:x="1"/>`, true},
			{`<d xmlns:c="urn:c" c:x="1"/>`, false},
		}},
		// 1.0 Attribute Wildcard Union: not(urn:t) ∪ {absent} has no 1.0
		// representation, so the schema is in error; 1.1 can express it.
		{"attr-wildcard-union-inexpressible-1.0", ver10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
		  <xs:complexType name="B"><xs:anyAttribute namespace="##other" processContents="skip"/></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:extension base="t:B"><xs:anyAttribute namespace="##local" processContents="skip"/></xs:extension></xs:complexContent></xs:complexType>
		</xs:schema>`, false, nil},
		{"attr-wildcard-union-expressible-1.1", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
		  <xs:complexType name="B"><xs:anyAttribute namespace="##other" processContents="skip"/></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:extension base="t:B"><xs:anyAttribute namespace="##local" processContents="skip"/></xs:extension></xs:complexContent></xs:complexType>
		  <xs:element name="d" type="t:D"/>
		</xs:schema>`, true, []docCheck{
			{`<t:d xmlns:t="urn:t" x="1"/>`, true},
			{`<t:d xmlns:t="urn:t" t:x="1"/>`, false},
		}},
		// 1.1: prohibited together with fixed is an error (src-attribute).
		{"attr-prohibited-fixed-1.1", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="c"><xs:attribute name="a" use="prohibited" fixed="x"/></xs:complexType></xs:schema>`, false, nil},

		// block against xsi:type (cvc-elt.4.3 with Type Derivation OK).
		{"block-restriction-on-element", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:sequence><xs:element name="a" minOccurs="0"/></xs:sequence></xs:complexType>
		  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>
		  <xs:complexType name="X"><xs:complexContent><xs:extension base="B"/></xs:complexContent></xs:complexType>
		  <xs:complexType name="XR"><xs:complexContent><xs:restriction base="X"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>
		  <xs:element name="e" type="B" block="restriction"/>
		  <xs:element name="f" type="B"/>
		</xs:schema>`, true, []docCheck{
			{`<e` + xsiDecl + ` xsi:type="R"/>`, false},
			{`<e` + xsiDecl + ` xsi:type="X"/>`, true},
			// XR derives from B by extension then restriction: blocked too.
			{`<e` + xsiDecl + ` xsi:type="XR"/>`, false},
			{`<f` + xsiDecl + ` xsi:type="XR"/>`, true},
		}},
		{"block-on-type-all", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B" block="#all"><xs:sequence><xs:element name="a" minOccurs="0"/></xs:sequence></xs:complexType>
		  <xs:complexType name="X"><xs:complexContent><xs:extension base="B"/></xs:complexContent></xs:complexType>
		  <xs:element name="e" type="B"/>
		</xs:schema>`, true, []docCheck{
			{`<e` + xsiDecl + ` xsi:type="X"/>`, false},
			{`<e` + xsiDecl + ` xsi:type="B"/>`, true},
		}},
		{"block-default", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" blockDefault="extension">
		  <xs:complexType name="B" block=""><xs:sequence><xs:element name="a" minOccurs="0"/></xs:sequence></xs:complexType>
		  <xs:complexType name="X"><xs:complexContent><xs:extension base="B"/></xs:complexContent></xs:complexType>
		  <xs:element name="e" type="B"/>
		  <xs:element name="g" type="B" block=""/>
		</xs:schema>`, true, []docCheck{
			{`<e` + xsiDecl + ` xsi:type="X"/>`, false},
			{`<g` + xsiDecl + ` xsi:type="X"/>`, true},
		}},
		{"block-simple-restriction", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:none">
		  <xs:simpleType name="S"><xs:restriction base="xs:int"><xs:maxInclusive value="5"/></xs:restriction></xs:simpleType>
		  <xs:element name="e" type="xs:int" block="restriction"/>
		</xs:schema>`, true, []docCheck{
			{`<e` + xsiDecl + ` xsi:type="S">3</e>`, false},
		}},

		// Year 0000: not a valid 1.0 year (Datatypes 1.0 §3.2.7), but 1.1
		// has a year zero (1.1 §3.3.7: 0000 is 1 BCE).
		{"year-zero-1.0", ver10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="d" type="xs:date"/><xs:element name="y" type="xs:gYear"/><xs:element name="t" type="xs:dateTime"/></xs:schema>`, true, []docCheck{
			{`<d>0000-01-01</d>`, false},
			{`<y>0000</y>`, false},
			{`<t>0000-01-01T00:00:00</t>`, false},
			{`<d>-0001-01-01</d>`, true},
		}},
		{"year-zero-1.1", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="d" type="xs:date"/><xs:element name="y" type="xs:gYear"/></xs:schema>`, true, []docCheck{
			{`<d>0000-01-01</d>`, true},
			{`<y>0000</y>`, true},
		}},

		// Typed values in assertions: a list-typed attribute atomizes to its
		// items; a simple-typed child to its typed value.
		{"assert-list-typed-value", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:simpleType name="L"><xs:list itemType="xs:int"/></xs:simpleType>
		  <xs:element name="e"><xs:complexType><xs:sequence><xs:element name="n" type="xs:decimal" maxOccurs="unbounded"/></xs:sequence>
		    <xs:attribute name="l" type="L"/>
		    <xs:assert test="count(data(@l)) = 3 and sum(data(@l)) = 6"/>
		    <xs:assert test="n[1] = 1.5"/></xs:complexType></xs:element>
		</xs:schema>`, true, []docCheck{
			{`<e l="1 2 3"><n>1.50</n></e>`, true},
			{`<e l="1 2"><n>1.5</n></e>`, false},
			{`<e l="1 2 3"><n>2</n></e>`, false},
		}},
		{"assert-union-typed-value", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:simpleType name="U"><xs:union memberTypes="xs:int xs:date"/></xs:simpleType>
		  <xs:element name="e"><xs:complexType><xs:sequence><xs:element name="u" type="U" maxOccurs="2"/></xs:sequence>
		    <xs:assert test="data(u[1]) instance of xs:int and data(u[2]) instance of xs:date"/></xs:complexType></xs:element>
		</xs:schema>`, true, []docCheck{
			{`<e><u>1</u><u>2020-01-01</u></e>`, true},
			{`<e><u>2020-01-01</u><u>1</u></e>`, false},
		}},
	}
	for _, tc := range cases {
		for _, v := range versionsOf(tc.ver % skipBug) {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if tc.ver >= skipBug {
					t.Skip("BUG: engine verdict disagrees with the spec")
				}
				if got := schemaVerdict(t, tc.schema, v); got != tc.valid {
					t.Fatalf("schema valid=%v, want %v", got, tc.valid)
				}
				for _, d := range tc.docs {
					if got := instanceVerdict(t, tc.schema, d.inst, v); got != d.valid {
						t.Errorf("%s: valid=%v, want %v", d.inst, got, d.valid)
					}
				}
			})
		}
	}
}
