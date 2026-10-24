package xslt

import (
	"path/filepath"
	"strings"
	"testing"
)

// XSLT 3.0 §5.3.? xsl:global-context-item declarations and §20.? dynamic
// calls of the XSLT-defined functions.
func TestCompileSpecGlobalContextItemDecls(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "gci.xsl", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:global-context-item use="optional"/></xsl:stylesheet>`)
	specWrite(t, dir, "gci-same.xsl", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:global-context-item use="required"/></xsl:stylesheet>`)
	specRunAll(t, dir, []specCase{
		// XTSE3087: more than one in a module, or inconsistent across modules.
		{name: "two in one module", code: "XTSE3087", sheet: `<xsl:global-context-item/><xsl:global-context-item/><xsl:template match="/">x</xsl:template>`},
		{name: "inconsistent across modules", code: "XTSE3087", sheet: `<xsl:include href="gci.xsl"/><xsl:global-context-item use="required"/><xsl:template match="/">x</xsl:template>`},
		{name: "consistent across modules", want: "x", sheet: `<xsl:include href="gci-same.xsl"/><xsl:global-context-item use="required"/><xsl:template match="/">x</xsl:template>`},
		{name: "function-lookup of an XSLT function", want: "true 3.0", sheet: `<xsl:template match="/"><xsl:value-of select="exists(function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'available-system-properties'), 0)), function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'system-property'), 1)('xsl:version')"/></xsl:template>`},
		{name: "function-lookup with a wrong arity", want: "false", sheet: `<xsl:template match="/"><xsl:value-of select="exists(function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'system-property'), 3))"/></xsl:template>`},
	})
}

// XSLT 3.0 §3.5.3.? overriding an attribute set of a used package, with
// xsl:original naming the overridden declaration.
func TestCompileSpecAttributeSetOverride(t *testing.T) {
	dir := t.TempDir()
	specWrite(t, dir, "lib.xsl", `<xsl:package name="urn:lib" version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:attribute-set name="s" visibility="public"><xsl:attribute name="a">lib</xsl:attribute></xsl:attribute-set>
</xsl:package>`)
	pkgs := []PackageSource{{Name: "urn:lib", Path: filepath.Join(dir, "lib.xsl")}}
	cases := []struct {
		name, sheet, want string
	}{
		{"original is included", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:use-package name="urn:lib"><xsl:override><xsl:attribute-set name="s" use-attribute-sets="xsl:original" visibility="public"><xsl:attribute name="b">top</xsl:attribute></xsl:attribute-set></xsl:override></xsl:use-package>
<xsl:output method="text"/>
<xsl:template match="/"><xsl:variable name="e"><e xsl:use-attribute-sets="s"/></xsl:variable><xsl:value-of select="sort($e/e/@*, (), name#1) ! (name(.) || '=' || .)"/></xsl:template></xsl:stylesheet>`, "a=lib b=top"},
		{"override replaces", `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:use-package name="urn:lib"><xsl:override><xsl:attribute-set name="s" visibility="public"><xsl:attribute name="b">top</xsl:attribute></xsl:attribute-set></xsl:override></xsl:use-package>
<xsl:output method="text"/>
<xsl:template match="/"><xsl:variable name="e"><e xsl:use-attribute-sets="s"/></xsl:variable><xsl:value-of select="sort($e/e/@*, (), name#1) ! (name(.) || '=' || .)"/></xsl:template></xsl:stylesheet>`, "b=top"},
	}
	for _, c := range cases {
		ss, err := CompileAtWithPackages(c.sheet, filepath.Join(dir, "main.xsl"), nil, pkgs)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		out, _, err := ss.Transform("<doc/>", nil)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
		} else if strings.TrimSpace(out) != c.want {
			t.Errorf("%s: got %q want %q", c.name, out, c.want)
		}
	}
}
