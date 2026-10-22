# XSD (`xsdtests`): failing cases

XSD 1.0: 39,346 of 39,351 pass, 5 fail. XSD 1.1: 41,499 of 41,503 pass,
4 fail. Nothing is reported as unsupported. The full investigation history
is in `tools/xsd_results.txt` (sections M28 and M30).

## XSD 1.0

### `particlesZ001.i`
**Status:** Accepted (suite contradiction).
The schema is expected to be invalid, but the valid twin test in the same
`schemaTest` needs the opposite reading of the RecurseAsIfGroup rule.
go-xslt follows the reading the suite needs elsewhere (wrapper group keeps
the occurrence range only when the base group repeats without bound, see
`internal/xsd/restriction.go`). Changing it is a net loss of one case.

### `particlesZ023`, `particlesZ024`, `groupH021v`
**Status:** Accepted (suite contradiction).
A choice restricted by dropping branches. The same mapping with element
branches is expected *valid* by the version-independent tests
`particlesT003`, `particlesT005` and `particlesT006`. No XSD 1.0 clause makes
these three invalid and those three valid.

### `idB005`
**Status:** Accepted.
The only defect a processor can detect is a type reference with the wrong
letter case (`keyinfo` instead of `keyInfo`). The broader rule that would
reject it (schema-wide uniqueness of `@id`) is refuted by about 20 valid
cross-document tests.

## XSD 1.1

### `all308.xsd`
**Status:** Accepted (disputed in the suite).
Marked disputed in the suite (W3C bug 6202).

### `particlesZ026`
**Status:** Accepted.
The test-suite task force declared the outcome implementation-determined
(W3C bug 4071).

### `particlesZ033_g`
**Status:** Accepted.
Expected invalid, but no XSD 1.1 rule it violates can be found: XSD 1.1
allows the competition between an element and a wildcard that would make it
invalid in 1.0.

### `ste110.i`
**Status:** Accepted (deliberate price).
XSD 1.1 forbids every circular union (`cos-no-circular-unions`; the suite's
`simple017`, `simple020` and others). `ste110.i` expects one such cycle to
be accepted, and its valid twin is excluded as queried. See
`internal/xsd/loader.go`.
