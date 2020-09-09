package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	kb "github.com/koykov/helpers/keybind"
	v "github.com/koykov/helpers/verbose"

	"github.com/koykov/conply"
)

var (
	ply     *Player
	options conply.Options
	keybind *kb.Keybind
	verbose *v.Verbose

	nc0      = flag.Bool("no-cache", false, "Ignore cache data")
	nc1      = flag.Bool("nc", false, `Alias for "--no-cache"`)
	channel  = flag.Int("c", 0, "Channel ID.")
	verbose1 = flag.Bool("v", false, "Verbosity level 1")
	verbose2 = flag.Bool("vv", false, "Verbosity level 2")
	verbose3 = flag.Bool("vvv", false, "Verbosity level 3")

	sigStop = make(chan os.Signal)
)

func init() {
	flag.Parse()

	options = conply.Options{}

	// Define verbosity level.
	switch {
	case *verbose3:
		options["verboseLevel"] = v.LevelDebug3
	case *verbose2:
		options["verboseLevel"] = v.LevelDebug2
	case *verbose1:
		options["verboseLevel"] = v.LevelDebug1
	default:
		options["verboseLevel"] = v.LevelInfo
	}
	// Cache control.
	options["noCache"] = *nc0 || *nc1
	// Predefined channel.
	options["channel"] = uint64(*channel)

	verbose = v.NewVerbose(options["verboseLevel"].(v.VerbosityLevel))
	ply = NewPlayer(verbose, options)
	verbose.Debug1f("Init options:\n%s", options.PrettyPrint())
	if err := ply.Init(); err != nil {
		verbose.Fail("Initialization failed due to error: ", err)
		_ = conply.Halt(1)
	} else {
		verbose.Debug1("Player has initialized")
	}

	// Init keybinding.
	hkPath, _ := conply.GetHKPath(Bundle)
	keybind = kb.NewKeybind(ply)
	if err := keybind.LoadFromFile(hkPath); err != nil {
		verbose.Fail("Hotkeys will unavailable during this session due to error: ", err)
	}
	if err := keybind.Init(); err != nil {
		verbose.Fail("Hotkeys will unavailable during this session due to error: ", err)
	}
}

