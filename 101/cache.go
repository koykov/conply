package main

import (
	"fmt"
	"github.com/koykov/conply"
	"strings"
)

type ChannelGroups map[uint64]ChannelGroup

type ChannelGroup struct {
	Id       uint64   `json:"id"`
	Title    string   `json:"title"`
	Channels Channels `json:"channels"`
}

// Build a human readable list of a groups.
func (cg *ChannelGroups) PrettyPrint() string {
	var res []string
	for _, group := range *cg {
		res = append(res, fmt.Sprintf("%d - %s", group.Id, group.Title))
	}
	return strings.Join(res, "\n")
}

type Channels map[uint64]ChannelCache

type ChannelCache struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
}

// Build a human readable list of a channels.
func (c *Channels) PrettyPrint() string {
	var res []string
	for _, channel := range *c {
		res = append(res, fmt.Sprintf("%d - %s", channel.Id, channel.Title))
	}
	return strings.Join(res, "\n")
}

// Load groups/channels list from the cache.
func ChannelsFromCache(path string) (ChannelGroups, error) {
	cc := ChannelGroups{}
	err := conply.UnmarshalFile(path, &cc)
	return cc, err
}
