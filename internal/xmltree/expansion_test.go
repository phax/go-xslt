package xmltree

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// allocated reports the bytes f allocates.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// Entity expansion is refused BEFORE it builds anything past expansionLimit:
// the error alone is not enough, a check made after building the value let
// a 699-byte document allocate gigabytes on the way to it.
func TestEntityExpansionAllocationIsBounded(t *testing.T) {
	chain := `<!ENTITY a "` + strings.Repeat("a", 100) + `">` +
		`<!ENTITY b "` + strings.Repeat("&a;", 10) + `">` +
		`<!ENTITY c "` + strings.Repeat("&b;", 10) + `">` +
		`<!ENTITY d "` + strings.Repeat("&c;", 10) + `">` +
		`<!ENTITY e "` + strings.Repeat("&d;", 10) + `">` +
		`<!ENTITY f "` + strings.Repeat("&e;", 10) + `">` // f: 10 MB, under the limit
	var params strings.Builder
	params.WriteString(`<!ENTITY % p0 "` + strings.Repeat("x", 100) + `">`)
	for i := 1; i <= 4; i++ {
		fmt.Fprintf(&params, `<!ENTITY %% p%d "%s">`, i, strings.Repeat(fmt.Sprintf("%%p%d;", i-1), 10))
	}
	params.WriteString(`<!ENTITY g "` + strings.Repeat("%p4;", 2000) + `">`)
	for _, c := range []struct{ name, src string }{
		{"one value, 100 references to a 10 MB entity", `<!DOCTYPE r [` + chain + `<!ENTITY z "` + strings.Repeat("&f;", 100) + `">]><r>&z;</r>`},
		{"one value, 1000 references to a 10 MB entity", `<!DOCTYPE r [` + chain + `<!ENTITY z "` + strings.Repeat("&f;", 1000) + `">]><r>&z;</r>`},
		{"parameter entities", `<!DOCTYPE r [` + params.String() + `]><r>&g;</r>`},
	} {
		var err error
		n := allocated(func() { _, err = Parse(c.src) })
		if err == nil || !strings.Contains(err.Error(), "entity expansion exceeds") {
			t.Errorf("%s: want an entity-expansion error, got %v", c.name, err)
		}
		// The limit is 16 MiB; building values up to it, with a builder's
		// doubling, stays far below this.
		if n > 256<<20 {
			t.Errorf("%s: allocated %d MB before refusing", c.name, n>>20)
		}
	}
}

// The streaming reader leaves a document with a long entity value to the
// full parser, which bounds the expansion before building it.
func TestStreamLeavesLongEntitiesToTheParser(t *testing.T) {
	_, err := NewStreamReader(strings.NewReader(`<!DOCTYPE r [<!ENTITY e "` + strings.Repeat("x", 25) + `">]><r>&e;</r>`))
	if !errors.Is(err, ErrStreamUnsupported) {
		t.Errorf("long entity value: want ErrStreamUnsupported, got %v", err)
	}
	if _, err := NewStreamReader(strings.NewReader(`<!DOCTYPE r [<!ENTITY nbsp "&#160;">]><r>&nbsp;</r>`)); err != nil {
		t.Errorf("short entity value: %v", err)
	}
}
