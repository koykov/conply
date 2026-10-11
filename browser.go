package conply

import (
	"net/http"
	"time"
)

// DefaultBrowserUA is a realistic desktop browser User-Agent string. Sites
// protected by a user-agent based Cloudflare Managed Challenge (for example
// www.rockradio.com) serve a 403 challenge to bare Go clients and a normal
// page to a browser-like client.
const DefaultBrowserUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// BrowserUserAgents is a pool of realistic browser User-Agents. Some AudioAddict
// guest tokens are UA-derived, and a given UA can be served an empty tracklist;
// rotating within this pool is the recovery strategy.
var BrowserUserAgents = []string{
	DefaultBrowserUA,
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
}

const (
	browserAccept         = "text/html,application/xhtml+xml,application/xml;q=0.9,application/json;q=0.8,*/*;q=0.7"
	browserAcceptLanguage = "en-US,en;q=0.9"
)

// browserTransport injects browser-like headers into every outgoing request
// unless the caller has already set the header explicitly.
type browserTransport struct {
	rt http.RoundTripper
	ua string
}

func (t *browserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.ua)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", browserAccept)
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", browserAcceptLanguage)
	}
	return t.rt.RoundTrip(req)
}

// NewBrowserClient returns an *http.Client that mimics a browser well enough to
// pass a user-agent based Cloudflare Managed Challenge. An empty ua falls back
// to DefaultBrowserUA. Accept-Encoding is deliberately left untouched so that
// Go's transparent gzip handling keeps working.
func NewBrowserClient(ua string) *http.Client {
	if ua == "" {
		ua = DefaultBrowserUA
	}
	var transport = http.DefaultTransport
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = t.Clone()
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &browserTransport{rt: transport, ua: ua},
	}
}
