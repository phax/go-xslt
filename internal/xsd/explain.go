package xsd

// Explaining a content-model mismatch.
//
// The matcher in content.go decides WHETHER children satisfy a content model;
// it cannot say what is wrong. When it has said "no" (and only then), this file
// replays the children through a small position automaton built from the same
// particle tree and names the defects one by one: a required element that is
// missing, an element that occurs more often than allowed, one that is not
// allowed at all, one that stands in the wrong place. The repair it assumes is
// always the cheapest one — fewest missing elements (a shortest path through
// the automaton) — so that independent defects are reported independently.
//
// Nothing here decides validity. A model it cannot express (xs:all, open
// content, huge occurrence bounds) yields no explanation, and the caller keeps
// the generic diagnostic.

import (
	"sort"
	"strconv"
	"strings"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

const (
	nfaMaxStates = 60000 // beyond this the model is not explained
	nfaMaxUnroll = 120   // largest min / (max-min) that is unrolled copy by copy
	nfaMaxErrors = 100   // defects reported per parent
)

type nfaLabel struct {
	decl *ElementDecl
	wc   *wildcard
	part *particle  // the element/wildcard particle (its occurrence bounds)
	anc  []ancEntry // enclosing groups, innermost first
}

// ancEntry is an enclosing group together with the particle that holds it (the
// group's own occurrence bounds).
type ancEntry struct {
	g *modelGroup
	p *particle
}

type nfaEdge struct {
	to  int
	lab *nfaLabel
}

type nfa struct {
	eps    [][]int
	out    [][]nfaEdge
	reps   [][]int // reverse epsilon edges
	rout   [][]int // reverse labelled edges: source states
	start  int
	accept int
}

type nfaBuilder struct {
	n     *nfa
	anc   []ancEntry
	depth int
	fail  bool
}

func (b *nfaBuilder) state() int {
	if len(b.n.eps) >= nfaMaxStates {
		b.fail = true
		return 0
	}
	b.n.eps = append(b.n.eps, nil)
	b.n.out = append(b.n.out, nil)
	b.n.reps = append(b.n.reps, nil)
	b.n.rout = append(b.n.rout, nil)
	return len(b.n.eps) - 1
}

func (b *nfaBuilder) epsEdge(from, to int) {
	if b.fail {
		return
	}
	b.n.eps[from] = append(b.n.eps[from], to)
	b.n.reps[to] = append(b.n.reps[to], from)
}

func (b *nfaBuilder) edge(from, to int, l *nfaLabel) {
	if b.fail {
		return
	}
	b.n.out[from] = append(b.n.out[from], nfaEdge{to, l})
	b.n.rout[to] = append(b.n.rout[to], from)
}

// particle builds the fragment for p with its occurrence bounds, unrolled.
func (b *nfaBuilder) particle(p *particle) (in, out int) {
	in = b.state()
	cur := in
	if p.min > nfaMaxUnroll {
		b.fail = true
		return in, in
	}
	for i := 0; i < p.min && !b.fail; i++ {
		ti, to := b.term(p)
		b.epsEdge(cur, ti)
		cur = to
	}
	if p.max == unbounded {
		hub := b.state()
		b.epsEdge(cur, hub)
		ti, to := b.term(p)
		b.epsEdge(hub, ti)
		b.epsEdge(to, hub)
		return in, hub
	}
	opt := p.max - p.min
	if opt > nfaMaxUnroll {
		b.fail = true
		return in, in
	}
	exit := b.state()
	for i := 0; i < opt && !b.fail; i++ {
		b.epsEdge(cur, exit)
		ti, to := b.term(p)
		b.epsEdge(cur, ti)
		cur = to
	}
	b.epsEdge(cur, exit)
	return in, exit
}

// term builds one occurrence of p's term.
func (b *nfaBuilder) term(p *particle) (in, out int) {
	in, out = b.state(), b.state()
	if b.fail {
		return
	}
	switch t := p.term.(type) {
	case *ElementDecl:
		b.edge(in, out, &nfaLabel{decl: t, part: p, anc: b.anc})
	case *wildcard:
		b.edge(in, out, &nfaLabel{wc: t, part: p, anc: b.anc})
	case *modelGroup:
		if t.compositor == cAll || b.depth > 100 {
			b.fail = true
			return
		}
		saved := b.anc
		b.anc = append([]ancEntry{{t, p}}, b.anc...)
		b.depth++
		if t.compositor == cChoice {
			for _, sub := range t.particles {
				si, so := b.particle(sub)
				b.epsEdge(in, si)
				b.epsEdge(so, out)
			}
		} else {
			cur := in
			for _, sub := range t.particles {
				si, so := b.particle(sub)
				b.epsEdge(cur, si)
				cur = so
			}
			b.epsEdge(cur, out)
		}
		b.depth--
		b.anc = saved
	default:
		b.fail = true
	}
	return
}

func buildNFA(p *particle) (*nfa, bool) {
	b := &nfaBuilder{n: &nfa{}}
	in, out := b.particle(p)
	if b.fail {
		return nil, false
	}
	b.n.start, b.n.accept = in, out
	return b.n, true
}

// sortedStates lists a state set in ascending order, so that everything derived
// from it (the order of alternatives, which label "was used") is deterministic.
func sortedStates(set map[int]bool) []int {
	qs := make([]int, 0, len(set))
	for q := range set {
		qs = append(qs, q)
	}
	sort.Ints(qs)
	return qs
}

func (n *nfa) closure(set map[int]bool) map[int]bool {
	out := make(map[int]bool, len(set))
	var stack []int
	for q := range set {
		out[q] = true
		stack = append(stack, q)
	}
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, e := range n.eps[q] {
			if !out[e] {
				out[e] = true
				stack = append(stack, e)
			}
		}
	}
	return out
}

