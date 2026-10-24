package xpath

import (
	"testing"
	"time"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// TestDTSpecComponents runs the examples of F&O 3.1 §8.2 (duration
// components) and §9.5 (date/time component extraction).
func TestDTSpecComponents(t *testing.T) {
	dtfRun(t, []dtfCase{
		// §8.2 duration components
		{expr: `years-from-duration(xs:yearMonthDuration("P20Y15M"))`, want: "21"},
		{expr: `years-from-duration(xs:yearMonthDuration("-P15M"))`, want: "-1"},
		{expr: `years-from-duration(xs:dayTimeDuration("-P2DT15H"))`, want: "0"},
		{expr: `months-from-duration(xs:yearMonthDuration("P20Y15M"))`, want: "3"},
		{expr: `months-from-duration(xs:yearMonthDuration("-P20Y18M"))`, want: "-6"},
		{expr: `months-from-duration(xs:dayTimeDuration("-P2DT15H0M0S"))`, want: "0"},
		{expr: `days-from-duration(xs:dayTimeDuration("P3DT10H"))`, want: "3"},
		{expr: `days-from-duration(xs:dayTimeDuration("P3DT55H"))`, want: "5"},
		{expr: `days-from-duration(xs:yearMonthDuration("P3Y5M"))`, want: "0"},
		{expr: `days-from-duration(xs:dayTimeDuration("-P3DT10H"))`, want: "-3"},
		{expr: `hours-from-duration(xs:dayTimeDuration("P3DT10H"))`, want: "10"},
		{expr: `hours-from-duration(xs:dayTimeDuration("P3DT12H32M12S"))`, want: "12"},
		{expr: `hours-from-duration(xs:dayTimeDuration("PT123H"))`, want: "3"},
		{expr: `hours-from-duration(xs:dayTimeDuration("-P3DT10H"))`, want: "-10"},
		{expr: `minutes-from-duration(xs:dayTimeDuration("P3DT10H"))`, want: "0"},
		{expr: `minutes-from-duration(xs:dayTimeDuration("-P5DT12H30M"))`, want: "-30"},
		{expr: `seconds-from-duration(xs:dayTimeDuration("P3DT10H12.5S"))`, want: "12.5"},
		{expr: `seconds-from-duration(xs:dayTimeDuration("-PT256S"))`, want: "-16"},
		{expr: `seconds-from-duration(xs:dayTimeDuration("P3DT8H2M1.03S"))`, want: "1.03"},
		{expr: `seconds-from-duration(xs:dayTimeDuration("PT0S"))`, want: "0"},
		{expr: `seconds-from-duration(xs:dayTimeDuration("PT1S")) instance of xs:decimal`, want: "true"},
		{expr: `count(years-from-duration(()))`, want: "0"},
		{expr: `count(seconds-from-duration(()))`, want: "0"},

		// §9.5 dateTime components
		{expr: `year-from-dateTime(xs:dateTime("1999-05-31T13:20:00-05:00"))`, want: "1999"},
		{expr: `year-from-dateTime(xs:dateTime("1999-05-31T21:30:00-05:00"))`, want: "1999"},
		{expr: `year-from-dateTime(xs:dateTime("1999-12-31T19:20:00"))`, want: "1999"},
		{expr: `year-from-dateTime(xs:dateTime("1999-12-31T24:00:00"))`, want: "2000"},
		{expr: `year-from-dateTime(xs:dateTime("-0002-06-06T00:00:00"))`, want: "-2"},
		{expr: `month-from-dateTime(xs:dateTime("1999-05-31T13:20:00-05:00"))`, want: "5"},
		{expr: `month-from-dateTime(xs:dateTime("1999-12-31T19:20:00-05:00"))`, want: "12"},
		{expr: `month-from-dateTime(adjust-dateTime-to-timezone(xs:dateTime("1999-12-31T19:20:00-05:00"), xs:dayTimeDuration("PT0S")))`, want: "1"},
		{expr: `day-from-dateTime(xs:dateTime("1999-05-31T13:20:00-05:00"))`, want: "31"},
		{expr: `day-from-dateTime(xs:dateTime("1999-12-31T20:00:00-05:00"))`, want: "31"},
		{expr: `day-from-dateTime(adjust-dateTime-to-timezone(xs:dateTime("1999-12-31T19:20:00-05:00"), xs:dayTimeDuration("PT0S")))`, want: "1"},
		{expr: `hours-from-dateTime(xs:dateTime("1999-05-31T08:20:00-05:00"))`, want: "8"},
		{expr: `hours-from-dateTime(xs:dateTime("1999-12-31T21:20:00-05:00"))`, want: "21"},
		{expr: `hours-from-dateTime(adjust-dateTime-to-timezone(xs:dateTime("1999-12-31T21:20:00-05:00"), xs:dayTimeDuration("PT0S")))`, want: "2"},
		{expr: `hours-from-dateTime(xs:dateTime("1999-12-31T12:00:00"))`, want: "12"},
		{expr: `hours-from-dateTime(xs:dateTime("1999-12-31T24:00:00"))`, want: "0"},
		{expr: `minutes-from-dateTime(xs:dateTime("1999-05-31T13:20:00-05:00"))`, want: "20"},
		{expr: `minutes-from-dateTime(xs:dateTime("1999-05-31T13:30:00+05:30"))`, want: "30"},
		{expr: `seconds-from-dateTime(xs:dateTime("1999-05-31T13:20:00-05:00"))`, want: "0"},
		{expr: `seconds-from-dateTime(xs:dateTime("1999-05-31T13:20:12.25Z"))`, want: "12.25"},
		{expr: `string(timezone-from-dateTime(xs:dateTime("1999-05-31T13:20:00-05:00")))`, want: "-PT5H"},
		{expr: `string(timezone-from-dateTime(xs:dateTime("2000-06-12T13:20:00Z")))`, want: "PT0S"},
		{expr: `count(timezone-from-dateTime(xs:dateTime("2004-08-27T00:00:00")))`, want: "0"},
		{expr: `count(year-from-dateTime(()))`, want: "0"},

		// §9.5 date components
		{expr: `year-from-date(xs:date("1999-05-31"))`, want: "1999"},
		{expr: `year-from-date(xs:date("2000-01-01+05:00"))`, want: "2000"},
		{expr: `year-from-date(xs:date("-0002-06-01"))`, want: "-2"},
		{expr: `month-from-date(xs:date("1999-05-31-05:00"))`, want: "5"},
		{expr: `month-from-date(xs:date("2000-01-01+05:00"))`, want: "1"},
		{expr: `day-from-date(xs:date("1999-05-31-05:00"))`, want: "31"},
		{expr: `day-from-date(xs:date("2000-01-01+05:00"))`, want: "1"},
		{expr: `string(timezone-from-date(xs:date("1999-05-31-05:00")))`, want: "-PT5H"},
		{expr: `string(timezone-from-date(xs:date("2000-06-12Z")))`, want: "PT0S"},
		{expr: `count(timezone-from-date(xs:date("2000-06-12")))`, want: "0"},

		// §9.5 time components
		{expr: `hours-from-time(xs:time("11:23:00"))`, want: "11"},
		{expr: `hours-from-time(xs:time("21:23:00"))`, want: "21"},
		{expr: `hours-from-time(xs:time("01:23:00+05:00"))`, want: "1"},
		{expr: `hours-from-time(adjust-time-to-timezone(xs:time("01:23:00+05:00"), xs:dayTimeDuration("PT0S")))`, want: "20"},
		{expr: `hours-from-time(xs:time("24:00:00"))`, want: "0"},
		{expr: `minutes-from-time(xs:time("13:00:00Z"))`, want: "0"},
		{expr: `minutes-from-time(xs:time("13:45:00Z"))`, want: "45"},
		{expr: `seconds-from-time(xs:time("13:20:10.5"))`, want: "10.5"},
		{expr: `string(timezone-from-time(xs:time("13:20:00-05:00")))`, want: "-PT5H"},
		{expr: `count(timezone-from-time(xs:time("13:20:00")))`, want: "0"},
		{expr: `count(hours-from-time(()))`, want: "0"},
	})
}

// TestDTSpecDateTimeConstructor runs F&O 3.1 §9.3.1 fn:dateTime.
func TestDTSpecDateTimeConstructor(t *testing.T) {
	dtfRun(t, []dtfCase{
		{expr: `string(dateTime(xs:date("1999-12-31"), xs:time("12:00:00")))`, want: "1999-12-31T12:00:00"},
		{expr: `string(dateTime(xs:date("1999-12-31"), xs:time("24:00:00")))`, want: "1999-12-31T00:00:00"},
		// A timezone on either argument goes onto the result.
		{expr: `string(dateTime(xs:date("1999-12-31Z"), xs:time("12:00:00")))`, want: "1999-12-31T12:00:00Z"},
		{expr: `string(dateTime(xs:date("1999-12-31"), xs:time("12:00:00+01:00")))`, want: "1999-12-31T12:00:00+01:00"},
		{expr: `string(dateTime(xs:date("1999-12-31+01:00"), xs:time("12:00:00+01:00")))`, want: "1999-12-31T12:00:00+01:00"},
		{expr: `string(dateTime(xs:date("1999-12-31"), xs:time("12:00:00.25")))`, want: "1999-12-31T12:00:00.25"},
		{expr: `dateTime(xs:date("1999-12-31"), xs:time("12:00:00")) instance of xs:dateTime`, want: "true"},
		// Either argument empty → empty sequence.
		{expr: `count(dateTime((), xs:time("12:00:00")))`, want: "0"},
		{expr: `count(dateTime(xs:date("1999-12-31"), ()))`, want: "0"},
		// FORG0008: both have a timezone and they differ.
		{expr: `dateTime(xs:date("1999-12-31+01:00"), xs:time("12:00:00Z"))`, err: "FORG0008"},
	})
}

// TestDTSpecCurrent checks fn:current-dateTime/-date/-time and
// fn:implicit-timezone (F&O 3.1 §15.3–15.6): they are taken from the dynamic
// context (here Context.Now, pinned) and are stable within one evaluation.
func TestDTSpecCurrent(t *testing.T) {
	now := time.Date(2020, 2, 29, 23, 15, 30, 500_000_000, time.UTC)
	for _, c := range []struct{ expr, want string }{
		{`string(current-dateTime())`, "2020-02-29T23:15:30.5Z"},
		{`string(current-date())`, "2020-02-29Z"},
		{`string(current-time())`, "23:15:30.5Z"},
		{`current-dateTime() eq dateTime(current-date(), current-time())`, "true"},
		{`current-dateTime() instance of xs:dateTime`, "true"},
		{`timezone-from-dateTime(current-dateTime()) eq implicit-timezone()`, "true"},
	} {
		got, err := dtfEval(c.expr, now)
		if err != nil || got != c.want {
			t.Errorf("%s = (%q, %v), want %q", c.expr, got, err, c.want)
		}
	}
}

// TestDTSpecDefaultLanguage checks fn:default-language (§5.1 of F&O 3.1
// 'default language' in the dynamic context): the host's setting when one
// is configured, an xs:language in every case.
func TestDTSpecDefaultLanguage(t *testing.T) {
	doc, _ := xmltree.Parse("<r/>")
	p, err := Parse(`default-language()`)
	if err != nil {
		t.Fatal(err)
	}
	v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1, DefaultLanguage: "de"})
	if err != nil || ToString(v) != "de" {
		t.Errorf("default-language() with DefaultLanguage=de = (%v, %v)", v, err)
	}
	dtfRun(t, []dtfCase{
		{expr: `default-language() instance of xs:language`, want: "true"},
	})
}
