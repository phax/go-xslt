package xsd

import (
	"encoding/xml"
	"sort"
	"sync"

	"github.com/tim-riep/go-xslt/internal/xpath"
)

// This file is the read-only schema component model that the public package
// github.com/tim-riep/go-xslt/xsd re-exports (as type aliases). The *View types
// wrap the compiler's own components, which stay free to change: only what is
// exposed here is API.
//
// A SchemaView is immutable and safe for concurrent use. Wrappers are created
// lazily and cached, so the same component always yields the same *View — a
// Particle's element term is == to the one GlobalElements returned for a ref.

// Variety of a simple type.
type Variety string

const (
	VarietyAtomic Variety = "atomic"
	VarietyList   Variety = "list"
	VarietyUnion  Variety = "union"
)

// ContentKind of a complex type.
type ContentKind string

const (
	ContentEmpty       ContentKind = "empty"
	ContentSimple      ContentKind = "simple"
	ContentElementOnly ContentKind = "elementOnly"
	ContentMixed       ContentKind = "mixed"
)

// Compositor of a model group.
type Compositor string

const (
	CompositorSequence Compositor = "sequence"
	CompositorChoice   Compositor = "choice"
	CompositorAll      Compositor = "all"
)

// Use of an attribute in a complex type.
type Use string

const (
	UseOptional   Use = "optional"
	UseRequired   Use = "required"
	UseProhibited Use = "prohibited"
)

// Unbounded is a Particle's MaxOccurs for maxOccurs="unbounded".
const Unbounded = -1

// TypeDefinitionView is a simple or complex type definition.
type TypeDefinitionView interface {
	// Name is the expanded name; the zero xml.Name for an anonymous type.
	Name() xml.Name
	// IsSimple reports a simple type (atomic, list or union).
	IsSimple() bool
	// BuiltinBase is the built-in type this type is, or is derived from through
	// restriction steps: {xs}decimal for a restriction of a restriction of
	// xs:decimal, {xs}positiveInteger for xs:positiveInteger itself. For a list
	// it is the item type's, for a union and for a complex type without simple
	// content it is {xs}anySimpleType / {xs}anyType respectively — see
	// SimpleTypeView.MemberTypes for the union's members.
	BuiltinBase() xml.Name
}

// TermView is an element declaration, a model group or a wildcard.
type TermView interface{ isTermView() }

func (*ElementView) isTermView()    {}
func (*ModelGroupView) isTermView() {}
func (*WildcardView) isTermView()   {}

// SchemaView is a compiled schema (the union of the documents given to Compile
// and everything they include/import/redefine/override).
type SchemaView struct {
	s *Schema

	mu    sync.Mutex
	elems map[*ElementDecl]*ElementView
	simp  map[*SimpleType]*SimpleTypeView
	cplx  map[*ComplexType]*ComplexTypeView
	attrs map[*AttributeDecl]*AttributeDeclView
	any   *ComplexTypeView
}

// View returns the read-only component model of a compiled schema.
func (s *Schema) View() *SchemaView {
	return &SchemaView{s: s,
		elems: map[*ElementDecl]*ElementView{}, simp: map[*SimpleType]*SimpleTypeView{},
		cplx: map[*ComplexType]*ComplexTypeView{}, attrs: map[*AttributeDecl]*AttributeDeclView{}}
}

func qn(n xname) xml.Name { return xml.Name{Space: n.Space, Local: n.Local} }

func sortedNames[T any](m map[xname]T) []xname {
	ks := make([]xname, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool {
		if ks[i].Space != ks[j].Space {
			return ks[i].Space < ks[j].Space
		}
		return ks[i].Local < ks[j].Local
	})
	return ks
}

// GlobalElements are the global element declarations, ordered by namespace
// then local name.
func (v *SchemaView) GlobalElements() []*ElementView {
	var out []*ElementView
	for _, k := range sortedNames(v.s.elements) {
		out = append(out, v.element(v.s.elements[k]))
	}
	return out
}

// GlobalTypes are the named type definitions (user-defined and built-in ones
// the schema declares), ordered by namespace then local name.
func (v *SchemaView) GlobalTypes() []TypeDefinitionView {
	var out []TypeDefinitionView
	for _, k := range sortedNames(v.s.types) {
		if t := v.typeDef(v.s.types[k]); t != nil {
			out = append(out, t)
		}
	}
	return out
}

