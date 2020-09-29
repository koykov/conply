# Xradio

A part of a group of console players. Provide a possibility to listen the following stations without 30 min limit:
* [rockradio.com](https://www.rockradio.com)
* [jazzradio.com](https://www.jazzradio.com)
* [zenradio.com](https://www.zenradio.com)
* [radiotunes.com](https://www.radiotunes.com)
* [classicalradio.com](https://www.classicalradio.com)

## Requirements

Since these stations shares a tracks in *mp4* format the player requires installed [ffmpeg](https://www.ffmpeg.org/) to convert the track to *mp3* format.
This also allows to add ID3 tags to the downloaded track.

## Installation

Make sure you have downloaded *conply* to your *$GOPATH/src*, see [readme](../readme.md) for exact instruction.

Then run:
```bash
go build -o $GOPATH/bin/xradio github.com/koykov/conply/xradio
```

As a result you should have *xradio* binary in the corresponding directory.

## Usage

Simple run:
```bash
$GOPATH/bin/xradio <station>
```
for example
```bash
$GOPATH/bin/xradio jazzradio.com
```

The player, after short delay, will show you a list of the channels of given station and ask you about channel ID.
Just type the most interesting channel ID and enjoy the music.

You may use *xradio* more handy. Run the following commands:
```bash
cd $GOPATH/bin
xradio generate
```
and see output
```bash
generate alias rockradio for station https://www.rockradio.com
generate alias jazzradio for station https://www.jazzradio.com
generate alias classicradio for station https://www.classicalradio.com
generate alias radiotunes for station https://www.radiotunes.com
generate alias zenradio for station https://www.zenradio.com
```

Then just run
```bash
$GOPATH/bin/rockradio
```
instead of
```bash
$GOPATH/bin/xradio rockradio.com
```

Check option **--help** to see all possibility options.
