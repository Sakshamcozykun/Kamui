# kamui

> Pull streams from any dimension, straight into your terminal.

A Go CLI that searches free media sites listed in [FMHY](https://fmhy.net), then
plays the result in `mpv`. Named after Obito's jutsu: pulling something out of
one dimension into another.

## Status

Early. Building in public, one layer at a time.

| Layer | State |
| --- | --- |
| Wiki fetch + ETag cache | done |
| Markdown → source list parser | done |
| Adapter layer + reachability probe | next |
| TMDb title resolution | planned |
| Parallel search | planned |
| `mpv` playback | planned |

```
$ kamui sources --section multi-server --max 5
STREAMING SITES / MULTI-SERVER
  ★  1Shows      http://1shows.org
  ★  AniCine     https://anicine.sbs/
  ★  Primeshows  https://www.primeshows.org/
  ★  ZetMoon     https://zetmoon.live/
  ★  TonkaCine   https://tonkacine.watch/

5 sources
```

## What FMHY is, and isn't

FMHY is a community-curated wiki of **links to websites**. It tells you what
exists and, via stars and ordering, which ones people rate. It is not an API:
it does not know how to search any of those sites.

That distinction sets the design. FMHY supplies **discovery and ranking**;
kamui supplies **execution and health tracking**. A third of the listed sites
are typically dead or blocked at any given moment, so "the wiki updates itself,
therefore the tool never breaks" is not something this project claims.

## Install

Requires Go 1.22+.

```bash
git clone https://github.com/sakshamcozykun/Kamui
cd Kamui
go build -o kamui .
```

`mpv`, `yt-dlp` and `ffmpeg` become required once playback lands.

## Usage

```bash
kamui sources                 # every listed site
kamui sources --starred       # FMHY's top picks only
kamui sources --section anime # filter by section name
kamui sources --max 10        # cap the output
```

Wiki pages are cached under `~/.cache/kamui/fmhy/` and revalidated with
`If-None-Match`, so repeat runs cost one conditional request instead of a full
download.

## Layout

```
kamui/
├── main.go              # CLI entry point and commands
└── internal/
    └── fmhy/            # wiki fetching, caching, parsing
        ├── fetch.go
        ├── parse.go
        └── testdata/    # miniature wiki used by tests
```

## Notes

A personal project for learning Go: packages and interfaces, error handling,
HTTP caching, and goroutines. Built in the open, a layer at a time.
