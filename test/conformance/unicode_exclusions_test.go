package conformance

import (
	"os"
	"strings"
)

// The two test sets that depend on a Unicode version (unicode-90 is written
// for Unicode 9.0, regex-classes for 3.1 to 6.0) run like every other set.
// Only the cases below are excluded, each for one of two reasons that are
// about the case, not about the engine:
//
//   - "unicode-version": the expected result is the exact character data of an
//     old Unicode version, and the Unicode version is implementation-defined
//     (F&O 3.1, Conformance; XSD 1.1 Part 2 likewise). This engine uses
//     Unicode 15.0, and agrees with it code point for code point (checked
//     against Go's tables; TestUnicodeCategoryEscapesAgainstSuiteData does
//     the same against the suite's 9.0 data, allowing exactly the code points
//     whose category changed).
//   - "suite-defect": no implementation can pass the case as written.
//
// The analysis behind every entry is in docs/accepted-failures/xslt30.md.
// XSLT30_U90=1 runs every case anyway.
func unicodeExclusion(set string, tc xsltCase) string {
	if os.Getenv("XSLT30_U90") != "" {
		return ""
	}
	switch set {
	case "regex-classes":
		if regexClassesVersionCases[tc.Name] {
			return "unicode-version"
		}
	case "unicode-90":
		return unicode90Exclusion(tc)
	}
	return ""
}

// regexClassesVersionCases test a general category (\p{Lu}, \p{C}, …) against
// the characters of Unicode 3.1, 5.2 or 6.0. The block cases of the set run.
var regexClassesVersionCases = setOf(
	"regex-classes-001", "regex-classes-003", "regex-classes-004", "regex-classes-006",
	"regex-classes-007", "regex-classes-008", "regex-classes-009", "regex-classes-011",
	"regex-classes-012", "regex-classes-013", "regex-classes-014", "regex-classes-015",
	"regex-classes-016", "regex-classes-017", "regex-classes-018", "regex-classes-019",
	"regex-classes-020", "regex-classes-022", "regex-classes-023", "regex-classes-024",
	"regex-classes-026", "regex-classes-027", "regex-classes-028", "regex-classes-029",
	"regex-classes-030", "regex-classes-031", "regex-classes-032", "regex-classes-033",
	"regex-classes-036",
)

// unicode90ClassFreeModes are the modes of the generated unicode-90 cases that
// do not depend on which category is tested (string-length, translate, and
// tautologies like [\p{X}-[\p{X}]]). Every other mode tests \p{X} for the
// category X named by the stylesheet parameter charclass, which the catalog
// never passes: it defaults to Ll, so the case for Zs asks whether the Zs
// characters are Ll. Only the Ll cases therefore test what they mean to.
var unicode90ClassFreeModes = setOf(
	"fn-matches9", "fn-matches10", "fn-matches12", "fn-string-length1", "fn-string-length2",
	"fn-translate1", "fn-translate2", "fn-translate3", "fn-translate4", "fn-replace4",
)

// unicode90LlVersionCases run over every code point outside the 9.0 Ll data
// (or count all of them); 170 code points became Ll after Unicode 9.0, so the
// answer is the 9.0 database's. Not one 9.0 Ll character changed category, so
// the Ll cases that test only the 9.0 characters themselves run.
var unicode90LlVersionCases = setOf(
	"unicode90-Ll-005", "unicode90-Ll-006", "unicode90-Ll-007", "unicode90-Ll-008",
	"unicode90-Ll-014", "unicode90-Ll-015", "unicode90-Ll-017", "unicode90-Ll-018",
	"unicode90-Ll-020", "unicode90-Ll-022", "unicode90-Ll-024", "unicode90-Ll-032",
	"unicode90-Ll-036", "unicode90-Ll-037", "unicode90-Ll-038",
)

func unicode90Exclusion(tc xsltCase) string {
	name := tc.Name
	switch name {
	case "unicode90-001", "unicode90-002", "unicode90-003", "unicode90-004",
		"unicode90-005", "unicode90-006", "unicode90-007", "unicode90-008":
		// The stylesheet counts over all planes (… 65537 to 1114111); the
		// expected counts are totals over the BMP only (\d + \D = 63,454, the
		// figure in the stylesheet's own comment).
		return "suite-defect"
	case "unicode90-Ll-033", "unicode90-Ll-035":
		// Compare with string-join(*/c, ''), but the <c> elements are empty
		// (the character is in @c): always false.
		return "suite-defect"
	case "unicode90-C-025", "unicode90-Cn-025":
		// The data holds U+FFFE and U+FFFF, which XML cannot contain, as
		// c=""; their string-length cannot equal the element count.
		return "suite-defect"
	}
	if unicode90LlVersionCases[name] {
		return "unicode-version"
	}
	if strings.HasPrefix(name, "unicode90-Ll-") || !strings.HasPrefix(name, "unicode90-") || tc.Test.InitialMode == nil {
		return ""
	}
	if !unicode90ClassFreeModes[tc.Test.InitialMode.Name] {
		return "suite-defect" // charclass not passed: tests Ll, not its category
	}
	return ""
}

func setOf(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}