// GlobalAttributes are the global attribute declarations, ordered by name.
func (v *SchemaView) GlobalAttributes() []*AttributeDeclView {
	var out []*AttributeDeclView
	for _, k := range sortedNames(v.s.attributes) {
		out = append(out, v.attr(v.s.attributes[k]))
	}
	return out
}

// Element returns the global element declaration with the given name, or nil.
func (v *SchemaView) Element(name xml.Name) *ElementView {
	if d := v.s.elements[xname{name.Space, name.Local}]; d != nil {
		return v.element(d)
	}
	return nil
}

// Type returns the named type definition (a schema type or a built-in one), or nil.
func (v *SchemaView) Type(name xml.Name) TypeDefinitionView {
	if t, ok := v.s.types[xname{name.Space, name.Local}]; ok {
		return v.typeDef(t)
	}
	if name.Space == xsNS {
		if at, ok := xpath.AtomTypeByName(name.Local); ok {
			return v.simple(&SimpleType{variety: vAtomic, baseBuiltin: true, prim: at})
		}
		if name.Local == "anyType" {
			return v.anyType()
		}
	}
	return nil
}

// SubstitutionMembers are the global elements that may substitute for head,
// transitively (members of members), head itself excluded, in name order.
func (v *SchemaView) SubstitutionMembers(head xml.Name) []*ElementView {
	seen := map[*ElementDecl]bool{}
	var collect func(h xname)
	var decls []*ElementDecl
	collect = func(h xname) {
		for _, m := range v.s.substMembers[h] {
			if seen[m] {
				continue
			}
			seen[m] = true
			decls = append(decls, m)
			collect(m.name)
		}
	}
	collect(xname{head.Space, head.Local})
	sort.Slice(decls, func(i, j int) bool {
		a, b := decls[i].name, decls[j].name
		if a.Space != b.Space {
			return a.Space < b.Space
		}
		return a.Local < b.Local
	})
	out := make([]*ElementView, 0, len(decls))
	for _, d := range decls {
		out = append(out, v.element(d))
	}
	return out
}

func (v *SchemaView) element(d *ElementDecl) *ElementView {
	v.mu.Lock()
	defer v.mu.Unlock()
	if e, ok := v.elems[d]; ok {
		return e
	}
	e := &ElementView{v: v, d: d}
	v.elems[d] = e
	return e
}

func (v *SchemaView) attr(d *AttributeDecl) *AttributeDeclView {
	v.mu.Lock()
	defer v.mu.Unlock()
	if a, ok := v.attrs[d]; ok {
		return a
	}
	a := &AttributeDeclView{v: v, d: d}
	v.attrs[d] = a
	return a
}

func (v *SchemaView) simple(t *SimpleType) *SimpleTypeView {
	if t == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if w, ok := v.simp[t]; ok {
		return w
	}
	w := &SimpleTypeView{v: v, t: t}
	v.simp[t] = w
	return w
}

func (v *SchemaView) complexType(t *ComplexType) *ComplexTypeView {
	v.mu.Lock()
	defer v.mu.Unlock()
	if w, ok := v.cplx[t]; ok {
		return w
	}
	w := &ComplexTypeView{v: v, t: t}
	v.cplx[t] = w
	return w
}

// anyType is the ur-type, standing in for an element declared without a type.
func (v *SchemaView) anyType() *ComplexTypeView {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.any == nil {
		v.any = &ComplexTypeView{v: v, t: &ComplexType{name: xname{xsNS, "anyType"}, kind: contentMixed,
			particle: &particle{min: 1, max: 1, term: &modelGroup{compositor: cSeq, particles: []*particle{
				{min: 0, max: unbounded, term: &wildcard{nsMode: "any", process: "lax"}}}}},
			attrWildcard: &wildcard{nsMode: "any", process: "lax"}}, builtin: true}
	}
	return v.any
}

func (v *SchemaView) typeDef(t Type) TypeDefinitionView {
	switch t := t.(type) {
	case *SimpleType:
		return v.simple(t)
	case *ComplexType:
		return v.complexType(t)
	}
	return nil
}

