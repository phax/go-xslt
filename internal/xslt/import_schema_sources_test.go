package xslt

import (
	"path/filepath"
	"strings"
	"testing"
)

// xsl:import-schema (XSLT 3.0 §3.15): where the schema document comes from
// (inline, @schema-location, the host) and the static errors of the
// declaration itself.

func svDImportSheet(decl, body string) string {
	return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" exclude-result-prefixes="#all">` + decl + `
  <xsl:output method="text"/>
  <xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
}

const svDInlineT = `<xs:schema targetNamespace="urn:t"><xs:simpleType name="Small"><xs:restriction base="xs:integer"><xs:maxInclusive value="10"/></xs:restriction></xs:simpleType></xs:schema>`

func TestImportSchemaDeclarationErrors(t *testing.T) {
	cases := []struct{ name, decl, code string }{
		// "The zero-length string is not a valid namespace URI" (§3.15).
		{"empty namespace", `<xsl:import-schema namespace="" schema-location="rec.xsd"/>`, "XTSE0020"},
		// The content model is xs:schema? — XTSE0010.
		{"two inline schemas", `<xsl:import-schema>` + svDInlineT + svDInlineT + `</xsl:import-schema>`, "XTSE0010"},
		{"non-schema child", `<xsl:import-schema><t:foo/></xsl:import-schema>`, "XTSE0010"},
		// [ERR XTSE0215]: inline schema with schema-location, or with a
		// conflicting namespace attribute.
		{"inline with schema-location", `<xsl:import-schema schema-location="rec.xsd">` + svDInlineT + `</xsl:import-schema>`, "XTSE0215"},
		{"inline namespace conflict", `<xsl:import-schema namespace="urn:other">` + svDInlineT + `</xsl:import-schema>`, "XTSE0215"},
		{"inline without targetNamespace but namespace given", `<xsl:import-schema namespace="urn:t"><xs:schema/></xsl:import-schema>`, "XTSE0215"},
		// [ERR XTSE0220]: the synthetic schema document violates XSD's own
		// constraints — here, two global element declarations of one name, and
		// a located document whose target namespace is not the one imported
		// (src-import, XSD 1.0 Part 1 §4.2.3).
		{"duplicate declarations", `<xsl:import-schema><xs:schema targetNamespace="urn:t"><xs:element name="a"/><xs:element name="a"/></xs:schema></xsl:import-schema>`, "XTSE0220"},
		{"located document in another namespace", `<xsl:import-schema namespace="urn:t" schema-location="other-ns.xsd"/>`, "XTSE0220"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svDRunSA(t, svDImportSheet(tc.decl, `<xsl:value-of select="1"/>`), "<x/>")
			svDWantCode(t, err, tc.code)
		})
	}
}

// TestImportSchemaComponentsUsable checks that imported simple types are
// usable: as a constructor function, in cast/castable, and in instance of.
func TestImportSchemaComponentsUsable(t *testing.T) {
	for _, decl := range []string{
		`<xsl:import-schema>` + svDInlineT + `</xsl:import-schema>`,
		`<xsl:import-schema namespace="urn:t">` + svDInlineT + `</xsl:import-schema>`,
		`<xsl:import-schema namespace="urn:t" schema-location="rec.xsd"/>`,
	} {
		got, err := svDRunSA(t, svDImportSheet(decl,
			`<xsl:value-of select="t:Small('7') instance of t:Small, '11' castable as t:Small, ('3' cast as t:Small) + 1, t:Small('7') instance of xs:integer"/>`), "<x/>")
		if err != nil {
			t.Fatalf("%s: %v", decl, err)
		}
		if got != "true false 4 true" {
			t.Errorf("%s: got %q, want %q", decl, got, "true false 4 true")
		}
	}
}

