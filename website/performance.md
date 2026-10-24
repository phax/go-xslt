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
| copy the whole document | 0.60 s | 0.74 s | 0.81× | 1,108 MB | 1,067 MB |
| recursive apply-templates, rebuilding every node | 1.13 s | 1.33 s | 0.85× | 1,180 MB | 1,074 MB |
| sort all records by two keys, text output | 0.77 s | 20.89 s * | — | 822 MB | 587 MB |
| Muenchian grouping with `xsl:key` | 0.71 s | 0.91 s | 0.78× | 845 MB | 538 MB |
| predicates, string and number functions | 0.62 s | 0.87 s | 0.72× | 778 MB | 590 MB |
| XSD 1.0 validation | 0.60 s | 0.53 s | 1.13× | 622 MB | 539 MB |

\* Not a sorting measurement. Almost all of libxslt's time here goes into
building the 5.6 MB text output: the libxslt 1.1.35 that macOS ships appends
each piece to one text node and measures that node's whole length again
every time, which grows quadratically. The same stylesheet without
`xsl:sort` takes it just as long; the sorting itself costs libxslt about
1.5 s. Current libxslt releases avoid this, so the row is left out of the
comparison.

The machine was not idle during this run, which slows both tools alike; the
ratios are what carries over.

## Reading the numbers

- go-xslt is faster than the C libraries at copying, template rules,
  grouping and XPath, and about 1.1 times slower at validation. It needs
  somewhat more memory for the document tree.
- The comparison is limited to what both can do. go-xslt also implements
  XSLT 3.0, XPath 3.1 and XSD 1.1, which libxslt and libxml2 do not.
- For documents too large for memory, XSLT 3.0
  [streaming](/guide/streaming) processes them in a few megabytes, whatever
  their size.
- A program that runs the same stylesheet many times should
  [compile it once](/guide/compiled-stylesheets); the numbers above include
  compiling the stylesheet on every run.

## Where the time goes

Four rounds of profiling and measurement brought go-xslt from 1.5–4.5 times
libxslt's time to where it is now:

- reusing evaluation contexts (per predicate, key walk, template match and
  XPath evaluation) instead of allocating one per node, carving parsed
  nodes from blocks, and matching XSD content models without per-element
  maps;
- an own XML tokenizer in place of `encoding/xml` that hands out pieces of
  the document instead of copies — it accepts and rejects exactly what
  `encoding/xml` did, which differential fuzzing and the W3C XML
  Conformance Test Suite check — about 260 MB less per 37 MB document;
- skipping the DTD passes for documents without a DOCTYPE, fewer
  allocations per template call, and serializing without building strings;
- matching simple patterns such as `node()` and `@*` without a full
  evaluation context, keeping `defer` out of the hottest functions, and a
  faster sort where the result is provably the same.

An earlier version of this page put a third of the time into the operating
system's memory calls. That came from the CPU profiler on macOS, which books
kernel time to the wrong system call; measured directly, those calls take a
few milliseconds. The garbage collector is the largest real share now.
What remains is mostly the document and result trees themselves.

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
