Rockradio.com player
====================

A part of a group of console players. Provide a possibility to listen rockradio.com station without 30 min limit.

## Requirements

Since rockradio shares a tracks in *mp4* format the player requires installed [ffmpeg](https://www.ffmpeg.org/) to convert the track to *mp3* format.
This also allows to add ID3 tags to the downloaded track.

## Installation

Make sure you have downloaded *conply* to your *$GOPATH/src*, see [readme](../readme.md) for exact instruction.

Then run:
```bash
go build -o $GOPATH/bin/rrply github.com/koykov/conply/rockradio
```

As a result you should have *rrply* binary in the corresponding directory.

## Usage

Simple run:
```bash
$GOPATH/bin/rrply
```

The player, after short delay, will show you a list of a channels and ask you about channel ID. Just type the most interesting and enjoy the music.

Check
```bash
$GOPATH/bin/rrply --help
```
to see all possibility options.