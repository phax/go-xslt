package xslt

import (
	"fmt"

	"github.com/tim-riep/go-xslt/internal/xmltree"
	"github.com/tim-riep/go-xslt/internal/xpath"
)

// ---------------------------------------------------------------------------
// Package-local declarations (XSLT 3.0 §3.5): keys, decimal formats,
// accumulators, namespace aliases, character maps and xsl:output are NOT
// shared between a package and the packages it uses — each package sees only
// its own. Execution flattens the packages' modules into one module list, so
// when more than one package takes part (Stylesheet.multiPkg) these
// declarations are filed under a name qualified by their package, and looked
// up from a call site with that site's package. With a single package every
// qualified name is the plain name, so nothing changes for ordinary
// stylesheets.
// ---------------------------------------------------------------------------

const pkgSep = "\x1f"

// pkgKeyOf is the qualifier for the package el is written in ("" when only one
// package is involved).
func pkgKeyOf(multi bool, pkgOf map[*xmltree.Node]*xmltree.Node, el *xmltree.Node) string {
	if !multi || el == nil {
		return ""
	}
	top := topElement(el)
	if p, ok := pkgOf[top]; ok {
		top = p
	}
	return fmt.Sprintf("%p", top)
}

func scopedName(pk, name string) string {
	if pk == "" {
		return name
	}
	return pk + pkgSep + name
}

func splitScoped(s string) (pk, name string) {
	for i := 0; i < len(s); i++ {
		if s[i] == pkgSep[0] {
			return s[:i], s[i+1:]
		}
	}
	return "", s
}

func (ss *Stylesheet) pkgKey(el *xmltree.Node) string { return pkgKeyOf(ss.multiPkg, ss.pkgOf, el) }
func (c *compiler) pkgKey(el *xmltree.Node) string    { return pkgKeyOf(c.multiPkg, c.pkgOf, el) }

// inPrincipal reports whether el is written in the principal package.
func (ss *Stylesheet) inPrincipal(el *xmltree.Node) bool {
	if ss.principalRoot == nil {
		return true
	}
	top := topElement(el)
	return top == ss.principalRoot || ss.principalMods[top]
}

// decimalFormatsFor is the set of xsl:decimal-format declarations visible to an
// expression written at el: its own package's when several packages take part.
func (ss *Stylesheet) decimalFormatsFor(el *xmltree.Node) map[string]xpath.DecimalFormat {
	if !ss.multiPkg {
		return ss.decFmts
	}
	return ss.decFmtsPkg[ss.pkgKey(el)]
}

// nsAliasTables returns the xsl:namespace-alias table (stylesheet URI -> result
// name) and its import precedences for the package el is written in.
func (c *compiler) nsAliasTables(el *xmltree.Node) (map[string]xmltree.Name, map[string]int) {
	if !c.multiPkg {
		if c.nsAlias == nil {
			c.nsAlias = map[string]xmltree.Name{}
			c.nsAliasPrec = map[string]int{}
		}
		return c.nsAlias, c.nsAliasPrec
	}
	pk := c.pkgKey(el)
	if c.nsAliasPkg == nil {
		c.nsAliasPkg = map[string]map[string]xmltree.Name{}
		c.nsAliasPrecPkg = map[string]map[string]int{}
	}
	if c.nsAliasPkg[pk] == nil {
		c.nsAliasPkg[pk] = map[string]xmltree.Name{}
		c.nsAliasPrecPkg[pk] = map[string]int{}
	}
	return c.nsAliasPkg[pk], c.nsAliasPrecPkg[pk]
}

// aliasesFor is the alias table in force for a literal result element.
func (c *compiler) aliasesFor(el *xmltree.Node) map[string]xmltree.Name {
	a, _ := c.nsAliasTables(el)
	return a
}
