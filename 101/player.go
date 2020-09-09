package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	kb "github.com/koykov/helpers/keybind"
	v "github.com/koykov/helpers/verbose"
	"github.com/koykov/vlc"
	"github.com/mikkyang/id3-go"

	"github.com/koykov/conply"
)

const (
	Bundle         = "101.ru"
	Version        = "v0.1"
	CacheExpire    = 7 * 24 * 3600
	DelayAfterFail = 5
)

type Player struct {
	cache        ChannelGroups
	group        *ChannelGroup
	channel      *ChannelCache
	track        *Track
	grIdx        uint64
	chIdx        uint64
	nextFetch    uint64
	prevTrackUid uint64
	trackUid     uint64

	vlc      *vlc.Vlc
	status   conply.Status
	ticks    map[string]<-chan time.Time
	sigUtime int64
	muxDl    sync.Mutex

	verbose *v.Verbose
}

// The constructor.
func NewPlayer(verbose *v.Verbose, options map[string]interface{}) *Player {
	ply := Player{
		cache:  make(ChannelGroups, 0),
		grIdx:  options["channel"].(uint64),
		chIdx:  options["channel"].(uint64),
		status: conply.StatusPlay,
		ticks: map[string]<-chan time.Time{
			"track": make(chan time.Time),
		},
		verbose: verbose,
	}

	ply.verbose.Info(Bundle + " " + Version)

	return &ply
}

// Initialize the player.
// Checks and creates (if needed) environment and hotkeys config file.
func (ply *Player) Init() error {
	// Check and create the working environment.
	ply.verbose.Debug1("Check and prepare the environment")
	if err := conply.PrepareEnv(Bundle); err != nil {
		ply.verbose.Fail("Error preparing the environment")
		return err
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		ply.verbose.Warning("Couldn't find ffmpeg installed. That's not a problem for playing, but downloading will unavailable.")
	}
	ply.verbose.Debug2("Environment is OK")

	// Default hotkeys.
	hotkeys := []*kb.Hotkey{
		{"Pause", "sig-toggle-pause"},
		{"Control-Shift-k", "sig-toggle-pause"},
		{"Control-Shift-d", "sig-download"},
	}
	// Check (and create if needed) hotkeys config file.
	hkPath, _ := conply.GetHKPath(Bundle)
	ply.verbose.Debug1("Reading hotkeys config data: ", hkPath)
	if !conply.FileExists(hkPath) {
		ply.verbose.Debug2("Hotkeys config data not found")
		err := conply.MarshalFile(hkPath, hotkeys, true)
		if err != nil {
			ply.verbose.Fail("Failed attempt of create hotkeys config")
			return err
		} else {
			ply.verbose.Debug3("Hotkeys config was filled with default hotkeys list")
		}
	}

	// Initialize VLC player.
	var err error
	ply.verbose.Debug1("Initialize VLC")
	if ply.vlc, err = vlc.NewVlc([]string{"--quiet", "--no-video"}); err != nil {
		return err
	}
	ply.verbose.Debug2("VLC is ready")

	return nil
}

// Release player resources.
func (ply *Player) Release() error {
	err := ply.vlc.Release()
	return err
}

// Cleanup callback will call before finishing the work.
func (ply *Player) Cleanup() (err error) {
	ply.verbose.Debug1("Caught SIGTERM signal")
	ply.verbose.Debug3("Release keybinding")
	err = keybind.Release()
	if err != nil {
		return err
	}
	ply.verbose.Debug3("Release VLC player")
	err = ply.Release()
	if err != nil {
		return err
	}
	return nil
}

