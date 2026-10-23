# Performance

go-xslt is compared here with `xsltproc` and `xmllint`, the command-line tools
of libxslt and libxml2, the C libraries most XML tooling is built on. Both
run the same XSLT 1.0 stylesheets on the same document, so they do the same
work; each run is timed as a whole process (start, parse, transform,
serialize).

Apple M4, macOS, libxml2 2.9.13 / libxslt 1.1.35; input 200,000 records,
37 MB; median of 5 runs.

| Workload | go-xslt | libxslt / libxml2 | ratio | peak memory go-xslt | peak memory libxml2 |
|---|---|---|---|---|---|
| copy the whole document | 1.51 s | 0.80 s | 1.9× | 1,452 MB | 1,067 MB |
| recursive apply-templates, rebuilding every node | 3.20 s | 1.45 s | 2.2× | 1,785 MB | 1,074 MB |
| sort all records by two keys | 2.17 s | 22.16 s | 0.1× | 1,098 MB | 587 MB |
| Muenchian grouping with `xsl:key` | 3.29 s | 1.01 s | 3.3× | 1,240 MB | 538 MB |
| predicates, string and number functions | 1.48 s | 0.93 s | 1.6× | 961 MB | 590 MB |
| XSD 1.0 validation | 2.64 s | 0.59 s | 4.5× | 1,228 MB | 539 MB |

## Reading the numbers

- On most workloads go-xslt takes 1.5 to 4.5 times as long as the C
  libraries and needs more memory for the document tree. Sorting is the
  exception: libxslt's sort is ten times slower here.
- The comparison is limited to what both can do. go-xslt also implements
  XSLT 3.0, XPath 3.1 and XSD 1.1, which libxslt and libxml2 do not.
- For documents too large for memory, XSLT 3.0
  [streaming](/guide/streaming) processes them in a few megabytes, whatever
  their size.
- A program that runs the same stylesheet many times should
  [compile it once](/guide/compiled-stylesheets); the numbers above include
  compiling the stylesheet on every run.

Both processors produce the same results. In two workloads the text differs
in the last digits of sums: XPath 1.0 writes a number with as many digits as
are needed to identify it exactly (`17251.089999999997`), libxslt rounds to 15
digits (`17251.09`). The benchmark compares numbers to 12 significant digits.

## Running the benchmark

```bash
go run scripts/bench/main.go            # -n runs, -records size of the input
```

It builds the command-line tool from the checkout, generates the input, and
prints this table for your machine. It needs `xsltproc` and `xmllint`, which
macOS ships and Linux distributions package as `xsltproc` and
`libxml2-utils`.
