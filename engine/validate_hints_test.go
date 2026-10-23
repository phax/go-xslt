package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// Without schema documents an instance is validated by its own
// xsi:schemaLocation / xsi:noNamespaceSchemaLocation hints and xsi:type.
func TestValidateWithoutSchemas(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("n.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a" type="xs:int"/></xs:schema>`)
	write("f1.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="foo" elementFormDefault="qualified">
  <xs:element name="root"><xs:complexType><xs:sequence><xs:any processContents="lax" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="e"/></xs:schema>`)
	write("f2.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="foo"><xs:element name="other"/></xs:schema>`)
	// xmlns:p="" is not namespace-well-formed XML 1.0: no schema document.
	write("bad.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p=""><xs:element name="a"/></xs:schema>`)

	const xsi = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`
	for _, c := range []struct {
		name, inst string
		ver        engine.XSDVersion
		valid      bool
	}{
		{"hint valid", `<a ` + xsi + ` xsi:noNamespaceSchemaLocation="n.xsd">5</a>`, engine.XSD10, true},
		{"hint invalid", `<a ` + xsi + ` xsi:noNamespaceSchemaLocation="n.xsd">x</a>`, engine.XSD10, false},
		{"xsi:type", `<a ` + xsi + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:date">2005-01-01</a>`, engine.XSD10, true},
		{"misspelled xsi:Type", `<a ` + xsi + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:Type="xs:int">1</a>`, engine.XSD10, false},
		{"no declaration", `<a/>`, engine.XSD10, false},
		{"broken hint", `<a ` + xsi + ` xsi:noNamespaceSchemaLocation="bad.xsd"/>`, engine.XSD10, false},
		// XSD 1.0 §4.3.2: a new hint for a namespace already validated.
		{"late hint 1.0", `<f:root ` + xsi + ` xmlns:f="foo" xsi:schemaLocation="foo f1.xsd"><f:e xsi:schemaLocation="foo f2.xsd"/></f:root>`, engine.XSD10, false},
		{"late hint 1.1", `<f:root ` + xsi + ` xmlns:f="foo" xsi:schemaLocation="foo f1.xsd"><f:e xsi:schemaLocation="foo f2.xsd"/></f:root>`, engine.XSD11, true},
		{"repeated hint", `<f:root ` + xsi + ` xmlns:f="foo" xsi:schemaLocation="foo f1.xsd"><f:e xsi:schemaLocation="foo f1.xsd"/></f:root>`, engine.XSD10, true},
	} {
		r := engine.Validate(engine.ValidateRequest{Instance: c.inst, Version: c.ver, BaseDir: dir})
		if r.Valid != c.valid {
			t.Errorf("%s: valid=%v, want %v (%+v)", c.name, r.Valid, c.valid, r.Diagnostics)
		}
	}
}

func TestParseRejectsPrefixUndeclaringInXML10(t *testing.T) {
	r := engine.EvalXPath(engine.XPathRequest{Expression: `count(parse-xml('<a xmlns:p=""/>'))`})
	if len(r.Diagnostics) == 0 {
		t.Errorf("xmlns:p=\"\" in an XML 1.0 document must be rejected, got %q", r.Value)
	}
	r = engine.EvalXPath(engine.XPathRequest{Expression: `count(parse-xml('<a xmlns=""/>'))`})
	if len(r.Diagnostics) != 0 || r.Value != "1" {
		t.Errorf(`xmlns="" is allowed, got %q %+v`, r.Value, r.Diagnostics)
	}
}
