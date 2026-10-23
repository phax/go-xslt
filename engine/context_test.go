package engine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// A done context stops every kind of run. The context is cancelled before the
// run starts, so the verdict does not depend on how fast the machine is.
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stoppedBy := func(name string, diags []engine.Diagnostic) {
		t.Helper()
		if len(diags) == 0 || !strings.Contains(diags[0].Message, "context canceled") {
			t.Errorf("%s: want a cancellation, got %+v", name, diags)
		}
	}

	// A long loop of instructions.
	loop := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template">
    <xsl:iterate select="1 to 9000000"><xsl:value-of select="."/></xsl:iterate>
  </xsl:template>
</xsl:stylesheet>`
	req := engine.Request{Stylesheet: loop, InitialTemplate: "xsl:initial-template"}
	stoppedBy("TransformContext", engine.TransformContext(ctx, req).Diagnostics)
	var b strings.Builder
	stoppedBy("TransformToContext", engine.TransformToContext(ctx, &b, req).Diagnostics)
	sheet, diags := engine.Compile(req)
	if sheet == nil {
		t.Fatal(diags)
	}
	stoppedBy("Stylesheet.TransformContext", sheet.TransformContext(ctx, engine.TransformRequest{InitialTemplate: "xsl:initial-template"}).Diagnostics)

	// One XPath expression doing all the work, inside a single instruction.
	xp := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template"><xsl:value-of select="count(for $i in 1 to 9000000 return $i * 2)"/></xsl:template>
</xsl:stylesheet>`
	stoppedBy("XPath inside XSLT", engine.TransformContext(ctx, engine.Request{Stylesheet: xp, InitialTemplate: "xsl:initial-template"}).Diagnostics)
	stoppedBy("EvalXPathContext", engine.EvalXPathContext(ctx, engine.XPathRequest{Expression: "sum(for $i in 1 to 9000000 return $i)"}).Diagnostics)

	// Validation of a document with many elements.
	inst := "<r>" + strings.Repeat("<i>1</i>", 5000) + "</r>"
	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="r"><xs:complexType><xs:sequence>
  <xs:element name="i" type="xs:int" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element></xs:schema>`
	v := engine.ValidateContext(ctx, engine.ValidateRequest{Schemas: []string{schema}, Instance: inst})
	if v.Valid {
		t.Error("ValidateContext: a cancelled validation is not valid")
	}
	stoppedBy("ValidateContext", v.Diagnostics)

	// Without cancellation the same calls still work.
	if r := engine.EvalXPathContext(context.Background(), engine.XPathRequest{Expression: "sum(1 to 100)"}); r.Value != "5050" {
		t.Errorf("background context: %q %+v", r.Value, r.Diagnostics)
	}
	if v := engine.ValidateContext(context.Background(), engine.ValidateRequest{Schemas: []string{schema}, Instance: inst}); !v.Valid {
		t.Errorf("background validation: %+v", v.Diagnostics)
	}
}

func TestMaxOutputBytes(t *testing.T) {
	xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template name="xsl:initial-template">
    <xsl:for-each select="1 to 1000">0123456789</xsl:for-each>
    <xsl:result-document href="side.txt"><xsl:for-each select="1 to 100">0123456789</xsl:for-each></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`
	req := func(max int64) engine.Request {
		return engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template", Limits: engine.Limits{MaxOutputBytes: max}}
	}
	if res := engine.Transform(req(11_000)); res.HasErrors() {
		t.Errorf("11,000 bytes allowed: %+v", res.Diagnostics)
	}
	// 10,000 principal + 1,000 secondary bytes do not fit in 10,500.
	if res := engine.Transform(req(10_500)); !res.HasErrors() || res.Diagnostics[0].Code != "XPDY0130" {
		t.Errorf("10,500 bytes: want XPDY0130, got %+v", res.Diagnostics)
	}
	var b strings.Builder
	if res := engine.TransformTo(&b, req(5_000)); !res.HasErrors() || b.Len() > 5_000 {
		t.Errorf("TransformTo 5,000 bytes: wrote %d, %+v", b.Len(), res.Diagnostics)
	}
}
