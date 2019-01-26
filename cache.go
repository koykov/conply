package conply

import (
	"fmt"
	"strings"
)

type ChannelsCache map[uint64]*ChannelCache

type ChannelCache struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
}

func ChannelsFromCache(path string) (ChannelsCache, error) {
	cc := ChannelsCache{}
	err := UnmarshalFile(path, &cc)
	return cc, err
}

func (cc *ChannelsCache) PrettyPrint() string {
	var res []string
	for _, channel := range *cc {
		res = append(res, fmt.Sprintf("%d - %s", channel.Id, channel.Title))
	}
	return strings.Join(res, "\n")
}