func (s *Schema) labelMatches(l *nfaLabel, nn xname) bool {
	if l.decl != nil {
		return s.matchesElementName(l.decl, nn)
	}
	return wildcardMatches(l.wc, nn)
}

// step consumes one child from the state set cur: the new (closed) set, and
// the labels the child was matched by.
func (n *nfa) step(s *Schema, cur map[int]bool, nn xname) (map[int]bool, []*nfaLabel) {
	next := map[int]bool{}
	var labs []*nfaLabel
	for _, q := range sortedStates(cur) {
		for _, e := range n.out[q] {
			if s.labelMatches(e.lab, nn) {
				next[e.to] = true
				labs = append(labs, e.lab)
			}
		}
	}
	return n.closure(next), labs
}

const nfaInf = int(^uint(0) >> 2)

// distToGoal is, for every state, the fewest labelled edges (elements that
// would have to be inserted) on a path to a goal state; epsilon edges are free.
func (n *nfa) distToGoal(goal func(q int) bool) []int {
	dist := make([]int, len(n.eps))
	for i := range dist {
		dist[i] = nfaInf
	}
	// 0-1 BFS on the reverse graph, as two alternating frontiers
	var cur, next []int
	for q := range dist {
		if goal(q) {
			dist[q] = 0
			cur = append(cur, q)
		}
	}
	for d := 0; len(cur) > 0; d++ {
		for i := 0; i < len(cur); i++ { // cur grows while epsilon edges are followed
			q := cur[i]
			if dist[q] != d {
				continue
			}
			for _, f := range n.reps[q] {
				if dist[f] > d {
					dist[f] = d
					cur = append(cur, f)
				}
			}
			for _, f := range n.rout[q] {
				if dist[f] > d+1 {
					dist[f] = d + 1
					next = append(next, f)
				}
			}
		}
		cur, next = next, nil
	}
	return dist
}

// insertions walks a cheapest insertion path from cur (closed) to a goal state
// given dist, returning the alternative labels at each step and the closed
// state set reached.
func (n *nfa) insertions(cur map[int]bool, dist []int) (steps [][]*nfaLabel, reached map[int]bool) {
	d := nfaInf
	for q := range cur {
		if dist[q] < d {
			d = dist[q]
		}
	}
	if d >= nfaInf {
		return nil, cur
	}
	for ; d > 0; d-- {
		var alts []*nfaLabel
		targets := map[int]bool{}
		for _, q := range sortedStates(cur) {
			if dist[q] != d {
				continue
			}
			for _, e := range n.out[q] {
				if dist[e.to] == d-1 {
					alts = append(alts, e.lab)
					targets[e.to] = true
				}
			}
		}
		steps = append(steps, alts)
		cur = n.closure(targets)
	}
	return steps, cur
}

