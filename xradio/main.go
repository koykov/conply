package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	kb "github.com/koykov/helpers/keybind"
	v "github.com/koykov/helpers/verbose"

	"github.com/koykov/conply"
)

var (
	ply       *Player
	options   conply.Options
	keybind   *kb.Keybind
	verbose   *v.Verbose
	waitGroup *sync.WaitGroup

	stations = Stations{
		{"rock", "rockradio", "https://www.rockradio.com", "https://api.audioaddict.com/v1"},
		{"rockradio", "rockradio", "https://www.rockradio.com", "https://api.audioaddict.com/v1"},
		{"rockradio.com", "rockradio", "https://www.rockradio.com", "https://api.audioaddict.com/v1"},
		{"jazz", "jazzradio", "https://www.jazzradio.com", "https://api.audioaddict.com/v1"},
		{"jazzradio", "jazzradio", "https://www.jazzradio.com", "https://api.audioaddict.com/v1"},
		{"jazzradio.com", "jazzradio", "https://www.jazzradio.com", "https://api.audioaddict.com/v1"},
		{"classic", "classicradio", "https://www.classicalradio.com", "https://api.audioaddict.com/v1"},
		{"classicradio", "classicradio", "https://www.classicalradio.com", "https://api.audioaddict.com/v1"},
		{"classicradio.com", "classicradio", "https://www.classicalradio.com", "https://api.audioaddict.com/v1"},
		{"tunes", "radiotunes", "https://www.radiotunes.com", "https://api.audioaddict.com/v1"},
		{"radiotunes", "radiotunes", "https://www.radiotunes.com", "https://api.audioaddict.com/v1"},
		{"radiotunes.com", "radiotunes", "https://www.radiotunes.com", "https://api.audioaddict.com/v1"},
		{"zen", "zenradio", "https://www.zenradio.com", "https://api.audioaddict.com/v1"},
		{"zenradio", "zenradio", "https://www.zenradio.com", "https://api.audioaddict.com/v1"},
		{"zenradio.com", "zenradio", "https://www.zenradio.com", "https://api.audioaddict.com/v1"},
	}
	station *Station

	nc0      = flag.Bool("no-cache", false, "Ignore cache data")
	nc1      = flag.Bool("nc", false, `Alias for "--no-cache"`)
	channel  = flag.Int("c", 0, "Channel ID.")
	verbose1 = flag.Bool("v", false, "Verbosity level 1")
	verbose2 = flag.Bool("vv", false, "Verbosity level 2")
	verbose3 = flag.Bool("vvv", false, "Verbosity level 3")

	sigStop = make(chan os.Signal)
)

