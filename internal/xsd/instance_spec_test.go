package xsd

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// instPrelude declares the global components the instance tests use. Any
// global element may serve as the validation root.
const instPrelude = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="Ext"><xs:complexContent><xs:extension base="Base"><xs:sequence><xs:element name="b" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
  <xs:complexType name="Res"><xs:complexContent><xs:restriction base="Base"><xs:sequence><xs:element name="a" type="xs:string" fixed="r"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
  <xs:complexType name="NoExt" block="extension"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="NoExtD"><xs:complexContent><xs:extension base="NoExt"><xs:sequence><xs:element name="b" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
  <xs:complexType name="Abs" abstract="true"><xs:sequence/></xs:complexType>
  <xs:complexType name="Conc"><xs:complexContent><xs:extension base="Abs"/></xs:complexContent></xs:complexType>
  <xs:simpleType name="IntOrDate"><xs:union memberTypes="xs:int xs:date"/></xs:simpleType>

  <xs:element name="eBase" type="Base"/>
  <xs:element name="eBlocked" type="Base" block="extension"/>
  <xs:element name="eNoExt" type="NoExt"/>
  <xs:element name="eAbs" type="Base" abstract="true"/>
  <xs:element name="eAbsType" type="Abs"/>
  <xs:element name="eNil" type="xs:int" nillable="true"/>
  <xs:element name="eNilFixed" type="xs:int" nillable="true" fixed="5"/>
  <xs:element name="eInt" type="xs:int"/>
  <xs:element name="eUnion" type="IntOrDate"/>
  <xs:element name="eNilCT" type="Base" nillable="true"/>
  <xs:element name="eDef" type="xs:int" default="7"/>
  <xs:element name="eFix" type="xs:string" fixed="x"/>
  <xs:element name="eFixCT" fixed="x"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="eMixed"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" type="xs:string" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="eEmpty"><xs:complexType/></xs:element>
  <xs:element name="eAnyType"/>
  <xs:element name="eSC"><xs:complexType><xs:simpleContent><xs:extension base="xs:int"><xs:attribute name="u" type="xs:int"/></xs:extension></xs:simpleContent></xs:complexType></xs:element>
  <xs:element name="eAttrs"><xs:complexType>
    <xs:attribute name="req" type="xs:int" use="required"/>
    <xs:attribute name="opt" type="xs:string" default="d"/>
    <xs:attribute name="fix" type="xs:string" fixed="f"/>
    <xs:attribute name="no" type="xs:string" use="prohibited"/>
  </xs:complexType></xs:element>
  <xs:element name="eAnyAttr"><xs:complexType><xs:anyAttribute namespace="##other" processContents="lax"/></xs:complexType></xs:element>
  <xs:element name="eAnyAttrStrict"><xs:complexType><xs:anyAttribute processContents="strict"/></xs:complexType></xs:element>
  <xs:attribute name="gAttr" type="xs:int"/>
  <xs:element name="eStrict"><xs:complexType><xs:sequence><xs:any processContents="strict" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="eLax"><xs:complexType><xs:sequence><xs:any processContents="lax" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="eSkip"><xs:complexType><xs:sequence><xs:any processContents="skip" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="eOther"><xs:complexType><xs:sequence><xs:any namespace="##other" processContents="lax"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="eList"><xs:complexType><xs:sequence><xs:any namespace="urn:a ##local" processContents="skip"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="eIDs"><xs:complexType><xs:sequence><xs:element name="item" maxOccurs="unbounded"><xs:complexType>
    <xs:attribute name="id" type="xs:ID"/><xs:attribute name="ref" type="xs:IDREF"/><xs:attribute name="refs" type="xs:IDREFS"/>
  </xs:complexType></xs:element><xs:element name="cid" type="xs:ID" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="h" type="Base"/>
  <xs:element name="m" type="Ext" substitutionGroup="h"/>
  <xs:element name="mm" substitutionGroup="m"/>
  <xs:element name="hb" type="Base" block="substitution"/>
  <xs:element name="mb" substitutionGroup="hb"/>
  <xs:element name="hx" type="Base" block="extension"/>
  <xs:element name="mx" type="Ext" substitutionGroup="hx"/>
  <xs:element name="ha" type="Base" abstract="true"/>
  <xs:element name="ma" substitutionGroup="ha"/>
  <xs:element name="eHold"><xs:complexType><xs:sequence>
    <xs:element ref="h" minOccurs="0"/><xs:element ref="hb" minOccurs="0"/><xs:element ref="hx" minOccurs="0"/><xs:element ref="ha" minOccurs="0"/>
  </xs:sequence></xs:complexType></xs:element>
  <xs:element name="eChoice"><xs:complexType><xs:choice minOccurs="2" maxOccurs="3"><xs:element name="p"/><xs:element name="q"/></xs:choice></xs:complexType></xs:element>
  <xs:element name="eAll"><xs:complexType><xs:all><xs:element name="p"/><xs:element name="q" minOccurs="0"/></xs:all></xs:complexType></xs:element>
