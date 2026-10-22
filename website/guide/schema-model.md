# Reading a schema

Package `github.com/tim-riep/go-xslt/xsd` gives read-only access to a compiled
XSD 1.0 or 1.1 schema. It is meant for tools that check something against a
schema *without* an instance document, for example whether a path in a
template exists, has the right type and the right cardinality.

To validate documents, use [`engine.Validate`](/guide/validation) instead.

## Compiling

```go
import "github.com/tim-riep/go-xslt/xsd"

schema, diags := xsd.Compile(xsd.Request{
	Schemas: []string{mainXSD},
	BaseDir: dir,      // xs:import, xs:include, xs:redefine, xs:override
	Version: xsd.V10,  // or xsd.V11
})
if schema == nil {
	return fmt.Errorf("schema: %s", diags[0].Message)
}
```

A schema can compile with warnings, for example for references to types it
never defines. `schema.Unresolved()` lists them.

A `*xsd.Schema` is immutable and safe for concurrent use.

## Walking elements and types

```go
for _, el := range schema.GlobalElements() {
	ct, ok := el.Type().(*xsd.ComplexType)
	if !ok {
		continue
	}
	for _, p := range ct.ContentParticles() {
		// p.MinOccurs, p.MaxOccurs (xsd.Unbounded is -1)
		switch t := p.Term.(type) {
		case *xsd.Element:    // a child element declaration
		case *xsd.ModelGroup: // a nested sequence, choice or all
		case *xsd.Wildcard:   // xs:any
		_ = t
		}
	}
	for _, a := range ct.Attributes() {
		_ = a // a.Name, a.Type, a.Use
	}
}
```

Content models and attributes are the **effective** ones, after extension and
restriction have been applied.

## Looking up a type

```go
td := schema.Type(xml.Name{Space: ns, Local: "Money"})
if td.IsSimple() {
	fmt.Println(td.BuiltinBase()) // {http://www.w3.org/2001/XMLSchema}decimal
}
```

`BuiltinBase` follows restriction chains down to the built-in type.

## Substitution groups

`schema.SubstitutionMembers(head)` lists the members of a substitution group,
transitively.

## Types in the package

| Type | Is |
|---|---|
| `Schema` | A compiled schema |
| `Element` | An element declaration; also a `Term` |
| `ComplexType`, `SimpleType` | Type definitions; both are a `TypeDefinition` |
| `Particle` | A `Term` with `MinOccurs` and `MaxOccurs` |
| `ModelGroup` | A sequence, choice or all group |
| `Wildcard` | `xs:any` or `xs:anyAttribute` |
| `AttributeUse`, `AttributeDecl` | An attribute of a complex type; a global attribute declaration |
| `Facets` | The effective constraining facets of a simple type |
| `Documentation` | One `xs:documentation` with `Lang`, `Source` and `Text` |

The full list of methods is on
[pkg.go.dev](https://pkg.go.dev/github.com/tim-riep/go-xslt/xsd).
