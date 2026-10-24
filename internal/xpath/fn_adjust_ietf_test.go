package xpath

import "testing"

// TestAdjustToTimezone runs the examples of F&O 3.1 §9.6.1–9.6.3 that give
// an explicit $timezone (the one-argument examples assume an implicit
// timezone of -05:00; this implementation's is PT0S, checked separately).
func TestAdjustToTimezone(t *testing.T) {
	const tz10 = `xs:dayTimeDuration("-PT10H")`
	dtfRun(t, []dtfCase{
		// §9.6.1 fn:adjust-dateTime-to-timezone
		{expr: `string(adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00'), ` + tz10 + `))`, want: "2002-03-07T10:00:00-10:00"},
		{expr: `string(adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00-07:00'), ` + tz10 + `))`, want: "2002-03-07T07:00:00-10:00"},
		{expr: `string(adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00-07:00'), xs:dayTimeDuration("PT10H")))`, want: "2002-03-08T03:00:00+10:00"},
		{expr: `string(adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T00:00:00+01:00'), xs:dayTimeDuration("-PT8H")))`, want: "2002-03-06T15:00:00-08:00"},
		{expr: `string(adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00'), ()))`, want: "2002-03-07T10:00:00"},
		{expr: `string(adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00-07:00'), ()))`, want: "2002-03-07T10:00:00"},
		// §9.6.2 fn:adjust-date-to-timezone
		{expr: `string(adjust-date-to-timezone(xs:date("2002-03-07"), ` + tz10 + `))`, want: "2002-03-07-10:00"},
		{expr: `string(adjust-date-to-timezone(xs:date("2002-03-07-07:00"), ` + tz10 + `))`, want: "2002-03-06-10:00"},
		{expr: `string(adjust-date-to-timezone(xs:date("2002-03-07"), ()))`, want: "2002-03-07"},
		{expr: `string(adjust-date-to-timezone(xs:date("2002-03-07-07:00"), ()))`, want: "2002-03-07"},
		// §9.6.3 fn:adjust-time-to-timezone
		{expr: `string(adjust-time-to-timezone(xs:time("10:00:00"), ` + tz10 + `))`, want: "10:00:00-10:00"},
		{expr: `string(adjust-time-to-timezone(xs:time("10:00:00-07:00"), ` + tz10 + `))`, want: "07:00:00-10:00"},
		{expr: `string(adjust-time-to-timezone(xs:time("10:00:00"), ()))`, want: "10:00:00"},
		{expr: `string(adjust-time-to-timezone(xs:time("10:00:00-07:00"), ()))`, want: "10:00:00"},
		{expr: `string(adjust-time-to-timezone(xs:time("10:00:00-07:00"), xs:dayTimeDuration("PT10H")))`, want: "03:00:00+10:00"},
		// The adjusted time is a proper xs:time: equal to the literal.
		{expr: `adjust-time-to-timezone(xs:time("10:00:00-07:00"), xs:dayTimeDuration("PT10H")) eq xs:time("03:00:00+10:00")`, want: "true"},
		{expr: `adjust-date-to-timezone(xs:date("2002-03-07-07:00"), ` + tz10 + `) eq xs:date("2002-03-06-10:00")`, want: "true"},
		// Fractional seconds survive.
		{expr: `string(adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00.5Z'), xs:dayTimeDuration('PT1H')))`, want: "2002-03-07T11:00:00.5+01:00"},
		// Empty $arg → empty sequence.
		{expr: `count(adjust-dateTime-to-timezone((), ` + tz10 + `))`, want: "0"},
		{expr: `count(adjust-date-to-timezone(()))`, want: "0"},
		{expr: `count(adjust-time-to-timezone(()))`, want: "0"},
		// FODT0003: offset beyond ±PT14H or not an integral number of minutes.
		{expr: `adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00'), xs:dayTimeDuration('PT14H1M'))`, err: "FODT0003"},
		{expr: `adjust-date-to-timezone(xs:date('2002-03-07'), xs:dayTimeDuration('-PT15H'))`, err: "FODT0003"},
		{expr: `adjust-time-to-timezone(xs:time('10:00:00'), xs:dayTimeDuration('PT1H0M30S'))`, err: "FODT0003"},
		{expr: `adjust-time-to-timezone(xs:time('10:00:00'), xs:dayTimeDuration('PT14H0M0.001S'))`, err: "FODT0003"},
		// The extreme offsets themselves are allowed.
		{expr: `string(adjust-time-to-timezone(xs:time('10:00:00Z'), xs:dayTimeDuration('-PT14H')))`, want: "20:00:00-14:00"},
	})
}

