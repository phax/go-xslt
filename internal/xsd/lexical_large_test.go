package xsd

import (
	"encoding/base64"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xpath"
)

// largeBase64 returns about size bytes of base64 text over random data,
// broken into 76-character lines when wrapped (as MIME encoders and many
// e-invoice generators emit an embedded PDF).
func largeBase64(size int, wrapped bool) string {
	raw := make([]byte, size/4*3)
	rand.New(rand.NewSource(1)).Read(raw)
	s := base64.StdEncoding.EncodeToString(raw)
	if !wrapped {
		return s
	}
	var b strings.Builder
	for len(s) > 76 {
		b.WriteString(s[:76])
		b.WriteString("\n")
		s = s[76:]
	}
	b.WriteString(s)
	return b.String()
}

// BenchmarkValidateLargeBase64 validates one xs:base64Binary element holding
// about 25 MB of text — an e-invoice with an embedded PDF attachment.
func BenchmarkValidateLargeBase64(b *testing.B) {
	sch, err := Compile([]string{`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	  <xs:element name="v" type="xs:base64Binary"/>
	</xs:schema>`}, "", Version11)
	if err != nil {
		b.Fatal(err)
	}
	for _, wrapped := range []bool{false, true} {
		name := "oneline"
		if wrapped {
			name = "wrapped76"
		}
		doc := "<v>" + largeBase64(25<<20, wrapped) + "</v>"
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(doc)))
			for i := 0; i < b.N; i++ {
				res, err := sch.Validate(doc)
				if err != nil || !res.Valid {
					b.Fatalf("valid=%v err=%v", res, err)
				}
			}
		})
	}
}

// BenchmarkLexicalLargeBase64 isolates the whitespace collapse and the
// lexical check of a 25 MB base64Binary value (no XML parsing).
func BenchmarkLexicalLargeBase64(b *testing.B) {
	for _, wrapped := range []bool{false, true} {
		name := "oneline"
		if wrapped {
			name = "wrapped76"
		}
		v := largeBase64(25<<20, wrapped)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(v)))
			for i := 0; i < b.N; i++ {
				if err := checkPrimitiveLexical(xpath.XSbase64Binary, applyWhiteSpace("collapse", v)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The regexps the hand-written scanners replaced, kept as their oracle.
var (
	oldWSRun     = regexp.MustCompile(`[ \t\r\n]+`)
	oldHexBinary = regexp.MustCompile(`^([0-9A-Fa-f]{2})*$`)
	oldBase64    = regexp.MustCompile(`^((([A-Za-z0-9+/] ?){4})*(([A-Za-z0-9+/] ?){3}[A-Za-z0-9+/]|([A-Za-z0-9+/] ?){2}[AEIMQUYcgkosw048] ?=|[A-Za-z0-9+/] ?[AQgw] ?= ?=))?$`)
)

// oldCollapse is the previous collapse, minus its one deviation: TrimSpace
// also dropped non-XML Unicode whitespace (U+00A0, U+0085, U+2000…) at the
// ends, which XSD keeps. The oracle trims XML whitespace only.
func oldCollapse(s string) string {
	return strings.Trim(oldWSRun.ReplaceAllString(s, " "), " ")
}

// allStrings calls f with every string of length 0..maxLen over alphabet.
func allStrings(alphabet string, maxLen int, f func(string)) {
	buf := make([]byte, 0, maxLen)
	var rec func()
	rec = func() {
		f(string(buf))
		if len(buf) == maxLen {
			return
		}
		for i := 0; i < len(alphabet); i++ {
			buf = append(buf, alphabet[i])
			rec()
			buf = buf[:len(buf)-1]
		}
	}
	rec()
}

func TestLexicalScannersMatchRegexps(t *testing.T) {
	examples := []string{
		"", " ", "  ", "=", "==", "A", "AA", "AAA", "AAAA", "AAAA ", " AAAA",
		"AAAA  AAAA", "A A A A", "A A A A ", "AQ==", "AQ= =", "AQ = =", "A Q = =",
		"AQ== ", "AB==", "AAE=", "AAB=", "AAE= ", "AAE =", "AA=A", "A===", "====",
		"AAAAAQ==", "AQ==AAAA", "AAA=AAAA", "AAAA\tAAAA", "AAAA\nAAAA", "AAA!",
		"AAAé", "+/+/", "-_-_", "AAAAA", "AAAAAA==", "AAAAAAA=", "AQ=",
		"00", "0", "0g", "aF", "a F", "0123456789abcdefABCDEF", "abc", " 00", "00 ",
	}
	for _, v := range examples {
		if got, want := validBase64Binary(v), oldBase64.MatchString(v); got != want {
			t.Errorf("base64Binary %q: got %v, regexp says %v", v, got, want)
		}
		if got, want := validHexBinary(v), oldHexBinary.MatchString(v); got != want {
			t.Errorf("hexBinary %q: got %v, regexp says %v", v, got, want)
		}
	}
	// Exhaustive over alphabets that hit every branch: a B04 char (A), a
	// B16-only char (E), a plain char (B), padding, space, other whitespace
	// and an invalid character.
	allStrings("AEB= \t!", 8, func(v string) {
		if got, want := validBase64Binary(v), oldBase64.MatchString(v); got != want {
			t.Fatalf("base64Binary %q: got %v, regexp says %v", v, got, want)
		}
	})
	allStrings("0aFg ", 7, func(v string) {
		if got, want := validHexBinary(v), oldHexBinary.MatchString(v); got != want {
			t.Fatalf("hexBinary %q: got %v, regexp says %v", v, got, want)
		}
	})
	// Real encodings of every length, plain and wrapped.
	r := rand.New(rand.NewSource(2))
	for n := 0; n < 200; n++ {
		raw := make([]byte, n)
		r.Read(raw)
		for _, v := range []string{base64.StdEncoding.EncodeToString(raw), largeBase64Wrap(base64.StdEncoding.EncodeToString(raw), 5)} {
			for _, c := range []string{v, applyWhiteSpace("collapse", v)} {
				if got, want := validBase64Binary(c), oldBase64.MatchString(c); got != want {
					t.Fatalf("base64Binary %q: got %v, regexp says %v", c, got, want)
				}
			}
		}
	}
}

func largeBase64Wrap(s string, width int) string {
	var b strings.Builder
	for len(s) > width {
		b.WriteString(s[:width])
		b.WriteString("\r\n")
		s = s[width:]
	}
	b.WriteString(s)
	return b.String()
}

func TestCollapseWhiteSpace(t *testing.T) {
	examples := []string{
		"", " ", "\t", "a", " a", "a ", " a ", "a b", "a  b", "a\tb", "a\r\nb",
		"\n\na b\t\t", "ab", "a b c", "a \t b", " a ", "  a  ",
		"é  ü", "a\vb",
	}
	for _, v := range examples {
		if got, want := collapseWhiteSpace(v), oldCollapse(v); got != want {
			t.Errorf("collapse %q: got %q, want %q", v, got, want)
		}
	}
	allStrings("ab \t\r\n", 7, func(v string) {
		if got, want := collapseWhiteSpace(v), oldCollapse(v); got != want {
			t.Fatalf("collapse %q: got %q, want %q", v, got, want)
		}
	})
	// XSD whitespace is #x20 #x9 #xD #xA only: other Unicode spaces survive,
	// also at the ends.
	if got := applyWhiteSpace("collapse", "  x  "); got != "  x  " {
		t.Errorf("collapse must keep U+00A0: got %q", got)
	}
}
