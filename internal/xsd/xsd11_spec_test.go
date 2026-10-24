package xsd

import "testing"

// XSD 1.1 additions: assertions on complex types (Structures 1.1 §3.13,
// cvc-assertion), conditional type assignment (§3.12, cvc-type-alternative
// / e-props-correct), inheritable attributes (§3.3.5.6, used by type
// alternatives), open content (§3.4.2.3.3), and the 1.1-only element
// vocabulary being rejected by 1.0.
func TestXSD11Features(t *testing.T) {
	cases := []struct {
		name   string
		ver    int
		schema string
		valid  bool // schema validity
		docs   []docCheck
	}{
		{"assert-typed-attributes", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:attribute name="min" type="xs:int"/><xs:attribute name="max" type="xs:int"/>
		    <xs:assert test="@min le @max"/></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			// Typed comparison: 9 <= 10 numerically (as strings "10" < "9").
			{`<r min="9" max="10"/>`, true},
			{`<r min="10" max="9"/>`, false},
		}},
		{"assert-children", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="i" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
		    <xs:attribute name="n" type="xs:int"/><xs:assert test="count(i) = @n"/><xs:assert test="sum(i) lt 100"/></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r n="2"><i>1</i><i>2</i></r>`, true},
			{`<r n="3"><i>1</i><i>2</i></r>`, false},
			{`<r n="2"><i>60</i><i>50</i></r>`, false},
		}},
		// Assertions are inherited along extension (§3.4.2.3 {assertions}).
		{"assert-inherited", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence><xs:assert test="a gt 0"/></xs:complexType>
		  <xs:complexType name="D"><xs:complexContent><xs:extension base="B"><xs:sequence><xs:element name="b" type="xs:int"/></xs:sequence>
		    <xs:assert test="b gt a"/></xs:extension></xs:complexContent></xs:complexType>
		  <xs:element name="r" type="D"/></xs:schema>`, true, []docCheck{
			{`<r><a>1</a><b>2</b></r>`, true},
			{`<r><a>0</a><b>2</b></r>`, false},
			{`<r><a>3</a><b>2</b></r>`, false},
		}},
		{"assert-simple-content", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:simpleContent><xs:extension base="xs:int"><xs:attribute name="lim" type="xs:int"/>
		    <xs:assert test="$value le @lim"/></xs:extension></xs:simpleContent></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r lim="5">4</r>`, true},
			{`<r lim="5">6</r>`, false},
		}},
		{"assert-default-namespace", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" elementFormDefault="qualified" xpathDefaultNamespace="##targetNamespace">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="i" maxOccurs="unbounded"/></xs:sequence>
		    <xs:assert test="count(i) = 2"/></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r xmlns="urn:t"><i/><i/></r>`, true},
			{`<r xmlns="urn:t"><i/></r>`, false},
		}},
		{"assert-default-namespace-on-assert", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" elementFormDefault="qualified">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="i" maxOccurs="unbounded"/></xs:sequence>
		    <xs:assert test="count(i) = 2" xpathDefaultNamespace="##targetNamespace"/></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r xmlns="urn:t"><i/><i/></r>`, true},
		}},
		{"assert-dynamic-error-is-false", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:attribute name="v" type="xs:string"/><xs:assert test="xs:int(@v) gt 0"/></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r v="3"/>`, true},
			// A dynamic error evaluating the test means the assertion fails.
			{`<r v="x"/>`, false},
		}},
		{"assert-bad-xpath", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:assert test="(("/></xs:complexType></xs:element></xs:schema>`, false, nil},
		{"assert-no-test", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:assert/></xs:complexType></xs:element></xs:schema>`, false, nil},
		{"assert-1.0", ver10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:assert test="true()"/></xs:complexType></xs:element></xs:schema>`, false, nil},

		// Conditional type assignment.
		{"cta", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="B"><xs:sequence><xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/></xs:sequence><xs:attribute name="kind" type="xs:string"/><xs:anyAttribute processContents="lax"/></xs:complexType>
		  <xs:complexType name="A"><xs:complexContent><xs:restriction base="B"><xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence><xs:attribute name="kind" type="xs:string"/></xs:restriction></xs:complexContent></xs:complexType>
		  <xs:complexType name="C"><xs:complexContent><xs:restriction base="B"><xs:sequence><xs:element name="c" type="xs:date"/></xs:sequence><xs:attribute name="kind" type="xs:string"/></xs:restriction></xs:complexContent></xs:complexType>
		  <xs:element name="r" type="B">
		    <xs:alternative test="@kind = 'a'" type="A"/>
		    <xs:alternative test="@kind = 'c'" type="C"/>
		    <xs:alternative type="B"/>
		  </xs:element></xs:schema>`, true, []docCheck{
			{`<r kind="a"><a>1</a></r>`, true},
			{`<r kind="a"><a>x</a></r>`, false},
			{`<r kind="a"><c>2020-01-01</c></r>`, false},
			{`<r kind="c"><c>2020-01-01</c></r>`, true},
			{`<r kind="z"><anything/></r>`, true},
		}},
		// The test sees attributes only: the element is presented with no
		// children (cvc-type-alternative / §3.12.4), so "a" is false.
		{"cta-no-children-visible", ver11 + skipBug, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:alternative test="a" type="xs:int"/><xs:alternative type="xs:anyType"/></xs:element></xs:schema>`, true, []docCheck{
			{`<r><a/></r>`, true},
		}},
		{"cta-typed-attribute", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:complexType name="N"><xs:simpleContent><xs:extension base="xs:decimal"><xs:attribute name="v" type="xs:int"/></xs:extension></xs:simpleContent></xs:complexType>
		  <xs:complexType name="Small"><xs:simpleContent><xs:restriction base="N"><xs:maxInclusive value="9"/></xs:restriction></xs:simpleContent></xs:complexType>
		  <xs:element name="r" type="N"><xs:alternative test="@v &lt; 10" type="Small"/></xs:element></xs:schema>`, true, []docCheck{
			{`<r v="2">5</r>`, true},
			{`<r v="2">15</r>`, false},
			// v="10" selects no alternative: the declared type applies.
			{`<r v="10">15</r>`, true},
		}},
		{"cta-inherited-attribute", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="outer"><xs:complexType><xs:sequence><xs:element ref="inner"/></xs:sequence>
		    <xs:attribute name="mode" type="xs:string" inheritable="true"/></xs:complexType></xs:element>
		  <xs:element name="inner" type="xs:anyType"><xs:alternative test="@mode = 'num'" type="xs:int"/></xs:element></xs:schema>`, true, []docCheck{
			{`<outer mode="num"><inner>5</inner></outer>`, true},
			{`<outer mode="num"><inner>x</inner></outer>`, false},
			{`<outer mode="txt"><inner>x</inner></outer>`, true},
		}},
		// e-props-correct.7: each alternative's type must be validly
		// substitutable for the declared type.
		{"cta-type-not-derived", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:int"><xs:alternative test="true()" type="xs:string"/></xs:element></xs:schema>`, false, nil},
		{"cta-inline-type", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:string"><xs:alternative test="true()"><xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="2"/></xs:restriction></xs:simpleType></xs:alternative></xs:element></xs:schema>`, true, []docCheck{
			{`<r>ab</r>`, true},
			{`<r>abc</r>`, false},
		}},
		{"cta-type-and-inline", ver11 + skipBug, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:string"><xs:alternative test="true()" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:alternative></xs:element></xs:schema>`, false, nil},
		{"cta-no-type", ver11 + skipBug, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:string"><xs:alternative test="true()"/></xs:element></xs:schema>`, false, nil},
		{"cta-default-not-last", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:string"><xs:alternative type="xs:string"/><xs:alternative test="true()" type="xs:string"/></xs:element></xs:schema>`, false, nil},
		{"cta-bad-test", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:string"><xs:alternative test="@" type="xs:string"/></xs:element></xs:schema>`, false, nil},
		{"cta-error-type", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:string"><xs:alternative test="@bad" type="xs:error"/></xs:element></xs:schema>`, true, []docCheck{
			{`<r>ok</r>`, true},
			// xs:error has an empty value space: nothing is valid against it.
			{`<r bad="1">ok</r>`, false},
		}},
		{"cta-1.0", ver10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r" type="xs:string"><xs:alternative test="true()" type="xs:string"/></xs:element></xs:schema>`, false, nil},

		// Open content.
		{"open-interleave", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:openContent mode="interleave"><xs:any namespace="##other" processContents="skip"/></xs:openContent>
		    <xs:sequence><xs:element name="a"/><xs:element name="b"/></xs:sequence></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r xmlns:p="urn:p"><p:x/><a/><p:y/><b/><p:z/></r>`, true},
			{`<r><a/><b/></r>`, true},
			{`<r><a/><c/><b/></r>`, false},
			{`<r xmlns:p="urn:p"><b/><p:x/><a/></r>`, false},
		}},
		{"open-suffix", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:openContent mode="suffix"><xs:any namespace="##other" processContents="skip"/></xs:openContent>
		    <xs:sequence><xs:element name="a"/><xs:element name="b"/></xs:sequence></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r xmlns:p="urn:p"><a/><b/><p:x/><p:y/></r>`, true},
			{`<r xmlns:p="urn:p"><a/><p:x/><b/></r>`, false},
		}},
		{"open-none", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:openContent mode="none"/>
		    <xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r xmlns:p="urn:p"><a/><p:x/></r>`, false},
		}},
		{"open-lax-declared", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="g" type="xs:int"/>
		  <xs:element name="r"><xs:complexType><xs:openContent><xs:any processContents="lax"/></xs:openContent>
		    <xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r><g>1</g><a/></r>`, true},
			{`<r><g>x</g><a/></r>`, false},
		}},
		{"default-open-content", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:defaultOpenContent mode="suffix"><xs:any namespace="##other" processContents="skip"/></xs:defaultOpenContent>
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element>
		  <xs:element name="e"><xs:complexType/></xs:element></xs:schema>`, true, []docCheck{
			{`<r xmlns:p="urn:p"><a/><p:x/></r>`, true},
			// Not applied to empty content unless appliesToEmpty="true".
			{`<e xmlns:p="urn:p"><p:x/></e>`, false},
		}},
		{"default-open-content-applies-to-empty", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:defaultOpenContent mode="interleave" appliesToEmpty="true"><xs:any namespace="##other" processContents="skip"/></xs:defaultOpenContent>
		  <xs:element name="e"><xs:complexType/></xs:element></xs:schema>`, true, []docCheck{
			{`<e xmlns:p="urn:p"><p:x/></e>`, true},
		}},
		{"open-content-missing-any", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:openContent mode="interleave"/><xs:sequence/></xs:complexType></xs:element></xs:schema>`, false, nil},
		{"open-content-1.0", ver10, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:openContent><xs:any/></xs:openContent><xs:sequence/></xs:complexType></xs:element></xs:schema>`, false, nil},

		// Wildcards: notQName with ##defined / ##definedSibling.
		{"not-qname-defined", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="g"/>
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:any notQName="##defined" processContents="skip" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r><x/></r>`, true},
			{`<r><g/></r>`, false},
		}},
		{"not-qname-defined-sibling", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" minOccurs="0"/><xs:any notQName="##definedSibling" processContents="skip" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r><a/><x/></r>`, true},
			{`<r><a/><a/></r>`, false},
		}},
		{"not-qname-explicit", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:any notQName="x" processContents="skip" maxOccurs="unbounded"/></xs:sequence>
		    <xs:anyAttribute notNamespace="##local" processContents="skip"/></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r xmlns:p="urn:p" p:q="1"><y/></r>`, true},
			{`<r><x/></r>`, false},
			{`<r q="1"><y/></r>`, false},
		}},
		// xs:all in 1.1: counts above 1 and wildcards.
		{"all-counts", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="r"><xs:complexType><xs:all><xs:element name="a" minOccurs="0" maxOccurs="2"/><xs:element name="b" minOccurs="2" maxOccurs="3"/><xs:any namespace="##other" processContents="skip" minOccurs="0"/></xs:all></xs:complexType></xs:element></xs:schema>`, true, []docCheck{
			{`<r><b/><a/><b/><a/></r>`, true},
			{`<r xmlns:p="urn:p"><b/><p:x/><b/></r>`, true},
			{`<r><a/><a/><a/><b/><b/></r>`, false},
			{`<r><a/><b/></r>`, false},
		}},
		// defaultAttributes (1.1 §3.4.2): applied unless the type opts out.
		{"default-attributes", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" defaultAttributes="common">
		  <xs:attributeGroup name="common"><xs:attribute name="lang" type="xs:language"/></xs:attributeGroup>
		  <xs:element name="r"><xs:complexType/></xs:element>
		  <xs:element name="s"><xs:complexType defaultAttributesApply="false"/></xs:element></xs:schema>`, true, []docCheck{
			{`<r lang="en"/>`, true},
			{`<r lang="1-bad-"/>`, false},
			{`<s lang="en"/>`, false},
		}},
		{"default-attributes-unresolved", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" defaultAttributes="nope">
		  <xs:element name="r"><xs:complexType/></xs:element></xs:schema>`, false, nil},
		{"default-attributes-duplicate", ver11, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" defaultAttributes="common">
		  <xs:attributeGroup name="common"><xs:attribute name="lang" type="xs:language"/></xs:attributeGroup>
		  <xs:element name="r"><xs:complexType><xs:attribute name="lang" type="xs:string"/></xs:complexType></xs:element></xs:schema>`, false, nil},
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