// ElementView is an element declaration.
type ElementView struct {
	v *SchemaView
	d *ElementDecl
}

func (e *ElementView) Name() xml.Name { return qn(e.d.name) }

// Type is the declared type; xs:anyType when the declaration names none.
func (e *ElementView) Type() TypeDefinitionView {
	if t := e.v.typeDef(e.d.typ); t != nil {
		return t
	}
	return e.v.anyType()
}
func (e *ElementView) Nillable() bool { return e.d.nillable }
func (e *ElementView) Abstract() bool { return e.d.abstract }

// SubstitutionGroups are the heads this element is a member of.
func (e *ElementView) SubstitutionGroups() []xml.Name {
	out := make([]xml.Name, 0, len(e.d.substGroups))
	for _, g := range e.d.substGroups {
		out = append(out, qn(g))
	}
	return out
}
func (e *ElementView) Default() (string, bool) { return e.d.def, e.d.hasDefault }
func (e *ElementView) Fixed() (string, bool)   { return e.d.fixed, e.d.hasFixed }

// SimpleTypeView is a simple type definition.
type SimpleTypeView struct {
	v *SchemaView
	t *SimpleType
}

func (t *SimpleTypeView) IsSimple() bool { return true }

// Name is the type's expanded name; for a reference to a built-in type that is
// the built-in's own ({xs}decimal); empty for an anonymous type.
func (t *SimpleTypeView) Name() xml.Name {
	if !t.t.name.zero() {
		return qn(t.t.name)
	}
	if t.t.baseBuiltin && !t.t.declared && t.t.variety == vAtomic {
		return t.BuiltinBase()
	}
	return xml.Name{}
}

func builtinName(at xpath.AtomType) xml.Name {
	n := at.String() // "xs:decimal"
	if len(n) > 3 && n[:3] == "xs:" {
		n = n[3:]
	}
	return xml.Name{Space: xsNS, Local: n}
}

func (t *SimpleTypeView) BuiltinBase() xml.Name {
	switch t.t.variety {
	case vList:
		if t.t.item != nil {
			return (&SimpleTypeView{v: t.v, t: t.t.item}).BuiltinBase()
		}
		return xml.Name{Space: xsNS, Local: "anySimpleType"}
	case vUnion:
		return xml.Name{Space: xsNS, Local: "anySimpleType"}
	}
	at := t.t.resolvePrim()
	if at == xpath.XSanyAtomicType {
		return xml.Name{Space: xsNS, Local: "anySimpleType"}
	}
	return builtinName(at)
}

func (t *SimpleTypeView) Variety() Variety {
	switch t.t.variety {
	case vList:
		return VarietyList
	case vUnion:
		return VarietyUnion
	}
	return VarietyAtomic
}

// Base is the type this one restricts; nil when it is a built-in or derives
// directly from one.
func (t *SimpleTypeView) Base() TypeDefinitionView {
	if t.t.base == nil {
		return nil
	}
	return t.v.simple(t.t.base)
}

// ItemType is a list's item type; nil otherwise.
func (t *SimpleTypeView) ItemType() TypeDefinitionView {
	if t.t.item == nil {
		return nil
	}
	return t.v.simple(t.t.item)
}

// MemberTypes are a union's member types; nil otherwise.
func (t *SimpleTypeView) MemberTypes() []TypeDefinitionView {
	var out []TypeDefinitionView
	for _, m := range t.t.members {
		out = append(out, t.v.simple(m))
	}
	return out
}

// Enumeration are the enumeration facet values declared along the restriction
// chain (the nearest step that declares any), in document order.
func (t *SimpleTypeView) Enumeration() []string {
	for s, n := t.t, 0; s != nil && n < 64; s, n = s.base, n+1 {
		if s.facets.hasEnum {
			return append([]string{}, s.facets.enumeration...)
		}
	}
	return nil
}

// ComplexTypeView is a complex type definition.
type ComplexTypeView struct {
	v       *SchemaView
	t       *ComplexType
	builtin bool
}

func (t *ComplexTypeView) IsSimple() bool { return false }
func (t *ComplexTypeView) Name() xml.Name { return qn(t.t.name) }