// cardDefect is one finding of explainContent.
type cardDefect struct {
	kind string // "missing" | "too-many" | "unexpected" | "order"
	kid  *xmltree.Node
	alts []*nfaLabel // missing: the alternatives; order: what was expected
	prev *nfaLabel   // too-many: the label the earlier occurrences used
	grp  *ancEntry   // too-many: the group that occurs more often than allowed
}

// explainContent names what keeps kids from satisfying p, or returns nil when
// it cannot (an unsupported model, or the automaton accepts what the matcher
// refused — in both cases the caller reports the generic mismatch).
func (s *Schema) explainContent(p *particle, kids []*xmltree.Node) []cardDefect {
	if p == nil || len(kids) > 5000 {
		return nil
	}
	n, ok := buildNFA(p)
	if !ok {
		return nil
	}
	var defects []cardDefect
	declared := map[*ElementDecl]bool{}
	for q := range n.out {
		for _, e := range n.out[q] {
			if e.lab.decl != nil {
				declared[e.lab.decl] = true
			}
		}
	}
	cur := n.closure(map[int]bool{n.start: true})
	var consumed []*nfaLabel
	var lastLabs []*nfaLabel        // the labels the previous child was matched by
	moved := map[*ElementDecl]int{} // elements reported as misplaced, expected to turn up later
	insert := func(steps [][]*nfaLabel) {
		for _, alts := range steps {
			defects = append(defects, cardDefect{kind: "missing", alts: alts})
		}
	}
	for i, kid := range kids {
		if len(defects) >= nfaMaxErrors {
			return defects
		}
		nn := nameOf(kid)
		if next, labs := n.step(s, cur, nn); len(next) > 0 {
			cur = next
			consumed = append(consumed, labs...)
			lastLabs = labs
			continue
		}
		// a misplaced element reported earlier turns up where it belongs
		var mv *ElementDecl
		for d, c := range moved {
			if c > 0 && s.matchesElementName(d, nn) {
				mv = d
			}
		}
		if mv != nil {
			moved[mv]--
			continue
		}
		dist := n.distToGoal(func(q int) bool {
			for _, e := range n.out[q] {
				if s.labelMatches(e.lab, nn) {
					return true
				}
			}
			return false
		})
		best := nfaInf
		for q := range cur {
			if dist[q] < best {
				best = dist[q]
			}
		}
		if best < nfaInf {
			steps, reached := n.insertions(cur, dist)
			later := false
			if len(steps) > 0 {
				for _, alt := range steps[0] {
					for _, k := range kids[i+1:] {
						if s.labelMatches(alt, nameOf(k)) {
							later = true
						}
					}
				}
			}
			if later {
				// the element that should come first is there, just later:
				// this child is out of order rather than the other missing
				defects = append(defects, cardDefect{kind: "order", kid: kid, alts: steps[0]})
				for _, alt := range steps[0] {
					if alt.decl != nil {
						moved[alt.decl]++
					}
				}
				insert(steps[1:])
			} else {
				insert(steps)
			}
			next, labs := n.step(s, reached, nn)
			cur = next
			consumed = append(consumed, labs...)
			lastLabs = labs
			continue
		}
		// nothing can be inserted to make the child acceptable: it is surplus
		var prev *nfaLabel
		for j := len(consumed) - 1; j >= 0; j-- {
			if s.labelMatches(consumed[j], nn) {
				prev = consumed[j]
				break
			}
		}
		var grp *ancEntry
		// The element's own wording wins; otherwise a group that has used up its
		// occurrences (below the content model itself) is the better culprit.
		if prev == nil || len(cardMessages("too-many", []*nfaLabel{prev})) == 0 {
			grp = s.exhaustedGroup(p, lastLabs, nn)
		}
		switch {
		case prev != nil && grp == nil:
			defects = append(defects, cardDefect{kind: "too-many", kid: kid, prev: prev})
		case grp != nil:
			defects = append(defects, cardDefect{kind: "too-many", kid: kid, prev: prev, grp: grp})
		default:
			known := false
			for d := range declared {
				if s.matchesElementName(d, nn) {
					known = true
					break
				}
			}
			d := cardDefect{kind: "unexpected", kid: kid}
			if known {
				d.kind = "order"
				d.alts = n.expectedFrom(cur)
			}
			defects = append(defects, d)
		}
	}
	if !cur[n.accept] {
		dist := n.distToGoal(func(q int) bool { return q == n.accept })
		steps, _ := n.insertions(cur, dist)
		insert(steps)
	}
	return defects
}

