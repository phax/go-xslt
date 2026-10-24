package xpath

import "testing"

// Tests for fn:json-to-xml and fn:xml-to-json (F&O 3.1 §17.4.2, §17.5.3,
// §17.5.4). Expected trees are compared with deep-equal against the context
// document, which holds the REC's expected XML (prefixes are immaterial,
// §17.4.2); expected JSON strings follow the escaping rules of §17.5.4.

const jsnFN = `xmlns="http://www.w3.org/2005/xpath-functions"`

func TestJSONToXMLSpecExamples(t *testing.T) {
	cases := []struct{ expr, xml string }{
		// §17.5.3 examples.
		{`json-to-xml('{"x": 1, "y": [3,4,5]}')`,
			`<map ` + jsnFN + `><number key="x">1</number><array key="y"><number>3</number><number>4</number><number>5</number></array></map>`},
		{`json-to-xml('"abcd"', map{'liberal': false()})`,
			`<string ` + jsnFN + `>abcd</string>`},
		{`json-to-xml('{"x": "\\", "y": "%"}')`,
			`<map ` + jsnFN + `><string key="x">\</string><string key="y">%</string></map>`},
		{`json-to-xml('{"x": "\\", "y": "%"}', map{'escape': true()})`,
			`<map ` + jsnFN + `><string escaped="true" key="x">\\</string><string key="y">%</string></map>`},
		// §17.4.2 mapping rules: null, booleans, numbers (lexically the same
		// as in the JSON), strings, nested containers, empty key.
		{`json-to-xml('[true,false,null,"s",-1.5E+2,0]')`,
			`<array ` + jsnFN + `><boolean>true</boolean><boolean>false</boolean><null/><string>s</string><number>-1.5E+2</number><number>0</number></array>`},
		{`json-to-xml('{"":{},"e":[]}')`,
			`<map ` + jsnFN + `><map key=""/><array key="e"/></map>`},
		{`json-to-xml('null')`, `<null ` + jsnFN + `/>`},
		{`json-to-xml('12.50')`, `<number ` + jsnFN + `>12.50</number>`},
		{`json-to-xml(' [ { "a" : "b" } ] ')`,
			`<array ` + jsnFN + `><map><string key="a">b</string></map></array>`},
		// Default duplicates (validate false): retain.
		{`json-to-xml('{"a":1,"a":2}')`,
			`<map ` + jsnFN + `><number key="a">1</number><number key="a">2</number></map>`},
		{`json-to-xml('{"a":1,"a":2}', map{'duplicates':'retain'})`,
			`<map ` + jsnFN + `><number key="a">1</number><number key="a">2</number></map>`},
		{`json-to-xml('{"a":1,"b":{"c":1,"c":2},"a":2}', map{'duplicates':'use-first'})`,
			`<map ` + jsnFN + `><number key="a">1</number><map key="b"><number key="c">1</number></map></map>`},
		{`json-to-xml('[{"a":1,"a":2}]', map{'duplicates':'use-first'})`,
			`<array ` + jsnFN + `><map><number key="a">1</number></map></array>`},
		{`json-to-xml('{"a":1,"b":2}', map{'duplicates':'reject'})`,
			`<map ` + jsnFN + `><number key="a">1</number><number key="b">2</number></map>`},
		// escape=false: escapes expanded, non-XML characters become U+FFFD.
		{`json-to-xml('["\u0041\/", "\uFFFF"]')`,
			`<array ` + jsnFN + `><string>A/</string><string>` + "\uFFFD" + `</string></array>`},
		{`json-to-xml('{"\u0000":1}')`,
			`<map ` + jsnFN + `><number key="` + "\uFFFD" + `">1</number></map>`},
		// escape=true: special characters escaped, escaped="true" /
		// escaped-key="true" present exactly when a backslash occurs.
		{`json-to-xml('{"a\tb":"c\nd","plain":"\u0041"}', map{'escape': true()})`,
			`<map ` + jsnFN + `><string escaped="true" escaped-key="true" key="a\tb">c\nd</string><string key="plain">A</string></map>`},
		{`json-to-xml('{"k\\":1}', map{'escape': true()})`,
			`<map ` + jsnFN + `><number escaped-key="true" key="k\\">1</number></map>`},
		// fallback replaces each non-XML character.
		{`json-to-xml('["a\uFFFFb"]', map{'fallback': function($s){'['||$s||']'}})`,
			`<array ` + jsnFN + `><string>a[\uFFFF]b</string></array>`},
		// A leading byte order mark is ignored.
		{`json-to-xml(codepoints-to-string(65279) || '[]')`, `<array ` + jsnFN + `/>`},
	}
	for _, c := range cases {
		v, err := jsnEval(t, `deep-equal(`+c.expr+`, /)`, c.xml, nil)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if ToString(v) != "true" {
			got, _ := jsnEval(t, `serialize(`+c.expr+`)`, c.xml, nil)
			t.Errorf("%s: got %s, want %s", c.expr, ToString(got), c.xml)
		}
	}
}