// Catch hotkeys signals.
func (ply *Player) Catch(signal string) error {
	now := time.Now().UnixNano()
	if now-ply.sigUtime < conply.SigUtimeMin {
		return conply.ErrMultipleCatch
	}
	ply.sigUtime = now

	ply.verbose.Debug1("caught signal: ", signal)
	switch signal {
	case "sig-toggle-pause":
		if ply.status == conply.StatusPlay {
			err := ply.Pause()
			if err != nil {
				ply.verbose.Fail("Pause failed due to error: ", err)
			} else {
				ply.verbose.Debug2("Playing paused")
			}
		} else if ply.status == conply.StatusPause {
			err := ply.Resume()
			if err != nil {
				ply.verbose.Fail("Resume failed due to error: ", err)
			} else {
				ply.verbose.Debug2("Playing resumed")
			}
		}
	case "sig-download":
		go func(ply *Player) {
			err, warn := ply.Download()
			switch {
			case err != nil:
				ply.verbose.Fail("Downloading failed with error: ", err)
			case warn != nil:
				ply.verbose.Warning(warn)
			default:
				ply.verbose.Debug1("Track has been successfully downloaded")
			}
		}(ply)
	}
	return nil
}

// Play the current track.
func (ply *Player) Play() (err error) {
	if ply.track == nil {
		return errors.New("undefined track, call SetTrack() first")
	}
	trackUrl := ply.track.GetURL()
	err = ply.vlc.PlayURL(trackUrl)
	if err != nil {
		return err
	}
	switch {
	case ply.status == conply.StatusPause:
		ply.verbose.Debug3("Instantly pause new track since current status is Pause")
		return ply.vlc.Pause()
	case ply.status == conply.StatusStop:
		ply.verbose.Debug3("Instantly stop new track since current status is Pause")
		return ply.vlc.Stop()
	default:
		ply.verbose.Debug3("Track URL: ", trackUrl)
		ply.status = conply.StatusPlay
	}
	ply.prevTrackUid = ply.trackUid
	return
}

// Stop playing.
func (ply *Player) Stop() error {
	ply.status = conply.StatusStop
	return ply.vlc.Stop()
}

// Pause playing.
func (ply *Player) Pause() error {
	if ply.status == conply.StatusPause {
		return nil
	}
	ply.status = conply.StatusPause
	return ply.vlc.Pause()
}

// Resume playing.
func (ply *Player) Resume() error {
	if ply.status == conply.StatusPlay {
		return nil
	}
	ply.status = conply.StatusPlay
	return ply.vlc.Resume()
}

// Get current status.
func (ply *Player) GetStatus() conply.Status {
	return ply.status
}

// Download the track.
func (ply *Player) Download() (error, error) {
	// Make download process safety.
	ply.muxDl.Lock()
	defer ply.muxDl.Unlock()

	chTitle := ply.channel.Title

	// Check environment.
	bundle := Bundle + conply.PS + ply.group.Title
	dlDir, err := conply.GetDlDir(bundle, chTitle)
	if err != nil {
		return err, nil
	}
	if !conply.FileExists(dlDir) {
		if err := conply.Mkdir(dlDir); err != nil {
			return err, nil
		}
	}

	// Check if track already has downloaded.
	url := ply.track.GetURL()
	dest := dlDir + conply.PS + ply.track.ComposeDlTitle() + ".mp3"

	if conply.FileExists(dest) {
		return nil, errors.New(fmt.Sprintf(`Downloading skipped due to file "%s" already exists`, dest))
	}
	ply.verbose.Debug3f("Track is ready to download:\n * source URL: %s\n * dest: %s", url, dest)

	// Download the file.
	err = conply.FileDl(url, dest)
	if err != nil {
		return err, nil
	}
	ply.verbose.Debug2("Track is successfully downloaded to ", dest)

	// Try to set ID3 tags.
	ply.verbose.Debug3("Try to set ID3 tags...")
	tag, err := id3.Open(dest)
	if err != nil {
		return err, nil
	}
	about := ply.track.Result.About
	tag.SetTitle(about.Title)
	tag.SetArtist(about.Artist)
	if len(about.Album.Title) > 0 {
		tag.SetAlbum(about.Album.Title)
		tag.SetAlbum(about.Album.ReleaseDate)
	} else {
		tag.SetAlbum(chTitle)
	}
	if err := tag.Close(); err != nil {
		return err, nil
	}
	ply.verbose.Debug3("ID3 tags has been added to track.")

	return nil, nil
}

// Sets the current track to play.
func (ply *Player) SetTrack(track *Track) {
	ply.track = track
}

