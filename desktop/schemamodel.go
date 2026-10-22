package main

import (
	"encoding/xml"
	"strconv"

	"github.com/tim-riep/go-xslt/engine"
	"github.com/tim-riep/go-xslt/xsd"
)

// SchemaModelRequest asks for the resolved structure of a schema: Schemas[0]
// is the entry document, relative xs:import/xs:include locations resolve
// against BaseDir. Schema files are read only inside ProjectDir (BaseDir
// itself when ProjectDir is empty): a schema cannot pull in files from
// elsewhere on the disk.
type SchemaModelRequest struct {
	Schemas    []string `json:"schemas"`
	BaseDir    string   `json:"baseDir"`
	ProjectDir string   `json:"projectDir"`
	// Version is "1.0" (default) or "1.1".
	Version string `json:"version"`
}

// SMNode is one term of a content model: an element, a sequence/choice/all
// group (Children) or an xs:any wildcard. Max is -1 for "unbounded".
type SMNode struct {
	Kind     string    `json:"kind"` // element | sequence | choice | all | any
	Name     string    `json:"name,omitempty"`
	NS       string    `json:"ns,omitempty"`
	TypeID   string    `json:"typeId,omitempty"`
	TypeName string    `json:"typeName,omitempty"`
	Min      int       `json:"min"`
	Max      int       `json:"max"`
	Nillable bool      `json:"nillable,omitempty"`
	Abstract bool      `json:"abstract,omitempty"`
	Default  string    `json:"default,omitempty"`
	Fixed    string    `json:"fixed,omitempty"`
	Subst    []string  `json:"substitutes,omitempty"`
	Mode     string    `json:"mode,omitempty"`
	Process  string    `json:"process,omitempty"`
	Children []*SMNode `json:"children,omitempty"`
}

// SMAttr is an attribute declaration or use.
type SMAttr struct {
	Name     string `json:"name"`
	NS       string `json:"ns,omitempty"`
	TypeID   string `json:"typeId,omitempty"`
	TypeName string `json:"typeName,omitempty"`
	Use      string `json:"use,omitempty"`
	Default  string `json:"default,omitempty"`
	Fixed    string `json:"fixed,omitempty"`
}

// SMType is a simple or complex type, listed once however often it is used —
// elements refer to it by TypeID, so a schema that reuses a type thousands of
// times stays small and the view expands it on demand.
type SMType struct {
	ID         string   `json:"id"`
	Name       string   `json:"name,omitempty"`
	NS         string   `json:"ns,omitempty"`
	Builtin    bool     `json:"builtin,omitempty"`
	Simple     bool     `json:"simple"`
	Abstract   bool     `json:"abstract,omitempty"`
	Mixed      bool     `json:"mixed,omitempty"`
	Derivation string   `json:"derivation,omitempty"`
	BaseID     string   `json:"baseId,omitempty"`
	BaseName   string   `json:"baseName,omitempty"`
	Content    string   `json:"content,omitempty"` // empty | simple | elementOnly | mixed
	Model      *SMNode  `json:"model,omitempty"`
	Attributes []SMAttr `json:"attributes,omitempty"`
	AnyAttr    string   `json:"anyAttribute,omitempty"`
	// Simple types and simple content.
	Variety     string   `json:"variety,omitempty"`
	Primitive   string   `json:"primitive,omitempty"`
	Enumeration []string `json:"enumeration,omitempty"`
	ItemID      string   `json:"itemId,omitempty"`
	ItemName    string   `json:"itemName,omitempty"`
	Members     []string `json:"members,omitempty"` // type ids
	ContentID   string   `json:"contentId,omitempty"`
}

// SchemaModel is the resolved structure of a schema, across all its files.
type SchemaModel struct {
	Elements    []*SMNode           `json:"elements"`
	Attributes  []SMAttr            `json:"attributes"`
	GlobalTypes []string            `json:"globalTypes"` // type ids, in order
	Types       map[string]*SMType  `json:"types"`
	Diagnostics []engine.Diagnostic `json:"diagnostics"`
}

// SchemaStructure compiles the schema set and returns its component model for
// the structure view.
func (s *XsltService) SchemaStructure(req SchemaModelRequest) SchemaModel {
	v := xsd.V10
	if req.Version == "1.1" {
		v = xsd.V11
	}
	var sec xsd.Security
	if req.ProjectDir != "" {
		sec.AllowedDirs = []string{req.ProjectDir}
	}
	sch, diags := xsd.Compile(xsd.Request{Schemas: req.Schemas, BaseDir: req.BaseDir, Version: v, Security: sec})
	out := SchemaModel{Elements: []*SMNode{}, Attributes: []SMAttr{}, GlobalTypes: []string{},
		Types: map[string]*SMType{}, Diagnostics: diags}
	if out.Diagnostics == nil {
		out.Diagnostics = []engine.Diagnostic{}
	}
	if sch == nil {
		return out
	}
	b := &modelBuilder{out: &out, ids: map[xsd.TypeDefinition]string{}}
	for _, e := range sch.GlobalElements() {
		out.Elements = append(out.Elements, b.element(e, 1, 1))
	}
	for _, a := range sch.GlobalAttributes() {
		def, hasDef := a.Default()
		fix, hasFix := a.Fixed()
		at := SMAttr{Name: a.Name().Local, NS: a.Name().Space}
		if t := a.Type(); t != nil {
			at.TypeID, at.TypeName = b.typeRef(t)
		}
		if hasDef {
			at.Default = def
		}
		if hasFix {
			at.Fixed = fix
		}
		out.Attributes = append(out.Attributes, at)
	}
	for _, t := range sch.GlobalTypes() {
		id, _ := b.typeRef(t)
		out.GlobalTypes = append(out.GlobalTypes, id)
	}
	return out
}

