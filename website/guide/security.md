# Security

Stylesheets, XPath expressions, schemas and source documents can refer to
other resources: `xsl:import`, `document()`, `fn:unparsed-text`, `xs:import`,
the external subset of a `DOCTYPE`. If this input is not trusted, unrestricted
access would let it read any file the process can read and copy it into the
output, or send requests to internal services.

go-xslt therefore restricts all of it by default. Only your program can widen
the limits, never the input.

## Defaults

| | Default | To widen |
|---|---|---|
| **Local files** | Only files under `BaseDir` and its subdirectories. Without `BaseDir`, no file is read at all. | `AllowedDirs` adds directories; `AllowAllFiles: true` lifts the limit. |
| **Network** (`http`, `https`) | Off. Such a URI is treated as an unavailable resource. | `AllowNetwork: true`; `AllowedHosts` limits it to certain hosts. |

The settings are in the `Security` field of `engine.Request`,
`engine.TransformRequest`, `engine.XPathRequest`, `engine.ValidateRequest` and
`xsd.Request`.

```go
res := engine.Transform(engine.Request{
	Stylesheet: src,
	BaseDir:    "/srv/jobs/42",
	Security: engine.Security{
		AllowedDirs:  []string{"/srv/shared/schemas"},
		AllowNetwork: true,
		AllowedHosts: []string{"data.example.com", ".cdn.example.com"},
	},
})
```

A host name with a leading dot also admits all its subdomains.

## Local files

The rule covers every read:

- stylesheet modules: `xsl:import`, `xsl:include`
- `xsl:import-schema` and the schema documents of `xs:include`, `xs:import`,
  `xs:redefine`, `xs:override` and `xsi:schemaLocation`
- `document()`, `fn:doc`, `fn:unparsed-text` and its variants, collections and
  `xsl:source-document`
- static evaluation in `use-when`
- the stylesheet a nested `fn:transform` loads
- external DTD subsets and entity files that any of these documents pull in

Symbolic links are resolved before the check, so a link inside `BaseDir` that
points outside it is refused.

To the stylesheet, a refused file looks like a missing one:
`doc-available()` returns false, and `fn:unparsed-text` raises `FOUT1170`.

## Network

- Redirects are checked against `AllowedHosts` too.
- Each response is limited to 64 MB and 30 seconds.
- URLs with a user name or password are refused.
- Within one run, each URI is fetched once and the answer is reused, as the
  F&O specification requires for `fn:doc` and `fn:unparsed-text`.
- A document fetched over the network is parsed without reading external DTDs
  or entity files, so a remote `DOCTYPE` cannot pull in local files.

::: warning Not covered by AllowNetwork
The locations of `xsl:import`, `xsl:include`, `xsl:import-schema` and
`xs:import` always stay local, as does static evaluation in `use-when`.
:::

## Writing files

go-xslt never writes files. The principal result is returned (or written to
the `io.Writer` you pass to `TransformTo`), and every document of
`xsl:result-document` comes back in `Result.SecondaryOutputs`, with its `Href`
as written in the stylesheet. Whether and where to store them is your
program's decision.

## Dynamic code

`xsl:evaluate` and `fn:transform` compile and run code that is only known at
run time. They run under the same rules as the stylesheet that calls them:
the same `Security` settings (a nested `fn:transform` cannot read more than
its caller), and the same [limits](#resource-limits), whose budget the nested
run spends from. `fn:transform` reads its stylesheet through the same file
check.

## Resource limits

A stylesheet can loop or recurse without end. `Limits` (a field of
`engine.Request` and `engine.TransformRequest`) bounds a run:

| Field | Default | Bounds |
|---|---|---|
| `MaxSteps` | 20,000,000 instructions | endless loops, e.g. an unbounded `xsl:iterate` |
| `MaxDepth` | 10,000 levels | endless recursion of templates and functions; it can only be lowered |
| `MaxOutputBytes` | no limit | the size of the results: the principal result and all `xsl:result-document` outputs together |

```go
res := engine.Transform(engine.Request{
	Stylesheet: src,
	Source:     xml,
	Limits:     engine.Limits{MaxSteps: 1_000_000, MaxDepth: 1000},
})
```

A run that exceeds a limit stops with `XPDY0130` ("implementation limit
exceeded"). Nested `fn:transform` runs spend from the caller's budget, so they
cannot multiply it.

### Time: cancel with a context

`TransformContext`, `TransformToContext`, `Stylesheet.TransformContext`,
`EvalXPathContext` and `ValidateContext` stop when their `context.Context` is
done, the usual Go way to give a request a deadline:

```go
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()
res := engine.TransformContext(ctx, req)
```

The context is checked every 1024 instructions, inside long XPath
evaluations and for every element validated, so a run stops shortly after the
deadline. The run then ends with an error diagnostic naming `ctx.Err()`.

Some limits always apply, because the input would otherwise crash the Go
process (a stack overflow is a fatal error that no `recover` catches):

| Input | Limit | Error |
|---|---|---|
| element nesting in a document | 10,000 levels | parse error |
| nesting of an XPath expression | 5,000 levels | `XPDY0130` |
| inline functions calling each other | 10,000 levels | `XPDY0130` |

Not limited by go-xslt: the memory a run uses for its trees. `MaxOutputBytes`
bounds the results, and the input is as large as you pass in; for a hard
memory limit, run the transformation in a separate process (for example the
[command-line tool](/guide/cli)) under the operating system's limits.

## Checklist for untrusted input

- Give each run its own `BaseDir` holding only what it may read, and no
  `AllowedDirs` beyond that.
- Leave `AllowNetwork` off, or restrict it with `AllowedHosts`.
- Set `Limits` to what your workload needs, and give every run a context
  with a deadline.
- Treat `SecondaryOutputs[].Href` as untrusted too before using it as a file
  name.
- For a hard memory limit, run in a separate process.

## Trusted input

If all input is your own, you can lift the file restriction:

```go
engine.Security{AllowAllFiles: true}
```

Before this option existed, absolute paths and paths outside `BaseDir` were
read without restriction. A program that relies on this behaviour sets
`AllowedDirs`, or `AllowAllFiles: true` for input it trusts.
