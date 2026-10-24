package xsd

import (
	"strings"
	"testing"
)

// reprSchema wraps body in an xs:schema element unless it already is one.
func reprSchema(body string) string {
	if strings.HasPrefix(strings.TrimSpace(body), "<xs:schema") {
		return body
	}
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xs:schema>`
}

// Schema documents whose validity is decided by the XML representation of
// schema components: the schema for schemas (Structures Appendix A) and the
// Schema Representation Constraints (src-*) and component constraints of
// XSD 1.0 / 1.1 Part 1 §3.
func TestSchemaRepresentation(t *testing.T) {
	const tns = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">`
	cases := []struct {
		name  string
		ver   int
		body  string
		valid bool
	}{
		// Attributes of schema elements (schema for schemas).
		{"xsd-namespace-attribute", verBoth, `<xs:element name="e" xs:type="xs:string"/>`, false},
		{"foreign-attribute", verBoth, `<xs:element name="e" xmlns:f="urn:f" f:x="1"/>`, true},
		{"unknown-unqualified-attribute", verBoth, `<xs:element name="e" colour="red"/>`, false},
		{"id-not-ncname", verBoth, `<xs:element name="e" id="1a"/>`, false},
		{"id-duplicate", verBoth, `<xs:element name="e" id="x"/><xs:element name="f" id="x"/>`, false},
		{"id-distinct", verBoth, `<xs:element name="e" id="x"/><xs:element name="f" id="y"/>`, true},
		{"boolean-bad", verBoth, `<xs:element name="e" nillable="yes"/>`, false},
		{"boolean-numeric", verBoth, `<xs:element name="e" nillable="1"/>`, true},
		{"element-block-list", verBoth, `<xs:element name="e" block="list"/>`, false},
		{"element-block-ok", verBoth, `<xs:element name="e" block="substitution extension"/>`, true},
		{"element-final-substitution", verBoth, `<xs:element name="e" final="substitution"/>`, false},
		{"complextype-block-substitution", verBoth, `<xs:complexType name="c" block="substitution"/>`, false},
		{"complextype-final-all", verBoth, `<xs:complexType name="c" final="#all"/>`, true},
		{"simpletype-final-list-union", verBoth, `<xs:simpleType name="s" final="list union"><xs:restriction base="xs:string"/></xs:simpleType>`, true},
		// simpleDerivationSet gained "extension" in 1.1.
		{"simpletype-final-extension-1.0", ver10, `<xs:simpleType name="s" final="extension"><xs:restriction base="xs:string"/></xs:simpleType>`, false},
		{"simpletype-final-extension-1.1", ver11, `<xs:simpleType name="s" final="extension"><xs:restriction base="xs:string"/></xs:simpleType>`, true},
		{"block-default-list", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" blockDefault="list"/>`, false},
		{"final-default-substitution", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" finalDefault="substitution"/>`, false},
		{"final-default-list", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" finalDefault="list union restriction"/>`, true},
		{"process-contents-bad", verBoth, `<xs:complexType name="c"><xs:sequence><xs:any processContents="never"/></xs:sequence></xs:complexType>`, false},
		{"use-bad", verBoth, `<xs:complexType name="c"><xs:attribute name="a" use="sometimes"/></xs:complexType>`, false},
		{"form-bad", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" form="global"/></xs:sequence></xs:complexType>`, false},
		{"element-form-default-bad", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="yes"/>`, false},
		{"unknown-xs-element", verBoth, `<xs:elephant name="e"/>`, false},

		// complexType children (schema for schemas: annotation?, then a
		// content alternative, then attributes).
		{"ct-annotation-after-content", verBoth, `<xs:complexType name="c"><xs:sequence/><xs:annotation/></xs:complexType>`, false},
		{"ct-two-annotations", verBoth, `<xs:complexType name="c"><xs:annotation/><xs:annotation/></xs:complexType>`, false},
		{"ct-direct-element", verBoth, `<xs:complexType name="c"><xs:element name="a"/></xs:complexType>`, false},
		{"ct-attribute-beside-simplecontent", verBoth, `<xs:complexType name="c"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent><xs:attribute name="a"/></xs:complexType>`, false},
		{"ct-simple-and-complex-content", verBoth, `<xs:complexType name="c"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent><xs:complexContent><xs:extension base="xs:anyType"/></xs:complexContent></xs:complexType>`, false},
		{"ct-content-and-group", verBoth, `<xs:complexType name="c"><xs:complexContent><xs:extension base="xs:anyType"/></xs:complexContent><xs:sequence/></xs:complexType>`, false},
		{"ct-two-groups", verBoth, `<xs:complexType name="c"><xs:sequence/><xs:choice/></xs:complexType>`, false},
		{"ct-attribute-before-group", verBoth, `<xs:complexType name="c"><xs:attribute name="a"/><xs:sequence/></xs:complexType>`, false},
		{"ct-ok", verBoth, `<xs:complexType name="c"><xs:annotation/><xs:sequence><xs:element name="a"/></xs:sequence><xs:attribute name="b"/><xs:anyAttribute/></xs:complexType>`, true},

		// simpleContent (src-ct.2, derivation-ok-restriction.5.1).
		{"sc-empty", verBoth, `<xs:complexType name="c"><xs:simpleContent/></xs:complexType>`, false},
		{"sc-restrict-anytype-without-simpletype", verBoth, `<xs:complexType name="c"><xs:simpleContent><xs:restriction base="xs:anyType"/></xs:simpleContent></xs:complexType>`, false},
		{"sc-restrict-anytype-with-simpletype", verBoth, `<xs:complexType name="c"><xs:simpleContent><xs:restriction base="xs:anyType"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType>`, true},
		{"sc-extension-of-element-only", verBoth, `<xs:complexType name="b"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType><xs:complexType name="c"><xs:simpleContent><xs:extension base="b"/></xs:simpleContent></xs:complexType>`, false},
		{"sc-facet-after-attribute", verBoth, `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:int"/></xs:simpleContent></xs:complexType><xs:complexType name="c"><xs:simpleContent><xs:restriction base="b"><xs:attribute name="a"/><xs:maxInclusive value="3"/></xs:restriction></xs:simpleContent></xs:complexType>`, false},
		{"sc-simpletype-after-attribute", verBoth, `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:int"/></xs:simpleContent></xs:complexType><xs:complexType name="c"><xs:simpleContent><xs:restriction base="b"><xs:attribute name="a"/><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType>`, false},
		{"sc-inline-type-not-derived", verBoth + skipBug, `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:decimal"/></xs:simpleContent></xs:complexType><xs:complexType name="c"><xs:simpleContent><xs:restriction base="b"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType>`, false},
		{"sc-inline-type-derived", verBoth, `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:decimal"/></xs:simpleContent></xs:complexType><xs:complexType name="c"><xs:simpleContent><xs:restriction base="b"><xs:simpleType><xs:restriction base="xs:integer"/></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType>`, true},
		// 1.1: {final} of a simple type may contain extension, which forbids
		// using it as the base of a simpleContent extension (cos-ct-extends 1.1).
		{"sc-extension-of-final-simple", ver11, `<xs:simpleType name="s" final="extension"><xs:restriction base="xs:string"/></xs:simpleType><xs:complexType name="c"><xs:simpleContent><xs:extension base="s"/></xs:simpleContent></xs:complexType>`, false},

		// complexContent.
		{"cc-empty", verBoth, `<xs:complexType name="c"><xs:complexContent/></xs:complexType>`, false},
		{"cc-extension-of-simple-type", verBoth, `<xs:complexType name="c"><xs:complexContent><xs:extension base="xs:string"/></xs:complexContent></xs:complexType>`, false},
		{"cc-extension-of-simple-content", verBoth, `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType><xs:complexType name="c"><xs:complexContent><xs:extension base="b"><xs:sequence><xs:element name="a"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>`, false},
		{"cc-mixed-conflict-1.1", ver11, `<xs:complexType name="c" mixed="true"><xs:complexContent mixed="false"><xs:extension base="xs:anyType"/></xs:complexContent></xs:complexType>`, false},
		{"cc-mixed-agree", verBoth, `<xs:complexType name="c" mixed="true"><xs:complexContent mixed="true"><xs:extension base="xs:anyType"/></xs:complexContent></xs:complexType>`, true},
		// cos-ct-extends.1.4.3.2.2.1: element-only extension of a mixed base.
		{"cc-extend-mixed-element-only", verBoth, `<xs:complexType name="b" mixed="true"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType><xs:complexType name="c"><xs:complexContent><xs:extension base="b"><xs:sequence><xs:element name="z"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>`, false},
		{"cc-extend-mixed-empty", verBoth, `<xs:complexType name="b" mixed="true"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType><xs:complexType name="c"><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType>`, true},
		{"cc-extend-element-only-mixed", verBoth, `<xs:complexType name="b"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType><xs:complexType name="c" mixed="true"><xs:complexContent><xs:extension base="b"><xs:sequence><xs:element name="z"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>`, false},

		// Particles and model groups.
		{"attribute-in-sequence", verBoth, `<xs:complexType name="c"><xs:sequence><xs:attribute name="a"/></xs:sequence></xs:complexType>`, false},
		{"group-ref-missing", verBoth, `<xs:complexType name="c"><xs:group/></xs:complexType>`, false},
		{"group-ref-unresolved", verBoth, `<xs:complexType name="c"><xs:group ref="nope"/></xs:complexType>`, false},
		{"group-ref-not-imported", verBoth, `<xs:complexType name="c" xmlns:o="urn:o"><xs:group ref="o:g"/></xs:complexType>`, false},
		{"group-self-cycle", verBoth, `<xs:group name="g"><xs:sequence><xs:group ref="g"/></xs:sequence></xs:group>`, false},
		{"group-indirect-cycle", verBoth, `<xs:group name="g"><xs:sequence><xs:group ref="h"/></xs:sequence></xs:group><xs:group name="h"><xs:choice><xs:group ref="g"/></xs:choice></xs:group>`, false},
		{"group-ok", verBoth, `<xs:group name="g"><xs:sequence><xs:element name="a"/></xs:sequence></xs:group><xs:complexType name="c"><xs:sequence><xs:group ref="g" maxOccurs="2"/></xs:sequence></xs:complexType>`, true},
		{"group-without-name", verBoth, `<xs:group><xs:sequence/></xs:group>`, false},
		{"group-two-compositors", verBoth, `<xs:group name="g"><xs:sequence/><xs:choice/></xs:group>`, false},
		{"group-def-with-occurs", verBoth, `<xs:group name="g" maxOccurs="2"><xs:sequence/></xs:group>`, false},
		{"min-greater-than-max", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" minOccurs="3" maxOccurs="2"/></xs:sequence></xs:complexType>`, false},
		{"max-occurs-bad", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" maxOccurs="many"/></xs:sequence></xs:complexType>`, false},
		{"min-occurs-negative", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" minOccurs="-1"/></xs:sequence></xs:complexType>`, false},
		// cos-all-limited: an all group is the whole content model, max 1.
		{"all-in-sequence", verBoth, `<xs:complexType name="c"><xs:sequence><xs:all><xs:element name="a"/></xs:all></xs:sequence></xs:complexType>`, false},
		{"all-ref-in-sequence", verBoth, `<xs:group name="g"><xs:all><xs:element name="a"/></xs:all></xs:group><xs:complexType name="c"><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType>`, false},
		{"all-max-two", verBoth, `<xs:complexType name="c"><xs:all maxOccurs="2"><xs:element name="a"/></xs:all></xs:complexType>`, false},
		{"all-child-max-two-1.0", ver10, `<xs:complexType name="c"><xs:all><xs:element name="a" maxOccurs="2"/></xs:all></xs:complexType>`, false},
		{"all-child-max-two-1.1", ver11, `<xs:complexType name="c"><xs:all><xs:element name="a" maxOccurs="2"/></xs:all></xs:complexType>`, true},
		{"all-with-any-1.0", ver10, `<xs:complexType name="c"><xs:all><xs:any/></xs:all></xs:complexType>`, false},
		{"all-with-any-1.1", ver11, `<xs:complexType name="c"><xs:all><xs:any/></xs:all></xs:complexType>`, true},
		{"all-with-choice", verBoth, `<xs:complexType name="c"><xs:all><xs:choice/></xs:all></xs:complexType>`, false},
		// Element Declarations Consistent (cos-element-consistent).
		{"edc-same-name-different-type", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" type="xs:int"/><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`, false},
		{"edc-same-name-same-type", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" type="xs:int"/><xs:element name="b"/><xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType>`, true},
		{"edc-nested-group", verBoth, `<xs:complexType name="c"><xs:choice><xs:element name="a" type="xs:int"/><xs:sequence><xs:element name="a" type="xs:date"/></xs:sequence></xs:choice></xs:complexType>`, false},

		// Local element declarations (src-element).
		{"local-element-ref-and-name", verBoth, `<xs:element name="g"/><xs:complexType name="c"><xs:sequence><xs:element ref="g" name="x"/></xs:sequence></xs:complexType>`, false},
		{"local-element-no-name", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element type="xs:string"/></xs:sequence></xs:complexType>`, false},
		{"local-element-ref-with-type", verBoth, `<xs:element name="g"/><xs:complexType name="c"><xs:sequence><xs:element ref="g" type="xs:string"/></xs:sequence></xs:complexType>`, false},
		{"local-element-ref-with-nillable", verBoth, `<xs:element name="g"/><xs:complexType name="c"><xs:sequence><xs:element ref="g" nillable="true"/></xs:sequence></xs:complexType>`, false},
		{"local-element-ref-unresolved", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element ref="g"/></xs:sequence></xs:complexType>`, false},
		{"local-element-type-and-inline", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element></xs:sequence></xs:complexType>`, false},
		{"local-element-abstract", verBoth, `<xs:complexType name="c"><xs:sequence><xs:element name="a" abstract="true"/></xs:sequence></xs:complexType>`, false},
		{"local-element-subst-group", verBoth, `<xs:element name="h"/><xs:complexType name="c"><xs:sequence><xs:element name="a" substitutionGroup="h"/></xs:sequence></xs:complexType>`, false},
		{"global-element-with-occurs", verBoth, `<xs:element name="g" minOccurs="0"/>`, false},
		{"global-element-with-ref", verBoth, `<xs:element name="g"/><xs:element ref="g"/>`, false},
		{"global-element-form", verBoth, `<xs:element name="g" form="qualified"/>`, false},
		{"element-default-and-fixed", verBoth, `<xs:element name="g" type="xs:string" default="a" fixed="a"/>`, false},
		{"element-default-not-of-type", verBoth, `<xs:element name="g" type="xs:int" default="x"/>`, false},
		{"element-fixed-of-type", verBoth, `<xs:element name="g" type="xs:int" fixed=" 7 "/>`, true},
		{"element-default-element-only", verBoth, `<xs:element name="g" default="x"><xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element>`, false},
		{"element-default-mixed-emptiable", verBoth, `<xs:element name="g" default="x"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`, true},
		// 1.0 e-props-correct.4 (the ID clause): no value constraint on an
		// ID-typed element; removed in 1.1.
		{"element-id-default-1.0", ver10, `<xs:element name="g" type="xs:ID" default="a"/>`, false},
		{"element-id-default-1.1", ver11, `<xs:element name="g" type="xs:ID" default="a"/>`, true},
		// 1.1 targetNamespace on local declarations: only below a complex
		// restriction whose base is not xs:anyType, and never with @form.
		{"local-tns-1.0", ver10, tns + `<xs:complexType name="c"><xs:sequence><xs:element name="a" targetNamespace="urn:t"/></xs:sequence></xs:complexType></xs:schema>`, false},
		{"local-tns-outside-restriction-1.1", ver11, tns + `<xs:complexType name="c"><xs:sequence><xs:element name="a" targetNamespace="urn:x"/></xs:sequence></xs:complexType></xs:schema>`, false},
		{"local-tns-with-form-1.1", ver11 + skipBug, tns + `<xs:complexType name="c"><xs:sequence><xs:element name="a" targetNamespace="urn:t" form="qualified"/></xs:sequence></xs:complexType></xs:schema>`, false},
		{"local-tns-same-1.1", ver11, tns + `<xs:complexType name="c"><xs:sequence><xs:element name="a" targetNamespace="urn:t"/></xs:sequence></xs:complexType></xs:schema>`, true},
		{"local-tns-in-restriction-1.1", ver11, tns + `<xs:complexType name="b"><xs:sequence><xs:any namespace="##any" processContents="lax"/></xs:sequence></xs:complexType>
		  <xs:complexType name="c"><xs:complexContent><xs:restriction base="t:b"><xs:sequence><xs:element name="a" targetNamespace="urn:x"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:schema>`, true},
		{"local-attr-tns-with-form-1.1", ver11 + skipBug, tns + `<xs:complexType name="c"><xs:attribute name="a" targetNamespace="urn:t" form="qualified"/></xs:complexType></xs:schema>`, false},

		// Wildcards (src-wildcard; 1.1 notNamespace / notQName).
		{"wildcard-any-in-list", verBoth, `<xs:complexType name="c"><xs:sequence><xs:any namespace="##any urn:a"/></xs:sequence></xs:complexType>`, false},
		{"wildcard-other-in-list", verBoth, `<xs:complexType name="c"><xs:sequence><xs:any namespace="urn:a ##other"/></xs:sequence></xs:complexType>`, false},
		{"wildcard-list-ok", verBoth, `<xs:complexType name="c"><xs:sequence><xs:any namespace="urn:a ##local ##targetNamespace"/></xs:sequence></xs:complexType>`, true},
		{"wildcard-bad-keyword", verBoth, `<xs:complexType name="c"><xs:sequence><xs:any namespace="##foo"/></xs:sequence></xs:complexType>`, false},
		{"wildcard-ns-and-notns-1.1", ver11, `<xs:complexType name="c"><xs:sequence><xs:any namespace="urn:a" notNamespace="urn:b"/></xs:sequence></xs:complexType>`, false},
		{"wildcard-notns-1.0", ver10 + skipBug, `<xs:complexType name="c"><xs:sequence><xs:any notNamespace="urn:b"/></xs:sequence></xs:complexType>`, false},
		{"wildcard-notns-keywords-1.1", ver11, `<xs:complexType name="c"><xs:sequence><xs:any notNamespace="##local ##targetNamespace urn:b"/></xs:sequence></xs:complexType>`, true},
		{"wildcard-notns-any-1.1", ver11, `<xs:complexType name="c"><xs:sequence><xs:any notNamespace="##any"/></xs:sequence></xs:complexType>`, false},
		{"wildcard-notqname-defined-1.1", ver11, `<xs:complexType name="c"><xs:sequence><xs:any notQName="##defined ##definedSibling"/></xs:sequence></xs:complexType>`, true},
		{"anyattribute-notqname-sibling-1.1", ver11, `<xs:complexType name="c"><xs:anyAttribute notQName="##definedSibling"/></xs:complexType>`, false},
		{"anyattribute-notqname-defined-1.1", ver11, `<xs:complexType name="c"><xs:anyAttribute notQName="##defined"/></xs:complexType>`, true},
		{"wildcard-notqname-bad-1.1", ver11, `<xs:complexType name="c"><xs:sequence><xs:any notQName="1x"/></xs:sequence></xs:complexType>`, false},
		{"wildcard-notqname-1.0", ver10 + skipBug, `<xs:complexType name="c"><xs:sequence><xs:any notQName="a"/></xs:sequence></xs:complexType>`, false},
		{"anyattribute-twice", verBoth, `<xs:complexType name="c"><xs:anyAttribute/><xs:anyAttribute/></xs:complexType>`, false},
		{"anyattribute-before-attribute", verBoth, `<xs:complexType name="c"><xs:anyAttribute/><xs:attribute name="a"/></xs:complexType>`, false},

		// Attribute declarations and uses (src-attribute, ct-props-correct).
		{"attr-default-and-fixed", verBoth, `<xs:attribute name="a" default="x" fixed="x"/>`, false},
		{"attr-default-required", verBoth, `<xs:complexType name="c"><xs:attribute name="a" default="x" use="required"/></xs:complexType>`, false},
		{"attr-fixed-required", verBoth, `<xs:complexType name="c"><xs:attribute name="a" fixed="x" use="required"/></xs:complexType>`, true},
		{"attr-ref-and-type", verBoth, `<xs:attribute name="g"/><xs:complexType name="c"><xs:attribute ref="g" type="xs:string"/></xs:complexType>`, false},
		{"attr-ref-and-name", verBoth, `<xs:attribute name="g"/><xs:complexType name="c"><xs:attribute ref="g" name="h"/></xs:complexType>`, false},
		{"attr-ref-unresolved", verBoth, `<xs:complexType name="c"><xs:attribute ref="nope"/></xs:complexType>`, false},
		{"attr-no-name", verBoth, `<xs:complexType name="c"><xs:attribute type="xs:string"/></xs:complexType>`, false},
		{"attr-type-and-inline", verBoth, `<xs:attribute name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`, false},
		{"attr-complex-type", verBoth, `<xs:complexType name="ct"/><xs:attribute name="a" type="ct"/>`, false},
		{"attr-name-xmlns", verBoth, `<xs:attribute name="xmlns"/>`, false},
		{"attr-tns-xsi", verBoth, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="http://www.w3.org/2001/XMLSchema-instance"><xs:attribute name="foo"/></xs:schema>`, false},
		{"attr-default-not-of-type", verBoth, `<xs:attribute name="a" type="xs:boolean" default="maybe"/>`, false},
		{"attr-global-use", verBoth, `<xs:attribute name="a" use="optional"/>`, false},
		{"attr-duplicate-use", verBoth, `<xs:complexType name="c"><xs:attribute name="a"/><xs:attribute name="a"/></xs:complexType>`, false},
		{"attr-duplicate-via-group", verBoth, `<xs:attributeGroup name="g"><xs:attribute name="a"/></xs:attributeGroup><xs:complexType name="c"><xs:attribute name="a"/><xs:attributeGroup ref="g"/></xs:complexType>`, false},
		{"attr-use-fixed-vs-decl-fixed", verBoth, `<xs:attribute name="g" type="xs:string" fixed="x"/><xs:complexType name="c"><xs:attribute ref="g" fixed="y"/></xs:complexType>`, false},
		// ct-props-correct.5 (1.0 only): two ID attributes on one type.
		{"two-id-attributes-1.0", ver10, `<xs:complexType name="c"><xs:attribute name="a" type="xs:ID"/><xs:attribute name="b" type="xs:ID"/></xs:complexType>`, false},
		{"two-id-attributes-1.1", ver11, `<xs:complexType name="c"><xs:attribute name="a" type="xs:ID"/><xs:attribute name="b" type="xs:ID"/></xs:complexType>`, true},
		{"id-attribute-fixed-1.0", ver10, `<xs:attribute name="a" type="xs:ID" fixed="x"/>`, false},

		// Attribute groups.
		{"attrgroup-unresolved", verBoth, `<xs:complexType name="c"><xs:attributeGroup ref="nope"/></xs:complexType>`, false},
		{"attrgroup-cycle-1.0", ver10, `<xs:attributeGroup name="g"><xs:attributeGroup ref="h"/></xs:attributeGroup><xs:attributeGroup name="h"><xs:attributeGroup ref="g"/></xs:attributeGroup>`, false},
		{"attrgroup-element-child", verBoth, `<xs:attributeGroup name="g"><xs:element name="e"/></xs:attributeGroup>`, false},

		// Simple types (src-simple-type, src-list-itemType-or-simpleType,
		// src-union-memberTypes-or-simpleTypes).
		{"st-no-derivation", verBoth, `<xs:simpleType name="s"/>`, false},
		{"st-two-derivations", verBoth, `<xs:simpleType name="s"><xs:restriction base="xs:string"/><xs:list itemType="xs:int"/></xs:simpleType>`, false},
		{"st-restriction-base-and-inline", verBoth, `<xs:simpleType name="s"><xs:restriction base="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:restriction></xs:simpleType>`, false},
		{"st-restriction-neither", verBoth, `<xs:simpleType name="s"><xs:restriction/></xs:simpleType>`, false},
		{"st-list-both", verBoth, `<xs:simpleType name="s"><xs:list itemType="xs:int"><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:list></xs:simpleType>`, false},
		{"st-list-neither", verBoth, `<xs:simpleType name="s"><xs:list/></xs:simpleType>`, false},
		{"st-list-of-list", verBoth, `<xs:simpleType name="l"><xs:list itemType="xs:int"/></xs:simpleType><xs:simpleType name="s"><xs:list itemType="l"/></xs:simpleType>`, false},
		{"st-union-neither", verBoth, `<xs:simpleType name="s"><xs:union/></xs:simpleType>`, false},
		{"st-union-ok", verBoth, `<xs:simpleType name="s"><xs:union memberTypes="xs:int"><xs:simpleType><xs:restriction base="xs:date"/></xs:simpleType></xs:union></xs:simpleType>`, true},
		{"st-restrict-complex", verBoth, `<xs:complexType name="ct"/><xs:simpleType name="s"><xs:restriction base="ct"/></xs:simpleType>`, false},
		{"st-restrict-anysimpletype", verBoth, `<xs:simpleType name="s"><xs:restriction base="xs:anySimpleType"/></xs:simpleType>`, false},
		{"st-restrict-final", verBoth, `<xs:simpleType name="b" final="restriction"><xs:restriction base="xs:string"/></xs:simpleType><xs:simpleType name="s"><xs:restriction base="b"/></xs:simpleType>`, false},
		{"st-list-of-final", verBoth, `<xs:simpleType name="b" final="list"><xs:restriction base="xs:string"/></xs:simpleType><xs:simpleType name="s"><xs:list itemType="b"/></xs:simpleType>`, false},
		{"st-union-of-final", verBoth, `<xs:simpleType name="b" final="union"><xs:restriction base="xs:string"/></xs:simpleType><xs:simpleType name="s"><xs:union memberTypes="b"/></xs:simpleType>`, false},
		{"st-self-cycle", verBoth, `<xs:simpleType name="s"><xs:restriction base="s"/></xs:simpleType>`, false},
		{"st-union-cycle", ver11, `<xs:simpleType name="s"><xs:union memberTypes="u"/></xs:simpleType><xs:simpleType name="u"><xs:union memberTypes="s"/></xs:simpleType>`, false},
		{"st-local-with-name", verBoth, `<xs:element name="e"><xs:simpleType name="s"><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`, false},
		{"st-global-without-name", verBoth, `<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`, false},
		{"ct-type-cycle", verBoth, `<xs:complexType name="a"><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType><xs:complexType name="b"><xs:complexContent><xs:extension base="a"/></xs:complexContent></xs:complexType>`, false},
		{"duplicate-global-type", verBoth, `<xs:simpleType name="s"><xs:restriction base="xs:string"/></xs:simpleType><xs:complexType name="s"/>`, false},
		{"duplicate-global-element", verBoth, `<xs:element name="e"/><xs:element name="e"/>`, false},

		// Substitution groups (e-props-correct.4, .6 / 1.1 cycles).
		{"subst-type-not-derived", verBoth, `<xs:element name="h" type="xs:int"/><xs:element name="m" substitutionGroup="h" type="xs:date"/>`, false},
		{"subst-type-derived", verBoth, `<xs:element name="h" type="xs:decimal"/><xs:element name="m" substitutionGroup="h" type="xs:int"/>`, true},
		{"subst-type-inherited", verBoth, `<xs:element name="h" type="xs:int"/><xs:element name="m" substitutionGroup="h"/>`, true},
		{"subst-head-final-extension", verBoth, `<xs:complexType name="b"/><xs:complexType name="d"><xs:complexContent><xs:extension base="b"><xs:sequence><xs:element name="z"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
		  <xs:element name="h" type="b" final="extension"/><xs:element name="m" substitutionGroup="h" type="d"/>`, false},
		{"subst-head-final-restriction-ok", verBoth, `<xs:complexType name="b"/><xs:complexType name="d"><xs:complexContent><xs:extension base="b"><xs:sequence><xs:element name="z"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
		  <xs:element name="h" type="b" final="restriction"/><xs:element name="m" substitutionGroup="h" type="d"/>`, true},
		{"subst-cycle", verBoth, `<xs:element name="a" substitutionGroup="b"/><xs:element name="b" substitutionGroup="a"/>`, false},
		{"subst-unresolved-head", verBoth + skipBug, `<xs:element name="a" substitutionGroup="nope"/>`, false},
		{"subst-multiple-heads-1.0", ver10 + skipBug, `<xs:element name="h1"/><xs:element name="h2"/><xs:element name="m" substitutionGroup="h1 h2"/>`, false},
		{"subst-multiple-heads-1.1", ver11, `<xs:element name="h1"/><xs:element name="h2"/><xs:element name="m" substitutionGroup="h1 h2"/>`, true},

		// Identity-constraint names share one symbol space per namespace.
		{"identity-duplicate-name", verBoth, `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
		  <xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="."/></xs:key><xs:unique name="k"><xs:selector xpath="a"/><xs:field xpath="."/></xs:unique></xs:element>`, false},
		{"identity-keyref-unresolved", verBoth, `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
		  <xs:keyref name="r" refer="nope"><xs:selector xpath="a"/><xs:field xpath="."/></xs:keyref></xs:element>`, false},
		{"identity-keyref-field-count", verBoth, `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
		  <xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="."/></xs:key><xs:keyref name="r" refer="k"><xs:selector xpath="a"/><xs:field xpath="."/><xs:field xpath="@x"/></xs:keyref></xs:element>`, false},
		{"identity-selector-attribute", verBoth, `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
		  <xs:key name="k"><xs:selector xpath="a/@x"/><xs:field xpath="."/></xs:key></xs:element>`, false},
		{"identity-field-bad-xpath", verBoth, `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
		  <xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="../b"/></xs:key></xs:element>`, false},
		{"identity-no-field", verBoth, `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
		  <xs:key name="k"><xs:selector xpath="a"/></xs:key></xs:element>`, false},

		// Notations.
		{"notation-ok", verBoth, `<xs:notation name="n" public="p"/><xs:simpleType name="s"><xs:restriction base="xs:NOTATION"><xs:enumeration value="n"/></xs:restriction></xs:simpleType>`, true},
		{"notation-without-ids", verBoth, `<xs:notation name="n"/>`, false},
		{"notation-direct-use", verBoth, `<xs:element name="e" type="xs:NOTATION"/>`, false},
		{"notation-enum-undeclared", verBoth, `<xs:simpleType name="s"><xs:restriction base="xs:NOTATION"><xs:enumeration value="nope"/></xs:restriction></xs:simpleType>`, false},
		{"notation-no-enumeration", verBoth, `<xs:simpleType name="s"><xs:restriction base="xs:NOTATION"/></xs:simpleType><xs:element name="e" type="s"/>`, false},
	}
	for _, tc := range cases {
		for _, v := range versionsOf(tc.ver % skipBug) {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if tc.ver >= skipBug {
					t.Skip("BUG: engine verdict disagrees with the spec")
				}
				if got := schemaVerdict(t, reprSchema(tc.body), v); got != tc.valid {
					t.Fatalf("schema valid=%v, want %v", got, tc.valid)
				}
			})
		}
	}
}
