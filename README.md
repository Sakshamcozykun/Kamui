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
| Concurrent reachability probe + tier fingerprinting | done |
| Generic parallel search across live sites | done |
| TMDb title resolution | planned |
| `mpv` playback | planned |

```
$ kamui probe --max 12
STATUS   TIER        CODE  MS    HOST
alive    p-stream    200   3404  zenox.lol
alive    plain       505   618   arrowtv.net
blocked  cloudflare  403   561   popcornmovies.st
dead     unknown     -     timeout  cinelite.xyz
```

```
$ kamui search --max 20 "toy story 4"
searching 14 live sites...
SCORE  SITE       TITLE                           URL
1.80   Reelix     Toy Story 4 7.5 • 2019 • Movie  https://reelix.ac/movie/toy-story-4-...
1.30   Chillflix  Toy Story 4 7.5 · 2019 · Movie  https://chillflix.lol/movie/301528
1.00   ArrowTV    ⭐ 7.5 Toy Story 4 2019         https://arrowtv.net/movie/301528
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

`kamui` uses Go's standard `flag` package, which stops parsing at the first
positional argument. **Put flags before the title.**

```bash
kamui sources                          # every listed site
kamui sources --starred                # FMHY's top picks only
kamui sources --section anime          # filter by section name
kamui sources --max 10                 # cap the output

kamui probe                            # check every site (cached for 12h)
kamui probe --section multi-server     # check one section
kamui probe --refresh                  # check again, ignoring the cache
kamui probe --json                     # machine-readable output

kamui search "toy story 4"             # parallel search across live sites
kamui search --max 30 --debug "dune 2" # limit sites, trace each attempt
```

Wiki pages are cached under `~/.cache/kamui/fmhy/` and revalidated with
`If-None-Match`, so repeat runs cost one conditional request instead of a full
download. Probe results go to `~/.cache/kamui/probe.json`.

## Layout

```
kamui/
├── main.go                  # thin entry point
├── cmd/                     # subcommands
│   ├── root.go              # dispatch
│   ├── common.go            # cache dir, source filtering
│   ├── sources.go
│   ├── probe.go
│   └── search.go
└── internal/
    ├── fmhy/                # wiki fetching, caching, parsing
    ├── probe/               # concurrent HTTP checks, tier fingerprints
    └── search/              # concurrent generic search
```

## How search works

There is no per-site scraper. For each live source, kamui tries four common
search URL shapes (`/search?q=`, `/?s=`, `/search/<slug>`, `/search?keyword=`),
then looks at server-rendered `<a>` tags whose **visible text** contains every
word of the query. First shape with a match wins.

That means plain HTML sites work with zero configuration, and JavaScript-only
sites return nothing. That is a deliberate trade: it stays honest and needs no
new code when FMHY adds a site. Site-specific adapters are the next step for
the families that the probe identifies as sharing an open-source codebase.

## Notes

A personal project for learning Go: packages and interfaces, error handling,
HTTP caching, goroutines, worker pools, and cancellation. Built in the open,
a layer at a time.
