package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	v "github.com/koykov/helpers/verbose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/koykov/conply"
)

// ---------------------------------------------------------------------------
// Shared fixtures
// ---------------------------------------------------------------------------

const (
	// A track whose content carries no assets at all.
	emptyContentJSON = `{"content":{}}`
	// Playable tracks; content.length feeds channel.Length accumulation.
	playableAJSON = `{"id":1,"content":{"length":120,"assets":[{"url":"//content.example/a.mp4"}]}}`
	playableBJSON = `{"id":2,"content":{"length":30.5,"assets":[{"url":"//content.example/b.mp4"}]}}`
	// A track with an asset entry whose url is empty: must be treated as
	// unplayable and must NOT contribute to channel.Length.
	emptyURLJSON = `{"id":3,"content":{"length":10,"assets":[{"url":""}]}}`

	stationPageWithToken = `{"audio_token":"deadbeefcafe0123"}`
	stationPageNoToken   = `<html><body>no token in this page</body></html>`
)

// tracklistJSON renders an AudioAddict-like channel payload.
func tracklistJSON(tracks ...string) string {
	return `{"channel_id":1,"tracks":[` + strings.Join(tracks, ",") + `]}`
}

// uaCaptureServer returns a 200-OK server that reports, on the buffered
// channel, the User-Agent it observed. Exactly one request is expected.
func uaCaptureServer(t *testing.T) (*httptest.Server, <-chan string) {
	t.Helper()
	hits := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits <- r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	return srv, hits
}

// newTestPlayer builds a Player suitable for retrieval tests: real
// user-agent session, but pointed at a test server via Station.API/Station.
func newTestPlayer(api, station string, key, atoken string, chIdx uint64, uas []string, uaIdx int) *Player {
	return &Player{
		atoken:  atoken,
		station: &Station{API: api, Station: station, Key: key},
		chIdx:   chIdx,
		uas:     uas,
		uaIdx:   uaIdx,
		session: newSession(uas[uaIdx]),
	}
}

// ---------------------------------------------------------------------------
// Track.GetURL
// ---------------------------------------------------------------------------

func TestTrackGetURL(t *testing.T) {
	cases := []struct {
		name  string
		track *Track
		want  string
	}{
		{
			name:  "nil receiver returns empty",
			track: nil,
			want:  "",
		},
		{
			name:  "zero value track returns empty",
			track: &Track{},
			want:  "",
		},
		{
			name:  "content without assets returns empty",
			track: &Track{Content: Content{Length: 42}},
			want:  "",
		},
		{
			name:  "single asset returns its url",
			track: &Track{Content: Content{Assets: []Asset{{Url: "https://cdn.example/x.mp3"}}}},
			want:  "https://cdn.example/x.mp3",
		},
		{
			name:  "first asset wins when several present",
			track: &Track{Content: Content{Assets: []Asset{{Url: "first"}, {Url: "second"}}}},
			want:  "first",
		},
		{
			name:  "asset with empty url returns empty",
			track: &Track{Content: Content{Assets: []Asset{{Url: ""}}}},
			want:  "",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Must never panic, including for the nil receiver.
			assert.Equal(t, tt.want, tt.track.GetURL(), "GetURL()")
		})
	}
}

// ---------------------------------------------------------------------------
// RetrieveTracks
// ---------------------------------------------------------------------------

