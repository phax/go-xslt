package xpath

import "testing"

// TestURIEncodingFunctions covers fn:encode-for-uri, fn:iri-to-uri and
// fn:escape-html-uri (F&O 3.1 §6.2-§6.4) with the REC's examples.
func TestURIEncodingFunctions(t *testing.T) {
	coreRun(t, "", nil, []coreCase{
		{expr: `encode-for-uri("http://www.example.com/00/Weather/CA/Los%20Angeles#ocean")`,
			want: "http%3A%2F%2Fwww.example.com%2F00%2FWeather%2FCA%2FLos%2520Angeles%23ocean"},
		{expr: `concat("http://www.example.com/", encode-for-uri("~bébé"))`, want: "http://www.example.com/~b%C3%A9b%C3%A9"},
		{expr: `concat("http://www.example.com/", encode-for-uri("100% organic"))`, want: "http://www.example.com/100%25%20organic"},
		{expr: `encode-for-uri(())`, want: ""},
		{expr: `encode-for-uri(1)`, err: "XPTY0004"},

		{expr: `iri-to-uri("http://www.example.com/00/Weather/CA/Los%20Angeles#ocean")`, want: "http://www.example.com/00/Weather/CA/Los%20Angeles#ocean"},
		{expr: `iri-to-uri("http://www.example.com/~bébé")`, want: "http://www.example.com/~b%C3%A9b%C3%A9"},
		// Characters that must be escaped: space, < > " { } | \ ^ ` .
		{expr: `iri-to-uri('a b<c>"{}|\^` + "`" + `')`, want: "a%20b%3Cc%3E%22%7B%7D%7C%5C%5E%60"},
		{expr: `iri-to-uri("http://h/p?a=1&b=[2]")`, want: "http://h/p?a=1&b=[2]"},
		{expr: `iri-to-uri(())`, want: ""},
		{expr: `iri-to-uri(1)`, err: "XPTY0004"},

		{expr: `escape-html-uri("http://www.example.com/00/Weather/CA/Los Angeles#ocean")`, want: "http://www.example.com/00/Weather/CA/Los Angeles#ocean"},
		{expr: `escape-html-uri("javascript:if (navigator.browserLanguage == 'fr') window.open('http://www.example.com/~bébé');")`,
			want: "javascript:if (navigator.browserLanguage == 'fr') window.open('http://www.example.com/~b%C3%A9b%C3%A9');"},
		{expr: `escape-html-uri(codepoints-to-string((97, 9, 127)))`, want: "a%09%7F"},
		{expr: `escape-html-uri(())`, want: ""},
		{expr: `escape-html-uri(1)`, err: "XPTY0004"},
	})
}

