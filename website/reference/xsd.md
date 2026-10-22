# Package xsd

```go
import "github.com/tim-riep/go-xslt/xsd"
```

A read-only view of a compiled XSD 1.0 or 1.1 schema. For a guided
introduction see [Reading a schema](/guide/schema-model). To validate
documents use [`engine.Validate`](/guide/validation).

Full generated documentation:
[pkg.go.dev/github.com/tim-riep/go-xslt/xsd](https://pkg.go.dev/github.com/tim-riep/go-xslt/xsd)

## Compile

```go
func Compile(req Request) (schema *Schema, diags []Diagnostic)

type Request struct {
	Schemas  []string // the first is the entry schema
	BaseDir  string   // resolves xs:import, xs:include, xs:redefine, xs:override
	Version  Version  // V10 (default) or V11
	Security Security // same type as engine.Security
}
```

On failure the schema is `nil`. A construct the compiler does not support is
reported as such.

## Schema

| Method | Returns |
|---|---|
| `GlobalElements() []*Element` | All global element declarations |
| `GlobalTypes() []TypeDefinition` | All global type definitions |
| `GlobalAttributes() []*AttributeDecl` | All global attribute declarations |
| `Element(xml.Name) *Element` | One global element |
| `Type(xml.Name) TypeDefinition` | One type, including built-in types |
| `SubstitutionMembers(head xml.Name) []*Element` | The members of a substitution group, transitively |
| `Unresolved() []string` | References to types the schema does not define |

## Element

`Name()`, `Type()`, `Abstract()`, `Nillable()`, `Default()`, `Fixed()`,
`SubstitutionGroups()`, `Documentation()`.

## ComplexType

| Method | Returns |
|---|---|
| `Name()` | The type name; empty for an anonymous type |
| `Base()`, `Derivation()` | The base type and `extension` or `restriction` |
| `BuiltinBase()` | The built-in base type for simple content |
| `ContentKind()` | `ContentEmpty`, `ContentSimple`, `ContentElementOnly` or `ContentMixed` |
| `ContentModel() *Particle` | The effective content model |
| `ContentParticles() []*Particle` | The particles of the top-level group |
| `SimpleContent() *SimpleType` | The type of simple content |
| `Attributes() []*AttributeUse` | The effective attributes |
| `AttributeWildcard() *Wildcard` | The `xs:anyAttribute`, if any |
| `Abstract()`, `Mixed()`, `IsSimple()`, `Documentation()` | |

## SimpleType

| Method | Returns |
|---|---|
| `Name()`, `Base()`, `BuiltinBase()` | Name, direct base, built-in base through restrictions |
| `Variety()` | `VarietyAtomic`, `VarietyList` or `VarietyUnion` |
| `ItemType()`, `MemberTypes()` | For list and union types |
| `Facets()`, `Enumeration()` | The effective facets |
| `IsSimple()`, `Documentation()` | |

## Particle

```go
type Particle struct {
	MinOccurs     int
	MaxOccurs     int  // Unbounded (-1) for maxOccurs="unbounded"
	Term          Term // *Element, *ModelGroup or *Wildcard
	Documentation []Documentation
}
```

## AttributeUse

```go
type AttributeUse struct {
	Name        xml.Name
	Type        *SimpleType
	Use         Use // UseOptional, UseRequired or UseProhibited
	Default     string
	HasDefault  bool
	Fixed       string
	HasFixed    bool
	Declaration *AttributeDecl
	Documentation []Documentation
}
```

## Constants

| Group | Constants |
|---|---|
| Version | `V10`, `V11` |
| Variety | `VarietyAtomic`, `VarietyList`, `VarietyUnion` |
| ContentKind | `ContentEmpty`, `ContentSimple`, `ContentElementOnly`, `ContentMixed` |
| Compositor | `CompositorSequence`, `CompositorChoice`, `CompositorAll` |
| Use | `UseOptional`, `UseRequired`, `UseProhibited` |
| Occurrence | `Unbounded` (-1) |
