# Conformance

go-xslt is measured against the official W3C test suites. The numbers count
**applicable** cases: a case is run when its declared dependencies match what
the engine is (for example, XQuery-only cases are not run against an XPath
processor). Skipped cases are reported separately and never counted.

Measured on 2026-10-11, release `v0.0.6`.

| Suite | Source | Pass rate |
|---|---|---|
| XSLT 3.0, basic processor | [w3c/xslt30-test](https://github.com/w3c/xslt30-test) | **99.9%** (8,614 of 8,626) |
| XSLT 3.0 with streaming, schema-awareness and backwards compatibility | same | **99.8%** (12,267 of 12,296) |
| XPath 3.1 and F&O 3.1 (QT3) | [w3c/qt3tests](https://github.com/w3c/qt3tests) | **99.8%** (22,099 of 22,153) |
| XSD 1.0 | [w3c/xsdtests](https://github.com/w3c/xsdtests) | **99.99%** (39,346 of 39,351) |
| XSD 1.1 | same | **99.99%** (41,499 of 41,503) |

Of the basic XSLT cases, 100% of those written for XSLT 1.0 pass, 99.8% of
those for 2.0 and 99.8% of those for 3.0.

## Every failure is documented

Each case that fails is listed with its reason in
[`docs/accepted-failures/`](https://github.com/tim-riep/go-xslt/tree/main/docs/accepted-failures),
one file per suite. The test harness enforces this: a failing case that is not
listed there makes the test fail.

The remaining failures fall into a few groups:

- cases that need live web pages (they pass when network access is enabled)
- contradictions inside the suites, confirmed against an independent
  processor
- features outside the scope of the engine: XQuery modules, XPath's optional
  static typing, UCA `reorder`, XInclude
- a few edge cases whose fix would cost far more than it is worth, each with
  its reason

## Excluded cases

Two test sets were written for old Unicode versions (`unicode-90` for 9.0,
`regex-classes` for 3.1 to 6.0). Their cases run like any other; only these
are excluded, each with a documented reason:

- **Unicode version (44 cases).** The expected result is the character data
  of the old version. Which Unicode version a processor supports is
  implementation-defined; go-xslt uses Unicode 15.0 and agrees with it for
  every code point.
- **Defects in the suite (1,048 cases).** No implementation can pass them as
  written: most generated `unicode-90` cases never pass the category they are
  meant to test, so they all test lowercase letters.

The full list and the reasoning are in
[`docs/accepted-failures/xslt30.md`](https://github.com/tim-riep/go-xslt/blob/main/docs/accepted-failures/xslt30.md).

## Running the suites yourself

The quickest way is one command from a checkout of the repository. It clones
the three suites at the commits the numbers above were measured against, runs
the four measurements and prints a table of your results next to the
published ones:

```bash
make conformance        # or: scripts/conformance.sh
```

Logs and the list of failing cases are written to `conformance-results/`.
`--only xslt,xslt-full,qt3,xsd` runs a subset and `--latest` uses the current
`HEAD` of each suite instead (results may then differ by a case or two).

To do it by hand: the suites are not part of the repository. Clone them into
`test/conformance/`; without them, the conformance tests skip.

```bash
git clone --depth 1 https://github.com/w3c/xslt30-test.git test/conformance/xslt30-test
git clone --depth 1 https://github.com/w3c/qt3tests.git test/conformance/qt3tests
git clone --depth 1 https://github.com/w3c/xsdtests.git test/conformance/xsdtests
```

```bash
go test ./test/conformance/...
```

The environment variables for progress output, failure lists and the combined
XSLT mode are described in
[`test/conformance/README.md`](https://github.com/tim-riep/go-xslt/blob/main/test/conformance/README.md).
The suites change over time, so the current `HEAD` of each may give results
that differ by a case or two.
