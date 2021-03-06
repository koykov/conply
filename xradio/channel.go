package main

import "strings"

// Rockradio channel.
type Channel struct {
	Id     uint64 `json:"channel_id"`
	Expiry string `json:"expires_on"`
	Length float64
	Tracks []Track `json:"tracks"`
}

// Currently playing representation of the channel.
type CurrentlyPlaying struct {
	Id    uint64 `json:"channel_id"`
	Key   string `json:"channel_key"`
	Name  string
	Track Track `json:"track"`
}

// Build a human readable list of a channels.
func (c *Channel) PrettyPrint() string {
	list := make([]string, 0)
	for _, track := range c.Tracks {
		list = append(list, " * "+track.ComposeTitle())
	}
	return strings.Join(list, "\n")
}
