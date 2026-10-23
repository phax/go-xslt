# Security policy

## Supported versions

go-xslt is before 1.0. Security fixes go into the next release; only the
latest release is supported.

| Version | Supported |
|---|---|
| latest release | yes |
| older releases | no |

## Reporting a vulnerability

Please report security problems **privately**, through GitHub's
[private vulnerability reporting](https://github.com/tim-riep/go-xslt/security/advisories/new),
not in a public issue.

Please include the smallest input that shows the problem (stylesheet, source
document, schema or XPath expression), the `Security` and `Limits` settings
you used, and the version (`go-xslt version`, or the module version you
import).

go-xslt is maintained by one person. You can expect an acknowledgement within
a week, and a fix as soon as the problem is understood. Once the fix is
released, the report is published as a security advisory and, if you wish,
names you.

## What counts as a vulnerability

The library is meant to process input it does not trust, within the limits
the calling program sets (see the
[security guide](https://tim-riep.github.io/go-xslt/guide/security)). A
vulnerability is any way for a stylesheet, document, schema or expression to
get past those limits, for example to:

- read a file outside `BaseDir` and `AllowedDirs`
- reach the network when `AllowNetwork` is off, or a host outside
  `AllowedHosts`
- write a file at all
- crash the process (a panic that is not turned into a diagnostic, a stack
  overflow)
- run past `Limits.MaxSteps`, `MaxDepth` or `MaxOutputBytes`, or past the
  cancellation of its `context.Context`

The desktop app reads the files of the project folder it was given; a way
for a project's files to make it read or run anything else counts too.

## What does not

- Behaviour the settings allow: with `AllowAllFiles` or `AllowNetwork`, a
  stylesheet may read what they permit.
- Memory use. The library does not limit the memory of the trees a run
  builds; that is documented, and a hard limit needs a separate process.
- Slow but finite runs within the limits you set.
- Results that differ from the specifications. Those are bugs; please report
  them as ordinary [issues](https://github.com/tim-riep/go-xslt/issues).
