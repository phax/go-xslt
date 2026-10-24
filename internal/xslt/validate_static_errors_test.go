package xslt

import (
	"strings"
	"testing"
)

// The static checks of validate.go, driven through Compile: each case is a
// stylesheet the XSLT 3.0 REC (https://www.w3.org/TR/xslt-30/) declares to be
// in error, with the code the REC assigns. The section or error definition
// that makes it an error is named per case.

// svDSheet wraps top-level content in a 3.0 stylesheet with the usual
// prefixes bound.
func svDSheet(body string) string {
	return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xsl:stylesheet>`
}

// svDCompile compiles with explicit Features (never the process defaults).
func svDCompile(sheet string, f Features) (*Stylesheet, error) {
	return CompileWith(sheet, "", CompileOptions{Features: &f})
}

func svDWantCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("got no error, want %s", code)
	}
	if !strings.Contains(err.Error(), code) {
		t.Fatalf("got %v, want %s", err, code)
	}
}

func TestStaticErrorsOfStylesheetElements(t *testing.T) {
	tmpl := func(body string) string {
		return svDSheet(`<xsl:template match="/">` + body + `</xsl:template>`)
	}
	cases := []struct {
		name, sheet, code string
	}{
		// [ERR XTSE0809]: #default excluded with no default namespace.
		{"lre exclude #default without default ns", tmpl(`<out xsl:exclude-result-prefixes="#default"/>`), "XTSE0809"},
		// [ERR XTSE0808]: an excluded prefix must be bound.
		{"lre exclude unbound prefix", tmpl(`<out xsl:exclude-result-prefixes="nope"/>`), "XTSE0808"},
		{"stylesheet exclude unbound prefix", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="nope"><xsl:template match="/"/></xsl:stylesheet>`, "XTSE0808"},
		// [ERR XTSE1430]: extension-element-prefixes on a literal result element.
		{"lre extension prefix unbound", tmpl(`<out xsl:extension-element-prefixes="nope"/>`), "XTSE1430"},
		{"lre extension #default without default ns", tmpl(`<out xsl:extension-element-prefixes="#default"/>`), "XTSE1430"},
		// [ERR XTSE0805]: unknown XSLT-namespace attribute on an LRE.
		{"lre unknown xsl attribute", tmpl(`<out xsl:bogus="1"/>`), "XTSE0805"},
		// [ERR XTSE0110]: [xsl:]version must be an xs:decimal.
		{"lre version not decimal", tmpl(`<out xsl:version="three"/>`), "XTSE0110"},
		{"lre version with exponent", tmpl(`<out xsl:version="3e0"/>`), "XTSE0110"},
		{"stylesheet version not decimal", `<xsl:stylesheet version="x.y" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"/></xsl:stylesheet>`, "XTSE0110"},
		{"stylesheet version sign only", `<xsl:stylesheet version="+" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"/></xsl:stylesheet>`, "XTSE0110"},
		// [ERR XTSE0020]: values outside an attribute's permitted values.
		{"lre inherit-namespaces not boolean", tmpl(`<out xsl:inherit-namespaces="maybe"/>`), "XTSE0020"},
		{"lre expand-text not boolean", tmpl(`<out xsl:expand-text="maybe"/>`), "XTSE0020"},
		{"stylesheet expand-text not boolean", `<xsl:stylesheet version="3.0" expand-text="TRUE" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"/></xsl:stylesheet>`, "XTSE0020"},
		{"call-template name not a QName", tmpl(`<xsl:call-template name="1x"/>`), "XTSE0020"},
		{"copy-of copy-namespaces not boolean", tmpl(`<xsl:copy-of select="." copy-namespaces="perhaps"/>`), "XTSE0020"},
		{"message terminate fixed value", tmpl(`<xsl:message terminate="NO"/>`), "XTSE0020"},
		{"decimal-format zero-digit two chars", svDSheet(`<xsl:decimal-format zero-digit="00"/>`), "XTSE0020"},
		{"global-context-item use out of vocabulary", svDSheet(`<xsl:global-context-item use="sometimes"/>`), "XTSE0020"},
		{"output standalone out of vocabulary", svDSheet(`<xsl:output standalone="perhaps"/>`), "XTSE0020"},
		{"mode on-no-match out of vocabulary", svDSheet(`<xsl:mode on-no-match="explode"/>`), "XTSE0020"},
		{"mode on-multiple-match out of vocabulary", svDSheet(`<xsl:mode on-multiple-match="first"/>`), "XTSE0020"},
		{"mode typed out of vocabulary", svDSheet(`<xsl:mode typed="sometimes"/>`), "XTSE0020"},
		{"mode visibility out of vocabulary", svDSheet(`<xsl:mode name="m" visibility="secret"/>`), "XTSE0020"},
		{"apply-templates mode not a QName", tmpl(`<xsl:apply-templates mode="1bad"/>`), "XTSE0020"},
		{"function new-each-time is not boolean", svDSheet(`<xsl:function name="f:f" xmlns:f="urn:f" new-each-time="Yes"><xsl:sequence select="1"/></xsl:function>`), "XTSE0020"},
		// [ERR XTSE3089]: as with use="absent".
		{"global-context-item absent with as", svDSheet(`<xsl:global-context-item use="absent" as="item()"/>`), "XTSE3089"},
		// [ERR XTSE0120]: text at the top level.
		{"top-level text", svDSheet(`hello<xsl:template match="/"/>`), "XTSE0120"},
		// [ERR XTSE3120] / [ERR XTSE3130]: xsl:break / xsl:next-iteration.
		{"break not last in iterate", tmpl(`<xsl:iterate select="1 to 3"><xsl:break/><xsl:sequence select="."/></xsl:iterate>`), "XTSE3120"},
		{"break inside a literal result element", tmpl(`<xsl:iterate select="1 to 3"><x><xsl:break/></x></xsl:iterate>`), "XTSE3120"},
		{"break inside for-each inside iterate", tmpl(`<xsl:iterate select="1 to 3"><xsl:for-each select="1"><xsl:break/></xsl:for-each></xsl:iterate>`), "XTSE3120"},
		{"next-iteration param not declared", tmpl(`<xsl:iterate select="1 to 3"><xsl:param name="p" select="0"/><xsl:next-iteration><xsl:with-param name="q" select="1"/></xsl:next-iteration></xsl:iterate>`), "XTSE3130"},
		// [ERR XTSE0010]: element in a context where it is not permitted, a
		// required attribute omitted, or content not allowed.
		{"when outside choose", tmpl(`<xsl:if test="true()"><xsl:when test="true()"/></xsl:if>`), "XTSE0010"},
		{"when inside a literal result element", tmpl(`<x><xsl:when test="true()"/></x>`), "XTSE0010"},
		{"call-template without name", tmpl(`<xsl:call-template/>`), "XTSE0010"},
		{"accumulator without rules", svDSheet(`<xsl:accumulator name="a" initial-value="0"/>`), "XTSE0010"},
		{"namespace-alias with content", svDSheet(`<xsl:namespace-alias stylesheet-prefix="#default" result-prefix="#default"><x/></xsl:namespace-alias>`), "XTSE0010"},
		{"choose with a literal result element child", tmpl(`<xsl:choose><x/><xsl:when test="1"/></xsl:choose>`), "XTSE0010"},
		{"choose with text", tmpl(`<xsl:choose>text<xsl:when test="1"/></xsl:choose>`), "XTSE0010"},
		{"otherwise before when", tmpl(`<xsl:choose><xsl:otherwise/><xsl:when test="1"/></xsl:choose>`), "XTSE0010"},
		{"two otherwise", tmpl(`<xsl:choose><xsl:when test="1"/><xsl:otherwise/><xsl:otherwise/></xsl:choose>`), "XTSE0010"},
		{"choose without when", tmpl(`<xsl:choose><xsl:otherwise/></xsl:choose>`), "XTSE0010"},
		{"param after body", svDSheet(`<xsl:template name="t"><x/><xsl:param name="p"/></xsl:template>`), "XTSE0010"},
		{"sort after body", tmpl(`<xsl:for-each select="*"><x/><xsl:sort/></xsl:for-each>`), "XTSE0010"},
		{"non-matching before matching", tmpl(`<xsl:analyze-string select="'a'" regex="a"><xsl:non-matching-substring/><xsl:matching-substring/></xsl:analyze-string>`), "XTSE0010"},
		{"fallback before non-matching", tmpl(`<xsl:analyze-string select="'a'" regex="a"><xsl:fallback/><xsl:non-matching-substring/></xsl:analyze-string>`), "XTSE0010"},
		{"element child of an empty element", tmpl(`<xsl:copy-of select="."><x/></xsl:copy-of>`), "XTSE0010"},
		// [ERR XTSE0260]: whitespace kept by xml:space="preserve" in an empty element.
		{"preserved whitespace in an empty element", svDSheet(`<xsl:strip-space elements="*" xml:space="preserve"> </xsl:strip-space>`), "XTSE0260"},
		// [ERR XTSE1130]
		{"analyze-string with neither substring branch", tmpl(`<xsl:analyze-string select="'a'" regex="a"><xsl:fallback/></xsl:analyze-string>`), "XTSE1130"},
		// [ERR XTSE0090]: attributes not defined for an XSLT element.
		{"unknown attribute on value-of", tmpl(`<xsl:value-of select="1" colour="red"/>`), "XTSE0090"},
		{"xsl-namespace attribute on value-of", tmpl(`<xsl:value-of select="1" xsl:select="2"/>`), "XTSE0090"},
		{"static on a local variable", tmpl(`<xsl:variable name="v" static="yes" select="1"/>`), "XTSE0090"},
		// [ERR XTSE0500]
		{"name-only template with mode", svDSheet(`<xsl:template name="t" mode="m"/>`), "XTSE0500"},
		{"name-only template with priority", svDSheet(`<xsl:template name="t" priority="1"/>`), "XTSE0500"},
		{"match-only template with visibility", svDSheet(`<xsl:template match="/" visibility="public"/>`), "XTSE0500"},
		// [ERR XTSE0550]
		{"mode list duplicate token", svDSheet(`<xsl:template match="/" mode="a a"/>`), "XTSE0550"},
		{"mode list #all with others", svDSheet(`<xsl:template match="/" mode="#all a"/>`), "XTSE0550"},
		{"mode list empty", svDSheet(`<xsl:template match="/" mode=" "/>`), "XTSE0550"},
		{"mode list invalid token", svDSheet(`<xsl:template match="/" mode="1a"/>`), "XTSE0550"},
		// [ERR XTSE0280]: unbound prefix in a QName-valued attribute.
		{"mode list unbound prefix", svDSheet(`<xsl:template match="/" mode="p:m"/>`), "XTSE0280"},
		{"call-template unbound prefix", tmpl(`<xsl:call-template name="p:t"/>`), "XTSE0280"},
		{"apply-templates mode unbound prefix", tmpl(`<xsl:apply-templates mode="p:m"/>`), "XTSE0280"},
		// [ERR XTSE0080]: reserved namespace in a declared name.
		{"template in the XSLT namespace", svDSheet(`<xsl:template name="xsl:foo"/>`), "XTSE0080"},
		{"variable in the xml namespace", svDSheet(`<xsl:variable name="xml:v" select="1"/>`), "XTSE0080"},
		// Select attribute and content are mutually exclusive.
		{"value-of select and content", tmpl(`<xsl:value-of select="1">x</xsl:value-of>`), "XTSE0870"},                       // [ERR XTSE0870]
		{"attribute select and content", tmpl(`<x><xsl:attribute name="a" select="1"><y/></xsl:attribute></x>`), "XTSE0840"}, // [ERR XTSE0840]
		{"comment select and content", tmpl(`<xsl:comment select="1">x</xsl:comment>`), "XTSE0940"},                          // [ERR XTSE0940]
		// [ERR XTSE0975]
		{"number value with count", tmpl(`<xsl:number value="1" count="x"/>`), "XTSE0975"},
		{"number value with level", tmpl(`<xsl:number value="1" level="any"/>`), "XTSE0975"},
		// [ERR XTSE1570]
		{"output method unknown", svDSheet(`<xsl:output method="pdf"/>`), "XTSE1570"},
		{"output method not a QName", svDSheet(`<xsl:output method="1x"/>`), "XTSE1570"},
		// [ERR XTSE0670] / [ERR XTSE0580]
		{"duplicate with-param", tmpl(`<xsl:call-template name="t"><xsl:with-param name="a" select="1"/><xsl:with-param name="a" select="2"/></xsl:call-template>`), "XTSE0670"},
		{"duplicate param", svDSheet(`<xsl:template name="t"><xsl:param name="a"/><xsl:param name="a"/></xsl:template>`), "XTSE0580"},
		// [ERR XTSE0125]
		{"unknown default collation", `<xsl:stylesheet version="3.0" default-collation="urn:no-such-collation" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"/></xsl:stylesheet>`, "XTSE0125"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svDCompile(tc.sheet, Features{})
			svDWantCode(t, err, tc.code)
		})
	}
}

