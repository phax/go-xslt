package xpath

import (
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

const caAxesDoc = `<r a="0"><x id="1"><y id="2"/><y id="3">t</y></x><!--c--><?pi d?><?other e?><z id="4"><w id="5" xml:lang="en"/></z></r>`

// TestCaAxes checks every axis of XPath 3.1 §3.3.2.1 on a small tree, with
// reverse-axis positional predicates counted nearest-first (§3.3.2.2) and
// path results in document order (§3.3.1.1).
func TestCaAxes(t *testing.T) {
	caRun(t, caAxesDoc, nil, []caCase{
		{expr: "string-join(//y[@id='3']/ancestor::*/name(), ',')", want: "r,x"},
		{expr: "//y[@id='3']/ancestor::*[1]/name()", want: "x"},
		{expr: "//y[@id='3']/ancestor::*[last()]/name()", want: "r"},
		{expr: "count(//y[@id='2']/ancestor-or-self::node())", want: "4"},
		{expr: "string(//y[@id='3']/ancestor-or-self::*[1]/@id)", want: "3"},
		{expr: "string(//y[@id='3']/ancestor-or-self::*[2]/@id)", want: "1"},
		{expr: "count(/..)", want: "0"},
		{expr: "count(/ancestor::node())", want: "0"},
		{expr: "name(//y[@id='2']/parent::*)", want: "x"},
		{expr: "name(//@id[. = '2']/..)", want: "y"},
		{expr: "count(/r/self::r) + count(/r/self::x)", want: "1"},
		{expr: "string(/r/descendant::*[2]/@id)", want: "2"},
		{expr: "string(/r/descendant::*[last()]/@id)", want: "5"},
		{expr: "string(//x/descendant-or-self::*[1]/@id)", want: "1"},
		{expr: "string(//x/descendant-or-self::*[3]/@id)", want: "3"},
		{expr: "count(//x/descendant::node())", want: "3"},
		{expr: "string-join(//y[@id='2']/following-sibling::*/@id, ',')", want: "3"},
		{expr: "string(//y[@id='3']/preceding-sibling::*[1]/@id)", want: "2"},
		{expr: "string(//z/preceding-sibling::*[1]/@id)", want: "1"},
		{expr: "count(//z/preceding-sibling::node())", want: "4"},
		{expr: "count(//z/preceding-sibling::node()[2]/self::processing-instruction('pi'))", want: "1"},
		{expr: "count(/r/preceding-sibling::node()) + count(/following-sibling::node())", want: "0"},
		{expr: "count(//@a/following-sibling::node())", want: "0"},
		{expr: "string-join(//y[@id='2']/following::*/@id, ',')", want: "3,4,5"},
		{expr: "count(//y[@id='2']/following::node())", want: "7"},
		{expr: "string(//y[@id='2']/following::*[2]/@id)", want: "4"},
		{expr: "count(//x/following::*)", want: "2"},
		{expr: "string-join(//w/preceding::*/@id, ',')", want: "1,2,3"},
		{expr: "string(//w/preceding::*[1]/@id)", want: "3"},
		{expr: "count(//w/preceding::node())", want: "7"},
		{expr: "count(/r/preceding::node()) + count(/r/following::node())", want: "0"},
		// An attribute's following axis includes its parent's descendants;
		// its preceding axis excludes its ancestors (XDM 3.1 §2.4 order).
		{expr: "count(//x/@id/following::*)", want: "4"},
		{expr: "string(//x/@id/following::*[1]/@id)", want: "2"},
		{expr: "count(//x/@id/preceding::node())", want: "0"},
		{expr: "string(//z/@id/preceding::*[1]/@id)", want: "3"},
		// Node tests on non-principal axes.
		{expr: "count(//processing-instruction())", want: "2"},
		{expr: "string(//processing-instruction('pi'))", want: "d"},
		{expr: "count(//processing-instruction('none'))", want: "0"},
		{expr: "count(//comment()) + count(//text())", want: "2"},
		{expr: "count(//x/node())", want: "2"},
		{expr: "count(/r/attribute::node())", want: "1"},
		{expr: "count(//@*)", want: "7"},
		{expr: "count(//attribute::*:lang) + count(//@xml:lang)", want: "2"},
		{expr: "count(//*:y) + count(//Q{}y)", want: "4"},
		{expr: "count(/self::document-node(element(r)))", want: "1"},
		{expr: "count(/self::document-node(element(x)))", want: "0"},
		{expr: "count(/self::document-node())", want: "1"},
		{expr: "count(/r/self::document-node())", want: "0"},
		{expr: "count(//element(y)) + count(//element(*))", want: "8"},
		{expr: "count(//attribute(id)) + count(//attribute(*))", want: "12"},
		{expr: "count(//x/self::attribute(id)) + count(//@id/self::element(*))", want: "0"},
		{expr: "count(//self::namespace-node())", want: "0"},
	})
}

// TestCaKindTestTypes checks element(N,T)/attribute(N,T) against untyped
// nodes (XPath 3.1 §2.5.5.3/§2.5.5.5): an unvalidated element is annotated
// xs:untyped, an unvalidated attribute xs:untypedAtomic.
func TestCaKindTestTypes(t *testing.T) {
	xs := func(c *Context) { c.NS = caNS{"s": nsXS, "u": "urn:u"} }
	caRun(t, `<r><y a="1"/><y/><u:y xmlns:u="urn:u"/></r>`, xs, []caCase{
		{expr: "count(//element(y, xs:untyped))", want: "2"},
		{expr: "count(//element(y, s:untyped))", want: "2"},
		{expr: "count(//element(y, Q{http://www.w3.org/2001/XMLSchema}anyType))", want: "2"},
		{expr: "count(//element(*, xs:anyType))", want: "4"},
		{expr: "count(//element(y, xs:string))", want: "0"},
		{expr: "count(//element(y, xs:untypedAtomic))", want: "0"},
		{expr: "count(//element(y, xs:anyAtomicType))", want: "0"},
		{expr: "count(//element(u:y))", want: "1"},
		{expr: "count(//element(Q{urn:u}y))", want: "1"},
		{expr: "count(//u:*)", want: "1"},
		{expr: "count(//@attribute(a, xs:untypedAtomic))", want: "1"},
		{expr: "count(//@attribute(*, xs:anyAtomicType))", want: "1"},
		{expr: "count(//@attribute(a, xs:anySimpleType))", want: "1"},
		{expr: "count(//@attribute(a, xs:untyped))", want: "0"},
		{expr: "count(//@attribute(a, xs:integer))", want: "0"},
		{expr: "count(//@attribute(u:a))", want: "0"},
		{expr: "count(//zz:y)", code: "XPST0081"},
	})
	// xpath-default-namespace applies to unprefixed element names, not to
	// attributes and not to Q{}local.
	dns := func(c *Context) { c.DefaultElemNS = "urn:u" }
	caRun(t, `<r a="1"><u:y xmlns:u="urn:u" a="2"/></r>`, dns, []caCase{
		{expr: "count(//y) + count(//element(y))", want: "2"},
		{expr: "count(//Q{}r) + count(/r)", want: "1"},
		{expr: "count(//@a)", want: "2"},
		{expr: "count(//*)", want: "2"},
	})
}

// TestCaKindTestAnnotated checks the type part of element/attribute tests
// against nodes that carry a type annotation (XPath 3.1 §2.5.5.3: matches
// when the annotation derives from T; element(N,T) excludes nilled elements,
// element(N,T?) does not).
func TestCaKindTestAnnotated(t *testing.T) {
	doc, err := xmltree.Parse(`<r a="5" b="x y"><e>7</e><n/><v/></r>`)
	if err != nil {
		t.Fatal(err)
	}
	root := doc.Children[0]
	root.Attrs[0].SetTypeAnno(int32(XSinteger))
	root.Attrs[1].SetTypeAnno(int32(XSnmtoken))
	root.Attrs[1].SetListTyped(true)
	e, n, v := root.Children[0], root.Children[1], root.Children[2]
	e.SetTypeAnno(int32(XSinteger))
	n.SetTypeAnno(int32(XSinteger))
	n.SetNilled(true)
	v.SetSchemaType(&xmltree.SchemaTypeName{Namespace: nsXS, Local: "untypedAtomic"})
	cases := []struct{ expr, want string }{
		{"count(/r/@attribute(a, xs:integer))", "1"},
		{"count(/r/@attribute(a, xs:decimal))", "1"},
		{"count(/r/@attribute(a, xs:short))", "0"},
		{"count(/r/@attribute(a, xs:string))", "0"},
		{"count(/r/@attribute(a, xs:untypedAtomic))", "0"},
		{"count(/r/@attribute(b, xs:NMTOKENS))", "1"},
		{"count(/r/@attribute(b, xs:NMTOKEN))", "0"},
		{"count(/r/@attribute(a, xs:NMTOKENS))", "0"},
		{"count(/r/element(e, xs:integer))", "1"},
		{"count(/r/element(e, xs:untyped))", "0"},
		{"count(/r/element(n, xs:integer))", "0"},
		{"count(/r/element(n, xs:integer?))", "1"},
		{"count(/r/element(n))", "1"},
		{"count(/r/element(v, xs:untypedAtomic))", "1"},
		{"count(/r/element(v, xs:untyped))", "0"},
	}
	for _, c := range cases {
		p, err := Parse(c.expr)
		if err != nil {
			t.Errorf("parse %q: %v", c.expr, err)
			continue
		}
		got, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1})
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if ToString(got) != c.want {
			t.Errorf("%s = %s, want %s", c.expr, ToString(got), c.want)
		}
	}
}

