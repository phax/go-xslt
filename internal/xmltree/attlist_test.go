package xmltree

import "testing"

func TestAttlistNamespaceDeclIsNotAnAttribute(t *testing.T) {
	src := `<?xml version="1.0"?><!DOCTYPE schema [<!ATTLIST schema xmlns:ds CDATA #FIXED "urn:ds" kind CDATA "k">]>` +
		`<schema xmlns:ds="urn:ds"/>`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	root := RootElement(doc)
	for _, a := range root.Attrs {
		if a.Name.Local == "ds" {
			t.Fatalf("xmlns:ds became an attribute: %+v", a.Name)
		}
	}
	if v, ok := root.AttrLocal("kind"); !ok || v != "k" {
		t.Errorf("ordinary ATTLIST default lost: %q %v", v, ok)
	}
}
