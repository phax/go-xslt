package fileuri

import "testing"

func TestFromPath(t *testing.T) {
	for _, c := range []struct {
		win        bool
		in, expect string
	}{
		{false, "/home/x/d", "file:///home/x/d"},
		{false, "/home/x/d/", "file:///home/x/d/"},
		{true, `C:\Users\x\d`, "file:///C:/Users/x/d"},
		{true, "C:/Users/x/", "file:///C:/Users/x/"},
		{true, `\\srv\share\d`, "file://srv/share/d"},
		{true, "/already/rooted", "file:///already/rooted"},
	} {
		if got := fromPath(c.in, c.win); got != c.expect {
			t.Errorf("fromPath(%q, win=%v) = %q, want %q", c.in, c.win, got, c.expect)
		}
	}
}

func TestToPath(t *testing.T) {
	for _, c := range []struct {
		win        bool
		in, expect string
	}{
		{false, "file:///home/x/d.xml", "/home/x/d.xml"},
		{false, "file:///home/x/a%20b.xml", "/home/x/a b.xml"},
		{true, "file:///C:/Users/x/d.xml", `C:\Users\x\d.xml`},
		{true, "file://C:/Users/x/d.xml", `C:\Users\x\d.xml`}, // the old, malformed form
		{true, "file://srv/share/d.xml", `\\srv\share\d.xml`},
		{true, "file://localhost/C:/d.xml", `C:\d.xml`},
	} {
		got, ok := toPath(c.in, c.win)
		if !ok || got != c.expect {
			t.Errorf("toPath(%q, win=%v) = %q,%v, want %q", c.in, c.win, got, ok, c.expect)
		}
	}
	if _, ok := toPath("http://x/y", false); ok {
		t.Error("an http URI is not a file path")
	}
}

func TestStripFile(t *testing.T) {
	for _, c := range []struct {
		win        bool
		in, expect string
	}{
		{false, "file:///home/x/d.xml", "/home/x/d.xml"},
		{false, "/plain/path", "/plain/path"},
		{false, "file:///a%20b", "/a%20b"}, // not decoded: the string came from a path
		{true, "file:///C:/Users/x/d.xml", `C:\Users\x\d.xml`},
		{true, "file://C:/Users/x/d.xml", `C:\Users\x\d.xml`},
		{true, "file://srv/share/d.xml", `\\srv\share\d.xml`},
		{true, `C:\already\a\path`, `C:\already\a\path`},
	} {
		if got := stripFile(c.in, c.win); got != c.expect {
			t.Errorf("stripFile(%q, win=%v) = %q, want %q", c.in, c.win, got, c.expect)
		}
	}
}
