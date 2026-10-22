# XSLT 3.0

By default go-xslt is an XSLT 3.0 **basic processor**, and 99.9% of the
applicable W3C test cases pass. Beyond that, it implements streaming,
schema-awareness and XSLT 1.0 backwards-compatible processing, three of the
optional features of the specification.

## Instructions

**Templates and modes.** `xsl:template` (match and named), `xsl:mode`
including `on-no-match`, priorities and the built-in template rules,
`xsl:apply-templates`, `xsl:next-match`, `xsl:apply-imports`, tunnel
parameters, and import precedence through `xsl:import` and `xsl:include`.

**Building the result.** `xsl:value-of`, `xsl:text`, `xsl:element`,
`xsl:attribute`, `xsl:namespace`, `xsl:comment`,
`xsl:processing-instruction`, `xsl:copy` (with `select`), `xsl:copy-of`,
`xsl:sequence`, `xsl:document`, `xsl:number`, `xsl:where-populated`,
attribute sets with `use-attribute-sets`, attribute value templates and text
value templates (`expand-text="yes"`).

**Control flow.** `xsl:if`, `xsl:choose`, `xsl:for-each`,
`xsl:iterate` with `xsl:next-iteration`, `xsl:break` and `xsl:on-completion`,
`xsl:try` and `xsl:catch`, `xsl:call-template`, `xsl:variable`, `xsl:param`,
`xsl:with-param`.

**Sorting and grouping.** `xsl:sort`, `xsl:perform-sort`,
`xsl:for-each-group` with all four grouping methods, and `xsl:merge` with
several sources and typed merge keys.

**Functions and keys.** `xsl:function` with overloading by arity and
`new-each-time`/`cache`, `xsl:key` with composite and collation-aware keys,
`xsl:accumulator`, `xsl:analyze-string`, `xsl:evaluate`.

**Maps.** `xsl:map` and `xsl:map-entry`.

**Output.** `xsl:output` with the methods `xml`, `html`, `text`, `json` and
`adaptive` and the full set of serialization parameters, `xsl:character-map`,
`xsl:result-document` for any number of secondary outputs, each with its own
serialization.

**Diagnostics.** `xsl:message` and `xsl:assert`.

**Static compilation.** Static parameters (`xsl:param static="yes"`), shadow
attributes (`_name="{…}"`) and `use-when`.

**Packages.** `xsl:package`, `xsl:use-package`, `xsl:override`, `xsl:accept`,
`xsl:expose` and `xsl:original`, with visibility rules. A package is a valid
top-level stylesheet just like `xsl:stylesheet`. Packages are executed in one
combined tree, so a used package's private components are hidden where names
are resolved, but they are not fully isolated at run time.

## Optional features

### Streaming

Guaranteed streaming runs in bounded memory. It is always available; see
[Streaming](/guide/streaming).

### Schema-awareness

`xsl:import-schema`, `validation` and `type` attributes that really validate,
typed input, and the sequence types `element(N, T)` and `schema-element(N)`.
This follows the definition of a schema-aware processor in XSLT 3.0 section
26.2: type errors are raised at run time. XPath's optional static typing
feature is not implemented.

### Backwards-compatible processing

When an `[xsl:]version` below 2.0 is in scope, the XSLT 1.0 rules for
comparisons, arithmetic and conversions apply to the instructions in that
scope, as section 3.9.1 describes.

::: info Turning the features on
Schema-awareness and backwards-compatible processing are off by default.
Their switches are currently internal and not yet part of the public `engine`
API; the conformance tests turn them on. Without them, `xsl:import-schema` is
rejected and a version below 2.0 is processed as 3.0.
:::

## Not implemented

- `exsl:node-set` and other EXSLT extensions. They are not part of a W3C
  specification, and XSLT 3.0 does not need them.
- XInclude.
- XPath's optional static typing feature.

## System properties

`system-property('xsl:supports-streaming')`,
`system-property('xsl:is-schema-aware')` and
`system-property('xsl:supports-backwards-compatibility')` report what the
processor currently claims.
