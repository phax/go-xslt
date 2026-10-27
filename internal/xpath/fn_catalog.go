package xpath

// builtinCatalogSrc is the declared signature of every built-in function of
// the fn:, math:, map: and array: namespaces, one line per arity, written as
// the F&O 3.1 function catalog writes them (the lines were generated from the
// catalog's own machine-readable source, specifications/xpath-functions-31/
// src/function-catalog.xml of w3c/qtspecs, and the parameter NAMES are the
// spec's). It is the single source of three things:
//
//   - the arity range of each function (XPST0017 outside it),
//   - the function conversion rules (XPath 3.1 §3.1.5.2) applied to every
//     argument before the implementation sees it — see fn_argcheck.go,
//   - the declared types of a function item built from a built-in.
//
// A function registered in coreFuncs/mathFuncs/mapFuncs/arrayFuncs without a
// line here fails TestBuiltinCatalogComplete, so no built-in is ever
// unchecked by accident. Rules for the notation:
//
//   - Parameters are `$name as SequenceType`, comma separated; the type may
//     itself contain commas inside parentheses (function(item(), item())).
//   - A trailing `...` after the last parameter marks it variadic: that
//     parameter's type applies to every further argument (fn:concat, the only
//     such function, is specified as 2-or-more xs:anyAtomicType? arguments).
//   - The return type is documentation (and the declared type of a function
//     item); `none` is fn:error's "never returns".
//   - A parameter of a function type (function(item()) as xs:boolean, …) is
//     checked as function(*): the implementation validates the arity and
//     converts arguments and results at call time, so the full function
//     coercion of §3.1.5.3 (wrapping the supplied function) is not applied.
//
// To add a function: implement it (FNAPI.md), then add its line(s) here.
const builtinCatalogSrc = `
fn:concat($arg1 as xs:anyAtomicType?, $arg2 as xs:anyAtomicType?, ...) as xs:string
fn:QName($paramURI as xs:string?, $paramQName as xs:string) as xs:QName
fn:abs($arg as xs:numeric?) as xs:numeric?
fn:adjust-date-to-timezone($arg as xs:date?) as xs:date?
fn:adjust-date-to-timezone($arg as xs:date?, $timezone as xs:dayTimeDuration?) as xs:date?
fn:adjust-dateTime-to-timezone($arg as xs:dateTime?) as xs:dateTime?
fn:adjust-dateTime-to-timezone($arg as xs:dateTime?, $timezone as xs:dayTimeDuration?) as xs:dateTime?
fn:adjust-time-to-timezone($arg as xs:time?) as xs:time?
fn:adjust-time-to-timezone($arg as xs:time?, $timezone as xs:dayTimeDuration?) as xs:time?
fn:analyze-string($input as xs:string?, $pattern as xs:string) as element(fn:analyze-string-result)
fn:analyze-string($input as xs:string?, $pattern as xs:string, $flags as xs:string) as element(fn:analyze-string-result)
fn:apply($function as function(*), $array as array(*)) as item()*
fn:available-environment-variables() as xs:string*
fn:avg($arg as xs:anyAtomicType*) as xs:anyAtomicType?
fn:base-uri() as xs:anyURI?
fn:base-uri($arg as node()?) as xs:anyURI?
fn:boolean($arg as item()*) as xs:boolean
fn:ceiling($arg as xs:numeric?) as xs:numeric?
fn:codepoint-equal($comparand1 as xs:string?, $comparand2 as xs:string?) as xs:boolean?
fn:codepoints-to-string($arg as xs:integer*) as xs:string
fn:collation-key($key as xs:string) as xs:base64Binary
fn:collation-key($key as xs:string, $collation as xs:string) as xs:base64Binary
fn:collection() as item()*
fn:collection($arg as xs:string?) as item()*
fn:compare($comparand1 as xs:string?, $comparand2 as xs:string?) as xs:integer?
fn:compare($comparand1 as xs:string?, $comparand2 as xs:string?, $collation as xs:string) as xs:integer?
fn:contains($arg1 as xs:string?, $arg2 as xs:string?) as xs:boolean
fn:contains($arg1 as xs:string?, $arg2 as xs:string?, $collation as xs:string) as xs:boolean
fn:contains-token($input as xs:string*, $token as xs:string) as xs:boolean
fn:contains-token($input as xs:string*, $token as xs:string, $collation as xs:string) as xs:boolean
fn:count($arg as item()*) as xs:integer
fn:current-date() as xs:date
fn:current-dateTime() as xs:dateTimeStamp
fn:current-time() as xs:time
fn:data() as xs:anyAtomicType*
fn:data($arg as item()*) as xs:anyAtomicType*
fn:dateTime($arg1 as xs:date?, $arg2 as xs:time?) as xs:dateTime?
fn:day-from-date($arg as xs:date?) as xs:integer?
fn:day-from-dateTime($arg as xs:dateTime?) as xs:integer?
fn:days-from-duration($arg as xs:duration?) as xs:integer?
fn:deep-equal($parameter1 as item()*, $parameter2 as item()*) as xs:boolean
fn:deep-equal($parameter1 as item()*, $parameter2 as item()*, $collation as xs:string) as xs:boolean
fn:default-collation() as xs:string
fn:default-language() as xs:language
fn:distinct-values($arg as xs:anyAtomicType*) as xs:anyAtomicType*
fn:distinct-values($arg as xs:anyAtomicType*, $collation as xs:string) as xs:anyAtomicType*
fn:doc($uri as xs:string?) as document-node()?
fn:doc-available($uri as xs:string?) as xs:boolean
fn:document-uri() as xs:anyURI?
fn:document-uri($arg as node()?) as xs:anyURI?
fn:element-with-id($arg as xs:string*) as element()*
fn:element-with-id($arg as xs:string*, $node as node()) as element()*
fn:empty($arg as item()*) as xs:boolean
fn:encode-for-uri($uri-part as xs:string?) as xs:string
fn:ends-with($arg1 as xs:string?, $arg2 as xs:string?) as xs:boolean
fn:ends-with($arg1 as xs:string?, $arg2 as xs:string?, $collation as xs:string) as xs:boolean
fn:environment-variable($name as xs:string) as xs:string?
fn:error() as none
fn:error($code as xs:QName?) as none
fn:error($code as xs:QName?, $description as xs:string) as none
fn:error($code as xs:QName?, $description as xs:string, $error-object as item()*) as none
fn:escape-html-uri($uri as xs:string?) as xs:string
fn:exactly-one($arg as item()*) as item()
fn:exists($arg as item()*) as xs:boolean
fn:false() as xs:boolean
fn:filter($seq as item()*, $f as function(item()) as xs:boolean) as item()*
fn:floor($arg as xs:numeric?) as xs:numeric?
fn:fold-left($seq as item()*, $zero as item()*, $f as function(item()*, item()) as item()*) as item()*
fn:fold-right($seq as item()*, $zero as item()*, $f as function(item(), item()*) as item()*) as item()*
fn:for-each($seq as item()*, $action as function(item()) as item()*) as item()*
fn:for-each-pair($seq1 as item()*, $seq2 as item()*, $action as function(item(), item()) as item()*) as item()*
fn:format-date($value as xs:date?, $picture as xs:string) as xs:string?
fn:format-date($value as xs:date?, $picture as xs:string, $language as xs:string?, $calendar as xs:string?, $place as xs:string?) as xs:string?
fn:format-dateTime($value as xs:dateTime?, $picture as xs:string) as xs:string?
fn:format-dateTime($value as xs:dateTime?, $picture as xs:string, $language as xs:string?, $calendar as xs:string?, $place as xs:string?) as xs:string?
fn:format-integer($value as xs:integer?, $picture as xs:string) as xs:string
fn:format-integer($value as xs:integer?, $picture as xs:string, $lang as xs:string?) as xs:string
fn:format-number($value as xs:numeric?, $picture as xs:string) as xs:string
fn:format-number($value as xs:numeric?, $picture as xs:string, $decimal-format-name as xs:string?) as xs:string
fn:format-time($value as xs:time?, $picture as xs:string) as xs:string?
fn:format-time($value as xs:time?, $picture as xs:string, $language as xs:string?, $calendar as xs:string?, $place as xs:string?) as xs:string?
fn:function-arity($func as function(*)) as xs:integer
fn:function-lookup($name as xs:QName, $arity as xs:integer) as function(*)?
fn:function-name($func as function(*)) as xs:QName?
fn:generate-id() as xs:string
fn:generate-id($arg as node()?) as xs:string
fn:has-children() as xs:boolean
fn:has-children($node as node()?) as xs:boolean
fn:head($arg as item()*) as item()?
fn:hours-from-dateTime($arg as xs:dateTime?) as xs:integer?
fn:hours-from-duration($arg as xs:duration?) as xs:integer?
fn:hours-from-time($arg as xs:time?) as xs:integer?
fn:id($arg as xs:string*) as element()*
fn:id($arg as xs:string*, $node as node()) as element()*
fn:idref($arg as xs:string*) as node()*
fn:idref($arg as xs:string*, $node as node()) as node()*
fn:implicit-timezone() as xs:dayTimeDuration
fn:in-scope-prefixes($element as element()) as xs:string*
fn:index-of($seq as xs:anyAtomicType*, $search as xs:anyAtomicType) as xs:integer*
fn:index-of($seq as xs:anyAtomicType*, $search as xs:anyAtomicType, $collation as xs:string) as xs:integer*
fn:innermost($nodes as node()*) as node()*
fn:insert-before($target as item()*, $position as xs:integer, $inserts as item()*) as item()*
fn:iri-to-uri($iri as xs:string?) as xs:string
fn:json-doc($href as xs:string?) as item()?
fn:json-doc($href as xs:string?, $options as map(*)) as item()?
fn:json-to-xml($json-text as xs:string?) as document-node()?
fn:json-to-xml($json-text as xs:string?, $options as map(*)) as document-node()?
fn:lang($testlang as xs:string?) as xs:boolean
fn:lang($testlang as xs:string?, $node as node()) as xs:boolean
fn:last() as xs:integer
fn:load-xquery-module($module-uri as xs:string) as map(*)
fn:load-xquery-module($module-uri as xs:string, $options as map(*)) as map(*)
fn:local-name() as xs:string
fn:local-name($arg as node()?) as xs:string
fn:local-name-from-QName($arg as xs:QName?) as xs:NCName?
fn:lower-case($arg as xs:string?) as xs:string
fn:matches($input as xs:string?, $pattern as xs:string) as xs:boolean
fn:matches($input as xs:string?, $pattern as xs:string, $flags as xs:string) as xs:boolean
fn:max($arg as xs:anyAtomicType*) as xs:anyAtomicType?
fn:max($arg as xs:anyAtomicType*, $collation as xs:string) as xs:anyAtomicType?
fn:min($arg as xs:anyAtomicType*) as xs:anyAtomicType?
fn:min($arg as xs:anyAtomicType*, $collation as xs:string) as xs:anyAtomicType?
fn:minutes-from-dateTime($arg as xs:dateTime?) as xs:integer?
fn:minutes-from-duration($arg as xs:duration?) as xs:integer?
fn:minutes-from-time($arg as xs:time?) as xs:integer?
fn:month-from-date($arg as xs:date?) as xs:integer?
fn:month-from-dateTime($arg as xs:dateTime?) as xs:integer?
fn:months-from-duration($arg as xs:duration?) as xs:integer?
fn:name() as xs:string
fn:name($arg as node()?) as xs:string
fn:namespace-uri() as xs:anyURI
fn:namespace-uri($arg as node()?) as xs:anyURI
fn:namespace-uri-for-prefix($prefix as xs:string?, $element as element()) as xs:anyURI?
fn:namespace-uri-from-QName($arg as xs:QName?) as xs:anyURI?
fn:nilled() as xs:boolean?
fn:nilled($arg as node()?) as xs:boolean?
fn:node-name() as xs:QName?
fn:node-name($arg as node()?) as xs:QName?
fn:normalize-space() as xs:string
fn:normalize-space($arg as xs:string?) as xs:string
fn:normalize-unicode($arg as xs:string?) as xs:string
fn:normalize-unicode($arg as xs:string?, $normalizationForm as xs:string) as xs:string
fn:not($arg as item()*) as xs:boolean
fn:number() as xs:double
fn:number($arg as xs:anyAtomicType?) as xs:double
fn:one-or-more($arg as item()*) as item()+
fn:outermost($nodes as node()*) as node()*
fn:parse-ietf-date($value as xs:string?) as xs:dateTime?
fn:parse-json($json-text as xs:string?) as item()?
fn:parse-json($json-text as xs:string?, $options as map(*)) as item()?
fn:parse-xml($arg as xs:string?) as document-node(element(*))?
fn:parse-xml-fragment($arg as xs:string?) as document-node()?
fn:path() as xs:string?
fn:path($arg as node()?) as xs:string?
fn:position() as xs:integer
fn:prefix-from-QName($arg as xs:QName?) as xs:NCName?
fn:random-number-generator() as map(xs:string, item())
fn:random-number-generator($seed as xs:anyAtomicType?) as map(xs:string, item())
fn:remove($target as item()*, $position as xs:integer) as item()*
fn:replace($input as xs:string?, $pattern as xs:string, $replacement as xs:string) as xs:string
fn:replace($input as xs:string?, $pattern as xs:string, $replacement as xs:string, $flags as xs:string) as xs:string
fn:resolve-QName($qname as xs:string?, $element as element()) as xs:QName?
fn:resolve-uri($relative as xs:string?) as xs:anyURI?
fn:resolve-uri($relative as xs:string?, $base as xs:string) as xs:anyURI?
fn:reverse($arg as item()*) as item()*
fn:root() as node()
fn:root($arg as node()?) as node()?
fn:round($arg as xs:numeric?) as xs:numeric?
fn:round($arg as xs:numeric?, $precision as xs:integer) as xs:numeric?
fn:round-half-to-even($arg as xs:numeric?) as xs:numeric?
fn:round-half-to-even($arg as xs:numeric?, $precision as xs:integer) as xs:numeric?
fn:seconds-from-dateTime($arg as xs:dateTime?) as xs:decimal?
fn:seconds-from-duration($arg as xs:duration?) as xs:decimal?
fn:seconds-from-time($arg as xs:time?) as xs:decimal?
fn:serialize($arg as item()*) as xs:string
fn:serialize($arg as item()*, $params as item()?) as xs:string
fn:sort($input as item()*) as item()*
fn:sort($input as item()*, $collation as xs:string?) as item()*
fn:sort($input as item()*, $collation as xs:string?, $key as function(item()) as xs:anyAtomicType*) as item()*
fn:starts-with($arg1 as xs:string?, $arg2 as xs:string?) as xs:boolean
fn:starts-with($arg1 as xs:string?, $arg2 as xs:string?, $collation as xs:string) as xs:boolean
fn:static-base-uri() as xs:anyURI?
fn:string() as xs:string
fn:string($arg as item()?) as xs:string
fn:string-join($arg1 as xs:anyAtomicType*) as xs:string
fn:string-join($arg1 as xs:anyAtomicType*, $arg2 as xs:string) as xs:string
fn:string-length() as xs:integer
fn:string-length($arg as xs:string?) as xs:integer
fn:string-to-codepoints($arg as xs:string?) as xs:integer*
fn:subsequence($sourceSeq as item()*, $startingLoc as xs:double) as item()*
fn:subsequence($sourceSeq as item()*, $startingLoc as xs:double, $length as xs:double) as item()*
fn:substring($sourceString as xs:string?, $start as xs:double) as xs:string
fn:substring($sourceString as xs:string?, $start as xs:double, $length as xs:double) as xs:string
fn:substring-after($arg1 as xs:string?, $arg2 as xs:string?) as xs:string
fn:substring-after($arg1 as xs:string?, $arg2 as xs:string?, $collation as xs:string) as xs:string
fn:substring-before($arg1 as xs:string?, $arg2 as xs:string?) as xs:string
fn:substring-before($arg1 as xs:string?, $arg2 as xs:string?, $collation as xs:string) as xs:string
fn:sum($arg as xs:anyAtomicType*) as xs:anyAtomicType
fn:sum($arg as xs:anyAtomicType*, $zero as xs:anyAtomicType?) as xs:anyAtomicType?
fn:tail($arg as item()*) as item()*
fn:timezone-from-date($arg as xs:date?) as xs:dayTimeDuration?
fn:timezone-from-dateTime($arg as xs:dateTime?) as xs:dayTimeDuration?
fn:timezone-from-time($arg as xs:time?) as xs:dayTimeDuration?
fn:tokenize($input as xs:string?) as xs:string*
fn:tokenize($input as xs:string?, $pattern as xs:string) as xs:string*
fn:tokenize($input as xs:string?, $pattern as xs:string, $flags as xs:string) as xs:string*
fn:trace($value as item()*) as item()*
fn:trace($value as item()*, $label as xs:string) as item()*
fn:transform($options as map(*)) as map(*)
fn:translate($arg as xs:string?, $mapString as xs:string, $transString as xs:string) as xs:string
fn:true() as xs:boolean
fn:unordered($sourceSeq as item()*) as item()*
fn:unparsed-text($href as xs:string?) as xs:string?
fn:unparsed-text($href as xs:string?, $encoding as xs:string) as xs:string?
fn:unparsed-text-available($href as xs:string?) as xs:boolean
fn:unparsed-text-available($href as xs:string?, $encoding as xs:string) as xs:boolean
fn:unparsed-text-lines($href as xs:string?) as xs:string*
fn:unparsed-text-lines($href as xs:string?, $encoding as xs:string) as xs:string*
fn:upper-case($arg as xs:string?) as xs:string
fn:uri-collection() as xs:anyURI*
fn:uri-collection($arg as xs:string?) as xs:anyURI*
fn:xml-to-json($input as node()?) as xs:string?
fn:xml-to-json($input as node()?, $options as map(*)) as xs:string?
fn:year-from-date($arg as xs:date?) as xs:integer?
fn:year-from-dateTime($arg as xs:dateTime?) as xs:integer?
fn:years-from-duration($arg as xs:duration?) as xs:integer?
fn:zero-or-one($arg as item()*) as item()?
math:acos($arg as xs:double?) as xs:double?
math:asin($arg as xs:double?) as xs:double?
math:atan($arg as xs:double?) as xs:double?
math:atan2($y as xs:double, $x as xs:double) as xs:double
math:cos($theta as xs:double?) as xs:double?
math:exp($arg as xs:double?) as xs:double?
math:exp10($arg as xs:double?) as xs:double?
math:log($arg as xs:double?) as xs:double?
math:log10($arg as xs:double?) as xs:double?
math:pi() as xs:double
math:pow($x as xs:double?, $y as xs:numeric) as xs:double?
math:sin($theta as xs:double?) as xs:double?
math:sqrt($arg as xs:double?) as xs:double?
math:tan($theta as xs:double?) as xs:double?
map:contains($map as map(*), $key as xs:anyAtomicType) as xs:boolean
map:entry($key as xs:anyAtomicType, $value as item()*) as map(*)
map:find($input as item()*, $key as xs:anyAtomicType) as array(*)
map:for-each($map as map(*), $action as function(xs:anyAtomicType, item()*) as item()*) as item()*
map:get($map as map(*), $key as xs:anyAtomicType) as item()*
map:keys($map as map(*)) as xs:anyAtomicType*
map:merge($maps as map(*)*) as map(*)
map:merge($maps as map(*)*, $options as map(*)) as map(*)
map:put($map as map(*), $key as xs:anyAtomicType, $value as item()*) as map(*)
map:remove($map as map(*), $keys as xs:anyAtomicType*) as map(*)
map:size($map as map(*)) as xs:integer
array:append($array as array(*), $appendage as item()*) as array(*)
array:filter($array as array(*), $function as function(item()*) as xs:boolean) as array(*)
array:flatten($input as item()*) as item()*
array:fold-left($array as array(*), $zero as item()*, $function as function(item()*, item()*) as item()*) as item()*
array:fold-right($array as array(*), $zero as item()*, $function as function(item()*, item()*) as item()*) as item()*
array:for-each($array as array(*), $action as function(item()*) as item()*) as array(*)
array:for-each-pair($array1 as array(*), $array2 as array(*), $function as function(item()*, item()*) as item()*) as array(*)
array:get($array as array(*), $position as xs:integer) as item()*
array:head($array as array(*)) as item()*
array:insert-before($array as array(*), $position as xs:integer, $member as item()*) as array(*)
array:join($arrays as array(*)*) as array(*)
array:put($array as array(*), $position as xs:integer, $member as item()*) as array(*)
array:remove($array as array(*), $positions as xs:integer*) as array(*)
array:reverse($array as array(*)) as array(*)
array:size($array as array(*)) as xs:integer
array:sort($array as array(*)) as array(*)
array:sort($array as array(*), $collation as xs:string?) as array(*)
array:sort($array as array(*), $collation as xs:string?, $key as function(item()*) as xs:anyAtomicType*) as array(*)
array:subarray($array as array(*), $start as xs:integer) as array(*)
array:subarray($array as array(*), $start as xs:integer, $length as xs:integer) as array(*)
array:tail($array as array(*)) as array(*)
`