// TestAdjustImplicitTimezone: the one-argument forms adjust to the implicit
// timezone (§9.6.1 rule: "If $timezone is not specified, then the effective
// value of $timezone is the value of the implicit timezone in the dynamic
// context"), so they equal the two-argument form with implicit-timezone().
func TestAdjustImplicitTimezone(t *testing.T) {
	dtfRun(t, []dtfCase{
		{expr: `let $v := xs:dateTime('2002-03-07T10:00:00-07:00') return
			string(adjust-dateTime-to-timezone($v)) eq string(adjust-dateTime-to-timezone($v, implicit-timezone()))`, want: "true"},
		{expr: `let $v := xs:dateTime('2002-03-07T10:00:00') return
			string(adjust-dateTime-to-timezone($v)) eq string(adjust-dateTime-to-timezone($v, implicit-timezone()))`, want: "true"},
		{expr: `let $v := xs:date('2002-03-07-07:00') return
			string(adjust-date-to-timezone($v)) eq string(adjust-date-to-timezone($v, implicit-timezone()))`, want: "true"},
		{expr: `let $v := xs:time('10:00:00-07:00') return
			string(adjust-time-to-timezone($v)) eq string(adjust-time-to-timezone($v, implicit-timezone()))`, want: "true"},
		// Adjusting keeps the instant (for a value that had a timezone).
		{expr: `adjust-dateTime-to-timezone(xs:dateTime('2002-03-07T10:00:00-07:00')) eq xs:dateTime('2002-03-07T10:00:00-07:00')`, want: "true"},
		{expr: `implicit-timezone() instance of xs:dayTimeDuration`, want: "true"},
	})
}

