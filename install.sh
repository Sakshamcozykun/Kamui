#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'install: %s\n' "$*" >&2
  exit 1
}

command -v go >/dev/null 2>&1 || fail "Go 1.26 or newer is required by this Kamui checkout. Install it from https://go.dev/dl/ and rerun this script."
go_version="$(go env GOVERSION | sed 's/^go//')"
go_major="${go_version%%.*}"
go_rest="${go_version#*.}"
go_minor="${go_rest%%.*}"
if (( go_major < 1 || (go_major == 1 && go_minor < 26) )); then
  fail "Go 1.26 or newer is required by this Kamui checkout; found $go_version"
fi

if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
  brew install mpv yt-dlp ffmpeg
elif command -v apt-get >/dev/null 2>&1; then
  if [[ "$(id -u)" -eq 0 ]]; then
    apt-get update
    apt-get install -y mpv yt-dlp ffmpeg
  elif command -v sudo >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y mpv yt-dlp ffmpeg
  else
    fail "apt-get is available, but sudo is not; install mpv, yt-dlp, and ffmpeg first"
  fi
elif command -v dnf >/dev/null 2>&1; then
  if [[ "$(id -u)" -eq 0 ]]; then dnf install -y mpv yt-dlp ffmpeg
  elif command -v sudo >/dev/null 2>&1; then sudo dnf install -y mpv yt-dlp ffmpeg
  else fail "dnf is available, but sudo is not; install mpv, yt-dlp, and ffmpeg first"; fi
elif command -v pacman >/dev/null 2>&1; then
  if [[ "$(id -u)" -eq 0 ]]; then pacman -S --needed --noconfirm mpv yt-dlp ffmpeg
  elif command -v sudo >/dev/null 2>&1; then sudo pacman -S --needed --noconfirm mpv yt-dlp ffmpeg
  else fail "pacman is available, but sudo is not; install mpv, yt-dlp, and ffmpeg first"; fi
else
  fail "unsupported package manager; install mpv, yt-dlp, and ffmpeg, then rerun this script"
fi

install_dir="${KAMUI_INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$install_dir"
go build -o "$install_dir/kamui" .
printf 'Kamui installed at %s/kamui\n' "$install_dir"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Add this directory to PATH: %s\n' "$install_dir" ;;
esac
