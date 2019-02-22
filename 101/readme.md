101.ru player
=============

A part of a group of console players. Provide a possibility to listen 101.ru station.

## Installation

Make sure you have downloaded *conply* to your *$GOPATH/src*, see [readme](../readme.md) for exact instruction.

Then run:
```bash
go build -o $GOPATH/bin/101ply github.com/koykov/conply/101ply
```

As a result you should have *rrply* binary in the corresponding directory.

## Usage

Simple run:
```bash
$GOPATH/bin/101ply
```

The player will show you a list of a groups/channels and ask you about group and channel IDs. Just type the most interesting and enjoy the music.

Check
```bash
$GOPATH/bin/101ply --help
```
to see all possibility options.