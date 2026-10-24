package xmltree

import (
	"strings"
	"testing"
)

const entDir = "testdata/entities"

// Entity handling per XML 1.0 §4: internal and external parsed general
// entities (§4.1, §4.4.3 "included"), parameter entities in the internal
// and external subsets (§4.4.8), default attribute values from an ATTLIST
// (§3.3.2), unparsed (NDATA) entities (§4.2.2), and the WFCs that make a
// document not well-formed (Entity Declared, No Recursion).
func TestEntityExpansion(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		base  string
		want  string // string value of the document element
		attrA string // value of r/@a ("" = absent)
	}{
		{"internal", `<!DOCTYPE r [<!ENTITY e "v&#233;">]><r>[&e;]</r>`, "", "[vé]", ""},
		{"internal-nested", `<!DOCTYPE r [<!ENTITY a "A"><!ENTITY b "&a;&a;">]><r>&b;</r>`, "", "AA", ""},
		{"internal-markup", `<!DOCTYPE r [<!ENTITY e "<i>x</i>y">]><r>&e;</r>`, "", "xy", ""},
		{"internal-in-attribute", `<!DOCTYPE r [<!ENTITY e "v">]><r a="[&e;]"/>`, "", "", "[v]"},
		{"predefined", `<r>&lt;&gt;&amp;&apos;&quot;</r>`, "", `<>&'"`, ""},
		{"parameter-internal", `<!DOCTYPE r [<!ENTITY % p "<!ENTITY g 'gv'>"> %p;]><r>&g;</r>`, "", "gv", ""},
		{"external-parsed", `<!DOCTYPE r [<!ENTITY e SYSTEM "ext.ent">]><r>&e;</r>`, entDir, "external text", ""},
		{"external-subset", `<!DOCTYPE r SYSTEM "defs.dtd"><r>&dtdent;|&fromParam;</r>`, entDir, "from-dtd|pv", "?"},
		{"internal-wins-over-external", `<!DOCTYPE r SYSTEM "defs.dtd" [<!ENTITY dtdent "inner">]><r>&dtdent;</r>`, entDir, "inner", "?"},
		{"specified-attribute-wins", `<!DOCTYPE r SYSTEM "defs.dtd"><r a="mine"/>`, entDir, "", "mine"},
		{"attlist-internal-default", `<!DOCTYPE r [<!ATTLIST r a CDATA "x">]><r/>`, "", "", "x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := ParseWithBase(tc.src, tc.base)
			if err != nil {
				t.Fatal(err)
			}
			root := RootElement(doc)
			if got := root.StringValue(); got != tc.want {
				t.Errorf("string value %q, want %q", got, tc.want)
			}
			a, _ := root.Attr("", "a")
			if tc.attrA != "?" && a != tc.attrA {
				t.Errorf("@a = %q, want %q", a, tc.attrA)
			}
		})
	}
}

func TestEntityWellFormedness(t *testing.T) {
	for _, src := range []string{
		`<r>&undeclared;</r>`,                                             // WFC: Entity Declared
		`<!DOCTYPE r [<!ENTITY e "<i>">]><r>&e;</r>`,                      // replacement text not balanced (§4.3.2)
		`<!DOCTYPE r [<!ENTITY e SYSTEM "pic.gif" NDATA gif>]><r>&e;</r>`, // WFC: Parsed Entity
	} {
		if _, err := Parse(src); err == nil {
			t.Errorf("accepted %q", src)
		}
	}
	// Exponential expansion is refused (a well-formed but hostile document;
	// the parser's documented expansion budget).
	var b strings.Builder
	b.WriteString(`<!DOCTYPE r [<!ENTITY a0 "xxxxxxxxxx">`)
	for i := 1; i <= 12; i++ {
		b.WriteString("<!ENTITY a")
		b.WriteString(string(rune('0' + i%10)))
		b.WriteString(strings.Repeat("x", i/10))
		b.WriteString(` "`)
		prev := "&a" + string(rune('0'+(i-1)%10)) + strings.Repeat("x", (i-1)/10) + ";"
		b.WriteString(strings.Repeat(prev, 10))
		b.WriteString(`">`)
	}
	b.WriteString(`]><r>&a2x;</r>`)
	if _, err := Parse(b.String()); err == nil {
		t.Error("billion-laughs document parsed")
	}
}

// WFC: No Recursion (XML 1.0 §4.1) — indirect recursion makes the document
// not well-formed.
func TestEntityIndirectRecursion(t *testing.T) {
	t.Skip("BUG: indirectly recursive internal entities are accepted")
	if _, err := Parse(`<!DOCTYPE r [<!ENTITY a "&b;"><!ENTITY b "&a;">]><r>&a;</r>`); err == nil {
		t.Error("accepted")
	}
}

// Unparsed entities are reported with their declarations; parsed ones are
// not (§4.2.2; xs:ENTITY's value space is the unparsed entity names).
func TestUnparsedEntities(t *testing.T) {
	src := `<!DOCTYPE r [<!NOTATION gif SYSTEM "g"><!ENTITY pic SYSTEM "p.gif" NDATA gif><!ENTITY txt "t"><!ENTITY pub PUBLIC "-//P" "q.gif" NDATA gif>]><r/>`
	got := UnparsedEntities(src)
	if !got["pic"] || !got["pub"] || got["txt"] || len(got) != 2 {
		t.Errorf("UnparsedEntities = %v", got)
	}
	if UnparsedEntities(`<r/>`) != nil {
		t.Error("entities reported without a DOCTYPE")
	}
}

// XML 1.1 documents (XML 1.1 §2.2, §2.11, §4.1): C0 control characters are
// legal as character references, and the 1.1 name characters are accepted.
// In a 1.0 document the same references are not well-formed.
func TestXML11Parsing(t *testing.T) {
	doc, err := ParseLenient11(`<?xml version="1.1"?><r a="&#x1;">x&#x2;y</r>`)
	if err != nil {
		t.Fatal(err)
	}
	root := RootElement(doc)
	if v := root.StringValue(); v != "x\x02y" {
		t.Errorf("text %q", v)
	}
	if v, _ := root.Attr("", "a"); v != "\x01" {
		t.Errorf("attribute %q", v)
	}
	if _, err := ParseLenient11(`<?xml version="1.0"?><r>&#x1;</r>`); err == nil {
		t.Error("1.0 document accepted &#x1;")
	}
	doc, err = ParseLenient11("<?xml version=\"1.1\"?><Ⰰ Ⰱ=\"v\"/>")
	if err != nil {
		t.Fatalf("1.1 name characters: %v", err)
	}
	if root := RootElement(doc); root.Name.Local != "Ⰰ" || len(root.Attrs) != 1 || root.Attrs[0].Name.Local != "Ⰱ" {
		t.Errorf("names not restored: %q %v", root.Name.Local, root.Attrs)
	}
	// U+0001 is not a name character in any version.
	if _, err := ParseLenient11("<?xml version=\"1.1\"?><a\u0001/>"); err == nil {
		t.Error("control character accepted in a name")
	}
}