// BuiltinBase is the built-in simple type of a type with simple content
// (through extension and restriction), and {xs}anyType otherwise.
func (t *ComplexTypeView) BuiltinBase() xml.Name {
	if st := t.SimpleContent(); st != nil {
		return st.BuiltinBase()
	}
	return xml.Name{Space: xsNS, Local: "anyType"}
}

func (t *ComplexTypeView) Abstract() bool { return t.t.abstract }

// Base is the type this one is derived from; nil for xs:anyType and types with
// no explicit base.
func (t *ComplexTypeView) Base() TypeDefinitionView {
	if t.t.baseType == nil {
		return nil
	}
	return t.v.typeDef(t.t.baseType)
}

// Derivation is "extension", "restriction" or "" (no derivation).
func (t *ComplexTypeView) Derivation() string { return t.t.derivation }

func (t *ComplexTypeView) ContentKind() ContentKind {
	switch t.t.kind {
	case contentSimple:
		return ContentSimple
	case contentElementOnly:
		return ContentElementOnly
	case contentMixed:
		return ContentMixed
	}
	return ContentEmpty
}

// Mixed reports mixed content (character data between the child elements).
func (t *ComplexTypeView) Mixed() bool { return t.t.kind == contentMixed }

// SimpleContent is the simple type of the character content, nil unless the
// type has simple content (it then includes what the type inherits).
func (t *ComplexTypeView) SimpleContent() *SimpleTypeView {
	for c, n := t.t, 0; c != nil && n < 64; n++ {
		if c.kind == contentSimple && c.simpleType != nil {
			return t.v.simple(c.simpleType)
		}
		b, ok := c.baseType.(*ComplexType)
		if !ok {
			return nil
		}
		c = b
	}
	return nil
}

// ContentModel is the effective (after extension) content model, nil when the
// type has no element content.
func (t *ComplexTypeView) ContentModel() *ParticleView {
	p := t.v.s.effectiveParticle(t.t)
	if p == nil {
		return nil
	}
	return t.v.particle(p)
}

// ContentParticles are the particles of the content model's outermost group —
// or, if the model is a single element or wildcard, that one particle.
func (t *ComplexTypeView) ContentParticles() []*ParticleView {
	root := t.ContentModel()
	if root == nil {
		return nil
	}
	if g, ok := root.Term.(*ModelGroupView); ok && root.MinOccurs == 1 && root.MaxOccurs == 1 {
		return g.Particles
	}
	return []*ParticleView{root}
}

// Attributes are the effective attribute uses (after extension and restriction).
func (t *ComplexTypeView) Attributes() []*AttributeUseView {
	uses, _ := t.v.s.effectiveAttrs(t.t)
	out := make([]*AttributeUseView, 0, len(uses))
	for _, u := range uses {
		use := UseOptional
		switch {
		case u.prohibited:
			use = UseProhibited
		case u.required:
			use = UseRequired
		}
		def, hasDef := u.def, u.hasDefault
		fix, hasFix := u.fixed, u.hasFixed
		if !hasDef && !hasFix {
			def, hasDef, fix, hasFix = u.decl.def, u.decl.hasDefault, u.decl.fixed, u.decl.hasFixed
		}
		out = append(out, &AttributeUseView{Name: qn(u.decl.name), Type: t.v.simple(u.decl.typ), Use: use,
			Default: def, HasDefault: hasDef, Fixed: fix, HasFixed: hasFix})
	}
	return out
}

// AttributeWildcard is the effective xs:anyAttribute, nil when there is none.
func (t *ComplexTypeView) AttributeWildcard() *WildcardView {
	_, w := t.v.s.effectiveAttrs(t.t)
	if w == nil {
		return nil
	}
	return t.v.wildcard(w)
}

// ParticleView is a term with occurrence bounds.
type ParticleView struct {
	MinOccurs int
	MaxOccurs int // Unbounded (-1) for maxOccurs="unbounded"
	Term      TermView
}

// ModelGroupView is a sequence, choice or all group.
type ModelGroupView struct {
	Compositor Compositor
	Particles  []*ParticleView
}

// WildcardView is an xs:any or xs:anyAttribute.
type WildcardView struct {
	// Namespaces is "##any", "##other", or "##targetNamespace"-resolved URIs;
	// "" in the list stands for ##local.
	Mode            string // "any", "other" or "list"
	Namespaces      []string
	ProcessContents string // "strict", "lax" or "skip"
}

