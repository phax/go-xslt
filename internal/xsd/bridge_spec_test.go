package xsd

import (
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
	"github.com/tim-riep/go-xslt/internal/xpath"
)

// ValidateNode (validate-in-place on an existing tree) must reach the same
// verdict as Validate on the same document, and both must agree with the
// spec. Runs the instance table of the composite prelude through both paths
// as whole documents (Document: true, so ID/IDREF rules apply).
func TestValidateNodeAgreesWithValidate(t *testing.T) {
	cases := []struct {
		inst  string
		valid bool
	}{
		{`<eBase><a>x</a></eBase>`, true},
		{`<eBase><b>x</b></eBase>`, false},
		{`<eBase` + xsiDecl + ` xsi:type="Ext"><a>x</a><b>1</b></eBase>`, true},
		{`<eBlocked` + xsiDecl + ` xsi:type="Ext"><a>x</a><b>1</b></eBlocked>`, false},
		{`<eAbsType` + xsiDecl + ` xsi:type="Conc"/>`, true},
		{`<eAbs><a>x</a></eAbs>`, false},
		{`<eNil` + xsiDecl + ` xsi:nil="true"/>`, true},
		{`<eNil` + xsiDecl + ` xsi:nil="true">3</eNil>`, false},
		{`<eDef/>`, true},
		{`<eFix>y</eFix>`, false},
		{`<eMixed>t<a>x</a></eMixed>`, true},
		{`<eEmpty>x</eEmpty>`, false},
		{`<eSC u="1">5</eSC>`, true},
		{`<eAttrs req="1"/>`, true},
		{`<eAttrs/>`, false},
		{`<eStrict><eDef>7</eDef></eStrict>`, true},
		{`<eStrict><zz/></eStrict>`, false},
		{`<eLax><zz/></eLax>`, true},
		{`<eIDs><item id="a"/><item ref="a"/></eIDs>`, true},
		{`<eIDs><item id="a"/><item id="a"/></eIDs>`, false},
		{`<eIDs><item ref="b"/></eIDs>`, false},
		{`<eHold><m><a>x</a><b>1</b></m></eHold>`, true},
		{`<eHold><mb><a>x</a></mb></eHold>`, false},
		{`<eChoice><p/><q/></eChoice>`, true},
		{`<eAll><q/></eAll>`, false},
		{`<eUnion>2020-01-01</eUnion>`, true},
	}
	for _, v := range []Version{Version10, Version11} {
		s, err := Compile([]string{instPrelude}, "", v)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range cases {
			res, err := s.Validate(tc.inst)
			if err != nil {
				t.Fatalf("%v %s: %v", v, tc.inst, err)
			}
			doc, err := xmltree.ParseLenient11(tc.inst)
			if err != nil {
				t.Fatal(err)
			}
			nodeErr := s.ValidateNode(doc, NodeValidateOptions{Strict: true, Document: true})
			if res.Valid != tc.valid {
				t.Errorf("%v %s: Validate valid=%v, want %v", v, tc.inst, res.Valid, tc.valid)
			}
			if (nodeErr == nil) != tc.valid {
				t.Errorf("%v %s: ValidateNode err=%v, want valid=%v", v, tc.inst, nodeErr, tc.valid)
			}
		}
	}
}

