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

The setup script installs Kamui, then asks which features you want. Movie
search and browser links need no extra media tools. Selecting terminal movie
playback, YouTube streaming, or music/radio installs `mpv`, `yt-dlp`, and
`ffmpeg` with `winget`. Open a new terminal after setup. If `kamui` is not
found, ensure `%USERPROFILE%\go\bin` is on your user `PATH`.

### Linux and macOS

Install Go 1.26+ and the system package manager for your platform, then:

```sh
git clone https://github.com/sakshamcozykun/Kamui
cd Kamui
sh scripts/setup.sh
```

The script installs Kamui, then shows the same feature menu as Windows. It uses
apt, dnf, pacman, or Homebrew to install playback tools only when you select a
playback feature. Package installation may ask for administrator approval.
At the feature menu, press Enter for movie search and browser links only, enter
multiple choices such as `2,4`, or type `all`.

From the repository root, `bash install.sh` checks the Go version, builds
Kamui into `$HOME/.local/bin`, then asks which features you want. It installs
`mpv`, `yt-dlp`, and `ffmpeg` only if you select terminal playback, YouTube
streaming, or music/radio. Add `$HOME/.local/bin` to `PATH` if needed.

If you choose only movie search and browser links, the media tools are skipped.
The other built-in commands remain available either way. Install playback tools
later by running `kamui doctor`.

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
kamui stream "https://..."    # resolve a supported URL and play in mpv
kamui doctor                  # check or install playback tools
```

## Commands

| Command | Purpose |
| --- | --- |
| `kamui <title>` | Show movie metadata and matching browser links |
| `kamui play <title>` | Try to play a title in `mpv`; prints a browser link if playback fails |
| `kamui select <title>` | Choose a result and print its page link |
| `kamui search <title>` | Search reachable sources without playback |
| `kamui anime <title>` | Search only anime streaming sources (including donghua) |
| `kamui music <query>` | Choose among music results and play audio or a radio mix |
| `kamui stream <url>` | Resolve direct media, YouTube, Vimeo, or Twitch URLs and play them in `mpv` |
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

### Stream resolvers

`kamui stream <url>` accepts direct MP4, WebM, Matroska, HLS (`.m3u8`), and
DASH (`.mpd`) URLs, plus supported pages from YouTube, Vimeo, and Twitch.
Platform extraction runs locally through `yt-dlp`; `mpv` receives the resolved
media URL. No external decoding API is used. Access controls and DRM are not
bypassed, and a site can still reject playback because of login, region, age,
or URL expiration requirements.

The initial platform adapters target three widely used general video services:

| Platform | Why it is included | What makes URL resolution non-trivial |
| --- | --- | --- |
| YouTube | Broad public video and live-video coverage | Playback URLs can be short-lived and served through adaptive formats or manifests; yt-dlp tracks the site's changing player logic. |
| Vimeo | Common creator, portfolio, and embedded video hosting | Privacy settings and adaptive HLS/DASH delivery can affect which media URLs are available to the current viewer. |
| Twitch | Major live-stream and video-on-demand platform | Live playback uses changing HLS playlists and session-specific access details; live and archived video URLs behave differently. |

These are supported video platforms rather than a ranking of third-party
embedded hosts. yt-dlp's extractor list is broad and changes over time; support
for any site is not guaranteed. Add another `stream.Resolver` and register it
in `internal/stream` to extend the pipeline. Use the stream command only with
media you have permission to access.

## Development

```sh
go test ./...
go build ./...
```

GitHub Actions tests and builds on Linux and Windows. Pushing a `v*` tag builds
release binaries for Windows, Linux, and macOS (amd64 and arm64).
