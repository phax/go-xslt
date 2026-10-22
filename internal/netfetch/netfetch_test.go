package netfetch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestZeroPolicyAllowsNothing(t *testing.T) {
	if f := New(Policy{}); f != nil {
		t.Fatal("New(Policy{}) must return nil")
	}
	var f *Fetcher
	if _, err := f.Fetch("http://example.com/"); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("nil Fetcher: got %v, want ErrNotAllowed", err)
	}
}

func TestFetchCachesFirstAnswer(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte{'v', byte('0' + n)})
	}))
	defer srv.Close()
	f := New(Policy{AllowNetwork: true})
	a, err := f.Fetch(srv.URL + "/x#frag")
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.Fetch(srv.URL + "/x")
	if err != nil {
		t.Fatal(err)
	}
	if string(a.Body) != "v1" || string(b.Body) != "v1" || hits.Load() != 1 {
		t.Fatalf("got %q/%q after %d requests, want one request answering v1", a.Body, b.Body, hits.Load())
	}
	if a.Charset != "ISO-8859-1" || a.MediaType != "text/plain" {
		t.Fatalf("content type: got %q %q", a.MediaType, a.Charset)
	}
}

func TestAllowedHosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()
	if _, err := New(Policy{AllowNetwork: true, AllowedHosts: []string{"127.0.0.1"}}).Fetch(srv.URL); err != nil {
		t.Fatalf("listed host refused: %v", err)
	}
	if _, err := New(Policy{AllowNetwork: true, AllowedHosts: []string{"example.com"}}).Fetch(srv.URL); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("unlisted host: got %v, want ErrNotAllowed", err)
	}
	f := New(Policy{AllowNetwork: true, AllowedHosts: []string{".example.com"}})
	for host, want := range map[string]bool{"example.com": true, "a.example.com": true, "badexample.com": false} {
		u, _ := url.Parse("http://" + host + "/")
		if got := f.check(u) == nil; got != want {
			t.Errorf("%s: allowed=%v, want %v", host, got, want)
		}
	}
}

func TestRedirectIsChecked(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("secret")) }))
	defer target.Close()
	// localhost and 127.0.0.1 reach the same server under different names,
	// so a list naming only localhost must refuse the redirect to 127.0.0.1.
	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer front.Close()
	frontURL := strings.Replace(front.URL, "127.0.0.1", "localhost", 1)
	f := New(Policy{AllowNetwork: true, AllowedHosts: []string{"localhost"}})
	if _, err := f.Fetch(frontURL); err == nil || !strings.Contains(err.Error(), "not in the allowed list") {
		t.Fatalf("redirect to an unlisted host: got %v", err)
	}
}

func TestRefusals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer srv.Close()
	f := New(Policy{AllowNetwork: true, MaxBytes: 10})
	if _, err := f.Fetch(srv.URL + "/big"); err == nil {
		t.Error("a body over MaxBytes must be refused")
	}
	if _, err := f.Fetch(srv.URL + "/missing"); err == nil {
		t.Error("a 404 must be an error")
	}
	if _, err := f.Fetch(strings.Replace(srv.URL, "http://", "http://user:pw@", 1)); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("credentials in the URL: got %v, want ErrNotAllowed", err)
	}
	if _, err := f.Fetch("file:///etc/passwd"); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("file scheme: got %v, want ErrNotAllowed", err)
	}
}