// TestStaticChecksAcceptValidForms pins the legal shapes next to the errors
// above, so a check that over-reaches is caught too.
func TestStaticChecksAcceptValidForms(t *testing.T) {
	tmpl := func(body string) string {
		return svDSheet(`<xsl:template match="/">` + body + `</xsl:template>`)
	}
	for _, tc := range []struct{ name, sheet string }{
		// #all and a bound prefix are both legal exclusions (§11.1.3).
		{"exclude #all and bound prefix", tmpl(`<out xmlns:p="urn:p" xsl:exclude-result-prefixes="#all p"/>`)},
		{"exclude #default with default ns", tmpl(`<out xmlns="urn:d" xsl:exclude-result-prefixes="#default"/>`)},
		// §8.3: a tail position may be inside xsl:if, xsl:choose branches.
		{"break inside last xsl:if", tmpl(`<xsl:iterate select="1 to 3"><xsl:if test=". = 2"><xsl:break/></xsl:if></xsl:iterate>`)},
		{"break in non-final when", tmpl(`<xsl:iterate select="1 to 3"><xsl:choose><xsl:when test=". = 2"><xsl:break/></xsl:when><xsl:otherwise><xsl:sequence select="."/></xsl:otherwise></xsl:choose></xsl:iterate>`)},
		{"next-iteration with declared param", tmpl(`<xsl:iterate select="1 to 3"><xsl:param name="p" select="0"/><xsl:next-iteration><xsl:with-param name="p" select="1"/></xsl:next-iteration></xsl:iterate>`)},
		{"break followed by a comment", tmpl(`<xsl:iterate select="1 to 3"><xsl:break/><!-- done --></xsl:iterate>`)},
		// XTSE0080's own exception: xsl:initial-template.
		{"xsl:initial-template", svDSheet(`<xsl:template name="xsl:initial-template"/>`)},
		// The mode list's reserved tokens and a bound EQName.
		{"mode tokens", svDSheet(`<xsl:template match="/" mode="#default #unnamed Q{urn:m}m"/>`)},
		// An EQName is not a prefixed QName (no XTSE0280).
		{"call-template EQName", svDSheet(`<xsl:template name="Q{urn:x}t"/><xsl:template match="/"><xsl:call-template name=" Q{urn:x}t "/></xsl:template>`)},
		// The context item declaration precedes the parameters (§10.1.1).
		{"context-item before params", svDSheet(`<xsl:template name="t"><xsl:context-item use="absent"/><xsl:param name="p"/><x/></xsl:template>`)},
		{"sort first in perform-sort", tmpl(`<xsl:perform-sort select="1"><xsl:sort/></xsl:perform-sort>`)},
		{"version is a decimal", tmpl(`<out xsl:version="3.0"/>`)},
		{"lre boolean attributes", tmpl(`<out xsl:inherit-namespaces="no" xsl:expand-text="1"/>`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svDCompile(tc.sheet, Features{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestForwardsCompatibleProcessingIgnoresUnknown pins §3.9: with an effective
// version above 3.0, an unknown XSLT element is ignored with its content, an
// unknown top-level XSLT element is skipped, and an attribute a known element
// does not define is ignored.
func TestForwardsCompatibleProcessingIgnoresUnknown(t *testing.T) {
	sheet := `<xsl:stylesheet version="4.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:future-declaration><xsl:call-template/></xsl:future-declaration>
  <xsl:template match="/">
    <xsl:value-of select="'ok'" future-attribute="x"/>
    <xsl:future-instruction><xsl:fallback/><xsl:when test="1"/></xsl:future-instruction>
  </xsl:template>
</xsl:stylesheet>`
	ss, err := svDCompile(sheet, Features{})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	out, _, err := ss.Transform("<a/>", nil)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if out != "ok" {
		t.Errorf("got %q, want %q", out, "ok")
	}
}

// TestSchemaAttributesWithoutSchemaAwareness pins [ERR XTSE1660]: a
// non-schema-aware processor rejects [xsl:]type anywhere, and
// [xsl:]validation / [xsl:]default-validation with a value other than strip,
// preserve or lax.
func TestSchemaAttributesWithoutSchemaAwareness(t *testing.T) {
	tmpl := func(body string) string {
		return svDSheet(`<xsl:template match="/">` + body + `</xsl:template>`)
	}
	for _, tc := range []struct{ name, sheet string }{
		{"lre xsl:type", tmpl(`<out xsl:type="xs:string"/>`)},
		{"lre xsl:validation strict", tmpl(`<out xsl:validation="strict"/>`)},
		{"lre xsl:default-validation strict", tmpl(`<out xsl:default-validation="strict"/>`)},
		{"copy-of validation strict", tmpl(`<xsl:copy-of select="." validation="strict"/>`)},
		{"stylesheet default-validation strict", `<xsl:stylesheet version="3.0" default-validation="strict" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"/></xsl:stylesheet>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svDCompile(tc.sheet, Features{})
			svDWantCode(t, err, "XTSE1660")
		})
	}
	// lax and preserve are explicitly allowed without schema awareness.
	for _, ok := range []string{
		tmpl(`<out xsl:validation="lax"/>`),
		tmpl(`<out xsl:validation="preserve" xsl:default-validation="strip"/>`),
		tmpl(`<xsl:copy-of select="." validation="lax"/>`),
	} {
		if _, err := svDCompile(ok, Features{}); err != nil {
			t.Errorf("unexpected error for %s: %v", ok, err)
		}
	}
	// Under schema awareness the default-validation vocabulary is still only
	// preserve|strip (§25.4: "The permitted values are preserve and strip"),
	// so strict there is an invalid value: XTSE0020.
	for _, bad := range []string{
		tmpl(`<out xsl:default-validation="strict"/>`),
		`<xsl:stylesheet version="3.0" default-validation="lax" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"/></xsl:stylesheet>`,
	} {
		_, err := svDCompile(bad, Features{SchemaAware: true})
		svDWantCode(t, err, "XTSE0020")
	}
}

// TestBackwardsCompatibleVersionNotClaimed pins [ERR XTDE0160]: an element
// with an effective version below 2.0, evaluated by a processor not
// configured for backwards-compatible behaviour.
func TestBackwardsCompatibleVersionNotClaimed(t *testing.T) {
	sheet := svDSheet(`<xsl:template match="/"><out xsl:version="1.0"/></xsl:template>`)
	ss, err := svDCompile(sheet, Features{})
	if err == nil {
		_, _, err = ss.Transform("<a/>", nil)
	}
	svDWantCode(t, err, "XTDE0160")
	ss, err = svDCompile(sheet, Features{BackwardsCompatible: true})
	if err != nil {
		t.Fatalf("with BackwardsCompatible: %v", err)
	}
	if _, _, err := ss.Transform("<a/>", nil); err != nil {
		t.Fatalf("with BackwardsCompatible: %v", err)
	}
}
