# xpath package — function-implementation API contract

Read this fully before writing any function. All code goes in package `xpath`
under `internal/xpath/`. Your job: implement a family of XPath/F&O 3.1 functions
in a new file `fn_<family>.go` and tests in `fn_<family>_test.go`.

## Value model

A value is an `Object` (= `any`). It is one of:
- `NodeSet` (`[]*xmltree.Node`) — a node sequence
- `Sequence` (`[]Item`) — a general sequence; `Item` = `any`
- a single `*Atomic` — a typed atomic value
- `*Map`, `*Array`, `*Function` — XPath 3.1 maps/arrays/function items
- bare `string`/`float64`/`bool` may appear at boundaries (treat like atomics)

Use these helpers (already defined — DO NOT redefine):
- `Items(o Object) []Item` — flatten any value to items
- `FromItems(items []Item) Object` — build a value (all-nodes→NodeSet, 1→item, else Sequence)
- `ToString(o) string`, `ToNumber(o) float64`, `ToBool(o) bool`
- `ToNodeSet(o) (NodeSet, bool)`
- `Atomize(o Object) ([]Item, error)` — fn:data; items are `*Atomic`
- `CastTo(it Item, t AtomType) (*Atomic, error)`, `Castable(it, t) bool`
- `itemString(it Item) string`, `itemNumber(it Item) float64`

## Atomic values (`atomic.go`)

`type AtomType int` constants: `XSstring XSboolean XSdecimal XSinteger XSdouble
XSfloat XSdate XSdateTime XStime XSduration XSyearMonthDuration XSdayTimeDuration
XShexBinary XSbase64Binary XSanyURI XSqname XSuntypedAtomic` (+ derived integer
types and gregorian types). `AtomTypeByName("xs:integer")` → (AtomType, bool).

Constructors (return `*Atomic`):
- `NewString(s)`, `NewUntyped(s)`, `NewAnyURI(s)`, `NewBool(b)`
- `NewInteger(int64)`, `NewIntegerBig(*big.Int)`, `NewDouble(float64)`, `NewFloat(float64)`
- `NewDecimal(*big.Rat)`, `NewDecimalFromString(s) (*Atomic, bool)`
- `NewQName(xmltree.Name)`, `NewBinary(t AtomType, []byte)`
- `NewDateTime(t AtomType, time.Time, hasTZ bool)`, `NewDuration(t AtomType, Duration)`

`type Duration struct { Months int; Secs float64 }`

`*Atomic` accessors: `.T` (AtomType), `.Lexical() string` (canonical form),
`.Float() float64`, `.Bool() bool`, `.IsNumeric() bool`.
To read date/time and duration fields from an `*Atomic`, cast it first via
`CastTo`/`Atomize`; the time/duration are stored internally — for date/time
functions, parse from `a.Lexical()` using the existing `parseDateTimeValue(t,
s)` and `parseDurationValue(t, s)` helpers in `datetime.go`, or accept an
already-typed `*Atomic` and read via reflection-free accessors you add **with a
family-prefixed name**. Simplest: re-parse `a.Lexical()`.

Internal date/time helpers in `datetime.go` you MAY use:
`parseDateTimeValue(t AtomType, s string) (time.Time, bool, error)`,
`parseDurationValue(t AtomType, s string) (Duration, error)`,
`formatDateTime`, `formatDuration`.

## Function signature & registration

```go
type coreFunc func(ctx *Context, args []Object) (Object, error)
```
Register in an `init()` in your file by ADDING to the shared maps (already
allocated as package-var literals):
```go
func init() {
    coreFuncs["my-func"] = fnMyFunc      // fn: namespace
    mapFuncs["my-map-func"] = ...        // map: namespace (only map family)
    arrayFuncs["my-array-func"] = ...    // array: namespace (only array family)
    mathFuncs["tan"] = ...               // math: namespace (only numeric family)
}
```
Argument helpers (defined — do not redefine): `arg(args, i) Object` (nil if
missing), `argN(args, i) Object` (empty string if missing).

