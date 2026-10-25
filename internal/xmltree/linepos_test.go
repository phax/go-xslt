package xmltree

import (
	"math/rand"
	"os"
	"sort"
	"testing"
)

// atRef is linePos.at as it was, one byte at a time.
func (p *linePos) atRef(off int) (int, int) {
	if off > len(p.src) {
		off = len(p.src)
	}
	for p.scanned < off {
		if p.src[p.scanned] == '\n' {
			p.line++
			p.lineStart = p.scanned + 1
		}
		p.scanned++
	}
	return p.line, off - p.lineStart + 1
}

func TestLinePosMatchesReference(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for trial := 0; trial < 2000; trial++ {
		b := make([]byte, r.Intn(200))
		for i := range b {
			b[i] = "ab\n\r<"[r.Intn(5)]
		}
		src := string(b)
		offs := make([]int, r.Intn(30))
		for i := range offs {
			offs[i] = r.Intn(len(src)+5) - 2
		}
		sort.Ints(offs)
		p, q := newLinePos(src), newLinePos(src)
		for _, o := range offs {
			l1, c1 := p.at(o)
			l2, c2 := q.atRef(o)
			if l1 != l2 || c1 != c2 {
				t.Fatalf("%q at %d: (%d,%d), want (%d,%d)", src, o, l1, c1, l2, c2)
			}
		}
	}
}

func BenchmarkLinePosAt(b *testing.B) {
	data, err := os.ReadFile(os.Getenv("BENCH_XML"))
	if err != nil {
		b.Skip("set BENCH_XML to a large XML file")
	}
	src := string(data)
	var offs []int
	for i := 0; i < len(src); i++ {
		if src[i] == '<' {
			offs = append(offs, i)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := newLinePos(src)
		for _, o := range offs {
			p.at(o)
		}
	}
}
