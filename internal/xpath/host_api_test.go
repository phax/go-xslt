package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// peItems evaluates expr over peDoc and returns its items.
func peItems(t *testing.T, expr string) []Item {
	t.Helper()
	v, err := peRun(expr, peDoc, peNS{"p": "urn:p"})
	if err != nil {
		t.Fatalf("%s: %v", expr, err)
	}
	return Items(v)
}

// TestPeItemKind checks ItemKind's documented labels: the node kind for a
// node, the atomic type's QName, or map/array/function.
func TestPeItemKind(t *testing.T) {
	cases := []struct{ expr, want string }{
		{"/", "document"},
		{"/r", "element"},
		{"/r/a[1]/@x", "attribute"},
		{"/r/a[1]/text()", "text"},
		{"/r/comment()", "comment"},
		{"/r/processing-instruction()", "processing-instruction"},
		{"/r/namespace::p", "namespace"},
		{"1", "xs:integer"},
		{"1.5", "xs:decimal"},
		{"xs:date('2020-01-01')", "xs:date"},
		{"map{}", "map"},
		{"[]", "array"},
		{"upper-case#1", "function"},
	}
	for _, c := range cases {
		items := peItems(t, c.expr)
		if len(items) != 1 {
			t.Fatalf("%s: %d items", c.expr, len(items))
		}
		if got := ItemKind(items[0]); got != c.want {
			t.Errorf("ItemKind(%s) = %q, want %q", c.expr, got, c.want)
		}
	}
	// Bare Go scalars stand for xs:string / xs:double / xs:boolean.
	for _, c := range []struct {
		it   Item
		want string
	}{{"s", "xs:string"}, {2.5, "xs:double"}, {true, "xs:boolean"}, {struct{}{}, "item"}} {
		if got := ItemKind(c.it); got != c.want {
			t.Errorf("ItemKind(%#v) = %q, want %q", c.it, got, c.want)
		}
	}
}

// TestPeItemString: nodes and atomics show their string value; maps, arrays
// and functions show an adaptive serialization (Serialization 3.1 §10).
func TestPeItemString(t *testing.T) {
	nospace := func(s string) string { return strings.ReplaceAll(s, " ", "") }
	if got := ItemString(peItems(t, "/r/a[2]")[0]); got != "two" {
		t.Errorf("element: %q", got)
	}
	if got := ItemString(peItems(t, "42")[0]); got != "42" {
		t.Errorf("integer: %q", got)
	}
	if got := nospace(ItemString(peItems(t, "map{'a': 'b'}")[0])); got != `map{"a":"b"}` {
		t.Errorf("map: %q", got)
	}
	if got := nospace(ItemString(peItems(t, "[1, 'x']")[0])); got != `[1,"x"]` {
		t.Errorf("array: %q", got)
	}
	if got := ItemString(peItems(t, "upper-case#1")[0]); !strings.Contains(got, "upper-case") || !strings.HasSuffix(got, "#1") {
		t.Errorf("function: %q", got)
	}
}

