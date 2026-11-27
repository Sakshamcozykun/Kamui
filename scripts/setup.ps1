$ErrorActionPreference = 'Stop'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go 1.26+ is required. Install it from https://go.dev/dl/ and reopen PowerShell.'
}
$goVersion = (go env GOVERSION) -replace '^go', ''
$goMajor = [int](($goVersion -split '\.')[0] -replace '[^0-9].*$', '')
$goMinor = [int](($goVersion -split '\.')[1] -replace '[^0-9].*$', '')
if ($goMajor -lt 1 -or ($goMajor -eq 1 -and $goMinor -lt 26)) {
    throw "Go 1.26+ is required; found $goVersion. Install it from https://go.dev/dl/ and reopen PowerShell."
}

Write-Host 'Installing Kamui...'
go install .
if ($LASTEXITCODE -ne 0) { throw "go install failed (exit $LASTEXITCODE)" }

Write-Host 'Caching the source catalogue and checking the top 50 sites...'
go run . probe --max 50 --timeout 12s
if ($LASTEXITCODE -ne 0) { Write-Warning 'Initial source cache could not be completed; Kamui will populate it on first search.' }

Write-Host ''
Write-Host 'Choose the features you want:'
Write-Host '  1) Movie search and browser links (no extra media tools)'
Write-Host '  2) Movie playback in the terminal'
Write-Host '  3) YouTube video streaming'
Write-Host '  4) Music player and similar-song radio'
Write-Host 'Choose one or more numbers separated by commas, or type "all".'

while ($true) {
    $selection = Read-Host 'Features [1]'
    if ([string]::IsNullOrWhiteSpace($selection)) { $selection = '1' }
    if ($selection.Trim().ToLowerInvariant() -eq 'all') { $selection = '1,2,3,4' }
    $features = @($selection -split '[,\s]+' | Where-Object { $_ -ne '' })
    $invalid = @($features | Where-Object { $_ -notin @('1', '2', '3', '4') })
    if ($features.Count -gt 0 -and $invalid.Count -eq 0) { break }
    Write-Host 'Enter 1, 2, 3, or 4 separated by commas, or "all".' -ForegroundColor Yellow
}

$needsPlayback = @($features | Where-Object { $_ -in @('2', '3', '4') }).Count -gt 0
if ($needsPlayback) {
    $tools = @(
        @{ Name = 'mpv'; Id = 'shinchiro.mpv'; Paths = @((Join-Path $env:ProgramFiles 'MPV Player\mpv.exe'), (Join-Path $env:ProgramFiles 'mpv\mpv.exe')) },
        @{ Name = 'yt-dlp'; Id = 'yt-dlp.yt-dlp'; Paths = @((Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Links\yt-dlp.exe')) },
        @{ Name = 'ffmpeg'; Id = 'Gyan.FFmpeg'; Paths = @((Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Links\ffmpeg.exe')) }
    )
    $missingTools = @()
    foreach ($tool in $tools) {
        $installed = [bool](Get-Command $tool.Name -ErrorAction SilentlyContinue)
        if (-not $installed) {
            $installed = @($tool.Paths | Where-Object { Test-Path -LiteralPath $_ }).Count -gt 0
        }
        if ($installed) {
            Write-Host "[ok] $($tool.Name) is already installed."
        } else {
            $missingTools += $tool
        }
    }
    if ($missingTools.Count -gt 0 -and -not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw 'winget was not found. Install App Installer from the Microsoft Store, then run kamui doctor to install playback tools later.'
    }
    foreach ($tool in $missingTools) {
        Write-Host "Installing $($tool.Name) with winget..."
        winget install --exact --id $tool.Id --source winget --accept-package-agreements --accept-source-agreements
        if ($LASTEXITCODE -ne 0) { throw "winget failed to install $($tool.Name) (exit $LASTEXITCODE). Run 'kamui doctor' to retry later." }
    }
} else {
    Write-Host "Skipping playback tools. Install them later by running 'kamui doctor'."
}

Write-Host "Setup complete. Close and reopen this terminal, then run: kamui doctor"
