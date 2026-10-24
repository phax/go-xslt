package xpath

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// tySchema is a small in-memory stand-in for a host's compiled schema,
// implementing every optional hook internal/xpath consults. Its components:
//
//	my:hatsize    xs:integer restricted to 1..20
//	my:code       xs:NCName restriction
//	my:codes      list of my:code
//	my:numOrDate  union(my:hatsize, xs:date)
//	my:dateOrTime union(xs:date, xs:time)
//	my:qn         xs:QName restriction
//	my:nota       xs:NOTATION restriction
//	my:integer    xs:integer restriction (shares a built-in's local name)
//	my:Base, my:Sub (complex; Sub derives from Base)
//	element my:item (type my:Base), element my:special (type my:Sub,
//	substitution group my:item), attribute my:code (type my:code)
type tySchema struct{}

const tyNS = "urn:ty"

func tyT(local string) xmltree.SchemaTypeName {
	return xmltree.SchemaTypeName{Namespace: tyNS, Local: local, Complex: local == "Base" || local == "Sub"}
}

func tyXS(local string) xmltree.SchemaTypeName {
	return xmltree.SchemaTypeName{Namespace: nsXS, Local: local}
}

var tyBases = map[xmltree.SchemaTypeName][]xmltree.SchemaTypeName{
	tyT("hatsize"): {tyXS("integer")},
	tyT("integer"): {tyXS("integer")},
	tyT("code"):    {tyXS("NCName")},
	tyT("Sub"):     {tyT("Base")},
	// union membership
	tyXS("date"): {tyT("numOrDate"), tyT("dateOrTime")},
	tyXS("time"): {tyT("dateOrTime")},
}

func init() {
	tyBases[tyT("hatsize")] = append(tyBases[tyT("hatsize")], tyT("numOrDate"))
}

func tyLocal(lexical string) (string, bool) {
	if strings.HasPrefix(lexical, "my:") {
		return strings.TrimPrefix(lexical, "my:"), true
	}
	if strings.HasPrefix(lexical, "Q{"+tyNS+"}") {
		return strings.TrimPrefix(lexical, "Q{"+tyNS+"}"), true
	}
	return "", false
}

func (tySchema) LookupSchemaType(lexical string) (xmltree.SchemaTypeName, bool) {
	l, ok := tyLocal(lexical)
	if !ok {
		return xmltree.SchemaTypeName{}, false
	}
	switch l {
	case "hatsize", "code", "codes", "numOrDate", "dateOrTime", "qn", "nota", "integer", "Base", "Sub":
		return tyT(l), true
	}
	return xmltree.SchemaTypeName{}, false
}

func (tySchema) LookupSchemaElement(lexical string) (xmltree.Name, xmltree.SchemaTypeName, bool) {
	l, ok := tyLocal(lexical)
	if !ok {
		return xmltree.Name{}, xmltree.SchemaTypeName{}, false
	}
	switch l {
	case "item":
		return xmltree.Name{Space: tyNS, Local: l}, tyT("Base"), true
	case "special":
		return xmltree.Name{Space: tyNS, Local: l}, tyT("Sub"), true
	}
	return xmltree.Name{}, xmltree.SchemaTypeName{}, false
}

func (tySchema) LookupSchemaAttribute(lexical string) (xmltree.Name, xmltree.SchemaTypeName, bool) {
	if l, ok := tyLocal(lexical); ok && l == "code" {
		return xmltree.Name{Space: tyNS, Local: l}, tyT("code"), true
	}
	return xmltree.Name{}, xmltree.SchemaTypeName{}, false
}

func (s tySchema) DerivesFrom(got, want xmltree.SchemaTypeName) bool {
	if got == want {
		return true
	}
	for _, b := range tyBases[got] {
		if s.DerivesFrom(b, want) {
			return true
		}
	}
	return false
}

func (tySchema) SubstitutesFor(member, head xmltree.Name) bool {
	return member.Space == tyNS && member.Local == "special" && head.Space == tyNS && head.Local == "item"
}

func (tySchema) ElementDeclType(name xmltree.Name) (xmltree.SchemaTypeName, bool) {
	if name.Space != tyNS {
		return xmltree.SchemaTypeName{}, false
	}
	switch name.Local {
	case "item":
		return tyT("Base"), true
	case "special":
		return tyT("Sub"), true
	}
	return xmltree.SchemaTypeName{}, false
}

func tyHatsizeOK(lexical string) error {
	a, err := CastTo(NewString(lexical), XSinteger)
	if err != nil {
		return err
	}
	if n := a.Float(); n < 1 || n > 20 {
		return fmt.Errorf("err:FORG0001: %s is outside my:hatsize", lexical)
	}
	return nil
}

