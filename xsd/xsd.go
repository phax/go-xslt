// Package xsd is the public, read-only schema component model of the goxslt
// XSD 1.0/1.1 processor: compile schema documents once and walk the resulting
// element declarations, type definitions, content models and attributes —
// without an instance document. It exists for tools that check things against a
// schema statically (for example that an XPath binding in a template names a
// path the data schema allows, has a matching type and a matching cardinality).
//
//	schema, diags := xsd.Compile(xsd.Request{Schemas: docs, BaseDir: dir, Version: xsd.V10})
//	for _, el := range schema.GlobalElements() {
//	    ct, ok := el.Type().(*xsd.ComplexType)
//	    ...
//	}
//
// A *Schema is immutable and safe for concurrent use. The types here are views
// onto the compiler's own components, which are not part of the API.
//
// To validate instance documents use package engine's Validate.
package xsd

import (
	"errors"

	"github.com/tim-riep/go-xslt/internal/engine"
	"github.com/tim-riep/go-xslt/internal/xsd"
)

// Version selects XML Schema 1.0 or 1.1.
type Version int

const (
	// V10 is W3C XML Schema 1.0 (the default).
	V10 Version = iota
	// V11 is W3C XML Schema 1.1 (assertions, open content, conditional types).
	V11
)

// Request is the input of Compile.
type Request struct {
	// Schemas holds the schema document source texts. The first is the entry
	// schema; the rest are additional documents available to xs:import/include
	// that reference each other by namespace alone.
	Schemas []string
	// BaseDir resolves relative schemaLocations of xs:import / xs:include /
	// xs:redefine / xs:override across files. Empty disables disk resolution.
	BaseDir string
	Version Version
	// Security limits which schema files may be read; the zero value allows
	// files under BaseDir only (see engine.Security).
	Security Security
}

// Security is engine.Security: AllowedDirs widens, AllowAllFiles lifts the
// file restriction.
type Security = engine.Security

// Diagnostic is an error from Compile, with the position in the schema document
// where known (Line and Col are 1-based, 0 means unknown).
type Diagnostic = engine.Diagnostic

type (
	// Schema is a compiled schema. See Compile.
	Schema = xsd.SchemaView
	// Element is an element declaration; it is also a Term.
	Element = xsd.ElementView
	// Documentation is one xs:documentation (Lang, Source, Text).
	Documentation = xsd.Documentation
	// Facets are the effective constraining facets of a simple type.
	Facets = xsd.Facets
	// TypeDefinition is a simple or complex type: *SimpleType or *ComplexType.
	TypeDefinition = xsd.TypeDefinitionView
	// SimpleType is an atomic, list or union type.
	SimpleType = xsd.SimpleTypeView
	// ComplexType is a complex type definition.
	ComplexType = xsd.ComplexTypeView
	// Particle is a Term with its occurrence bounds.
	Particle = xsd.ParticleView
	// Term is an *Element, a *ModelGroup or a *Wildcard.
	Term = xsd.TermView
	// ModelGroup is a sequence, choice or all group.
	ModelGroup = xsd.ModelGroupView
	// Wildcard is an xs:any or xs:anyAttribute.
	Wildcard = xsd.WildcardView
	// AttributeDecl is a global attribute declaration.
	AttributeDecl = xsd.AttributeDeclView
	// AttributeUse is an attribute of a complex type.
	AttributeUse = xsd.AttributeUseView

	Variety     = xsd.Variety
	ContentKind = xsd.ContentKind
	Compositor  = xsd.Compositor
	Use         = xsd.Use
)

const (
	VarietyAtomic = xsd.VarietyAtomic
	VarietyList   = xsd.VarietyList
	VarietyUnion  = xsd.VarietyUnion

	ContentEmpty       = xsd.ContentEmpty
	ContentSimple      = xsd.ContentSimple
	ContentElementOnly = xsd.ContentElementOnly
	ContentMixed       = xsd.ContentMixed

	CompositorSequence = xsd.CompositorSequence
	CompositorChoice   = xsd.CompositorChoice
	CompositorAll      = xsd.CompositorAll

	UseOptional   = xsd.UseOptional
	UseRequired   = xsd.UseRequired
	UseProhibited = xsd.UseProhibited

	// Unbounded is a Particle's MaxOccurs for maxOccurs="unbounded".
	Unbounded = xsd.Unbounded
)

// Compile compiles the schema documents of req, following xs:import,
// xs:include, xs:redefine and xs:override across files (resolved against
// req.BaseDir). On failure the Schema is nil and the diagnostics say why; a
// construct the compiler does not support is reported as such rather than
// guessed at. A returned Schema may still come with warnings: references to
// undefined types (see Schema.Unresolved). It never panics.
func Compile(req Request) (schema *Schema, diags []Diagnostic) {
	defer func() {
		if r := recover(); r != nil {
			schema = nil
			diags = []Diagnostic{{Severity: engine.SeverityError, Code: "INTERNAL", Phase: "compile",
				Message: "internal schema compiler error"}}
		}
	}()
	v := xsd.Version10
	if req.Version == V11 {
		v = xsd.Version11
	}
	s, err := xsd.CompileWithDocsFS(req.Schemas, req.BaseDir, v, engine.FileGuard(req.Security, req.BaseDir))
	if err != nil {
		d := Diagnostic{Severity: engine.SeverityError, Phase: "compile", Message: err.Error()}
		var xe xsd.Error
		if errors.As(err, &xe) {
			d.Line, d.Col, d.Code = xe.Line, xe.Col, xe.Code
		}
		if errors.Is(err, xsd.ErrUnsupported) {
			d.Code = "UNSUPPORTED"
		}
		return nil, []Diagnostic{d}
	}
	view := s.View()
	// A reference to a type the schema never defines is not a compile error here
	// (the validator reports it when it is used), but a model walker has nothing
	// to read from such a declaration — say so, as warnings on a usable Schema.
	for _, u := range view.Unresolved() {
		diags = append(diags, Diagnostic{Severity: engine.SeverityWarning, Code: "src-resolve", Phase: "compile",
			Message: u + " refers to a type that is not defined in the schema"})
	}
	return view, diags
}
