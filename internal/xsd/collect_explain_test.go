package xsd

import (
	"strings"
	"testing"
)

// Error collection: every error, in document order, capped by maxErrors,
// with the failing element's path; content-model errors classified by Kind
// and Particle (xsd.go's Error contract). The verdict is fixed by the spec;
// these cases check that the diagnostics describe the defects the instance
// actually has.
func TestErrorCollection(t *testing.T) {
	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	  <xs:element name="doc"><xs:complexType><xs:sequence>
	    <xs:element name="row" maxOccurs="unbounded"><xs:complexType><xs:sequence>
	      <xs:element name="n" type="xs:int"/>
	    </xs:sequence><xs:attribute name="id" type="xs:int" use="required"/></xs:complexType></xs:element>
	  </xs:sequence></xs:complexType></xs:element>
	</xs:schema>`
	s, err := Compile([]string{schema}, "", Version10)
	if err != nil {
		t.Fatal(err)
	}
	inst := `<doc>
	  <row id="1"><n>x</n></row>
	  <row><n>2</n></row>
	  <row id="y"><n>3</n></row>
	</doc>`
	res, err := s.ValidateOpts(inst, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid || len(res.Errors) != 3 {
		t.Fatalf("want 3 errors, got valid=%v %v", res.Valid, res.Errors)
	}
	for i := 1; i < len(res.Errors); i++ {
		a, b := res.Errors[i-1], res.Errors[i]
		if a.Line > b.Line || (a.Line == b.Line && a.Col > b.Col) {
			t.Errorf("errors not in document order: %v then %v", a, b)
		}
	}
	wantPaths := []string{"/doc[1]/row[1]/n[1]", "/doc[1]/row[2]", "/doc[1]/row[3]"}
	for i, e := range res.Errors {
		if e.Path != wantPaths[i] {
			t.Errorf("error %d path = %q, want %q", i, e.Path, wantPaths[i])
		}
		if e.Line == 0 {
			t.Errorf("error %d has no line", i)
		}
	}
	res, err = s.ValidateOpts(inst, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid || len(res.Errors) != 2 {
		t.Fatalf("maxErrors=2: got %d errors", len(res.Errors))
	}
	res, err = s.ValidateOpts(`<doc><row id="1"><n>1</n></row></doc>`, "", 1)
	if err != nil || !res.Valid {
		t.Fatalf("valid instance: %v %v", res, err)
	}
}

func TestContentModelExplanation(t *testing.T) {
	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	  <xs:element name="s"><xs:complexType><xs:sequence>
	    <xs:element name="a"/><xs:element name="b"/><xs:element name="c" minOccurs="0" maxOccurs="2"/>
	  </xs:sequence></xs:complexType></xs:element>
	  <xs:element name="ch"><xs:complexType><xs:choice>
	    <xs:element name="x"/><xs:element name="y"/>
	  </xs:choice></xs:complexType></xs:element>
	</xs:schema>`
	s, err := Compile([]string{schema}, "", Version11)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		inst     string
		kind     string
		particle string
	}{
		{`<s><a/><c/></s>`, "missing", "b"},
		{`<s><a/><b/><c/><c/><c/></s>`, "too-many", "c"},
		{`<s><a/><b/><z/></s>`, "unexpected", "z"},
		{`<s><b/><a/></s>`, "order", ""},
		{`<s><a/></s>`, "missing", "b"},
	}
	for _, tc := range cases {
		res, err := s.Validate(tc.inst)
		if err != nil {
			t.Fatal(err)
		}
		if res.Valid {
			t.Fatalf("%s: valid", tc.inst)
		}
		found := false
		for _, e := range res.Errors {
			if e.Kind == tc.kind && (tc.particle == "" || strings.HasSuffix(e.Particle, tc.particle)) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no %s error for %q in %+v", tc.inst, tc.kind, tc.particle, res.Errors)
		}
	}
	// Two independent defects are both reported.
	res, err := s.Validate(`<s><a/><c/><c/><c/></s>`)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, e := range res.Errors {
		kinds[e.Kind] = true
	}
	if !kinds["missing"] || !kinds["too-many"] {
		t.Errorf("want missing b and too-many c, got %+v", res.Errors)
	}
	// An empty choice that requires one branch.
	res, err = s.Validate(`<ch/>`)
	if err != nil || res.Valid {
		t.Fatalf("empty choice: %v %v", res, err)
	}
	if res.Errors[0].Kind != "missing" {
		t.Errorf("empty choice kind = %q, want missing", res.Errors[0].Kind)
	}
}

// msg:message on element declarations and groups replaces the generic
// content-model diagnostic for the matching Kind (on="missing"/"too-many"),
// chosen by language like assertion messages.
func TestCardinalityMessages(t *testing.T) {
	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:msg="` + MessagesNS + `">
	  <xs:element name="s"><xs:complexType><xs:sequence>
	    <xs:element name="a"><xs:annotation><xs:appinfo>
	      <msg:message on="missing" rule="R1" xml:lang="en">a is required</msg:message>
	      <msg:message on="missing" rule="R1" xml:lang="de">a fehlt</msg:message>
	      <msg:message on="too-many" rule="R2">only one a</msg:message>
	    </xs:appinfo></xs:annotation></xs:element>
	    <xs:element name="b" minOccurs="0"/>
	  </xs:sequence>
	  <xs:attribute name="k" use="required"><xs:annotation><xs:appinfo>
	    <msg:message on="missing" rule="R3" xml:lang="fr">k manque</msg:message>
	    <msg:message on="missing" rule="R3">k missing</msg:message>
	  </xs:appinfo></xs:annotation></xs:attribute>
	  </xs:complexType></xs:element>
	</xs:schema>`
	s, err := Compile([]string{schema}, "", Version11)
	if err != nil {
		t.Fatal(err)
	}
	find := func(errs []Error, rule string) *Error {
		for i := range errs {
			if errs[i].Rule == rule {
				return &errs[i]
			}
		}
		return nil
	}
	for _, tc := range []struct {
		inst, lang, rule, text string
	}{
		{`<s k="1"><b/></s>`, "de", "R1", "a fehlt"},
		{`<s k="1"><b/></s>`, "de-CH", "R1", "a fehlt"},
		{`<s k="1"><b/></s>`, "en", "R1", "a is required"},
		{`<s k="1"><a/><a/></s>`, "en", "R2", "only one a"},
		{`<s><a/></s>`, "fr", "R3", "k manque"},
		{`<s><a/></s>`, "it", "R3", "k missing"},
	} {
		res, err := s.ValidateLang(tc.inst, tc.lang)
		if err != nil {
			t.Fatal(err)
		}
		if res.Valid {
			t.Fatalf("%s valid", tc.inst)
		}
		e := find(res.Errors, tc.rule)
		if e == nil || e.Message != tc.text {
			t.Errorf("%s lang=%s: want rule %s %q, got %+v", tc.inst, tc.lang, tc.rule, tc.text, res.Errors)
		}
	}
}
