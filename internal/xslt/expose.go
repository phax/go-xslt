package xslt

import (
	"strings"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// ---------------------------------------------------------------------------
// xsl:expose (XSLT 3.0 §3.5.3.1): the package manifest's rules for the
// visibility of the package's own components.
//
// The visibility of a named template, function, variable, attribute set or
// mode is the first of these that applies:
//
//  1. the LAST xsl:expose that names the component explicitly (an EQName or a
//     NamedFunctionRef, not a wildcard);
//  2. the component's own @visibility;
//  3. the last xsl:expose that matches it by a partial wildcard (p:*, *:local,
//     Q{uri}*);
//  4. the last xsl:expose whose names is the bare wildcard *;
//  5. private.
//
// An explicit xsl:expose together with an explicit @visibility must agree
// (XTSE3010) according to the §3.5.3.1 table: an xsl:expose can lower a
// component's visibility but never raise it.
// ---------------------------------------------------------------------------

// exposeVisibilityKinds are the legal xsl:expose/@component values.
var exposeKinds = map[string]bool{
	"template": true, "function": true, "attribute-set": true,
	"variable": true, "mode": true, "*": true,
}

// exposeVisibilities are the legal xsl:expose/@visibility values.
var exposeVisibilities = map[string]bool{
	"public": true, "private": true, "final": true, "abstract": true,
}

// exposePermitted implements the consistency table: may an xsl:expose that
// explicitly names a component with exposed visibility `exposed` be combined
// with a declaration whose own @visibility is `declared`?
func exposePermitted(exposed, declared string) bool {
	switch exposed {
	case "public":
		return declared == "public"
	case "private":
		return declared == "public" || declared == "private" || declared == "final"
	case "final":
		return declared == "public" || declared == "final"
	case "abstract":
		return declared == "abstract"
	}
	return false
}

// exposeToken is one parsed token of xsl:expose/@names.
type exposeToken struct {
	text     string
	wildcard int // 0 explicit name, 1 partial wildcard (p:*, *:l, Q{u}*), 2 bare *
}

func classifyExposeToken(tok string) exposeToken {
	switch {
	case tok == "*":
		return exposeToken{tok, 2}
	case strings.Contains(tok, "*"):
		return exposeToken{tok, 1}
	}
	return exposeToken{tok, 0}
}

// packageExposure computes the effective visibility of every named component
// declared directly in the package manifest root, as a map keyed by the
// declaration element. It returns nil (no override: declared visibility, else
// private, stands) when the package has no xsl:expose. xsl:param components and
// the unnamed mode are never touched by an xsl:expose.
func packageExposure(root *xmltree.Node) (map[*xmltree.Node]string, map[string]string, error) {
	var exposes []*xmltree.Node
	var comps []pkgComponent
	for _, ch := range elementChildren(root) {
		if ch.Name.Space != NS {
			continue
		}
		if ch.Name.Local == "expose" {
			exposes = append(exposes, ch)
			continue
		}
		comp, ok := componentOf(ch)
		if !ok || ch.Name.Local == "param" {
			continue // an xsl:param is always public; no xsl:expose matches it
		}
		if comp.kind == "mode" && comp.name == "" {
			continue // the unnamed mode is always private
		}
		comps = append(comps, comp)
	}
	if len(exposes) == 0 {
		return nil, nil, nil
	}
	// Modes used only implicitly (a template's @mode, no xsl:mode element) are
	// components too as far as xsl:expose is concerned (package-001j): the
	// result for them is reported by name, since there is no declaration to
	// key it by.
	declaredModes := map[string]bool{}
	for _, comp := range comps {
		if comp.kind == "mode" {
			declaredModes[comp.name] = true
		}
	}
	var implicit []pkgComponent
	seenImplicit := map[string]bool{}
	for _, ch := range elementChildren(root) {
		if ch.Name.Space != NS || ch.Name.Local != "template" {
			continue
		}
		m, ok := ch.AttrLocal("mode")
		if !ok {
			continue
		}
		for _, tok := range strings.Fields(m) {
			if tok == "#all" || tok == "#default" || tok == "#unnamed" || tok == "#current" {
				continue
			}
			name := clarkName(resolveQName(ch, tok))
			if declaredModes[name] || seenImplicit[name] {
				continue
			}
			seenImplicit[name] = true
			implicit = append(implicit, pkgComponent{kind: "mode", name: name, el: &xmltree.Node{Kind: xmltree.KindElement}})
		}
	}
	comps = append(comps, implicit...)

	type rule struct {
		el     *xmltree.Node
		kind   string
		vis    string
		tokens []exposeToken
	}
	var rules []rule
	for _, ex := range exposes {
		kind := strings.TrimSpace(attrOr(ex, "component", ""))
		vis := strings.TrimSpace(attrOr(ex, "visibility", ""))
		if !exposeKinds[kind] {
			return nil, nil, errAt(ex, "err:XTSE0020: invalid xsl:expose/@component %q", kind)
		}
		if !exposeVisibilities[vis] {
			return nil, nil, errAt(ex, "err:XTSE0020: invalid xsl:expose/@visibility %q", vis)
		}
		r := rule{el: ex, kind: kind, vis: vis}
		for _, tok := range strings.Fields(attrOr(ex, "names", "")) {
			if strings.HasPrefix(tok, "#") {
				// "#unnamed" is not an EQName: the unnamed mode cannot be named.
				return nil, nil, errAt(ex, "err:XTSE0020: %q is not a valid name in xsl:expose/@names", tok)
			}
			if i := strings.IndexByte(tok, ':'); i > 0 && !strings.HasPrefix(tok, "Q{") && !strings.HasPrefix(tok, "*:") {
				if _, ok := ex.LookupPrefix(tok[:i]); !ok {
					return nil, nil, errAt(ex, "err:XTSE0020: no namespace is declared for the prefix of %q in xsl:expose/@names", tok)
				}
			}
			t := classifyExposeToken(tok)
			if kind == "*" && t.wildcard == 0 {
				return nil, nil, errAt(ex, "err:XTSE3022: xsl:expose component=\"*\" requires a wildcard in @names, not %q", tok)
			}
			r.tokens = append(r.tokens, t)
		}
		rules = append(rules, r)
	}

	// matches reports whether a rule's token selects comp.
	matches := func(r rule, t exposeToken, comp pkgComponent) bool {
		if r.kind != "*" && r.kind != comp.kind {
			return false
		}
		if t.wildcard == 0 && comp.kind == "function" && !strings.Contains(t.text, "#") {
			return false // an explicit function name must carry its arity
		}
		if t.wildcard == 0 && comp.kind != "function" && strings.Contains(t.text, "#") {
			return false
		}
		return acceptTokenMatches(r.el, t.text, comp)
	}

	// XTSE3020: every explicit token must select at least one component.
	for _, r := range rules {
		for _, t := range r.tokens {
			if t.wildcard != 0 {
				continue
			}
			found := false
			for _, comp := range comps {
				if matches(r, t, comp) {
					found = true
					break
				}
			}
			if !found {
				return nil, nil, errAt(r.el, "err:XTSE3020: xsl:expose names %q, which matches no %s in the package", t.text, kindLabel(r.kind))
			}
		}
	}

	out := make(map[*xmltree.Node]string, len(comps))
	implicitOut := map[string]string{}
	for _, comp := range comps {
		var explicit, partial, all string
		for _, r := range rules { // document order: the last match wins
			for _, t := range r.tokens {
				if !matches(r, t, comp) {
					continue
				}
				switch t.wildcard {
				case 0:
					explicit = r.vis
				case 1:
					partial = r.vis
				default:
					all = r.vis
				}
			}
		}
		declared := ""
		if v, ok := comp.el.AttrLocal("visibility"); ok {
			declared = strings.TrimSpace(v)
		}
		if explicit != "" && declared != "" && !exposePermitted(explicit, declared) {
			return nil, nil, errAt(comp.el, "err:XTSE3010: xsl:expose makes this %s %s, which is inconsistent with its declared visibility %q", comp.kind, explicit, declared)
		}
		vis := "private"
		switch {
		case explicit != "":
			vis = explicit
		case declared != "":
			vis = declared
		case partial != "":
			vis = partial
		case all != "":
			vis = all
		}
		if vis == "abstract" && declared != "abstract" {
			return nil, nil, errAt(comp.el, "err:XTSE3025: xsl:expose cannot make this %s abstract; it was not declared with visibility=\"abstract\"", comp.kind)
		}
		if seenImplicit[comp.name] && comp.kind == "mode" && len(comp.el.Children) == 0 && comp.el.Parent == nil {
			implicitOut[comp.name] = vis
			continue
		}
		out[comp.el] = vis
	}
	return out, implicitOut, nil
}

func kindLabel(kind string) string {
	if kind == "*" {
		return "component"
	}
	return kind
}

// applyExposure writes the effective visibilities packageExposure computed
// onto the compiled templates, functions and modes, which is where the
// entry-point checks (initial template, initial mode, xsl:evaluate) read them.
func applyExposure(ss *Stylesheet, root *xmltree.Node) error {
	vis, implicit, err := packageExposure(root)
	if err != nil || vis == nil {
		return err
	}
	for name, v := range implicit {
		if v == "private" {
			if ss.implicitPrivateModes == nil {
				ss.implicitPrivateModes = map[string]bool{}
			}
			ss.implicitPrivateModes[name] = true
		}
	}
	for _, t := range ss.templates {
		if v, ok := vis[t.el]; ok && t.name != "" {
			t.visibility = v
		}
	}
	for _, fd := range ss.functions {
		if v, ok := vis[fd.el]; ok {
			fd.visibility = v
		}
	}
	for _, md := range ss.modes {
		if v, ok := vis[md.el]; ok {
			md.visibility = v
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Hiding. Execution flattens a used package's modules into the using
// package's module list, so every component of the used package would be
// reachable by name. What the used package does NOT make visible — its private
// and hidden components, and anything the using package's xsl:accept lowers to
// hidden — must still be unreachable from the using package's own code
// (XSLT 3.0 §3.5.3): the reference is to an undeclared component, i.e. the same
// static/dynamic error as for any other undeclared name (XPST0017 function,
// XPST0008 variable, XTSE0650 template, XTSE0710 attribute set).
//
// hiddenAt records, per package manifest root, the symbolic names (see
// pkgComponent.symbolic) hidden from code written in that package; the check
// sits at the points where a name is resolved (hiddenRef).
// ---------------------------------------------------------------------------

// usedVisibility is what one xsl:use-package element exposes to its using
// package, as symbolic-name sets.
type usedVisibility struct {
	hidden, visible map[string]bool
	// abstract: components accepted AS abstract (an explicit accept said so);
	// explicit: components some xsl:accept names explicitly or by a wildcard
	// that applies, i.e. whose visibility here was stated rather than defaulted.
	abstract, explicit map[string]bool
}

// noteUsedVisibility records the hidden/visible split of the package usedRoot
// as seen through the xsl:use-package element use (whose exports are exports,
// after the used package's own xsl:expose rules).
func (c *compiler) noteUsedVisibility(use, usedRoot *xmltree.Node, exports []pkgComponent) {
	using := use.Parent
	if using == nil {
		return
	}
	if c.usesOf == nil {
		c.usesOf = map[*xmltree.Node][]*xmltree.Node{}
	}
	c.usesOf[using] = append(c.usesOf[using], usedRoot)
	uv := usedVisibility{hidden: map[string]bool{}, visible: map[string]bool{}, abstract: map[string]bool{}, explicit: map[string]bool{}}
	exported := map[string]bool{}
	for _, comp := range exports {
		exported[comp.symbolic()] = true
		uv.visible[comp.symbolic()] = true
	}
	// An xsl:accept that lowers an exported component to hidden removes it.
	for _, comp := range exports {
		final, stated := acceptFinal(use, comp)
		if stated {
			uv.explicit[comp.symbolic()] = true
		}
		if final == "abstract" {
			uv.abstract[comp.symbolic()] = true
		}
		if final == "hidden" {
			delete(uv.visible, comp.symbolic())
			uv.hidden[comp.symbolic()] = true
		}
	}
	// Everything the used package declares but does not export is hidden.
	for _, ch := range elementChildren(usedRoot) {
		if ch.Name.Space != NS || ch.Name.Local == "param" {
			continue
		}
		comp, ok := componentOf(ch)
		if !ok || (comp.kind == "mode" && comp.name == "") {
			continue
		}
		if !exported[comp.symbolic()] {
			uv.hidden[comp.symbolic()] = true
		}
	}
	if c.useVis == nil {
		c.useVis = map[*xmltree.Node][]usedVisibility{}
	}
	c.useVis[using] = append(c.useVis[using], uv)
}

// finishHidden turns the per-use-package sets recorded for root into the final
// hidden set: a name is hidden unless some used package makes it visible or the
// using package declares a component of that name itself (then the reference
// means that one).
func (c *compiler) finishHidden(root *xmltree.Node, own, incRoots []*xmltree.Node) error {
	infos := c.useVis[root]
	if len(infos) == 0 {
		return nil
	}
	ownDeclared := map[string]bool{}
	for _, ch := range own {
		if comp, ok := componentOf(ch); ok {
			ownDeclared[comp.symbolic()] = true
		}
	}
	visible := map[string]bool{}
	for _, uv := range infos {
		for sym := range uv.visible {
			visible[sym] = true
		}
	}
	// XTSE3080: a top-level package may not contain an abstract component,
	// referenced or not: neither one accepted as abstract and left unoverridden
	// nor one it declares abstract itself.
	if root == c.principalRoot {
		for _, uv := range infos {
			for sym := range uv.abstract {
				if !ownDeclared[sym] {
					return errAt(root, "err:XTSE3080: a top-level package cannot contain the abstract component %s", sym)
				}
			}
		}
	}
	if root == c.principalRoot {
		c.acceptedExplicit = map[string]bool{}
		for _, uv := range infos {
			for sym := range uv.explicit {
				c.acceptedExplicit[sym] = true
			}
		}
	}
	// XTSE3050: a component this package declares OUTSIDE xsl:override may not
	// have the symbolic name of a component it accepts from a used package —
	// overriding is only done with xsl:override. A template rule joins a mode
	// the same way (override-m-018).
	for _, ch := range own {
		if c.overrideNodes[ch] {
			continue
		}
		if comp, ok := componentOf(ch); ok && ch.Name.Local != "param" && visible[comp.symbolic()] {
			return errAt(ch, "err:XTSE3050: %s %q is accepted from a used package and cannot be redeclared outside xsl:override", comp.kind, comp.name)
		}
		if ch.Name.Space == NS && ch.Name.Local == "template" {
			if _, hasMatch := ch.AttrLocal("match"); hasMatch {
				for _, tok := range strings.Fields(attrOr(ch, "mode", "")) {
					if tok == "#all" || tok == "#default" || tok == "#unnamed" || tok == "#current" {
						continue
					}
					if visible["mode#"+clarkName(resolveQName(ch, tok))] {
						return errAt(ch, "err:XTSE3050: mode %q comes from a used package; template rules are added to it with xsl:override", tok)
					}
				}
			}
		}
	}
	final := map[string]bool{}
	for _, uv := range infos {
		for sym := range uv.hidden {
			if !visible[sym] && !ownDeclared[sym] {
				final[sym] = true
			}
		}
	}
	if len(final) == 0 {
		// Nothing hidden for the whole package: drop what the nested
		// modules' own (partial) passes recorded.
		delete(c.hiddenAt, root)
		for _, r := range incRoots {
			delete(c.hiddenAt, r)
		}
		return nil
	}
	if c.hiddenAt == nil {
		c.hiddenAt = map[*xmltree.Node]map[string]bool{}
	}
	c.hiddenAt[root] = final
	for _, r := range incRoots {
		c.hiddenAt[r] = final
	}
	return nil
}

// mergeUseVis folds the per-use-package visibility recorded for an included or
// imported module into its including package's.
func mergeUseVis(m map[*xmltree.Node][]usedVisibility, root, sub *xmltree.Node) map[*xmltree.Node][]usedVisibility {
	if len(m[sub]) == 0 {
		return m
	}
	m[root] = append(m[root], m[sub]...)
	return m
}

// hiddenRef reports whether the symbolic name sym is hidden from code written
// at el, i.e. el sits in a package that uses another package which keeps that
// component to itself.
func (ss *Stylesheet) hiddenRef(el *xmltree.Node, sym string) bool {
	if len(ss.hiddenAt) == 0 || el == nil {
		return false
	}
	root := el
	for root.Parent != nil && root.Parent.Kind == xmltree.KindElement {
		root = root.Parent
	}
	return ss.hiddenAt[root][sym]
}

// globalCtxDecl is one xsl:global-context-item seen while compiling the
// principal package: the module it sits in and what it says.
type globalCtxDecl struct {
	top     *xmltree.Node
	use, as string
}

// inPrincipalPackage reports whether the module whose root element is top is
// part of the principal package (the root module or one it includes/imports),
// as opposed to a used library package.
func (c *compiler) inPrincipalPackage(top *xmltree.Node) bool {
	if c.principalRoot == nil || top == c.principalRoot {
		return true
	}
	for _, r := range c.incRoots[c.principalRoot] {
		if r == top {
			return true
		}
	}
	return false
}

// originalFunc resolves xsl:original(...) written at el: the function that the
// enclosing overriding xsl:function replaces, or nil when el is not inside one.
func (ss *Stylesheet) originalFunc(el *xmltree.Node, arity int) *FuncDef {
	if len(ss.originalFuncs) == 0 {
		return nil
	}
	for n := el; n != nil; n = n.Parent {
		if n.Kind == xmltree.KindElement && n.Name.Space == NS && n.Name.Local == "function" {
			if fd := ss.originalFuncs[n]; fd != nil && len(fd.params) == arity {
				return fd
			}
			return nil
		}
	}
	return nil
}

// noteOriginalVar records, for an xsl:override of a global variable/param, the
// declaration it replaces (the latest earlier one of the same name).
func (c *compiler) noteOriginalVar(ss *Stylesheet, vd *VarDef, el *xmltree.Node) {
	if !c.overrideNodes[el] {
		return
	}
	// A package that overrides a variable sees ITS override from its own code
	// even when another package's override of the same variable ends up the
	// global winner (the diamond case: two packages each override one used
	// package differently). Only consulted when it is not the winner.
	if c.multiPkg {
		if ss.overrideVars == nil {
			ss.overrideVars = map[string]map[string]*VarDef{}
		}
		pk := c.pkgKey(el)
		if ss.overrideVars[pk] == nil {
			ss.overrideVars[pk] = map[string]*VarDef{}
		}
		ss.overrideVars[pk][clark(vd.name.Space, vd.name.Local)] = vd
	}
	for i := len(ss.globals) - 1; i >= 0; i-- {
		if p := ss.globals[i]; p.name == vd.name {
			if ss.originalVars == nil {
				ss.originalVars = map[*xmltree.Node]*VarDef{}
			}
			ss.originalVars[el] = p
			return
		}
	}
}

// originalVar resolves $xsl:original written at el: the global variable the
// enclosing overriding declaration replaces.
func (ss *Stylesheet) originalVar(el *xmltree.Node) *VarDef {
	if len(ss.originalVars) == 0 {
		return nil
	}
	for n := el; n != nil; n = n.Parent {
		if n.Kind == xmltree.KindElement && n.Name.Space == NS && (n.Name.Local == "variable" || n.Name.Local == "param") {
			if vd := ss.originalVars[n]; vd != nil {
				return vd
			}
		}
	}
	return nil
}

// checkOwnAbstract is XTSE3080 for the components the principal package declares
// itself: it may not declare one abstract, referenced or not.
func checkOwnAbstract(roots []*xmltree.Node) error {
	for _, r := range roots {
		for _, ch := range elementChildren(r) {
			if comp, ok := componentOf(ch); ok && ch.Name.Local != "param" {
				if v, has := ch.AttrLocal("visibility"); has && strings.TrimSpace(v) == "abstract" {
					return errAt(ch, "err:XTSE3080: a top-level package cannot declare the abstract %s %q", comp.kind, comp.name)
				}
			}
		}
	}
	return nil
}

// acceptFinal is the visibility a used package's exported component ends up
// with under the xsl:accept children of use ("" = the component's own, kept):
// as with xsl:expose an accept naming the component explicitly beats a wildcard
// one, and the last of equals wins. An abstract component is only accepted as
// abstract — any other accepted visibility is an invalid combination that
// counts as no match (accept-911) — and with no usable accept it becomes
// hidden (accept-910). stated reports whether an accept applied.
func acceptFinal(use *xmltree.Node, comp pkgComponent) (final string, stated bool) {
	explicit, partial, all := "", "", ""
	for _, ac := range elementChildren(use) {
		if ac.Name.Space != NS || ac.Name.Local != "accept" {
			continue
		}
		vis := strings.TrimSpace(attrOr(ac, "visibility", ""))
		if !acceptMatches(ac, comp) {
			continue
		}
		// The most specific token of this accept that matches decides its class.
		class := 2
		for _, tok := range strings.Fields(attrOr(ac, "names", "")) {
			if !acceptTokenMatches(ac, tok, comp) {
				continue
			}
			switch {
			case tok == "*":
			case isWildcardToken(tok):
				if class > 1 {
					class = 1
				}
			default:
				class = 0
			}
		}
		switch class {
		case 0:
			explicit = vis
		case 1:
			partial = vis
		default:
			all = vis
		}
	}
	final = explicit
	if final == "" {
		final = partial
	}
	if final == "" {
		final = all
	}
	if comp.visibility == "abstract" && final != "abstract" {
		return "hidden", false
	}
	return final, final != ""
}
