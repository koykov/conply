package main

import (
	"fmt"
	"github.com/koykov/conply"
)

type Track struct {
	Status    uint64 `json:"status"`
	Result    Result `json:"result"`
	ErrorCode uint64 `json:"errorCode"`
}

type Result struct {
	About About `json:"about"`
	Stat  Stat  `json:"stat"`
}

type About struct {
	Title  string  `json:"title"`
	Artist string  `json:"title_executor"`
	Audio  []Audio `json:"audio"`
	Album  Album   `json:"album"`
}

type Audio struct {
	TrackUid uint64 `json:"trackuid"`
	Filename string `json:"filename"`
}

type Album struct {
	Title       string `json:"title"`
	ReleaseDate string `json:"releaseDate"`
}

type Stat struct {
	StartSong  uint64 `json:"startSong"`
	FinishSong uint64 `json:"finishSong"`
	ServerTime uint64 `json:"serverTime"`
}

// Build track's title to build a download path.
func (t *Track) ComposeTitle() string {
	about := t.Result.About
	return fmt.Sprintf("%s - %s [%s] - %s", about.Artist, about.Title, about.Album.Title, t.GetDurationStr())
}

// Build track's title to build a download path.
func (t *Track) ComposeDlTitle() string {
	about := t.Result.About
	return fmt.Sprintf("%s - %s", about.Artist, about.Title)
}

// Return duration as formatted string like mm:ss.
func (t *Track) GetDurationStr() string {
	diff := uint64(t.Result.Stat.FinishSong - t.Result.Stat.ServerTime)
	return conply.FormatTime(diff)
}

// Return track's URL to play or download.
func (t *Track) GetURL() string {
	return t.Result.About.Audio[0].Filename
}