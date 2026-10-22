# QT3 (`qt3tests`, XPath 3.1 / F&O 3.1): failing cases

22,099 of 22,153 applicable cases pass, 54 fail. Measured offline.

### XQuery modules
`fn-load-xquery-module-014`, `fn-load-xquery-module-015`,
`fn-load-xquery-module-018`, `fn-load-xquery-module-019`,
`fn-load-xquery-module-020`, `fn-load-xquery-module-021`,
`fn-load-xquery-module-022`, `fn-load-xquery-module-025`,
`fn-load-xquery-module-026`, `fn-load-xquery-module-027`,
`fn-load-xquery-module-028`, `fn-load-xquery-module-029`,
`fn-load-xquery-module-030`, `fn-load-xquery-module-033`,
`fn-load-xquery-module-034`, `fn-load-xquery-module-035`,
`fn-load-xquery-module-036`, `fn-load-xquery-module-037`,
`fn-load-xquery-module-040`, `fn-load-xquery-module-041`,
`fn-load-xquery-module-042`, `fn-load-xquery-module-061`,
`fn-load-xquery-module-064`, `fn-load-xquery-module-065`,
`fn-load-xquery-module-068`, `fn-load-xquery-module-069`,
`fn-load-xquery-module-070`, `fn-load-xquery-module-073`,
`fn-load-xquery-module-074`, `fn-function-lookup-760`

**Status:** Accepted (decision).
They load XQuery library modules (`fn:load-xquery-module`). go-xslt is an
XPath and XSLT processor and has no XQuery grammar (prolog, `declare
function`, `import module`). It would mean building an XQuery front end.

### XPath static typing
`fn-filter-007`, `fn-filter-016`, `fn-filter-019`, `fn-filter-022`,
`fn-for-each-pair-032`, `fn-function-arity-010`, `fn-function-arity-012`,
`fn-function-lookup-713`, `fn-function-name-010`, `fn-has-children-012`,
`fn-has-children-013`, `fn-innermost-008`, `fn-innermost-010`,
`fn-outermost-008`, `fn-outermost-010`, `fn-unparsed-text-009`,
`fn-unparsed-text-available-007`, `fn-unparsed-text-available-009`,
`fn-unparsed-text-available-011`, `fn-unparsed-text-lines-009`

**Status:** Accepted (optional feature not implemented).
Each declares the dependency `feature staticTyping` and expects XPTY0004 from
*static* type inference. A typical example is
`if (current-dateTime() eq …) then "utf-8" else 1` as an argument, whose
branch types do not all fit the parameter. At run time only one branch is
taken, so a processor without the optional Static Typing Feature correctly
raises nothing. go-xslt implements only the narrow, unconditional part
(provably empty axis steps, behind `xpath.ParseStaticTyping`).

### `fn-transform-901`, `fn-transform-902`, `fn-transform-err-14`
**Status:** Accepted (contradicts a real implementation).
Each expects `fn:transform` to be unavailable. go-xslt implements
`fn:transform`, so these cannot pass without removing a working feature.

### `fn-unparsed-text-054a`
**Status:** Accepted.
Reads https://timeanddate.com fifty times. Offline it cannot pass; online
(`QT3_NETWORK=1`) the server now answers every client with HTTP 403 (bot
protection), so it fails there too.