// TestCaNamespaceAxis checks the namespace axis (XPath 3.1 §3.3.2.1, XDM 3.1
// §6.4): one node per in-scope binding including the implicit xml one, an
// xmlns="" undeclaration removes the default, and each element has its own
// namespace nodes.
func TestCaNamespaceAxis(t *testing.T) {
	src := `<r xmlns:p="urn:p"><z xmlns:q="urn:q"><w/></z><d xmlns="urn:d"><s xmlns=""/></d></r>`
	caRun(t, src, nil, []caCase{
		{expr: "count(/r/z/w/namespace::*)", want: "3"},
		{expr: "string-join(sort(for $n in /r/z/w/namespace::* return name($n)), ',')", want: "p,q,xml"},
		{expr: "string(/r/z/w/namespace::q)", want: "urn:q"},
		{expr: "string(/r/namespace::xml)", want: "http://www.w3.org/XML/1998/namespace"},
		{expr: "count(/r/namespace::node()) + count(/r/namespace::namespace-node())", want: "4"},
		{expr: "count(/namespace::*) + count(/r/text()/namespace::*)", want: "0"},
		{expr: "count(//*:d/namespace::*)", want: "3"},
		{expr: "count(//*:d/namespace::*[name() = ''])", want: "1"},
		{expr: "count(//s/namespace::*)", want: "2"},
		{expr: "count(//s/namespace::*[name() = ''])", want: "0"},
		{expr: "count(/r/z/w/namespace::* | /r/z/w/namespace::*)", want: "3"},
		{expr: "count(/r/z/namespace::* | /r/z/w/namespace::*)", want: "6"},
		{expr: "/r/z/namespace::q/.. is /r/z", want: "true"},
		{expr: "/r/z/w/namespace::p/.. is /r/z/w", want: "true"},
		{expr: "count(/r/z/w/namespace::p/ancestor::*)", want: "3"},
		{expr: "count(/r/z/w/namespace::p/following::*)", want: "2"},
		{expr: "count(/r/z/w/namespace::p/preceding::*)", want: "0"},
		{expr: "count(/r/z/w/namespace::*/self::namespace-node())", want: "3"},
		{expr: "count(/r/z/w/namespace::p/self::*)", want: "0"},
	})

	// InScopeNamespaceNodes is the namespace axis itself.
	doc, err := xmltree.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	z := doc.Children[0].Children[0]
	w := z.Children[0]
	if got := len(InScopeNamespaceNodes(w)); got != 3 {
		t.Errorf("InScopeNamespaceNodes(w) has %d nodes, want 3", got)
	}
	if got := len(InScopeNamespaceNodes(doc)); got != 0 {
		t.Errorf("InScopeNamespaceNodes(document) has %d nodes, want 0", got)
	}
	// An element marked as a namespace-inheritance barrier
	// (inherit-namespaces="no", XSLT 3.0 §11.9.1.1) has only its own
	// declarations in scope, plus xml.
	z.SetNSBarrier(true)
	if got := len(InScopeNamespaceNodes(z)); got != 2 {
		t.Errorf("barrier z has %d namespace nodes, want 2 (q, xml)", got)
	}
	if got := len(InScopeNamespaceNodes(w)); got != 2 {
		t.Errorf("child of barrier z has %d namespace nodes, want 2 (q, xml)", got)
	}
}

