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

go-xslt never writes files. The documents of `xsl:result-document` come back in
`Result.SecondaryOutputs`.

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

## Trusted input

If all input is your own, you can lift the file restriction:

```go
engine.Security{AllowAllFiles: true}
```

Before this option existed, absolute paths and paths outside `BaseDir` were
read without restriction. A program that relies on this behaviour sets
`AllowedDirs`, or `AllowAllFiles: true` for input it trusts.
