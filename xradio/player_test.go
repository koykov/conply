package main

import "testing"

func TestAudioTokenRegex(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string // "" means the pattern must not match
	}{
		{
			name: "realistic station html fragment",
			html: `...var stream={"stream":"http://x","audio_token":"bc082b98dbbc625717a5867f71e94027","channel_id":1};...`,
			want: "bc082b98dbbc625717a5867f71e94027",
		},
		{
			name: "token among other json fields",
			html: `{"audio_token":"0a1b2c3d4e5f60718293a4b5c6d7e8f9","channel":"rock","id":42}`,
			want: "0a1b2c3d4e5f60718293a4b5c6d7e8f9",
		},
		{
			name: "first token wins when repeated",
			html: `"audio_token":"aaaa1111" ... "audio_token":"bbbb2222"`,
			want: "aaaa1111",
		},
		{
			name: "no token present",
			html: `<html><body>nothing to see here</body></html>`,
			want: "",
		},
		{
			name: "uppercase token does not match the lowercase-only pattern",
			html: `"audio_token":"ABC123DEF456"`,
			want: "",
		},
		{
			name: "token containing a separator does not match",
			html: `"audio_token":"abc-123"`,
			want: "",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := atokenRe.FindStringSubmatch(tt.html)
			got := ""
			if m != nil {
				got = m[1]
			}
			if got != tt.want {
				t.Fatalf("atokenRe.FindStringSubmatch(%q) capture = %q; want %q", tt.html, got, tt.want)
			}
		})
	}
}
