package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/koykov/conply"
)

// SessionProvider abstracts how xradio fetches station pages / API responses.
// Today the only implementation is the user-agent session; a browser-backed
// session can be added later without touching player logic.
type SessionProvider interface {
	Get(url string) ([]byte, error) // returns *conply.CFChallengeError on a Cloudflare challenge/block
	Name() string                   // e.g. "user-agent" — for logs/errors
}

// uaSession fetches content using browser-like request headers.
type uaSession struct {
	client *http.Client
}

func newSession(ua string) *uaSession {
	return &uaSession{client: conply.NewBrowserClient(ua)}
}

func (s *uaSession) Get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if cf := conply.DetectCF(resp, body); cf != nil {
		return nil, cf
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %d from %s: %s", resp.StatusCode, url, string(body))
	}

	return body, nil
}

func (s *uaSession) Name() string {
	return "user-agent"
}
