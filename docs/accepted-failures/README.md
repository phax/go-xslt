# Accepted conformance failures

Every W3C test case that go-xslt currently fails is listed in one of these
files, with the reason it fails and why that is accepted (or that it is still
open):

| File | Suite |
|---|---|
| [xslt30.md](xslt30.md) | W3C `xslt30-test`, default and combined (`XSLT30_FULL=1`) mode |
| [qt3.md](qt3.md) | W3C `qt3tests` (XPath 3.1 / F&O 3.1) |
| [xsd.md](xsd.md) | W3C `xsdtests` (XSD 1.0 and 1.1) |

Each entry names its cases in backticks. The conformance harnesses check
this: when a suite is run, any failing case whose name does not appear in
its file makes the test fail with "undocumented failures". A new failure
therefore has to be fixed, or written down here with its reason. When a
documented case starts passing, remove its entry by hand.

## Status values

- **Accepted**: analysed; the failure is a consequence of the
  specification, of a contradiction inside the suite, or of a deliberate
  decision. Not to be re-attempted without new evidence.
- **Open**: not analysed far enough to call it accepted. These are
  candidates for further work.

The numbers are measured offline. The few cases that need live web pages are
marked; with `XSLT30_NETWORK=1` / `QT3_NETWORK=1` they can be run online.
