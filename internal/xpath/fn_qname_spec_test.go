package xpath

import "testing"

// TestQNameFunctionsSpec covers the error conditions and edge cases of the
// F&O 3.1 §10.1-§10.2 QName functions.
func TestQNameFunctionsSpec(t *testing.T) {
	const doc = `<r xmlns:p="urn:p" xmlns="urn:d"><c xmlns=""><e xmlns:q="urn:q" a="1"/></c></r>`
	coreRun(t, doc, nil, []coreCase{
		// §10.1.2 fn:QName
		{expr: `namespace-uri-from-QName(QName("http://www.example.com/example", "person"))`, want: "http://www.example.com/example"},
		{expr: `prefix-from-QName(QName("http://www.example.com/example", "ht:person"))`, want: "ht"},
		{expr: `local-name-from-QName(QName("", "person"))`, want: "person"},
		{expr: `local-name-from-QName(QName((), "person"))`, want: "person"},
		{expr: `QName("urn:x", "a:b") eq QName("urn:x", "c:b")`, want: "true"},
		{expr: `QName("urn:x", "b") eq QName("urn:y", "b")`, want: "false"},
		{expr: `QName("urn:x", "a b")`, err: "FOCA0002"},
		{expr: `QName("urn:x", "1a")`, err: "FOCA0002"},
		{expr: `QName("", "p:a")`, err: "FOCA0002"},
		{expr: `QName((), "p:a")`, err: "FOCA0002"},
		{expr: `QName(1, "a")`, err: "XPTY0004"},
		{expr: `QName("urn:x", 1)`, err: "XPTY0004"},
		// §10.2.2-§10.2.4 accessors; the result types are xs:NCName/xs:anyURI.
		{expr: `prefix-from-QName(QName("urn:x", "a"))`, want: "()"},
		{expr: `prefix-from-QName(QName("urn:x", "p:a")) instance of xs:NCName`, want: "true"},
		{expr: `local-name-from-QName(QName("urn:x", "p:a")) instance of xs:NCName`, want: "true"},
		{expr: `namespace-uri-from-QName(QName("urn:x", "p:a")) instance of xs:anyURI`, want: "true"},
		{expr: `local-name-from-QName(1)`, err: "XPTY0004"},
		{expr: `prefix-from-QName(xs:date("2020-01-01"))`, err: "XPTY0004"},
		{expr: `namespace-uri-from-QName((QName("a", "b"), QName("a", "c")))`, err: "XPTY0004"},
		{expr: `local-name-from-QName((QName("a", "b"), QName("a", "c")))`, err: "XPTY0004"},
		// §10.2.5 fn:namespace-uri-for-prefix
		{expr: `namespace-uri-for-prefix("p", //e)`, want: "urn:p"},
		{expr: `namespace-uri-for-prefix("q", //e)`, want: "urn:q"},
		{expr: `namespace-uri-for-prefix("xml", //e)`, want: "http://www.w3.org/XML/1998/namespace"},
		{expr: `namespace-uri-for-prefix("", /*)`, want: "urn:d"},
		{expr: `namespace-uri-for-prefix((), /*)`, want: "urn:d"},
		// q is declared only on a descendant.
		{expr: `namespace-uri-for-prefix("q", /*)`, want: "()"},
		{expr: `namespace-uri-for-prefix("p", //@a)`, err: "XPTY0004"},
		{expr: `namespace-uri-for-prefix("p", 1)`, err: "XPTY0004"},
		{expr: `namespace-uri-for-prefix("p", ())`, err: "XPTY0004"},
		// §10.2.6 fn:in-scope-prefixes ("xml" is always in scope).
		{expr: `string-join(sort(in-scope-prefixes(//e)), ',')`, want: "p,q,xml"},
		{expr: `string-join(sort(in-scope-prefixes(/*)), ',')`, want: ",p,xml"},
		{expr: `in-scope-prefixes(//@a)`, err: "XPTY0004"},
		{expr: `in-scope-prefixes(/)`, err: "XPTY0004"},
		// §10.2.1 fn:resolve-QName
		{expr: `namespace-uri-from-QName(resolve-QName("hello", //e))`, want: ""},
		{expr: `namespace-uri-from-QName(resolve-QName("hello", /*))`, want: "urn:d"},
		{expr: `namespace-uri-from-QName(resolve-QName(" q:hello ", //e))`, want: "urn:q"},
		{expr: `resolve-QName("zz:hello", //e)`, err: "FONS0004"},
		{expr: `resolve-QName("p:+thing", //e)`, err: "FOCA0002"},
		{expr: `resolve-QName("p:thi:ng", //e)`, err: "FOCA0002"},
		{expr: `resolve-QName("+thing", //e)`, err: "FOCA0002"},
		{expr: `resolve-QName("", //e)`, err: "FOCA0002"},
		{expr: `resolve-QName("a", 1)`, err: "XPTY0004"},
		{expr: `resolve-QName("a", //@a)`, err: "XPTY0004"},
	})
}