func (tySchema) CastToSchemaType(t xmltree.SchemaTypeName, lexical string) (AtomType, error) {
	switch t.Local {
	case "hatsize":
		return XSinteger, tyHatsizeOK(lexical)
	case "integer":
		_, err := CastTo(NewString(lexical), XSinteger)
		return XSinteger, err
	case "code":
		_, err := CastTo(NewString(lexical), XSncname)
		return XSncname, err
	case "numOrDate":
		if tyHatsizeOK(lexical) == nil {
			return XSinteger, nil
		}
		_, err := CastTo(NewString(lexical), XSdate)
		return XSdate, err
	case "dateOrTime":
		if _, err := CastTo(NewString(lexical), XSdate); err == nil {
			return XSdate, nil
		}
		_, err := CastTo(NewString(lexical), XStime)
		return XStime, err
	case "qn":
		return XSqname, nil
	case "nota":
		return XSnotation, nil
	}
	return 0, fmt.Errorf("err:XPST0051: %s is not a simple type", t.Local)
}

func (tySchema) CastToSchemaListType(t xmltree.SchemaTypeName, lexical string, _ *xmltree.Node) (AtomType, *xmltree.SchemaTypeName, bool, error) {
	if t.Local != "codes" {
		return 0, nil, false, nil
	}
	for _, tok := range strings.Fields(lexical) {
		if _, err := CastTo(NewString(tok), XSncname); err != nil {
			return 0, nil, true, err
		}
	}
	code := tyT("code")
	return XSncname, &code, true, nil
}

func (tySchema) UnionCastMember(t xmltree.SchemaTypeName, lexical string, _ *xmltree.Node) (xmltree.SchemaTypeName, bool, bool) {
	switch t.Local {
	case "numOrDate":
		if tyHatsizeOK(lexical) == nil {
			return tyT("hatsize"), false, true
		}
		return tyXS("date"), true, true
	case "dateOrTime":
		if _, err := CastTo(NewString(lexical), XSdate); err == nil {
			return tyXS("date"), true, true
		}
		return tyXS("time"), true, true
	}
	return xmltree.SchemaTypeName{}, false, false
}

// tySchemaRun parses expr schema-aware against tySchema and evaluates it with
// the schema hooks and the my → urn:ty binding in the context.
func tySchemaRun(expr string, ctxNode *xmltree.Node) string {
	p, err := ParseSchemaAware(expr, tySchema{})
	if err != nil {
		return "!" + tyErrCode(err)
	}
	if ctxNode == nil {
		ctxNode, _ = xmltree.Parse("<r/>")
	}
	ctx := &Context{Node: ctxNode, Pos: 1, Size: 1, SchemaTypes: tySchema{}, NS: testNSMap{"my": tyNS}}
	v, err := p.Eval(ctx)
	if err != nil {
		return "!" + tyErrCode(err)
	}
	return ToString(v)
}