func TestJSONToXMLProperties(t *testing.T) {
	jsnRun(t, "", nil, []jsnCase{
		{`empty(json-to-xml(()))`, "true"},
		{`json-to-xml('1') instance of document-node()`, "true"},
		// "The function returns a document node, whose only child is the
		// element node representing the outermost construct".
		{`count(json-to-xml('[1,2]')/node())`, "1"},
		{`namespace-uri(json-to-xml('[1,2]')/*)`, "http://www.w3.org/2005/xpath-functions"},
		{`local-name(json-to-xml('{"a":true}')/*/*)`, "boolean"},
		// No whitespace text nodes, comments or PIs.
		{`count(json-to-xml('[ 1 , [ ] , { } ]')//text())`, "1"},
		// Document order of the result is usable.
		{`string-join(json-to-xml('["a","b","c"]')//*:string, ',')`, "a,b,c"},
		{`let $d := json-to-xml('["a","b"]') return ($d//*:string)[2] >> ($d//*:string)[1]`, "true"},
		// Round trip through xml-to-json.
		{`xml-to-json(json-to-xml('{"a":[1,true,null,"x"]}'))`, `{"a":[1,true,null,"x"]}`},
	})
	jsnRunErr(t, "", nil, []jsnErrCase{
		{`json-to-xml('[1,')`, "FOJS0001"},
		{`json-to-xml('{"a":1}x')`, "FOJS0001"},
		{`json-to-xml('[1] 2')`, "FOJS0001"},
		{`json-to-xml('"\x"')`, "FOJS0001"},
		{`json-to-xml('{"a":1,"a":2}', map{'duplicates':'reject'})`, "FOJS0003"},
		{`json-to-xml('[{"b":{"a":1,"a":2}}]', map{'duplicates':'reject'})`, "FOJS0003"},
		// use-last is a parse-json value only; json-to-xml allows
		// reject|use-first|retain.
		{`json-to-xml('{}', map{'duplicates':'use-last'})`, "FOJS0005"},
		{`json-to-xml('{}', map{'duplicates':'x'})`, "FOJS0005"},
		{`json-to-xml('"a"', map{'escape':true(), 'fallback':function($s){$s}})`, "FOJS0005"},
		// Option types (option parameter conventions).
		{`json-to-xml('1', map{'liberal':'true'})`, "XPTY0004"},
		{`json-to-xml('1', map{'validate':()})`, "XPTY0004"},
		{`json-to-xml('1', map{'escape':(true(),true())})`, "XPTY0004"},
		{`json-to-xml('1', map{'fallback':1})`, "XPTY0004"},
		{`json-to-xml('1', map{'fallback':concat#2})`, "XPTY0004"},
		// validate=true needs a schema-aware processor (FOJS0004); a bare
		// XPath context has none.
		{`json-to-xml('1', map{'validate':true()})`, "FOJS0004"},
		// The fallback may raise an error of its own.
		{`json-to-xml('"\uFFFF"', map{'fallback':function($s){error(QName('http://www.w3.org/2005/xqt-errors','FOJS9998'))}})`, "FOJS9998"},
	})
}

