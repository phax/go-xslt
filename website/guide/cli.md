# Command line

`go-xslt` runs transformations, validation and XPath expressions from a shell,
a pipeline or CI, like `xsltproc` and `xmllint`:

```bash
go install github.com/tim-riep/go-xslt/cmd/go-xslt@latest
```

It is part of the library module, so it needs nothing beyond Go.

## Transform

```bash
go-xslt transform -s style.xsl input.xml > output.xml
go-xslt transform -s style.xsl -p title="Annual report" -o report.html input.xml
cat input.xml | go-xslt transform -s style.xsl -
go-xslt transform -s style.xsl            # no source: starts at xsl:initial-template
go-xslt transform input.xml               # the source names its own stylesheet
```

| Option | |
|---|---|
| `-s file` | The stylesheet. Without it, the source must embed one (`<?xml-stylesheet href="#id"?>`). |
| `-o file` | Write the result to a file instead of stdout. |
| `-p name=value` | A stylesheet parameter (repeatable), as a string. |
| `-it name` | Start with this named template. |
| `-d dir` | Write the `xsl:result-document` outputs into this directory. Only relative paths inside it are written. Without `-d` they are not written, and a note says so. |
| `-schema-aware`, `-backwards-compatible`, `-strict-streaming` | The [optional features](/reference/engine#features). |
| `-max-steps n`, `-max-depth n` | The [resource limits](/guide/security#resource-limits). |

`xsl:message` output goes to stderr.

## Validate

```bash
go-xslt validate -x order.xsd order.xml
go-xslt validate -xsd 1.1 -x main.xsd -x types.xsd -lang de order.xml
go-xslt validate order.xml                # uses the instance's xsi:schemaLocation
```

Prints `order.xml: valid` or `invalid` with every error. `-max-errors n`
stops after n errors, `-q` prints nothing.

## XPath

```bash
go-xslt xpath -s order.xml 'sum(//item/@price)'
go-xslt xpath 'format-date(current-date(), "[D] [MNn] [Y]", "de", (), ())'
go-xslt xpath -t '1 to 3, "x"'            # with each item's type
```

Each item of the result is printed on its own line.

## Files and network

Each command may read the files in the directories of the files named on its
command line: the stylesheet, the source, the schemas. To allow more:

| Option | |
|---|---|
| `-allow-dir dir` | Also read files in this directory (repeatable). |
| `-allow-all-files` | Read any file. |
| `-allow-network` | Retrieve `http` and `https` URIs. |
| `-allow-host name` | Only from these hosts (repeatable). |

See [Security](/guide/security).

## Exit status and messages

| Status | |
|---|---|
| 0 | success; the instance is valid |
| 1 | the run failed, or the instance is invalid |
| 2 | wrong usage |

Errors are printed as `file:line:col: error: [CODE] message`, which editors and
CI systems pick up as locations.
