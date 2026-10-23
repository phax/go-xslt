# Getting started

## Requirements

Go 1.25 or newer.

## Install

```bash
go get github.com/tim-riep/go-xslt
```

For the [command-line tool](/guide/cli):

```bash
go install github.com/tim-riep/go-xslt/cmd/go-xslt@latest
```

## Your first transformation

```go
package main

import (
	"fmt"

	"github.com/tim-riep/go-xslt/engine"
)

const stylesheet = `<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:for-each select="//book">
      <xsl:value-of select="title || ' (' || @year || ')'"/>
      <xsl:text>&#10;</xsl:text>
    </xsl:for-each>
  </xsl:template>
</xsl:stylesheet>`

const source = `<library>
  <book year="1951"><title>Foundation</title></book>
  <book year="1965"><title>Dune</title></book>
</library>`

func main() {
	res := engine.Transform(engine.Request{
		Stylesheet: stylesheet,
		Source:     source,
	})
	if res.HasErrors() {
		for _, d := range res.Diagnostics {
			fmt.Printf("%s %d:%d %s %s\n", d.Severity, d.Line, d.Col, d.Code, d.Message)
		}
		return
	}
	fmt.Print(res.Output)
}
```

Output:

```text
Foundation (1951)
Dune (1965)
```

`engine.Transform` never panics. A syntax error in the stylesheet, a malformed
source document or a dynamic error at run time all come back in
`res.Diagnostics`, with the W3C error code and the position. See
[Diagnostics](/guide/diagnostics).

## Next steps

- [Transformations](/guide/transform): parameters, initial templates, files
  and secondary outputs
- [Compiled stylesheets](/guide/compiled-stylesheets): compile once, run many
  times, concurrently
- [XPath evaluation](/guide/xpath): evaluate an expression without a stylesheet
- [XSD validation](/guide/validation): validate an instance document
- [Security](/guide/security): what a stylesheet may read