// expectedFrom lists the labels that could be consumed next from cur.
func (n *nfa) expectedFrom(cur map[int]bool) []*nfaLabel {
	var out []*nfaLabel
	seen := map[*ElementDecl]bool{}
	for _, q := range sortedStates(cur) {
		for _, e := range n.out[q] {
			if e.lab.decl != nil {
				if seen[e.lab.decl] {
					continue
				}
				seen[e.lab.decl] = true
			}
			out = append(out, e.lab)
		}
	}
	return out
}

// --- turning findings into diagnostics --------------------------------------

// displayName renders an expanded name the way the instance writes it: the
// local name when the namespace is the default one there, prefix:local when a
// prefix is bound to it, Clark notation otherwise.
func displayName(el *xmltree.Node, n xname) string {
	if n.Space == "" {
		return n.Local
	}
	ns := el.InScopeNamespaces()
	if ns[""] == n.Space {
		return n.Local
	}
	var prefixes []string
	for p, u := range ns {
		if p != "" && u == n.Space {
			prefixes = append(prefixes, p)
		}
	}
	if len(prefixes) > 0 {
		sort.Strings(prefixes)
		return prefixes[0] + ":" + n.Local
	}
	return "{" + n.Space + "}" + n.Local
}

func (l *nfaLabel) display(el *xmltree.Node) string {
	if l.decl != nil {
		return displayName(el, l.decl.name)
	}
	return "(any element)"
}

func joinAlts(el *xmltree.Node, alts []*nfaLabel, max int) string {
	var names []string
	seen := map[string]bool{}
	for _, a := range alts {
		nm := a.display(el)
		if !seen[nm] {
			seen[nm] = true
			names = append(names, nm)
		}
	}
	more := false
	if max > 0 && len(names) > max {
		names, more = names[:max], true
	}
	s := strings.Join(names, " or ")
	if more {
		s += " or …"
	}
	return s
}

// cardMessages picks the msg:message entries that apply to a cardinality error
// of kind on the given labels: the declaration's own, else those of the
// innermost group that holds them all (an xs:choice's appinfo speaks for the
// alternatives).
func cardMessages(kind string, labs []*nfaLabel) []assertMsg {
	filter := func(ms []assertMsg) []assertMsg {
		var out []assertMsg
		for _, m := range ms {
			if m.on == "" || m.on == kind {
				out = append(out, m)
			}
		}
		return out
	}
	if len(labs) == 0 {
		return nil
	}
	distinct := map[*ElementDecl]bool{}
	for _, l := range labs {
		distinct[l.decl] = true
	}
	if len(distinct) == 1 && labs[0].decl != nil {
		if ms := filter(labs[0].decl.msgs); len(ms) > 0 {
			return ms
		}
	}
	for _, g := range labs[0].anc {
		inAll := true
		for _, l := range labs[1:] {
			found := false
			for _, a := range l.anc {
				if a.g == g.g {
					found = true
					break
				}
			}
			if !found {
				inAll = false
				break
			}
		}
		if inAll {
			if ms := filter(g.g.msgs); len(ms) > 0 {
				return ms
			}
		}
	}
	return nil
}

func clark(n xname) string {
	if n.Space == "" {
		return n.Local
	}
	return "{" + n.Space + "}" + n.Local
}

