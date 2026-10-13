package conformance

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tim-riep/go-xslt/internal/xpath"
)

// TestUnicodeCategoryEscapesAgainstSuiteData checks the XSD category escapes
// (\p{Lu}, \P{Lu}, \p{L}, ... \p{C}) of the regex engine against the Unicode 9.0
// data the W3C xslt30-test suite ships in misc/unicode-90/docs, for every code
// point.
//
// Why this instead of the suite's own unicode-90 test cases (which the harness
// skips as "unicode-version-pinned"): the supplied stylesheet runs one
// matches() call per code point per case and takes minutes per case, and every
// generated case omits the charclass parameter its stylesheet needs (it
// defaults to 'Ll', so a case for Zs asks whether the Zs characters are Ll).
// Neither is a statement about the Unicode version. What IS version-dependent
// is exactly the set of code points whose category differs between Unicode 9.0
// (the suite's data) and the 15.0 tables Go ships, and the XPath F&O spec makes
// the version implementation-defined. So the engine must agree with the suite
// on every code point, except those whose category changed — and the number of
// changed code points is pinned, which keeps the exception honest.
func TestUnicodeCategoryEscapesAgainstSuiteData(t *testing.T) {
	docs := filepath.Join(repoRoot(), "test", "conformance", "xslt30-test", "tests", "misc", "unicode-90", "docs")
	if _, err := os.Stat(docs); err != nil {
		t.Skip("xslt30-test suite not present")
	}
	leaves := []string{"Cc", "Cf", "Cn", "Co", "Cs", "Ll", "Lm", "Lo", "Lt", "Lu", "Mc", "Me", "Mn",
		"Nd", "Nl", "No", "Pc", "Pd", "Pe", "Pf", "Pi", "Po", "Ps", "Sc", "Sk", "Sm", "So", "Zl", "Zp", "Zs"}
	cp := regexp.MustCompile(`\sd="(\d+)"`)
	cat9 := make([]string, 0x110000)
	for _, leaf := range leaves {
		b, err := os.ReadFile(filepath.Join(docs, "unicode-"+leaf+".xml"))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range cp.FindAllStringSubmatch(string(b), -1) {
			n, _ := strconv.Atoi(m[1])
			cat9[n] = leaf
		}
	}

	// Every non-surrogate code point as one string, and each rune's start byte.
	var sb strings.Builder
	offset := make([]int, 0x110000)
	for r := rune(0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		offset[r] = sb.Len()
		sb.WriteRune(r)
	}
	all := sb.String()
	runeAt := map[int]rune{}
	for r := rune(0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		runeAt[offset[r]] = r
	}

	// engineHas[cat] is the set of code points \p{cat} matches.
	engineSet := func(name string) []bool {
		re, err := xpath.CompileRegex(`\p{`+name+`}`, "")
		if err != nil {
			t.Fatalf("\\p{%s}: %v", name, err)
		}
		set := make([]bool, 0x110000)
		for _, loc := range re.FindAllStringIndex(all, -1) {
			if loc[1]-loc[0] != utf8.RuneLen(runeAt[loc[0]]) {
				t.Fatalf("\\p{%s} matched more than one rune at %d", name, loc[0])
			}
			set[runeAt[loc[0]]] = true
		}
		return set
	}

	names := append([]string{}, leaves...)
	names = append(names, "C", "L", "M", "N", "P", "S", "Z")
	changed := map[rune]bool{}
	for _, name := range names {
		eng := engineSet(name)
		for r := rune(0); r <= 0x10FFFF; r++ {
			if r >= 0xD800 && r <= 0xDFFF {
				continue
			}
			if cat9[r] == "" {
				continue // not in the suite's data: the XML-illegal controls U+0000..U+001F
			}
			in9 := strings.HasPrefix(cat9[r], name)
			if eng[r] != in9 {
				changed[r] = true
			}
		}
	}
	// Code points whose general category differs between Unicode 9.0 and 15.0,
	// counted from the two DerivedGeneralCategory.txt files: 21,066 — almost all
	// unassigned (Cn) in 9.0 and assigned since (18,935 became Lo), plus a few
	// reclassifications such as Lo to Ll.
	const wantChanged = 21066
	if len(changed) != wantChanged {
		t.Errorf("%d code points disagree with the Unicode 9.0 data, want exactly the %d whose category changed in Unicode 15.0", len(changed), wantChanged)
	}
}
