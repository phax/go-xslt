package xslt

import "sync/atomic"

// The SCHEMA-AWARENESS CLAIM: what this run presents itself as, to the two
// places XSLT 3.0 lets a stylesheet ask — system-property('xsl:is-schema-aware')
// and the XTSE1660 family of "that construct needs a schema-aware processor"
// static errors.
//
// The default is false. xsl:import-schema (decl_import_schema.go) and real
// runtime validation dispatch (schema_validation.go) are both now built and
// gated behind this same switch — a schema-aware run genuinely compiles
// imported components, validates constructed/retrieved nodes against them,
// and annotates real xmltree.Node.SchemaType/TypeAnno values, all inert at
// the default setting. What is NOT built: XPath 3.0 static typing (§26.1,
// out of scope permanently — see the schema-awareness roadmap entry).
//
// The claim belongs to the compiled stylesheet (Features.SchemaAware, set per
// compile through CompileWith and exposed as engine.Features). This
// process-wide switch is only the DEFAULT for a compile that is given no
// Features — what the XSLT30 conformance harness sets under XSLT30_FULL=1
// (xsltFullHarness in test/conformance/xslt30_test.go). Every construct a
// basic processor must reject asks the stylesheet's Features, never this
// variable directly.
var schemaAware atomic.Bool

// SetSchemaAware sets the process-wide default of Features.SchemaAware.
func SetSchemaAware(v bool) { schemaAware.Store(v) }

// validationValueOK reports whether a [xsl:]validation / [xsl:]default-validation
// value is one this run can honour. XSLT 3.0 §3.15 requires a NON-schema-aware
// processor to treat preserve and lax AS IF they were strip — both mean
// "validate however the calling application says", and for such a processor
// that is not at all — so only "strict" genuinely demands typed data, which a
// schema-aware run can supply. Anything outside the four-value vocabulary is
// XTSE1660 either way.
func validationValueOK(v string, f Features) bool {
	switch v {
	case "strip", "preserve", "lax":
		return true
	case "strict":
		return f.SchemaAware
	}
	return false
}
