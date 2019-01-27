package conply

import "os"

type Status int

const (
	StatusPlay Status = iota
	StatusPause
	StatusStop

	PS = string(os.PathSeparator)
)

// The player interface.
type Player interface {
	Init() error
	Release() error
	Play() error
	Stop() error
	Pause() error
	Resume() error
	GetStatus() Status
	Download() error
	Catch(signal string) error
	Cleanup() error
}
