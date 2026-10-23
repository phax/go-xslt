package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tim-riep/go-xslt/internal/xmltree"
	"github.com/tim-riep/go-xslt/internal/xpath"
)

// XPathRequest evaluates a standalone XPath 3.1 expression, optionally against
// an XML context document.
type XPathRequest struct {
	// Expression is the XPath 3.1 expression to evaluate.
	Expression string `json:"expression"`
	// Source is an optional XML document. When non-empty its document node is
	// the context item, so path expressions like /root/child work; when empty
	// the expression is evaluated with no context item (e.g. 1 + 2,
	// current-dateTime(), string-length('abc')).
	Source string `json:"source"`
	// BaseDir resolves relative URIs in fn:doc/fn:unparsed-text/fn:collection.
	// Empty disables disk resolution (those functions report the resource as
	// unavailable rather than erroring).
	BaseDir string `json:"baseDir"`
	// Security limits which files and network resources the expression may
	// read; the zero value allows files under BaseDir only, and no network.
	Security Security `json:"security"`
	// Namespaces binds prefixes for the expression (prefix → namespace URI),
	// e.g. {"inv": "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"}.
	// Bound as well, unless an entry here says otherwise: the prefixes the
	// document element of Source declares, and xml, xs, xsi, fn, math, map,
	// array and err. A prefix bound nowhere is the error XPST0081. Without a
	// prefix, Q{uri}local names any namespace.
	Namespaces map[string]string `json:"namespaces,omitempty"`
	// DefaultElementNamespace is the namespace of element names written
	// without a prefix; empty, the default, means no namespace.
	DefaultElementNamespace string `json:"defaultElementNamespace,omitempty"`
	// Variables binds $name to a string. A name may have a prefix bound in
	// Namespaces ("p:name").
	Variables map[string]string `json:"variables,omitempty"`
	// VariableExprs binds $name to the value of an XPath expression, so it
	// can be typed or a node: "42", "xs:date('2026-01-31')", "//item[1]".
	// It is evaluated like Expression, against Source, but cannot refer to
	// other variables. A name here wins over the same name in Variables.
	VariableExprs map[string]string `json:"variableExprs,omitempty"`
}

// predeclaredNS are the prefixes every standalone expression may use.
var predeclaredNS = map[string]string{
	"xml":   "http://www.w3.org/XML/1998/namespace",
	"xs":    "http://www.w3.org/2001/XMLSchema",
	"xsi":   "http://www.w3.org/2001/XMLSchema-instance",
	"fn":    "http://www.w3.org/2005/xpath-functions",
	"math":  "http://www.w3.org/2005/xpath-functions/math",
	"map":   "http://www.w3.org/2005/xpath-functions/map",
	"array": "http://www.w3.org/2005/xpath-functions/array",
	"err":   "http://www.w3.org/2005/xqt-errors",
}

// nsBindings is the static namespace context of a standalone expression.
type nsBindings map[string]string

func (m nsBindings) ResolveNS(prefix string) (string, bool) {
	uri, ok := m[prefix]
	return uri, ok
}

// varBindings holds the variables of a standalone expression, by name as
// written ("local" or "prefix:local").
type varBindings map[string]xpath.Object

func (m varBindings) ResolveVar(prefix, local string) (xpath.Object, bool) {
	name := local
	if prefix != "" {
		name = prefix + ":" + local
	}
	v, ok := m[name]
	return v, ok
}

// XPathItem is one item of the result sequence.
type XPathItem struct {
	// Type is a short label: a node kind ("element", "attribute", …), an atomic
	// type QName ("xs:integer", …), or "map"/"array"/"function".
	Type string `json:"type"`
	// Value is the item's string value (element/attribute string value; atomic
	// lexical form).
	Value string `json:"value"`
}

// XPathResult is the outcome of an XPath evaluation.
type XPathResult struct {
	// Value is the string value of the whole result sequence.
	Value string `json:"value"`
	// Items is one entry per item in the result sequence.
	Items []XPathItem `json:"items"`
	// Count is the number of items in the result sequence.
	Count int `json:"count"`
	// Diagnostics carries parse/eval errors (non-empty on failure).
	Diagnostics []Diagnostic `json:"diagnostics"`
	// DurationMs is wall-clock evaluation time in milliseconds.
	DurationMs int64 `json:"durationMs"`
}

