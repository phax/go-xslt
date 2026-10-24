package xslt

import (
	"strings"
	"testing"
)

// xsl:accumulator (XSLT 3.0 §18.2). Expected values follow the informal and
// formal models of §18.2.3/§18.2.4: every node other than attributes and
// namespaces is visited twice; the pre-descent value is the accumulator after
// the node's start visit, the post-descent value after its end visit; when
// several rules match, the last in document order is used.

// mtEAccSheet builds a stylesheet with the given top-level declarations and a
// root template body. The unnamed mode makes every accumulator applicable to
// the source document (§18.2.2: for the initial match selection the
// applicable accumulators are those of the initial mode's xsl:mode).
func mtEAccSheet(decls, body string) string {
	return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  exclude-result-prefixes="xs map">
  <xsl:output method="text"/>
  <xsl:mode use-accumulators="#all"/>
  ` + decls + `
  <xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
}

const mtEAccSrc = `<doc><sec><p>a</p><p>b</p></sec><sec><p>c</p></sec></doc>`

// mtECount counts p elements: before = after the start rule.
const mtECount = `<xsl:accumulator name="n" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="p" select="$value + 1"/>
  </xsl:accumulator>`

func TestAccumulatorSpec(t *testing.T) {
	cases := []struct {
		name, decls, body, want string
	}{
		{"before-after", mtECount,
			`<xsl:for-each select="//sec">[<xsl:value-of select="accumulator-before('n'), accumulator-after('n')"/>]</xsl:for-each>`,
			"[0 2][2 3]"},
		{"document-node", mtECount,
			`<xsl:value-of select="accumulator-before('n'), accumulator-after('n')"/>`, "0 3"},
		// An end-phase rule changes the post-descent value of its node and
		// the values seen after it.
		{"end-phase", `<xsl:accumulator name="d" initial-value="()">
    <xsl:accumulator-rule match="sec" phase="start" select="($value, 's')"/>
    <xsl:accumulator-rule match="sec" phase="end" select="($value, 'e')"/>
  </xsl:accumulator>`,
			`<xsl:for-each select="//p">[<xsl:value-of select="accumulator-before('d')" separator=""/>]</xsl:for-each>`,
			"[s][s][ses]"},
		// Several matching rules: the last in document order wins.
		{"last-rule-wins", `<xsl:accumulator name="w" initial-value="''">
    <xsl:accumulator-rule match="p" select="'first'"/>
    <xsl:accumulator-rule match="*" select="'second'"/>
  </xsl:accumulator>`,
			`<xsl:value-of select="(//p)[1]/accumulator-before('w')"/>`, "second"},
		// Text nodes are visited; the rule's context item is the node.
		{"text-nodes", `<xsl:accumulator name="t" initial-value="''">
    <xsl:accumulator-rule match="text()" select="$value || ."/>
  </xsl:accumulator>`,
			`<xsl:value-of select="accumulator-after('t')"/>`, "abc"},
		// Rules whose value comes from a sequence constructor: a pure value
		// (xsl:sequence, xsl:choose, xsl:if), a map, and a node-building body.
		{"body-choose", `<xsl:accumulator name="c" initial-value="0">
    <xsl:accumulator-rule match="p">
      <xsl:choose>
        <xsl:when test=". = 'a'"><xsl:sequence select="$value + 10"/></xsl:when>
        <xsl:otherwise><xsl:sequence select="$value + 1"/></xsl:otherwise>
      </xsl:choose>
    </xsl:accumulator-rule>
  </xsl:accumulator>`,
			`<xsl:value-of select="accumulator-after('c')"/>`, "12"},
		{"body-choose-no-otherwise", `<xsl:accumulator name="c" initial-value="5">
    <xsl:accumulator-rule match="p">
      <xsl:choose><xsl:when test=". = 'z'"><xsl:sequence select="1"/></xsl:when></xsl:choose>
    </xsl:accumulator-rule>
  </xsl:accumulator>`,
			`<xsl:value-of select="count(accumulator-after('c'))"/>`, "0"},
		// The false xsl:if yields (), and () + 1 is () again.
		{"body-if", `<xsl:accumulator name="i" initial-value="0">
    <xsl:accumulator-rule match="p"><xsl:if test=". != 'b'"><xsl:sequence select="$value + 1"/></xsl:if></xsl:accumulator-rule>
  </xsl:accumulator>`,
			`<xsl:for-each select="//p">[<xsl:value-of select="count(accumulator-before('i'))"/>]</xsl:for-each>`, "[1][0][0]"},
		{"map-value", `<xsl:accumulator name="m" initial-value="map{}" as="map(xs:string, xs:integer)">
    <xsl:accumulator-rule match="p">
      <xsl:variable name="k" select="string(.)"/>
      <xsl:sequence select="map:put($value, $k, map:size($value) + 1)"/>
    </xsl:accumulator-rule>
  </xsl:accumulator>`,
			`<xsl:value-of select="let $m := accumulator-after('m') return ($m?a, $m?b, $m?c)"/>`, "1 2 3"},
		{"node-body", `<xsl:accumulator name="f" initial-value="()">
    <xsl:accumulator-rule match="p"><v><xsl:value-of select="."/></v></xsl:accumulator-rule>
  </xsl:accumulator>`,
			`<xsl:value-of select="(//p)[2]/accumulator-before('f')"/>`, "b"},
		// A path step's context item is the node asked about, not the
		// template's current node.
		{"path-context", mtECount,
			`<xsl:value-of select="//p/accumulator-before('n')"/>`, "1 2 3"},
		// A temporary tree is a tree to which all accumulators apply.
		{"temporary-tree", mtECount,
			`<xsl:variable name="t"><x><p/><p/></x></xsl:variable><xsl:value-of select="$t/x/accumulator-after('n')"/>`, "2"},
		// fn:copy-of keeps the accumulator values of the original (§18.2.2).
		{"copy-of-preserves", mtECount,
			`<xsl:value-of select="copy-of((//p)[3])/accumulator-before('n')"/>`, "3"},
		// The accumulator name is an EQName.
		{"eqname", `<xsl:accumulator name="Q{urn:x}n" initial-value="7">
    <xsl:accumulator-rule match="p" select="$value"/>
  </xsl:accumulator>`,
			`<xsl:value-of select="accumulator-before('Q{urn:x}n')"/>`, "7"},
		// $value is not in scope in @match; text that merely looks like it
		// inside a string literal or comment is not a reference.
		{"value-in-literal", `<xsl:accumulator name="s" initial-value="0">
    <xsl:accumulator-rule match="p[. != '$value'] (: $value :)" select="$value + 1"/>
  </xsl:accumulator>`,
			`<xsl:value-of select="accumulator-after('s')"/>`, "3"},
		// A dynamic error in a rule is reported only by a call that depends
		// on it (§18.2.5); with no call it goes unreported.
		{"unreported-error", `<xsl:accumulator name="e" initial-value="0">
    <xsl:accumulator-rule match="p" select="1 idiv 0"/>
  </xsl:accumulator>`,
			`ok`, "ok"},
		// A mode whose use-accumulators is #all can read the accumulator.
		{"mode-names-accumulator", mtECount + `<xsl:mode name="m" use-accumulators="#all"/>`,
			`<xsl:apply-templates select="//sec[2]" mode="m"/></xsl:template><xsl:template match="sec" mode="m"><xsl:value-of select="accumulator-before('n')"/>`, "2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mtERun(t, mtEAccSheet(c.decls, c.body), mtEAccSrc)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestAccumulatorSpecErrors(t *testing.T) {
	cases := []struct {
		name, decls, body string
		codes             []string
	}{
		// XTDE3340: no accumulator of that name.
		{"XTDE3340", mtECount, `<xsl:value-of select="accumulator-before('nope')"/>`, []string{"XTDE3340"}},
		// XTTE3360: the context item is an attribute, or not a node.
		{"XTTE3360-attribute", mtECount + ``, `<xsl:value-of select="parse-xml('&lt;a b=&quot;1&quot;/&gt;')/a/@b/accumulator-before('n')"/>`, []string{"XTTE3360"}},
		{"XTTE3360-atomic", mtECount, `<xsl:for-each select="1 to 2"><xsl:value-of select="accumulator-before('n')"/></xsl:for-each>`, []string{"XTTE3360"}},
		// XTDE3362: the mode's use-accumulators list does not include it.
		{"XTDE3362-mode", mtECount + `<xsl:mode name="none" use-accumulators=""/>`,
			`<xsl:apply-templates select="doc" mode="none"/></xsl:template><xsl:template match="doc" mode="none"><xsl:value-of select="accumulator-before('n')"/>`, []string{"XTDE3362"}},
		// The declared type applies to the initial value and every rule
		// result (function conversion rules; XPTY0004 when impossible).
		{"as-initial", `<xsl:accumulator name="x" initial-value="'s'" as="xs:integer">
    <xsl:accumulator-rule match="p" select="1"/>
  </xsl:accumulator>`, `<xsl:value-of select="accumulator-before('x')"/>`, []string{"XPTY0004"}},
		{"as-rule", `<xsl:accumulator name="x" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="p" select="'s'"/>
  </xsl:accumulator>`, `<xsl:value-of select="accumulator-after('x')"/>`, []string{"XPTY0004"}},
		// A dynamic error in a rule surfaces at a dependent call (§18.2.5).
		{"rule-error-reported", `<xsl:accumulator name="e" initial-value="0">
    <xsl:accumulator-rule match="p" select="1 idiv 0"/>
  </xsl:accumulator>`, `<xsl:value-of select="accumulator-after('e')"/>`, []string{"FOAR0001"}},
		// XTDE3400: the value depends on itself.
		{"XTDE3400", `<xsl:accumulator name="c" initial-value="0">
    <xsl:accumulator-rule match="p" select="$value + accumulator-before('c')"/>
  </xsl:accumulator>`, `<xsl:value-of select="accumulator-after('c')"/>`, []string{"XTDE3400"}},
		// XTSE3350: two declarations with one name at one precedence.
		{"XTSE3350", mtECount + mtECount, `x`, []string{"XTSE3350"}},
		// XTSE0010: an accumulator needs at least one rule.
		{"XTSE0010-no-rules", `<xsl:accumulator name="x" initial-value="0"/>`, `x`, []string{"XTSE0010"}},
		// XTSE0010: a required attribute is missing.
		{"XTSE0010-no-name", `<xsl:accumulator initial-value="0"><xsl:accumulator-rule match="p" select="1"/></xsl:accumulator>`, `x`, []string{"XTSE0010"}},
		{"XTSE0010-no-initial-value", `<xsl:accumulator name="a"><xsl:accumulator-rule match="p" select="1"/></xsl:accumulator>`, `x`, []string{"XTSE0010"}},
		{"XTSE0010-no-match", `<xsl:accumulator name="a" initial-value="0"><xsl:accumulator-rule select="1"/></xsl:accumulator>`, `x`, []string{"XTSE0010"}},
		// XTSE0020: streamable is an XSLT boolean.
		{"XTSE0020-streamable", `<xsl:accumulator name="x" initial-value="0" streamable="No">
    <xsl:accumulator-rule match="p" select="1"/>
  </xsl:accumulator>`, `x`, []string{"XTSE0020"}},
		// XPST0008: $value is not in scope in @match.
		{"XPST0008-match", `<xsl:accumulator name="x" initial-value="0">
    <xsl:accumulator-rule match="p[$value = 0]" select="1"/>
  </xsl:accumulator>`, `x`, []string{"XPST0008"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := mtERun(t, mtEAccSheet(c.decls, c.body), mtEAccSrc)
			if err == nil {
				t.Fatalf("want one of %v, got success", c.codes)
			}
			for _, code := range c.codes {
				if strings.Contains(err.Error(), "err:"+code) {
					return
				}
			}
			t.Errorf("want one of %v, got %v", c.codes, err)
		})
	}
}
