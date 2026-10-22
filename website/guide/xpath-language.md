# XPath 3.1

go-xslt implements the whole XPath 3.1 language. The same engine runs inside
stylesheets and in [`engine.EvalXPath`](/guide/xpath).

## Language

- all operators with their precedence: value, general and node comparisons,
  `||`, `union`, `intersect`, `except`, `instance of`, `treat as`,
  `castable as`, `cast as`, the arrow operator `=>`, the simple map operator
  `!` and ranges with `to`
- `for`, `let`, `some`, `every` and `if`
- maps and arrays, with the lookup operator `?`
- inline functions, named function references such as `f#1`, and partial
  application such as `f(?, 2)`
- `Q{uri}local` names and comments `(: … :)`

## Types

A typed atomic value system based on XSD: integers and decimals of arbitrary
precision, `xs:double` and `xs:float`, dates, times and durations, binary
types, `xs:anyURI`, `xs:QName` and `xs:untypedAtomic`, with casting and
atomization.

## Functions

The Functions and Operators 3.1 catalog:

| Area | Examples |
|---|---|
| Strings and regular expressions | `substring-before`, `tokenize`, `replace`, `analyze-string` |
| Numbers | `round-half-to-even`, `format-number`, `format-integer`, `math:*` |
| Dates and times | `format-dateTime`, `adjust-dateTime-to-timezone`, `current-date` |
| Sequences | `distinct-values`, `index-of`, `sort`, `fold-left`, `for-each-pair` |
| Nodes and documents | `doc`, `collection`, `unparsed-text`, `path`, `innermost` |
| Maps and arrays | `map:merge`, `map:for-each`, `array:flatten`, `array:sort` |
| JSON | `parse-json`, `json-doc`, `json-to-xml`, `xml-to-json`, `serialize` |
| Higher-order | `function-lookup`, `apply`, `filter`, `for-each` |
| Other | `fn:transform`, `random-number-generator`, `collation-key` |

`fn:transform` runs a real XSLT transformation, also when called from a
standalone XPath expression.

## Locale support

`format-date`, `format-time` and `format-dateTime` know English and German
month and day names. `format-integer` spells out numbers in English, German,
French and Italian, as cardinal and ordinal words.

Collations follow the Unicode Collation Algorithm through `golang.org/x/text`,
including `caseFirst`, `caseLevel`, `backwards` and `numeric`. The `reorder`
parameter is not supported.

## Regular expressions

Regular expressions follow the XSD/XPath grammar and are checked against it.
Back-references, which Go's RE2 engine cannot handle, are supported through a
separate matcher that is only used for patterns that need it.

## Not implemented

- XQuery, including `fn:load-xquery-module`
- the optional static typing feature of XPath 3.0, apart from a narrow check
  for provably empty axis steps
