package xsd

// Collecting validation: Validate reports every error it can find, not only
// the first. The collector lives on the per-call Schema copy (nil on the
// ValidateNode path, which keeps its stop-at-first-error behaviour), so the
// verdict is identical either way — only more of the document is looked at
// once it is known to be invalid.

import (
	"errors"
	"sort"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

type collector struct {
	errs []*Error
	max  int // 0 = unlimited
}

// errStopped unwinds validation once MaxErrors diagnostics are collected.
var errStopped = errors.New("xsd: error limit reached")

// errModelMismatch is returned by validateChildren after it has recorded a
// content-model error itself; the caller may still evaluate the element's
// assertions when the children could be assigned to declarations.
var errModelMismatch = errors.New("xsd: content model mismatch recorded")

// soft records err (an instance-validation *Error located at el) and reports
// whether validation may carry on. Anything that is not an *Error — notably
// ErrUnsupported — and every error when not collecting is passed up unchanged.
func (s *Schema) soft(err error, el *xmltree.Node) (bool, error) {
	var e *Error
	if s.coll == nil || !errors.As(err, &e) {
		return false, err
	}
	stampLoc(&err, el)
	s.coll.errs = append(s.coll.errs, e)
	if s.coll.max > 0 && len(s.coll.errs) >= s.coll.max {
		return false, errStopped
	}
	return true, nil
}

// sorted returns the collected errors by document position (stable, so the
// errors of one element keep the order they were found in).
func (c *collector) sorted() []Error {
	out := make([]Error, len(c.errs))
	for i, e := range c.errs {
		out[i] = *e
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Col < out[j].Col
	})
	return out
}

// assessDeclaredChildren validates the children of an element whose content
// model did not match, as far as their declarations can be told by name.
// It reports whether every child could be assigned.
func (s *Schema) assessDeclaredChildren(ct *ComplexType, kids []*xmltree.Node, scopes map[xname]*keyTable) (all bool, err error) {
	p := s.effectiveParticle(ct)
	decls := map[xname]*ElementDecl{}
	s.contentDecls(p, decls)
	fixed := map[xname][]string{}
	collectFixedsByName(p, fixed, map[*modelGroup]bool{})
	all = true
	for _, kid := range kids {
		nn := nameOf(kid)
		d, ok := decls[nn]
		if !ok {
			all = false
			continue
		}
		if e := s.validateElementAgainst(d, kid, scopes, fixed[nn]...); e != nil {
			if e == errStopped {
				return false, e
			}
			if cont, ret := s.soft(e, kid); !cont {
				return false, ret
			}
		}
	}
	return all, nil
}

// afterModelError finishes validateComplex when validateChildren failed. A
// recorded content-model mismatch still lets the declared children and — when
// every child could be assigned — the element's own assertions be checked.
func (s *Schema) afterModelError(err error, ct *ComplexType, el *xmltree.Node, kids []*xmltree.Node, scopes map[xname]*keyTable) error {
	if err != errModelMismatch {
		return err
	}
	all, e := s.assessDeclaredChildren(ct, kids, scopes)
	if e != nil {
		return e
	}
	if all {
		return s.checkAsserts(ct, el)
	}
	return nil
}

// failAssert reports a failed assertion: while collecting it is recorded and
// the next assertion is tried, otherwise it ends the check.
func (s *Schema) failAssert(a *assertion, el *xmltree.Node) (bool, error) {
	return s.soft(assertionFailed(a, el), el)
}

// collected is the number of errors recorded so far; uncollect withdraws
// those recorded after a collected() reading (see validateChildren's skip-
// wildcard excuse).
func (s *Schema) collected() int {
	if s.coll == nil {
		return 0
	}
	return len(s.coll.errs)
}

func (s *Schema) uncollect(n int) {
	if s.coll != nil && len(s.coll.errs) > n {
		s.coll.errs = s.coll.errs[:n]
	}
}
