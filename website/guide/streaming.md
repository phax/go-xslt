# Streaming and large documents

go-xslt implements XSLT 3.0 **guaranteed streaming**. A streamable
transformation reads its input one record at a time and keeps only the path
from the root to the current node in memory, however large the document is.

## Streaming the input

Mark the construct as streamable in the stylesheet:

```xml
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode streamable="yes"/>

  <xsl:template name="xsl:initial-template">
    <xsl:source-document streamable="yes" href="orders.xml">
      <totals>
        <xsl:for-each select="orders/order">
          <order id="{@id}" total="{sum(item/@price)}"/>
        </xsl:for-each>
      </totals>
    </xsl:source-document>
  </xsl:template>
</xsl:stylesheet>
```

Streaming applies to `xsl:source-document streamable="yes"`, streamable
modes, streamable functions and attribute sets, and the streaming form of
`xsl:fork`.

Before running, the compiler checks each streamable construct against the
streamability rules of XSLT 3.0 (section 19 and appendix J). A construct it
can prove streamable is executed one record at a time. One it cannot prove
runs with the whole document in memory, unless you ask for the strict
behaviour of the specification:

```go
res := engine.Transform(engine.Request{
	Stylesheet: src,
	Features:   engine.Features{StrictStreaming: true},
})
```

With `StrictStreaming`, such a construct is a compile error, `XTSE3430`, so a
stylesheet that is meant to run in bounded memory cannot silently fall back.
The analysis is cautious: some constructs are rejected that a more elaborate
analysis could accept; none is accepted that would not stream.

## Streaming the output

`engine.TransformTo` writes the result to an `io.Writer`. When the stylesheet
and its `xsl:output` allow it, the result tree is serialized and released
while it is being built:

```go
out, err := os.Create("totals.xml")
if err != nil {
	return err
}
defer out.Close()

res := engine.TransformTo(out, engine.Request{
	Stylesheet:      stylesheetText,
	InitialTemplate: "xsl:initial-template",
	BaseDir:         dataDir,
})
```

When that is not safe, the result is built in full first and then written.
The bytes are identical either way.

## Measured memory

| | Document size | Peak heap, streamed | Peak heap, buffered |
|---|---|---|---|
| Input side | 2 to 82 MB | about 2.0 MB, flat | up to about 630 MB |
| Output side | 1.2 to 49 MB | about 2.5 MB, flat | up to about 355 MB |

## Without streaming

For ordinary, non-streamable stylesheets, `TransformFromReader` avoids one copy
of the source document, and `TransformTo` avoids holding the serialized result
as a string. The source tree itself is still built in memory.
