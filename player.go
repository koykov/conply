package conply

import (
	"errors"
	"os"
)

type Status int

const (
	StatusPlay Status = iota
	StatusPause
	StatusStop

	PS = string(os.PathSeparator)

	SigUtimeMin = 500000
)

var (
	ErrMultipleCatch = errors.New("multiple key press caught")
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
	Download() (error, error)
	Catch(signal string) error
	Cleanup() error
}
