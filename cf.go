package conply

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

// Kinds of Cloudflare mitigation responses.
const (
	CFKindChallenge = "challenge"
	CFKindBlock     = "block"
)

// cfChallengeBodyMarkers are fragments that appear in a Cloudflare Managed
// Challenge interstitial page ("Just a moment...").
var cfChallengeBodyMarkers = []string{
	"Just a moment",
	"_cf_chl_opt",
	"challenge-platform",
	"cf-chl",
}

// CFChallengeError reports that a request was answered by a Cloudflare
// challenge or WAF block page instead of the expected content.
type CFChallengeError struct {
	URL    string
	Status int
	Kind   string
}

func (e *CFChallengeError) Error() string {
	return fmt.Sprintf("cloudflare %s on %s (status %d): the site requires browser validation the user-agent session could not pass; update the User-Agent or switch to a browser-backed session",
		e.Kind, e.URL, e.Status)
}

// DetectCF inspects an HTTP response and its body and returns a populated
// *CFChallengeError when the response is a Cloudflare challenge or block page.
// It returns nil for any other response, including a nil resp.
func DetectCF(resp *http.Response, body []byte) *CFChallengeError {
	if resp == nil {
		return nil
	}

	var url string
	if resp.Request != nil && resp.Request.URL != nil {
		url = resp.Request.URL.String()
	}
	status := resp.StatusCode

	// Detection is intentionally limited to 403/503 statuses to avoid false
	// positives on legitimate 200 pages that embed challenge scripts.
	if status == http.StatusForbidden || status == http.StatusServiceUnavailable {
		cfMitigated := strings.ToLower(resp.Header.Get("cf-mitigated"))
		if strings.Contains(cfMitigated, CFKindBlock) ||
			(status == http.StatusForbidden && bytes.Contains(body, []byte("Attention Required"))) {
			return &CFChallengeError{URL: url, Status: status, Kind: CFKindBlock}
		}
		serverCloudflare := strings.Contains(strings.ToLower(resp.Header.Get("Server")), "cloudflare")
		if strings.Contains(cfMitigated, CFKindChallenge) ||
			(serverCloudflare && containsAnyMarker(body, cfChallengeBodyMarkers)) {
			return &CFChallengeError{URL: url, Status: status, Kind: CFKindChallenge}
		}
	}

	return nil
}

func containsAnyMarker(body []byte, markers []string) bool {
	for _, m := range markers {
		if bytes.Contains(body, []byte(m)) {
			return true
		}
	}
	return false
}
