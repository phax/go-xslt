# Contributing

## Issues: welcome from v0.3.0 on

From version 0.3.0 on, issues are open. Bug reports are the most useful thing
you can send before 1.0. Please open an issue for:

- a wrong result, a missing or wrong error, or a crash
- a stylesheet, expression or schema another processor handles and go-xslt
  does not
- anything in the documentation that is unclear or wrong

A good report has the smallest input that shows the problem (stylesheet,
source document, schema or expression), what you expected, what go-xslt did,
and the version (`go-xslt version`, or the module version you import).

If a report concerns a security problem — a way for untrusted input to read
files, reach the network, or crash or hang the process despite the
[Security](https://tim-riep.github.io/go-xslt/guide/security) settings — please
use GitHub's private vulnerability reporting instead of a public issue; see
[SECURITY.md](SECURITY.md).

## Pull requests: not accepted

Pull requests are not accepted and are closed unreviewed. Until 1.0.0 the
public API (the `engine` and `xsd` packages), the desktop app's project format
and much of the internal architecture still change without notice, and
reviewing and integrating code against a moving target isn't practical. This
will be revisited once 1.0.0 ships.
