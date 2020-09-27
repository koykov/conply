package main

import (
	"fmt"

	"github.com/koykov/conply"
)

// Rockradio track.
type Track struct {
	Id        uint64  `json:"id"`
	Artist    string  `json:"display_artist"`
	Title     string  `json:"display_title"`
	Album     string  `json:"release"`
	AlbumDate string  `json:"release_date"`
	Content   Content `json:"content"`
}

// Substructure to store list of assets.
type Content struct {
	Length float64 `json:"length"`
	Assets []Asset `json:"assets"`
}

// Asset.
type Asset struct {
	Url string `json:"url"`
}

// Build track's title to display it.
func (t *Track) ComposeTitle() string {
	format := "%s - %s [%s] - %s"
	title := fmt.Sprintf(format, t.Artist, t.Title, t.Album, conply.FormatTime(uint64(t.Content.Length)))
	if len(t.Album) == 0 {
		format = "%s - %s - %s"
		title = fmt.Sprintf(format, t.Artist, t.Title, conply.FormatTime(uint64(t.Content.Length)))
	}
	return title
}

// Build track's title to build a download path.
func (t *Track) ComposeDlTitle() string {
	return fmt.Sprintf("%s - %s", t.Artist, t.Title)
}

// Return track's URL to play or download.
func (t *Track) GetURL() string {
	return t.Content.Assets[0].Url
}