func init() {
	// Check station alias.
	if len(os.Args) < 2 {
		v.NewVerbose(v.LevelFail).Fail("xradio: missing station operand\nTry \"xradio --help\" for more information")
		os.Exit(1)
	}
	// Get station alias.
	alias := os.Args[1]

	// Check generate mode.
	if alias == "generate" {
		generate()
		os.Exit(0)
	}

	// Display help message on --help option and exit.
	if alias == "--help" {
		fmt.Println(`Usage: xradio [<station alias>|generate] [options]`)
		fmt.Println(`Options:
  -c                Channel ID (omit to see list of possible channels)
  --nc, --no-cache  Ignore cache data
  -v, -vv, -vvv     Display verbose information of levels 1-3`)
		fmt.Println("\nStation aliases:")
		fmt.Println(stations.PrettyPrint())
		os.Exit(0)
	}

	station = stations.Look(alias)
	if station == nil {
		v.NewVerbose(v.LevelFail).Failf("xradio: unknown station \"%s\"\nTry \"xradio --help\" for more information", alias)
		os.Exit(1)
	}
	// Omit arg 1 (station name) to parse flags properly.
	os.Args = os.Args[1:]

	// Parse flags.
	flag.Parse()

	// Prepare options.
	options = conply.Options{"station": station}

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

	// Init the player.
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

	waitGroup = &sync.WaitGroup{}
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

	// Get audio token.
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		verbose.Debug1("Get audio token")
		if err := ply.RetrieveAToken(); err != nil {
			verbose.Fail("Couldn't retrieve audio token: ", err)
			_ = conply.Halt(1)
		} else {
			verbose.Debug2("Audio token retrieved: ", ply.atoken)
		}
	}()

	// Get channels.
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		verbose.Debug1("Get channels")

		// Check cache first.
		cacheFile, _ := conply.GetChannelsPathWS(Bundle, ply.station.Key)
		verbose.Debug2("Look for channels in cache ", cacheFile)
		regenRequire := !conply.FileExists(cacheFile) || conply.FileAge(cacheFile) > CacheExpire || options["noCache"].(bool)
		if regenRequire {
			verbose.Debug2(`Cache is invalid or expired or "--no-cache" options has applied, try to regenerate it`)
			verbose.Debug2("Looking for channels list in remote site")
			if err := ply.RetrieveChannels(); err != nil {
				verbose.Fail("Couldn't retrieve channels from remote site: ", err)
				_ = conply.Halt(1)
			} else {
				verbose.Debug2f("Channels list has been retrieved from remote site, total retrieved: %d", len(ply.cache))
				if err := conply.MarshalFile(cacheFile, ply.cache, true); err != nil {
					verbose.Fail("Writing cache error: ", err)
				} else {
					verbose.Debug3("Channels list has been saved in cache file ", cacheFile)
				}
			}
		} else {
			var err error
			verbose.Debug2("Get channels from cache")
			if ply.cache, err = ChannelsFromCache(cacheFile); err != nil {
				verbose.Fail("Cache reading problem: ", err)
				_ = conply.Halt(1)
			} else {
				verbose.Debug3f("Channels list has been retrieved from cache, total retrieved: %d", len(ply.cache))
			}
		}
	}()

	// Wait for retrieving token and channels.
	waitGroup.Wait()

	// Ask channel ID.
	if ply.chIdx > 0 {
		verbose.Debug1f("Channel predefined: %d", ply.chIdx)
	} else {
		verbose.Debug1("Ask for channel to play")
		reader := bufio.NewReader(os.Stdin)
		verbose.Infof("Channel ID:\n%s", ply.cache.PrettyPrint())
		attempts := 0
		for {
			fmt.Print("Channel: ")
			fail := false
			attempts++
			chIdx, err := reader.ReadString('\n')
			if err != nil {
				fail = true
				verbose.Fail("Couldn't receive channel ID from stdin: ", err)
			}
			verbose.Debug2f("Raw value you specified: %#v", chIdx)
			ply.chIdx, err = strconv.ParseUint(strings.Trim(chIdx, "\n"), 10, 64)
			if err != nil {
				fail = true
				verbose.Fail("Couldn't convert value to channel ID: ", err)
			} else {
				verbose.Debug3f("Channel ID from raw value: %v", ply.chIdx)
			}
			channel := ply.cache.GetGroupById(ply.chIdx)
			if channel == nil {
				fail = true
				verbose.Fail("Channel ID you specified doesn't exists, try again")
			}
			if !fail {
				break
			}
			if fail && attempts >= 3 {
				verbose.Failf("Oops, you've specified wrong channel ID %d times. Exiting.", attempts)
				_ = conply.Halt(1)
				break
			}
		}
	}
	channel := ply.cache.GetGroupById(ply.chIdx)
	verbose.Infof("Playing: %s", channel.Title)

	var (
		attempts     = 0
		RefreshToken = func(ply *Player) {
			if err := ply.RetrieveAToken(); err != nil {
				verbose.Fail("Couldn't retrieve audio token: ", err)
				_ = conply.Halt(1)
			} else {
				verbose.Debug2("Audio token retrieved: ", ply.atoken)
			}
		}
	)
	for {
		// Try to get chunk of tracks.
		attempts++
		err := ply.RetrieveTracks()
		if err != nil {
			if attempts >= 3 {
				// Attempts limit has been exceeded, stop executing.
				verbose.Failf("%d failed attempts when retrieve the tracks. Exiting.", attempts)
				_ = conply.Halt(1)
				break
			}
			verbose.Failf("Couldn't retrieve tracks: %s. Next attempt after 5 seconds", err)
			time.Sleep(time.Second * 5)
			continue
		}
		attempts = 0
		// Register a tick to refresh the audio token.
		ply.ticks["token"] = time.After(time.Duration(ply.channel.Length-5) * time.Second)

		verbose.Debug1f("%d tracks retrieved", len(ply.channel.Tracks))
		verbose.Debug2f("Tracks:\n%s", ply.channel.PrettyPrint())
		verbose.Debug1f("Next query after %s", conply.FormatTime(uint64(ply.channel.Length)))

		for i, track := range ply.channel.Tracks {
			// Register a tick at the end of the current track.
			ply.ticks["track"] = time.After(time.Duration(track.Content.Length) * time.Second)
			verbose.Info(track.ComposeTitle())

			// Play the track.
			ply.SetTrack(&track)
			go func(ply *Player) {
				err := ply.Play()
				if err != nil {
					verbose.Fail("Play failed due to error: ", err)
				}
			}(ply)

			finishTrack, nextTrack := false, false
			for {
				select {
				case <-ply.ticks["track"]:
					// Caught a signal of end of the current track.
					finishTrack = true
				case <-ply.signals["next"]:
					// Caught a signal to switch to the next track.
					nextTrack = true
					if i == len(ply.channel.Tracks)-1 {
						verbose.Debug1("Force refresh the Audio token due to you skipped the last track in the chunk")
						// We're skip the last track in the chunk. Need to have fresh audio token right now.
						RefreshToken(ply)
					}
				case <-ply.ticks["token"]:
					// It's time to get the fresh audio token.
					go func(ply *Player) {
						verbose.Debug1("Audio token expired, try to get fresh one")
						RefreshToken(ply)
					}(ply)
				}
				if finishTrack || nextTrack {
					// Stop the playing of current track and restore the status.
					status := ply.status
					_ = ply.Stop()
					ply.status = status
					switch {
					case finishTrack:
						verbose.Debug1("Current track finished, shift to the next")
					case nextTrack:
						verbose.Debug1("Current track skipped, shift to the next")
					}
					break
				}
			}
		}
	}
}

// Generate bash aliases for each station.
func generate() {
	verbose = v.NewVerbose(v.LevelInfo)
	processed := map[string]bool{}
	for _, st := range stations {
		if _, ok := processed[st.Key]; ok {
			continue
		}
		processed[st.Key] = true

		body := `#!/bin/bash

if ! command -v xradio &> /dev/null
then
    echo "xradio: command not found. Please check https://github.com/koykov/conply/blob/master/xradio/readme.md for installation instructions"
    exit 1
fi`
		body += "\n\nxradio " + st.Key + " \"$@\"\n"

		err := conply.FilePut(st.Key, body)
		if err != nil {
			verbose.Fail(err)
			return
		}
		err = os.Chmod(st.Key, 0775)
		if err != nil {
			verbose.Fail(err)
			return
		}
		verbose.Infof("generate alias %s for station %s", st.Key, st.Station)
	}
}
