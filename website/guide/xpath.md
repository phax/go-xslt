# XPath evaluation

`engine.EvalXPath` evaluates an XPath 3.1 expression on its own, without a
stylesheet.

```go
res := engine.EvalXPath(engine.XPathRequest{
	Expression: `sum(//item/@price) => format-number('#,##0.00')`,
	Source:     `<order><item price="9.5"/><item price="12"/></order>`,
})
if res.HasErrors() {
	log.Fatal(res.Diagnostics[0].Message)
}
fmt.Println(res.Value) // 21.50
```

## The request

| Field | Meaning |
|---|---|
| `Expression` | The XPath 3.1 expression. |
| `Source` | An optional XML document. When set, its document node is the context item, so `/order/item` works. When empty, there is no context item; expressions like `1 + 2` or `current-date()` still work. |
| `BaseDir` | The directory for relative URIs in `fn:doc`, `fn:unparsed-text` and `fn:collection`. Empty means these resources are unavailable. |
| `Security` | Which files and network resources the expression may read. See [Security](/guide/security). |

## The result

| Field | Meaning |
|---|---|
| `Value` | The string value of the whole result sequence. |
| `Items` | One entry per item, each with `Type` and `Value`. |
| `Count` | The number of items. |
| `Diagnostics` | Parse and evaluation errors. |
| `DurationMs` | Wall-clock time of the evaluation. |

`Type` is a short label: a node kind such as `element` or `attribute`, an
atomic type such as `xs:integer`, or `map`, `array` or `function`.

```go
res := engine.EvalXPath(engine.XPathRequest{Expression: `(1, 'two', 3.0, map{})`})
for _, it := range res.Items {
	fmt.Println(it.Type, it.Value)
}
// xs:integer 1
// xs:string two
// xs:decimal 3
// map map{}
```

For what the language supports, see [XPath 3.1](/guide/xpath-language).
