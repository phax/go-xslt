package xslt

import (
	"strings"
	"testing"
)

// Tests for conditional element inclusion ([xsl:]use-when, XSLT 3.0
// §3.13.1), shadow attributes (§3.13.2), static variables/parameters (§9.6)
// and the restricted context static expressions are evaluated in (§9.7),
// all within a single stylesheet module. Multi-module behaviour (stylesheet
// tree order, import precedence, XTSE3450) is in staticdecl_test.go.

const swBHead = `<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:my="urn:my">
<xsl:output method="text"/>
`

// swBSheet wraps declarations in a 3.0 stylesheet with text output.
func swBSheet(decls string) string { return swBHead + decls + "\n</xsl:stylesheet>" }

// swBRun compiles sheet (optionally with host static parameters) and runs it
// against a trivial source document.
func swBRun(t *testing.T, sheet string, static map[string]string) (string, error) {
	t.Helper()
	var ss *Stylesheet
	var err error
	if static != nil {
		ss, err = CompileAtWithStatic(sheet, "", static)
	} else {
		ss, err = Compile(sheet)
	}
	if err != nil {
		return "", err
	}
	out, _, err := ss.Transform("<doc><item/></doc>", nil)
	return out, err
}

// swBWantErrNoOtherCode asserts that err is non-nil and carries either the
// given code or no W3C code at all (see TestUseWhenStaticErrorsUncoded).
func swBWantErrNoOtherCode(t *testing.T, name string, err error, code string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: no error, want %s", name, code)
		return
	}
	msg := err.Error()
	if i := strings.Index(msg, "err:"); i >= 0 && !strings.Contains(msg, code) {
		t.Errorf("%s: error %q carries a code other than %s", name, msg, code)
	}
}

// swBWantErr asserts that err carries the given W3C error code.
func swBWantErr(t *testing.T, name string, err error, code string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: no error, want %s", name, code)
		return
	}
	if !strings.Contains(err.Error(), code) {
		t.Errorf("%s: error %q, want code %s", name, err, code)
	}
}

