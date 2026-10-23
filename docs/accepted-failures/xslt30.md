# XSLT 3.0 (`xslt30-test`): failing cases

Default mode (basic processor): 8,635 of 8,647 pass, 12 fail. Combined mode
(`XSLT30_FULL=1`, adding streaming, schema awareness and backwards
compatibility): 12,288 of 12,317 pass, 29 fail: the 12 below plus 17
combined-only cases. Measured offline.

## Default mode (also failing in combined mode)

### `unparsed-text-2002`, `unparsed-text-2003`
**Status:** Accepted (offline only).
Both read live pages from www.w3.org. The harness runs offline by default,
so that a result never depends on the network. With `XSLT30_NETWORK=1` both
pass.

### `base-uri-052`
**Status:** Accepted.
Needs XInclude, which go-xslt does not implement. XInclude is a separate
specification, and XSLT 3.0 does not require it.

### `collection-006`, `document-2401`, `document-2402`
**Status:** Accepted (decision).
`xsl:strip-space` must apply per package to documents loaded through
`fn:collection`, `document()` and `fn:doc`. That would mean carrying a
"current package" identity through every one of those calls down to the
resolver, a large change touching many call sites, for three tests. Declined.

### `output-0130`, `output-0715`
**Status:** Accepted (suite contradiction).
The reference files expect an XHTML `<meta http-equiv>` element in a form
that the reference processor itself no longer produces (checked against a live run).
The fixtures are stale; there is no single behaviour that matches both these
cases and the rest of the output test set.

### `copy-3002`
**Status:** Accepted (suite contradiction).
The catalog entry itself says "the test should fail".

### `copy-1220`, `copy-1221`
**Status:** Accepted.
Namespaces in scope through the environment are not carried onto a copied
element in the way these two cases expect. The only fix found so far loses
more cases than it gains, measured several times.

### `override-as-005`
**Status:** Accepted (decision).
Two packages each declare a *private* attribute set with the same name.
Attribute-set references are resolved without their package. Keeping the two
apart means threading the package through every `use-attribute-sets` site
and through the streamability analysis, for one test.

## Combined mode only

### `streamable-100`, `streamable-101`, `streamable-102`
**Status:** Accepted (suite asks for more than the REC).
Each stylesheet's own comment says it is streamable in one particular
processor "but not guaranteed-streamable according to W3C rules", and the catalog keywords say
`_WRONG:not-streamable`. go-xslt raises XTSE3430, which is what the REC
requires of a construct declared streamable that is not
guaranteed-streamable.

### `si-fork-901`, `si-fork-951`, `si-fork-954`
**Status:** Accepted (measured trade-off).
§19.8.4.20 makes an `xsl:fork` branch that returns streamed nodes roaming
and free-ranging, so XTSE3430 would be correct. Reporting it fails the
suite's own `si-fork-006` and eight cases sharing its stylesheet, which
expect the same shape to run. Measured: +1 / −9. go-xslt classifies the
construct as unstreamable and runs it buffered, with correct output. See
`internal/xslt/streamability.go`, `strmFork`.

### `si-fork-953`
**Status:** Accepted.
The case's own description asks "should this be an error?". The REC rule
for a sorted group inside `xsl:fork` does not settle it.

### `su-ascent-903`, `sx-square-array-201`
**Status:** Accepted (earlier analysis, not re-verified in this round).
Both expect XTSE3430 inside an `xsl:source-document`. go-xslt reports only
the streamability rejections it can prove; widening `xsl:source-document`
enforcement has been measured net-negative every time.
`sx-square-array-201` additionally depends on global variables being
evaluated lazily rather than eagerly. Both orders are allowed by the
spec, and document order across trees is implementation-defined.

### `sf-reverse-001`
**Status:** Accepted (expected result contradicts XPath).
The expression is `reverse(snapshot(/chapter)//section)/@id`, and the suite
expects the ids in reverse order. But the final `/@id` is a path step, and
XPath 3.1 §3.3.1 returns the result of `/` in document order with
duplicates removed. go-xslt returns `1 1.1 1.2 1.2.1 1.2.2 1.3`, which is
what the rule gives. The case itself points to W3C bug 24125.

