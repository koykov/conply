package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
				require.NoError(t, err, "Get()")
				assert.Equal(t, tt.wantBody, string(body), "Get() body")
				return
			}

			require.Error(t, err, "Get()")
			assert.Nil(t, body, "Get() body")

			var cf *conply.CFChallengeError
			if tt.wantCFKind != "" {
				require.ErrorAs(t, err, &cf)
				assert.Equal(t, tt.wantCFKind, cf.Kind, "CFChallengeError.Kind")
				assert.Equal(t, http.StatusForbidden, cf.Status, "CFChallengeError.Status")
				return
			}

			// Generic error path: must NOT be reported as a CF challenge.
			require.False(t, errors.As(err, &cf), "error = %v; want generic error, got CFChallengeError", err)
			if tt.wantErrSub != "" {
				assert.Contains(t, err.Error(), tt.wantErrSub, "error")
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
	_, err := s.Get(srv.URL)
	require.Error(t, err, "first Get()")

	body, err := s.Get(srv.URL)
	require.NoError(t, err, "second Get()")
	assert.Equal(t, "ok", string(body), "second Get() body")
}

func TestUASessionName(t *testing.T) {
	s := newSession("")
	assert.Equal(t, "user-agent", s.Name(), "Name()")
}