func TestUseWhenInclusion(t *testing.T) {
	cases := []struct {
		name, decls, want string
	}{
		{
			// §3.13.1: an element whose use-when is false is excluded with
			// its subtree; the remaining template rule handles "/".
			"template excluded",
			`<xsl:template match="/" priority="2" use-when="false()">X</xsl:template>
			 <xsl:template match="/">Y</xsl:template>`,
			"Y",
		},
		{
			"template kept",
			`<xsl:template match="/" priority="2" use-when="true()">X</xsl:template>
			 <xsl:template match="/">Y</xsl:template>`,
			"X",
		},
		{
			// The effective boolean value decides: a non-empty string is true,
			// the empty sequence false.
			"effective boolean value",
			`<xsl:template match="/"><xsl:text use-when="'a'">1</xsl:text><xsl:text use-when="()">2</xsl:text></xsl:template>`,
			"1",
		},
		{
			// xsl:use-when on a literal result element.
			"literal result element",
			`<xsl:template match="/"><a xsl:use-when="false()">X</a><b xsl:use-when="true()">Z</b></xsl:template>`,
			"Z",
		},
		{
			// "no static or dynamic errors will be reported in respect of the
			// element and its contents": an unknown XSLT instruction inside an
			// excluded element is never looked at.
			"errors inside excluded element",
			`<xsl:template match="/" priority="2" use-when="false()"><xsl:no-such-instruction/><xsl:value-of select="(("/></xsl:template>
			 <xsl:template match="/">ok</xsl:template>`,
			"ok",
		},
		{
			// Nested exclusion inside a kept element.
			"nested",
			`<xsl:template match="/"><xsl:if test="true()">a<xsl:if test="true()" use-when="false()">b</xsl:if>c</xsl:if></xsl:template>`,
			"ac",
		},
		{
			// A static parameter is in scope for a later use-when (§9.6).
			"static param in use-when",
			`<xsl:param name="debug" static="yes" select="true()"/>
			 <xsl:template match="/"><xsl:text use-when="$debug">D</xsl:text><xsl:text use-when="not($debug)">N</xsl:text></xsl:template>`,
			"D",
		},
		{
			// A prefixed static variable name resolves through the in-scope
			// namespaces of the referring element.
			"prefixed static variable",
			`<xsl:variable name="my:x" static="yes" select="3"/>
			 <xsl:template match="/"><xsl:text use-when="$my:x = 3">three</xsl:text></xsl:template>`,
			"three",
		},
		{
			// A later static variable sees an earlier one; both are also
			// ordinary global variables at run time (§9.6).
			"chained statics, runtime value",
			`<xsl:variable name="a" static="yes" select="2"/>
			 <xsl:variable name="b" static="yes" select="$a * 10"/>
			 <xsl:template match="/"><xsl:value-of select="$b"/></xsl:template>`,
			"20",
		},
		{
			// A static declaration's own use-when: false excludes it, the
			// later declaration of the same name is the one that counts.
			"use-when on static declaration",
			`<xsl:variable name="v" static="yes" select="'one'" use-when="false()"/>
			 <xsl:variable name="v" static="yes" select="'two'" use-when="true()"/>
			 <xsl:template match="/"><xsl:value-of select="$v"/></xsl:template>`,
			"two",
		},
		{
			// The as attribute applies the function conversion rules to the
			// static value: xs:integer is promoted to xs:double.
			"as conversion",
			`<xsl:variable name="d" static="yes" as="xs:double" select="1"/>
			 <xsl:template match="/"><xsl:value-of select="$d instance of xs:double"/></xsl:template>`,
			"true",
		},
		{
			// A static param with no select and empty content defaults to the
			// zero-length string (§9.3), also inside a shadow attribute.
			"param without select defaults to empty string",
			`<xsl:param name="p" static="yes"/>
			 <xsl:template match="/"><xsl:value-of _select="'[{$p}]'" select="'fallback'"/></xsl:template>`,
			"[]",
		},
	}
	for _, c := range cases {
		got, err := swBRun(t, swBSheet(c.decls), nil)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// TestUseWhenStaticFunctions checks the functions §9.7 puts in the static
// context, and their answers, from inside use-when expressions. Each case is
// a boolean expression that must be true; it selects "T" over "F".
func TestUseWhenStaticFunctions(t *testing.T) {
	exprs := []string{
		// system-property (§20.4.3): xsl:version is "3.0" for this processor;
		// a property in a non-XSLT namespace that the processor does not
		// define is the zero-length string.
		`system-property('xsl:version') = '3.0'`,
		`system-property('Q{http://www.w3.org/1999/XSL/Transform}version') = '3.0'`,
		`system-property('Q{urn:not-xslt}version') = ''`,
		`system-property('my:anything') = ''`,
		`system-property('xsl:supports-dynamic-evaluation') = ('yes', 'no')`,
		// element-available: implemented XSLT instructions are available,
		// unknown names in the XSLT namespace and foreign names are not.
		`element-available('xsl:for-each')`,
		`element-available('Q{http://www.w3.org/1999/XSL/Transform}iterate')`,
		`not(element-available('xsl:no-such-instruction'))`,
		`not(element-available('my:extension'))`,
		// function-available: F&O functions (with and without arity) and the
		// XSLT functions of the static context are available; the
		// XSLT-defined functions that need a dynamic XSLT context (current,
		// key) and stylesheet functions are not (§9.7, "Note that stylesheet
		// functions are not included in the context").
		`function-available('concat')`,
		`function-available('fn:abs', 1)`,
		`not(function-available('fn:abs', 2))`,
		`function-available('system-property')`,
		`function-available('element-available', 1)`,
		`not(function-available('current'))`,
		`not(function-available('key'))`,
		`not(function-available('my:f'))`,
		`not(function-available('no-such-function'))`,
		// type-available: only built-in types (no xsl:import-schema is seen).
		`type-available('xs:integer')`,
		`type-available('xs:untypedAtomic')`,
		`not(type-available('xs:no-such-type'))`,
		`not(type-available('my:t'))`,
		// available-system-properties includes the mandatory XSLT ones.
		`QName('http://www.w3.org/1999/XSL/Transform', 'version') = available-system-properties()`,
		`QName('http://www.w3.org/1999/XSL/Transform', 'vendor') = available-system-properties()`,
		// The F&O library is there.
		`string-join(('a','b'), '-') = 'a-b' and math:pi() gt 3`,
		`map:size(map{1:2}) = 1 and array:size([1,2]) = 2`,
	}
	for _, e := range exprs {
		sheet := swBSheet(`
<xsl:function name="my:f"><xsl:sequence select="1"/></xsl:function>
<xsl:template match="/"><xsl:text use-when="` + e + `">T</xsl:text><xsl:text use-when="not(` + e + `)">F</xsl:text></xsl:template>`)
		sheet = strings.Replace(sheet, `xmlns:my="urn:my"`, `xmlns:my="urn:my" xmlns:math="http://www.w3.org/2005/xpath-functions/math" xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:array="http://www.w3.org/2005/xpath-functions/array" xmlns:fn="http://www.w3.org/2005/xpath-functions"`, 1)
		got, err := swBRun(t, sheet, nil)
		if err != nil {
			t.Errorf("%s: %v", e, err)
			continue
		}
		if got != "T" {
			t.Errorf("use-when=%q: got %q, want T", e, got)
		}
	}
}

// TestUseWhenStaticErrors checks that errors in static expressions are
// reported (§9.7: a dynamic error is a static error that keeps its code).
func TestUseWhenStaticErrors(t *testing.T) {
	cases := []struct {
		name, decls, code string
	}{
		// §9.7: the context item is absent.
		{"context item", `<xsl:template match="/" use-when=".">x</xsl:template>`, "XPDY0002"},
		{"position", `<xsl:template match="/" use-when="position() = 1">x</xsl:template>`, "XPDY0002"},
		// Dynamic errors keep their own code.
		{"dynamic error in use-when", `<xsl:template match="/" use-when="1 idiv 0">x</xsl:template>`, "FOAR0001"},
		{"dynamic error in static select", `<xsl:variable name="v" static="yes" select="1 idiv 0"/>
			<xsl:template match="/">x</xsl:template>`, "FOAR0001"},
		// §9.6: static="yes" requires empty content.
		{"static param with content", `<xsl:param name="p" static="yes">text</xsl:param>
			<xsl:template match="/">x</xsl:template>`, "XTSE0010"},
		{"static variable with element content", `<xsl:variable name="v" static="yes"><xsl:sequence select="1"/></xsl:variable>
			<xsl:template match="/">x</xsl:template>`, "XTSE0010"},
		// Invalid names passed to the static-context XSLT functions.
		{"system-property bad name", `<xsl:template match="/" use-when="system-property('1bad')">x</xsl:template>`, "XTDE1390"},
		{"function-available bad name", `<xsl:template match="/" use-when="function-available('1bad')">x</xsl:template>`, "XTDE1400"},
		{"function-available unbound prefix", `<xsl:template match="/" use-when="function-available('nope:f')">x</xsl:template>`, "XTDE1400"},
		{"element-available bad name", `<xsl:template match="/" use-when="element-available('1bad')">x</xsl:template>`, "XTDE1440"},
		// fn:transform's argument must be a single map (function conversion
		// rules, XPath 3.1 §3.1.5.2).
		{"transform non-map", `<xsl:variable name="r" static="yes" select="transform(1)"/>
			<xsl:template match="/">x</xsl:template>`, "XPTY0004"},
	}
	for _, c := range cases {
		_, err := swBRun(t, swBSheet(c.decls), nil)
		swBWantErr(t, c.name, err, c.code)
	}
}

// TestUseWhenStaticErrorsUncoded covers static errors whose W3C code the
// engine does not (yet) attach to the message: the error must be raised, and
// it must not carry a DIFFERENT code. See the bug list of this test campaign
// (undefined variable / unknown function / syntax errors from internal/xpath
// carry no err: code).
func TestUseWhenStaticErrorsUncoded(t *testing.T) {
	cases := []struct {
		name, decls, code string
	}{
		// Only static variables are in scope; an ordinary global is not.
		{"non-static global", `<xsl:variable name="g" select="1"/>
			<xsl:template match="/" use-when="$g">x</xsl:template>`, "XPST0008"},
		// A static variable is not in scope within its own declaration, nor
		// before it in tree order.
		{"self reference", `<xsl:param name="p" static="yes" select="1" use-when="$p"/>
			<xsl:template match="/">x</xsl:template>`, "XPST0008"},
		{"forward reference in select", `<xsl:variable name="a" static="yes" select="$b"/>
			<xsl:variable name="b" static="yes" select="1"/>
			<xsl:template match="/">x</xsl:template>`, "XPST0008"},
		{"excluded declaration", `<xsl:variable name="v" static="yes" select="1" use-when="false()"/>
			<xsl:template match="/" use-when="$v">x</xsl:template>`, "XPST0008"},
		// Stylesheet functions are not in the static context.
		{"stylesheet function", `<xsl:function name="my:f"><xsl:sequence select="true()"/></xsl:function>
			<xsl:template match="/" use-when="my:f()">x</xsl:template>`, "XPST0017"},
		{"syntax error", `<xsl:template match="/" use-when="1 +">x</xsl:template>`, "XPST0003"},
	}
	for _, c := range cases {
		_, err := swBRun(t, swBSheet(c.decls), nil)
		swBWantErrNoOtherCode(t, c.name, err, c.code)
	}
}

// TestStaticHostParams checks externally supplied static parameter values
// (§9.7 dynamic context: "For static parameters where an external value is
// supplied: the externally-supplied value of the parameter").
func TestStaticHostParams(t *testing.T) {
	cases := []struct {
		name, decls string
		static      map[string]string
		want        string
	}{
		{
			"host value replaces default, seen by use-when and at run time",
			`<xsl:param name="mode" static="yes" select="'default'"/>
			 <xsl:template match="/"><xsl:text use-when="$mode = 'host'">H:</xsl:text><xsl:value-of select="$mode"/></xsl:template>`,
			map[string]string{"mode": "'host'"},
			"H:host",
		},
		{
			"prefixed name resolved against the declaration",
			`<xsl:param name="my:p" static="yes" select="0"/>
			 <xsl:template match="/"><xsl:value-of select="$my:p"/></xsl:template>`,
			map[string]string{"my:p": "1 + 1"},
			"2",
		},
		{
			"EQName spelling",
			`<xsl:param name="my:p" static="yes" select="0"/>
			 <xsl:template match="/"><xsl:value-of select="$my:p"/></xsl:template>`,
			map[string]string{"Q{urn:my}p": "7"},
			"7",
		},
		{
			// Only parameters are externally settable; a static variable keeps
			// its own value.
			"static variable is not settable",
			`<xsl:variable name="v" static="yes" select="'decl'"/>
			 <xsl:template match="/"><xsl:value-of select="$v"/></xsl:template>`,
			map[string]string{"v": "'host'"},
			"decl",
		},
		{
			// A supplied value is converted to the declared type with the
			// function conversion rules (xs:integer is an xs:decimal).
			"supplied value converted to as",
			`<xsl:param name="n" static="yes" as="xs:decimal" select="0.5"/>
			 <xsl:template match="/"><xsl:value-of select="$n instance of xs:decimal, $n"/></xsl:template>`,
			map[string]string{"n": "3"},
			"true 3",
		},
		{
			// The supplied value reaches the shadow attribute: the §3.13.2
			// "parameterize selection" example.
			"host value in shadow attribute",
			`<xsl:param name="filter" static="yes" as="xs:string" select="'true()'"/>
			 <xsl:template match="/"><xsl:value-of _select="count(//*[{$filter}])"/></xsl:template>`,
			map[string]string{"filter": "'self::item'"},
			"1",
		},
		{
			// Unknown host names are ignored.
			"undeclared name ignored",
			`<xsl:template match="/">ok</xsl:template>`,
			map[string]string{"nobody": "1"},
			"ok",
		},
	}
	for _, c := range cases {
		got, err := swBRun(t, swBSheet(c.decls), c.static)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}

	errs := []struct {
		name, decls string
		static      map[string]string
		code        string
	}{
		// §9.2.1: the conversion of a supplied parameter value to its
		// required type failing is XTTE0590.
		{"type mismatch", `<xsl:param name="n" static="yes" as="xs:integer" select="1"/><xsl:template match="/">x</xsl:template>`,
			map[string]string{"n": "'abc'"}, "XTTE0590"},
		{"supplied expression fails", `<xsl:param name="n" static="yes" select="1"/><xsl:template match="/">x</xsl:template>`,
			map[string]string{"n": "1 idiv 0"}, "FOAR0001"},
		// XTSE0010 is a property of the declaration: a supplied value does
		// not excuse non-empty content.
		{"content despite host value", `<xsl:param name="n" static="yes">1</xsl:param><xsl:template match="/">x</xsl:template>`,
			map[string]string{"n": "1"}, "XTSE0010"},
	}
	for _, c := range errs {
		_, err := swBRun(t, swBSheet(c.decls), c.static)
		swBWantErr(t, c.name, err, c.code)
	}
}

// TestShadowAttributes checks §3.13.2.
func TestShadowAttributes(t *testing.T) {
	cases := []struct {
		name, decls, want string
	}{
		{
			// The shadow value template supplies the expression text.
			"select from static param",
			`<xsl:param name="expr" static="yes" select="'1+2'"/>
			 <xsl:template match="/"><xsl:value-of _select="{$expr}"/></xsl:template>`,
			"3",
		},
		{
			// "If a shadow attribute is present, then any attribute node with
			// name N ... is ignored" — even an invalid one.
			"non-shadow attribute ignored, even if invalid",
			`<xsl:template match="/"><xsl:value-of select="((" _select="'ok'"/></xsl:template>`,
			"ok",
		},
		{
			// A shadow attribute without curly braces is a plain value.
			"constant shadow",
			`<xsl:template match="/"><xsl:value-of _select="'lit'"/></xsl:template>`,
			"lit",
		},
		{
			// A sequence in an AVT part is joined with single spaces.
			"sequence joined",
			`<xsl:template match="/"><xsl:value-of _select="'{(1, 2, 3)}'"/></xsl:template>`,
			"1 2 3",
		},
		{
			// _use-when is itself a shadow attribute (use-when has static
			// significance and is explicitly covered).
			"shadow use-when",
			`<xsl:param name="on" static="yes" select="'false()'"/>
			 <xsl:template match="/"><xsl:text _use-when="{$on}">X</xsl:text>Y</xsl:template>`,
			"Y",
		},
		{
			// _static on xsl:variable (explicitly named in §3.13.2).
			"shadow static",
			`<xsl:param name="mk" static="yes" select="'yes'"/>
			 <xsl:variable name="v" _static="{$mk}" select="true()"/>
			 <xsl:template match="/"><xsl:text use-when="$v">S</xsl:text></xsl:template>`,
			"S",
		},
		{
			// An excluded element's shadow attributes are never evaluated
			// (no errors are reported in respect of an excluded element).
			"shadow inside excluded element",
			`<xsl:template match="/"><xsl:if test="true()" use-when="false()"><xsl:value-of _select="{1 idiv 0}"/></xsl:if>ok</xsl:template>`,
			"ok",
		},
		{
			// On a literal result element an underscore attribute is an
			// ordinary attribute copied to the result.
			"literal result element keeps underscore attribute",
			`<xsl:variable name="x" as="element()"><out _one="1"/></xsl:variable>
			 <xsl:template match="/"><xsl:value-of select="name($x/@*), string($x/@*)"/></xsl:template>`,
			"_one 1",
		},
	}
	for _, c := range cases {
		got, err := swBRun(t, swBSheet(c.decls), nil)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}

	errs := []struct {
		name, decls, code string
	}{
		{"dynamic error in shadow", `<xsl:template match="/"><xsl:value-of _select="{1 idiv 0}"/></xsl:template>`, "FOAR0001"},
		{"context item in shadow", `<xsl:template match="/"><xsl:value-of _select="{.}"/></xsl:template>`, "XPDY0002"},
	}
	for _, c := range errs {
		_, err := swBRun(t, swBSheet(c.decls), nil)
		swBWantErr(t, c.name, err, c.code)
	}
	// Only static variables are in scope (§9.7); XPST0008 is not attached
	// by the engine (bug list), so only "an error, no other code" is checked.
	_, err := swBRun(t, swBSheet(`<xsl:variable name="g" select="'1'"/>
			<xsl:template match="/"><xsl:value-of _select="{$g}"/></xsl:template>`), nil)
	swBWantErrNoOtherCode(t, "non-static variable in shadow", err, "XPST0008")

	// The principal module's root element is expanded before any of its
	// declarations: a static parameter declared inside is not yet in scope
	// there (§3.13.2's xpath-default-namespace example says exactly this).
	rootShadow := `<xsl:stylesheet _version="{$v}" version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:param name="v" static="yes" select="'3.0'"/>
<xsl:template match="/">x</xsl:template></xsl:stylesheet>`
	_, err = swBRun(t, rootShadow, nil)
	swBWantErrNoOtherCode(t, "root shadow sees no inner static", err, "XPST0008")
}

// TestUseWhenStaticTransform: fn:transform is an F&O 3.1 function, so it is
// in the static context (§9.7) and a static variable may run one.
func TestUseWhenStaticTransform(t *testing.T) {
	sheet := swBSheet(`
<xsl:variable name="inner" static="yes" select="'&lt;xsl:stylesheet version=&quot;3.0&quot; xmlns:xsl=&quot;http://www.w3.org/1999/XSL/Transform&quot;&gt;&lt;xsl:template name=&quot;main&quot;&gt;hi&lt;/xsl:template&gt;&lt;/xsl:stylesheet&gt;'"/>
<xsl:variable name="r" static="yes" select="string(transform(map{'stylesheet-text': $inner, 'initial-template': QName('', 'main')})?output)"/>
<xsl:template match="/"><xsl:text use-when="$r = 'hi'">yes</xsl:text><xsl:text use-when="$r != 'hi'">no</xsl:text></xsl:template>`)
	got, err := swBRun(t, sheet, nil)
	if err != nil {
		t.Fatalf("compile/run: %v", err)
	}
	if got != "yes" {
		t.Errorf("got %q, want yes", got)
	}
}