// TestURIResolve covers fn:resolve-uri (F&O 3.1 §6.1), which resolves per
// RFC 3986 §5.2; the expectations are RFC 3986 §5.4's own reference
// resolution examples against the base "http://a/b/c/d;p?q".
func TestURIResolve(t *testing.T) {
	rfc := []struct{ rel, want string }{
		// §5.4.1 normal examples
		{"g:h", "g:h"},
		{"g", "http://a/b/c/g"},
		{"./g", "http://a/b/c/g"},
		{"g/", "http://a/b/c/g/"},
		{"/g", "http://a/g"},
		{"//g", "http://g"},
		{"?y", "http://a/b/c/d;p?y"},
		{"g?y", "http://a/b/c/g?y"},
		{"#s", "http://a/b/c/d;p?q#s"},
		{"g#s", "http://a/b/c/g#s"},
		{"g?y#s", "http://a/b/c/g?y#s"},
		{";x", "http://a/b/c/;x"},
		{"g;x", "http://a/b/c/g;x"},
		{"g;x?y#s", "http://a/b/c/g;x?y#s"},
		{"", "http://a/b/c/d;p?q"},
		{".", "http://a/b/c/"},
		{"./", "http://a/b/c/"},
		{"..", "http://a/b/"},
		{"../", "http://a/b/"},
		{"../g", "http://a/b/g"},
		{"../..", "http://a/"},
		{"../../", "http://a/"},
		{"../../g", "http://a/g"},
		// §5.4.2 abnormal examples
		{"../../../g", "http://a/g"},
		{"../../../../g", "http://a/g"},
		{"/./g", "http://a/g"},
		{"/../g", "http://a/g"},
		{"g.", "http://a/b/c/g."},
		{".g", "http://a/b/c/.g"},
		{"g..", "http://a/b/c/g.."},
		{"..g", "http://a/b/c/..g"},
		{"./../g", "http://a/b/g"},
		{"./g/.", "http://a/b/c/g/"},
		{"g/./h", "http://a/b/c/g/h"},
		{"g/../h", "http://a/b/c/h"},
		{"g;x=1/./y", "http://a/b/c/g;x=1/y"},
		{"g;x=1/../y", "http://a/b/c/y"},
		{"g?y/./x", "http://a/b/c/g?y/./x"},
		{"g?y/../x", "http://a/b/c/g?y/../x"},
		{"g#s/./x", "http://a/b/c/g#s/./x"},
		{"g#s/../x", "http://a/b/c/g#s/../x"},
		{"http:g", "http:g"},
	}
	var cases []coreCase
	for _, r := range rfc {
		cases = append(cases, coreCase{expr: `resolve-uri("` + r.rel + `", "http://a/b/c/d;p?q")`, want: r.want})
	}
	cases = append(cases,
		// §5.2.3 merge with an authority and an empty base path.
		coreCase{expr: `resolve-uri("g", "http://a")`, want: "http://a/g"},
		coreCase{expr: `resolve-uri("//h/x/../y?q", "http://a/b")`, want: "http://h/y?q"},
		coreCase{expr: `resolve-uri("g", "urn:x")`, want: "urn:g"},
		coreCase{expr: `resolve-uri("g", "http://a/b") instance of xs:anyURI`, want: "true"},
		coreCase{expr: `resolve-uri((), "http://a/b")`, want: "()"},
		// An invalid reference or a base that is not absolute: FORG0002.
		coreCase{expr: `resolve-uri("http:%%", "http://a/b")`, err: "FORG0002"},
		coreCase{expr: `resolve-uri(":", "http://a/b")`, err: "FORG0002"},
		coreCase{expr: `resolve-uri("##x", "http://a/b")`, err: "FORG0002"},
		coreCase{expr: `resolve-uri("1a:b", "http://a/b")`, err: "FORG0002"},
		coreCase{expr: `resolve-uri("g", "b/c")`, err: "FORG0002"},
		coreCase{expr: `resolve-uri("g", "http://a/b#frag")`, err: "FORG0002"},
		coreCase{expr: `resolve-uri("g", "http://a/%zz")`, err: "FORG0002"},
	)
	coreRun(t, "", nil, cases)
	// The one-argument form resolves against the static base URI.
	coreRun(t, "", func(c *Context) { c.BaseURI = "http://a/b/c/d;p?q" }, []coreCase{
		{expr: `resolve-uri("../g")`, want: "http://a/b/g"},
		{expr: `resolve-uri("http://x/y")`, want: "http://x/y"},
	})
}

// TestURIBaseURI covers fn:base-uri (F&O 3.1 §2.5) over XML Base: xml:base
// attributes of the ancestor-or-self elements, resolved innermost against
// outermost and against the document's base URI.
func TestURIBaseURI(t *testing.T) {
	const doc = `<r xml:base="http://ex.com/a/"><s xml:base="sub/" a="1"><t>x<!--c--><?pi d?></t></s><u xml:base="/top/"/></r>`
	coreRun(t, doc, nil, []coreCase{
		{expr: `base-uri(/r)`, want: "http://ex.com/a/"},
		{expr: `base-uri(//t)`, want: "http://ex.com/a/sub/"},
		{expr: `base-uri(//s/@a)`, want: "http://ex.com/a/sub/"},
		{expr: `base-uri(//t/text())`, want: "http://ex.com/a/sub/"},
		{expr: `base-uri(//t/comment())`, want: "http://ex.com/a/sub/"},
		{expr: `base-uri(//t/processing-instruction())`, want: "http://ex.com/a/sub/"},
		{expr: `base-uri(//u)`, want: "http://ex.com/top/"},
		{expr: `base-uri(//t) instance of xs:anyURI`, want: "true"},
		{expr: `base-uri(())`, want: "()"},
	})
	coreRun(t, `<r xml:base="x/"><t/></r>`, func(c *Context) { c.BaseURI = "http://h/doc.xml" }, []coreCase{
		{expr: `base-uri(//t)`, want: "http://h/x/"},
		{expr: `base-uri(/)`, want: "http://h/doc.xml"},
	})
}
