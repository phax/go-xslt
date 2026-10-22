// Package netfetch retrieves http and https resources for fn:doc, document(),
// fn:unparsed-text and friends, under a policy the host sets.
//
// Network access is OFF unless the host turns it on (Policy.AllowNetwork): a
// stylesheet or expression is often untrusted input, and letting it make
// requests would let it reach internal services or leak data in a URL. When it
// is on, AllowedHosts can still narrow it to a list of host names, and every
// redirect is checked against the same rules as the original request.
//
// A Fetcher belongs to one run and remembers every response it handed out, so
// two reads of the same URI see the same resource: F&O 3.1 requires fn:doc and
// fn:unparsed-text to be stable within an execution scope
// (fn-unparsed-text-054a reads one URI fifty times and compares).
package netfetch

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Policy says whether, and where, a run may retrieve resources over the
// network. The zero value allows nothing.
type Policy struct {
	// AllowNetwork permits http and https retrieval at all.
	AllowNetwork bool
	// AllowedHosts, when non-empty, limits retrieval to these host names
	// (compared case-insensitively, without the port). A leading "." matches
	// the domain and every subdomain: ".example.com".
	AllowedHosts []string
	// Timeout bounds one request, response body included; 0 means
	// DefaultTimeout.
	Timeout time.Duration
	// MaxBytes bounds one response body; 0 means DefaultMaxBytes.
	MaxBytes int64
}

// Defaults for the Policy fields left zero.
const (
	DefaultTimeout  = 30 * time.Second
	DefaultMaxBytes = 64 << 20
	maxRedirects    = 10
)

// ErrNotAllowed reports a request the policy forbids.
var ErrNotAllowed = errors.New("network access is not allowed")

// IsNetworkURI reports whether uri names an http or https resource.
func IsNetworkURI(uri string) bool {
	u := strings.ToLower(strings.TrimSpace(uri))
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// Response is one retrieved resource.
type Response struct {
	Body []byte
	// Charset is the charset parameter of the Content-Type header, "" when
	// the server sent none.
	Charset string
	// MediaType is the Content-Type without parameters.
	MediaType string
	// URL is where the body actually came from, after redirects.
	URL string
}

type result struct {
	resp *Response
	err  error
}

// Fetcher retrieves resources under one Policy and caches what it retrieved.
// It is safe for concurrent use.
type Fetcher struct {
	policy Policy
	client *http.Client

	mu    sync.Mutex
	cache map[string]result
}

// New returns a Fetcher for p, or nil when p allows no network access, so a
// caller can test the result for nil instead of re-reading the policy.
func New(p Policy) *Fetcher {
	if !p.AllowNetwork {
		return nil
	}
	if p.Timeout <= 0 {
		p.Timeout = DefaultTimeout
	}
	if p.MaxBytes <= 0 {
		p.MaxBytes = DefaultMaxBytes
	}
	f := &Fetcher{policy: p, cache: map[string]result{}}
	f.client = &http.Client{
		Timeout: p.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return f.check(req.URL)
		},
	}
	return f
}

// Policy returns the policy f was made with (the zero Policy for a nil f).
func (f *Fetcher) Policy() Policy {
	if f == nil {
		return Policy{}
	}
	return f.policy
}

// check applies the policy to one URL.
func (f *Fetcher) check(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q", ErrNotAllowed, u.Scheme)
	}
	if u.User != nil {
		// Credentials in a URL would be sent to the server and may end up in
		// its logs; a stylesheet has no business carrying them.
		return fmt.Errorf("%w: user information in the URL", ErrNotAllowed)
	}
	if len(f.policy.AllowedHosts) == 0 {
		return nil
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range f.policy.AllowedHosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if strings.HasPrefix(h, ".") {
			if host == h[1:] || strings.HasSuffix(host, h) {
				return nil
			}
			continue
		}
		if host == h {
			return nil
		}
	}
	return fmt.Errorf("%w: host %q is not in the allowed list", ErrNotAllowed, host)
}

// Fetch retrieves uri (its fragment identifier ignored). A nil Fetcher allows
// nothing. The first answer for a URI, success or failure, is the answer for
// the rest of the run.
func (f *Fetcher) Fetch(uri string) (*Response, error) {
	if f == nil {
		return nil, ErrNotAllowed
	}
	u, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return nil, err
	}
	u.Fragment = ""
	key := u.String()
	f.mu.Lock()
	if r, ok := f.cache[key]; ok {
		f.mu.Unlock()
		return r.resp, r.err
	}
	f.mu.Unlock()

	resp, err := f.get(u)

	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.cache[key]; ok {
		// Another goroutine got there first; its answer is the stable one.
		return r.resp, r.err
	}
	f.cache[key] = result{resp, err}
	return resp, err
}

func (f *Fetcher) get(u *url.URL) (*Response, error) {
	if err := f.check(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, fmt.Errorf("timed out retrieving %s", u)
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("retrieving %s: %s", u, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, f.policy.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > f.policy.MaxBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", u, f.policy.MaxBytes)
	}
	out := &Response{Body: body, URL: resp.Request.URL.String()}
	if mt, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil {
		out.MediaType = mt
		out.Charset = params["charset"]
	}
	return out, nil
}