// TestCaDocumentNodeTest checks document-node(E) on constructed document
// nodes (XPath 3.1 §2.5.5.1): it matches only a document with exactly one
// element child, optionally with comments and PIs, and no text.
func TestCaDocumentNodeTest(t *testing.T) {
	mk := func(kids ...*xmltree.Node) *xmltree.Node {
		d := &xmltree.Node{Kind: xmltree.KindDocument}
		for _, k := range kids {
			d.Append(k)
		}
		return d
	}
	el := func() *xmltree.Node { return xmltree.NewElement(xmltree.Name{Local: "a"}) }
	comment := &xmltree.Node{Kind: xmltree.KindComment, Value: "c"}
	cases := []struct {
		name string
		doc  *xmltree.Node
		want string
	}{
		{"one element", mk(el()), "1"},
		{"comment and element", mk(comment, el()), "1"},
		{"two elements", mk(el(), el()), "0"},
		{"text and element", mk(xmltree.NewText("x"), el()), "0"},
		{"no element", mk(xmltree.NewText("x")), "0"},
	}
	p, err := Parse("count(self::document-node(element(a)))")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		v, err := p.Eval(&Context{Node: c.doc, Pos: 1, Size: 1})
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if ToString(v) != c.want {
			t.Errorf("%s: got %s, want %s", c.name, ToString(v), c.want)
		}
	}
	// A parentless attribute has nothing on its following/preceding axes.
	attr := xmltree.NewAttribute(xmltree.Name{Local: "a"}, "1")
	p2, err := Parse("count(following::node()) + count(preceding::node()) + count(..)")
	if err != nil {
		t.Fatal(err)
	}
	v, err := p2.Eval(&Context{Node: attr, Pos: 1, Size: 1})
	if err != nil || ToString(v) != "0" {
		t.Errorf("parentless attribute axes: %v %v, want 0", ToString(v), err)
	}
}

// TestCaUnboundPrefix checks XPST0081 for a name test whose prefix the
// static context does not bind (XPath 3.1 §2.1.1, §3.3.2.2).
func TestCaUnboundPrefix(t *testing.T) {
	ns := func(c *Context) { c.NS = caNS{"p": "urn:p"} }
	caRun(t, "<r><y/></r>", ns, []caCase{
		{expr: "count(/r/descendant::zz:y)", code: "XPST0081"},
		{expr: "count(//element(zz:y))", code: "XPST0081"},
		{expr: "count(/r/zz:*)", code: "XPST0081"},
		{expr: "count(/r/@zz:a)", code: "XPST0081"},
	})
}
