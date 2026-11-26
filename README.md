# Kamui

> Media discovery and playback from your terminal.

Kamui is a cross-platform Go CLI for finding media sources, checking whether
they respond, viewing movie metadata, and playing supported links with `mpv`.
Its source catalogue is adapted from the community-maintained [FMHY
directory](https://fmhy.net).

## Features

- Search and rank results from reachable sources
- Open a selected result in your browser or try playback in `mpv`
- Search and play audio through `yt-dlp` and `mpv`, including a similar-song radio option
- Check source reachability and cache probe results
- Show OMDb ratings, runtime, genre, plot, and poster previews
- Save a watchlist and recent search history
- Install and diagnose playback tools from the CLI

## Install

Go 1.26+ is required to build from source.

### Windows

Install Go from [go.dev/dl](https://go.dev/dl/) and reopen PowerShell. Then:

```powershell
git clone https://github.com/sakshamcozykun/Kamui
cd Kamui
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\setup.ps1
```

The setup script installs Kamui plus `mpv`, `yt-dlp`, and `ffmpeg` with
`winget`. Open a new terminal after setup. If `kamui` is not found, ensure
`%USERPROFILE%\go\bin` is on your user `PATH`.

### Linux and macOS

Install Go 1.26+ and the system package manager for your platform, then:

```sh
git clone https://github.com/sakshamcozykun/Kamui
cd Kamui
sh scripts/setup.sh
```

The script uses apt, dnf, pacman, or Homebrew to install playback tools and
then installs Kamui. Package installation may ask for administrator approval.

To install the latest Go module without cloning:

```sh
go install github.com/sakshamcozykun/kamui@latest
```

That installs the Go CLI only. Install `mpv`, `yt-dlp`, and `ffmpeg` separately
if you want playback features.

## Quick start

```sh
kamui                         # command overview
kamui "Dune"                  # show movie details and browser links
kamui play "Dune"             # attempt playback
kamui select "Dune"           # choose a result and print its browser link
kamui music "artist - track"  # find tracks, then play or start radio
kamui doctor                  # check or install playback tools
```

## Commands

| Command | Purpose |
| --- | --- |
| `kamui <title>` | Show movie metadata and matching browser links |
| `kamui play <title>` | Try to play a title in `mpv`; prints a browser link if playback fails |
| `kamui select <title>` | Choose a result and print its page link |
| `kamui search <title>` | Search reachable sources without playback |
| `kamui music <query>` | Choose among music results and play audio or a radio mix |
| `kamui sources` | List sources, with optional section and starred filters |
| `kamui probe` | Check reachability; cached results can be refreshed |
| `kamui doctor` | Check dependencies and offer to install missing tools |
| `kamui library add/remove <title>` | Manage your watchlist |
| `kamui library watchlist/history` | View saved titles and recent searches |
| `kamui config` | Show configuration status |
| `kamui poster` | Configure the OMDb API key |

Use `kamui <command> --help` for flags and examples. Put flags before the title.

## Configuration

Movie ratings and details use OMDb. Get an API key at
[omdbapi.com/apikey.aspx](https://www.omdbapi.com/apikey.aspx), then run:

```sh
kamui poster
```

Kamui saves the key in your per-user config. Alternatively, set
`OMDB_API_KEY` in the environment.

Wiki and probe caches are kept under the operating system's user cache
directory. Watchlist and search history are stored in the user config folder.

## Search and playback notes

Kamui uses generic server-rendered HTML search patterns rather than a scraper
for every individual site. JavaScript-only result pages may not return results.
`kamui doctor` checks that `mpv`, `yt-dlp`, and `ffmpeg` are installed and
available. The poster preview uses terminal text rendering; its appearance
depends on terminal color and character support.

## Development

```sh
go test ./...
go build ./...
```

GitHub Actions tests and builds on Linux and Windows. Pushing a `v*` tag builds
release binaries for Windows, Linux, and macOS (amd64 and arm64).