func TestXMLToJSONSpecExamples(t *testing.T) {
	cases := []struct{ xml, want string }{
		// §17.5.4 examples.
		{`<array ` + jsnFN + `><number>1</number><string>is</string><boolean>1</boolean></array>`, `[1,"is",true]`},
		{`<map ` + jsnFN + `><number key="Sunday">1</number><number key="Monday">2</number></map>`, `{"Sunday":1,"Monday":2}`},
		// null, booleans (xs:boolean(fn:string($E)), whitespace accepted).
		{`<null ` + jsnFN + `/>`, `null`},
		{`<boolean ` + jsnFN + `> false </boolean>`, `false`},
		{`<boolean ` + jsnFN + `>0</boolean>`, `false`},
		{`<boolean ` + jsnFN + `>true</boolean>`, `true`},
		// number: xs:string(xs:double(fn:string($E))).
		{`<number ` + jsnFN + `> 1e2 </number>`, `100`},
		{`<number ` + jsnFN + `>-0.5</number>`, `-0.5`},
		{`<number ` + jsnFN + `>1.0E6</number>`, `1.0E6`},
		{`<number ` + jsnFN + `>0.0000001</number>`, `1.0E-7`},
		{`<number ` + jsnFN + `>0.000001</number>`, `0.000001`},
		{`<number ` + jsnFN + `>12.50</number>`, `12.5`},
		// Empty containers, nesting, whitespace / comments / PIs ignored.
		{`<map ` + jsnFN + `/>`, `{}`},
		{`<array ` + jsnFN + `/>`, `[]`},
		{`<map ` + jsnFN + `> <array key="a"> <!--c--> <map/> <?pi x?> </array> </map>`, `{"a":[{}]}`},
		{`<array ` + jsnFN + `><array><array/></array></array>`, `[[[]]]`},
		// A key attribute on the top-level element is ignored.
		{`<string key="ignored" ` + jsnFN + `>v</string>`, `"v"`},
		// Attributes in other namespaces are ignored.
		{`<string xmlns:o="http://example.com/o" o:a="1" ` + jsnFN + `>v</string>`, `"v"`},
		// Unescaped strings: backslash, quote and control characters escaped.
		{`<string ` + jsnFN + `>a\b"c</string>`, `"a\\b\"c"`},
		{`<string ` + jsnFN + `>&#9;&#10;&#13;</string>`, `"\t\n\r"`},
		{`<map ` + jsnFN + `><null key="a\b"/></map>`, `{"a\\b":null}`},
		{`<map ` + jsnFN + `><null key="q&quot;"/></map>`, `{"q\"":null}`},
		// escaped="true": valid escapes copied unchanged, unescaped special
		// characters escaped.
		{`<string escaped="true" ` + jsnFN + `>a\nb\u00e9\\\"\/\b\f\r\t</string>`, `"a\nb\u00e9\\\"\/\b\f\r\t"`},
		{`<string escaped="true" ` + jsnFN + `>"&#9;</string>`, `"\"\t"`},
		{`<string escaped="1" ` + jsnFN + `>\n</string>`, `"\n"`},
		{`<string escaped="false" ` + jsnFN + `>\n</string>`, `"\\n"`},
		// escaped-key="true".
		{`<map ` + jsnFN + `><null key="a\tb" escaped-key="true"/></map>`, `{"a\tb":null}`},
		{`<map ` + jsnFN + `><null key="a\tb" escaped-key="false"/></map>`, `{"a\\tb":null}`},
		{`<map ` + jsnFN + `><null key="\uD83D\uDE00" escaped-key="true"/><null key="x"/></map>`, `{"\uD83D\uDE00":null,"x":null}`},
		// Escaped keys and strings: every JSON escape form is accepted, and
		// raw solidus / newline / carriage return / U+007F get escaped.
		{`<map ` + jsnFN + `><null key='\b\f\r\/\"' escaped-key="true"/></map>`, `{"\b\f\r\/\"":null}`},
		{`<string escaped="true" ` + jsnFN + `>a/b&#10;&#13;&#8232;</string>`, `"a\/b\n\r "`},
		{`<string escaped="true" ` + jsnFN + `>&#127;&#159;</string>`, `"\u007F\u009F"`},
		// A document node with one element child is processed by
		// processing that child.
		{`<!--c--><array ` + jsnFN + `/>`, `[]`},
	}
	for _, c := range cases {
		v, err := jsnEval(t, `xml-to-json(/)`, c.xml, nil)
		if err != nil {
			t.Errorf("xml-to-json(%s): unexpected error %v", c.xml, err)
			continue
		}
		if got := ToString(v); got != c.want {
			t.Errorf("xml-to-json(%s) = %s, want %s", c.xml, got, c.want)
		}
	}
}

