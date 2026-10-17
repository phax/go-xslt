package engine_test

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

const scopeHead = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:f="urn:f" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:variable name="v" select="/r"/>
  <xsl:function name="f:g" as="xs:string"><xsl:sequence select="string($v/a)"/></xsl:function>
  <xsl:function name="f:h" as="xs:string"><xsl:param name="v"/><xsl:sequence select="string($v)"/></xsl:function>
  <xsl:function name="f:loc" as="xs:string"><xsl:variable name="v" select="'mine'"/><xsl:sequence select="string($v)"/></xsl:function>
  <xsl:function name="f:viaFn" as="xs:string"><xsl:variable name="v" select="'callerfn'"/><xsl:sequence select="f:g()"/></xsl:function>
`

func scopeRun(t *testing.T, body string) engine.Result {
	t.Helper()
	return engine.Transform(engine.Request{Stylesheet: scopeHead + body + "</xsl:stylesheet>", Source: "<r><a>global</a></r>"})
}

// XSLT 3.0 §9.10: a function or template body sees the globals, its own
// parameters and locals — not the local variables of whatever invoked it.
func TestBodiesDoNotSeeCallersLocals(t *testing.T) {
	cases := map[string]string{
		"template":  `<xsl:template match="/"><xsl:variable name="v" select="'local'"/><out><xsl:value-of select="f:g()"/></out></xsl:template>`,
		"function":  `<xsl:template match="/"><out><xsl:value-of select="f:viaFn()"/></out></xsl:template>`,
		"for-each":  `<xsl:template match="/"><out><xsl:for-each select="1"><xsl:variable name="v" select="'loc'"/><xsl:value-of select="f:g()"/></xsl:for-each></out></xsl:template>`,
		"iterate":   `<xsl:template match="/"><out><xsl:iterate select="1"><xsl:param name="v" select="'p'"/><xsl:value-of select="f:g()"/></xsl:iterate></out></xsl:template>`,
		"tmpl-with": `<xsl:template match="/"><xsl:call-template name="t"><xsl:with-param name="v" select="'wp'"/></xsl:call-template></xsl:template><xsl:template name="t"><xsl:param name="v"/><out><xsl:value-of select="f:g()"/></out></xsl:template>`,
		"closure":   `<xsl:template match="/"><xsl:variable name="v" select="'local'"/><xsl:variable name="fn" select="function() { f:g() }"/><out><xsl:value-of select="$fn()"/></out></xsl:template>`,
		"lookup":    `<xsl:template match="/"><xsl:variable name="v" select="'local'"/><out><xsl:value-of select="function-lookup(xs:QName('f:g'),0)()"/></out></xsl:template>`,
		// a template that reads a GLOBAL of the same name as a local of its caller
		"call-template":   `<xsl:template match="/"><xsl:variable name="v" select="'local'"/><xsl:call-template name="t"/></xsl:template><xsl:template name="t"><out><xsl:value-of select="$v/a"/></out></xsl:template>`,
		"apply-templates": `<xsl:template match="/"><xsl:variable name="v" select="'local'"/><xsl:apply-templates select="r"/></xsl:template><xsl:template match="r"><out><xsl:value-of select="$v/a"/></out></xsl:template>`,
	}
	for name, body := range cases {
		r := scopeRun(t, body)
		if len(r.Diagnostics) != 0 || !strings.Contains(r.Output, ">global</out>") {
			t.Errorf("%s: %q %v", name, r.Output, r.Diagnostics)
		}
	}
}

func TestFunctionOwnVariablesStillShadowGlobals(t *testing.T) {
	r := scopeRun(t, `<xsl:template match="/"><out><xsl:value-of select="f:h('param'), f:loc(), string($v/a)"/></out></xsl:template>`)
	if !strings.Contains(r.Output, ">param mine global</out>") || len(r.Diagnostics) != 0 {
		t.Errorf("%q %v", r.Output, r.Diagnostics)
	}
}

// A closure keeps the variables of the scope it was created in.
func TestClosureKeepsItsOwnScope(t *testing.T) {
	r := scopeRun(t, `<xsl:function name="f:mk"><xsl:variable name="c" select="'cap'"/><xsl:sequence select="function() { $c }"/></xsl:function>
	  <xsl:template match="/"><out><xsl:value-of select="f:mk()()"/></out></xsl:template>`)
	if !strings.Contains(r.Output, ">cap</out>") || len(r.Diagnostics) != 0 {
		t.Errorf("%q %v", r.Output, r.Diagnostics)
	}
}
