package main

import "strings"

// Rockradio channel.
type Channel struct {
	Id     uint64 `json:"channel_id"`
	Expiry string `json:"expires_on"`
	Length float64
	Tracks []Track `json:"tracks"`
}

// Build a human readable list of a channels.
func (c *Channel) PrettyPrint() string {
	list := make([]string, 0)
	for _, track := range c.Tracks {
		list = append(list, " * "+track.ComposeTitle())
	}
	return strings.Join(list, "\n")
}
