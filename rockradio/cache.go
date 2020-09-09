package main

import (
	"fmt"
	"strings"

	"github.com/koykov/conply"
)

// List of cached channels.
type ChannelsCache map[uint64]*ChannelCache

// Cached channel.
type ChannelCache struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
}

// Load channels list from the cache.
func ChannelsFromCache(path string) (ChannelsCache, error) {
	cc := ChannelsCache{}
	err := conply.UnmarshalFile(path, &cc)
	return cc, err
}

// Build a human readable list of a channels.
func (cc *ChannelsCache) PrettyPrint() string {
	var res []string
	for _, channel := range *cc {
		res = append(res, fmt.Sprintf("%d - %s", channel.Id, channel.Title))
	}
	return strings.Join(res, "\n")
}
