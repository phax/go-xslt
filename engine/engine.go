// Package engine is the public facade of the goxslt XSLT/XPath processor.
// It re-exports goxslt/internal/engine, the module's stable compatibility
// boundary, so external modules can consume the engine; everything behind
// this facade may be rewritten freely.
package engine

import (
	"context"
	"io"

	"github.com/tim-riep/go-xslt/internal/engine"
)

type (
	Request          = engine.Request
	TransformRequest = engine.TransformRequest
	// Security limits the files and network resources a run may read.
	Security = engine.Security
	// Features turns on optional XSLT 3.0 capabilities (schema awareness,
	// XSLT 1.0 backwards compatibility, strict streaming) per stylesheet.
	Features = engine.Features
	// Limits bounds what a run may execute (instruction steps, nesting depth).
	Limits = engine.Limits
	// Stylesheet is a stylesheet compiled once by Compile and run many times.
	Stylesheet      = engine.Stylesheet
	Result          = engine.Result
	Diagnostic      = engine.Diagnostic
	SecondaryOutput = engine.SecondaryOutput

	// XSD schema-validation API.
	ValidateRequest = engine.ValidateRequest
	ValidateResult  = engine.ValidateResult
	XSDVersion      = engine.XSDVersion

	// Standalone XPath 3.1 evaluation API.
	XPathRequest = engine.XPathRequest
	XPathResult  = engine.XPathResult
	XPathItem    = engine.XPathItem
)

const (
	SeverityError   = engine.SeverityError
	SeverityWarning = engine.SeverityWarning

	XSD10 = engine.XSD10
	XSD11 = engine.XSD11
)

// Transform runs req and returns the result. It never panics; errors are
// reported as Diagnostics on the Result.
func Transform(req Request) Result { return engine.Transform(req) }

// TransformContext is Transform that stops when ctx is done — a deadline, a
// cancelled request. The run then ends with an error diagnostic that names
// ctx.Err(). Cancellation is checked every 1024 instructions and inside long
// XPath evaluations, so a run stops shortly after ctx is done.
func TransformContext(ctx context.Context, req Request) Result {
	return engine.TransformContext(ctx, req)
}

// TransformTo runs req and writes the principal result to w rather than
// returning it in Result.Output, which stays empty; every other field is
// populated exactly as Transform populates it. A result large enough to matter
// never exists as one string, and when the stylesheet allows it the result tree
// is serialized and released while it is built — so a transformation that
// streams a large xsl:source-document through to its output runs in bounded
// memory on both sides. It never panics; errors are reported as Diagnostics.
func TransformTo(w io.Writer, req Request) Result { return engine.TransformTo(w, req) }

// TransformToContext is TransformTo that stops when ctx is done, like
// TransformContext.
func TransformToContext(ctx context.Context, w io.Writer, req Request) Result {
	return engine.TransformToContext(ctx, w, req)
}

// TransformFromReader is Transform with the source document read from src
// instead of req.Source (ignored), avoiding the extra copy a caller would
// otherwise pay converting their own io.Reader into a string before calling
// Transform. It never panics; errors (including a read failure on src) are
// reported as Diagnostics.
func TransformFromReader(src io.Reader, req Request) Result {
	return engine.TransformFromReader(src, req)
}

// Validate validates an XML instance against XSD 1.0/1.1 schemas. It never
// panics; errors are reported as Diagnostics with Valid=false.
func Validate(req ValidateRequest) ValidateResult { return engine.Validate(req) }

// ValidateContext is Validate that stops when ctx is done, like
// TransformContext.
func ValidateContext(ctx context.Context, req ValidateRequest) ValidateResult {
	return engine.ValidateContext(ctx, req)
}

// EvalXPath evaluates a standalone XPath 3.1 expression, optionally against an
// XML context document (req.Source). It never panics; errors are reported as
// Diagnostics.
func EvalXPath(req XPathRequest) XPathResult { return engine.EvalXPath(req) }

// EvalXPathContext is EvalXPath that stops when ctx is done, like
// TransformContext.
func EvalXPathContext(ctx context.Context, req XPathRequest) XPathResult {
	return engine.EvalXPathContext(ctx, req)
}

// Compile compiles req.Stylesheet (imports and includes resolved against
// req.BaseDir) without running it, so the same stylesheet can then be run many
// times against changing data — Stylesheet.Transform, TransformTo and
// TransformFromReader. A *Stylesheet is immutable and safe for concurrent use.
// Source, Params and InitialTemplate of req are ignored; they belong to each
// run (TransformRequest). On failure the Stylesheet is nil and the diagnostics
// carry the error with its line and column in the stylesheet. It never panics.
func Compile(req Request) (*Stylesheet, []Diagnostic) { return engine.Compile(req) }
