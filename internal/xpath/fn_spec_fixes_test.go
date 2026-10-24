package xpath

import "testing"

// Regression tests for wrong results on valid input, each expectation taken
// from the F&O 3.1 REC text quoted beside it.

func TestReplaceQFlagLiteralReplacement(t *testing.T) {
	// §5.6.4: "If the q flag is present, the replacement string is used as
	// is." — with 'i' too, the pattern stays literal as well.
	jsnRun(t, "", nil, []jsnCase{
		{`replace("A.b", "a.", "\", "iq")`, `\b`},
		{`replace("A.b", "a.", "$1", "iq")`, `$1b`},
		{`replace("AxB", "a.", "z", "iq")`, `AxB`},
		{`replace("a.b", "a.", "\", "q")`, `\b`},
	})
}

func TestNamespaceURIForPrefixUndeclaredDefault(t *testing.T) {
	// §10.2.5: "If $element has no default namespace, the function returns
	// the empty sequence." — xmlns="" undeclares it (the REC's own example).
	jsnRun(t, `<z:a xmlns="http://example.org/one" xmlns:z="http://example.org/two"><b xmlns=""/></z:a>`, nil, []jsnCase{
		{`count(namespace-uri-for-prefix("", //b))`, "0"},
		{`count(namespace-uri-for-prefix((), //b))`, "0"},
		{`namespace-uri-for-prefix("", /*)`, "http://example.org/one"},
		{`namespace-uri-for-prefix("z", //b)`, "http://example.org/two"},
	})
}

func TestFormatDateWidthAndDefaults(t *testing.T) {
	jsnRun(t, "", nil, []jsnCase{
		// §9.8.4.1 table: the default presentation modifier of F is n.
		{`format-date(xs:date('2002-12-31'), '[F]')`, "tuesday"},
		// §9.8.4.3: with a width modifier on a decimal digit pattern "The
		// maximum width, if specified, is ignored."
		{`format-date(xs:date('2002-12-31'), '[D,1-1]')`, "31"},
		{`format-date(xs:date('2002-12-31'), '[M,1-1]')`, "12"},
		// §9.8.4.3: "If FN is shorter than the minimum width, then it is
		// padded by appending spaces to the end of the name."
		{`format-date(xs:date('2002-12-31'), '[MNn,10]')`, "December  "},
		{`format-date(xs:date('2002-12-31'), '[FNn,3-3]')`, "Tue"},
		// §9.8.4.4: the year IS reduced modulo 10^max-width.
		{`format-date(xs:date('2002-12-31'), '[Y,2-2]')`, "02"},
		{`format-date(xs:date('2002-12-31'), '[Y01]')`, "02"},
	})
}

func TestCurrentDateTimeIsDateTimeStamp(t *testing.T) {
	// §15.3: fn:current-dateTime() as xs:dateTimeStamp; the operators on it
	// still return plain xs:dateTime.
	jsnRun(t, "", nil, []jsnCase{
		{`current-dateTime() instance of xs:dateTimeStamp`, "true"},
		{`current-dateTime() instance of xs:dateTime`, "true"},
		{`(current-dateTime() + xs:dayTimeDuration('PT1H')) instance of xs:dateTimeStamp`, "false"},
		{`adjust-dateTime-to-timezone(current-dateTime(), ()) instance of xs:dateTimeStamp`, "false"},
		{`current-dateTime() eq current-dateTime()`, "true"},
	})
}

func TestRandomNumberGeneratorFunctionsAnonymous(t *testing.T) {
	// §4.9.1: next and permute have "name: absent"; §16.1.2 function-name
	// returns () for an anonymous function.
	jsnRun(t, "", nil, []jsnCase{
		{`count(function-name(random-number-generator()?next))`, "0"},
		{`count(function-name(random-number-generator()?permute))`, "0"},
		{`function-arity(random-number-generator()?permute)`, "1"},
	})
}
