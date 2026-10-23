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
| `Namespaces` | Prefix bindings for the expression, prefix → namespace URI. |
| `DefaultElementNamespace` | The namespace of element names written without a prefix. Empty means no namespace. |
| `Variables` | `$name` → a string. |
| `VariableExprs` | `$name` → the value of an XPath expression, so it can be a number, a date, a map or nodes of `Source`. |

## Namespaces

A document in a namespace, such as a UBL invoice, needs its namespaces in the
expression. The prefixes the document element of `Source` declares are bound
automatically, so its own prefixes just work:

```go
ubl := `<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
                 xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
          <cbc:ID>R-1</cbc:ID></Invoice>`

engine.EvalXPath(engine.XPathRequest{Expression: "string(/*/cbc:ID)", Source: ubl}) // R-1
```

The document's **default** namespace has no prefix, and an unprefixed name in
XPath means "no namespace". There are three ways to reach such elements:

```go
// 1. Bind a prefix of your own:
engine.XPathRequest{Expression: "/inv:Invoice/cbc:ID", Source: ubl,
	Namespaces: map[string]string{"inv": "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"}}

// 2. Make it the default element namespace:
engine.XPathRequest{Expression: "/Invoice/cbc:ID", Source: ubl,
	DefaultElementNamespace: "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"}

// 3. Write the namespace into the name:
engine.XPathRequest{Expression: "/Q{urn:oasis:names:specification:ubl:schema:xsd:Invoice-2}Invoice", Source: ubl}
```

The prefixes `xml`, `xs`, `xsi`, `fn`, `math`, `map`, `array` and `err` are
always bound. An entry in `Namespaces` wins over the document's own binding of
the same prefix. A prefix bound nowhere is the error `XPST0081`.

## Variables

```go
res := engine.EvalXPath(engine.XPathRequest{
	Expression:    "count($items[@price > $min])",
	Source:        order,
	Variables:     map[string]string{"currency": "EUR"},
	VariableExprs: map[string]string{"min": "10", "items": "//item"},
})
```

`Variables` gives strings. `VariableExprs` evaluates each expression against
`Source`, so it can be typed (`10` is an `xs:integer`, `xs:date('2026-01-31')`
a date) or select nodes. Variable expressions cannot refer to each other.

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
