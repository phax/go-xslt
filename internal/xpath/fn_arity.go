package xpath

import "fmt"

// Arity catalog for fn:function-lookup and named-function references. It records
// the valid arities of standard functions so that (a) a standard function the
// engine does not implement (doc, collection, …) is still reported as existing,
// and (b) a lookup at an invalid arity (name#2, round#3) correctly yields the
// empty sequence. Functions absent from the catalog fall back to a name-only
// existence check (no arity validation), so omissions never reject a real call.

const arityVariadic = 1 << 30

// fnArityExtra lists the fn: functions that exist for name lookup although
// the F&O 3.1 catalogue (fn_catalog.go, the source of every other function's
// arities) does not define them.
var fnArityExtra = map[string][2]int{
	"put": {2, 3},
}

// stdArity returns the [min,max] arity range of a standard function and whether
// the function is part of the catalog at all.
func stdArity(ns, local string) (lo, hi int, known bool) {
	switch ns {
	case nsFn, nsMath, nsMap, nsArray:
		if sig := builtinSigOf(ns, local); sig != nil {
			return sig.lo, sig.hi, true
		}
		if ns == nsFn {
			r, ok := fnArityExtra[local]
			return r[0], r[1], ok
		}
	case nsXS:
		// Atomic (and built-in list) type constructors take exactly one argument.
		if _, ok := AtomTypeByName(local); ok {
			return 1, 1, true
		}
		if _, ok := xsListItemType(local); ok {
			return 1, 1, true
		}
	}
	return 0, 0, false
}

// stdArityOutside reports whether n falls outside the arities of a KNOWN
// standard function: outside its [lo,hi] range, or inside it at an arity the
// catalogue does not define (format-date/-time/-dateTime exist only as #2 and
// #5 — format-date-inpt-er4: the 3-argument call is XPST0017).
func stdArityOutside(ns, local string, lo, hi, n int) bool {
	if n < lo || (hi != arityVariadic && n > hi) {
		return true
	}
	if sig := builtinSigOf(ns, local); sig != nil {
		return !sig.hasArity(n)
	}
	return false
}

// checkStdArity rejects a DIRECT call at an arity the catalog forbids
// (XPST0017) — abs(), abs(1,2), string(1,2) were silently tolerated before.
func checkStdArity(ns, local string, n int) error {
	lo, hi, known := stdArity(ns, local)
	if !known {
		return nil
	}
	if stdArityOutside(ns, local, lo, hi, n) {
		return fmt.Errorf("err:XPST0017: %s() does not accept %d arguments", local, n)
	}
	return nil
}

// StdFuncArity reports whether ns/local is a cataloged standard function
// (the fn:/math:/map:/array: namespaces, or an xs: atomic/list-type
// constructor) and, if so, its [lo,hi] arity range. Exported for host
// (XSLT) implementations of fn:function-available.
func StdFuncArity(ns, local string) (lo, hi int, known bool) {
	return stdArity(ns, local)
}

// StdArityOK reports whether n is a valid arity for the cataloged standard
// function ns/local (StdFuncArity must already report it known).
func StdArityOK(ns, local string, n int) bool {
	if _, _, known := StdFuncArity(ns, local); !known {
		return false
	}
	lo, hi, _ := StdFuncArity(ns, local)
	return !stdArityOutside(ns, local, lo, hi, n)
}