// TestPeSerializeJSON: the json output method (Serialization 3.1 §9) writes
// one value; more than one item is SERE0023, duplicate key strings are
// SERE0022 unless allow-duplicate-names is set.
func TestPeSerializeJSON(t *testing.T) {
	nospace := func(s string) string { return strings.NewReplacer(" ", "", "\n", "").Replace(s) }
	if s, err := SerializeJSON(nil, OutputParams{}); err != nil || s != "" {
		t.Errorf("empty: %q %v", s, err)
	}
	if _, err := SerializeJSON(peItems(t, "(1, 2)"), OutputParams{}); err == nil || !strings.Contains(err.Error(), "SERE0023") {
		t.Errorf("two items: %v, want SERE0023", err)
	}
	if s, err := SerializeJSON(peItems(t, "map{'a': 'x'}"), OutputParams{}); err != nil || nospace(s) != `{"a":"x"}` {
		t.Errorf("map: %q %v", s, err)
	}
	if s, err := SerializeJSON(peItems(t, "[true(), 'a', ()]"), OutputParams{}); err != nil || nospace(s) != `[true,"a",null]` {
		t.Errorf("array: %q %v", s, err)
	}
	// A text node is serialized with the node output method; its string
	// becomes a JSON string.
	if s, err := SerializeJSON(peItems(t, "/r/a[1]/text()"), OutputParams{}); err != nil || s != `"one"` {
		t.Errorf("text node: %q %v", s, err)
	}
	// A character not representable in the encoding is escaped.
	if s, err := SerializeJSON(peItems(t, "'\u00e9'"), OutputParams{Encoding: "us-ascii"}); err != nil || !strings.EqualFold(s, `"\u00e9"`) {
		t.Errorf("us-ascii: %q %v", s, err)
	}
	// A mapped character is written as its replacement.
	if s, err := SerializeJSON(peItems(t, "'bab'"), OutputParams{CharacterMap: map[string]string{"a": "X"}}); err != nil || s != `"bXb"` {
		t.Errorf("character map: %q %v", s, err)
	}
	dup := peItems(t, "map{1: 'a', '1': 'b'}")
	if _, err := SerializeJSON(dup, OutputParams{}); err == nil || !strings.Contains(err.Error(), "SERE0022") {
		t.Errorf("duplicate names: %v, want SERE0022", err)
	}
	if s, err := SerializeJSON(dup, OutputParams{AllowDuplicateNames: true}); err != nil || strings.Count(s, `"1"`) != 2 {
		t.Errorf("allowed duplicate names: %q %v", s, err)
	}
}

// TestPeSerializeAdaptive: the adaptive method (Serialization 3.1 §10)
// quotes strings, writes numbers as their lexical form, and joins items
// with item-separator, a newline by default.
func TestPeSerializeAdaptive(t *testing.T) {
	items := peItems(t, "(1, 'a')")
	if s, err := SerializeAdaptive(items, OutputParams{}); err != nil || s != "1\n\"a\"" {
		t.Errorf("default separator: %q %v", s, err)
	}
	if s, err := SerializeAdaptive(items, OutputParams{ItemSeparator: "|", HasItemSeparator: true}); err != nil || s != `1|"a"` {
		t.Errorf("custom separator: %q %v", s, err)
	}
	if s, err := SerializeAdaptive(peItems(t, "true()"), OutputParams{}); err != nil || s != "true()" {
		t.Errorf("boolean: %q %v", s, err)
	}
}

// TestPeRawResultItem: a node standing for a real item is replaced by that
// item, an atomic-marked text node by its typed value (or a string when it
// has no type), and every other item is returned unchanged.
func TestPeRawResultItem(t *testing.T) {
	m := NewMap()
	carrier := xmltree.NewText("")
	carrier.SetRealItem(m)

	typed := xmltree.NewText("3")
	typed.SetAtomic(true)
	typed.SetTypeAnno(int32(XSinteger))

	untyped := xmltree.NewText("abc")
	untyped.SetAtomic(true)

	plain := xmltree.NewText("plain")
	atom := NewString("s")

	got := RawResultItems([]Item{carrier, typed, untyped, plain, atom, nil})
	if got[0] != Item(m) {
		t.Errorf("carrier: %#v", got[0])
	}
	if a, ok := got[1].(*Atomic); !ok || a.T != XSinteger || a.Lexical() != "3" {
		t.Errorf("typed: %#v", got[1])
	}
	if a, ok := got[2].(*Atomic); !ok || a.T != XSstring || a.Lexical() != "abc" {
		t.Errorf("untyped atomic: %#v", got[2])
	}
	if got[3] != Item(plain) {
		t.Errorf("plain node changed: %#v", got[3])
	}
	if got[4] != Item(atom) {
		t.Errorf("atomic changed: %#v", got[4])
	}
	if got[5] != nil {
		t.Errorf("nil changed: %#v", got[5])
	}
}