// After a successful ValidateNode the tree carries the PSVI-derived
// properties: defaulted attributes are supplied (cvc-complex-type.4 /
// schema-default), nilled is recorded (§3.3.5.1 [nil]), and IDs are marked.
func TestValidateNodePSVIProperties(t *testing.T) {
	s, err := Compile([]string{instPrelude}, "", Version11)
	if err != nil {
		t.Fatal(err)
	}
	el := bridgeParse(t, `<eAttrs req="1"/>`)
	if err := s.ValidateNode(el, NodeValidateOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	if v, ok := el.Attr("", "opt"); !ok || v != "d" {
		t.Errorf("default attribute opt = %q, %v; want \"d\"", v, ok)
	}
	if v, ok := el.Attr("", "fix"); !ok || v != "f" {
		t.Errorf("fixed attribute fix = %q, %v; want \"f\"", v, ok)
	}
	for _, a := range el.Attrs {
		if a.Name.Local == "req" && xpath.AtomType(a.TypeAnno()) != xpath.XSint {
			t.Errorf("@req TypeAnno = %d, want xs:int", a.TypeAnno())
		}
	}

	nilEl := bridgeParse(t, `<eNil`+xsiDecl+` xsi:nil="true"/>`)
	if err := s.ValidateNode(nilEl, NodeValidateOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	if !nilEl.Nilled() {
		t.Error("xsi:nil=true element is not marked nilled")
	}

	ids := bridgeParse(t, `<eIDs><item id="a" refs="a a"/></eIDs>`)
	if err := s.ValidateNode(ids, NodeValidateOptions{Strict: true, Document: true}); err != nil {
		t.Fatal(err)
	}
	item := ids.Children[0]
	for _, a := range item.Attrs {
		switch a.Name.Local {
		case "id":
			if a.IDKind() != xmltree.IDKindID {
				t.Errorf("@id IDKind = %d, want ID", a.IDKind())
			}
		case "refs":
			if a.IDKind() != xmltree.IDKindIDREFS || !a.ListTyped() {
				t.Errorf("@refs IDKind=%d ListTyped=%v, want IDREFS list", a.IDKind(), a.ListTyped())
			}
		}
	}
}

// ID/IDREF are document-level: an element-rooted episode does not resolve
// IDREFs to IDs outside it, a document-rooted one does (cvc-id.1).
func TestValidateNodeIDScope(t *testing.T) {
	s, err := Compile([]string{instPrelude}, "", Version10)
	if err != nil {
		t.Fatal(err)
	}
	inst := `<eIDs><item ref="nowhere"/></eIDs>`
	if err := s.ValidateNode(bridgeParse(t, inst), NodeValidateOptions{Strict: true}); err != nil {
		t.Errorf("element episode: %v", err)
	}
	if err := s.ValidateNode(bridgeParse(t, inst), NodeValidateOptions{Strict: true, Document: true}); err == nil {
		t.Error("document episode accepted a dangling IDREF")
	}
}

// StripElementOnlyWhitespace drops whitespace text from element-only
// content, and only there.
func TestValidateNodeStripsElementOnlyWhitespace(t *testing.T) {
	s, err := Compile([]string{instPrelude}, "", Version11)
	if err != nil {
		t.Fatal(err)
	}
	el := bridgeParse(t, "<eBase>\n  <a> x </a>\n</eBase>")
	if err := s.ValidateNode(el, NodeValidateOptions{Strict: true, StripElementOnlyWhitespace: true}); err != nil {
		t.Fatal(err)
	}
	if len(el.Children) != 1 || el.Children[0].Kind != xmltree.KindElement {
		t.Fatalf("children after stripping: %d", len(el.Children))
	}
	// xs:string content keeps its whitespace.
	if got := el.Children[0].Children[0].Value; got != " x " {
		t.Errorf("simple content text = %q, want \" x \"", got)
	}
	mixed := bridgeParse(t, "<eMixed>\n  <a>x</a>\n</eMixed>")
	if err := s.ValidateNode(mixed, NodeValidateOptions{Strict: true, StripElementOnlyWhitespace: true}); err != nil {
		t.Fatal(err)
	}
	if len(mixed.Children) != 3 {
		t.Errorf("mixed content lost text nodes: %d children", len(mixed.Children))
	}
}

// Validating against a named type, an attribute node, and strict vs lax
// with no matching declaration.
func TestValidateNodeTargets(t *testing.T) {
	s, err := Compile([]string{instPrelude}, "", Version11)
	if err != nil {
		t.Fatal(err)
	}
	ext := &xmltree.SchemaTypeName{Local: "Ext", Complex: true}
	if err := s.ValidateNode(bridgeParse(t, `<anything><a>x</a><b>1</b></anything>`), NodeValidateOptions{Type: ext}); err != nil {
		t.Errorf("named type Ext: %v", err)
	}
	if err := s.ValidateNode(bridgeParse(t, `<anything><a>x</a></anything>`), NodeValidateOptions{Type: ext}); err == nil {
		t.Error("named type Ext accepted missing b")
	}
	intT := &xmltree.SchemaTypeName{Namespace: xsNS, Local: "int"}
	if err := s.ValidateNode(bridgeParse(t, `<n> 42 </n>`), NodeValidateOptions{Type: intT}); err != nil {
		t.Errorf("named type xs:int: %v", err)
	}
	if err := s.ValidateNode(bridgeParse(t, `<n>4.2</n>`), NodeValidateOptions{Type: intT}); err == nil {
		t.Error("xs:int accepted 4.2")
	}
	if err := s.ValidateNode(bridgeParse(t, `<n>1</n>`), NodeValidateOptions{Type: &xmltree.SchemaTypeName{Local: "NoSuchType"}}); err == nil {
		t.Error("an unknown named type validated")
	}
	// Strict: an undeclared element fails; lax: it is not assessed at all.
	if err := s.ValidateNode(bridgeParse(t, `<undeclared/>`), NodeValidateOptions{Strict: true}); err == nil {
		t.Error("strict accepted an undeclared element")
	}
	if err := s.ValidateNode(bridgeParse(t, `<undeclared/>`), NodeValidateOptions{}); err != nil {
		t.Errorf("lax rejected an undeclared element: %v", err)
	}
	// A global attribute declaration validates an attribute node.
	host := bridgeParse(t, `<x gAttr="12"/>`)
	if err := s.ValidateNode(host.Attrs[0], NodeValidateOptions{Strict: true}); err != nil {
		t.Errorf("attribute gAttr=12: %v", err)
	}
	bad := bridgeParse(t, `<x gAttr="zz"/>`)
	if err := s.ValidateNode(bad.Attrs[0], NodeValidateOptions{Strict: true}); err == nil {
		t.Error("attribute gAttr=zz accepted")
	}
	if err := s.ValidateNode(nil, NodeValidateOptions{}); err == nil {
		t.Error("a nil node validated")
	}
}

// The XPath-facing casting seams: casting a lexical form to a named simple
// type (XPath 3.1 §3.14 over imported types), union member selection, and
// list types (§3.14.2: casting to a list type yields a sequence of items).
func TestSchemaTypeCasting(t *testing.T) {
	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:c" xmlns="urn:c">
	  <xs:simpleType name="Small"><xs:restriction base="xs:int"><xs:maxInclusive value="9"/></xs:restriction></xs:simpleType>
	  <xs:simpleType name="U"><xs:union memberTypes="xs:int xs:date"/></xs:simpleType>
	  <xs:simpleType name="UN"><xs:union><xs:simpleType><xs:restriction base="xs:string"><xs:enumeration value="none"/></xs:restriction></xs:simpleType></xs:union></xs:simpleType>
	  <xs:simpleType name="L"><xs:list itemType="xs:int"/></xs:simpleType>
	  <xs:complexType name="C"/>
	  <xs:element name="h" type="xs:anyType"/>
	  <xs:element name="m" substitutionGroup="h"/>
	  <xs:element name="b" substitutionGroup="h" abstract="true"/>
	</xs:schema>`
	s, err := Compile([]string{schema}, "", Version11)
	if err != nil {
		t.Fatal(err)
	}
	n := func(local string) xmltree.SchemaTypeName {
		return xmltree.SchemaTypeName{Namespace: "urn:c", Local: local}
	}
	if prim, err := s.CastToSchemaType(n("Small"), "7"); err != nil || prim != xpath.XSdecimal && prim != xpath.XSint && prim != xpath.XSinteger {
		t.Errorf("Small 7: %v %v", prim, err)
	}
	if _, err := s.CastToSchemaType(n("Small"), "10"); err == nil {
		t.Error("Small accepted 10")
	}
	if prim, err := s.CastToSchemaType(n("U"), "2020-01-01"); err != nil || prim != xpath.XSdate {
		t.Errorf("U date: %v %v", prim, err)
	}
	if _, err := s.CastToSchemaType(n("U"), "x"); err == nil {
		t.Error("U accepted x")
	}
	if _, err := s.CastToSchemaType(n("C"), "x"); err == nil {
		t.Error("cast to a complex type succeeded")
	}
	if _, err := s.CastToSchemaType(n("Nope"), "x"); err == nil {
		t.Error("cast to an unknown type succeeded")
	}
	if m, builtin, ok := s.UnionCastMember(n("U"), "5", nil); !ok || !builtin || m.Local != "" {
		t.Errorf("UnionCastMember(U, 5) = %+v builtin=%v ok=%v; want the built-in xs:int member", m, builtin, ok)
	}
	if _, _, ok := s.UnionCastMember(n("Small"), "5", nil); ok {
		t.Error("UnionCastMember answered for a non-union")
	}
	if _, _, ok := s.UnionCastMember(n("U"), "zz", nil); ok {
		t.Error("UnionCastMember found a member for an invalid value")
	}
	// A list type: the item type's primitive, and every item validated.
	if prim, _, isList, err := s.CastToSchemaListType(n("L"), " 1 2  3 ", nil); err != nil || !isList || prim != xpath.XSint {
		t.Errorf("list cast = %v, %v, %v; want xs:int items", prim, isList, err)
	}
	if _, _, isList, err := s.CastToSchemaListType(n("L"), "1 x", nil); !isList || err == nil {
		t.Error("list cast accepted a bad item")
	}
	// Not a list: the caller falls back to the atomic path.
	if _, _, isList, err := s.CastToSchemaListType(n("Small"), "1", nil); isList || err != nil {
		t.Errorf("atomic type reported as list: %v %v", isList, err)
	}
	if _, err := s.CastToSchemaType(n("L"), "1 2"); err == nil {
		t.Error("CastToSchemaType accepted a list type (its value is a sequence)")
	}
	// Substitution: a member substitutes for its head; an abstract member
	// is still a member of the group (only abstract elements cannot appear).
	if !s.SubstitutesFor(xmltree.Name{Space: "urn:c", Local: "m"}, xmltree.Name{Space: "urn:c", Local: "h"}) {
		t.Error("m does not substitute for h")
	}
	if s.SubstitutesFor(xmltree.Name{Space: "urn:c", Local: "h"}, xmltree.Name{Space: "urn:c", Local: "m"}) {
		t.Error("h substitutes for its own member")
	}
	if s.SubstitutesFor(xmltree.Name{Space: "urn:c", Local: "m"}, xmltree.Name{Space: "urn:c", Local: "nope"}) {
		t.Error("substitution for an undeclared head")
	}
}
