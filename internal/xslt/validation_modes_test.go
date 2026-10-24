package xslt

import (
	"path/filepath"
	"strings"
	"testing"
)

// [xsl:]validation and [xsl:]type at run time (XSLT 3.0 §25.4, "Validation"),
// observed through instance-of tests on the constructed nodes. The schema is
// testdata/validation/rec.xsd (namespace urn:t): t:n is xs:integer, t:rec is
// t:RecT (a sequence of t:n with an xs:ID @id), t:Small is xs:integer <= 10,
// t:MyQName restricts xs:QName, @t:code is a global t:Small attribute and
// t:ids holds i elements with xs:ID/xs:IDREF attributes.

func svDDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.Abs(filepath.Join("testdata", "validation"))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// svDSASheet is a schema-aware text-output stylesheet importing rec.xsd.
func svDSASheet(top, body string) string {
	return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" exclude-result-prefixes="#all">
  <xsl:import-schema namespace="urn:t" schema-location="rec.xsd"/>
  <xsl:output method="text"/>` + top + `
  <xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
}

// svDRunSA compiles schema-aware against testdata/validation and runs once
// on src; it returns the trimmed output, or the compile or run error.
func svDRunSA(t *testing.T, sheet, src string) (string, error) {
	t.Helper()
	dir := svDDir(t)
	ss, err := CompileWith(sheet, dir, CompileOptions{Features: &Features{SchemaAware: true}})
	if err != nil {
		return "", err
	}
	res, err := ss.TransformFull(src, nil, dir)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Output), nil
}

