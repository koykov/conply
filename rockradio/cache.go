package main

import (
	"fmt"
	"strings"

	"github.com/koykov/conply"
)

// List of cached channels.
type ChannelsCache []*ChannelCache

// Cached channel.
type ChannelCache struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
}

// Get the channel by given ID.
func (cc *ChannelsCache) GetGroupById(id uint64) *ChannelCache {
	for _, group := range *cc {
		if group.Id == id {
			return group
		}
	}
	return nil
}

// Build a human readable list of a channels.
func (cc *ChannelsCache) PrettyPrint() string {
	var res []string
	for _, channel := range *cc {
		res = append(res, fmt.Sprintf("%d - %s", channel.Id, channel.Title))
	}
	return strings.Join(res, "\n")
}

func (cc *ChannelsCache) Len() int {
	return len(*cc)
}

func (cc *ChannelsCache) Swap(i, j int) {
	(*cc)[i], (*cc)[j] = (*cc)[j], (*cc)[i]
}

func (cc *ChannelsCache) Less(i, j int) bool {
	return (*cc)[i].Id < (*cc)[j].Id
}

// Load channels list from the cache.
func ChannelsFromCache(path string) (ChannelsCache, error) {
	cc := ChannelsCache{}
	err := conply.UnmarshalFile(path, &cc)
	return cc, err
}
