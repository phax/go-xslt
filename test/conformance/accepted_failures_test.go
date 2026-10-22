package conformance

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var reBackticked = regexp.MustCompile("`([^`\n]+)`")

// checkAcceptedFailures fails the test when a case fails that is not named in
// docs/accepted-failures/<doc>. Every failure must either be fixed or be
// written down there with its reason; see docs/accepted-failures/README.md.
func checkAcceptedFailures(t *testing.T, doc string, fails []string) {
	t.Helper()
	path := filepath.Join(repoRoot(), "docs", "accepted-failures", doc)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("cannot read %s: %v", path, err)
		return
	}
	documented := map[string]bool{}
	for _, m := range reBackticked.FindAllStringSubmatch(string(b), -1) {
		documented[m[1]] = true
	}
	seen := map[string]bool{}
	var missing []string
	for _, f := range fails {
		if !documented[f] && !seen[f] {
			seen[f] = true
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("undocumented failures: fix them, or add them with a reason to docs/accepted-failures/%s: %s",
			doc, strings.Join(missing, ", "))
	}
}
