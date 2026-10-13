package engine

import (
	"fmt"
	"strings"

	"github.com/tim-riep/go-xslt/internal/xslt"
)

// toString renders a recovered panic value for a diagnostic message.
func toString(v any) string {
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("%v", v)
}

// run compiles the stylesheet and executes the transformation.
func run(req Request) Result {
	if req.Stylesheet == "" {
		return Result{Diagnostics: []Diagnostic{{
			Severity: SeverityError,
			Code:     "XTSE0010",
			Phase:    "parse",
			Message:  "empty stylesheet",
		}}}
	}

	ss, err := xslt.CompileFrom(req.Stylesheet, req.BaseDir)
	if err != nil {
		return Result{Diagnostics: []Diagnostic{diagFromErr(err, "compile")}}
	}

	entry := xslt.Entry{Template: req.InitialTemplate, Params: req.Params}
	rr, terr := ss.TransformEntry(req.Source, entry, req.BaseDir)
	if terr != nil {
		return Result{Diagnostics: []Diagnostic{diagFromErr(terr, "run")}}
	}
	res := Result{Output: rr.Output, Method: rr.Method, Messages: rr.Messages}
	for _, s := range rr.Secondary {
		res.SecondaryOutputs = append(res.SecondaryOutputs, SecondaryOutput{Href: s.Href, Content: s.Content, Method: s.Method})
	}
	return res
}

// diagFromErr converts an xslt/xpath error into a Diagnostic, extracting
// line/column when available.
func diagFromErr(err error, phase string) Diagnostic {
	d := Diagnostic{Severity: SeverityError, Phase: phase, Message: err.Error(), Code: errCode(err.Error())}
	if ce, ok := err.(*xslt.CompileError); ok {
		d.Line, d.Col = ce.Line, ce.Col
	}
	return d
}

// errCode extracts the W3C error code of a message of the form "err:CODE: ..."
// (XPST0003, FORG0001, XTDE0040, ...); "" when the message does not carry one.
func errCode(msg string) string {
	rest, ok := strings.CutPrefix(msg, "err:")
	if !ok {
		return ""
	}
	end := 0
	for end < len(rest) && (rest[end] >= 'A' && rest[end] <= 'Z' || rest[end] >= '0' && rest[end] <= '9') {
		end++
	}
	if end < 6 {
		return ""
	}
	return rest[:end]
}