func TestRetrieveTracks(t *testing.T) {
	type capturedRequest struct {
		path     string
		rawQuery string
	}

	cases := []struct {
		name           string
		atoken         string
		body           string
		wantErrNoAudio bool
		wantErrSub     string
		wantRequests   int
		wantTracks     int
		wantURLs       []string
		wantLength     float64
	}{
		{
			name:           "all tracks with empty content return errNoAudioContent without panic",
			atoken:         "x",
			body:           tracklistJSON(emptyContentJSON, emptyContentJSON),
			wantErrNoAudio: true,
			wantRequests:   1,
		},
		{
			name:         "all playable tracks are kept and urls are rewritten to https",
			atoken:       "x",
			body:         tracklistJSON(playableAJSON, playableBJSON),
			wantRequests: 1,
			wantTracks:   2,
			wantURLs:     []string{"https://content.example/a.mp4", "https://content.example/b.mp4"},
			wantLength:   150.5,
		},
		{
			name:         "mixed playable and empty tracks keep only playable ones",
			atoken:       "x",
			body:         tracklistJSON(playableAJSON, emptyContentJSON, playableBJSON, emptyURLJSON),
			wantRequests: 1,
			wantTracks:   2,
			wantURLs:     []string{"https://content.example/a.mp4", "https://content.example/b.mp4"},
			wantLength:   150.5,
		},
		{
			name:           "asset with empty url is unplayable and contributes no length",
			atoken:         "x",
			body:           tracklistJSON(emptyURLJSON),
			wantErrNoAudio: true,
			wantRequests:   1,
		},
		{
			name:         "empty access token is rejected before any request",
			atoken:       "",
			body:         tracklistJSON(playableAJSON),
			wantErrSub:   "invalid access token",
			wantRequests: 0,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var (
				mu  sync.Mutex
				got []capturedRequest
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				got = append(got, capturedRequest{path: r.URL.Path, rawQuery: r.URL.RawQuery})
				mu.Unlock()
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			ply := &Player{
				atoken:  tt.atoken,
				station: &Station{API: srv.URL, Key: "testkey"},
				chIdx:   1,
				session: newSession(""),
			}

			err := ply.RetrieveTracks()

			mu.Lock()
			reqs := append([]capturedRequest(nil), got...)
			mu.Unlock()

			require.Len(t, reqs, tt.wantRequests, "HTTP requests")
			for _, rq := range reqs {
				assert.Equal(t, "/testkey/routines/channel/1", rq.path, "request path")
				assert.Contains(t, rq.rawQuery, "audio_token=x", "request query")
			}

			switch {
			case tt.wantErrNoAudio:
				require.ErrorIs(t, err, errNoAudioContent, "RetrieveTracks()")
				assert.Nil(t, ply.channel, "ply.channel")
			case tt.wantErrSub != "":
				require.Error(t, err, "RetrieveTracks()")
				assert.Contains(t, err.Error(), tt.wantErrSub, "RetrieveTracks() error")
			default:
				require.NoError(t, err, "RetrieveTracks()")
				require.NotNil(t, ply.channel, "ply.channel")
				require.Len(t, ply.channel.Tracks, tt.wantTracks, "len(ply.channel.Tracks)")
				for i, want := range tt.wantURLs {
					assert.Equal(t, want, ply.channel.Tracks[i].GetURL(), "Tracks[%d].GetURL()", i)
				}
				assert.Equal(t, tt.wantLength, ply.channel.Length, "ply.channel.Length")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rotateUA
// ---------------------------------------------------------------------------

func TestRotateUA(t *testing.T) {
	three := []string{"UA-alpha", "UA-beta", "UA-gamma"}

	cases := []struct {
		name    string
		uas     []string
		start   int
		wantOK  bool
		wantIdx int
	}{
		{name: "single UA cannot rotate", uas: []string{"UA-only"}, start: 0, wantOK: false, wantIdx: 0},
		{name: "empty pool cannot rotate", uas: nil, start: 0, wantOK: false, wantIdx: 0},
		{name: "advance one step from first", uas: three, start: 0, wantOK: true, wantIdx: 1},
		{name: "advance one step from middle", uas: three, start: 1, wantOK: true, wantIdx: 2},
		{name: "wrap around from last", uas: three, start: 2, wantOK: true, wantIdx: 0},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv, hits := uaCaptureServer(t)
			defer srv.Close()

			ply := &Player{uas: tt.uas, uaIdx: tt.start, session: newSession("initial-UA")}

			require.Equal(t, tt.wantOK, ply.rotateUA(), "rotateUA()")
			assert.Equal(t, tt.wantIdx, ply.uaIdx, "uaIdx")
			require.NotNil(t, ply.session, "session after rotateUA")

			// On a successful rotation the session must be rebuilt from the
			// new UA; on a failed one the original session must be untouched.
			wantUA := "initial-UA"
			if tt.wantOK {
				wantUA = tt.uas[tt.wantIdx]
			}
			_, err := ply.session.Get(srv.URL)
			require.NoError(t, err, "session.Get")
			assert.Equal(t, wantUA, <-hits, "request User-Agent")
		})
	}
}

// ---------------------------------------------------------------------------
// RetrieveTracksWithFallback
// ---------------------------------------------------------------------------

// TestRetrieveTracksWithFallbackRotatesToPlayableUA: UA #1 is served an empty
// tracklist, UA #2 a populated one; the fallback must rotate and succeed.
func TestRetrieveTracksWithFallbackRotatesToPlayableUA(t *testing.T) {
	uas := []string{"UA-one", "UA-two"}

	var channelReqs, stationReqs int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if strings.Contains(r.URL.Path, "routines/channel") {
			atomic.AddInt32(&channelReqs, 1)
			if ua == uas[0] {
				_, _ = io.WriteString(w, tracklistJSON(emptyContentJSON))
			} else {
				_, _ = io.WriteString(w, tracklistJSON(playableAJSON, playableBJSON))
			}
			return
		}
		atomic.AddInt32(&stationReqs, 1)
		_, _ = io.WriteString(w, stationPageWithToken)
	}))
	defer srv.Close()

	ply := newTestPlayer(srv.URL, srv.URL, "testkey", "tok", 1, uas, 0)
	require.NoError(t, ply.RetrieveTracksWithFallback())
	assert.GreaterOrEqual(t, atomic.LoadInt32(&channelReqs), int32(2), "channel requests (fallback did not rotate)")
	require.NotNil(t, ply.channel, "ply.channel")
	require.Len(t, ply.channel.Tracks, 2, "ply.channel.Tracks")
}

