# XSD validation

`engine.Validate` validates an XML instance document against XSD 1.0 or 1.1
schemas.

```go
res := engine.Validate(engine.ValidateRequest{
	Schemas:  []string{schemaText},
	Instance: instanceText,
	Version:  engine.XSD11, // engine.XSD10 is the default
})
if !res.Valid {
	for _, d := range res.Diagnostics {
		fmt.Printf("%d:%d %s %s\n", d.Line, d.Col, d.Path, d.Message)
	}
}
```

## The request

| Field | Meaning |
|---|---|
| `Schemas` | Schema documents as text. The first is the entry schema; the others are available to `xs:import` and `xs:include` by namespace. |
| `Instance` | The XML document to validate. |
| `Version` | `engine.XSD10` (default) or `engine.XSD11`. |
| `BaseDir` | The directory for relative `schemaLocation`s in `xs:import`, `xs:include`, `xs:redefine`, `xs:override` and `xsi:schemaLocation`. Empty means no file is read. |
| `Language` | The `xml:lang` of the [assertion messages](/guide/assertion-messages) to report, for example `de`. |
| `MaxErrors` | Stop after this many errors. 0 means no limit. |
| `Security` | Which schema files may be read. See [Security](/guide/security). |

## The result

`Valid` is true exactly when the instance is schema-valid. `Diagnostics` holds
schema compile errors and validation errors, and is never empty when `Valid`
is false.

## All errors, not only the first

`Validate` does not stop at the first problem. It reports every error it can
find, in document order, each with `Line`, `Col` and `Path` (for example
`/Invoice[1]/InvoiceLine[2]`):

- content-model errors and simple-type or facet errors of different elements
- every failed assertion, including those inherited by extension
- every missing required attribute

A content-model error of a parent does not hide the errors of its children.
`Valid` is the same as it would be without collecting.

## Precise content-model errors

A child element that is missing, occurs too often, is not allowed, or stands
in the wrong place is reported as exactly that. `Diagnostic.Kind` tells which:

| Kind | Reported at |
|---|---|
| `missing` | the parent |
| `too-many` | each surplus occurrence |
| `unexpected` | the child that is not allowed |
| `order` | the child that is out of place |

`Diagnostic.Particle` is the expanded name of the element declaration
(`{namespace}local`). Messages use the prefix the instance uses, for example
`element cac:Party is missing in Invoice`.

For `xs:all` groups and open content the validator falls back to one generic
message for the parent. The validity verdict is never affected by this.

## What is supported

- simple types with all facets, including the built-in list types
- complex content models: sequence, choice and all; wildcards; substitution
  groups; extension and restriction; Unique Particle Attribution and
  particle restriction checks
- identity constraints: `xs:key`, `xs:keyref`, `xs:unique`
- schemas spread over several documents: `xs:import`, `xs:include`,
  `xs:redefine`, `xs:override`
- XSD 1.1: `xs:assert`, `xs:assertion`, conditional type assignment,
  `xs:openContent`, `explicitTimezone`, negated wildcards
- validation of the schema documents themselves against the schema for
  schemas

A construct the validator does not implement is reported as unsupported,
never answered with a guessed verdict.

## Inspecting a schema without an instance

To walk a schema's elements, types and content models, see
[Reading a schema](/guide/schema-model).
