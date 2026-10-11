// Package fileuri converts between filesystem paths and "file:" URIs.
//
// POSIX paths already begin with "/", so "file://" + path is a valid URI; a
// Windows path does not ("C:\dir"), and the same concatenation yields
// "file://C:/dir", in which net/url reads "C:" as the HOST and loses the drive.
// Every place that builds or consumes a file URI goes through here instead.
// The rules are pure string functions parameterised by the platform, so both
// flavours are tested on any OS; the exported wrappers pick the running one.
package fileuri

import (
	"net/url"
	"runtime"
	"strings"
)

const scheme = "file://"

// FromPath returns the "file:" URI of an absolute filesystem path.
func FromPath(p string) string { return fromPath(p, runtime.GOOS == "windows") }

// ToPath returns the filesystem path a "file:" URI denotes (percent-escapes
// decoded). ok is false when uri is not a parseable file URI with a path.
func ToPath(uri string) (path string, ok bool) { return toPath(uri, runtime.GOOS == "windows") }

// StripFile removes a leading "file://" from a URI that was built from a path,
// leaving an OS path — without decoding anything, so a path containing "%"
// survives. Anything that is not a file URI is returned unchanged.
func StripFile(uri string) string { return stripFile(uri, runtime.GOOS == "windows") }

func fromPath(p string, windows bool) string {
	if !windows {
		return scheme + p
	}
	s := strings.ReplaceAll(p, `\`, "/")
	switch {
	case strings.HasPrefix(s, "//"): // UNC: \\host\share\d -> file://host/share/d
		return "file:" + s
	case strings.HasPrefix(s, "/"):
		return scheme + s
	}
	return scheme + "/" + s // drive path: C:/d -> file:///C:/d
}

func isDrive(s string) bool {
	return len(s) >= 2 && s[1] == ':' && (s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z')
}

func toPath(uri string, windows bool) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	if !windows {
		return u.Path, u.Path != ""
	}
	p := u.Path
	switch {
	case isDrive(u.Host) && len(u.Host) == 2:
		// The malformed "file://C:/d" older versions wrote: the drive is in the host.
		p = u.Host + p
	case u.Host != "" && u.Host != "localhost":
		return `\\` + u.Host + backslash(p), true // UNC
	case len(p) >= 3 && p[0] == '/' && isDrive(p[1:]):
		p = p[1:] // "/C:/d" -> "C:/d"
	}
	return backslash(p), p != ""
}

func backslash(p string) string { return strings.ReplaceAll(p, "/", `\`) }

func stripFile(uri string, windows bool) string {
	rest, ok := strings.CutPrefix(uri, scheme)
	if !ok {
		return uri
	}
	if !windows {
		return rest
	}
	switch {
	case len(rest) >= 3 && rest[0] == '/' && isDrive(rest[1:]):
		rest = rest[1:]
	case isDrive(rest):
	case strings.HasPrefix(rest, "/"):
	default:
		rest = "//" + rest // host/share/d
	}
	return strings.ReplaceAll(rest, "/", `\`)
}
