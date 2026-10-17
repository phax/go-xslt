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

// Namespaces in XML 1.0 §3: a namespace declaration "may be provided directly
// or by default". A DTD default binds the prefix (and the default namespace)
// where the element does not declare it, and an explicit declaration wins.
func TestAttlistNamespaceDefaultBinds(t *testing.T) {
	src := `<!DOCTYPE r [<!ATTLIST r xmlns:p CDATA #FIXED "urn:p" xmlns CDATA "urn:d" xmlns:q CDATA "urn:q-default">]>` +
		`<r xmlns:q="urn:q-explicit"><p:a xmlns:p="urn:p"/><c/></r>`
	doc, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	r := RootElement(doc)
	if r.Name.Space != "urn:d" {
		t.Errorf("default namespace from the DTD: root is in %q", r.Name.Space)
	}
	ns := r.InScopeNamespaces()
	if ns["p"] != "urn:p" || ns["q"] != "urn:q-explicit" {
		t.Errorf("in-scope: %v (explicit q must beat the default)", ns)
	}
	for _, a := range r.Attrs {
		t.Errorf("unexpected attribute %+v", a.Name)
	}
	// the streaming reader agrees
	sr := RootElement(streamAll(t, src))
	if sr.Name.Space != "urn:d" || sr.InScopeNamespaces()["p"] != "urn:p" || sr.InScopeNamespaces()["q"] != "urn:q-explicit" || len(sr.Attrs) != 0 {
		t.Errorf("stream reader: %q %v %d attrs", sr.Name.Space, sr.InScopeNamespaces(), len(sr.Attrs))
	}
}
