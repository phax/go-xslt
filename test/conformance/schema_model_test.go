package conformance

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/xsd"
)

// TestSchemaModelWalksTheXSDCorpus compiles every schema document of the W3C XSD
// test suite that the compiler accepts (as a stand-alone entry schema, XSD 1.1
// so the superset of constructs is allowed) and walks the whole public
// component model of the result: every global element, type and attribute,
// every content model, particle, attribute use and substitution group, with a
// guard against the cycles recursive types make. It checks nothing about the
// values — only that the model can be walked for any schema the compiler
// accepts, without a panic or a nil where the API promises a value.
func TestSchemaModelWalksTheXSDCorpus(t *testing.T) {
	root := filepath.Join(repoRoot(), "test", "conformance", "xsdtests")
	if _, err := os.Stat(root); err != nil {
		t.Skip("xsdtests suite not present")
	}
	var files []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".xsd") {
			files = append(files, p)
		}
		return nil
	})
	compiled, walked := 0, 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		s, _ := xsd.Compile(xsd.Request{Schemas: []string{string(b)}, BaseDir: filepath.Dir(f), Version: xsd.V11})
		if s == nil {
			continue
		}
		compiled++
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: model walk panicked: %v", f, r)
				}
			}()
			walkSchemaModel(t, f, s)
			walked++
		}()
	}
	t.Logf("%d schema documents, %d compiled, %d walked", len(files), compiled, walked)
	if compiled == 0 {
		t.Fatal("no schema compiled")
	}
}

func walkSchemaModel(t *testing.T, file string, s *xsd.Schema) {
	seenType := map[xsd.TypeDefinition]bool{}
	seenEl := map[*xsd.Element]bool{}
	var typ func(td xsd.TypeDefinition)
	var el func(e *xsd.Element)
	var particle func(p *xsd.Particle)
	particle = func(p *xsd.Particle) {
		if p == nil || p.Term == nil {
			t.Errorf("%s: nil particle/term", file)
			return
		}
		if p.MinOccurs < 0 || (p.MaxOccurs != xsd.Unbounded && p.MaxOccurs < p.MinOccurs) {
			t.Errorf("%s: bad occurs %d..%d", file, p.MinOccurs, p.MaxOccurs)
		}
		switch tm := p.Term.(type) {
		case *xsd.Element:
			el(tm)
		case *xsd.ModelGroup:
			for _, c := range tm.Particles {
				particle(c)
			}
		case *xsd.Wildcard:
		default:
			t.Errorf("%s: unknown term %T", file, tm)
		}
	}
	typ = func(td xsd.TypeDefinition) {
		if td == nil || seenType[td] {
			return
		}
		seenType[td] = true
		_ = td.Name()
		_ = td.BuiltinBase()
		switch tt := td.(type) {
		case *xsd.SimpleType:
			_ = tt.Variety()
			_ = tt.Enumeration()
			typ(tt.Base())
			typ(tt.ItemType())
			for _, m := range tt.MemberTypes() {
				typ(m)
			}
		case *xsd.ComplexType:
			_ = tt.Mixed()
			_ = tt.ContentKind()
			_ = tt.Derivation()
			_ = tt.AttributeWildcard()
			if sc := tt.SimpleContent(); sc != nil {
				typ(sc)
			}
			typ(tt.Base())
			if cm := tt.ContentModel(); cm != nil {
				particle(cm)
			}
			_ = tt.ContentParticles()
			for _, a := range tt.Attributes() {
				if a.Type != nil {
					typ(a.Type)
				}
			}
		}
	}
	el = func(e *xsd.Element) {
		if e == nil || seenEl[e] {
			return
		}
		seenEl[e] = true
		_ = e.Name()
		_ = e.SubstitutionGroups()
		if e.Type() == nil {
			t.Errorf("%s: element %v has no Type()", file, e.Name())
			return
		}
		typ(e.Type())
	}
	for _, e := range s.GlobalElements() {
		el(e)
		_ = s.SubstitutionMembers(xml.Name{Space: e.Name().Space, Local: e.Name().Local})
	}
	for _, td := range s.GlobalTypes() {
		typ(td)
	}
	for _, a := range s.GlobalAttributes() {
		if a.Type() != nil {
			typ(a.Type())
		}
	}
	_ = s.Unresolved()
}
