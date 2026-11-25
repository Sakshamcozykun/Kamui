#!/usr/bin/env sh
set -eu

if ! command -v go >/dev/null 2>&1; then
  echo 'Go 1.26+ is required. Install it from https://go.dev/dl/ and rerun.' >&2
  exit 1
fi

OS=$(uname -s)
case "$OS" in
  Darwin)
    command -v brew >/dev/null 2>&1 || { echo 'Install Homebrew first: https://brew.sh/' >&2; exit 1; }
    brew install mpv yt-dlp ffmpeg
    ;;
  Linux*)
    if command -v apt-get >/dev/null 2>&1; then
      sudo apt-get update
      sudo apt-get install -y mpv yt-dlp ffmpeg
    elif command -v dnf >/dev/null 2>&1; then
      sudo dnf install -y mpv yt-dlp ffmpeg
    elif command -v pacman >/dev/null 2>&1; then
      sudo pacman -Sy --needed mpv yt-dlp ffmpeg
    else
      echo 'No supported package manager found (apt, dnf, or pacman).' >&2
      exit 1
    fi
    ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

go install .
echo 'Setup complete. Reopen your terminal, then run: kamui doctor'
