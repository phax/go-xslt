package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// releasesURL is GitHub's "latest release" endpoint for this project. It
// excludes drafts and pre-releases.
const releasesURL = "https://api.github.com/repos/tim-riep/go-xslt/releases/latest"

// UpdateService asks GitHub whether a newer release exists. It is the app's
// only network access, and it runs only when the user leaves the update check
// switched on (the frontend decides when to call it).
type UpdateService struct{}

// UpdateInfo is the outcome of CheckForUpdate.
type UpdateInfo struct {
	// Current is this build's version ("dev" for a development build).
	Current string `json:"current"`
	// Latest is the newest release's version, without a leading "v".
	Latest string `json:"latest"`
	// URL is the newest release's page.
	URL string `json:"url"`
	// Newer reports that Latest is newer than Current. Always false for a
	// development build, which has no version to compare.
	Newer bool `json:"newer"`
	// Error says why the check failed; empty on success.
	Error string `json:"error,omitempty"`
}

// AppVersion returns this build's version.
func (s *UpdateService) AppVersion() string { return appVersion }

// CheckForUpdate fetches the latest release from GitHub and compares it with
// this build. A failure (offline, rate limit) is reported in Error, never as
// a Go error, so the frontend can ignore it quietly.
func (s *UpdateService) CheckForUpdate() UpdateInfo {
	info := UpdateInfo{Current: appVersion}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "go-xslt-desktop/"+appVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		info.Error = fmt.Sprintf("GitHub answered %s", resp.Status)
		return info
	}
	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		info.Error = err.Error()
		return info
	}
	info.Latest = strings.TrimPrefix(rel.TagName, "v")
	info.URL = rel.HTMLURL
	info.Newer = isNewer(info.Latest, appVersion)
	return info
}

// isNewer reports whether release version latest is newer than current. Both
// are dotted numbers, optionally with a "v" prefix; current may carry git
// describe's "-N-gHASH[-dirty]" suffix of a build between tags, which counts as
// its tag (so the next tag is newer). A development build ("dev") or a version
// that does not parse is never older than anything.
func isNewer(latest, current string) bool {
	l, ok1 := parseVersion(latest)
	c, ok2 := parseVersion(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < len(l) || i < len(c); i++ {
		var a, b int
		if i < len(l) {
			a = l[i]
		}
		if i < len(c) {
			b = c[i]
		}
		if a != b {
			return a > b
		}
	}
	return false
}

func parseVersion(v string) ([]int, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return nil, false
	}
	var out []int
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}
