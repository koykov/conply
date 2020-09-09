package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"regexp"
	"strconv"
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
	Bundle      = "rockradio"
	Version     = "v0.1"
	CacheExpire = 7 * 24 * 3600
)

// Rockradio.com player.
type Player struct {
	atoken  string
	cache   ChannelsCache
	channel *Channel
	track   *Track
	chIdx   uint64

	vlc      *vlc.Vlc
	status   conply.Status
	ticks    map[string]<-chan time.Time
	signals  map[string]chan bool
	sigUtime int64
	muxDl    sync.Mutex

	verbose *v.Verbose
}

// The constructor.
func NewPlayer(verbose *v.Verbose, options map[string]interface{}) *Player {
	ply := Player{
		cache:  make(ChannelsCache),
		chIdx:  options["channel"].(uint64),
		status: conply.StatusPlay,
		ticks: map[string]<-chan time.Time{
			"track": make(chan time.Time),
			"token": make(chan time.Time),
			"next":  make(chan time.Time),
		},
		signals: map[string]chan bool{
			"next": make(chan bool),
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
		{"Control-Shift-l", "sig-next"},
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

	case "sig-next":
		ply.signals["next"] <- true
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

	chTitle := ply.cache[ply.chIdx].Title

	// Check environment.
	dlDir, err := conply.GetDlDir(Bundle, chTitle)
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

	// Check if ffmpeg installed.
	ffmpegBin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return err, nil
	}

	// Start ffmpeg and wait until it will download and convert the track.
	cmd := exec.Command(ffmpegBin, "-i", url, dest)
	if err := cmd.Start(); err != nil {
		return err, nil
	}
	if err := cmd.Wait(); err != nil {
		return err, nil
	}
	ply.verbose.Debug2("Track is successfully downloaded to ", dest)

	// Try to set ID3 tags.
	ply.verbose.Debug3("Try to set ID3 tags...")
	tag, err := id3.Open(dest)
	if err != nil {
		return err, nil
	}
	tag.SetTitle(ply.track.Title)
	tag.SetArtist(ply.track.Artist)
	if len(ply.track.Album) > 0 {
		tag.SetAlbum(ply.track.Album)
		tag.SetAlbum(ply.track.AlbumDate)
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

// Fetch fresh audio token.
func (ply *Player) RetrieveAToken() error {
	response, err := http.Get("https://www.rockradio.com")
	if err != nil {
		return err
	}

	source, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := response.Body.Close(); err != nil {
		return err
	}

	re := regexp.MustCompile(`"audio_token":"([a-z0-9]+)"`)
	res := re.FindStringSubmatch(string(source))
	if res == nil {
		return errors.New("couldn't parse remote site to retrieve audio token")
	}

	ply.atoken = res[1]
	return nil
}

// Get list of channels from remote site.
func (ply *Player) RetrieveChannels() error {
	response, err := http.Get("https://www.rockradio.com")
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return err
	}

	doc.Find("div.submenu.channels li").Each(func(i int, selection *goquery.Selection) {
		id, exists := selection.Attr("data-channel-id")
		title := selection.Find("a").Find("span").Text()
		if exists {
			cid, _ := strconv.ParseUint(path.Base(id), 0, 64)
			ply.cache[cid] = &ChannelCache{
				Id: cid, Title: title,
			}
		}
	})

	return response.Body.Close()
}

// Get chunk of tracks for nearest ~1/2h.
func (ply *Player) RetrieveTracks() error {
	if len(ply.atoken) == 0 {
		return errors.New("invalid access token")
	}

	ts := time.Now().UnixNano() / 1000000
	channelUrl := fmt.Sprintf("https://www.rockradio.com/_papi/v1/rockradio/routines/channel/%d?audio_token=%s&_=%d", ply.chIdx, ply.atoken, ts)
	response, err := http.Get(channelUrl)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var channel Channel
	err = conply.Unmarshal(string(buf), &channel)
	if err != nil {
		return err
	}
	for i, track := range channel.Tracks {
		channel.Tracks[i].Content.Assets[0].Url = "https:" + track.Content.Assets[0].Url
		channel.Length += track.Content.Length
	}

	ply.channel = &channel

	return nil
}
