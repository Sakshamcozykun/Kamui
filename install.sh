#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'install: %s\n' "$*" >&2
  exit 1
}

command -v go >/dev/null 2>&1 || fail "Go 1.26 or newer is required. Install it from https://go.dev/dl/ and rerun this script."
go_version="$(go env GOVERSION | sed 's/^go//')"
go_major="${go_version%%.*}"
go_rest="${go_version#*.}"
go_minor="${go_rest%%.*}"
if (( go_major < 1 || (go_major == 1 && go_minor < 26) )); then
  fail "Go 1.26 or newer is required; found $go_version"
fi

install_dir="${KAMUI_INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$install_dir"
go build -o "$install_dir/kamui" .
printf 'Kamui installed at %s/kamui\n' "$install_dir"

cat <<'EOF'

Choose the features you want:
  1) Movie search and browser links (no extra media tools)
  2) Movie playback in the terminal
  3) YouTube video streaming
  4) Music player and similar-song radio

Choose one or more numbers separated by commas, or type "all".
EOF
while true; do
  read -r -p 'Features [1]: ' selection || selection=
  selection="${selection//,/ }"
  selection="$(printf '%s' "$selection" | xargs)"
  [[ -n "$selection" ]] || selection=1
  [[ "$selection" == "all" ]] && selection='1 2 3 4'
  valid=1
  playback=0
  for feature in $selection; do
    case "$feature" in
      1) ;;
      2|3|4) playback=1 ;;
      *) valid=0 ;;
    esac
  done
  if (( valid == 1 )); then break; fi
  printf 'Enter 1, 2, 3, or 4 separated by commas, or "all".\n' >&2
done

if (( playback == 1 )); then
  missing=()
  for tool in mpv yt-dlp ffmpeg; do
    if ! command -v "$tool" >/dev/null 2>&1; then
      missing+=("$tool")
    fi
  done
  if (( ${#missing[@]} == 0 )); then
    printf 'Playback tools are already installed.\n'
  elif [[ "$(uname -s)" == "Darwin" ]]; then
    command -v brew >/dev/null 2>&1 || fail 'Install Homebrew first: https://brew.sh/'
    brew install "${missing[@]}"
  elif command -v apt-get >/dev/null 2>&1; then
    if [[ "$(id -u)" == 0 ]]; then
      apt-get update
      apt-get install -y "${missing[@]}"
    elif command -v sudo >/dev/null 2>&1; then
      sudo apt-get update
      sudo apt-get install -y "${missing[@]}"
    else
      fail 'apt-get is available, but sudo is not; install the missing playback tools yourself.'
    fi
  elif command -v dnf >/dev/null 2>&1; then
    if [[ "$(id -u)" == 0 ]]; then dnf install -y "${missing[@]}"
    elif command -v sudo >/dev/null 2>&1; then sudo dnf install -y "${missing[@]}"
    else fail 'dnf is available, but sudo is not; install the missing playback tools yourself.'; fi
  elif command -v pacman >/dev/null 2>&1; then
    if [[ "$(id -u)" == 0 ]]; then pacman -S --needed --noconfirm "${missing[@]}"
    elif command -v sudo >/dev/null 2>&1; then sudo pacman -S --needed --noconfirm "${missing[@]}"
    else fail 'pacman is available, but sudo is not; install the missing playback tools yourself.'; fi
  else
    fail 'No supported package manager found. Install mpv, yt-dlp, and ffmpeg yourself, then run kamui doctor.'
  fi
else
  printf "Skipping playback tools. Install them later by running: kamui doctor\n"
fi

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Add this directory to PATH: %s\n' "$install_dir" ;;
esac
printf 'Setup complete. Reopen your terminal, then run: kamui doctor\n'
