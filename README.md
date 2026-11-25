# kamui

> Pull streams from any dimension, straight into your terminal.

A Go learning project, Go CLI that searches free media sites listed in [FMHY](https://fmhy.net), then
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

Requires Go 1.26+.

### Windows prerequisite

Install Go from [go.dev/dl](https://go.dev/dl/), then close and reopen
Command Prompt or PowerShell so the Go commands are added to `PATH`. Verify
the installation:

```powershell
go version
```

If Windows still cannot find `go`, restart the terminal (or sign out and back
in) and try again.

After installation, if `kamui` is still not recognized, add
`%USERPROFILE%\go\bin` to your user `PATH`, then reopen the terminal.

```bash
git clone https://github.com/sakshamcozykun/Kamui
cd Kamui
go install .
```

Go downloads and builds Kamui's Go dependencies automatically, including the
terminal image renderer used for movie posters. No separate `catimg` install
command is needed. Full-resolution posters require a terminal that supports
Kitty, iTerm2, or Sixel graphics; other terminals may show a text fallback.

To run the checkout directly while developing instead of using the installed
binary, use `go run .` from the repository directory.

This installs `kamui` globally in Go's bin directory. Make sure that directory
is on your `PATH` (`go env GOPATH`/`bin`, or `go env GOBIN` when it is set),
then open a new terminal and run:

Movie ratings and OMDb poster metadata require an OMDb API key. After
installing Kamui, run `kamui poster`; it prompts for your key and saves it in
your per-user Kamui config. Get a key at
[OMDb](https://www.omdbapi.com/apikey.aspx). The `OMDB_API_KEY` environment
variable can also be used instead of the saved key.

```bash
kamui
```

To install the latest version directly without cloning the repository:

```bash
go install github.com/sakshamcozykun/kamui@latest
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
kamui doctor                           # check mpv, yt-dlp, and ffmpeg
kamui music "artist - track"           # search and play audio
kamui music "https://example/..."      # play audio from a supported URL
kamui config                           # show configuration status
kamui library add "dune"               # add to watchlist
kamui library remove "dune"            # remove from watchlist
kamui library watchlist                # show saved titles
kamui library history                  # show recent searches
```

Wiki pages are cached under `~/.cache/kamui/fmhy/` and revalidated with
`If-None-Match`, so repeat runs cost one conditional request instead of a full
download. Probe results go to the user cache directory. Watchlist and recent
search history are stored under the user config directory.

GitHub Actions tests and builds on Linux and Windows. Pushing a `v*` tag builds
release binaries for Windows, Linux, and macOS (amd64 and arm64).

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
