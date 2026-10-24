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
| copy the whole document | 1.33 s | 0.78 s | 1.7× | 1,248 MB | 1,067 MB |
| recursive apply-templates, rebuilding every node | 2.51 s | 1.38 s | 1.8× | 1,402 MB | 1,074 MB |
| sort all records by two keys | 1.45 s | 21.07 s | 0.07× | 913 MB | 587 MB |
| Muenchian grouping with `xsl:key` | 1.48 s | 0.93 s | 1.6× | 921 MB | 538 MB |
| predicates, string and number functions | 1.24 s | 0.89 s | 1.4× | 860 MB | 590 MB |
| XSD 1.0 validation | 1.23 s | 0.55 s | 2.3× | 700 MB | 539 MB |

## Reading the numbers

- On most workloads go-xslt takes 1.4 to 2.3 times as long as the C
  libraries and needs more memory for the document tree. Sorting is the
  exception: libxslt's sort is more than ten times slower here.
- The comparison is limited to what both can do. go-xslt also implements
  XSLT 3.0, XPath 3.1 and XSD 1.1, which libxslt and libxml2 do not.
- For documents too large for memory, XSLT 3.0
  [streaming](/guide/streaming) processes them in a few megabytes, whatever
  their size.
- A program that runs the same stylesheet many times should
  [compile it once](/guide/compiled-stylesheets); the numbers above include
  compiling the stylesheet on every run.

## Where the time goes

A CPU profile of the copy workload shows that parsing is not the bottleneck:
the parser takes about 12% of the CPU time, `encoding/xml` within it 4%. Most
of the time goes to obtaining memory for the document tree — 35% in the
operating system's memory calls while the heap grows past 1 GB — and to the
garbage collector scanning it, which runs on other cores and hardly adds wall
time (switching it off saves under 10%). The lever is therefore the memory
a run allocates, not the XML tokenizer: allocating less (one reused
evaluation context per predicate and key walk instead of one per node, nodes
carved from blocks, content models matched without per-element maps) is what
brought grouping from 3.0 to 1.6 times libxslt's time, validation from 4.5
to 2.3 times libxml2's, and apply-templates from 2.2 to 1.8 times. One
reused context per XPath evaluation in templates, sort keys atomized once,
and date and number checks in XSD validation without building values
followed.

## Same results

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
