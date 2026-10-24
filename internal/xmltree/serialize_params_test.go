package xmltree

import (
	"bytes"
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) *Node {
	t.Helper()
	doc, err := Parse(src)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return doc
}

// Serialization 3.1 parameters on the xml, html and text methods. Expected
// strings follow the parameter definitions (§5 XML output method, §7 HTML
// output method, §10 text method, §4 character maps / normalization-form).
func TestSerializationParameters(t *testing.T) {
	cases := []struct {
		name string
		src  string
		opts SerializeOptions
		want string // exact output
	}{
		{"declaration-default", `<a/>`, SerializeOptions{}, `<?xml version="1.0" encoding="UTF-8"?><a/>`},
		{"declaration-standalone", `<a/>`, SerializeOptions{Standalone: "yes"}, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><a/>`},
		{"declaration-encoding", `<a/>`, SerializeOptions{Encoding: "ISO-8859-1"}, `<?xml version="1.0" encoding="ISO-8859-1"?><a/>`},
		{"omit-declaration", `<a/>`, SerializeOptions{OmitXMLDeclaration: true}, `<a/>`},
		{"doctype-system", `<a/>`, SerializeOptions{OmitXMLDeclaration: true, DoctypeSystem: "a.dtd"}, `<!DOCTYPE a SYSTEM "a.dtd"><a/>`},
		{"doctype-public", `<a/>`, SerializeOptions{OmitXMLDeclaration: true, DoctypeSystem: "a.dtd", DoctypePublic: "-//X//Y"}, `<!DOCTYPE a PUBLIC "-//X//Y" "a.dtd"><a/>`},
		{"doctype-system-with-quote", `<a/>`, SerializeOptions{OmitXMLDeclaration: true, DoctypeSystem: `x"y.dtd`}, `<!DOCTYPE a SYSTEM 'x"y.dtd'><a/>`},
		// Characters outside the encoding's repertoire become character
		// references (§5: "... MUST be output as a character reference").
		{"ascii-repertoire", `<a b="é">ü€</a>`, SerializeOptions{OmitXMLDeclaration: true, Encoding: "US-ASCII"}, `<a b="&#233;">&#252;&#8364;</a>`},
		{"latin1-repertoire", `<a>ü€</a>`, SerializeOptions{OmitXMLDeclaration: true, Encoding: "ISO-8859-1"}, `<a>ü&#8364;</a>`},
		{"bom", `<a/>`, SerializeOptions{OmitXMLDeclaration: true, ByteOrderMark: true}, "\ufeff<a/>"},
		{"cdata-elements", `<a><b>x &lt; y</b><c>z</c></a>`, SerializeOptions{OmitXMLDeclaration: true, CDATAElements: []Name{{Local: "b"}}}, `<a><b><![CDATA[x < y]]></b><c>z</c></a>`},
		// "]]>" cannot occur inside one CDATA section: it is split.
		{"cdata-split", `<b>a]]&gt;b</b>`, SerializeOptions{OmitXMLDeclaration: true, CDATAElements: []Name{{Local: "b"}}}, `<b><![CDATA[a]]]]><![CDATA[>b]]></b>`},
		{"cdata-outside-repertoire", `<b>aé</b>`, SerializeOptions{OmitXMLDeclaration: true, Encoding: "US-ASCII", CDATAElements: []Name{{Local: "b"}}}, `<b><![CDATA[a]]>&#233;</b>`},
		{"normalization-nfc", "<a b=\"a\u030a\">a\u030a</a>", SerializeOptions{OmitXMLDeclaration: true, NormalizationForm: "NFC"}, "<a b=\"\u00e5\">\u00e5</a>"},
		{"normalization-nfd", "<a>\u00e5</a>", SerializeOptions{OmitXMLDeclaration: true, NormalizationForm: "NFD"}, "<a>a\u030a</a>"},
		{"character-map", `<a b="x">x&lt;y</a>`, SerializeOptions{OmitXMLDeclaration: true, CharacterMap: map[rune]string{'x': "&XX;"}}, `<a b="&XX;">&XX;&lt;y</a>`},
		// XML 1.1 output: control characters as character references.
		{"xml11-controls", "<a>\u0085x</a>", SerializeOptions{XMLVersion: "1.1"}, `<?xml version="1.1" encoding="UTF-8"?><a>&#133;x</a>`},
		// Text method: string value only, no markup, no escaping.
		{"text-method", `<a>x<b>&lt;y</b>z</a>`, SerializeOptions{Method: "text"}, `x<yz`},
		{"text-method-map", `<a>abc</a>`, SerializeOptions{Method: "text", CharacterMap: map[rune]string{'b': "B"}}, `aBc`},
		{"text-method-bom", `<a>x</a>`, SerializeOptions{Method: "text", ByteOrderMark: true}, "\ufeffx"},
		// HTML method: void elements have no end tag; boolean-looking
		// attributes are written as given; URI attributes are %-escaped.
		{"html-void", `<html><body><br/><p>x</p></body></html>`, SerializeOptions{Method: "html"}, `<html><body><br><p>x</p></body></html>`},
		{"html-uri-escape", `<a href="ä b">x</a>`, SerializeOptions{Method: "html"}, `<a href="%C3%A4 b">x</a>`},
		{"html-uri-no-escape", `<a href="ä">x</a>`, SerializeOptions{Method: "html", NoEscapeURIAttributes: true}, `<a href="ä">x</a>`},
		{"html-script-raw", `<script>a &lt; b</script>`, SerializeOptions{Method: "html"}, `<script>a < b</script>`},
		{"html5-doctype", `<html><body/></html>`, SerializeOptions{Method: "html", HTMLVersion: "5"}, `<!DOCTYPE html><html><body></body></html>`},
		{"html5-doctype-not-html-root", `<div/>`, SerializeOptions{Method: "html", HTMLVersion: "5"}, `<div></div>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParse(t, tc.src)
			got := Serialize(doc, tc.opts)
			if got != tc.want {
				t.Fatalf("got  %q\nwant %q", got, tc.want)
			}
			// WriteTo produces the same bytes as Serialize.
			var buf bytes.Buffer
			if err := WriteTo(&buf, doc, tc.opts); err != nil {
				t.Fatal(err)
			}
			if buf.String() != tc.want {
				t.Fatalf("WriteTo = %q, want %q", buf.String(), tc.want)
			}
		})
	}
}

// include-content-type (§7.1 / §6.1): a meta element is added as the first
// child of head, naming the media type and encoding; an existing one is
// replaced.
func TestSerializeContentTypeMeta(t *testing.T) {
	doc := mustParse(t, `<html><head><meta http-equiv="Content-Type" content="old"/><title>t</title></head></html>`)
	got := Serialize(doc, SerializeOptions{Method: "html", IncludeContentType: true, Encoding: "UTF-8", MediaType: "text/html"})
	if !strings.Contains(got, `<head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8">`) {
		t.Fatalf("no leading content-type meta: %s", got)
	}
	if strings.Contains(got, `content="old"`) {
		t.Fatalf("existing content-type meta kept: %s", got)
	}
}

// item-separator (§2 sequence normalization): written between top-level
// items when the items are kept separate.
func TestSerializeItemSeparator(t *testing.T) {
	doc := mustParse(t, `<r><a/><b/></r>`)
	root := RootElement(doc)
	got := Serialize(root, SerializeOptions{OmitXMLDeclaration: true, SeparateItems: true, ItemSeparator: "|"})
	if got != `<r><a/><b/></r>` {
		t.Fatalf("single item: %q", got)
	}
	frag := &Node{Kind: KindDocument, Children: root.Children}
	got = Serialize(frag, SerializeOptions{OmitXMLDeclaration: true, SeparateItems: true, ItemSeparator: "|"})
	if got != `<a/>|<b/>` {
		t.Fatalf("two items: %q", got)
	}
	got = Serialize(frag, SerializeOptions{Method: "text", SeparateItems: true, ItemSeparator: "|"})
	if got != `|` {
		t.Fatalf("text items: %q", got)
	}
}

// Indentation must not add whitespace inside mixed content or in elements
// listed in suppress-indentation (§5.1.3 indent / suppress-indentation).
func TestSerializeIndent(t *testing.T) {
	doc := mustParse(t, `<r><a><b/></a><m>t<i/></m><s><x/></s></r>`)
	got := Serialize(doc, SerializeOptions{OmitXMLDeclaration: true, Indent: true, SuppressIndentation: []Name{{Local: "s"}}})
	for _, want := range []string{"<m>t<i/></m>", "<s><x/></s>", "\n  <a>"} {
		if !strings.Contains(got, want) {
			t.Errorf("indent output lacks %q:\n%s", want, got)
		}
	}
	// Re-parsing yields the same element structure.
	back := mustParse(t, got)
	if len(childElems(RootElement(back))) != 3 {
		t.Errorf("structure changed by indentation:\n%s", got)
	}
}

func childElems(n *Node) []*Node {
	var out []*Node
	for _, c := range n.Children {
		if c.Kind == KindElement {
			out = append(out, c)
		}
	}
	return out
}

// Retrieved-text decoding (F&O 3.1 §14.4 fn:unparsed-text: external
// encoding information first, then a BOM / XML declaration).
func TestDecodeRetrievedText(t *testing.T) {
	latin1 := []byte("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><a>\xe9</a>")
	if got := DecodeRetrievedText(latin1); got != "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><a>é</a>" {
		t.Errorf("latin1 by declaration: %q", got)
	}
	if got := DecodeRetrievedText([]byte("\xef\xbb\xbfhi")); got != "hi" {
		t.Errorf("UTF-8 BOM: %q", got)
	}
	if got := DecodeRetrievedText([]byte("plain")); got != "plain" {
		t.Errorf("plain: %q", got)
	}
	if got := DecodeRetrievedTextAs([]byte("caf\xe9"), "ISO-8859-1"); got != "café" {
		t.Errorf("charset ISO-8859-1: %q", got)
	}
	if got := DecodeRetrievedTextAs([]byte("\x80"), "windows-1252"); got != "€" {
		t.Errorf("charset windows-1252: %q", got)
	}
	if got := DecodeRetrievedTextAs([]byte("ok"), ""); got != "ok" {
		t.Errorf("no charset: %q", got)
	}
	// A latin1 document parses with the declared encoding.
	doc, err := Parse(string(latin1))
	if err != nil {
		t.Fatal(err)
	}
	if v := RootElement(doc).StringValue(); v != "é" {
		t.Errorf("parsed latin1 value %q", v)
	}
}
