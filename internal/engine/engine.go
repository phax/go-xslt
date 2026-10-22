// Package engine is the stable public facade for the XSLT processor.
//
// The GUI service layer and the conformance test harness depend ONLY on this
// package. Everything behind it (xdm, xmltree, xpath, xslt) may be rewritten
// freely without breaking callers.
package engine

import (
	"time"

	"github.com/tim-riep/go-xslt/internal/fsaccess"
	"github.com/tim-riep/go-xslt/internal/netfetch"
)

// Security says which local files and which network resources a run may read.
// The zero value is the safe default: files under BaseDir only, no network.
// A stylesheet, an expression, a schema and a source document are often
// untrusted: unrestricted file access would let them read anything the
// process can (keys, configuration) and copy it into the output, and network
// access would let them reach internal services or send data away in a URL.
// Only the caller widens these limits, never the input.
type Security struct {
	// AllowedDirs are further directories (with their subdirectories) whose
	// files may be read, besides BaseDir.
	AllowedDirs []string `json:"allowedDirs,omitempty"`
	// AllowAllFiles lifts the file restriction entirely: any path the
	// process can read, as before this option existed. Only for trusted
	// input.
	AllowAllFiles bool `json:"allowAllFiles,omitempty"`
	// AllowNetwork lets document(), fn:doc, fn:doc-available,
	// fn:unparsed-text(-lines/-available), fn:collection members and
	// xsl:source-document retrieve http and https URIs. Off by default, in
	// which case such a URI is an unavailable resource. A document retrieved
	// over the network is parsed without reading any external DTD or entity
	// files.
	AllowNetwork bool `json:"allowNetwork"`
	// AllowedHosts, when non-empty, limits network access to these host
	// names; a leading "." also admits every subdomain (".example.com").
	// Redirects are checked against the same list.
	AllowedHosts []string `json:"allowedHosts,omitempty"`
}

// FileGuard is the file guard for a run of s rooted at baseDir: reads are
// confined to baseDir and s.AllowedDirs (no BaseDir and no AllowedDirs: no
// file reads at all), unless s.AllowAllFiles, which returns nil (no check).
// It is a function rather than a method so it does not show up on the public
// engine.Security / xsd.Security alias.
func FileGuard(s Security, baseDir string) *fsaccess.Guard {
	if s.AllowAllFiles {
		return nil
	}
	return fsaccess.New(append([]string{baseDir}, s.AllowedDirs...)...)
}

// fetcher is the per-run retriever for s, nil when s allows no network access.
func (s Security) fetcher() *netfetch.Fetcher {
	return netfetch.New(netfetch.Policy{AllowNetwork: s.AllowNetwork, AllowedHosts: s.AllowedHosts})
}

// Request is a single transformation request.
type Request struct {
	// Stylesheet is the XSLT source text.
	Stylesheet string `json:"stylesheet"`
	// Source is the XML input document. May be empty when InitialTemplate is set.
	Source string `json:"source"`
	// Params are top-level stylesheet parameters (string-typed for now).
	Params map[string]string `json:"params"`
	// InitialTemplate optionally names an xsl:template to invoke first
	// (xsl:initial-template) instead of matching against the source root.
	InitialTemplate string `json:"initialTemplate"`
	// BaseDir is the workspace directory used to resolve relative URIs in
	// xsl:import/xsl:include, document()/doc(), unparsed-text(), and to write
	// xsl:result-document. Empty disables disk resolution.
	BaseDir string `json:"baseDir"`
	// Security limits which files and network resources the run may read;
	// the zero value allows files under BaseDir only, and no network.
	Security Security `json:"security"`
}

// SecondaryOutput is a result produced by xsl:result-document.
type SecondaryOutput struct {
	Href    string `json:"href"`
	Content string `json:"content"`
	Method  string `json:"method"`
}

// Severity levels for diagnostics.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Diagnostic is a single error or warning produced while compiling or running
// a transformation. Line/Col are 1-based; 0 means "unknown".
type Diagnostic struct {
	Severity string `json:"severity"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	// Code is a W3C-style error code where known (e.g. "XPST0003").
	Code    string `json:"code"`
	Message string `json:"message"`
	// Phase is where the diagnostic originated: "parse", "compile" or "run".
	Phase string `json:"phase"`
	// Rule is the rule identifier (msg:message/@rule) of a failed XSD
	// assertion; empty when not applicable.
	Rule string `json:"rule,omitempty"`
	// Path locates the element a failed XSD assertion was evaluated on, e.g.
	// /Doc[1]/Item[2]; empty when not applicable.
	Path string `json:"path,omitempty"`
	// Particle is the element declaration a content-model error refers to, as
	// an expanded name ({namespace}local); empty otherwise.
	Particle string `json:"particle,omitempty"`
	// Kind tells content-model errors apart: "missing", "too-many",
	// "unexpected" or "order"; empty otherwise.
	Kind string `json:"kind,omitempty"`
}

// Result is the outcome of a transformation. Output holds the serialized
// result tree; Diagnostics holds any errors/warnings. When a fatal error
// occurs, Output is empty and Diagnostics contains at least one error.
type Result struct {
	Output      string       `json:"output"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	// DurationMs is wall-clock transform time in milliseconds.
	DurationMs int64 `json:"durationMs"`
	// Method is the effective output method ("xml", "html", "text", "json").
	Method string `json:"method"`
	// Messages holds text emitted by xsl:message.
	Messages []string `json:"messages"`
	// SecondaryOutputs holds documents produced by xsl:result-document.
	SecondaryOutputs []SecondaryOutput `json:"secondaryOutputs"`
}

// HasErrors reports whether the result contains any error-severity diagnostic.
func (r Result) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Transform runs req and returns the result. It never panics: any internal
// panic is recovered and surfaced as an error diagnostic, so a partially
// implemented engine can never crash the host application.
func Transform(req Request) (res Result) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Code:     "INTERNAL",
				Phase:    "run",
				Message:  "internal engine error: " + toString(r),
			})
		}
		res.DurationMs = time.Since(start).Milliseconds()
	}()

	return run(req)
}
