package xpath

import (
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// rsResolver is an in-memory ResourceResolver (and CollectionResolver)
// serving fixed texts, documents and collections.
type rsResolver struct {
	texts map[string]string
	docs  map[string]*xmltree.Node
	colls map[string][]*xmltree.Node
}

func (r *rsResolver) ResolveText(uri string) (string, bool) {
	s, ok := r.texts[uri]
	return s, ok
}

func (r *rsResolver) ResolveDoc(uri string) (*xmltree.Node, bool) {
	d, ok := r.docs[uri]
	return d, ok
}

func (r *rsResolver) ResolveCollection(uri string) ([]*xmltree.Node, bool) {
	c, ok := r.colls[uri]
	return c, ok
}

// rsTextOnly implements ResourceResolver but NOT CollectionResolver.
type rsTextOnly struct{}

func (rsTextOnly) ResolveText(string) (string, bool)       { return "", false }
func (rsTextOnly) ResolveDoc(string) (*xmltree.Node, bool) { return nil, false }

func rsDoc(t *testing.T, xml, base string) *xmltree.Node {
	t.Helper()
	d, err := xmltree.Parse(xml)
	if err != nil {
		t.Fatalf("parse %q: %v", xml, err)
	}
	d.SetBase(base)
	return d
}

func rsNewResolver(t *testing.T) *rsResolver {
	a := rsDoc(t, "<a>1</a>", "http://example.com/c/a.xml")
	b := rsDoc(t, "<b>2</b>", "http://example.com/c/b.xml")
	return &rsResolver{
		texts: map[string]string{
			"http://example.com/t/plain.txt": "hello world",
			"http://example.com/t/lines.txt": "abc\r\ndef\rghi\njkl\n",
			"http://example.com/t/blank.txt": "a\n\nb",
			"http://example.com/t/empty.txt": "",
			"http://example.com/t/nul.txt":   "a\x00b",
			"http://example.com/t/bad.txt":   "a\xffb",
		},
		docs: map[string]*xmltree.Node{
			"http://example.com/d/doc.xml": rsDoc(t, "<doc><x/></doc>", "http://example.com/d/doc.xml"),
		},
		colls: map[string][]*xmltree.Node{
			"":                          {a},
			"http://example.com/c/coll": {a, b},
		},
	}
}

// TestResourceUnparsedText covers fn:unparsed-text, fn:unparsed-text-lines
// and fn:unparsed-text-available (F&O 3.1 §14.8.5-§14.8.7).
func TestResourceUnparsedText(t *testing.T) {
	res := rsNewResolver(t)
	setup := func(c *Context) {
		c.Resolver = res
		c.BaseURI = "http://example.com/t/base.xq"
	}
	coreRun(t, "", setup, []coreCase{
		// absolute and relative (against the static base URI) hrefs
		{expr: `unparsed-text('http://example.com/t/plain.txt')`, want: "hello world"},
		{expr: `unparsed-text('plain.txt')`, want: "hello world"},
		{expr: `unparsed-text('plain.txt', 'utf-8')`, want: "hello world"},
		// "If $href is the empty sequence, the function returns the empty sequence."
		{expr: `unparsed-text(())`, want: "()"},
		// FOUT1170: fragment identifier, or resource cannot be retrieved
		{expr: `unparsed-text('plain.txt#frag')`, err: "FOUT1170"},
		{expr: `unparsed-text('missing.txt')`, err: "FOUT1170"},
		// FOUT1190: invalid encoding name; characters not permitted in XML
		{expr: `unparsed-text('plain.txt', '123')`, err: "FOUT1190"},
		{expr: `unparsed-text('nul.txt')`, err: "FOUT1190"},
		// $href as xs:string?, $encoding as xs:string
		{expr: `unparsed-text(1)`, err: "XPTY0004"},
		{expr: `unparsed-text(('a','b'))`, err: "XPTY0004"},
		{expr: `unparsed-text('plain.txt', ())`, err: "XPTY0004"},

		// unparsed-text-lines: lines are separated by CRLF, CR or LF, and a
		// terminating newline does not produce a trailing empty line.
		{expr: `unparsed-text-lines('lines.txt')`, want: "abc|def|ghi|jkl"},
		{expr: `count(unparsed-text-lines('blank.txt'))`, want: "3"},
		{expr: `unparsed-text-lines('blank.txt')[2]`, want: ""},
		{expr: `unparsed-text-lines('empty.txt')`, want: "()"},
		{expr: `unparsed-text-lines('plain.txt', 'UTF-8')`, want: "hello world"},
		{expr: `unparsed-text-lines(())`, want: "()"},
		{expr: `unparsed-text-lines('missing.txt')`, err: "FOUT1170"},
		{expr: `unparsed-text-lines('plain.txt', '-x')`, err: "FOUT1190"},
		{expr: `unparsed-text-lines(1)`, err: "XPTY0004"},
		{expr: `unparsed-text-lines('plain.txt', 1)`, err: "XPTY0004"},

		// unparsed-text-available: true exactly when unparsed-text would
		// succeed; false otherwise, never an error for the retrieval outcome.
		{expr: `unparsed-text-available('plain.txt')`, want: "true"},
		{expr: `unparsed-text-available('plain.txt', 'utf-8')`, want: "true"},
		{expr: `unparsed-text-available('missing.txt')`, want: "false"},
		{expr: `unparsed-text-available('plain.txt#f')`, want: "false"},
		{expr: `unparsed-text-available('nul.txt')`, want: "false"},
		{expr: `unparsed-text-available('bad.txt')`, want: "false"},
		{expr: `unparsed-text-available('plain.txt', '123')`, want: "false"},
		{expr: `unparsed-text-available(())`, want: "false"},
		{expr: `unparsed-text-available(2)`, err: "XPTY0004"},
		{expr: `unparsed-text-available('plain.txt', ())`, err: "XPTY0004"},
	})
	// With no resolver nothing can be retrieved.
	coreRun(t, "", nil, []coreCase{
		{expr: `unparsed-text('http://example.com/t/plain.txt')`, err: "FOUT1170"},
		{expr: `unparsed-text-available('http://example.com/t/plain.txt')`, want: "false"},
	})
}

// TestResourceDoc covers fn:doc and fn:doc-available (F&O 3.1 §14.6.1/§14.6.2).
func TestResourceDoc(t *testing.T) {
	res := rsNewResolver(t)
	setup := func(c *Context) {
		c.Resolver = res
		c.BaseURI = "http://example.com/d/base.xq"
	}
	coreRun(t, "", setup, []coreCase{
		{expr: `name(doc('http://example.com/d/doc.xml')/*)`, want: "doc"},
		{expr: `name(doc('doc.xml')/*)`, want: "doc"},
		{expr: `doc('doc.xml') instance of document-node()`, want: "true"},
		// stability: two calls with the same URI return the same node
		{expr: `doc('doc.xml') is doc('http://example.com/d/doc.xml')`, want: "true"},
		{expr: `doc(())`, want: "()"},
		{expr: `doc('nope.xml')`, err: "FODC0002"},
		{expr: `doc(1)`, err: "XPTY0004"},
		{expr: `doc-available('doc.xml')`, want: "true"},
		{expr: `doc-available('nope.xml')`, want: "false"},
		// "If $uri is the empty sequence, this function returns false."
		{expr: `doc-available(())`, want: "false"},
		{expr: `doc-available(xs:integer(2))`, err: "XPTY0004"},
	})
	coreRun(t, "", nil, []coreCase{
		{expr: `doc('http://example.com/d/doc.xml')`, err: "FODC0002"},
		{expr: `doc-available('http://example.com/d/doc.xml')`, want: "false"},
	})
}

// TestResourceCollection covers fn:collection and fn:uri-collection
// (F&O 3.1 §14.6.6/§14.6.7).
func TestResourceCollection(t *testing.T) {
	res := rsNewResolver(t)
	setup := func(c *Context) {
		c.Resolver = res
		c.BaseURI = "http://example.com/c/base.xq"
	}
	coreRun(t, "", setup, []coreCase{
		{expr: `count(collection('http://example.com/c/coll'))`, want: "2"},
		{expr: `collection('coll') ! string(.)`, want: "1|2"},
		// no argument or the empty sequence: the default collection
		{expr: `collection() ! string(.)`, want: "1"},
		{expr: `collection(()) ! string(.)`, want: "1"},
		{expr: `collection('nope')`, err: "FODC0002"},
		{expr: `collection(1)`, err: "XPTY0004"},
		// uri-collection yields the URIs of the collection's documents
		{expr: `uri-collection('coll')`, want: "http://example.com/c/a.xml|http://example.com/c/b.xml"},
		{expr: `uri-collection('coll')[1] instance of xs:anyURI`, want: "true"},
		{expr: `uri-collection()`, want: "http://example.com/c/a.xml"},
		{expr: `uri-collection('nope')`, err: "FODC0002"},
	})
	// A resolver without collection support: the default collection is
	// absent (FODC0002) and so is every named one.
	coreRun(t, "", func(c *Context) { c.Resolver = rsTextOnly{} }, []coreCase{
		{expr: `collection()`, err: "FODC0002"},
		{expr: `collection('http://example.com/x')`, err: "FODC0002"},
	})
}

// TestResourceParseXML covers fn:parse-xml and fn:parse-xml-fragment
// (F&O 3.1 §14.9.1/§14.9.2), including the REC's examples.
func TestResourceParseXML(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `parse-xml("<alpha>abcd</alpha>") instance of document-node(element(alpha))`, want: "true"},
		{expr: `string(parse-xml("<alpha>abcd</alpha>"))`, want: "abcd"},
		{expr: `parse-xml(())`, want: "()"},
		{expr: `parse-xml("<a>")`, err: "FODC0006"},
		// namespace well-formedness: unbound element or attribute prefix
		{expr: `parse-xml("<p:a/>")`, err: "FODC0006"},
		{expr: `parse-xml("<a q:x='1'/>")`, err: "FODC0006"},
		{expr: `namespace-uri(parse-xml("<p:a xmlns:p='urn:p'/>")/*)`, want: "urn:p"},
		{expr: `string(parse-xml("<a xmlns:q='urn:q' q:x='1'/>")/*/@*:x)`, want: "1"},
		{expr: `namespace-uri(parse-xml("<a xmlns='urn:d'/>")/*)`, want: "urn:d"},
		// a parsed document was not retrieved from anywhere
		{expr: `document-uri(parse-xml("<a/>"))`, want: "()"},

		// parse-xml-fragment REC examples
		{expr: `count(parse-xml-fragment("<alpha>abcd</alpha><beta>abcd</beta>")/*)`, want: "2"},
		{expr: `parse-xml-fragment("<alpha>abcd</alpha><beta>abcd</beta>") instance of document-node()`, want: "true"},
		{expr: `count(parse-xml-fragment("He was <i>so</i> kind")/node())`, want: "3"},
		{expr: `string(parse-xml-fragment("He was <i>so</i> kind"))`, want: "He was so kind"},
		{expr: `count(parse-xml-fragment("")/node())`, want: "0"},
		{expr: `count(parse-xml-fragment(" ")/text())`, want: "1"},
		{expr: `parse-xml-fragment('<?xml version="1.0" encoding="UTF-8" standalone="yes"?><a/>')`, err: "FODC0006"},
		{expr: `parse-xml-fragment(())`, want: "()"},
		// a text declaration must have an encoding declaration
		{expr: `count(parse-xml-fragment('<?xml encoding="utf-8"?><a/><b/>')/*)`, want: "2"},
		{expr: `parse-xml-fragment('<?xml version="1.0"?><a/>')`, err: "FODC0006"},
		{expr: `parse-xml-fragment('<!DOCTYPE a><a/>')`, err: "FODC0006"},
		{expr: `parse-xml-fragment('<a>')`, err: "FODC0006"},
		{expr: `parse-xml-fragment('<p:a/>')`, err: "FODC0006"},
		// the fragment's top-level nodes have the document node as parent
		{expr: `let $d := parse-xml-fragment('<a/><b/>') return ($d/b/.. is $d)`, want: "true"},
	})
}