type modelBuilder struct {
	out  *SchemaModel
	ids  map[xsd.TypeDefinition]string
	anon int
}

func nameKey(n xml.Name) string { return "{" + n.Space + "}" + n.Local }

// typeRef registers t (and, transitively, everything it refers to) and returns
// its id and display name.
func (b *modelBuilder) typeRef(t xsd.TypeDefinition) (id, name string) {
	if t == nil {
		return "", ""
	}
	n := t.Name()
	if id, ok := b.ids[t]; ok {
		return id, n.Local
	}
	if n.Local != "" {
		id = nameKey(n)
	} else {
		b.anon++
		id = "#anon" + strconv.Itoa(b.anon)
	}
	b.ids[t] = id
	if _, done := b.out.Types[id]; done {
		return id, n.Local
	}
	st := &SMType{ID: id, Name: n.Local, NS: n.Space, Simple: t.IsSimple()}
	b.out.Types[id] = st // before recursing: types may refer to themselves
	switch t := t.(type) {
	case *xsd.SimpleType:
		st.Builtin = n.Space == "http://www.w3.org/2001/XMLSchema"
		b.simple(st, t)
	case *xsd.ComplexType:
		b.complex(st, t)
	}
	return id, n.Local
}

func (b *modelBuilder) simple(st *SMType, t *xsd.SimpleType) {
	st.Variety = string(t.Variety())
	st.Primitive = t.BuiltinBase().Local
	st.Enumeration = t.Enumeration()
	if base := t.Base(); base != nil {
		st.Derivation = "restriction"
		st.BaseID, st.BaseName = b.typeRef(base)
	}
	if it := t.ItemType(); it != nil {
		st.ItemID, st.ItemName = b.typeRef(it)
	}
	for _, m := range t.MemberTypes() {
		id, _ := b.typeRef(m)
		st.Members = append(st.Members, id)
	}
}

func (b *modelBuilder) complex(st *SMType, t *xsd.ComplexType) {
	st.Abstract = t.Abstract()
	st.Mixed = t.Mixed()
	st.Derivation = t.Derivation()
	if base := t.Base(); base != nil {
		st.BaseID, st.BaseName = b.typeRef(base)
	}
	switch t.ContentKind() {
	case xsd.ContentSimple:
		st.Content = "simple"
		if sc := t.SimpleContent(); sc != nil {
			st.ContentID, _ = b.typeRef(sc)
			st.Primitive = sc.BuiltinBase().Local
		}
	case xsd.ContentElementOnly:
		st.Content = "elementOnly"
	case xsd.ContentMixed:
		st.Content = "mixed"
	default:
		st.Content = "empty"
	}
	if m := t.ContentModel(); m != nil {
		st.Model = b.particle(m)
	}
	for _, a := range t.Attributes() {
		at := SMAttr{Name: a.Name.Local, NS: a.Name.Space, Use: string(a.Use)}
		if a.Type != nil {
			at.TypeID, at.TypeName = b.typeRef(a.Type)
		}
		if a.HasDefault {
			at.Default = a.Default
		}
		if a.HasFixed {
			at.Fixed = a.Fixed
		}
		st.Attributes = append(st.Attributes, at)
	}
	if w := t.AttributeWildcard(); w != nil {
		st.AnyAttr = w.Mode + "/" + w.ProcessContents
	}
}

func (b *modelBuilder) element(e *xsd.Element, min, max int) *SMNode {
	n := &SMNode{Kind: "element", Name: e.Name().Local, NS: e.Name().Space, Min: min, Max: max,
		Nillable: e.Nillable(), Abstract: e.Abstract()}
	if t := e.Type(); t != nil {
		n.TypeID, n.TypeName = b.typeRef(t)
	}
	if d, ok := e.Default(); ok {
		n.Default = d
	}
	if f, ok := e.Fixed(); ok {
		n.Fixed = f
	}
	for _, g := range e.SubstitutionGroups() {
		n.Subst = append(n.Subst, g.Local)
	}
	return n
}

func (b *modelBuilder) particle(p *xsd.Particle) *SMNode {
	switch t := p.Term.(type) {
	case *xsd.Element:
		return b.element(t, p.MinOccurs, p.MaxOccurs)
	case *xsd.ModelGroup:
		n := &SMNode{Kind: string(t.Compositor), Min: p.MinOccurs, Max: p.MaxOccurs}
		for _, c := range t.Particles {
			n.Children = append(n.Children, b.particle(c))
		}
		return n
	case *xsd.Wildcard:
		return &SMNode{Kind: "any", Min: p.MinOccurs, Max: p.MaxOccurs, Mode: t.Mode, Process: t.ProcessContents}
	}
	return &SMNode{Kind: "any", Min: p.MinOccurs, Max: p.MaxOccurs}
}
