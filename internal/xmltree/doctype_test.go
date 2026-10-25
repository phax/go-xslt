package xmltree

import (
	"strings"
	"testing"
)

var doctypeCases = []string{"", "!", "<!", "<!DOCTYPE", "!DOCTYPE", "<<!DOCTYPE", "<!--x--><!DOCTYPE a>", "<!DOCTYP",
	"x<!DOCTYPEy<!DOCTYPE", "<a>!</a><!DOCTYPE", "!!<!!DOCTYPE<!DOCTYPE", "<a/>", "<!DOCTYPE<!DOCTYPE"}

func TestIndexDoctype(t *testing.T) {
	for _, s := range doctypeCases {
		if got, want := indexDoctype(s), strings.Index(s, "<!DOCTYPE"); got != want {
			t.Errorf("indexDoctype(%q) = %d, want %d", s, got, want)
		}
	}
}

func FuzzIndexDoctype(f *testing.F) {
	for _, s := range doctypeCases {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := indexDoctype(s), strings.Index(s, "<!DOCTYPE"); got != want {
			t.Fatalf("indexDoctype(%q) = %d, want %d", s, got, want)
		}
	})
}