// TestTyUserDefinedSimpleTypes covers casting to and testing against
// user-defined atomic, list and union types: F&O 3.1 §19.3 (casting to a
// derived type checks the facets, FORG0001 otherwise; to a list type casts
// each token to the item type; to a union type yields the first member type
// that accepts the value) and XPath 3.1 §2.5.5.2/§2.5.6.1 (an atomic value
// matches a type if its type annotation is or derives from it; union types
// count their member types as derived).
func TestTyUserDefinedSimpleTypes(t *testing.T) {
	cases := []tyCase{
		{`'5' cast as my:hatsize`, "5"},
		{`('5' cast as my:hatsize) instance of my:hatsize`, "true"},
		{`('5' cast as my:hatsize) instance of xs:integer`, "true"},
		{`xs:untypedAtomic(' 7 ') cast as my:hatsize`, "7"},
		{`5 instance of my:hatsize`, "false"},
		{`'50' cast as my:hatsize`, "!FORG0001"},
		{`'50' castable as my:hatsize`, "false"},
		{`'5' castable as my:hatsize`, "true"},
		{`count(() cast as my:hatsize?)`, "0"},
		{`() cast as my:hatsize`, "!XPTY0004"},
		{`(1, 2) cast as my:hatsize`, "!XPTY0004"},
		{`() castable as my:hatsize?`, "true"},
		{`() castable as my:hatsize`, "false"},
		{`(1, 2) castable as my:hatsize`, "false"},
		// A built-in local name redefined by the schema: the prefixed name is
		// the user type, not xs:integer.
		{`5 instance of my:integer`, "false"},
		{`(5 cast as my:integer) instance of my:integer`, "true"},
		{`5 instance of xs:integer`, "true"},
		// List types.
		{`count('a b c' cast as my:codes)`, "3"},
		{`('a b' cast as my:codes)[2] instance of my:code`, "true"},
		{`('a b' cast as my:codes)[1] instance of xs:NCName`, "true"},
		{`'a 1b' cast as my:codes`, "!FORG0001"},
		// Union types.
		{`('7' cast as my:numOrDate) instance of my:hatsize`, "true"},
		{`('7' cast as my:numOrDate) instance of my:numOrDate`, "true"},
		{`('2000-01-01' cast as my:numOrDate) instance of xs:date`, "true"},
		{`('2000-01-01' cast as my:numOrDate) instance of my:numOrDate`, "true"},
		{`'x' cast as my:numOrDate`, "!FORG0001"},
		{`xs:date('2000-01-01') instance of my:dateOrTime`, "true"},
		{`xs:time('10:00:00') instance of my:dateOrTime`, "true"},
		{`1 instance of my:dateOrTime`, "false"},
		{`('10:00:00' cast as my:dateOrTime) instance of xs:time`, "true"},
		// QName / NOTATION restrictions: the casting table admits only
		// string-like and QName/NOTATION sources (XPTY0004 otherwise); an
		// unbound prefix is FONS0004 (F&O 3.1 §19.2.4 / §10.2.5).
		{`1 cast as my:qn`, "!XPTY0004"},
		{`local-name-from-QName('my:x' cast as my:qn)`, "x"},
		{`string('my:n' cast as my:nota)`, "my:n"},
		{`'zz:n' cast as my:nota`, "!FONS0004"},
		{`1 cast as my:nota`, "!XPTY0004"},
		// Constructor functions for user-defined types (XPath 3.1 §3.1.5.4 /
		// F&O 3.1 §18.6) take xs:anyAtomicType? and cast.
		{`my:hatsize('9') instance of my:hatsize`, "true"},
		{`count(my:hatsize(()))`, "0"},
		{`my:hatsize((1, 2))`, "!XPTY0004"},
		{`my:hatsize('30')`, "!FORG0001"},
		// Function conversion against a user-defined type: xs:untypedAtomic
		// is cast to it; any other atomic value must already be an instance.
		{`function($x as my:hatsize) {$x instance of my:hatsize}(xs:untypedAtomic('3'))`, "true"},
		{`function($x as my:hatsize) {$x}(3)`, "!XPTY0004"},
	}
	for _, c := range cases {
		if got := tySchemaRun(c.expr, nil); got != c.want {
			t.Errorf("%s\n  got  %q\n  want %q", c.expr, got, c.want)
		}
	}
}

// TestTySchemaElementTests checks element tests resolved against the schema
// (XPath 3.1 §2.5.5.3/§2.5.5.4): element(*, T) matches by the annotation's
// derivation; schema-element(N) matches N and the members of its
// substitution group whose annotation derives from their own declared type.
func TestTySchemaElementTests(t *testing.T) {
	doc := &xmltree.Node{Kind: xmltree.KindDocument}
	root := xmltree.NewElement(xmltree.Name{Local: "root"})
	doc.Append(root)
	add := func(local string, anno *xmltree.SchemaTypeName) {
		el := xmltree.NewElement(xmltree.Name{Space: tyNS, Local: local, Prefix: "my"})
		el.SetSchemaType(anno)
		root.Append(el)
	}
	base, sub := tyT("Base"), tyT("Sub")
	add("item", &base)
	add("special", &sub)
	add("other", &base)
	add("item", nil)
	cases := []tyCase{
		{`count(*[. instance of element(*, my:Base)])`, "3"},
		{`count(*[. instance of element(*, my:Sub)])`, "1"},
		{`count(*[. instance of schema-element(my:item)])`, "2"},
		{`count(*[. instance of element(my:item)])`, "2"},
		{`count(*[. instance of schema-element(my:special)])`, "1"},
		{`count(schema-element(my:item))`, "2"},
	}
	for _, c := range cases {
		if got := tySchemaRun(c.expr, root); got != c.want {
			t.Errorf("%s\n  got  %q\n  want %q", c.expr, got, c.want)
		}
	}
}