func main() {
	// Wait for hotkeys.
	go keybind.Wait()

	// Register cleanup callback.
	signal.Notify(sigStop, os.Interrupt, syscall.SIGTERM)
	signal.Notify(sigStop, os.Interrupt, syscall.SIGKILL)
	go func() {
		<-sigStop
		if err := ply.Cleanup(); err != nil {
			verbose.Fail("Cleanup failed due to error: ", err)
			_ = conply.Halt(1)
		}
		verbose.Debug2("Cleanup finished")
		_ = conply.Halt(0)
	}()

	verbose.Debug1("Get groups and channels")

	// Check cache first.
	cacheFile, _ := conply.GetChannelsPath(Bundle)
	verbose.Debug2("Look for data in cache ", cacheFile)
	regenRequire := !conply.FileExists(cacheFile) || conply.FileAge(cacheFile) > CacheExpire || options["noCache"].(bool)
	if regenRequire {
		verbose.Debug2(`Cache is invalid or expired or "--no-cache" options has applied, try to regenerate it`)
		verbose.Debug2("Looking for groups and channels in remote site")
		if err := ply.RetrieveTree(); err != nil {
			verbose.Fail("Couldn't retrieve data from remote site: ", err)
			_ = conply.Halt(1)
		} else {
			verbose.Debug2f("Groups and channels list has been retrieved from remote site, total groups retrieved: %d", len(ply.cache))
			if err := conply.MarshalFile(cacheFile, ply.cache, true); err != nil {
				verbose.Fail("Writing cache error: ", err)
			} else {
				verbose.Debug3("Groups and channels list has been saved in cache file ", cacheFile)
			}
		}
	} else {
		var err error
		verbose.Debug2("Get channels from cache")
		if ply.cache, err = ChannelsFromCache(cacheFile); err != nil {
			verbose.Fail("Cache reading problem: ", err)
			_ = conply.Halt(1)
		} else {
			verbose.Debug3f("Groups and channels list has been retrieved from cache, total groups retrieved: %d", len(ply.cache))
		}
	}

	// Ask group/channel ID.
	if ply.chIdx > 0 {
		verbose.Debug1f("Channel predefined: %d", ply.chIdx)
		ply.group, ply.channel = ply.GetByChannelId(ply.chIdx)
	} else {
		bundles := []map[string]string{
			{
				"label":    "group",
				"label_uc": "Group",
			},
			{
				"label":    "channel",
				"label_uc": "Channel",
			},
		}
		for k, bundle := range bundles {
			verbose.Debug1f("Ask for %s to play", bundle["label"])
			reader := bufio.NewReader(os.Stdin)
			switch k {
			case 0:
				verbose.Infof("%s ID:\n%s", bundle["label_uc"], ply.cache.PrettyPrint())
			case 1:
				group := ply.cache[ply.grIdx]
				verbose.Infof("%s ID:\n%s", bundle["label_uc"], group.Channels.PrettyPrint())
			}
			attempts := 0
			for {
				fmt.Printf("%s: ", bundle["label_uc"])
				fail := false
				attempts++
				idx, err := reader.ReadString('\n')
				var idxParsed uint64
				if err != nil {
					fail = true
					verbose.Failf("Couldn't receive %s ID from stdin: ", bundle["label"], err)
				}
				verbose.Debug2f("Raw value you specified: %#v", idx)
				switch k {
				case 0:
					ply.grIdx, err = strconv.ParseUint(strings.Trim(idx, "\n"), 10, 64)
					idxParsed = ply.grIdx
				case 1:
					ply.chIdx, err = strconv.ParseUint(strings.Trim(idx, "\n"), 10, 64)
					idxParsed = ply.chIdx
				}
				if err != nil {
					fail = true
					verbose.Failf("Couldn't convert value to %s ID: ", bundle["label"], err)
				} else {
					verbose.Debug3f("%s ID from raw value: %v", bundle["label_uc"], idxParsed)
				}
				switch k {
				case 0:
					if _, ok := ply.cache[idxParsed]; !ok {
						fail = true
						verbose.Failf("%s ID you specified doesn't exists, try again", bundle["label_uc"])
					} else {
						group := ply.cache[idxParsed]
						ply.group = &group
					}
				case 1:
					if _, ok := ply.group.Channels[idxParsed]; !ok {
						fail = true
						verbose.Failf("%s ID you specified doesn't exists, try again", bundle["label_uc"])
					} else {
						channel := ply.group.Channels[idxParsed]
						ply.channel = &channel
					}
				}

				if !fail {
					break
				}
				if fail && attempts >= 3 {
					verbose.Failf("Oops, you've specified wrong %s ID %d times. Exiting.", bundle["label"], attempts)
					_ = conply.Halt(1)
					break
				}
			}
		}
	}

	verbose.Infof("Playing: %s/%s", ply.group.Title, ply.channel.Title)
	// Playing loop.
	attempts := 0
	for {
		err := ply.RetrieveTrack()
		attempts++
		ply.ticks["track"] = time.After(time.Duration(ply.nextFetch) * time.Second)
		if err != nil {
			if attempts >= 3 {
				// Attempts limit has been exceeded, stop executing.
				verbose.Failf("%d failed attempts when retrieve the track. Exiting.", attempts)
				_ = conply.Halt(1)
				break
			}
			verbose.Failf("Got error during retrieve the track: %s. I'll try again after %d seconds.", err, DelayAfterFail)
		} else {
			attempts = 0
		}
		if ply.trackUid != ply.prevTrackUid {
			verbose.Info(ply.track.ComposeTitle())
			err = ply.Play()
			if err != nil {
				verbose.Fail("Play failed due to error: ", err)
				ply.nextFetch = DelayAfterFail
			}
		}
		select {
		case <-ply.ticks["track"]:
			// Just waste the time.
		}
	}
}