// TestRetrieveTracksWithFallbackExhausted: every UA is served an empty
// tracklist; the returned error must be the plain exhausted-pool error and
// must NOT be reported as a Cloudflare challenge.
func TestRetrieveTracksWithFallbackExhausted(t *testing.T) {
	uas := []string{"UA-one", "UA-two"}

	var channelReqs int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "routines/channel") {
			atomic.AddInt32(&channelReqs, 1)
			_, _ = io.WriteString(w, tracklistJSON(emptyContentJSON))
			return
		}
		_, _ = io.WriteString(w, stationPageWithToken)
	}))
	defer srv.Close()

	ply := newTestPlayer(srv.URL, srv.URL, "testkey", "tok", 1, uas, 0)
	err := ply.RetrieveTracksWithFallback()
	require.Error(t, err, "RetrieveTracksWithFallback()")

	var cf *conply.CFChallengeError
	require.False(t, errors.As(err, &cf), "error = %v (%T); want plain exhausted-pool error, got CFChallengeError", err, err)
	assert.Contains(t, err.Error(), "user-agents", "error")
	assert.Equal(t, int32(len(uas)), atomic.LoadInt32(&channelReqs), "channel requests (one per UA)")
}

// TestRetrieveTracksWithFallbackCFNoRotation: a Cloudflare challenge from
// RetrieveTracks must be returned immediately, with no UA rotation (a CF
// challenge is UA-based, so retrying is pointless).
func TestRetrieveTracksWithFallbackCFNoRotation(t *testing.T) {
	uas := []string{"UA-one", "UA-two"}

	var channelReqs, stationReqs int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "routines/channel") {
			atomic.AddInt32(&channelReqs, 1)
			w.Header().Set("cf-mitigated", "challenge")
			w.Header().Set("Server", "cloudflare")
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, "<html><body>Just a moment...</body></html>")
			return
		}
		atomic.AddInt32(&stationReqs, 1)
		_, _ = io.WriteString(w, stationPageWithToken)
	}))
	defer srv.Close()

	ply := newTestPlayer(srv.URL, srv.URL, "testkey", "tok", 1, uas, 0)
	err := ply.RetrieveTracksWithFallback()
	require.Error(t, err, "RetrieveTracksWithFallback()")

	var cf *conply.CFChallengeError
	require.ErrorAs(t, err, &cf)
	assert.Equal(t, int32(1), atomic.LoadInt32(&channelReqs), "channel requests (must not rotate on CF challenge)")
	assert.Equal(t, int32(0), atomic.LoadInt32(&stationReqs), "station requests (must not re-mint token on CF challenge)")
}

