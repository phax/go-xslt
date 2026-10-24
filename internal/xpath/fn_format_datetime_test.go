package xpath

import (
	"strings"
	"testing"
	"time"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// dtfEval parses and evaluates expr against <r/> and returns the string value
// of the result, or the error. now, when non-zero, pins current-dateTime().
func dtfEval(expr string, now time.Time) (string, error) {
	p, err := Parse(expr)
	if err != nil {
		return "", err
	}
	doc, _ := xmltree.Parse("<r/>")
	v, err := p.Eval(&Context{Node: doc, Pos: 1, Size: 1, Now: now})
	if err != nil {
		return "", err
	}
	return ToString(v), nil
}

type dtfCase struct {
	expr    string
	want    string // expected string value (ignored when err is set)
	err     string // expected error code
	comment string
}

func dtfRun(t *testing.T, cases []dtfCase) {
	t.Helper()
	for _, c := range cases {
		got, err := dtfEval(c.expr, time.Time{})
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%s: got (%q, %v), want error %s", c.expr, got, err, c.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

// The values used by the examples of F&O 3.1 §9.8.5.
const (
	dtfD  = `xs:date('2002-12-31')`
	dtfT  = `xs:time('15:58:45.762+02:00')`
	dtfDT = `xs:dateTime('2002-12-31T15:58:45.762+02:00')`
)

// TestFormatDTSpecExamples runs the Gregorian examples of F&O 3.1 §9.8.5
// whose output is fully determined (no implementation-defined names beyond
// English/German month and day names).
func TestFormatDTSpecExamples(t *testing.T) {
	dtfRun(t, []dtfCase{
		{expr: `format-date(` + dtfD + `, "[Y0001]-[M01]-[D01]")`, want: "2002-12-31"},
		{expr: `format-date(` + dtfD + `, "[M]-[D]-[Y]")`, want: "12-31-2002"},
		{expr: `format-date(` + dtfD + `, "[D]-[M]-[Y]")`, want: "31-12-2002"},
		{expr: `format-date(` + dtfD + `, "[D1] [MI] [Y]")`, want: "31 XII 2002"},
		{expr: `format-date(` + dtfD + `, "[D1o] [MNn], [Y]", "en", (), ())`, want: "31st December, 2002"},
		{expr: `format-date(` + dtfD + `, "[D01] [MN,*-3] [Y0001]", "en", (), ())`, want: "31 DEC 2002"},
		{expr: `format-date(` + dtfD + `, "[MNn] [D], [Y]", "en", (), ())`, want: "December 31, 2002"},
		{expr: `format-date(` + dtfD + `, "[D] [MNn], [Y]", "de", (), ())`, want: "31 Dezember, 2002"},
		{expr: `format-date(` + dtfD + `, "[[[Y0001]-[M01]-[D01]]]")`, want: "[2002-12-31]"},
		{expr: `format-date(` + dtfD + `, "[Dwo] [MNn]", "de", (), ())`, want: "einunddreißigste Dezember"},
		{expr: `format-time(` + dtfT + `, "[h]:[m01] [PN]", "en", (), ())`, want: "3:58 PM"},
		{expr: `format-time(` + dtfT + `, "[h]:[m01]:[s01] [Pn]", "en", (), ())`, want: "3:58:45 pm"},
		{expr: `format-time(` + dtfT + `,"[H01]:[m01]")`, want: "15:58"},
		{expr: `format-time(` + dtfT + `,"[H01]:[m01]:[s01].[f001]")`, want: "15:58:45.762"},
		{expr: `format-time(` + dtfT + `,"[H01]:[m01]:[s01] [z,6-6]", "en", (), ())`, want: "15:58:45 GMT+02:00"},
		{expr: `format-dateTime(` + dtfDT + `, "[h].[m01][Pn] on [FNn], [D1o] [MNn]")`, want: "3.58pm on Tuesday, 31st December"},
		{expr: `format-dateTime(` + dtfDT + `, "[M01]/[D01]/[Y0001] at [H01]:[m01]:[s01]")`, want: "12/31/2002 at 15:58:45"},
	})
}

// TestFormatDTTimezones is the timezone table of F&O 3.1 §9.8.4.6 (time
// 12:00:00 at each of the five offsets), plus its no-timezone rules.
func TestFormatDTTimezones(t *testing.T) {
	offsets := []string{"-10:00", "-05:00", "+00:00", "+05:30", "+13:00"}
	table := []struct {
		marker string
		want   [5]string
	}{
		{"[Z]", [5]string{"-10:00", "-05:00", "+00:00", "+05:30", "+13:00"}},
		{"[Z0]", [5]string{"-10", "-5", "+0", "+5:30", "+13"}},
		{"[Z0:00]", [5]string{"-10:00", "-5:00", "+0:00", "+5:30", "+13:00"}},
		{"[Z00:00]", [5]string{"-10:00", "-05:00", "+00:00", "+05:30", "+13:00"}},
		{"[Z0000]", [5]string{"-1000", "-0500", "+0000", "+0530", "+1300"}},
		{"[Z00:00t]", [5]string{"-10:00", "-05:00", "Z", "+05:30", "+13:00"}},
		{"[z]", [5]string{"GMT-10:00", "GMT-05:00", "GMT+00:00", "GMT+05:30", "GMT+13:00"}},
		{"[ZZ]", [5]string{"W", "R", "Z", "+05:30", "+13:00"}},
	}
	var cases []dtfCase
	for _, row := range table {
		for i, off := range offsets {
			cases = append(cases, dtfCase{
				expr: `format-time(xs:time('12:00:00` + off + `'), '` + row.marker + `')`,
				want: row.want[i],
			})
		}
	}
	cases = append(cases,
		// "If the date/time value to be formatted does not include a timezone
		// offset, then the timezone component specifier is generally ignored
		// ... The exception is ... military timezones (format ZZ) ... J".
		dtfCase{expr: `format-time(xs:time('12:00:00'), '[H01][Z][z]')`, want: "12"},
		dtfCase{expr: `format-time(xs:time('12:00:00'), '[ZZ]')`, want: "J"},
		// Military letters A..I, K..M east, N..Y west (§9.8.4.6).
		dtfCase{expr: `format-time(xs:time('12:00:00+01:00'), '[ZZ]')`, want: "A"},
		dtfCase{expr: `format-time(xs:time('12:00:00+09:00'), '[ZZ]')`, want: "I"},
		dtfCase{expr: `format-time(xs:time('12:00:00+10:00'), '[ZZ]')`, want: "K"},
		dtfCase{expr: `format-time(xs:time('12:00:00+12:00'), '[ZZ]')`, want: "M"},
		dtfCase{expr: `format-time(xs:time('12:00:00-01:00'), '[ZZ]')`, want: "N"},
		dtfCase{expr: `format-time(xs:time('12:00:00-12:00'), '[ZZ]')`, want: "Y"},
		// Three or four digits, no separator: hours and minutes run together.
		dtfCase{expr: `format-time(xs:time('12:00:00-05:00'), '[Z001]')`, want: "-500"},
		// A separator other than ':' is used as given ("01.01" → +10.30).
		dtfCase{expr: `format-time(xs:time('12:00:00+10:30'), '[Z01.01]')`, want: "+10.30"},
		// The timezone of a dateTime and of a date is formatted the same way.
		dtfCase{expr: `format-dateTime(xs:dateTime('2002-12-31T12:00:00-05:00'), '[Z0000]')`, want: "-0500"},
		dtfCase{expr: `format-date(xs:date('2002-12-31Z'), '[Z00:00t]')`, want: "Z"},
		// [ZN] with an IANA name in $place: the zone's name (§9.8.4.6
		// table, last row: 12:00 at +00:00 in winter is 07:00 EST).
		dtfCase{expr: `format-dateTime(xs:dateTime('2002-01-15T12:00:00Z'), '[H00]:[m00] [ZN]', 'en', (), 'America/New_York')`, want: "07:00 EST"},
		// $place moves the value into the zone, DST included (§9.8.4.8).
		dtfCase{expr: `format-dateTime(xs:dateTime('2015-08-15T12:00:00Z'), '[H01]:[m01] [Z]', 'en', (), 'America/New_York')`, want: "08:00 -04:00"},
	)
	dtfRun(t, cases)
}

// TestFormatDTComponents covers the individual component specifiers and the
// presentation/width modifiers of F&O 3.1 §9.8.4.1–9.8.4.5.
func TestFormatDTComponents(t *testing.T) {
	dtfRun(t, []dtfCase{
		// d = day in year.
		{expr: `format-date(xs:date('2002-12-31'), '[d]')`, want: "365"},
		{expr: `format-date(xs:date('2004-12-31'), '[d]')`, want: "366"},
		{expr: `format-date(xs:date('2004-02-01'), '[d001]')`, want: "032"},
		// F/W in the ISO calendar: Monday = 1; week 1 contains the first
		// Thursday, so 2002-12-31 is in week 1 (of 2003) (§9.8.4.8).
		{expr: `format-date(xs:date('2002-12-31'), '[F1]', 'en', 'ISO', ())`, want: "2"},
		{expr: `format-date(xs:date('2002-12-29'), '[F1]', 'en', 'ISO', ())`, want: "7"},
		{expr: `format-date(xs:date('2002-12-31'), '[W]', 'en', 'ISO', ())`, want: "1"},
		{expr: `format-date(xs:date('2005-01-02'), '[W]', 'en', 'ISO', ())`, want: "53"},
		{expr: `format-date(xs:date('2013-03-21'), '[W01]', 'en', 'ISO', ())`, want: "12"},
		// w: the week belongs to the month of its Thursday (§9.8.4.8 example:
		// 29 January 2013 and 1 February 2013 are both week 5 of January).
		{expr: `format-date(xs:date('2013-01-29'), '[w]')`, want: "5"},
		{expr: `format-date(xs:date('2013-02-01'), '[w]')`, want: "5"},
		{expr: `format-date(xs:date('2013-02-04'), '[w]')`, want: "1"},
		{expr: `format-date(xs:date('2013-01-07'), '[w]')`, want: "2"},
		// Day of week by name, with widths (§9.8.4.3: a name longer than
		// the maximum width is abbreviated or truncated to fit).
		{expr: `format-date(xs:date('2002-12-31'), '[FNn]')`, want: "Tuesday"},
		{expr: `format-date(xs:date('2002-12-31'), '[FN]')`, want: "TUESDAY"},
		{expr: `format-date(xs:date('2002-12-31'), '[Fn]')`, want: "tuesday"},
		{expr: `format-date(xs:date('2002-12-31'), '[FNn,3-3]')`, want: "Tue"},
		{expr: `format-date(xs:date('2002-12-31'), '[FNn,*-2]')`, want: "Tu"},
		{expr: `format-date(xs:date('2002-12-30'), '[FNn]')`, want: "Monday"},
		{expr: `format-date(xs:date('2003-01-05'), '[FNn]')`, want: "Sunday"},
		// German month/day names.
		{expr: `format-date(xs:date('2003-03-04'), '[FNn], [D]. [MNn] [Y]', 'de', (), ())`, want: "Dienstag, 4. März 2003"},
		{expr: `format-date(xs:date('2003-03-04'), '[MN]', 'de-DE', (), ())`, want: "MÄRZ"},
		// Hours.
		{expr: `format-time(xs:time('00:30:00'), '[H]')`, want: "0"},
		{expr: `format-time(xs:time('00:30:00'), '[H01]')`, want: "00"},
		{expr: `format-time(xs:time('00:30:00'), '[h] [Pn]')`, want: "12 am"},
		{expr: `format-time(xs:time('12:30:00'), '[h] [Pn]')`, want: "12 pm"},
		{expr: `format-time(xs:time('23:30:00'), '[h] [PN]')`, want: "11 PM"},
		// m and s default to "01" (§9.8.4.1 table).
		{expr: `format-time(xs:time('09:05:07'), '[H]:[m]:[s]')`, want: "9:05:07"},
		{expr: `format-time(xs:time('09:05:07'), '[m1]/[s1]')`, want: "5/7"},
		// Year: modulo 10^N for a multi-digit pattern or a max width (§9.8.4.4).
		{expr: `format-date(xs:date('2002-12-31'), '[Y01]')`, want: "02"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y,2-2]')`, want: "02"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y1,2]')`, want: "2002"},
		{expr: `format-date(xs:date('0033-12-31'), '[Y0001]')`, want: "0033"},
		{expr: `format-date(xs:date('2008-12-31'), '[Y9,999,*]')`, want: "2,008"},
		{expr: `format-date(xs:date('2002-12-31'), '[YI]')`, want: "MMII"},
		{expr: `format-date(xs:date('2002-12-31'), '[Yi]')`, want: "mmii"},
		// Width modifier on a decimal pattern raises the mandatory digits.
		{expr: `format-date(xs:date('2002-03-04'), '[D,2]-[M,2]')`, want: "04-03"},
		{expr: `format-date(xs:date('2002-03-04'), '[D01]-[M01]')`, want: "04-03"},
		// Alphabetic numbering.
		{expr: `format-date(xs:date('2002-03-04'), '[Da]/[MA]')`, want: "d/C"},
		// Roman numbering of a month.
		{expr: `format-date(xs:date('2002-04-04'), '[Mi]')`, want: "iv"},
		// Ordinal and words for day and month (German, via $language).
		{expr: `format-date(xs:date('2002-03-01'), '[Dwo] [MNn]', 'de', (), ())`, want: "erste März"},
		{expr: `format-date(xs:date('2002-03-21'), '[Dw]', 'de', (), ())`, want: "einundzwanzig"},
		{expr: `format-date(xs:date('2002-03-21'), '[DW]', 'de', (), ())`, want: "EINUNDZWANZIG"},
		{expr: `format-date(xs:date('2002-03-21'), '[DWw]', 'de', (), ())`, want: "Einundzwanzig"},
		// Whitespace inside a variable marker is ignored (§9.8.4.1).
		{expr: `format-date(xs:date('2002-12-31'), '[ Y 0001 ]-[M 01]')`, want: "2002-12"},
		// Literal "]]" is a single ']'.
		{expr: `format-date(xs:date('2002-12-31'), '[D]]]')`, want: "31]"},
		// An unsupported format token falls back to the component's default
		// presentation ("If the implementation does not support the use of
		// the requested format token, it must use the default").
		{expr: `format-date(xs:date('2002-12-31'), '[MX]')`, want: "12"},
		// A single character after the component is the FIRST presentation
		// modifier (§9.8.4.1), so [Mo] is the unsupported token "o", which
		// falls back to the default "1" — not an ordinal.
		{expr: `format-date(xs:date('2002-12-31'), '[Mo]')`, want: "12"},
		// t/c as second modifier change nothing for decimal digits.
		{expr: `format-date(xs:date('2002-12-31'), '[D1c]/[D1t]')`, want: "31/31"},
		// Optional digit after mandatory digits is not a decimal digit
		// pattern: a syntax error in the picture.
		{expr: `format-date(xs:date('2002-12-31'), '[Y999#]')`, err: "FOFD1340"},
		// Grouping separator in the year pattern; two-or-more-digit pattern
		// "99;999" has W = 5 → modulo 10^5.
		{expr: `format-date(xs:date('2002-12-31'), '[Y99;999]')`, want: "02;002"},
		// Roman numerals shorter than the minimum width are padded by
		// appending spaces (§9.8.4.3 "For other presentation modifiers").
		{expr: `format-date(xs:date('2002-12-31'), '[YI,6]')`, want: "MMII  "},
		// am/pm marker truncated to a maximum width of 1 (§9.8.4.3).
		{expr: `format-time(xs:time('15:00:00'), '[PNn,1-1]')`, want: "P"},
		// [ZN] with no identifiable zone name: fallback format +01:01
		// (§9.8.4.6).
		{expr: `format-time(xs:time('12:00:00+05:00'), '[ZN]')`, want: "+05:00"},
		// A timezone is "never shortened" by a width modifier (§9.8.4.6).
		{expr: `format-time(xs:time('12:00:00+05:30'), '[z,2-2]')`, want: "GMT+05:30"},
		// Empty input → empty sequence.
		{expr: `count(format-date((), '[Y]'))`, want: "0"},
		{expr: `count(format-time((), '[H]'))`, want: "0"},
		{expr: `count(format-dateTime((), '[H]'))`, want: "0"},
		// An untypedAtomic node value is atomized... but untypedAtomic is not
		// xs:date: only real date/time items are accepted.
		{expr: `format-date('2002-12-31', '[Y]')`, err: "XPTY0004"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 5, (), ())`, err: "XPTY0004"},
	})
}

// TestFormatDTFractional covers fractional seconds (§9.8.4.5).
func TestFormatDTFractional(t *testing.T) {
	dtfRun(t, []dtfCase{
		// [f1] keeps every digit of the value; [f001] three; [f01] two,
		// truncated rather than rounded.
		{expr: `format-time(xs:time('10:00:00.7629'), '[f]')`, want: "7629"},
		{expr: `format-time(xs:time('10:00:00.7629'), '[f1]')`, want: "7629"},
		{expr: `format-time(xs:time('10:00:00.7629'), '[f001]')`, want: "762"},
		{expr: `format-time(xs:time('10:00:00.7699'), '[f01]')`, want: "76"},
		{expr: `format-time(xs:time('10:00:00.5'), '[f001]')`, want: "500"},
		// "If exactly one digit is required, ... [f1,1-1]".
		{expr: `format-time(xs:time('10:00:00.7629'), '[f1,1-1]')`, want: "7"},
		// A minimum width pads with mandatory digits.
		{expr: `format-time(xs:time('10:00:00.762'), '[f,6-*]')`, want: "762000"},
		// Optional digits: [f0#] has at most two digits, at least one.
		{expr: `format-time(xs:time('10:00:00.5'), '[f0#]')`, want: "5"},
		{expr: `format-time(xs:time('10:00:00.567'), '[f0#]')`, want: "56"},
		// Grouping separator in the fractional pattern (§9.8.4.5's algorithm:
		// 25.8235 with 999'### → 823'5).
		{expr: `format-time(xs:time('10:00:25.8235'), "[f999'###]")`, want: "823'5"},
		// A zero fraction under [f1] is "0".
		{expr: `format-time(xs:time('10:00:00'), '[f1]')`, want: "0"},
		// Non-ASCII digit family: Arabic-Indic digits.
		{expr: `format-time(xs:time('10:00:00.25'), '[f` + "٠٠" + `]')`, want: "٢٥"},
	})
}

// TestFormatDTErrors covers FOFD1340 (bad picture syntax, §9.8.4.1/9.8.4.2,
// bad $calendar, §9.8.4.8) and FOFD1350 (component not available, §9.8.4.1).
func TestFormatDTErrors(t *testing.T) {
	dtfRun(t, []dtfCase{
		{expr: `format-date(xs:date('2002-12-31'), '[Y')`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[]')`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[X]')`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y,0]')`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y,3-2]')`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y,1-0]')`, err: "FOFD1340"},
		{expr: `format-time(xs:time('10:00:00.5'), '[f#9]')`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[H]')`, err: "FOFD1350"},
		{expr: `format-date(xs:date('2002-12-31'), '[m]')`, err: "FOFD1350"},
		{expr: `format-date(xs:date('2002-12-31'), '[f]')`, err: "FOFD1350"},
		{expr: `format-date(xs:date('2002-12-31'), '[P]')`, err: "FOFD1350"},
		{expr: `format-time(xs:time('10:00:00'), '[Y]')`, err: "FOFD1350"},
		{expr: `format-time(xs:time('10:00:00'), '[D]')`, err: "FOFD1350"},
		{expr: `format-time(xs:time('10:00:00'), '[F]')`, err: "FOFD1350"},
		{expr: `format-time(xs:time('10:00:00'), '[W]')`, err: "FOFD1350"},
		// $calendar: not an EQName, or a no-namespace name not in the list.
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 'en', '1AD', ())`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 'en', 'XYZ', ())`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 'en', 'Q{http://x', ())`, err: "FOFD1340"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 'en', '1x:AD', ())`, err: "FOFD1340"},
	})
}

