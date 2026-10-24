package xpath

import "fmt"

// Hand-written matchers for the three date/time lexical grammars every XSD
// validation and every xs:date/xs:dateTime/xs:time cast goes through. Each is
// the exact equivalent of the regular expression it stands beside (reDateTime,
// reDate, reTime in datetime.go) — the same grammar, the same captured
// substrings, "" for an absent optional group — but returns the groups in a
// fixed-size array instead of the slice regexp.FindStringSubmatch allocates
// per call, and without the regexp machinery. TestDateTimeLexicalMatchers
// checks the two agree, string for string, over a generated corpus.
//
// Grammar notes shared by all three: \d in Go regexp is ASCII [0-9] only, and
// $ is the end of the text (not of a line), so the matchers accept ASCII digits
// only and require the string to end where the grammar does.

// dateTimeParts are the capture groups of reDateTime: year (with its sign),
// month, day, hour, minute, seconds (with any fraction), timezone.
type dateTimeParts [7]string

// digitsFrom returns the index just past the run of ASCII digits starting at i.
func digitsFrom(s string, i int) int {
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

// twoDigitsAt reports whether s[i:i+2] is exactly two ASCII digits.
func twoDigitsAt(s string, i int) bool {
	return i+2 <= len(s) && s[i] >= '0' && s[i] <= '9' && s[i+1] >= '0' && s[i+1] <= '9'
}

// matchYearPrefix matches (-?\d{4,}) at the start of s, returning the group and
// the index just past it.
func matchYearPrefix(s string) (year string, end int, ok bool) {
	i := 0
	if i < len(s) && s[i] == '-' {
		i++
	}
	j := digitsFrom(s, i)
	if j-i < 4 {
		return "", 0, false
	}
	return s[:j], j, true
}

// matchTimeZoneEnd matches (Z|[+-]\d{2}:\d{2})?$ starting at i: the rest of s
// must be empty, "Z", or a signed hh:mm offset.
func matchTimeZoneEnd(s string, i int) (tz string, ok bool) {
	rest := s[i:]
	switch {
	case rest == "":
		return "", true
	case rest == "Z":
		return rest, true
	case len(rest) == 6 && (rest[0] == '+' || rest[0] == '-') && twoDigitsAt(rest, 1) && rest[3] == ':' && twoDigitsAt(rest, 4):
		return rest, true
	}
	return "", false
}

// matchSeconds matches \d{2}(?:\.\d+)? at i, returning the index just past it.
func matchSeconds(s string, i int) (end int, ok bool) {
	if !twoDigitsAt(s, i) {
		return 0, false
	}
	j := i + 2
	if j < len(s) && s[j] == '.' {
		k := digitsFrom(s, j+1)
		if k == j+1 {
			return 0, false // a '.' must be followed by at least one digit
		}
		j = k
	}
	return j, true
}

// matchDateTime is reDateTime:
// ^(-?\d{4,})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2}(?:\.\d+)?)(Z|[+-]\d{2}:\d{2})?$
func matchDateTime(s string) (p dateTimeParts, ok bool) {
	year, i, ok := matchYearPrefix(s)
	if !ok || i >= len(s) || s[i] != '-' || !twoDigitsAt(s, i+1) {
		return p, false
	}
	mo := s[i+1 : i+3]
	i += 3
	if i >= len(s) || s[i] != '-' || !twoDigitsAt(s, i+1) {
		return p, false
	}
	dy := s[i+1 : i+3]
	i += 3
	if i >= len(s) || s[i] != 'T' || !twoDigitsAt(s, i+1) {
		return p, false
	}
	hh := s[i+1 : i+3]
	i += 3
	if i >= len(s) || s[i] != ':' || !twoDigitsAt(s, i+1) {
		return p, false
	}
	mm := s[i+1 : i+3]
	i += 3
	if i >= len(s) || s[i] != ':' {
		return p, false
	}
	j, ok := matchSeconds(s, i+1)
	if !ok {
		return p, false
	}
	ss := s[i+1 : j]
	tz, ok := matchTimeZoneEnd(s, j)
	if !ok {
		return p, false
	}
	return dateTimeParts{year, mo, dy, hh, mm, ss, tz}, true
}

// matchDate is reDate: ^(-?\d{4,})-(\d{2})-(\d{2})(Z|[+-]\d{2}:\d{2})?$
// (groups year, month, day, timezone).
func matchDate(s string) (p [4]string, ok bool) {
	year, i, ok := matchYearPrefix(s)
	if !ok || i >= len(s) || s[i] != '-' || !twoDigitsAt(s, i+1) {
		return p, false
	}
	mo := s[i+1 : i+3]
	i += 3
	if i >= len(s) || s[i] != '-' || !twoDigitsAt(s, i+1) {
		return p, false
	}
	dy := s[i+1 : i+3]
	i += 3
	tz, ok := matchTimeZoneEnd(s, i)
	if !ok {
		return p, false
	}
	return [4]string{year, mo, dy, tz}, true
}

// matchTime is reTime: ^(\d{2}):(\d{2}):(\d{2}(?:\.\d+)?)(Z|[+-]\d{2}:\d{2})?$
// (groups hour, minute, seconds, timezone).
func matchTime(s string) (p [4]string, ok bool) {
	if !twoDigitsAt(s, 0) || len(s) < 3 || s[2] != ':' || !twoDigitsAt(s, 3) || len(s) < 6 || s[5] != ':' {
		return p, false
	}
	j, ok := matchSeconds(s, 6)
	if !ok {
		return p, false
	}
	tz, ok := matchTimeZoneEnd(s, j)
	if !ok {
		return p, false
	}
	return [4]string{s[0:2], s[3:5], s[6:j], tz}, true
}

// CheckDateTimeLexical reports whether s is a valid lexical form of the
// date/time type t: exactly the verdict CastTo(NewString(s), t) reaches, with
// the same error, but without building either atomic (an XSD validation asks
// this of every date in an instance and keeps neither value). handled is false
// when t is not a date/time type at all, and the caller casts as before.
func CheckDateTimeLexical(t AtomType, s string) (handled bool, err error) {
	if !isDateTimeType(t) {
		return false, nil
	}
	_, hasTZ, err := parseDateTimeValue(t, s)
	if err != nil {
		return true, fmt.Errorf("err:FORG0001: %v", err)
	}
	if t == XSdateTimeStamp && !hasTZ {
		return true, fmt.Errorf("err:FORG0001: xs:dateTimeStamp requires a timezone")
	}
	return true, nil
}