`Context` fields you may read: `.Node *xmltree.Node`, `.Pos int`, `.Size int`,
`.CtxItem Item`, `.NS NamespaceResolver`. For functions with an optional
argument that defaults to the context item, use the context node/item.

Return errors with the spec error code in the message, e.g.
`fmt.Errorf("err:FORG0001: invalid value")`.

## Signature and argument checking (REQUIRED for every function)

Every built-in needs its F&O signature, one line per arity, in
`builtinCatalogSrc` (`fn_catalog.go`), e.g.

```
fn:substring($sourceString as xs:string?, $start as xs:double) as xs:string
fn:substring($sourceString as xs:string?, $start as xs:double, $length as xs:double) as xs:string
```

`dispatchFunc` looks the function up there and applies the function conversion
rules (XPath 3.1 §3.1.5.2) to the arguments BEFORE calling you (`fn_argcheck.go`),
for a direct call, a named reference, `fn:function-lookup`, a partial
application and a dynamic call alike. So your function can rely on its
declared types: an atomic parameter holds atomic values of the declared type
(or a subtype; a numeric may be any numeric type where `xs:double` is declared
— promotion is a validity check, read it with `ToNumber`/`Float`; an
`xs:string` parameter may be an `xs:anyURI` or `xs:untypedAtomic`, or an
untyped node, read it with `ToString`/`Lexical`), a parameter with no
occurrence indicator holds exactly one item, `?` at most one. A mismatch is
XPTY0004 (an `xs:untypedAtomic` for `xs:QName`: XPTY0117), an `xs:untypedAtomic`
that cannot be cast to the declared type is the cast's error (FORG0001). Do not
repeat these checks in the implementation. Exceptions: a function-typed
parameter is checked as `function(*)` (check its arity and convert what you pass
and get back yourself), and XSLT 1.0 compatibility mode (`Context.BC10`) skips
the check entirely, so never crash on an argument of the wrong shape.
`TestBuiltinCatalogComplete` fails for a registered function without a line.

The XSLT-defined functions (`current`, `key`, `document`, …) are declared the
same way in `xsltSigSrc` (`internal/xslt/functions.go`).

## Tests

Use the shared helper (defined in `helpers_test.go` — do not redefine):
```go
func TestFnX(t *testing.T) {
    cases := []struct{ expr, want string }{
        {"my:func('a')", "A"},
    }
    for _, c := range cases {
        if got := xpStr(t, c.expr); got != c.want {
            t.Errorf("%s = %q, want %q", c.expr, got, c.want)
        }
    }
}
```
`xpStr(t, expr)` parses+evaluates `expr` against a `<r/>` document and returns
`ToString` of the result. For functions needing nodes, build the document in
the expression is not possible — instead test node functions against a parsed
tree using `xpStrCtx(t, expr, xmlString)` (also in helpers_test.go).

## Rules to avoid breaking the shared package

1. ONE new file `fn_<family>.go` + ONE `fn_<family>_test.go`. Do not edit other files, except to add your signature lines to `fn_catalog.go`.
2. Prefix all your private helper funcs/vars with your family initials
   (e.g. `dt`, `uri`, `qn`, `sq`) to avoid name clashes with other agents.
3. Do NOT redefine anything listed above or any already-registered function.
4. Do NOT run `go build`/`go test` on the package (sibling files are being
   written concurrently and will not compile until integration). Just write
   correct code per this contract.
5. Use only Go stdlib: strings, math, math/big, time, regexp, unicode,
   unicode/utf8, net/url, encoding/base64, encoding/hex, encoding/json, sort, fmt.

## Already-registered functions — DO NOT redefine

fn: last position count local-name name namespace-uri id root string concat
starts-with ends-with contains substring substring-before substring-after
string-length normalize-space translate upper-case lower-case matches replace
string-join exists empty abs tokenize distinct-values reverse subsequence
index-of head tail min max avg boolean not true false lang number sum floor
ceiling round for-each filter fold-left fold-right for-each-pair sort

math: pi sqrt pow sin cos exp log log10
map: get contains size keys put remove entry merge for-each
array: size get append flatten join for-each
