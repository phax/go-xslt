// Package fsaccess decides which local files a run may read.
//
// A stylesheet, an XPath expression, a schema and a source document can all
// name files: xsl:import, document(), fn:unparsed-text, xs:import, a DOCTYPE's
// external subset. When those inputs are not trusted, an unrestricted reader
// would let them read anything the process can (keys, configuration, other
// users' data) and copy it into the output. A Guard confines every such read
// to a set of root directories.
//
// A nil *Guard allows everything. That is the behaviour of the internal
// packages when no guard is handed in (the conformance harnesses, the
// package tests); the public engine always hands one in, restricted to
// BaseDir unless the caller says otherwise.
package fsaccess

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrOutsideRoots reports a read the guard refuses.
var ErrOutsideRoots = errors.New("file access outside the allowed directories")

// Guard confines file reads to its root directories.
type Guard struct {
	roots []string // absolute, cleaned, symlinks resolved
}

// New returns a Guard that allows reads under the given directories (and
// their subdirectories). Empty entries are ignored; with no usable directory
// at all, every read is refused.
func New(dirs ...string) *Guard {
	g := &Guard{}
	for _, d := range dirs {
		if strings.TrimSpace(d) == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			continue
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		g.roots = append(g.roots, filepath.Clean(abs))
	}
	return g
}

// Check reports whether path may be read: nil when it may, an error wrapping
// ErrOutsideRoots when it may not. The path's symlinks are resolved first, so a
// link inside a root that points outside it is refused. A path that does not
// exist is judged by its nearest existing ancestor, so the caller gets its
// ordinary "not found" rather than a refusal for a missing file inside a root.
func (g *Guard) Check(path string) error {
	if g == nil {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrOutsideRoots, path)
	}
	real := resolveExisting(filepath.Clean(abs))
	for _, r := range g.roots {
		if within(real, r) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrOutsideRoots, path)
}

// Allowed is Check as a boolean.
func (g *Guard) Allowed(path string) bool { return g.Check(path) == nil }

// ReadFile is os.ReadFile behind Check.
func (g *Guard) ReadFile(path string) ([]byte, error) {
	if err := g.Check(path); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// Open is os.Open behind Check.
func (g *Guard) Open(path string) (io.ReadCloser, error) {
	if err := g.Check(path); err != nil {
		return nil, err
	}
	return os.Open(path)
}

// ReadDir is os.ReadDir behind Check.
func (g *Guard) ReadDir(path string) ([]fs.DirEntry, error) {
	if err := g.Check(path); err != nil {
		return nil, err
	}
	return os.ReadDir(path)
}

// Stat is os.Stat behind Check.
func (g *Guard) Stat(path string) (fs.FileInfo, error) {
	if err := g.Check(path); err != nil {
		return nil, err
	}
	return os.Stat(path)
}

// resolveExisting resolves the symlinks of the longest existing prefix of p and
// appends the rest unchanged.
func resolveExisting(p string) string {
	rest := ""
	cur := p
	for {
		if real, err := filepath.EvalSymlinks(cur); err == nil {
			if rest == "" {
				return real
			}
			return filepath.Join(real, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p
		}
		if rest == "" {
			rest = filepath.Base(cur)
		} else {
			rest = filepath.Join(filepath.Base(cur), rest)
		}
		cur = parent
	}
}

func within(p, root string) bool {
	if p == root {
		return true
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
