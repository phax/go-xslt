package xsd

import "testing"

// Identity constraints (Structures §3.11): the restricted XPath subset of
// selectors and fields (§3.11.6 c-selector-xpath / c-fields-xpaths), and
// cvc-identity-constraint (field tuples, value equality, key presence,
// keyref resolution).
func TestIdentityConstraintSemantics(t *testing.T) {
	// lib holds book children; ic is inserted into lib's declaration.
	lib := func(ic string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		  <xs:element name="lib"><xs:complexType><xs:sequence>
		    <xs:element name="book" minOccurs="0" maxOccurs="unbounded"><xs:complexType><xs:sequence>
		      <xs:element name="title" type="xs:string" minOccurs="0"/>
		      <xs:element name="author" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
		      <xs:element name="meta" minOccurs="0"><xs:complexType><xs:sequence><xs:element name="m" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>
		    </xs:sequence>
		    <xs:attribute name="isbn" type="xs:string"/><xs:attribute name="n" type="xs:decimal"/>
		    <xs:attribute name="s" type="xs:string"/><xs:attribute name="a" type="xs:string"/><xs:attribute name="b" type="xs:string"/>
		    </xs:complexType></xs:element>
		    <xs:element name="ref" minOccurs="0" maxOccurs="unbounded"><xs:complexType>
		      <xs:attribute name="to" type="xs:string"/><xs:attribute name="num" type="xs:decimal"/><xs:attribute name="int" type="xs:integer"/>
		    </xs:complexType></xs:element>
		  </xs:sequence></xs:complexType>` + ic + `</xs:element>
		</xs:schema>`
	}
	const keyISBN = `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="@isbn"/></xs:key>`
	cases := []struct {
		name  string
		ic    string
		inst  string
		valid bool
	}{
		{"key-ok", keyISBN, `<lib><book isbn="1"/><book isbn="2"/></lib>`, true},
		{"key-duplicate", keyISBN, `<lib><book isbn="1"/><book isbn="1"/></lib>`, false},
		{"key-missing-field", keyISBN, `<lib><book isbn="1"/><book/></lib>`, false},
		{"unique-missing-field", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="@isbn"/></xs:unique>`, `<lib><book isbn="1"/><book/><book/></lib>`, true},
		{"unique-duplicate", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="@isbn"/></xs:unique>`, `<lib><book isbn="1"/><book isbn="1"/></lib>`, false},
		{"key-element-field", `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="title"/></xs:key>`, `<lib><book><title>A</title></book><book><title>B</title></book></lib>`, true},
		{"key-element-field-dup", `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="title"/></xs:key>`, `<lib><book><title>A</title></book><book><title>A</title></book></lib>`, false},
		// cvc-identity-constraint.3: a field selecting more than one node.
		{"field-multiple-nodes", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="author"/></xs:unique>`, `<lib><book><author>x</author><author>y</author></book></lib>`, false},
		// A field must select an element of simple type / simple content.
		{"field-complex-element", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="meta"/></xs:unique>`, `<lib><book><meta><m/></meta></book></lib>`, false},
		// Tuples: duplicates only when every field is equal.
		{"multi-field-distinct", `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="@a"/><xs:field xpath="@b"/></xs:key>`, `<lib><book a="1" b="1"/><book a="1" b="2"/></lib>`, true},
		{"multi-field-dup", `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="@a"/><xs:field xpath="@b"/></xs:key>`, `<lib><book a="1" b="2"/><book a="1" b="2"/></lib>`, false},
		// Value equality is in the value space of the field's type.
		{"decimal-value-equal", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="@n"/></xs:unique>`, `<lib><book n="1.0"/><book n="01"/></lib>`, false},
		{"decimal-value-distinct", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="@n"/></xs:unique>`, `<lib><book n="1.0"/><book n="1.5"/></lib>`, true},
		{"string-lexical-distinct", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="@s"/></xs:unique>`, `<lib><book s="1.0"/><book s="1"/></lib>`, true},
		// Selector forms of the subset.
		{"selector-union", `<xs:unique name="u"><xs:selector xpath="book | ref"/><xs:field xpath="@isbn"/></xs:unique>`, `<lib><book isbn="1"/><book isbn="2"/></lib>`, true},
		{"selector-descendant", `<xs:unique name="u"><xs:selector xpath=".//author"/><xs:field xpath="."/></xs:unique>`, `<lib><book><author>x</author></book><book><author>x</author></book></lib>`, false},
		{"selector-child-axis", `<xs:unique name="u"><xs:selector xpath="child::book/child::title"/><xs:field xpath="."/></xs:unique>`, `<lib><book><title>x</title></book><book><title>x</title></book></lib>`, false},
		{"selector-wildcard", `<xs:unique name="u"><xs:selector xpath="*"/><xs:field xpath="@isbn"/></xs:unique>`, `<lib><book isbn="1"/><book isbn="1"/></lib>`, false},
		{"field-attribute-axis", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="attribute::isbn"/></xs:unique>`, `<lib><book isbn="1"/><book isbn="1"/></lib>`, false},
		{"field-descendant", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath=".//m"/></xs:unique>`, `<lib><book><meta><m>1</m></meta></book><book><meta><m>1</m></meta></book></lib>`, false},
		// keyref (cvc-identity-constraint.4.3).
		{"keyref-ok", keyISBN + `<xs:keyref name="r" refer="k"><xs:selector xpath="ref"/><xs:field xpath="@to"/></xs:keyref>`, `<lib><book isbn="1"/><ref to="1"/><ref/></lib>`, true},
		{"keyref-dangling", keyISBN + `<xs:keyref name="r" refer="k"><xs:selector xpath="ref"/><xs:field xpath="@to"/></xs:keyref>`, `<lib><book isbn="1"/><ref to="2"/></lib>`, false},
		{"keyref-to-unique", `<xs:unique name="u"><xs:selector xpath="book"/><xs:field xpath="@isbn"/></xs:unique><xs:keyref name="r" refer="u"><xs:selector xpath="ref"/><xs:field xpath="@to"/></xs:keyref>`, `<lib><book isbn="1"/><ref to="1"/></lib>`, true},
		{"keyref-value-space", `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="@n"/></xs:key><xs:keyref name="r" refer="k"><xs:selector xpath="ref"/><xs:field xpath="@num"/></xs:keyref>`, `<lib><book n="2"/><ref num="2.00"/></lib>`, true},
		// decimal and integer share the decimal value space.
		{"keyref-integer-vs-decimal", `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="@n"/></xs:key><xs:keyref name="r" refer="k"><xs:selector xpath="ref"/><xs:field xpath="@int"/></xs:keyref>`, `<lib><book n="3.0"/><ref int="3"/></lib>`, true},
		// Values of unrelated primitive types are never equal.
		{"keyref-string-vs-decimal", `<xs:key name="k"><xs:selector xpath="book"/><xs:field xpath="@n"/></xs:key><xs:keyref name="r" refer="k"><xs:selector xpath="ref"/><xs:field xpath="@to"/></xs:keyref>`, `<lib><book n="3"/><ref to="3"/></lib>`, false},
	}
	for _, tc := range cases {
		for _, v := range []Version{Version10, Version11} {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if got := instanceVerdict(t, lib(tc.ic), tc.inst, v); got != tc.valid {
					t.Fatalf("valid=%v, want %v", got, tc.valid)
				}
			})
		}
	}
}