// TestIETFDate runs the examples and grammar of F&O 3.1 §9.9.1
// fn:parse-ietf-date.
func TestIETFDate(t *testing.T) {
	dtfRun(t, []dtfCase{
		// The five examples of the REC.
		{expr: `string(parse-ietf-date("Wed, 06 Jun 1994 07:29:35 GMT"))`, want: "1994-06-06T07:29:35Z"},
		{expr: `string(parse-ietf-date("Wed, 6 Jun 94 07:29:35 GMT"))`, want: "1994-06-06T07:29:35Z"},
		{expr: `string(parse-ietf-date("Wed Jun 06 11:54:45 EST 2013"))`, want: "2013-06-06T11:54:45-05:00"},
		{expr: `string(parse-ietf-date("Sunday, 06-Nov-94 08:49:37 GMT"))`, want: "1994-11-06T08:49:37Z"},
		{expr: `string(parse-ietf-date("Wed, 6 Jun 94 07:29:35 +0500"))`, want: "1994-06-06T07:29:35+05:00"},
		// Empty input → empty sequence.
		{expr: `count(parse-ietf-date(()))`, want: "0"},
		// The result is an xs:dateTime.
		{expr: `parse-ietf-date("06 Jun 1994 07:29:35 GMT") instance of xs:dateTime`, want: "true"},
		// No dayname; no seconds; no zone (00:00 assumed).
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29"))`, want: "1994-06-06T07:29:00Z"},
		// Case-insensitive.
		{expr: `string(parse-ietf-date("wed, 06 JUN 1994 07:29:35 gmt"))`, want: "1994-06-06T07:29:35Z"},
		// Leading/trailing and repeated whitespace (tab, newline) accepted.
		{expr: `string(parse-ietf-date("  Wed,` + "\t" + `06  Jun 1994` + "\n" + `07:29:35 UT  "))`, want: "1994-06-06T07:29:35Z"},
		// Fractional seconds.
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29:35.25 UTC"))`, want: "1994-06-06T07:29:35.25Z"},
		// Single-digit hours.
		{expr: `string(parse-ietf-date("06 Jun 1994 7:29:35 GMT"))`, want: "1994-06-06T07:29:35Z"},
		// Every tzname of the translation table.
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 EDT"))`, want: "1994-06-06T07:29:00-04:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 CST"))`, want: "1994-06-06T07:29:00-06:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 CDT"))`, want: "1994-06-06T07:29:00-05:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 MST"))`, want: "1994-06-06T07:29:00-07:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 MDT"))`, want: "1994-06-06T07:29:00-06:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 PST"))`, want: "1994-06-06T07:29:00-08:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 PDT"))`, want: "1994-06-06T07:29:00-07:00"},
		// tzoffset: H, HH, HMM, HHMM, and with a colon.
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 -5"))`, want: "1994-06-06T07:29:00-05:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 +05"))`, want: "1994-06-06T07:29:00+05:00"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 +130"))`, want: "1994-06-06T07:29:00+01:30"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 -0530"))`, want: "1994-06-06T07:29:00-05:30"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 -5:00"))`, want: "1994-06-06T07:29:00-05:00"},
		// Offset with zone name: the tzname is ignored.
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29 +0100 (GMT)"))`, want: "1994-06-06T07:29:00+01:00"},
		// No space needed between time and timezone (S? in the grammar).
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29:35GMT"))`, want: "1994-06-06T07:29:35Z"},
		{expr: `string(parse-ietf-date("06 Jun 1994 07:29:35-0500"))`, want: "1994-06-06T07:29:35-05:00"},
		// dsep may be a hyphen with optional spaces.
		{expr: `string(parse-ietf-date("06 - Jun - 1994 07:29 GMT"))`, want: "1994-06-06T07:29:00Z"},
		// A four-digit year is taken as given (0070), a two-digit one +1900.
		{expr: `string(parse-ietf-date("06 Jun 0070 07:29 GMT"))`, want: "0070-06-06T07:29:00Z"},
		{expr: `string(parse-ietf-date("06 Jun 70 07:29 GMT"))`, want: "1970-06-06T07:29:00Z"},
		// asctime without a timezone.
		{expr: `string(parse-ietf-date("Sun Nov  6 08:49:37 1994"))`, want: "1994-11-06T08:49:37Z"},
		// 24:00:00 is the end of the day, as in xs:dateTime.
		{expr: `string(parse-ietf-date("06 Jun 1994 24:00:00 GMT"))`, want: "1994-06-07T00:00:00Z"},
		// Leap day in a leap year.
		{expr: `string(parse-ietf-date("29 Feb 2000 00:00 GMT"))`, want: "2000-02-29T00:00:00Z"},

		// FORG0010: input not matching the grammar, or an invalid date.
		{expr: `parse-ietf-date("")`, err: "FORG0010"},
		{expr: `parse-ietf-date("31 Feb 2000 00:00 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("29 Feb 1999 00:00 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("32 Jan 2000 00:00 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("00 Jan 2000 00:00 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("123 Jan 2000 00:00 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 199 07:29 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1 07:29 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Foo 1994 07:29 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("Xyz, 06 Jun 1994 07:29 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 25:00 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:60 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29 CET")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29 -05:0")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 7 GMT")`, err: "FORG0010"},
		// Comma only directly after a dayname, followed by whitespace.
		{expr: `parse-ietf-date("Wed,06 Jun 1994 07:29 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06, Jun 1994 07:29 GMT")`, err: "FORG0010"},
		// A parenthesised tzname only after a tzoffset, and it must be a tzname.
		{expr: `parse-ietf-date("06 Jun 1994 07:29 (GMT)")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29 +0100 (CET)")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29 +0100 (GMT")`, err: "FORG0010"},
		// Junk tokens, duplicated fields.
		{expr: `parse-ietf-date("06 Jun 1994 1995 07:29 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29 08:30 GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29 GMT GMT")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29:35GMT GMT")`, err: "FORG0010"},
		// Offset beyond ±14:00 or minutes ≥ 60 is not a valid timezone.
		{expr: `parse-ietf-date("06 Jun 1994 07:29 -15:00")`, err: "FORG0010"},
		{expr: `parse-ietf-date("06 Jun 1994 07:29 -05:60")`, err: "FORG0010"},
	})
}
