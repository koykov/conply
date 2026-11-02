package conply

import (
	"net/http"
	"testing"
)

const cfTestURL = "https://example.test/station"

// makeCFResponse builds an *http.Response without touching the network. The
// request URL is attached so DetectCF can populate CFChallengeError.URL.
func makeCFResponse(status int, header http.Header, rawURL string) *http.Response {
	var req *http.Request
	if rawURL != "" {
		req, _ = http.NewRequest(http.MethodGet, rawURL, nil)
	}
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Request:    req,
	}
}

func TestDetectCF(t *testing.T) {
	cases := []struct {
		name       string
		resp       *http.Response
		body       []byte
		wantNil    bool
		wantKind   string
		wantStatus int
		wantURL    string
	}{
		{
			name: "403 with cf-mitigated challenge header is a challenge",
			resp: func() *http.Response {
				h := http.Header{}
				h.Set("cf-mitigated", "challenge")
				return makeCFResponse(http.StatusForbidden, h, cfTestURL)
			}(),
			body:       []byte("<html>verify</html>"),
			wantKind:   CFKindChallenge,
			wantStatus: http.StatusForbidden,
			wantURL:    cfTestURL,
		},
		{
			name: "403 with Server cloudflare and Just a moment body is a challenge",
			resp: func() *http.Response {
				h := http.Header{}
				h.Set("Server", "cloudflare")
				return makeCFResponse(http.StatusForbidden, h, cfTestURL)
			}(),
			body:       []byte("<html><body>Just a moment...</body></html>"),
			wantKind:   CFKindChallenge,
			wantStatus: http.StatusForbidden,
			wantURL:    cfTestURL,
		},
		{
			name: "403 with cf-mitigated block header is a block",
			resp: func() *http.Response {
				h := http.Header{}
				h.Set("cf-mitigated", "block")
				return makeCFResponse(http.StatusForbidden, h, cfTestURL)
			}(),
			body:       []byte("<html>you have been blocked</html>"),
			wantKind:   CFKindBlock,
			wantStatus: http.StatusForbidden,
			wantURL:    cfTestURL,
		},
		{
			name:       "403 with Attention Required body is a block",
			resp:       makeCFResponse(http.StatusForbidden, nil, cfTestURL),
			body:       []byte("<html><title>Attention Required! | Cloudflare</title></html>"),
			wantKind:   CFKindBlock,
			wantStatus: http.StatusForbidden,
			wantURL:    cfTestURL,
		},
		{
			name: "403 with cf-mitigated block and challenge body marker is a block (block wins)",
			resp: func() *http.Response {
				h := http.Header{}
				h.Set("cf-mitigated", "block")
				h.Set("Server", "cloudflare")
				return makeCFResponse(http.StatusForbidden, h, cfTestURL)
			}(),
			body:       []byte("<html>Just a moment... _cf_chl_opt</html>"),
			wantKind:   CFKindBlock,
			wantStatus: http.StatusForbidden,
			wantURL:    cfTestURL,
		},
		{
			name:    "200 ordinary body is not a challenge",
			resp:    makeCFResponse(http.StatusOK, nil, cfTestURL),
			body:    []byte("<html><body>normal content</body></html>"),
			wantNil: true,
		},
		{
			name:    "nil response is not a challenge",
			resp:    nil,
			body:    []byte("anything"),
			wantNil: true,
		},
		{
			name: "200 body embedding challenge scripts is not a false positive",
			resp: func() *http.Response {
				h := http.Header{}
				h.Set("Server", "cloudflare")
				return makeCFResponse(http.StatusOK, h, cfTestURL)
			}(),
			body:    []byte(`<script src="/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1"></script><script>window._cf_chl_opt={};</script>`),
			wantNil: true,
		},
		{
			name: "503 with Server cloudflare and challenge marker is a challenge",
			resp: func() *http.Response {
				h := http.Header{}
				h.Set("Server", "cloudflare")
				return makeCFResponse(http.StatusServiceUnavailable, h, cfTestURL)
			}(),
			body:       []byte("<html><body>Just a moment...</body></html>"),
			wantKind:   CFKindChallenge,
			wantStatus: http.StatusServiceUnavailable,
			wantURL:    cfTestURL,
		},
		{
			name: "403 with mixed-case Server header and marker is a challenge",
			resp: func() *http.Response {
				h := http.Header{}
				h.Set("Server", "CloudFlare")
				return makeCFResponse(http.StatusForbidden, h, cfTestURL)
			}(),
			body:       []byte("<html>Just a moment...</html>"),
			wantKind:   CFKindChallenge,
			wantStatus: http.StatusForbidden,
			wantURL:    cfTestURL,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := DetectCF(tt.resp, tt.body)

			if tt.wantNil {
				if got != nil {
					t.Fatalf("DetectCF() = %+v; want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("DetectCF() = nil; want Kind=%q Status=%d", tt.wantKind, tt.wantStatus)
			}
			if got.Kind != tt.wantKind {
				t.Errorf("Kind = %q; want %q", got.Kind, tt.wantKind)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %d; want %d", got.Status, tt.wantStatus)
			}
			if got.URL != tt.wantURL {
				t.Errorf("URL = %q; want %q", got.URL, tt.wantURL)
			}
		})
	}
}