// HasErrors reports whether the result contains any error-severity diagnostic.
func (r XPathResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// EvalXPath parses and evaluates req.Expression. It never panics: any internal
// panic is recovered and surfaced as an error diagnostic.
func EvalXPath(req XPathRequest) XPathResult {
	return EvalXPathContext(context.Background(), req)
}

// EvalXPathContext is EvalXPath that stops when ctx is done, like
// TransformContext.
func EvalXPathContext(ctx context.Context, req XPathRequest) (res XPathResult) {
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
	return runXPath(ctx, req)
}

func runXPath(cctx context.Context, req XPathRequest) XPathResult {
	parsed, err := xpath.Parse(req.Expression)
	if err != nil {
		d := xpathDiag("parse", err)
		if d.Code == "" {
			d.Code = "XPST0003" // a grammar error; other static errors carry their own code
		}
		return XPathResult{Diagnostics: []Diagnostic{d}}
	}
	ctx := &xpath.Context{Pos: 1, Size: 1}
	if cctx.Done() != nil {
		polls := 0
		ctx.Interrupt = func() error {
			polls++
			if polls&1023 == 0 && cctx.Err() != nil {
				return fmt.Errorf("evaluation stopped: %w", cctx.Err())
			}
			return nil
		}
	}
	if strings.TrimSpace(req.Source) != "" {
		// ParseLenient11WithBase (not the plain Parse EvalXPath used before)
		// so a DOCTYPE's external SYSTEM subset resolves against BaseDir too,
		// consistent with how the XSLT engine parses its own source document.
		doc, perr := xmltree.ParseLenient11WithBaseFS(req.Source, req.BaseDir, FileGuard(req.Security, req.BaseDir))
		if perr != nil {
			return XPathResult{Diagnostics: []Diagnostic{xpathDiag("parse", perr)}}
		}
		ctx.Node = doc
	}
	ns := nsBindings{}
	for p, uri := range predeclaredNS {
		ns[p] = uri
	}
	if ctx.Node != nil {
		if root := xmltree.RootElement(ctx.Node); root != nil {
			for _, d := range root.NSDecls() {
				if d.Name.Local != "" && d.Value != "" {
					ns[d.Name.Local] = d.Value
				}
			}
		}
	}
	for p, uri := range req.Namespaces {
		ns[p] = uri
	}
	ctx.NS = ns
	ctx.DefaultElemNS = req.DefaultElementNamespace
	if strings.TrimSpace(req.BaseDir) != "" {
		r := newStandaloneResolver(req.BaseDir)
		r.net = req.Security.fetcher()
		r.files = FileGuard(req.Security, req.BaseDir)
		ctx.Resolver = r
		ctx.BaseURI = baseDirURI(req.BaseDir)
	} else if f := req.Security.fetcher(); f != nil {
		// No directory, but the network is allowed: absolute http(s) URIs
		// still resolve, and nothing is ever read from disk.
		ctx.Resolver = &standaloneResolver{net: f, noDisk: true}
	}
	if len(req.Variables) > 0 || len(req.VariableExprs) > 0 {
		vars := varBindings{}
		for name, val := range req.Variables {
			vars[name] = xpath.NewString(val)
		}
		for name, expr := range req.VariableExprs {
			p, err := xpath.Parse(expr)
			var val xpath.Object
			if err == nil {
				val, err = p.Eval(ctx)
			}
			if err != nil {
				d := xpathDiag("parse", err)
				if d.Code == "" {
					d.Code = "XPST0003"
				}
				d.Message = fmt.Sprintf("variable $%s: %s", name, d.Message)
				return XPathResult{Diagnostics: []Diagnostic{d}}
			}
			vars[name] = val
		}
		ctx.Vars = vars
	}
	v, eerr := parsed.Eval(ctx)
	if eerr != nil {
		return XPathResult{Diagnostics: []Diagnostic{xpathDiag("run", eerr)}}
	}
	items := xpath.Items(v)
	res := XPathResult{Count: len(items)}
	parts := make([]string, len(items))
	for i, it := range items {
		s := xpath.ItemString(it)
		parts[i] = s
		res.Items = append(res.Items, XPathItem{Type: xpath.ItemKind(it), Value: s})
	}
	// Build the whole-sequence preview from the per-item host strings rather
	// than xpath.ToString, which is empty for maps/arrays (they have no string
	// value) and returns only the first item for multi-item sequences.
	res.Value = strings.Join(parts, "\n")
	return res
}

// xpathDiag turns an XPath parse/eval error into a Diagnostic, extracting a
// leading "err:CODE:" prefix (or a bare "XPST0003"-shaped token) into Code.
func xpathDiag(phase string, err error) Diagnostic {
	msg := err.Error()
	code := ""
	if rest, ok := strings.CutPrefix(msg, "err:"); ok {
		if i := strings.IndexByte(rest, ':'); i > 0 {
			code = rest[:i]
			msg = strings.TrimSpace(rest[i+1:])
		}
	} else if i := strings.Index(msg, "err:"); i >= 0 {
		// A code inside the message (the parser prefixes the expression).
		if j := strings.IndexByte(msg[i+4:], ':'); j > 0 {
			code = msg[i+4 : i+4+j]
		}
	}
	return Diagnostic{Severity: SeverityError, Phase: phase, Code: code, Message: msg}
}
