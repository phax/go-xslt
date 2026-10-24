package xpath

import (
	"math/rand"
	"testing"
)

// TestDateTimeLexicalMatchers pins the hand-written matchers to the regular
// expressions they replace: for every string of a generated corpus — valid
// forms, every single-character mutation of them, and random strings over the
// grammar's alphabet — matchDateTime/matchDate/matchTime must accept exactly
// when reDateTime/reDate/reTime do, with exactly the same capture groups.
func TestDateTimeLexicalMatchers(t *testing.T) {
	valid := []string{
		"2024-04-16", "-0001-12-31", "12345-01-01", "0000-01-01Z", "2024-04-16+05:30", "2024-04-16-14:00",
		"2024-04-16T13:20:00", "2024-04-16T13:20:00.5", "2024-04-16T13:20:00.123456789Z", "-2024-04-16T24:00:00-00:00",
		"13:20:00", "13:20:00.001", "24:00:00Z", "00:00:00+14:00",
	}
	alphabet := []byte("0123456789-+:TZ. x")
	corpus := map[string]bool{}
	add := func(s string) { corpus[s] = true }
	for _, v := range valid {
		add(v)
		for i := 0; i <= len(v); i++ {
			// deletions
			if i < len(v) {
				add(v[:i] + v[i+1:])
			}
			for _, c := range alphabet {
				// insertions and replacements
				add(v[:i] + string(c) + v[i:])
				if i < len(v) {
					add(v[:i] + string(c) + v[i+1:])
				}
			}
		}
	}
	rng := rand.New(rand.NewSource(1))
	for n := 0; n < 20000; n++ {
		b := make([]byte, rng.Intn(24))
		for i := range b {
			b[i] = alphabet[rng.Intn(len(alphabet))]
		}
		add(string(b))
	}
	// Non-ASCII digits and a trailing newline: \d and $ are ASCII-only and
	// end-of-text in Go's regexp, which the matchers must mirror.
	add("２０２４-04-16")
	add("2024-04-16\n")
	add("13:20:00\n")

	for s := range corpus {
		if m := reDateTime.FindStringSubmatch(s); m != nil {
			p, ok := matchDateTime(s)
			if !ok {
				t.Errorf("dateTime %q: regexp matches, matcher does not", s)
				continue
			}
			for i := range p {
				if p[i] != m[i+1] {
					t.Errorf("dateTime %q: group %d: matcher %q, regexp %q", s, i+1, p[i], m[i+1])
				}
			}
		} else if _, ok := matchDateTime(s); ok {
			t.Errorf("dateTime %q: matcher matches, regexp does not", s)
		}
		if m := reDate.FindStringSubmatch(s); m != nil {
			p, ok := matchDate(s)
			if !ok {
				t.Errorf("date %q: regexp matches, matcher does not", s)
				continue
			}
			for i := range p {
				if p[i] != m[i+1] {
					t.Errorf("date %q: group %d: matcher %q, regexp %q", s, i+1, p[i], m[i+1])
				}
			}
		} else if _, ok := matchDate(s); ok {
			t.Errorf("date %q: matcher matches, regexp does not", s)
		}
		if m := reTime.FindStringSubmatch(s); m != nil {
			p, ok := matchTime(s)
			if !ok {
				t.Errorf("time %q: regexp matches, matcher does not", s)
				continue
			}
			for i := range p {
				if p[i] != m[i+1] {
					t.Errorf("time %q: group %d: matcher %q, regexp %q", s, i+1, p[i], m[i+1])
				}
			}
		} else if _, ok := matchTime(s); ok {
			t.Errorf("time %q: matcher matches, regexp does not", s)
		}
	}
	t.Logf("%d strings compared", len(corpus))
}

// TestCheckDateTimeLexicalMatchesCast pins CheckDateTimeLexical to the cast it
// stands in for: the same verdict and error text as CastTo(NewString(s), t).
func TestCheckDateTimeLexicalMatchesCast(t *testing.T) {
	cases := []string{
		"2024-04-16", "2024-02-30", "2024-04-16T13:20:00", "2024-04-16T13:20:00Z", "2024-04-16T25:00:00",
		"13:20:00", "24:00:01", "2024", "2024-04", "--04", "--04-16", "---16", "junk", "", " 2024-04-16 ",
	}
	types := []AtomType{XSdate, XSdateTime, XSdateTimeStamp, XStime, XSgYear, XSgYearMonth, XSgMonth, XSgMonthDay, XSgDay}
	for _, ty := range types {
		for _, s := range cases {
			handled, err := CheckDateTimeLexical(ty, s)
			if !handled {
				t.Fatalf("%s not handled", ty)
			}
			_, cerr := CastTo(NewString(s), ty)
			switch {
			case (err == nil) != (cerr == nil):
				t.Errorf("%s %q: check err %v, cast err %v", ty, s, err, cerr)
			case err != nil && err.Error() != cerr.Error():
				t.Errorf("%s %q: check %q, cast %q", ty, s, err, cerr)
			}
		}
	}
	if handled, _ := CheckDateTimeLexical(XSinteger, "1"); handled {
		t.Error("xs:integer must not be handled")
	}
}