// TestTyTypedModeRewrite covers Pattern.TypedModeRewrite (XSLT 3.0 §6.7.4):
// in a mode with typed="strict" a pattern's leading element name test is
// read as schema-element(N), and naming no global element declaration is
// XTSE3105; typed="lax" leaves an unknown name alone.
func TestTyTypedModeRewrite(t *testing.T) {
	doc := &xmltree.Node{Kind: xmltree.KindDocument}
	root := xmltree.NewElement(xmltree.Name{Local: "root"})
	doc.Append(root)
	base, sub := tyT("Base"), tyT("Sub")
	item := xmltree.NewElement(xmltree.Name{Space: tyNS, Local: "item", Prefix: "my"})
	item.SetSchemaType(&base)
	root.Append(item)
	special := xmltree.NewElement(xmltree.Name{Space: tyNS, Local: "special", Prefix: "my"})
	special.SetSchemaType(&sub)
	root.Append(special)
	untyped := xmltree.NewElement(xmltree.Name{Space: tyNS, Local: "item", Prefix: "my"})
	root.Append(untyped)
	ctx := func(n *xmltree.Node) *Context {
		return &Context{Node: n, Pos: 1, Size: 1, SchemaTypes: tySchema{}, NS: testNSMap{"my": tyNS}}
	}

	p, err := ParsePatternSchemaAware("my:item", tySchema{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.TypedModeRewrite(tySchema{}, true); err != nil {
		t.Fatalf("rewrite of a declared name: %v", err)
	}
	for _, c := range []struct {
		n    *xmltree.Node
		want bool
	}{{item, true}, {special, true}, {untyped, false}} {
		got, err := p.Match(c.n, ctx(c.n))
		if err != nil || got != c.want {
			t.Errorf("typed my:item against %s (anno %v) = %v %v, want %v", c.n.Name.Local, c.n.SchemaType(), got, err, c.want)
		}
	}

	for _, src := range []string{"my:nope", "//my:nope", "Q{urn:other}x"} {
		p, err := ParsePattern(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.TypedModeRewrite(tySchema{}, true); tyErrCode(err) != "XTSE3105" {
			t.Errorf("strict rewrite of %s: %v, want XTSE3105", src, err)
		}
		if err := p.TypedModeRewrite(tySchema{}, false); err != nil {
			t.Errorf("lax rewrite of %s: %v, want nil", src, err)
		}
	}
	// Patterns whose principal node kind is not element, wildcards and
	// non-path patterns are left alone even in strict mode.
	for _, src := range []string{"@x", "*", "my:*", "text()", "$v", "."} {
		p, err := ParsePattern(src)
		if err != nil {
			continue
		}
		if err := p.TypedModeRewrite(tySchema{}, true); err != nil {
			t.Errorf("strict rewrite of %s: %v, want nil", src, err)
		}
	}
	var nilPat *Pattern
	if err := nilPat.TypedModeRewrite(tySchema{}, true); err != nil {
		t.Errorf("nil pattern: %v", err)
	}
}

// TestTyWithheldSchema covers ParseWithheldSchema: where the host withholds
// the schema components from an expression (XSLT 3.0 §3.14.1, a static
// expression evaluated before xsl:import-schema takes effect), a prefixed
// user type name is XTDE3160, while built-in types keep working.
func TestTyWithheldSchema(t *testing.T) {
	for _, c := range []struct{ expr, code string }{
		{"1 instance of my:hatsize", "XTDE3160"},
		{"1 cast as my:hatsize", "XTDE3160"},
		{"1 instance of xs:nope", "XPST0051"},
	} {
		_, err := ParseWithheldSchema(c.expr)
		if tyErrCode(err) != c.code {
			t.Errorf("ParseWithheldSchema(%q) = %v, want %s", c.expr, err, c.code)
		}
	}
	p, err := ParseWithheldSchema("1 instance of xs:integer")
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := xmltree.Parse("<r/>")
	if v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1}); err != nil || ToString(v) != "true" {
		t.Errorf("built-in instance of under a withheld schema = %v %v", v, err)
	}
	for _, bad := range []string{"1 +", "1 2", "'x"} {
		if _, err := ParseWithheldSchema(bad); err == nil {
			t.Errorf("ParseWithheldSchema(%q) should fail", bad)
		}
	}
	// With no lookup the schema-aware entry points are the ordinary ones.
	if _, err := ParseSchemaAware("1 + 1", nil); err != nil {
		t.Errorf("ParseSchemaAware(nil lookup): %v", err)
	}
	if _, err := ParsePatternSchemaAware("a", nil); err != nil {
		t.Errorf("ParsePatternSchemaAware(nil lookup): %v", err)
	}
	if st, err := ParseSeqTypeStringSchemaAware("xs:integer", nil); err != nil || st.Item.Atom != XSinteger {
		t.Errorf("ParseSeqTypeStringSchemaAware(nil lookup) = %v %v", st, err)
	}
}