// TestRetrieveTracksWithFallbackContinuesOnUnparseableToken: when the rotated
// UA yields a station page without an audio_token, the fallback must move on to
// the next UA instead of aborting.
func TestRetrieveTracksWithFallbackContinuesOnUnparseableToken(t *testing.T) {
	uas := []string{"UA-one", "UA-two", "UA-three"}

	var channelReqs, stationReqs int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if strings.Contains(r.URL.Path, "routines/channel") {
			atomic.AddInt32(&channelReqs, 1)
			// Only the last UA is served a populated tracklist.
			if ua == uas[len(uas)-1] {
				_, _ = io.WriteString(w, tracklistJSON(playableAJSON))
			} else {
				_, _ = io.WriteString(w, tracklistJSON(emptyContentJSON))
			}
			return
		}
		atomic.AddInt32(&stationReqs, 1)
		// The middle UA's station page cannot be parsed.
		if ua == uas[1] {
			_, _ = io.WriteString(w, stationPageNoToken)
			return
		}
		_, _ = io.WriteString(w, stationPageWithToken)
	}))
	defer srv.Close()

	ply := newTestPlayer(srv.URL, srv.URL, "testkey", "tok", 1, uas, 0)
	require.NoError(t, ply.RetrieveTracksWithFallback(), "must continue past unparseable token")
	assert.Equal(t, int32(len(uas)), atomic.LoadInt32(&channelReqs), "channel requests (loop must not abort on unparseable token)")
	assert.Equal(t, int32(len(uas)-1), atomic.LoadInt32(&stationReqs), "station requests (one token refresh per rotation)")
	require.NotNil(t, ply.channel, "ply.channel")
	require.Len(t, ply.channel.Tracks, 1, "ply.channel.Tracks")
}

// ---------------------------------------------------------------------------
// NewPlayer
// ---------------------------------------------------------------------------

// TestNewPlayerRandomStart verifies the constructor wires a valid random UA
// index and a non-nil session. Init is deliberately not called (no VLC).
func TestNewPlayerRandomStart(t *testing.T) {
	station := &Station{Alias: "test", Key: "testkey", Station: "http://127.0.0.1", API: "http://127.0.0.1"}
	const chIdx = uint64(7)
	options := conply.Options{"station": station, "channel": chIdx}
	verbose := v.NewVerbose(v.LevelFail)

	for i := 0; i < 16; i++ {
		ply := NewPlayer(verbose, options)
		require.NotNil(t, ply, "NewPlayer()")
		require.NotEmpty(t, ply.uas, "NewPlayer() built a player with an empty UA pool")
		require.GreaterOrEqual(t, ply.uaIdx, 0, "uaIdx")
		require.Less(t, ply.uaIdx, len(ply.uas), "uaIdx")
		require.NotNil(t, ply.session, "session")
		assert.Same(t, station, ply.station, "station")
		assert.Equal(t, chIdx, ply.chIdx, "chIdx")
	}
}
