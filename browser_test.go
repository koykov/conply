package conply

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedHeaders is what the test server observed for a single request.
type capturedHeaders struct {
	userAgent      string
	accept         string
	acceptLanguage string
	acceptEncoding string
}

// newCapturingServer returns a server that reports, on the buffered channel,
// the headers it observed. The handler writes a 200 so the client can complete.
func newCapturingServer(t *testing.T) (*httptest.Server, <-chan capturedHeaders) {
	t.Helper()
	hits := make(chan capturedHeaders, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits <- capturedHeaders{
			userAgent:      r.Header.Get("User-Agent"),
			accept:         r.Header.Get("Accept"),
			acceptLanguage: r.Header.Get("Accept-Language"),
			acceptEncoding: r.Header.Get("Accept-Encoding"),
		}
		w.WriteHeader(http.StatusOK)
	}))
	return srv, hits
}

func TestNewBrowserClientInjectsHeaders(t *testing.T) {
	cases := []struct {
		name           string
		newUA          string
		requestUA      string // when non-empty, set on the request before Do
		wantUA         string
		wantAccept     string
		wantAcceptLang string
	}{
		{
			name:           "empty ua falls back to DefaultBrowserUA",
			newUA:          "",
			wantUA:         DefaultBrowserUA,
			wantAccept:     browserAccept,
			wantAcceptLang: browserAcceptLanguage,
		},
		{
			name:           "explicit ua is sent verbatim",
			newUA:          "MyUA/1.0",
			wantUA:         "MyUA/1.0",
			wantAccept:     browserAccept,
			wantAcceptLang: browserAcceptLanguage,
		},
		{
			name:           "request-scoped User-Agent is not overwritten",
			newUA:          DefaultBrowserUA,
			requestUA:      "Custom/7.0",
			wantUA:         "Custom/7.0",
			wantAccept:     browserAccept,
			wantAcceptLang: browserAcceptLanguage,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv, hits := newCapturingServer(t)
			defer srv.Close()

			client := NewBrowserClient(tt.newUA)
			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			require.NoError(t, err, "http.NewRequest")
			if tt.requestUA != "" {
				req.Header.Set("User-Agent", tt.requestUA)
			}

			resp, err := client.Do(req)
			require.NoError(t, err, "client.Do")
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()

			got := <-hits
			assert.Equal(t, tt.wantUA, got.userAgent, "User-Agent")
			assert.Equal(t, tt.wantAccept, got.accept, "Accept")
			assert.Equal(t, tt.wantAcceptLang, got.acceptLanguage, "Accept-Language")
		})
	}
}