func TestValidationOutcomes(t *testing.T) {
	cases := []struct {
		name, top, body string
		want            string // expected output when code == ""
		code            string
	}{
		// §25.4.1.1 strict: the element is assessed against its top-level
		// declaration and annotated with the declared type.
		{name: "xsl:element strict valid",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="t:n" validation="strict">5</xsl:element></xsl:variable>
				<xsl:value-of select="$v instance of element(t:n, xs:integer), data($v) + 1"/>`,
			want: "true 6"},
		{name: "xsl:element strict invalid", code: "XTTE1510",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="t:n" validation="strict">five</xsl:element></xsl:variable><xsl:value-of select="$v"/>`},
		{name: "xsl:element strict undeclared", code: "XTTE1512",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="t:nope" validation="strict">5</xsl:element></xsl:variable><xsl:value-of select="$v"/>`},
		// lax: a declared element must conform (XTTE1515) ...
		{name: "xsl:element lax invalid", code: "XTTE1515",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="t:n" validation="lax">five</xsl:element></xsl:variable><xsl:value-of select="$v"/>`},
		{name: "xsl:element lax valid",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="t:n" validation="lax">7</xsl:element></xsl:variable>
				<xsl:value-of select="$v instance of element(*, xs:integer)"/>`,
			want: "true"},
		// ... and an undeclared one is not an error.
		{name: "xsl:element lax undeclared",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="t:nope" validation="lax">x</xsl:element></xsl:variable><xsl:value-of select="string($v)"/>`,
			want: "x"},
		// strip (also the default): xs:untyped.
		{name: "xsl:element strip",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="t:n" validation="strip">5</xsl:element></xsl:variable>
				<xsl:value-of select="$v instance of element(*, xs:untyped)"/>`,
			want: "true"},
		{name: "default is strip",
			body: `<xsl:variable name="v" as="element()"><t:n>5</t:n></xsl:variable>
				<xsl:value-of select="$v instance of element(*, xs:untyped)"/>`,
			want: "true"},
		// preserve on xsl:element: the new element is xs:anyType, contained
		// nodes keep their annotations.
		{name: "xsl:element preserve",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="wrap" validation="preserve"><xsl:element name="t:n" validation="strict">3</xsl:element></xsl:element></xsl:variable>
				<xsl:value-of select="$v instance of element(*, xs:anyType), $v instance of element(*, xs:untyped), $v/t:n instance of element(*, xs:integer)"/>`,
			want: "true false true"},
		// [xsl:]default-validation="preserve" makes preserve the effective value.
		{name: "default-validation preserve on a literal result element",
			body: `<xsl:variable name="v" as="element()"><wrap xsl:default-validation="preserve"><xsl:element name="t:n" validation="strict">3</xsl:element></wrap></xsl:variable>
				<xsl:value-of select="$v instance of element(*, xs:untyped), $v/t:n instance of element(*, xs:integer)"/>`,
			want: "false true"},
		// A parent's strip removes what a child's validation gave (§25.4.1.1:
		// the parent's validation is applied after the child's).
		{name: "parent strip overrides child",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="wrap" validation="strip"><xsl:element name="t:n" validation="strict">3</xsl:element></xsl:element></xsl:variable>
				<xsl:value-of select="$v/t:n instance of element(*, xs:untyped)"/>`,
			want: "true"},
		// Literal result elements use xsl:validation / xsl:type.
		{name: "lre xsl:validation strict",
			body: `<xsl:variable name="v" as="element()"><t:rec xsl:validation="strict"><t:n>1</t:n><t:n>2</t:n></t:rec></xsl:variable>
				<xsl:value-of select="$v instance of element(t:rec, t:RecT), sum($v/t:n)"/>`,
			want: "true 3"},
		{name: "lre xsl:validation strict invalid content", code: "XTTE1510",
			body: `<xsl:variable name="v" as="element()"><t:rec xsl:validation="strict"><t:n>1</t:n><other/></t:rec></xsl:variable><xsl:value-of select="$v"/>`},
		// §25.4.1.2: validation against a named type.
		{name: "lre xsl:type valid",
			body: `<xsl:variable name="v" as="element()"><e xsl:type="xs:integer">42</e></xsl:variable>
				<xsl:value-of select="$v instance of element(e, xs:integer)"/>`,
			want: "true"},
		{name: "xsl:element type invalid", code: "XTTE1540",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="e" type="t:Small">11</xsl:element></xsl:variable><xsl:value-of select="$v"/>`},
		{name: "xsl:element type user simple type",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="e" type="t:Small">10</xsl:element></xsl:variable>
				<xsl:value-of select="$v instance of element(*, t:Small), $v instance of element(*, xs:integer)"/>`,
			want: "true true"},
		// xs:untyped is the same as validation="strip".
		{name: "type xs:untyped",
			body: `<xsl:variable name="v" as="element()"><xsl:element name="e" type="xs:untyped"><xsl:element name="t:n" validation="strict">1</xsl:element></xsl:element></xsl:variable>
				<xsl:value-of select="$v instance of element(*, xs:untyped), $v/t:n instance of element(*, xs:untyped)"/>`,
			want: "true true"},
		// xs:untypedAtomic: like xs:string, but fails for element children.
		{name: "type xs:untypedAtomic with element children", code: "XTTE1540",
			body: `<xsl:variable name="v" as="element()"><e xsl:type="xs:untypedAtomic">a<b/>c</e></xsl:variable><xsl:value-of select="$v"/>`},
		// Attributes (§25.4.1): strict assessment against the global declaration,
		// type against a simple type. The parent is built with preserve, since
		// a parent's (default) strip would remove the attribute's annotation.
		{name: "xsl:attribute strict valid",
			body: `<xsl:variable name="v" as="element()"><e xsl:validation="preserve"><xsl:attribute name="t:code" validation="strict">4</xsl:attribute></e></xsl:variable>
				<xsl:value-of select="$v/@t:code instance of attribute(*, t:Small)"/>`,
			want: "true"},
		{name: "xsl:attribute strict invalid", code: "XTTE1510",
			body: `<xsl:variable name="v" as="element()"><e><xsl:attribute name="t:code" validation="strict">40</xsl:attribute></e></xsl:variable><xsl:value-of select="$v"/>`},
		{name: "xsl:attribute strict undeclared", code: "XTTE1512",
			body: `<xsl:variable name="v" as="element()"><e><xsl:attribute name="t:other" validation="strict">4</xsl:attribute></e></xsl:variable><xsl:value-of select="$v"/>`},
		{name: "xsl:attribute type invalid", code: "XTTE1540",
			body: `<xsl:variable name="v" as="element()"><e><xsl:attribute name="a" type="xs:date">yesterday</xsl:attribute></e></xsl:variable><xsl:value-of select="$v"/>`},
		{name: "xsl:attribute preserve is strip",
			body: `<xsl:variable name="v" as="element()"><e xsl:validation="preserve"><xsl:attribute name="a" type="xs:integer">4</xsl:attribute></e></xsl:variable>
				<xsl:variable name="w" as="element()"><e xsl:validation="preserve"><xsl:attribute name="a" select="$v/@a" validation="preserve"/></e></xsl:variable>
				<xsl:value-of select="$v/@a instance of attribute(*, xs:integer), $w/@a instance of attribute(*, xs:untypedAtomic)"/>`,
			want: "true true"},
		// §25.4.2: a validated document node must have exactly one element child
		// and no text nodes (XTTE1550); the type applies to that element.
		{name: "xsl:document strict valid",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document validation="strict"><t:n>8</t:n></xsl:document></xsl:variable>
				<xsl:value-of select="$d/t:n instance of element(t:n, xs:integer)"/>`,
			want: "true"},
		{name: "xsl:document two elements", code: "XTTE1550",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document validation="strict"><t:n>8</t:n><t:n>9</t:n></xsl:document></xsl:variable><xsl:value-of select="$d"/>`},
		{name: "xsl:document text child", code: "XTTE1550",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document validation="lax">text<t:n>8</t:n></xsl:document></xsl:variable><xsl:value-of select="$d"/>`},
		{name: "xsl:document type text child", code: "XTTE1550",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document type="t:RecT">text</xsl:document></xsl:variable><xsl:value-of select="$d"/>`},
		{name: "xsl:document type valid",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document type="t:RecT"><r><t:n>1</t:n></r></xsl:document></xsl:variable>
				<xsl:value-of select="$d/r instance of element(r, t:RecT)"/>`,
			want: "true"},
		// [ERR XTTE1555]: document-level ID/IDREF constraints.
		{name: "xsl:document duplicate ID", code: "XTTE1555",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document validation="strict"><t:ids><t:i id="a"/><t:i id="a"/></t:ids></xsl:document></xsl:variable><xsl:value-of select="count($d//*)"/>`},
		{name: "xsl:document dangling IDREF", code: "XTTE1555",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document validation="strict"><t:ids><t:i id="a" ref="b"/></t:ids></xsl:document></xsl:variable><xsl:value-of select="count($d//*)"/>`},
		{name: "xsl:document ID and IDREF satisfied",
			body: `<xsl:variable name="d" as="document-node()"><xsl:document validation="strict"><t:ids><t:i id="a" ref="b"/><t:i id="b"/></t:ids></xsl:document></xsl:variable>
				<xsl:value-of select="$d/id('b')/@id = 'b'"/>`,
			want: "true"},
		// xsl:copy-of with validation and type.
		{name: "copy-of strict",
			body: `<xsl:variable name="raw" as="element()"><t:n>12</t:n></xsl:variable>
				<xsl:variable name="v" as="element()"><xsl:copy-of select="$raw" validation="strict"/></xsl:variable>
				<xsl:value-of select="$raw instance of element(*, xs:untyped), $v instance of element(t:n, xs:integer)"/>`,
			want: "true true"},
		{name: "copy-of type invalid", code: "XTTE1540",
			body: `<xsl:variable name="raw" as="element()"><e>x</e></xsl:variable>
				<xsl:variable name="v" as="element()"><xsl:copy-of select="$raw" type="xs:integer"/></xsl:variable><xsl:value-of select="$v"/>`},
		// [ERR XTTE1535]: complex type, but an attribute is being copied.
		{name: "copy-of complex type with an attribute", code: "XTTE1535",
			body: `<xsl:variable name="raw" as="element()"><e a="1"/></xsl:variable>
				<xsl:variable name="v" as="element()"><e><xsl:copy-of select="$raw/@a" type="t:RecT"/></e></xsl:variable><xsl:value-of select="$v"/>`},
		// copy-of preserve keeps annotations; xsl:copy preserve makes the
		// copied element xs:anyType (§25.4.1.1).
		{name: "copy-of preserve and copy preserve",
			body: `<xsl:variable name="typed" as="element()"><xsl:element name="t:n" validation="strict">3</xsl:element></xsl:variable>
				<xsl:variable name="a" as="element()"><xsl:copy-of select="$typed" validation="preserve"/></xsl:variable>
				<xsl:variable name="b" as="element()"><xsl:for-each select="$typed"><xsl:copy validation="preserve"><xsl:sequence select="string(.)"/></xsl:copy></xsl:for-each></xsl:variable>
				<xsl:value-of select="$a instance of element(*, xs:integer), $b instance of element(*, xs:anyType), $b instance of element(*, xs:integer)"/>`,
			want: "true true false"},
		{name: "copy strict",
			body: `<xsl:variable name="raw" as="element()"><t:n>4</t:n></xsl:variable>
				<xsl:variable name="v" as="element()"><xsl:for-each select="$raw"><xsl:copy validation="strict">4</xsl:copy></xsl:for-each></xsl:variable>
				<xsl:value-of select="$v instance of element(t:n, xs:integer)"/>`,
			want: "true"},
		// copy-of of a mixture: text nodes are not validation targets.
		{name: "copy-of strict text and element",
			body: `<xsl:variable name="raw" as="element()"><w>txt<t:n>4</t:n></w></xsl:variable>
				<xsl:variable name="v" as="element()"><w xsl:validation="preserve"><xsl:copy-of select="$raw/node()" validation="strict"/></w></xsl:variable>
				<xsl:value-of select="string($v), $v/t:n instance of element(*, xs:integer)"/>`,
			want: "txt4 true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svDRunSA(t, svDSASheet(tc.top, tc.body), "<x/>")
			if tc.code != "" {
				svDWantCode(t, err, tc.code)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestValidationStaticErrors pins the compile-time errors of §25.4.
func TestValidationStaticErrors(t *testing.T) {
	cases := []struct{ name, body, code string }{
		// [ERR XTSE1505]: both type and validation.
		{"element type and validation", `<xsl:element name="e" type="xs:integer" validation="strict"/>`, "XTSE1505"},
		{"lre type and validation", `<e xsl:type="xs:integer" xsl:validation="strict"/>`, "XTSE1505"},
		// [ERR XTSE1520]: not a QName, unbound prefix, or no such type.
		{"type not a QName", `<xsl:element name="e" type="1bad"/>`, "XTSE1520"},
		{"type unbound prefix", `<xsl:element name="e" type="zz:T"/>`, "XTSE1520"},
		{"type not in schema", `<xsl:element name="e" type="t:NoSuchType"/>`, "XTSE1520"},
		// [ERR XTSE1530]: xsl:attribute with a complex type.
		{"attribute complex type", `<e><xsl:attribute name="a" type="t:RecT"/></e>`, "XTSE1530"},
		// [ERR XTSE0020]: default-validation only allows preserve and strip.
		{"default-validation lax", `<e xsl:default-validation="lax"/>`, "XTSE0020"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svDRunSA(t, svDSASheet("", tc.body), "<x/>")
			svDWantCode(t, err, tc.code)
		})
	}
}

// TestValidationParentlessNamespaceSensitiveAttribute pins [ERR XTTE1545]: a
// new attribute validated against a type derived from xs:QName. It may be
// raised statically or dynamically ("As with other type errors").
func TestValidationParentlessNamespaceSensitiveAttribute(t *testing.T) {
	for _, typ := range []string{"xs:QName", "t:MyQName"} {
		_, err := svDRunSA(t, svDSASheet("", `<e><xsl:attribute name="a" type="`+typ+`">t:x</xsl:attribute></e>`), "<x/>")
		svDWantCode(t, err, "XTTE1545")
	}
}

// TestValidationBuiltinTypesWithoutImport: the built-in types are always in
// the in-scope schema components (§3.15), so [xsl:]type="xs:integer" needs no
// xsl:import-schema; strict validation without one finds no declaration.
func TestValidationBuiltinTypesWithoutImport(t *testing.T) {
	sheet := func(body string) string {
		return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"><xsl:output method="text"/>
  <xsl:template match="/">` + body + `</xsl:template></xsl:stylesheet>`
	}
	got, err := svDRunSA(t, sheet(`<xsl:variable name="v" as="element()"><e xsl:type="xs:integer">5</e></xsl:variable><xsl:value-of select="$v instance of element(e, xs:integer)"/>`), "<x/>")
	if err != nil || got != "true" {
		t.Fatalf("got %q, %v; want \"true\"", got, err)
	}
	_, err = svDRunSA(t, sheet(`<xsl:variable name="v" as="element()"><e xsl:validation="strict">5</e></xsl:variable><xsl:value-of select="$v"/>`), "<x/>")
	svDWantCode(t, err, "XTTE1512")
	_, err = svDRunSA(t, sheet(`<xsl:variable name="v" as="element()"><e xsl:type="xs:integer">x</e></xsl:variable><xsl:value-of select="$v"/>`), "<x/>")
	svDWantCode(t, err, "XTTE1540")
}

// TestSourceDocumentValidation covers validating the primary source and an
// xsl:source-document input (§25.4: strict / lax requested for a source).
func TestSourceDocumentValidation(t *testing.T) {
	dir := svDDir(t)
	sheet := svDSASheet("", `<xsl:value-of select="/t:rec instance of element(t:rec, t:RecT), sum(/t:rec/t:n)"/>`)
	ss, err := CompileWith(sheet, dir, CompileOptions{Features: &Features{SchemaAware: true}})
	if err != nil {
		t.Fatal(err)
	}
	valid := `<t:rec xmlns:t="urn:t"><t:n>1</t:n><t:n>2</t:n></t:rec>`
	run := func(src, mode string) (string, error) {
		res, err := ss.TransformEntry(src, Entry{SourceValidation: mode}, dir)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(res.Output), nil
	}
	for _, mode := range []string{"strict", "lax"} {
		got, err := run(valid, mode)
		if err != nil || got != "true 3" {
			t.Errorf("%s valid: got %q, %v; want \"true 3\"", mode, got, err)
		}
	}
	// Unvalidated, the source is untyped.
	if got, err := run(valid, ""); err != nil || got != "false 3" {
		t.Errorf("no validation: got %q, %v; want \"false 3\"", got, err)
	}
	_, err = run(`<t:rec xmlns:t="urn:t"><t:n>x</t:n></t:rec>`, "strict")
	svDWantCode(t, err, "XTTE1510")
	_, err = run(`<t:rec xmlns:t="urn:t"><t:n>x</t:n></t:rec>`, "lax")
	svDWantCode(t, err, "XTTE1515")
	_, err = run(`<t:unknown xmlns:t="urn:t"/>`, "strict")
	svDWantCode(t, err, "XTTE1512")

	// §4.4.1: input-type-annotations="strip" discards the annotations a
	// validated source received.
	strip := strings.Replace(sheet, `version="3.0"`, `version="3.0" input-type-annotations="strip"`, 1)
	ss2, err := CompileWith(strip, dir, CompileOptions{Features: &Features{SchemaAware: true}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := ss2.TransformEntry(valid, Entry{SourceValidation: "strict"}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(res.Output); got != "false 3" {
		t.Errorf("input-type-annotations=strip: got %q, want \"false 3\"", got)
	}

	// The host may name the schema itself (Entry.SourceSchemas) for a
	// stylesheet that imports none.
	plain := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:t="urn:t"><xsl:output method="text"/>
  <xsl:template match="/"><xsl:value-of select="data(/t:rec/t:n[1]) instance of xs:integer"/></xsl:template></xsl:stylesheet>`
	ss3, err := CompileWith(plain, dir, CompileOptions{Features: &Features{SchemaAware: true}})
	if err != nil {
		t.Fatal(err)
	}
	res, err = ss3.TransformEntry(valid, Entry{SourceValidation: "strict", SourceSchemas: []string{filepath.Join(dir, "rec.xsd")}}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(res.Output); got != "true" {
		t.Errorf("host source schema: got %q, want \"true\"", got)
	}
	_, err = ss3.TransformEntry(`<t:rec xmlns:t="urn:t"><t:n>x</t:n></t:rec>`, Entry{SourceValidation: "strict", SourceSchemas: []string{filepath.Join(dir, "rec.xsd")}}, dir)
	svDWantCode(t, err, "XTTE1510")
}

func TestSourceDocumentInstructionValidation(t *testing.T) {
	for _, tc := range []struct{ name, attrs, file, want, code string }{
		{"strict valid", `validation="strict"`, "rec-valid.xml", "true", ""},
		{"strict invalid", `validation="strict"`, "rec-invalid.xml", "", "XTTE1510"},
		{"lax invalid", `validation="lax"`, "rec-invalid.xml", "", "XTTE1515"},
		{"no validation", ``, "rec-valid.xml", "false", ""},
		{"type", `type="t:RecT"`, "rec-valid.xml", "true", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `<xsl:source-document href="` + tc.file + `" ` + tc.attrs + `><xsl:value-of select="t:rec instance of element(*, t:RecT)"/></xsl:source-document>`
			got, err := svDRunSA(t, svDSASheet("", body), "<x/>")
			if tc.code != "" {
				svDWantCode(t, err, tc.code)
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

// TestResultDocumentValidation: xsl:result-document validates the document
// node it constructs (§25.4.2).
func TestResultDocumentValidation(t *testing.T) {
	_, err := svDRunSA(t, svDSASheet("", `<xsl:result-document validation="strict" method="xml"><t:n>x</t:n></xsl:result-document>`), "<x/>")
	svDWantCode(t, err, "XTTE1510")
	_, err = svDRunSA(t, svDSASheet("", `<xsl:result-document validation="strict" method="xml"><t:n>1</t:n><t:n>2</t:n></xsl:result-document>`), "<x/>")
	svDWantCode(t, err, "XTTE1550")
	if _, err := svDRunSA(t, svDSASheet("", `<xsl:result-document type="xs:integer" method="xml"><n>1</n></xsl:result-document>`), "<x/>"); err != nil {
		t.Errorf("valid result document: %v", err)
	}
}

// TestResultDocumentItemSeparatorIsValidated: sequence normalization
// (Serialization 3.1 §2, step 3) inserts the item-separator between the
// top-level items as text, so a validated result document of a comment and an
// element gains a text child and violates §25.4.2 (XTTE1550).
func TestResultDocumentItemSeparatorIsValidated(t *testing.T) {
	_, err := svDRunSA(t, svDSASheet("", `<xsl:result-document validation="strict" method="xml" item-separator="+"><xsl:comment>c</xsl:comment><t:n>1</t:n></xsl:result-document>`), "<x/>")
	svDWantCode(t, err, "XTTE1550")
	// Without the separator the same content is a valid document.
	if _, err := svDRunSA(t, svDSASheet("", `<xsl:result-document validation="strict" method="xml"><xsl:comment>c</xsl:comment><t:n>1</t:n></xsl:result-document>`), "<x/>"); err != nil {
		t.Errorf("without item-separator: %v", err)
	}
}
