package fsaccess

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGuard(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	in := filepath.Join(root, "sub", "a.xml")
	if err := os.MkdirAll(filepath.Dir(in), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in, []byte("<a/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}

	g := New(root)
	cases := map[string]bool{
		in:                               true,
		filepath.Join(root, "missing"):   true, // judged by its parent; the read then fails as "not found"
		filepath.Join(root, "sub", ".."): true,
		root:                             true,
		secret:                           false,
		link:                             false, // a link inside the root pointing out of it
		filepath.Join(root, "..", filepath.Base(outside), "secret.txt"): false,
		filepath.Join(root+"x", "a"):                                    false, // a sibling sharing the prefix
	}
	for p, want := range cases {
		if got := g.Allowed(p); got != want {
			t.Errorf("Allowed(%s) = %v, want %v", p, got, want)
		}
	}
	if _, err := g.ReadFile(secret); !errors.Is(err, ErrOutsideRoots) {
		t.Errorf("ReadFile outside: got %v", err)
	}
	if b, err := g.ReadFile(in); err != nil || string(b) != "<a/>" {
		t.Errorf("ReadFile inside: %q %v", b, err)
	}
	if New().Allowed(in) {
		t.Error("a guard with no roots must refuse everything")
	}
	var none *Guard
	if !none.Allowed(secret) {
		t.Error("a nil guard allows everything")
	}
}
