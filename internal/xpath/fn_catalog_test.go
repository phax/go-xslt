package xpath

import (
	"sort"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// Every built-in function registered in coreFuncs, mathFuncs, mapFuncs or
// arrayFuncs must have a signature in the catalogue (fn_catalog.go): the
// signature drives both the arity check and the function conversion rules, so
// a function without one would be silently unchecked.
func TestBuiltinCatalogComplete(t *testing.T) {
	ensureCatalog()
	for _, reg := range []struct {
		ns    string
		funcs map[string]coreFunc
		set   sigSet
	}{
		{nsFn, coreFuncs, sigFn},
		{nsMath, mathFuncs, sigMath},
		{nsMap, mapFuncs, sigMap},
		{nsArray, arrayFuncs, sigArray},
	} {
		var names []string
		for n := range reg.funcs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if reg.set[n] == nil {
				t.Errorf("%s:%s is registered but has no signature in builtinCatalogSrc", prefixForNS(reg.ns), n)
			}
		}
	}
}

// The catalogue defines no function the engine lacks, except the ones it
// knowingly does not implement (they still exist as function items that
// raise FOXX0001 when called).
func TestBuiltinCatalogOnlyKnownUnimplemented(t *testing.T) {
	ensureCatalog()
	unimplemented := map[string]bool{"fn:load-xquery-module": true}
	for _, reg := range []struct {
		pfx   string
		funcs map[string]coreFunc
		set   sigSet
	}{
		{"fn", coreFuncs, sigFn},
		{"math", mathFuncs, sigMath},
		{"map", mapFuncs, sigMap},
		{"array", arrayFuncs, sigArray},
	} {
		for n := range reg.set {
			if _, ok := reg.funcs[n]; !ok && !unimplemented[reg.pfx+":"+n] {
				t.Errorf("%s:%s has a signature but no implementation", reg.pfx, n)
			}
		}
	}
}

// The catalogue parses, its arities are consistent, and every parameter has a
// usable type.
func TestBuiltinCatalogWellFormed(t *testing.T) {
	ensureCatalog()
	for _, set := range []sigSet{sigFn, sigMath, sigMap, sigArray} {
		for name, sig := range set {
			if len(sig.forms) == 0 || len(sig.forms) != len(sig.arities) {
				t.Errorf("%s: %d forms for %d arities", name, len(sig.forms), len(sig.arities))
				continue
			}
			for i, f := range sig.forms {
				if len(f.params) != sig.arities[i] {
					t.Errorf("%s#%d: %d parameters", name, sig.arities[i], len(f.params))
				}
				for _, p := range f.params {
					if p.st == nil || p.name == "" {
						t.Errorf("%s#%d: parameter %+v has no type or name", name, sig.arities[i], p)
					}
					// Function-typed parameters are checked as function(*).
					if !p.st.Empty && p.st.Item.Kind == itFunction && !p.st.Item.Wild {
						t.Errorf("%s#%d: $%s keeps a typed function test", name, sig.arities[i], p.name)
					}
				}
			}
			if sig.lo != sig.arities[0] {
				t.Errorf("%s: lo %d, first arity %d", name, sig.lo, sig.arities[0])
			}
			if sig.hi != arityVariadic && sig.hi != sig.arities[len(sig.arities)-1] {
				t.Errorf("%s: hi %d, last arity %d", name, sig.hi, sig.arities[len(sig.arities)-1])
			}
		}
	}
	// The one variadic function.
	c := sigFn["concat"]
	if c == nil || c.hi != arityVariadic || c.lo != 2 || c.form(9) == nil || c.form(1) != nil {
		t.Errorf("fn:concat is not 2-or-more: %+v", c)
	}
}

// A host catalogue (XSLT's additions to fn:) uses the same notation and the
// same rules.
func TestHostSigCatalog(t *testing.T) {
	c, err := NewSigCatalog(`
fn:thing($name as xs:string, $n as xs:integer) as item()*
fn:thing($name as xs:string) as item()*
`)
	if err != nil {
		t.Fatal(err)
	}
	if known, ok := c.Has("thing", 2); !known || !ok {
		t.Error("thing#2 not catalogued")
	}
	if known, ok := c.Has("thing", 3); !known || ok {
		t.Error("thing#3 should be known but have no such arity")
	}
	if _, err := c.Check("thing", []Object{NewString("a"), NewString("b")}, &Context{}); err == nil {
		t.Error("string for xs:integer accepted")
	}
	out, err := c.Check("thing", []Object{NewUntyped("a"), NewUntyped("3")}, &Context{})
	if err != nil {
		t.Fatal(err)
	}
	if a, _ := out[1].(*Atomic); a == nil || a.T != XSinteger {
		t.Errorf("untypedAtomic not cast to xs:integer: %#v", out[1])
	}
	if _, err := NewSigCatalog("fn:bad($x xs:string) as item()*"); err == nil {
		t.Error("malformed catalogue accepted")
	}
}

// No built-in panics on an argument of any shape, whether the conversion
// rules accept it or not (and under backwards-compatible processing, where
// the check is skipped and the implementation sees the raw value).
func TestBuiltinsNeverPanicOnOddArguments(t *testing.T) {
	ensureCatalog()
	doc, err := xmltree.Parse(`<r a="1"><c>x</c><!--k--></r>`)
	if err != nil {
		t.Fatal(err)
	}
	node := NodeSet{doc.Children[0]}
	abs, _ := lookupFunction(&Context{}, nsFn, "abs", 1)
	candidates := []Object{
		Sequence{}, NewString("a"), NewInteger(1), NewDouble(1.5), NewBool(true),
		NewUntyped("2"), NewQName(xmltree.Name{Space: "urn:x", Local: "q"}),
		node, NodeSet{}, Sequence{NewInteger(1), NewInteger(2)}, Sequence{NewString("a"), NewString("b")},
		NewMap(), NewArray([]Object{NewInteger(1)}), abs, "bare", float64(2), true,
	}
	skip := map[string]bool{ // would read files, wait, or print
		"fn:doc": true, "fn:doc-available": true, "fn:collection": true, "fn:uri-collection": true,
		"fn:unparsed-text": true, "fn:unparsed-text-lines": true, "fn:unparsed-text-available": true,
		"fn:json-doc": true, "fn:transform": true, "fn:trace": true,
	}
	for _, bc := range []bool{false, true} {
		for pfx, uri := range catalogPrefixes {
			for name, sig := range sigSetFor(uri) {
				if skip[pfx+":"+name] {
					continue
				}
				fn := map[string]map[string]coreFunc{nsFn: coreFuncs, nsMath: mathFuncs, nsMap: mapFuncs, nsArray: arrayFuncs}[uri][name]
				if fn == nil {
					continue
				}
				for i, f := range sig.forms {
					arity := sig.arities[i]
					if f.rest != nil {
						arity = 3
					}
					for _, cand := range candidates {
						args := make([]Object, arity)
						for j := range args {
							args[j] = cand
						}
						func() {
							defer func() {
								if r := recover(); r != nil {
									t.Errorf("%s:%s#%d (bc=%v) panics on %T %v: %v", pfx, name, arity, bc, cand, cand, r)
								}
							}()
							ctx := &Context{Node: doc, Pos: 1, Size: 1, BC10: bc}
							_, _ = dispatchFunc(pfx, name, args, ctx)
						}()
					}
				}
			}
		}
	}
}