// Get tree of groups/channels.
func (ply *Player) RetrieveTree() error {
	ply.cache = make(ChannelGroups, 0)

	respGroups, err := http.Get("http://101.ru/radio-top")
	if err != nil {
		return err
	}
	defer func() {
		_ = respGroups.Body.Close()
	}()

	docGroups, err := goquery.NewDocumentFromReader(respGroups.Body)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	docGroups.Find("ul.channel-groups li").Each(func(i int, selection *goquery.Selection) {
		title := strings.Trim(selection.Find("a").Text(), "\n ")
		href, exists := selection.Find("a").Attr("href")
		if exists && len(title) > 0 {
			id, _ := strconv.ParseUint(path.Base(href), 0, 64)
			ply.cache = append(ply.cache, &ChannelGroup{
				id, title, make([]*ChannelCache, 0),
			})

			wg.Add(1)
			go func(id uint64) {
				defer wg.Done()

				respChannels, err := http.Get(fmt.Sprintf("http://101.ru/radio-top/group/%d", id))
				if err != nil {
					return
				}
				defer func() {
					_ = respChannels.Body.Close()
				}()
				docChannels, err := goquery.NewDocumentFromReader(respChannels.Body)
				if err != nil {
					return
				}

				docChannels.Find("div.grid a.grid__title").Each(func(i int, selection *goquery.Selection) {
					// title := selection.Find("a").Find(".h3").Text()
					title := selection.Find("span").Text()
					href, exists := selection.Attr("href")
					if exists {
						cid, _ := strconv.ParseUint(path.Base(href), 0, 64)
						group := ply.cache.GetGroupById(id)
						group.Channels = append(group.Channels, &ChannelCache{
							cid, title,
						})
					}
				})
			}(id)
		}
	})
	wg.Wait()

	// Sort tree for pretty view.
	for _, cg := range ply.cache {
		sort.Sort(&cg.Channels)
	}
	sort.Sort(&ply.cache)

	return nil
}

// Get current track from the channel.
func (ply *Player) RetrieveTrack() error {
	ply.nextFetch = 5

	playlistUrl := fmt.Sprintf("http://101.ru/api/channel/getTrackOnAir/%d/channel/?dataFormat=json", ply.chIdx)
	response, err := http.Get(playlistUrl)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &ply.track)
	if err != nil {
		return err
	}

	for i, file := range ply.track.Result.About.Audio {
		ply.trackUid = file.TrackUid
		playUrl := file.Filename
		// Check case when we got URL without schema and domain.
		re := regexp.MustCompile(`http[s]*:(.)`)
		res := re.FindStringSubmatch(ply.track.Result.About.Audio[0].Filename)
		prefix := ""
		if res == nil {
			prefix = "http://101.ru"
		}
		playUrl = prefix + playUrl

		// Check case with wrong URL (ex: http://cdn*.101.ru/vardata/modules/musicdb/files//vardata/modules/musicdb/files/*).
		//                                                  ^                             ^^
		re = regexp.MustCompile(`(/vardata/modules/musicdb/files/)`)
		dres := re.FindAllStringSubmatch(playUrl, -1)
		if len(dres) == 2 {
			playUrl = strings.Replace(playUrl, "/vardata/modules/musicdb/files/", "", 1)
		}
		ply.track.Result.About.Audio[i].Filename = playUrl
	}

	// Calculate next fetch period. Based on the difference between current timestamp and song start timestamp.
	diff := ply.track.Result.Stat.FinishSong - ply.track.Result.Stat.ServerTime
	if diff < 5 || diff > 1800 {
		diff = 5
	} else {
		diff -= 3
	}
	ply.nextFetch = diff

	return nil
}

// Look for group and channel by channel ID.
func (ply *Player) GetByChannelId(cid uint64) (*ChannelGroup, *ChannelCache) {
	for _, g := range ply.cache {
		for _, c := range g.Channels {
			if c.Id == cid {
				return g, c
			}
		}
	}
	return &ChannelGroup{0, "Undefined", nil}, &ChannelCache{cid, "Undefined"}
}
