# Package engine

```go
import "github.com/tim-riep/go-xslt/engine"
```

The public entry point for transformations, XPath evaluation and XSD
validation. No function in this package panics: errors come back as
[diagnostics](/guide/diagnostics).

Full generated documentation:
[pkg.go.dev/github.com/tim-riep/go-xslt/engine](https://pkg.go.dev/github.com/tim-riep/go-xslt/engine)

## Functions

| Function | Does |
|---|---|
| `Transform(Request) Result` | Compiles and runs a stylesheet. |
| `TransformTo(io.Writer, Request) Result` | Like `Transform`, but writes the principal result to the writer. `Result.Output` stays empty. |
| `TransformFromReader(io.Reader, Request) Result` | Like `Transform`, but reads the source document from the reader. `Request.Source` is ignored. |
| `Compile(Request) (*Stylesheet, []Diagnostic)` | Compiles a stylesheet for repeated use. |
| `EvalXPath(XPathRequest) XPathResult` | Evaluates a standalone XPath 3.1 expression. |
| `Validate(ValidateRequest) ValidateResult` | Validates an XML document against XSD schemas. |

## Stylesheet

A compiled stylesheet. Immutable and safe for concurrent use.

| Method | Does |
|---|---|
| `(*Stylesheet) Transform(TransformRequest) Result` | Runs the stylesheet. |
| `(*Stylesheet) TransformTo(io.Writer, TransformRequest) Result` | Runs it and writes the principal result to the writer. |
| `(*Stylesheet) TransformFromReader(io.Reader, TransformRequest) Result` | Runs it with the source read from the reader. |

## Request

```go
type Request struct {
	Stylesheet      string            // XSLT source text
	Source          string            // source document; may be empty with InitialTemplate
	Params          map[string]string // global stylesheet parameters
	InitialTemplate string            // named template to start with
	BaseDir         string            // resolves relative URIs; empty = no file reads
	Security        Security
}
```

## TransformRequest

One run of a compiled `Stylesheet`. The stylesheet and its `BaseDir` were
fixed by `Compile`.

```go
type TransformRequest struct {
	Source          string
	Params          map[string]string
	InitialTemplate string
	Security        Security
}
```

## Result

```go
type Result struct {
	Output           string            // principal result, serialized
	Diagnostics      []Diagnostic
	DurationMs       int64
	Method           string            // "xml", "html", "text" or "json"
	Messages         []string          // xsl:message output
	SecondaryOutputs []SecondaryOutput // xsl:result-document output
}

func (r Result) HasErrors() bool

type SecondaryOutput struct {
	Href    string
	Content string
	Method  string
}
```

## XPathRequest and XPathResult

```go
type XPathRequest struct {
	Expression string
	Source     string // optional context document
	BaseDir    string
	Security   Security
}

type XPathResult struct {
	Value       string      // string value of the whole sequence
	Items       []XPathItem // one per item
	Count       int
	Diagnostics []Diagnostic
	DurationMs  int64
}

func (r XPathResult) HasErrors() bool

type XPathItem struct {
	Type  string // "element", "xs:integer", "map", …
	Value string
}
```

## ValidateRequest and ValidateResult

```go
type ValidateRequest struct {
	Schemas   []string   // the first is the entry schema
	Instance  string
	Version   XSDVersion // XSD10 (default) or XSD11
	BaseDir   string
	Language  string     // xml:lang of assertion messages
	MaxErrors int        // 0 = no limit
	Security  Security   // file access only; no network
}

type ValidateResult struct {
	Valid       bool
	Diagnostics []Diagnostic
	DurationMs  int64
}
```

## Security

```go
type Security struct {
	AllowedDirs   []string // more readable directories besides BaseDir
	AllowAllFiles bool     // lift the file restriction (trusted input only)
	AllowNetwork  bool     // allow http and https
	AllowedHosts  []string // limit network access; ".example.com" admits subdomains
}
```

The zero value allows files under `BaseDir` and no network. See
[Security](/guide/security).

## Diagnostic

```go
type Diagnostic struct {
	Severity string // SeverityError or SeverityWarning
	Line     int
	Col      int
	Code     string // W3C error code
	Message  string
	Phase    string // "parse", "compile" or "run"
	Rule     string // XSD assertion rule
	Path     string // XSD element path
	Particle string // XSD content-model particle
	Kind     string // "missing", "too-many", "unexpected" or "order"
}
```

## Constants

| Constant | Value |
|---|---|
| `SeverityError` | `"error"` |
| `SeverityWarning` | `"warning"` |
| `XSD10` | `"1.0"` |
| `XSD11` | `"1.1"` |