// TestResourceEnvironment covers fn:environment-variable and
// fn:available-environment-variables (F&O 3.1 §14.10.4/§14.10.5).
func TestResourceEnvironment(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		// every advertised name has a value
		{expr: `every $n in available-environment-variables() satisfies exists(environment-variable($n))`, want: "true"},
		{expr: `environment-variable('GO_XSLT_NO_SUCH_VARIABLE_8d2f')`, want: "()"},
		{expr: `environment-variable(1)`, err: "XPTY0004"},
		{expr: `environment-variable(())`, err: "XPTY0004"},
	})
}

// TestResourceResolveAgainstBase pins the exported resolution helper on
// RFC 3986 §5.4.1 examples.
func TestResourceResolveAgainstBase(t *testing.T) {
	const base = "http://a/b/c/d;p?q"
	for _, c := range []struct{ base, ref, want string }{
		{base, "g", "http://a/b/c/g"},
		{base, "../g", "http://a/b/g"},
		{base, "/g", "http://a/g"},
		{base, "?y", "http://a/b/c/d;p?y"},
		{base, "g:h", "g:h"},
		// no base: the reference is returned unchanged
		{"", "rel/x.xml", "rel/x.xml"},
		// unparsable base or reference: returned unchanged
		{"http://a/%zz", "g", "g"},
		{base, "http://a/%zz", "http://a/%zz"},
	} {
		if got := ResolveAgainstBase(c.base, c.ref); got != c.want {
			t.Errorf("ResolveAgainstBase(%q, %q) = %q, want %q", c.base, c.ref, got, c.want)
		}
	}
}
