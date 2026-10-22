# Assertion messages

XSD has no place for a human-readable message on `xs:assert` or on a
content-model error. go-xslt reads one from the component's `xs:appinfo`, in
the namespace `https://github.com/tim-riep/go-xslt/xsd/messages`. Other
processors ignore it, and it never changes whether a document is valid.

## On assertions

```xml
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:msg="https://github.com/tim-riep/go-xslt/xsd/messages">
  …
  <xs:assert test="exists(Reason) or exists(ReasonCode)">
    <xs:annotation><xs:appinfo>
      <msg:message rule="BR-33" xml:lang="en">Reason or reason code is missing.</msg:message>
      <msg:message rule="BR-33" xml:lang="de">Grund oder Grundcode fehlt.</msg:message>
    </xs:appinfo></xs:annotation>
  </xs:assert>
```

When the assertion fails, the diagnostic has:

- `Message`: the entry whose `xml:lang` matches `ValidateRequest.Language`,
  else the entry without `xml:lang`, else the first entry
- `Rule`: the `rule` attribute, here `BR-33`
- `Line`, `Col` and `Path` of the element the assertion failed on

This works for assertions on complex types, assertions inherited by extension,
and `xs:assertion` facets on simple types.

## On cardinality errors

The same `msg:message` can sit in the `xs:appinfo` of an **element
declaration**, an **`xs:choice`** or an **`xs:sequence`**. It then words that
particle's content-model errors.

```xml
<xs:element name="DocumentNumber" type="xs:string">
  <xs:annotation><xs:appinfo>
    <msg:message rule="BR-2" on="missing" xml:lang="en">The document number is missing.</msg:message>
  </xs:appinfo></xs:annotation>
</xs:element>
```

The `on` attribute narrows a message to one kind of error: `missing` or
`too-many`. Without `on`, the message applies to both.

A choice's message speaks for all its alternatives. If a choice may occur only
once and a second alternative appears, that is reported as `too-many` of the
choice, with the choice's message, rather than as an element out of order.

## On missing attributes

A required attribute that is missing takes an `on="missing"` message from the
`xs:attribute`'s appinfo: first from the attribute use, else from the
attribute declaration. Every missing attribute is reported, not only the
first.
