package main

import (
	"fmt"

	"github.com/koykov/conply"
)

type Track struct {
	Status    uint64 `json:"status"`
	Message   string `json:"message"`
	Result    Result `json:"result"`
	ErrorCode uint64 `json:"errorCode"`
}

type Result struct {
	Short Short `json:"short"`
	Stat  Stat  `json:"stat"`
	Poll  bool  `json:"poll"`
}

type Short struct {
	Title             string `json:"title"`
	TitleTrack        string `json:"titleTrack"`
	TitleExecutor     string `json:"titleExecutor"`
	TitleExecutorFull string `json:"titleExecutorFull"`
	UidTrack          uint64 `json:"uidTrack"`
	MdbUidTrack       uint64 `json:"mdbUidTrack"`
	MdbUidExecutor    uint64 `json:"mdbUidExecutor"`
	MdbExecutorModer  string `json:"mdbExecutorModer"`
	MdbTrackModer     int    `json:"mdbTrackModer"`
	ContainExecutors  bool   `json:"containExecutors"`
	Cover             struct {
		CoverOriginal string `json:"coverOriginal"`
		CoverHTTP     string `json:"coverHTTP"`
		Cover50       string `json:"cover50"`
		Cover100      string `json:"cover100"`
		Cover150      string `json:"cover150"`
		Cover200      string `json:"cover200"`
		Cover250      string `json:"cover250"`
		Cover300      string `json:"cover300"`
		Cover350      string `json:"cover350"`
		Cover400      string `json:"cover400"`
		Uid           int    `json:"uid"`
	} `json:"cover"`
	ITunes struct {
		Url   string `json:"url"`
		Price string `json:"price"`
	} `json:"iTunes"`
	Album Album `json:"album"`

	Sample          string        `json:"sample"`
	Audiofile       string        `json:"audiofile"`
	Duration        string        `json:"duration"`
	StatusReady     int           `json:"statusReady"`
	Module          string        `json:"module"`
	Isgroup         int           `json:"isgroup"`
	IsNewTrack      bool          `json:"isNewTrack"`
	ExecutorsWorked []interface{} `json:"executorsWorked"`
	ModerTrack      int           `json:"moderTrack"`
	ModerExecutor   string        `json:"moderExecutor"`
	TimeUploadTrack int           `json:"timeUploadTrack"`
	HasVideo        bool          `json:"hasVideo"`
	Advertising     int           `json:"advertising"`
}

type Album struct {
	Uid        uint64 `json:"uid"`
	AlbumTitle string `json:"albumTitle"`
	Year       string `json:"year"`
}

type Stat struct {
	StartSong           uint64 `json:"startSong"`
	FinishSong          uint64 `json:"finishSong"`
	ServerTime          uint64 `json:"serverTime"`
	LastTime            int    `json:"lastTime"`
	ListenAuthUsers     int    `json:"listenAuthUsers"`
	ListenNoAuthUsers   int    `json:"listenNoAuthUsers"`
	ListenAllUsers      int    `json:"listenAllUsers"`
	StartSongTimeString string `json:"startSongTimeString"`
	StartSongDateString string `json:"startSongDateString"`
	StartSongYear       string `json:"startSongYear"`
}

// ComposeTitle builds track's title to build a download path.
func (t *Track) ComposeTitle() string {
	about := t.Result.Short
	return fmt.Sprintf("%s - %s [%s] - %s", about.TitleExecutor, about.Title, about.Album.AlbumTitle, t.GetDurationStr())
}

// ComposeDlTitle build track's title to build a download path.
func (t *Track) ComposeDlTitle() string {
	about := t.Result.Short
	return fmt.Sprintf("%s - %s", about.TitleExecutor, about.Title)
}

// GetDurationStr returns duration as formatted string like mm:ss.
func (t *Track) GetDurationStr() string {
	diff := t.Result.Stat.FinishSong - t.Result.Stat.ServerTime
	return conply.FormatTime(diff)
}

// GetURL returns track's URL to play or download.
func (t *Track) GetURL() string {
	return t.Result.Short.Audiofile
}
