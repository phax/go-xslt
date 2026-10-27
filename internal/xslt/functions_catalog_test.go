package xslt

import "testing"

// Every XSLT-defined function callFunc serves — the cases of its switch and
// the xsltFuncs registry — needs a signature in xsltSigSrc, so its arguments
// are checked by the function conversion rules and fn:function-available
// knows its arities.
func TestXSLTFunctionCatalogComplete(t *testing.T) {
	names := []string{
		"key", "regex-group", "current", "document", "current-output-uri",
		"copy-of", "snapshot", "available-system-properties", "system-property",
		"function-available", "element-available", "type-available",
		"unparsed-entity-uri", "unparsed-entity-public-id", "stream-available",
	}
	// current-map and last-map are engine-internal helpers of xsl:map (not
	// functions of the XSLT specification); fn:transform is a core F&O
	// function that the engine also serves here.
	internal := map[string]bool{"current-map": true, "last-map": true, "transform": true}
	for n := range xsltFuncs {
		if !internal[n] {
			names = append(names, n)
		}
	}
	for _, n := range names {
		if known, _ := xsltSigs.Has(n, 0); !known {
			t.Errorf("fn:%s is served by callFunc but has no signature in xsltSigSrc", n)
		}
	}
}
