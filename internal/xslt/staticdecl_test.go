package xslt

import (
	"testing"
)

// Static variables and parameters across stylesheet modules (XSLT 3.0 §9.6):
// stylesheet tree order (xsl:include/xsl:import replaced in place by the
// module's declarations), import precedence, and XTSE3450. The modules live
// in testdata/usewhen.

// swBMain is the location the principal module is compiled "from", so that
// relative hrefs resolve into testdata/usewhen.
const swBMain = "testdata/usewhen/main.xsl"

// swBRunMulti compiles a principal module located at swBMain and runs it
// against src.
func swBRunMulti(t *testing.T, sheet, src string, static map[string]string) (string, error) {
	t.Helper()
	ss, err := CompileAtWithStatic(sheet, swBMain, static)
	if err != nil {
		return "", err
	}
	out, _, err := ss.Transform(src, nil)
	return out, err
}

func TestStaticTreeOrderAcrossModules(t *testing.T) {
	cases := []struct {
		name, decls, src string
		static           map[string]string
		want             string
	}{
		{
			// The included module's static variable precedes, in tree order,
			// whatever follows the xsl:include.
			"included static visible after include",
			`<xsl:include href="inc_static.xsl"/>
			 <xsl:template match="/"><xsl:text use-when="$inc-v = 'from-inc'">seen:</xsl:text><xsl:value-of select="$inc-v"/></xsl:template>`,
			"<doc/>", nil, "seen:from-inc",
		},
		{
			// §3.13.1: use-when false on a secondary module's root excludes
			// all its children; its "/" template must not fire, so the
			// built-in rules produce the (empty) text of the document.
			"imported module excluded by root use-when",
			`<xsl:import href="imp_excluded.xsl"/>`,
			"<doc/>", nil, "",
		},
		{
			// Same name, same kind, identical values: consistent; the
			// principal (higher precedence) value is used.
			"consistent redeclaration",
			`<xsl:import href="imp_v1.xsl"/>
			 <xsl:variable name="v" static="yes" select="1"/>
			 <xsl:template match="/"><xsl:value-of select="$v"/></xsl:template>`,
			"<doc/>", nil, "1",
		},
		{
			// Both declarations are static PARAMETERS with an externally
			// supplied value: the consistency rule compares values only for
			// initialized variables, so this is consistent and the supplied
			// value is used.
			"externally supplied param in two modules",
			`<xsl:import href="imp_pv1.xsl"/>
			 <xsl:param name="v" static="yes" select="1"/>
			 <xsl:template match="/"><xsl:value-of select="$v"/></xsl:template>`,
			"<doc/>", map[string]string{"v": "5"}, "5",
		},
		{
			// A shadow use-when on an included module's root, from a static
			// parameter declared before the include: true keeps the module.
			"shadow root use-when true",
			`<xsl:param name="INC" static="yes" select="'true()'"/>
			 <xsl:include href="inc_shadow_usewhen.xsl"/>
			 <xsl:template match="/" priority="-5">PRINCIPAL</xsl:template>`,
			"<doc/>", nil, "INCLUDED",
		},
		{
			// ... and false excludes all of its children.
			"shadow root use-when false",
			`<xsl:param name="INC" static="yes" select="'false()'"/>
			 <xsl:include href="inc_shadow_usewhen.xsl"/>
			 <xsl:template match="/" priority="-5">PRINCIPAL</xsl:template>`,
			"<doc/>", nil, "PRINCIPAL",
		},
		{
			// The §3.13.2 example: <xsl:include _href="common{$VERSION}.xsl"/>.
			"shadow href picks the module",
			`<xsl:param name="which" static="yes" select="'a'"/>
			 <xsl:include _href="sel_{$which}.xsl"/>`,
			"<doc/>", nil, "A",
		},
		{
			"shadow href with supplied param",
			`<xsl:param name="which" static="yes" select="'a'"/>
			 <xsl:include _href="sel_{$which}.xsl"/>`,
			"<doc/>", map[string]string{"which": "'b'"}, "B",
		},
		{
			// An excluded xsl:include is not processed at all, so a missing
			// target is no error.
			"excluded include of missing module",
			`<xsl:include href="no-such-module.xsl" use-when="false()"/>
			 <xsl:template match="/">ok</xsl:template>`,
			"<doc/>", nil, "ok",
		},
	}
	for _, c := range cases {
		sheet := swBSheet(c.decls)
		got, err := swBRunMulti(t, sheet, c.src, c.static)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestStaticAcrossModulesErrors(t *testing.T) {
	cases := []struct {
		name, decls, code string
	}{
		// §9.6 [ERR XTSE3450]: the higher-precedence declaration comes later
		// in tree order and its value differs.
		{"different value", `<xsl:import href="imp_v1.xsl"/>
			<xsl:variable name="v" static="yes" select="2"/>
			<xsl:template match="/">x</xsl:template>`, "XTSE3450"},
		// Two sibling imports: the second has the higher precedence and
		// follows the first in tree order.
		{"sibling imports differ", `<xsl:import href="imp_v1.xsl"/>
			<xsl:import href="imp_v2.xsl"/>
			<xsl:template match="/">x</xsl:template>`, "XTSE3450"},
		// Consistency also requires the same kind: xsl:variable vs xsl:param.
		{"variable vs param", `<xsl:import href="imp_v1.xsl"/>
			<xsl:param name="v" static="yes" select="1"/>
			<xsl:template match="/">x</xsl:template>`, "XTSE3450"},
	}
	for _, c := range cases {
		_, err := swBRunMulti(t, swBSheet(c.decls), "<doc/>", nil)
		swBWantErr(t, c.name, err, c.code)
	}

	// Errors without an attached code in the engine (see
	// TestUseWhenStaticErrorsUncoded).
	uncoded := []struct {
		name, decls, code string
	}{
		// An excluded module contributes no static variables.
		{"static of excluded module", `<xsl:import href="imp_excluded.xsl"/>
			<xsl:template match="/" use-when="$excl">x</xsl:template>`, "XPST0008"},
		// [ERR XTSE0165]: the resource named by xsl:include cannot be
		// retrieved.
		{"missing module", `<xsl:include href="no-such-module.xsl"/>
			<xsl:template match="/">x</xsl:template>`, "XTSE0165"},
	}
	for _, c := range uncoded {
		_, err := swBRunMulti(t, swBSheet(c.decls), "<doc/>", nil)
		swBWantErrNoOtherCode(t, c.name, err, c.code)
	}
}

// TestStaticDeclarationsInMultiModule exercises the per-declaration rules of
// §9.6 in a stylesheet that has secondary modules (the cross-module tree
// order pass sees every declaration first).
func TestStaticDeclarationsInMultiModule(t *testing.T) {
	cases := []struct {
		name, decls string
		static      map[string]string
		want        string
	}{
		{
			// A declaration's own use-when: false excludes it.
			"use-when on declarations",
			`<xsl:include href="inc_static.xsl"/>
			 <my:data>user-defined data element, ignored</my:data>
			 <xsl:variable name="w" static="yes" select="1" use-when="false()"/>
			 <xsl:variable name="w" static="yes" select="2" use-when="$inc-v = 'from-inc'"/>
			 <xsl:template match="/"><xsl:value-of select="$w"/></xsl:template>`,
			nil, "2",
		},
		{
			// No select, empty content: the zero-length string (§9.3).
			"param without select",
			`<xsl:include href="inc_static.xsl"/>
			 <xsl:param name="e" static="yes"/>
			 <xsl:template match="/"><xsl:value-of _select="'[{$e}]'"/></xsl:template>`,
			nil, "[]",
		},
		{
			"supplied value",
			`<xsl:include href="inc_static.xsl"/>
			 <xsl:param name="e" static="yes" select="'d'"/>
			 <xsl:template match="/"><xsl:value-of select="$e"/></xsl:template>`,
			map[string]string{"e": "'h'"}, "h",
		},
	}
	for _, c := range cases {
		got, err := swBRunMulti(t, swBSheet(c.decls), "<doc/>", c.static)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}

	// §9.7: a dynamic error in any static expression is a static error that
	// keeps its code — wherever in the module graph it occurs.
	errs := []struct {
		name, decls string
		static      map[string]string
	}{
		{"use-when on declaration", `<xsl:include href="inc_static.xsl"/>
			<xsl:variable name="w" static="yes" select="1" use-when="1 idiv 0"/>`, nil},
		{"select", `<xsl:include href="inc_static.xsl"/>
			<xsl:variable name="w" static="yes" select="1 idiv 0"/>`, nil},
		{"supplied value", `<xsl:include href="inc_static.xsl"/>
			<xsl:param name="w" static="yes" select="1"/>`, map[string]string{"w": "1 idiv 0"}},
		{"select in imported module", `<xsl:import href="imp_badselect.xsl"/>`, nil},
		{"select two includes deep", `<xsl:include href="inc_nested.xsl"/>`, nil},
		{"root use-when of imported module", `<xsl:import href="imp_badroot.xsl"/>`, nil},
		{"shadow on root of included module", `<xsl:include href="inc_badshadow.xsl"/>`, nil},
		{"shadow on declaration of included module", `<xsl:include href="inc_badchild.xsl"/>`, nil},
		{"shadow on principal declaration", `<xsl:include href="inc_static.xsl"/>
			<xsl:template match="/" _mode="{1 idiv 0}">x</xsl:template>`, nil},
	}
	for _, c := range errs {
		_, err := swBRunMulti(t, swBSheet(c.decls), "<doc/>", c.static)
		swBWantErr(t, c.name, err, "FOAR0001")
	}
}