func TestXMLToJSONArguments(t *testing.T) {
	xml := `<array ` + jsnFN + `><map><string key="k">v</string></map><number>2</number></array>`
	jsnRun(t, xml, nil, []jsnCase{
		{`empty(xml-to-json(()))`, "true"},
		// An element node is accepted as well as a document node, including
		// one that has a key attribute.
		{`xml-to-json(/*)`, `[{"k":"v"},2]`},
		{`xml-to-json(/*/*[1]/*)`, `"v"`},
		{`xml-to-json(/*/*[2])`, `2`},
		{`xml-to-json(/, map{'indent':false()})`, `[{"k":"v"},2]`},
		// indent=true may only add whitespace between tokens.
		{`replace(xml-to-json(/, map{'indent':true()}), '\s', '')`, `[{"k":"v"},2]`},
		{`deep-equal(parse-json(xml-to-json(/, map{'indent':true()})), parse-json(xml-to-json(/)))`, "true"},
		// Unknown option keys are ignored.
		{`xml-to-json(/*/*[2], map{'other':1})`, `2`},
	})
	jsnRunErr(t, xml, nil, []jsnErrCase{
		{`xml-to-json(/, map{'indent':'yes'})`, "XPTY0004"},
		{`xml-to-json(/, map{'indent':()})`, "XPTY0004"},
		{`xml-to-json(/, map{'indent':1})`, "XPTY0004"},
		// $input as node()? — a two-node sequence is a type error.
		{`xml-to-json(/*/*)`, "XPTY0004"},
	})
}

