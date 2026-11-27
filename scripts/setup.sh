#!/usr/bin/env sh
set -eu

if ! command -v go >/dev/null 2>&1; then
  echo 'Go 1.26+ is required. Install it from https://go.dev/dl/ and rerun.' >&2
  exit 1
fi
go_version=$(go env GOVERSION | sed 's/^go//')
go_major=$(printf '%s' "$go_version" | cut -d. -f1)
go_minor=$(printf '%s' "$go_version" | cut -d. -f2 | sed 's/[^0-9].*$//')
if [ "$go_major" -lt 1 ] || { [ "$go_major" -eq 1 ] && [ "$go_minor" -lt 26 ]; }; then
  echo "Go 1.26+ is required; found $go_version." >&2
  exit 1
fi

echo 'Installing Kamui...'
go install .

echo 'Caching the source catalogue and checking the top 50 sites...'
if ! go run . probe --max 50 --timeout 12s; then
  echo 'Initial source cache could not be completed; Kamui will populate it on first search.' >&2
fi

cat <<'EOF'

Choose the features you want:
  1) Movie search and browser links (no extra media tools)
  2) Movie playback in the terminal
  3) YouTube video streaming
  4) Music player and similar-song radio

Choose one or more numbers separated by commas, or type "all".
EOF
while :; do
  printf 'Features [1]: '
  IFS= read -r selection || selection=
  selection=$(printf '%s' "$selection" | tr ',' ' ')
  if [ -z "$(printf '%s' "$selection" | tr -d '[:space:]')" ]; then selection=1; fi
  if [ "$selection" = "all" ]; then selection='1 2 3 4'; fi
  playback=0
  valid=1
  for feature in $selection; do
    case "$feature" in
      1) ;;
      2|3|4) playback=1 ;;
      *) valid=0 ;;
    esac
  done
  if [ "$valid" -eq 1 ]; then break; fi
  echo 'Enter 1, 2, 3, or 4 separated by commas, or "all".' >&2
done

if [ "$playback" -eq 1 ]; then
  missing=''
  for tool in mpv yt-dlp ffmpeg; do
    if ! command -v "$tool" >/dev/null 2>&1; then
      missing="$missing $tool"
    fi
  done
  if [ -z "$missing" ]; then
    echo 'Playback tools are already installed.'
  else
    OS=$(uname -s)
    case "$OS" in
      Darwin)
        command -v brew >/dev/null 2>&1 || { echo 'Install Homebrew first: https://brew.sh/' >&2; exit 1; }
        # shellcheck disable=SC2086
        brew install $missing
        ;;
      Linux*)
        if command -v apt-get >/dev/null 2>&1; then
          if [ "$(id -u)" -eq 0 ]; then apt-get update && apt-get install -y $missing
          else sudo apt-get update && sudo apt-get install -y $missing; fi
        elif command -v dnf >/dev/null 2>&1; then
          if [ "$(id -u)" -eq 0 ]; then dnf install -y $missing
          else sudo dnf install -y $missing; fi
        elif command -v pacman >/dev/null 2>&1; then
          if [ "$(id -u)" -eq 0 ]; then pacman -S --needed --noconfirm $missing
          else sudo pacman -S --needed --noconfirm $missing; fi
        else
          echo 'No supported package manager found (apt, dnf, or pacman).' >&2
          exit 1
        fi
        ;;
      *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
    esac
  fi
else
  echo 'Skipping playback tools. Install them later by running: kamui doctor'
fi

echo 'Setup complete. Reopen your terminal and run: kamui doctor'
