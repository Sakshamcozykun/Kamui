$ErrorActionPreference = 'Stop'

if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    throw 'winget was not found. Install App Installer from the Microsoft Store, then rerun this script.'
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go 1.26+ is required. Install it from https://go.dev/dl/ and reopen PowerShell.'
}

$packages = @(
    'shinchiro.mpv',
    'yt-dlp.yt-dlp',
    'Gyan.FFmpeg'
)
foreach ($package in $packages) {
    Write-Host "Installing $package with winget..."
    winget install --exact --id $package --source winget --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) { throw "winget failed to install $package (exit $LASTEXITCODE)" }
}

Write-Host 'Installing Kamui...'
go install .
if ($LASTEXITCODE -ne 0) { throw "go install failed (exit $LASTEXITCODE)" }

Write-Host 'Setup complete. Close and reopen this terminal, then run: kamui doctor'
