package xslt

import "testing"

// TestPackageVersionRanges pins the package-version-ranges semantics the
// xslt30-test use-package-2xx cases define: "2.0" and "2.0.0" are the SAME
// version, a pre-release suffix sorts BELOW the release it qualifies, two
// suffixes compare lexically, and the four range forms select accordingly.
func TestPackageVersionRanges(t *testing.T) {
	// The versions use-package-env-004 makes available.
	avail := []string{"1.0.0", "2.0.0", "2.0.0-alpha", "2.0.0-beta", "3.5.4"}

	match := func(spec string) []string {
		rs, err := parsePkgRanges(spec)
		if err != nil {
			t.Fatalf("parsePkgRanges(%q): %v", spec, err)
		}
		var out []string
		for _, v := range avail {
			pv, ok := parsePkgVersion(v)
			if !ok {
				t.Fatalf("parsePkgVersion(%q) failed", v)
			}
			if matchesAnyPkgRange(rs, pv) {
				out = append(out, v)
			}
		}
		return out
	}
	eq := func(spec string, want ...string) {
		t.Helper()
		got := match(spec)
		if len(got) != len(want) {
			t.Errorf("%q selected %v, want %v", spec, got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q selected %v, want %v", spec, got, want)
				return
			}
		}
	}

	eq("1.0.0", "1.0.0")                                           // use-package-201
	eq("2.0.0", "2.0.0")                                           // -202: the release only, not its pre-releases
	eq("2.0", "2.0.0")                                             // -212: missing decimal parts are zero
	eq("1.0.0, 2.0", "1.0.0", "2.0.0")                             // -203
	eq("1.5 to 2.5", "2.0.0", "2.0.0-alpha", "2.0.0-beta")         // -204
	eq("to 1.5", "1.0.0")                                          // -205
	eq("1.5+", "2.0.0", "2.0.0-alpha", "2.0.0-beta", "3.5.4")      // -206
	eq("3.5.*", "3.5.4")                                           // -207
	eq("2.0.0-alpha", "2.0.0-alpha")                               // -208
	eq("2.0.0-arable-environment.27 to 2.0.0-gamma", "2.0.0-beta") // -209: "alpha" < "arable..." lexically
	eq("2.0.0-a to 2.0.0-gamma", "2.0.0-alpha", "2.0.0-beta")      // -210
	eq("2.7.0-a to 5.0.0-gamma", "3.5.4")                          // -211
	eq("*", avail...)

	// A release outranks its own pre-releases (so "highest wins" picks 2.0.0).
	a, _ := parsePkgVersion("2.0.0")
	b, _ := parsePkgVersion("2.0.0-beta")
	if comparePkgVersion(a, b) <= 0 {
		t.Error("2.0.0 must rank above 2.0.0-beta")
	}

	// Invalid range forms are rejected (use-package-291..294 expect XTSE0020).
	for _, bad := range []string{"2.0.0-alpha:beta", "TotallyInvalid", "-3.6", "-alpha", ""} {
		if _, err := parsePkgRanges(bad); err == nil {
			t.Errorf("parsePkgRanges(%q) should be rejected", bad)
		}
	}
}

// A VersionPrefix may be the upper bound of a range (XSLT 3.0 §3.5.1:
// VersionTo ::= "to" S (PackageVersion | VersionPrefix), and the same for
// VersionFromTo); the cases are the REC's own examples (issue #8).
func TestPackageVersionRangeToPrefix(t *testing.T) {
	check := func(spec string, yes, no []string) {
		t.Helper()
		rs, err := parsePkgRanges(spec)
		if err != nil {
			t.Fatalf("parsePkgRanges(%q): %v", spec, err)
		}
		for _, list := range [][]string{yes, no} {
			for _, s := range list {
				v, ok := parsePkgVersion(s)
				if !ok {
					t.Fatalf("parsePkgVersion(%q) failed", s)
				}
				if got, want := matchesAnyPkgRange(rs, v), &list[0] == &yes[0]; got != want {
					t.Errorf("%q matches %q = %v, want %v", spec, s, got, want)
				}
			}
		}
	}
	// "to 3.3.* matches 1.5 or 2.0.6 or 3.3.4621, but not 3.4.0 or 3.4.0-beta"
	check("to 3.3.*", []string{"1.5", "2.0.6", "3.3.4621", "3.3", "3.3-beta"}, []string{"3.4.0", "3.4.0-beta"})
	// "1 to 5 matches 1.1, 2.1, 3.1, or 5.0 (but not 5.1), while 1 to 5.*
	// matches all of these, plus versions such as 5.7.2 (but not 6.0 or 6.0-beta)"
	check("1 to 5", []string{"1.1", "2.1", "3.1", "5.0"}, []string{"5.1", "5.7.2"})
	check("1 to 5.*", []string{"1.1", "2.1", "3.1", "5.0", "5.1", "5.7.2"}, []string{"6.0", "6.0-beta", "0.9"})
}
