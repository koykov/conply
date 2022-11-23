package main

import (
	"fmt"

	"github.com/koykov/conply"
	"github.com/koykov/jsonvector"
	"github.com/koykov/vector"
)

type Track struct {
	vec       jsonvector.Vector
	audiofile string
}

func NewTrack() *Track {
	vec := jsonvector.NewVector()
	return &Track{vec: *vec}
}

func (t *Track) Parse(p []byte) error {
	return t.vec.ParseCopy(p)
}

func (t Track) GetUidTrack() uint64 {
	u, _ := t.vec.DotUint("result.short.uidTrack")
	return u
}

func (t Track) GetAudiofile() string {
	return t.vec.DotString("result.short.audiofile")
}

func (t *Track) SetAudiofile(af string) {
	t.audiofile = af
}

func (t Track) GetShort() *vector.Node {
	return t.vec.Dot("result.short")
}

// ComposeTitle builds track's title to build a download path.
func (t *Track) ComposeTitle() string {
	about := t.GetShort()
	return fmt.Sprintf("%s - %s [%s] - %s", about.DotString("titleExecutor"), about.DotString("title"),
		about.DotString("album.albumTitle"), t.GetDurationStr())
}

// ComposeDlTitle build track's title to build a download path.
func (t *Track) ComposeDlTitle() string {
	about := t.GetShort()
	return fmt.Sprintf("%s - %s", about.DotString("titleExecutor"), about.DotString("title"))
}

// GetDurationStr returns duration as formatted string like mm:ss.
func (t *Track) GetDurationStr() string {
	diff := t.GetDiff()
	return conply.FormatTime(diff)
}

// GetURL returns track's URL to play or download.
func (t *Track) GetURL() string {
	return t.audiofile
}

func (t Track) GetDiff() uint64 {
	fs, _ := t.vec.DotUint("result.stat.finishSong")
	st, _ := t.vec.DotUint("result.stat.serverTime")
	return fs - st
}