// AttributeDeclView is a global attribute declaration.
type AttributeDeclView struct {
	v *SchemaView
	d *AttributeDecl
}

func (a *AttributeDeclView) Name() xml.Name          { return qn(a.d.name) }
func (a *AttributeDeclView) Type() *SimpleTypeView   { return a.v.simple(a.d.typ) }
func (a *AttributeDeclView) Default() (string, bool) { return a.d.def, a.d.hasDefault }
func (a *AttributeDeclView) Fixed() (string, bool)   { return a.d.fixed, a.d.hasFixed }

// AttributeUseView is an attribute of a complex type.
type AttributeUseView struct {
	Name       xml.Name
	Type       *SimpleTypeView
	Use        Use
	Default    string
	HasDefault bool
	Fixed      string
	HasFixed   bool
}

func (v *SchemaView) particle(p *particle) *ParticleView {
	out := &ParticleView{MinOccurs: p.min, MaxOccurs: p.max}
	switch t := p.term.(type) {
	case *ElementDecl:
		out.Term = v.element(t)
	case *modelGroup:
		g := &ModelGroupView{Compositor: CompositorSequence}
		switch t.compositor {
		case cChoice:
			g.Compositor = CompositorChoice
		case cAll:
			g.Compositor = CompositorAll
		}
		for _, c := range t.particles {
			g.Particles = append(g.Particles, v.particle(c))
		}
		out.Term = g
	case *wildcard:
		out.Term = v.wildcard(t)
	}
	return out
}

func (v *SchemaView) wildcard(w *wildcard) *WildcardView {
	mode := w.nsMode
	if mode == "" {
		mode = "any"
	}
	return &WildcardView{Mode: mode, Namespaces: append([]string{}, w.namespaces...), ProcessContents: w.process}
}

// Unresolved lists the declarations whose type (or list item / union member /
// base) reference names a type the schema does not define. The compiler accepts
// such a schema — XSD's test suite expects the error at validation time — but a
// tool walking the model must be told, since Type() of such a declaration has
// nothing to answer with.
func (v *SchemaView) Unresolved() []string {
	var out []string
	seenT := map[any]bool{}
	var walkType func(t Type, where string)
	var walkSimple func(t *SimpleType, where string)
	var walkParticle func(p *particle, where string)
	walkSimple = func(t *SimpleType, where string) {
		if t == nil {
			return
		}
		if t == absentType {
			out = append(out, where)
			return
		}
		if seenT[t] {
			return
		}
		seenT[t] = true
		walkSimple(t.base, where)
		walkSimple(t.item, where)
		for _, m := range t.members {
			walkSimple(m, where)
		}
	}
	walkParticle = func(p *particle, where string) {
		if p == nil {
			return
		}
		switch t := p.term.(type) {
		case *ElementDecl:
			if !seenT[t] {
				seenT[t] = true
				walkType(t.typ, "element "+t.name.String())
			}
		case *modelGroup:
			for _, c := range t.particles {
				walkParticle(c, where)
			}
		}
	}
	walkType = func(t Type, where string) {
		switch t := t.(type) {
		case *SimpleType:
			walkSimple(t, where)
		case *ComplexType:
			if seenT[t] {
				return
			}
			seenT[t] = true
			where := "type " + t.name.String()
			if t.name.zero() {
				where = "an anonymous complex type"
			}
			walkType(t.baseType, where)
			walkSimple(t.simpleType, where)
			walkParticle(t.particle, where)
			for _, u := range t.attrUses {
				walkSimple(u.decl.typ, "attribute "+u.decl.name.String())
			}
		}
	}
	for _, k := range sortedNames(v.s.elements) {
		d := v.s.elements[k]
		seenT[d] = true
		walkType(d.typ, "element "+d.name.String())
	}
	for _, k := range sortedNames(v.s.types) {
		walkType(v.s.types[k], "type "+k.String())
	}
	for _, k := range sortedNames(v.s.attributes) {
		walkSimple(v.s.attributes[k].typ, "attribute "+k.String())
	}
	sort.Strings(out)
	dedup := out[:0]
	for i, s := range out {
		if i == 0 || s != out[i-1] {
			dedup = append(dedup, s)
		}
	}
	return dedup
}
