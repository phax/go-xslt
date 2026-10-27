package xpath

import (
	"strings"
)

// Declared signatures of the built-in functions, so that a named function
// reference such as name#1 carries its parameter and return types and typed
// function tests can judge it precisely (instanceof128..130: name#1 is NOT a
// function(item()) as xs:string; ArrayTest-064/084: [floor#1, …, name#1] is
// not an array(function(xs:numeric?) as xs:numeric?); instanceof134:
// fn:filter's predicate parameter is function(item()) as xs:boolean, so
// filter#2 is not a function(function(*), item()*) as item()*, parameter
// types being contravariant). The signatures are the catalogue's
// (fn_catalog.go, parsed by fn_argcheck.go), the same lines that drive the
// argument checks of a call. Function COERCION of arguments stays arity-based
// (convertForParam), so a typed built-in passed where a different function
// type is expected is still accepted and checked when called.

// sigParseSeqType parses one SequenceType from its lexical form.
func sigParseSeqType(s string) (*SeqType, bool) {
	toks, err := lex(strings.TrimSpace(s))
	if err != nil {
		return nil, false
	}
	pr := &parser{toks: toks}
	st, err := pr.parseSequenceType()
	if err != nil || pr.cur().kind != tEOF {
		return nil, false
	}
	return st, true
}

// sigApply attaches the catalogued signature of a built-in function to a
// named function reference. A function with no catalogue entry (or whose
// return type is not a SequenceType, fn:error's "none") stays untyped.
func sigApply(f *Function) *Function {
	if f == nil {
		return f
	}
	sig := builtinSigOf(f.NS, f.Name)
	if sig == nil {
		return f
	}
	form := sig.form(f.Arity)
	if form == nil || form.retST == nil {
		return f
	}
	params := make([]*SeqType, f.Arity)
	for i := range params {
		p := form.spec(i)
		if p == nil {
			return f
		}
		params[i] = p.declared
	}
	f.Typed, f.Params, f.Ret = true, params, form.retST
	return f
}