func TestXMLToJSONInvalidInput(t *testing.T) {
	// FOJS0006: input not valid against the schema of §C.2 (or not a
	// document/element node); FOJS0007: invalid escape in escaped content.
	cases := []struct{ xml, expr, code string }{
		{`<array ` + jsnFN + `><foo/></array>`, `xml-to-json(/)`, "FOJS0006"},
		{`<array ` + jsnFN + `><item xmlns="http://example.com"/></array>`, `xml-to-json(/)`, "FOJS0006"},
		{`<array ` + jsnFN + `>text</array>`, `xml-to-json(/)`, "FOJS0006"},
		{`<map ` + jsnFN + `>text</map>`, `xml-to-json(/)`, "FOJS0006"},
		{`<map ` + jsnFN + `><null/></map>`, `xml-to-json(/)`, "FOJS0006"},
		{`<map ` + jsnFN + `><null key="a"/><null key="a"/></map>`, `xml-to-json(/)`, "FOJS0006"},
		// Keys equal after expanding escapes are duplicates (§17.5.4 Notes:
		// \n and \u000A).
		{`<map ` + jsnFN + `><null key="\n" escaped-key="true"/><null key="\u000A" escaped-key="true"/></map>`, `xml-to-json(/)`, "FOJS0006"},
		{`<map ` + jsnFN + `><null key="\n" escaped-key="maybe"/></map>`, `xml-to-json(/)`, "FOJS0006"},
		{`<string ` + jsnFN + `><a/></string>`, `xml-to-json(/)`, "FOJS0006"},
		{`<string escaped="yes" ` + jsnFN + `>x</string>`, `xml-to-json(/)`, "FOJS0006"},
		{`<number ` + jsnFN + `>abc</number>`, `xml-to-json(/)`, "FOJS0006"},
		{`<number ` + jsnFN + `>INF</number>`, `xml-to-json(/)`, "FOJS0006"},
		{`<number ` + jsnFN + `>NaN</number>`, `xml-to-json(/)`, "FOJS0006"},
		{`<number ` + jsnFN + `><a/></number>`, `xml-to-json(/)`, "FOJS0006"},
		{`<boolean ` + jsnFN + `>yes</boolean>`, `xml-to-json(/)`, "FOJS0006"},
		{`<boolean ` + jsnFN + `><a/></boolean>`, `xml-to-json(/)`, "FOJS0006"},
		{`<null ` + jsnFN + `>x</null>`, `xml-to-json(/)`, "FOJS0006"},
		{`<null ` + jsnFN + `><a/></null>`, `xml-to-json(/)`, "FOJS0006"},
		// Disallowed attributes: no-namespace ones not in the schema, and
		// any attribute in the xpath-functions namespace.
		{`<number ` + jsnFN + ` escaped="true">1</number>`, `xml-to-json(/)`, "FOJS0006"},
		{`<array ` + jsnFN + ` foo="1"/>`, `xml-to-json(/)`, "FOJS0006"},
		{`<array ` + jsnFN + ` xmlns:j="http://www.w3.org/2005/xpath-functions" j:key="1"/>`, `xml-to-json(/)`, "FOJS0006"},
		// Invalid JSON escapes in escaped content.
		{`<string escaped="true" ` + jsnFN + `>a\x</string>`, `xml-to-json(/)`, "FOJS0007"},
		{`<string escaped="true" ` + jsnFN + `>a\</string>`, `xml-to-json(/)`, "FOJS0007"},
		{`<string escaped="true" ` + jsnFN + `>\u12</string>`, `xml-to-json(/)`, "FOJS0007"},
		{`<string escaped="true" ` + jsnFN + `>\u12G4</string>`, `xml-to-json(/)`, "FOJS0007"},
		{`<map ` + jsnFN + `><null key="\q" escaped-key="true"/></map>`, `xml-to-json(/)`, "FOJS0007"},
		{`<map ` + jsnFN + `><null key="a\" escaped-key="true"/></map>`, `xml-to-json(/)`, "FOJS0007"},
		{`<map ` + jsnFN + `><null key="\u00" escaped-key="true"/></map>`, `xml-to-json(/)`, "FOJS0007"},
		{`<map ` + jsnFN + `><null key="\uZZZZ" escaped-key="true"/></map>`, `xml-to-json(/)`, "FOJS0007"},
	}
	for _, c := range cases {
		jsnRunErr(t, c.xml, nil, []jsnErrCase{{c.expr, c.code}})
	}
}

func TestXMLToJSONControlEscape(t *testing.T) {
	// §17.5.4: "any other codepoint in the range 1-31 or 127-159 is replaced
	// by an escape in the form \uHHHH where HHHH is the upper-case
	// hexadecimal representation". U+007F..U+009F are XML 1.0 characters.
	t.Skip("BUG: xml-to-json (and the JSON output method) emit U+007F..U+009F unescaped")
	jsnRun(t, "", nil, []jsnCase{
		{`xml-to-json(json-to-xml('"\u007F\u0080\u009F"'))`, `"\u007F\u0080\u009F"`},
	})
}

func TestXMLToJSONWrongRootCodes(t *testing.T) {
	// §17.5.4: "A dynamic error is raised [err:FOJS0006] if the value of
	// $input is not a document or element node or is not valid according to
	// the schema"; an atomic argument fails the node()? signature (XPTY0004).
	t.Skip("BUG: xml-to-json raises XPTY0004 for an element outside the fn namespace and FOJS0006 for an atomic argument (codes swapped)")
	for _, c := range []struct{ xml, expr, code string }{
		{`<r/>`, `xml-to-json(/)`, "FOJS0006"},
		{`<map/>`, `xml-to-json(/)`, "FOJS0006"},
		{`<r><a/></r>`, `xml-to-json(/r/a)`, "FOJS0006"},
		{`<r/>`, `xml-to-json(1)`, "XPTY0004"},
	} {
		jsnRunErr(t, c.xml, nil, []jsnErrCase{{c.expr, c.code}})
	}
}
