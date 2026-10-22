package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

func TestFeatureSchemaAware(t *testing.T) {
	xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema><xs:schema><xs:simpleType name="small"><xs:restriction base="xs:int"><xs:maxInclusive value="9"/></xs:restriction></xs:simpleType></xs:schema></xsl:import-schema>
  <xsl:template name="xsl:initial-template"><r aware="{system-property('xsl:is-schema-aware')}" a="{5 castable as small}" b="{50 castable as small}"/></xsl:template>
</xsl:stylesheet>`
	res := engine.Transform(engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template"})
	if !res.HasErrors() || !strings.Contains(res.Diagnostics[0].Message, "optional feature") {
		t.Fatalf("default: xsl:import-schema must be rejected, got %s %+v", res.Output, res.Diagnostics)
	}
	res = engine.Transform(engine.Request{Stylesheet: xsl, InitialTemplate: "xsl:initial-template",
		Features: engine.Features{SchemaAware: true}})
	if res.HasErrors() || !strings.Contains(res.Output, `aware="yes" a="true" b="false"`) {
		t.Fatalf("SchemaAware: got %s %+v", res.Output, res.Diagnostics)
	}
}

func TestFeatureBackwardsCompatible(t *testing.T) {
	xsl := `<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><r><xsl:value-of select="//a"/>|<xsl:value-of select="'x' + 1"/></r></xsl:template>
</xsl:stylesheet>`
	src := `<d><a>1</a><a>2</a></d>`
	res := engine.Transform(engine.Request{Stylesheet: xsl, Source: src})
	if !res.HasErrors() || res.Diagnostics[0].Code != "XPTY0004" {
		t.Fatalf("default (XSLT 3.0 rules): want XPTY0004, got %s %+v", res.Output, res.Diagnostics)
	}
	res = engine.Transform(engine.Request{Stylesheet: xsl, Source: src, Features: engine.Features{BackwardsCompatible: true}})
	if res.HasErrors() || !strings.Contains(res.Output, "<r>1|NaN</r>") {
		t.Fatalf("BackwardsCompatible (XSLT 1.0 rules): got %s %+v", res.Output, res.Diagnostics)
	}
}

func TestFeatureStrictStreaming(t *testing.T) {
	// Navigating to a sibling from a streamed node is not guaranteed-streamable.
	xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode streamable="yes"/>
  <xsl:template match="a"><x><xsl:value-of select="following-sibling::a"/></x></xsl:template>
</xsl:stylesheet>`
	src := `<d><a>1</a><a>2</a></d>`
	res := engine.Transform(engine.Request{Stylesheet: xsl, Source: src})
	if res.HasErrors() {
		t.Fatalf("default: an unprovable streamable mode must still run, got %+v", res.Diagnostics)
	}
	res = engine.Transform(engine.Request{Stylesheet: xsl, Source: src, Features: engine.Features{StrictStreaming: true}})
	if !res.HasErrors() || res.Diagnostics[0].Code != "XTSE3430" {
		t.Fatalf("StrictStreaming: want XTSE3430, got %s %+v", res.Output, res.Diagnostics)
	}
}

// A compiled stylesheet keeps the Features of its Compile request.
func TestFeaturesStayWithCompiledStylesheet(t *testing.T) {
	xsl := `<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><r><xsl:value-of select="//a"/></r></xsl:template>
</xsl:stylesheet>`
	bc, _ := engine.Compile(engine.Request{Stylesheet: xsl, Features: engine.Features{BackwardsCompatible: true}})
	modern, _ := engine.Compile(engine.Request{Stylesheet: xsl})
	src := `<d><a>1</a><a>2</a></d>`
	if r := bc.Transform(engine.TransformRequest{Source: src}); !strings.Contains(r.Output, "<r>1</r>") {
		t.Errorf("backwards-compatible: got %s", r.Output)
	}
	if r := modern.Transform(engine.TransformRequest{Source: src}); !strings.Contains(r.Output, "<r>1 2</r>") {
		t.Errorf("XSLT 3.0 rules: got %s", r.Output)
	}
}
