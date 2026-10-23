package licenses

import "testing"

// Every entry names an embedded license file, and no dependency is listed twice.
func TestEntriesHaveTheirLicense(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.Name] {
			t.Errorf("%s is listed twice", e.Name)
		}
		seen[e.Name] = true
		b, err := files.ReadFile(e.path)
		if err != nil || len(b) == 0 {
			t.Errorf("%s: license file %s missing or empty (%v)", e.Name, e.path, err)
		}
		if e.License == "" || e.Kind == "" {
			t.Errorf("%s: License and Kind are required", e.Name)
		}
	}
}
