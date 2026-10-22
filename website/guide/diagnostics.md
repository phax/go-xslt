# Diagnostics

No entry point of go-xslt panics. Every problem, including an internal panic,
comes back as an `engine.Diagnostic` in the result.

```go
type Diagnostic struct {
	Severity string // "error" or "warning"
	Line     int    // 1-based; 0 means unknown
	Col      int    // 1-based; 0 means unknown
	Code     string // W3C error code where known, e.g. "XPST0003"
	Message  string
	Phase    string // "parse", "compile" or "run"

	// Only set by XSD validation:
	Rule     string // msg:message/@rule of a failed assertion
	Path     string // e.g. /Doc[1]/Item[2]
	Particle string // {namespace}local of a content-model error
	Kind     string // "missing", "too-many", "unexpected" or "order"
}
```

`engine.SeverityError` and `engine.SeverityWarning` hold the two severity
values. `Result.HasErrors()` and `XPathResult.HasErrors()` tell whether any
diagnostic is an error.

## Where Line and Col point

| Phase | Position |
|---|---|
| `parse` | In the document that could not be parsed. |
| `compile` | In the stylesheet. |
| `run` | At the innermost stylesheet instruction the error passed through. |
| validation | In the instance document. Schema compile errors may have no position. |

## Error codes

`Code` is the error code defined by the W3C specification, for example:

| Prefix | Defined by | Example |
|---|---|---|
| `XTSE` | XSLT, static error | `XTSE0010` unknown XSLT element |
| `XTDE`, `XTTE` | XSLT, dynamic and type errors | `XTTE0570` value does not match `as` |
| `XPST`, `XPTY`, `XPDY` | XPath | `XPST0008` undeclared variable |
| `FO…` | Functions & Operators | `FORG0001` invalid cast |

An `xsl:message terminate="yes"` and `fn:error` produce the code they were
given.
