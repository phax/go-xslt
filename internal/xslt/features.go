package xslt

import (
	"runtime"
	"sync"
	"weak"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// Features are the optional XSLT 3.0 capabilities (§26) a stylesheet is
// compiled with. They belong to the compiled Stylesheet, not to the process:
// two stylesheets compiled with different Features can run side by side.
type Features struct {
	// SchemaAware: xsl:import-schema, [xsl:]validation and [xsl:]type, typed
	// nodes, system-property('xsl:is-schema-aware') = "yes".
	SchemaAware bool
	// BackwardsCompatible: genuine XSLT 1.0 behaviour where [xsl:]version is
	// below 2.0 (§3.9.1), instead of XSLT 3.0 rules.
	BackwardsCompatible bool
	// StrictStreaming: a construct declared streamable that the analysis
	// cannot prove guaranteed-streamable is XTSE3430, instead of running
	// with the whole document in memory.
	StrictStreaming bool
	// StreamingClaim is what system-property('xsl:supports-streaming')
	// reports. Streamed execution itself is always available.
	StreamingClaim bool
}

// DefaultFeatures are the Features of a compile that is not given any: the
// process-wide settings of SetSchemaAware, SetBackwardsCompatible,
// SetEnforceStreamability and SetStreamingClaim. Those setters exist for
// hosts that configure the whole process at once (the conformance harness);
// everything else passes Features with the compile (CompileWith).
func DefaultFeatures() Features {
	return Features{
		SchemaAware:         schemaAware.Load(),
		BackwardsCompatible: bcCompatClaim.Load(),
		StrictStreaming:     strmEnforce.Load(),
		StreamingClaim:      strmClaim.Load(),
	}
}

// featureRegistry maps the document node of every stylesheet module to the
// Features it was compiled with, so code that only holds a stylesheet node
// (validateTree, typeAvailable, …) finds them. The key is a weak pointer and
// the entry is removed when the tree is collected: a long-running host that
// compiles stylesheet after stylesheet must not keep every one of them alive.
var featureRegistry sync.Map // weak.Pointer[xmltree.Node] -> Features

// registerFeatures records f for the stylesheet module whose tree contains n.
func registerFeatures(n *xmltree.Node, f Features) {
	root := rootOfNode(n)
	if root == nil {
		return
	}
	weakStore(&featureRegistry, root, f)
}

// featuresAt returns the Features of the stylesheet el was written in, or the
// process-wide defaults for a node of no compiled stylesheet.
func featuresAt(el *xmltree.Node) Features {
	if root := rootOfNode(el); root != nil {
		if v, ok := featureRegistry.Load(weak.Make(root)); ok {
			return v.(Features)
		}
	}
	return DefaultFeatures()
}

// weakStore stores v under a weak key for root in m and removes the entry
// once root is no longer reachable. v must not reference root, or the entry
// would keep it alive.
func weakStore(m *sync.Map, root *xmltree.Node, v any) {
	k := weak.Make(root)
	if _, loaded := m.Swap(k, v); !loaded {
		runtime.AddCleanup(root, func(k weak.Pointer[xmltree.Node]) { m.Delete(k) }, k)
	}
}