// Namespaced selectors and fields: prefixes resolve against the schema
// document's in-scope namespaces; 1.1 xpathDefaultNamespace sets the
// namespace of unprefixed names (§3.11.2, §3.11.6).
func TestIdentityConstraintNamespaces(t *testing.T) {
	schema := func(ic, schemaAttrs string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified" attributeFormDefault="qualified"` + schemaAttrs + `>
		  <xs:element name="root"><xs:complexType><xs:sequence>
		    <xs:element name="it" maxOccurs="unbounded"><xs:complexType><xs:attribute name="id" type="xs:string"/></xs:complexType></xs:element>
		  </xs:sequence></xs:complexType>` + ic + `</xs:element></xs:schema>`
	}
	dup := `<t:root xmlns:t="urn:t"><t:it t:id="1"/><t:it t:id="1"/></t:root>`
	cases := []struct {
		name        string
		ver         int
		ic          string
		schemaAttrs string
		valid       bool // verdict on dup
	}{
		{"prefixed", verBoth, `<xs:unique name="u"><xs:selector xpath="t:it"/><xs:field xpath="@t:id"/></xs:unique>`, "", false},
		// Unprefixed names are in no namespace: the selector selects
		// nothing, so no duplicates are found.
		{"unprefixed-no-namespace", verBoth, `<xs:unique name="u"><xs:selector xpath="it"/><xs:field xpath="@t:id"/></xs:unique>`, "", true},
		{"ns-wildcard", verBoth, `<xs:unique name="u"><xs:selector xpath="t:*"/><xs:field xpath="@t:id"/></xs:unique>`, "", false},
		{"xpath-default-ns-selector", ver11, `<xs:unique name="u"><xs:selector xpath="it" xpathDefaultNamespace="##targetNamespace"/><xs:field xpath="@t:id"/></xs:unique>`, "", false},
		{"xpath-default-ns-schema", ver11, `<xs:unique name="u"><xs:selector xpath="it"/><xs:field xpath="@t:id"/></xs:unique>`, ` xpathDefaultNamespace="urn:t"`, false},
		{"xpath-default-ns-local", ver11, `<xs:unique name="u"><xs:selector xpath="it"/><xs:field xpath="@t:id"/></xs:unique>`, ` xpathDefaultNamespace="##local"`, true},
	}
	for _, tc := range cases {
		for _, v := range versionsOf(tc.ver) {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if got := instanceVerdict(t, schema(tc.ic, tc.schemaAttrs), dup, v); got != tc.valid {
					t.Fatalf("valid=%v, want %v", got, tc.valid)
				}
			})
		}
	}
}

// The selector/field XPath subset (§3.11.6): forms outside it make the
// schema invalid.
func TestIdentityXPathSubset(t *testing.T) {
	schema := func(sel, field string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
		  <xs:element name="r"><xs:complexType><xs:sequence><xs:any processContents="skip" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
		  <xs:unique name="u"><xs:selector xpath="` + sel + `"/><xs:field xpath="` + field + `"/></xs:unique></xs:element></xs:schema>`
	}
	cases := []struct {
		sel, field string
		valid      bool
	}{
		{"a", ".", true},
		{"a/b", "@x", true},
		{".//a", "c/@x", true},
		{"a | .//b", "@x | c", true},
		{"p:a/p:*", "@p:x", true},
		{"child::a", "attribute::x", true},
		{" a ", " @x ", true},
		{"*", "@*", true},
		{"/a", ".", false},
		{"//a", ".", false},
		{"a[1]", ".", false},
		{"a/..", ".", false},
		{"@x", ".", false},
		{"a/@x", ".", false},
		{"a", "@x/b", false},
		{"a", "text()", false},
		{"a", "q:x", false},
		{"a", "", false},
		{"descendant::a", ".", false},
		{"a", "./@x/.", false},
		{"a|", ".", false},
	}
	for _, tc := range cases {
		for _, v := range []Version{Version10, Version11} {
			t.Run(tc.sel+" ; "+tc.field+"/"+v.String(), func(t *testing.T) {
				if got := schemaVerdict(t, schema(tc.sel, tc.field), v); got != tc.valid {
					t.Fatalf("schema valid=%v, want %v", got, tc.valid)
				}
			})
		}
	}
}
