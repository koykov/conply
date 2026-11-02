package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/koykov/conply"
)

// Compile-time guarantee that the user-agent session satisfies the interface
// the rest of xradio depends on.
var _ SessionProvider = (*uaSession)(nil)

func TestUASessionGet(t *testing.T) {
	const okHTML = `<html><body>"audio_token":"bc082b98dbbc625717a5867f71e94027"</body></html>`

	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantBody   string
		wantErr    bool
		wantCFKind string // non-empty: expect *conply.CFChallengeError of this kind
		wantErrSub string // non-empty: expect error text to contain this
	}{
		{
			name: "2xx returns body and nil error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, okHTML)
			},
			wantBody: okHTML,
		},
		{
			name: "cloudflare challenge returns CFChallengeError",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("cf-mitigated", "challenge")
				w.Header().Set("Server", "cloudflare")
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, "<html>Just a moment...</html>")
			},
			wantErr:    true,
			wantCFKind: conply.CFKindChallenge,
		},
		{
			name: "non-2xx non-cf returns generic error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, "internal boom")
			},
			wantErr:    true,
			wantErrSub: "500",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			s := newSession("")
			body, err := s.Get(srv.URL)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Get() error = %v; want nil", err)
				}
				if string(body) != tt.wantBody {
					t.Errorf("Get() body = %q; want %q", string(body), tt.wantBody)
				}
				return
			}

			if err == nil {
				t.Fatal("Get() error = nil; want non-nil")
			}
			if body != nil {
				t.Errorf("Get() body = %q; want nil on error", string(body))
			}

			var cf *conply.CFChallengeError
			if tt.wantCFKind != "" {
				if !errors.As(err, &cf) {
					t.Fatalf("error = %v (%T); want *conply.CFChallengeError", err, err)
				}
				if cf.Kind != tt.wantCFKind {
					t.Errorf("CFChallengeError.Kind = %q; want %q", cf.Kind, tt.wantCFKind)
				}
				if cf.Status != http.StatusForbidden {
					t.Errorf("CFChallengeError.Status = %d; want %d", cf.Status, http.StatusForbidden)
				}
				return
			}

			// Generic error path: must NOT be reported as a CF challenge.
			if errors.As(err, &cf) {
				t.Fatalf("error = %v; want generic error, got CFChallengeError", err)
			}
			if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error = %q; want it to contain %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

// TestUASessionGetDrainsErrorBody verifies that a non-2xx response leaves the
// session usable: the error body is read and closed by Get, so no panic occurs
// and a following request on the same session still succeeds.
func TestUASessionGetDrainsErrorBody(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, strings.Repeat("x", 1024))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	s := newSession("")
	if body, err := s.Get(srv.URL); err == nil {
		t.Fatalf("first Get() error = nil (body %q); want non-nil", string(body))
	}

	body, err := s.Get(srv.URL)
	if err != nil {
		t.Fatalf("second Get() error = %v; want nil", err)
	}
	if string(body) != "ok" {
		t.Errorf("second Get() body = %q; want %q", string(body), "ok")
	}
}

func TestUASessionName(t *testing.T) {
	s := newSession("")
	if got := s.Name(); got != "user-agent" {
		t.Errorf("Name() = %q; want %q", got, "user-agent")
	}
}