// cardinalityErrors turns the findings for the children kids of el into
// diagnostics, or nil if there is nothing to report.
func (s *Schema) cardinalityErrors(el *xmltree.Node, p *particle, kids []*xmltree.Node) []*Error {
	defects := s.explainContent(p, kids)
	if len(defects) == 0 {
		return nil
	}
	parent := displayName(el, nameOf(el))
	var out []*Error
	for _, d := range defects {
		e := &Error{Code: "cvc-complex-type.2.4", Kind: d.kind}
		switch d.kind {
		case "missing":
			e.Line, e.Col, e.Path = int(el.Line), int(el.Col), elementPath(el)
			e.Message = "element " + joinAlts(el, d.alts, 6) + " is missing in " + parent
			if d.alts[0].decl != nil {
				e.Particle = clark(d.alts[0].decl.name)
			}
			e.msgs = cardMessages("missing", d.alts)
		case "too-many":
			k := d.kid
			e.Line, e.Col, e.Path = int(k.Line), int(k.Col), elementPath(k)
			e.Particle = clark(nameOf(k))
			if d.grp != nil {
				var names []string
				groupElementNames(el, d.grp.g, &names, map[*modelGroup]bool{})
				if len(names) > 6 {
					names = append(names[:6], "…")
				}
				e.Message = "element " + displayName(el, nameOf(k)) + ": the " + groupKindWord(d.grp.g) + " of " +
					strings.Join(names, ", ") + " occurs more often than allowed (maximum " + strconv.Itoa(d.grp.p.max) + ") in " + parent
				for _, m := range d.grp.g.msgs {
					if m.on == "" || m.on == "too-many" {
						e.msgs = append(e.msgs, m)
					}
				}
				break
			}
			limit := ""
			if d.prev.part != nil && d.prev.part.max != unbounded {
				limit = " (maximum " + strconv.Itoa(d.prev.part.max) + ")"
			}
			e.Message = "element " + displayName(el, nameOf(k)) + " occurs more often than allowed" + limit + " in " + parent
			e.msgs = cardMessages("too-many", []*nfaLabel{d.prev})
		case "unexpected":
			k := d.kid
			e.Line, e.Col, e.Path = int(k.Line), int(k.Col), elementPath(k)
			e.Particle = clark(nameOf(k))
			e.Message = "element " + displayName(el, nameOf(k)) + " is not allowed in " + parent
		case "order":
			k := d.kid
			e.Line, e.Col, e.Path = int(k.Line), int(k.Col), elementPath(k)
			e.Particle = clark(nameOf(k))
			e.Message = "element " + displayName(el, nameOf(k)) + " is not allowed at this position in " + parent
			if len(d.alts) > 0 {
				e.Message += ", expected " + joinAlts(el, d.alts, 6)
			}
		}
		out = append(out, e)
	}
	return out
}

// exhaustedGroup finds the enclosing group of the child just consumed that has
// used up its occurrences although the next child would be valid as the start
// of one more iteration of it — a second alternative of a choice that may occur
// once, say. That child is "too many" of the group, not out of order.
func (s *Schema) exhaustedGroup(root *particle, last []*nfaLabel, nn xname) *ancEntry {
	for _, l := range last {
		for i := range l.anc {
			a := l.anc[i]
			if a.p == nil || a.p == root || a.p.max == unbounded {
				continue
			}
			n, ok := buildNFA(&particle{min: 1, max: 1, term: a.g})
			if !ok {
				continue
			}
			start := n.closure(map[int]bool{n.start: true})
			if next, _ := n.step(s, start, nn); len(next) > 0 {
				return &a
			}
		}
	}
	return nil
}

// groupElementNames lists the element names a group is made of, depth first,
// for "the choice of A, B".
func groupElementNames(el *xmltree.Node, g *modelGroup, out *[]string, seen map[*modelGroup]bool) {
	if seen[g] {
		return
	}
	seen[g] = true
	for _, p := range g.particles {
		switch t := p.term.(type) {
		case *ElementDecl:
			*out = append(*out, displayName(el, t.name))
		case *modelGroup:
			groupElementNames(el, t, out, seen)
		}
	}
}

func groupKindWord(g *modelGroup) string {
	switch g.compositor {
	case cChoice:
		return "choice"
	case cAll:
		return "all group"
	}
	return "sequence"
}
