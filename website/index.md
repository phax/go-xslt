---
layout: home

hero:
  name: go-xslt
  text: XSLT 3.0, XPath 3.1 and XSD in pure Go
  tagline: Built from scratch, without libxslt, CGo or a JVM. Import it as a Go package, or use the desktop tester.
  image:
    src: /logo.svg
    alt: go-xslt logo
  actions:
    - theme: brand
      text: Get started
      link: /guide/getting-started
    - theme: alt
      text: Why go-xslt?
      link: /guide/what-is-go-xslt
    - theme: alt
      text: GitHub
      link: https://github.com/tim-riep/go-xslt

features:
  - title: XSLT 3.0
    details: Templates, modes, grouping, iterate, try/catch, merge, accumulators, packages and multiple result documents. 99.9% of the W3C xslt30-test suite passes.
    link: /guide/xslt
  - title: XPath 3.1
    details: The full language with typed atomic values, maps, arrays, higher-order functions and the Functions & Operators 3.1 catalog. 99.8% of the W3C QT3 suite passes.
    link: /guide/xpath-language
  - title: XSD 1.0 and 1.1
    details: A validator with assertions, conditional types, identity constraints and override. It reports every error with line, column and path. 99.99% of the W3C xsdtests pass.
    link: /guide/validation
  - title: Streaming
    details: Guaranteed streaming per XSLT 3.0 runs in bounded memory, on the input and the output side.
    link: /guide/streaming
  - title: Safe by default
    details: Untrusted stylesheets and documents can only read files under the base directory. Network access is off unless you turn it on.
    link: /guide/security
  - title: One dependency
    details: The library needs only golang.org/x/text. Every entry point is panic-safe and reports errors as diagnostics with W3C codes.
    link: /reference/engine
---