// TestImportSchemaInlinePrefixFromAncestor: an inline schema whose QName
// attribute values use a prefix bound only on xsl:stylesheet still resolves
// (the in-scope namespaces of xs:schema include the ancestor's).
func TestImportSchemaInlinePrefixFromAncestor(t *testing.T) {
	sheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="urn:my">
  <xsl:import-schema><xs:schema targetNamespace="urn:my">
    <xs:simpleType name="S"><xs:restriction base="xs:string"><xs:length value="2"/></xs:restriction></xs:simpleType>
    <xs:element name="e" type="my:S"/>
  </xs:schema></xsl:import-schema>
  <xsl:output method="text"/>
  <xsl:template match="/"><xsl:variable name="v" as="element()"><xsl:element name="my:e" validation="strict">ab</xsl:element></xsl:variable><xsl:value-of select="$v instance of element(my:e, my:S)"/></xsl:template>
</xsl:stylesheet>`
	got, err := svDRunSA(t, sheet, "<x/>")
	if err != nil || got != "true" {
		t.Fatalf("got %q, %v; want \"true\"", got, err)
	}
}

// TestImportSchemaXMLNamespace: importing the XML namespace without a
// location supplies the declarations of xml:lang, xml:space, xml:base and
// xml:id, so a strict validation of xml:space checks its enumeration.
func TestImportSchemaXMLNamespace(t *testing.T) {
	decl := `<xsl:import-schema namespace="http://www.w3.org/XML/1998/namespace"/>`
	body := func(v string) string {
		return `<xsl:variable name="e" as="element()"><e xsl:validation="preserve"><xsl:attribute name="xml:space" validation="strict">` + v + `</xsl:attribute></e></xsl:variable><xsl:value-of select="string($e/@xml:space)"/>`
	}
	got, err := svDRunSA(t, svDImportSheet(decl, body("preserve")), "<x/>")
	if err != nil || got != "preserve" {
		t.Fatalf("got %q, %v; want \"preserve\"", got, err)
	}
	_, err = svDRunSA(t, svDImportSheet(decl, body("foo-bar")), "<x/>")
	svDWantCode(t, err, "XTTE1510")
}

// TestImportSchemaHostSuppliedDocument: @schema-location is only a hint
// (§3.15); a schema the host supplies for the namespace is used when the
// hint leads nowhere or names another namespace.
func TestImportSchemaHostSuppliedDocument(t *testing.T) {
	dir := svDDir(t)
	f := Features{SchemaAware: true}
	body := `<xsl:value-of select="t:Small('3') instance of xs:integer"/>`
	compile := func(decl string, hs []SchemaSource) error {
		sheet := svDImportSheet(decl, body)
		ss, err := compileModule(sheet, dir, filepath.Join(dir, "main.xsl"), nil, nil, hs, CompileOptions{Features: &f})
		if err != nil {
			return err
		}
		res, err := ss.TransformFull("<x/>", nil, dir)
		if err != nil {
			return err
		}
		if got := strings.TrimSpace(res.Output); got != "true" {
			t.Errorf("%s: got %q, want \"true\"", decl, got)
		}
		return nil
	}
	rec := filepath.Join(dir, "rec.xsd")
	for _, tc := range []struct {
		name, decl string
		hs         []SchemaSource
	}{
		{"namespace only, host names namespace", `<xsl:import-schema namespace="urn:t"/>`, []SchemaSource{{Namespace: "urn:t", Path: rec}}},
		{"namespace only, host matched on targetNamespace", `<xsl:import-schema namespace="urn:t"/>`, []SchemaSource{{Path: filepath.Join(dir, "other-ns.xsd")}, {Path: rec}}},
		{"missing hint", `<xsl:import-schema namespace="urn:t" schema-location="no-such.xsd"/>`, []SchemaSource{{Namespace: "urn:t", Path: rec}}},
		{"hint in wrong namespace", `<xsl:import-schema namespace="urn:t" schema-location="other-ns.xsd"/>`, []SchemaSource{{Path: rec}}},
		{"unreadable host entries skipped", `<xsl:import-schema namespace="urn:t"/>`, []SchemaSource{{Path: ""}, {Namespace: "urn:t", Path: filepath.Join(dir, "absent.xsd")}, {Path: filepath.Join(dir, "absent.xsd")}, {Namespace: "urn:t", Path: rec}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := compile(tc.decl, tc.hs); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestSchemaComponentsForHost: the host-facing hooks exist exactly when the
// stylesheet is schema-aware and imported a schema, and resolve braced names.
func TestSchemaComponentsForHost(t *testing.T) {
	dir := svDDir(t)
	sheet := svDImportSheet(`<xsl:import-schema namespace="urn:t" schema-location="rec.xsd"/>`, `<xsl:value-of select="1"/>`)
	ss, err := CompileWith(sheet, dir, CompileOptions{Features: &Features{SchemaAware: true}})
	if err != nil {
		t.Fatal(err)
	}
	names, types := ss.SchemaComponentsForHost()
	if names == nil || types == nil {
		t.Fatal("hooks are nil for a schema-aware stylesheet that imports a schema")
	}
	if tn, ok := names.LookupSchemaType("Q{urn:t}RecT"); !ok || tn.Local != "RecT" || !tn.Complex {
		t.Errorf("LookupSchemaType(Q{urn:t}RecT) = %+v, %v", tn, ok)
	}
	if _, ok := names.LookupSchemaType("t:RecT"); ok {
		t.Error("a prefixed name resolved with no stylesheet element to resolve it against")
	}
	if n, _, ok := names.LookupSchemaElement("Q{urn:t}rec"); !ok || n.Space != "urn:t" || n.Local != "rec" {
		t.Errorf("LookupSchemaElement(Q{urn:t}rec) = %+v, %v", n, ok)
	}
	if _, _, ok := names.LookupSchemaAttribute("Q{urn:t}code"); !ok {
		t.Error("LookupSchemaAttribute(Q{urn:t}code) not found")
	}
	if _, _, ok := names.LookupSchemaElement("Q{urn:t}nope"); ok {
		t.Error("an undeclared element was found")
	}
	small, _ := names.LookupSchemaType("Q{urn:t}Small")
	integer, _ := names.LookupSchemaType("Q{http://www.w3.org/2001/XMLSchema}integer")
	if !types.DerivesFrom(small, integer) {
		t.Error("t:Small should derive from xs:integer")
	}
	// Not schema-aware, or nothing imported: no hooks.
	plain, err := CompileWith(svDSheet(`<xsl:template match="/"/>`), dir, CompileOptions{Features: &Features{SchemaAware: true}})
	if err != nil {
		t.Fatal(err)
	}
	if n, ty := plain.SchemaComponentsForHost(); n != nil || ty != nil {
		t.Error("hooks for a stylesheet that imports nothing")
	}
}
