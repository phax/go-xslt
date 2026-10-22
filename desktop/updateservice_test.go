package main

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.1.6", "0.1.5", true},
		{"v0.2.0", "0.1.9", true},
		{"0.1.5", "0.1.5", false},
		{"0.1.5", "0.1.6", false},
		{"0.1.6", "0.1.5-3-gabc1234", true},
		{"0.1.5", "0.1.5-3-gabc1234-dirty", false},
		{"0.1.10", "0.1.9", true},
		{"1.0", "0.9.9", true},
		{"0.1.6", "dev", false},
		{"garbage", "0.1.5", false},
	}
	for _, c := range cases {
		if got := isNewer(c.latest, c.current); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}