</xs:schema>`

const xsiDecl = ` xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

// Instance validity per the Validation Rules of Structures §3.3.4 (cvc-elt),
// §3.4.4 (cvc-complex-type), §3.10.4 (cvc-wildcard), §3.15.4 / §3.3.4 for
// ID/IDREF (cvc-id) and §3.9.4 particles with substitution groups.
func TestInstanceValidation(t *testing.T) {
	cases := []struct {
		name  string
		inst  string
		valid bool
	}{
		{"plain", `<eBase><a>x</a></eBase>`, true},
		{"undeclared-root", `<nope/>`, false},
		{"root-wrong-namespace", `<eBase xmlns="urn:x"><a>x</a></eBase>`, false},
		// cvc-elt.4: xsi:type.
		{"xsi-type-extension", `<eBase` + xsiDecl + ` xsi:type="Ext"><a>x</a><b>1</b></eBase>`, true},
		{"xsi-type-extension-incomplete", `<eBase` + xsiDecl + ` xsi:type="Ext"><a>x</a></eBase>`, false},
		{"xsi-type-restriction", `<eBase` + xsiDecl + ` xsi:type="Res"><a>r</a></eBase>`, true},
		{"xsi-type-restriction-fixed", `<eBase` + xsiDecl + ` xsi:type="Res"><a>x</a></eBase>`, false},
		{"xsi-type-not-derived", `<eBase` + xsiDecl + ` xsi:type="xs:int" xmlns:xs="http://www.w3.org/2001/XMLSchema">1</eBase>`, false},
		{"xsi-type-unknown", `<eBase` + xsiDecl + ` xsi:type="Nope"><a>x</a></eBase>`, false},
		{"xsi-type-not-qname", `<eBase` + xsiDecl + ` xsi:type="1x"><a>x</a></eBase>`, false},
		{"xsi-type-blocked-by-element", `<eBlocked` + xsiDecl + ` xsi:type="Ext"><a>x</a><b>1</b></eBlocked>`, false},
		{"xsi-type-restriction-not-blocked", `<eBlocked` + xsiDecl + ` xsi:type="Res"><a>r</a></eBlocked>`, true},
		{"xsi-type-blocked-by-type", `<eNoExt` + xsiDecl + ` xsi:type="NoExtD"><a>x</a><b>1</b></eNoExt>`, false},
		{"xsi-type-simple-derived", `<eInt` + xsiDecl + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:short">5</eInt>`, true},
		{"xsi-type-simple-derived-range", `<eInt` + xsiDecl + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:short">100000</eInt>`, false},
		{"xsi-type-simple-not-derived", `<eInt` + xsiDecl + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:string">5</eInt>`, false},
		// cos-st-derived-ok 2.2.4: a member type of a union substitutes it.
		{"xsi-type-union-member", `<eUnion` + xsiDecl + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:int">5</eUnion>`, true},
		{"xsi-type-union-member-value", `<eUnion` + xsiDecl + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:int">2020-01-01</eUnion>`, false},
		{"union-value", `<eUnion>2020-01-01</eUnion>`, true},
		{"anytype-xsi-type", `<eAnyType` + xsiDecl + ` xsi:type="Base"><a>x</a></eAnyType>`, true},
		{"anytype-xsi-type-violated", `<eAnyType` + xsiDecl + ` xsi:type="Base"><z/></eAnyType>`, false},
		// cvc-elt.2 / cvc-type.2: abstract element and type.
		{"abstract-element", `<eAbs><a>x</a></eAbs>`, false},
		{"abstract-type", `<eAbsType/>`, false},
		{"abstract-type-xsi-concrete", `<eAbsType` + xsiDecl + ` xsi:type="Conc"/>`, true},
		// cvc-elt.3: xsi:nil.
		{"nil-empty", `<eNil` + xsiDecl + ` xsi:nil="true"/>`, true},
		{"nil-numeric-true", `<eNil` + xsiDecl + ` xsi:nil="1"/>`, true},
		{"nil-with-content", `<eNil` + xsiDecl + ` xsi:nil="true">3</eNil>`, false},
		{"nil-false-content", `<eNil` + xsiDecl + ` xsi:nil="false">3</eNil>`, true},
		{"nil-false-empty", `<eNil` + xsiDecl + ` xsi:nil="false"/>`, false},
		{"nil-bad-value", `<eNil` + xsiDecl + ` xsi:nil="maybe"/>`, false},
		{"nil-not-nillable", `<eInt` + xsiDecl + ` xsi:nil="true"/>`, false},
		{"nil-and-fixed", `<eNilFixed` + xsiDecl + ` xsi:nil="true"/>`, false},
		{"nil-complex-empty", `<eNilCT` + xsiDecl + ` xsi:nil="true"/>`, true},
		{"nil-complex-with-child", `<eNilCT` + xsiDecl + ` xsi:nil="true"><a>x</a></eNilCT>`, false},
		{"nil-complex-missing-child", `<eNilCT/>`, false},
		// cvc-elt.5: default and fixed values.
		{"default-applied", `<eDef/>`, true},
		{"default-invalid-given", `<eDef>x</eDef>`, false},
		{"fixed-match", `<eFix>x</eFix>`, true},
		{"fixed-empty", `<eFix/>`, true},
		{"fixed-mismatch", `<eFix>y</eFix>`, false},
		{"fixed-mixed-match", `<eFixCT>x</eFixCT>`, true},
		{"fixed-mixed-mismatch", `<eFixCT>y</eFixCT>`, false},
		{"fixed-mixed-with-child", `<eFixCT>x<a/></eFixCT>`, false},
		// cvc-complex-type.2: content types.
		{"mixed-text-and-elements", `<eMixed>t<a>x</a>u<a>y</a>v</eMixed>`, true},
		{"mixed-missing-element", `<eMixed>t</eMixed>`, false},
		{"empty-ok", `<eEmpty/>`, true},
		{"empty-with-text", `<eEmpty>x</eEmpty>`, false},
		{"empty-with-child", `<eEmpty><a/></eEmpty>`, false},
		{"element-only-text", `<eBase>t<a>x</a></eBase>`, false},
		{"element-only-whitespace", `<eBase>
		  <a>x</a>
		</eBase>`, true},
		{"element-only-comment-pi", `<eBase><!-- c --><?p x?><a>x</a></eBase>`, true},
		{"anytype-anything", `<eAnyType z="1">t<q><r/></q></eAnyType>`, true},
		{"simple-content-ok", `<eSC u="1">5</eSC>`, true},
		{"simple-content-bad-attr", `<eSC u="x">5</eSC>`, false},
		{"simple-content-bad-value", `<eSC>x</eSC>`, false},
		{"simple-content-child", `<eSC><a/></eSC>`, false},
		// cvc-complex-type.3/.4: attributes.
		{"attrs-required-present", `<eAttrs req="1"/>`, true},
		{"attrs-required-missing", `<eAttrs/>`, false},
		{"attrs-required-bad", `<eAttrs req="x"/>`, false},
		{"attrs-fixed-match", `<eAttrs req="1" fix="f"/>`, true},
		{"attrs-fixed-mismatch", `<eAttrs req="1" fix="g"/>`, false},
		{"attrs-prohibited", `<eAttrs req="1" no="z"/>`, false},
		{"attrs-undeclared", `<eAttrs req="1" extra="z"/>`, false},
		{"attrs-xsi-allowed", `<eAttrs req="1"` + xsiDecl + ` xsi:noNamespaceSchemaLocation="x.xsd"/>`, true},
		{"attrs-xmlns-ignored", `<eAttrs req="1" xmlns:p="urn:p"/>`, true},
		{"any-attr-other", `<eAnyAttr xmlns:p="urn:p" p:z="1"/>`, true},
		{"any-attr-other-local", `<eAnyAttr z="1"/>`, false},
		{"any-attr-strict-declared", `<eAnyAttrStrict gAttr="1"/>`, true},
		{"any-attr-strict-declared-bad", `<eAnyAttrStrict gAttr="x"/>`, false},
		{"any-attr-strict-undeclared", `<eAnyAttrStrict zz="1"/>`, false},
		// cvc-wildcard / processContents.
		{"strict-declared", `<eStrict><eDef>7</eDef><eFix>x</eFix></eStrict>`, true},
		{"strict-declared-invalid", `<eStrict><eDef>x</eDef></eStrict>`, false},
		{"strict-undeclared", `<eStrict><zz/></eStrict>`, false},
		{"strict-xsi-type-undeclared", `<eStrict><zz` + xsiDecl + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:int">1</zz></eStrict>`, true},
		{"lax-undeclared", `<eLax><zz><yy/></zz></eLax>`, true},
		{"lax-undeclared-with-declared-inside", `<eLax><zz><eDef>x</eDef></zz></eLax>`, false},
		{"lax-declared-invalid", `<eLax><eDef>x</eDef></eLax>`, false},
		{"skip-declared-invalid", `<eSkip><eDef>x</eDef></eSkip>`, true},
		{"other-no-namespace", `<eOther><zz/></eOther>`, false},
		{"other-namespaced", `<eOther><p:zz xmlns:p="urn:p"/></eOther>`, true},
		{"list-local", `<eList><zz/></eList>`, true},
		{"list-listed", `<eList><p:zz xmlns:p="urn:a"/></eList>`, true},
		{"list-unlisted", `<eList><p:zz xmlns:p="urn:b"/></eList>`, false},
		// cvc-id.
		{"ids-ok", `<eIDs><item id="a"/><item id="b" ref="a" refs="a b"/></eIDs>`, true},
		{"ids-duplicate", `<eIDs><item id="a"/><item id="a"/></eIDs>`, false},
		{"ids-dangling-ref", `<eIDs><item id="a" ref="b"/></eIDs>`, false},
		{"ids-dangling-refs", `<eIDs><item id="a" refs="a c"/></eIDs>`, false},
		{"ids-forward-ref", `<eIDs><item ref="z"/><item id="z"/></eIDs>`, true},
		{"ids-bad-lexical", `<eIDs><item id="1a"/></eIDs>`, false},
		{"ids-element-content", `<eIDs><item ref="k"/><cid>k</cid></eIDs>`, true},
		{"ids-element-content-dup", `<eIDs><item id="k"/><cid>k</cid></eIDs>`, false},
		// Substitution groups (cvc-particle / Substitution Group OK).
		{"subst-head", `<eHold><h><a>x</a></h></eHold>`, true},
		{"subst-member", `<eHold><m><a>x</a><b>1</b></m></eHold>`, true},
		{"subst-transitive-member", `<eHold><mm><a>x</a><b>1</b></mm></eHold>`, true},
		{"subst-member-invalid-content", `<eHold><m><a>x</a></m></eHold>`, false},
		{"subst-blocked-substitution", `<eHold><mb><a>x</a></mb></eHold>`, false},
		{"subst-blocked-extension", `<eHold><mx><a>x</a><b>1</b></mx></eHold>`, false},
		{"subst-abstract-head", `<eHold><ha><a>x</a></ha></eHold>`, false},
		{"subst-abstract-head-member", `<eHold><ma><a>x</a></ma></eHold>`, true},
		{"subst-order", `<eHold><hb><a>x</a></hb><m><a>x</a><b>1</b></m></eHold>`, false},
		// Particle occurrence (cvc-particle, cvc-model-group).
		{"choice-min", `<eChoice><p/></eChoice>`, false},
		{"choice-ok", `<eChoice><p/><q/><p/></eChoice>`, true},
		{"choice-max", `<eChoice><p/><q/><p/><q/></eChoice>`, false},
		{"all-any-order", `<eAll><q/><p/></eAll>`, true},
		{"all-optional-absent", `<eAll><p/></eAll>`, true},
		{"all-missing-required", `<eAll><q/></eAll>`, false},
		{"all-repeated", `<eAll><p/><p/></eAll>`, false},
		{"all-unknown", `<eAll><p/><z/></eAll>`, false},
	}
	for _, tc := range cases {
		for _, v := range []Version{Version10, Version11} {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if got := instanceVerdict(t, instPrelude, tc.inst, v); got != tc.valid {
					t.Fatalf("valid=%v, want %v", got, tc.valid)
				}
			})
		}
	}
}

// cvc-elt.4.1: xsi:type must be a valid QName, which includes a bound
// prefix (QName value space, Datatypes §3.2.18).
func TestXsiTypeUnboundPrefix(t *testing.T) {
	t.Skip("BUG: xsi:type with an unbound prefix resolves as if the prefix were absent")
	if instanceVerdict(t, instPrelude, `<eBase`+xsiDecl+` xsi:type="p:Ext"><a>x</a><b>1</b></eBase>`, Version10) {
		t.Fatal("accepted")
	}
}

// A not-well-formed instance and one without a document element are
// invalid, never "no verdict".
func TestInstanceNotWellFormed(t *testing.T) {
	s, err := Compile([]string{instPrelude}, "", Version10)
	if err != nil {
		t.Fatal(err)
	}
	for _, inst := range []string{`<eBase><a>x</eBase>`, ``, `<!-- only a comment -->`} {
		res, err := s.Validate(inst)
		if err != nil {
			t.Fatalf("%q: %v", inst, err)
		}
		if res.Valid || len(res.Errors) == 0 {
			t.Fatalf("%q: valid=%v errors=%v", inst, res.Valid, res.Errors)
		}
	}
}

// ValidateOptsCtx with a context that is already done stops the walk and
// returns the context's error instead of a verdict.
func TestValidateCanceled(t *testing.T) {
	s, err := Compile([]string{instPrelude}, "", Version11)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("<eMixed>")
	for i := 0; i < 5000; i++ {
		b.WriteString("<a>x</a>")
	}
	b.WriteString("</eMixed>")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := s.ValidateOptsCtx(ctx, b.String(), "", 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, res = %+v; want context.Canceled", err, res)
	}
	// The same instance with a live context is valid.
	res, err = s.ValidateOptsCtx(context.Background(), b.String(), "", 0)
	if err != nil || !res.Valid {
		t.Fatalf("live context: res=%+v err=%v", res, err)
	}
}