### `strip-space-009`
**Status:** Accepted (not derivable from the REC).
Expects whitespace text nodes to be kept under an element whose type has an
XSD 1.1 assertion. XSLT 3.0 §4.4.2 keeps whitespace only for elements whose
type is simple or has simple content; it has no rule for assertions. This is
the behaviour of one particular processor.

### `validation-0201`
**Status:** Accepted.
The case compares the whole XHTML serialization with a reference file (indent="yes").
It found two real serializer bugs, both fixed: indentation was added inside
mixed content and next to inline elements (§5.1.4, §6.1.4), and the xhtml
method wrote `&nbsp;`, which is not well-formed XML without a DTD. The
harness also ignores the whitespace that indent="yes" leaves to the
serializer, which the catalog allows ("differences … capable of being
produced by a conformant implementation"). One difference remains: the reference
file has a line break before `</p>` right after the text "…United States.".
§6.1.4 allows added whitespace only before or after an element, or next to
existing whitespace, and neither applies there. go-xslt keeps the data
model unchanged at that point, so the case cannot pass without copying
the reference file's choice.

### `backwards-018`
**Status:** Accepted.
Needs the same XHTML `<meta>` heuristic as `output-0130` / `output-0715`
above. It is not a backwards-compatibility gap.

### `catalog-005`, `catalog-005b`
**Status:** Accepted.
They need schema components the stylesheet never imports (the schema for
XSLT itself, XTTE1512). They rely on a global schema cache; §3.16 makes only
imported components available.

### `docbook-001`, `docbook-002`
**Status:** Accepted (decision).
The DocBook XSL stylesheets call the EXSLT extension `exsl:node-set()`
without checking that it exists. EXSLT is not a W3C specification, its
definition is informal (what the function does with node-sets, numbers and
booleans is not stated), and XSLT 3.0 does not require it. Not implemented.

## Excluded, not run (Unicode-version test sets)

`regex-classes` and `unicode-90` were written for old Unicode versions. Their
cases run like all others, except these, which the harness skips with a reason
(`test/conformance/unicode_exclusions_test.go`; `XSLT30_U90=1` runs them
anyway):

### Reason `unicode-version` (44 cases)
**Status:** Accepted (implementation-defined).
The expected result is the character data of Unicode 3.1 to 6.0
(`regex-classes`) or 9.0 (`unicode-90`). Which version of Unicode is supported
is implementation-defined (F&O 3.1, Conformance; XSD 1.1 Part 2). go-xslt
uses Unicode 15.0, and its category escapes agree with Unicode 15.0 at every
code point (checked against Go's tables; the separate test
`TestUnicodeCategoryEscapesAgainstSuiteData` checks the 9.0 data, allowing
exactly the code points whose category changed).
- `regex-classes`: the 29 general-category cases (`regex-classes-001`, -003,
  -004, -006 to -009, -011 to -020, -022 to -024, -026 to -033, -036).
- `unicode-90`: the 15 `Ll` cases that run over every code point outside the
  9.0 Ll data (`unicode90-Ll-005` to -008, -014, -015, -017, -018, -020,
  -022, -024, -032, -036 to -038); 170 code points became Ll after 9.0. No
  9.0 Ll character changed category, so the Ll cases over the 9.0 characters
  themselves run and pass.

### Reason `suite-defect` (1,048 cases)
**Status:** Accepted (suite defect).
No implementation can pass these as written:
- The generated `unicode90-<Category>-NNN` cases for every category but `Ll`
  whose mode depends on the category (1,036 cases): the stylesheet tests
  `\p{$charclass}`, but the catalog never passes `charclass`, so it defaults
  to `Ll` and the case for `Zs` asks whether the `Zs` characters are
  lowercase letters. The class-independent modes (`string-length`,
  `translate`, tautologies) run for every category and pass.
- `unicode90-001` to `-008`: the stylesheet counts over all planes, the
  expected counts are BMP-only totals (`\d` + `\D` = 63,454, the figure in the
  stylesheet's own comment).
- `unicode90-Ll-033`, `-035`: compare with `string-join(*/c, '')`, but the
  `<c>` elements are empty (the character is in `@c`).
- `unicode90-C-025`, `unicode90-Cn-025`: the data writes U+FFFE and U+FFFF,
  which XML cannot contain, as `c=""`, so the string length cannot equal the
  element count.