// TestFormatDTCalendarLanguage covers the $language/$calendar arguments
// (§9.8.4.8): AD and ISO are supported as such; a recognised but unsupported
// calendar, and an unsupported language, fall back — and the output must
// then identify what was actually used.
func TestFormatDTCalendarLanguage(t *testing.T) {
	dtfRun(t, []dtfCase{
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 'en', 'AD', ())`, want: "2002"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 'en', 'ISO', ())`, want: "2002"},
		{expr: `format-date(xs:date('2002-12-31'), '[Y]', 'en', (), ())`, want: "2002"},
	})
	for _, expr := range []string{
		`format-date(xs:date('2002-12-31'), '[D] [MNn] [Y]', 'en', 'OS', ())`,
		`format-date(xs:date('2002-12-31'), '[D] [MNn] [Y]', 'en', 'Q{http://example.com/cal}mine', ())`,
	} {
		got, err := dtfEval(expr, time.Time{})
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		// The fallback must identify the calendar actually used.
		if !strings.HasPrefix(got, "[Calendar:") {
			t.Errorf("%s = %q, want a [Calendar: ...] prefix", expr, got)
		}
	}
	// Swedish is not supported: the result must identify the language
	// actually used, and the names come from that language.
	expr := `format-date(xs:date('2002-12-31'), '[FNn] [D] [MNn] [Y]', 'sv', (), ())`
	got, err := dtfEval(expr, time.Time{})
	if err != nil {
		t.Fatalf("%s: %v", expr, err)
	}
	if !strings.HasPrefix(got, "[Language:") || !strings.HasSuffix(got, "Tuesday 31 December 2002") {
		t.Errorf("%s = %q, want a [Language: ...] prefix and English names", expr, got)
	}
	// A purely numeric picture needs no language-dependent data, so the
	// same unsupported language changes nothing.
	expr = `format-date(xs:date('2002-12-31'), '[D]/[M]/[Y]', 'sv', (), ())`
	if got, err := dtfEval(expr, time.Time{}); err != nil || got != "31/12/2002" {
		t.Errorf("%s = (%q, %v), want 31/12/2002", expr, got, err)
	}
}

// TestFormatDTEraCalendarNames checks the implementation-defined E and C
// components (§9.8.4.7) only for what the spec does fix: they are names,
// and the era of a BC year differs from that of an AD year.
func TestFormatDTEraCalendarNames(t *testing.T) {
	ad, err1 := dtfEval(`format-date(xs:date('2002-12-31'), '[E]')`, time.Time{})
	bc, err2 := dtfEval(`format-date(xs:date('-0043-03-13'), '[E]')`, time.Time{})
	cal, err3 := dtfEval(`format-date(xs:date('2002-12-31'), '[C]')`, time.Time{})
	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("errors: %v %v %v", err1, err2, err3)
	}
	if ad == "" || bc == "" || ad == bc {
		t.Errorf("[E]: AD %q, BC %q — want two different non-empty era names", ad, bc)
	}
	if cal == "" {
		t.Errorf("[C] is empty, want a calendar name")
	}
	// The year itself is an absolute value (§9.8.4.1 "year (absolute value)").
	if got, err := dtfEval(`format-date(xs:date('-0043-03-13'), '[Y]')`, time.Time{}); err != nil || got != "43" {
		t.Errorf("[Y] of -0043 = (%q, %v), want 43", got, err)
	}
}
