package main

import (
	"fmt"
	"strings"

	"github.com/koykov/conply"
)

type ChannelGroups []*ChannelGroup

type ChannelGroup struct {
	Id       uint64   `json:"id"`
	Title    string   `json:"title"`
	Channels Channels `json:"channels"`
}

// Get the group by given ID.
func (cg *ChannelGroups) GetGroupById(id uint64) *ChannelGroup {
	for _, group := range *cg {
		if group.Id == id {
			return group
		}
	}
	return nil
}

// Build a human readable list of a groups.
func (cg *ChannelGroups) PrettyPrint() string {
	var res []string
	for _, group := range *cg {
		res = append(res, fmt.Sprintf("%d - %s", group.Id, group.Title))
	}
	return strings.Join(res, "\n")
}

func (cg *ChannelGroups) Len() int {
	return len(*cg)
}

func (cg *ChannelGroups) Swap(i, j int) {
	(*cg)[i], (*cg)[j] = (*cg)[j], (*cg)[i]
}

func (cg *ChannelGroups) Less(i, j int) bool {
	return (*cg)[i].Id < (*cg)[j].Id
}

type Channels []*ChannelCache

type ChannelCache struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
}

// Get the channel from group by given ID.
func (c *Channels) GetChannelById(id uint64) *ChannelCache {
	for _, channel := range *c {
		if channel.Id == id {
			return channel
		}
	}
	return nil
}

// Build a human readable list of a channels.
func (c *Channels) PrettyPrint() string {
	var res []string
	for _, channel := range *c {
		res = append(res, fmt.Sprintf("%d - %s", channel.Id, channel.Title))
	}
	return strings.Join(res, "\n")
}

func (c *Channels) Len() int {
	return len(*c)
}

func (c *Channels) Swap(i, j int) {
	(*c)[i], (*c)[j] = (*c)[j], (*c)[i]
}

func (c *Channels) Less(i, j int) bool {
	return (*c)[i].Id < (*c)[j].Id
}

// Load groups/channels list from the cache.
func ChannelsFromCache(path string) (ChannelGroups, error) {
	cc := ChannelGroups{}
	err := conply.UnmarshalFile(path, &cc)
	return cc, err
}
