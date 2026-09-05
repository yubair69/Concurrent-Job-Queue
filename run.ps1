# PixelForge - build the frontend if needed, then start the server.
# Usage:  .\run.ps1            (build if stale, then run)
#         .\run.ps1 -Fresh     (force a frontend rebuild)

param(
    [switch]$Fresh,
    [int]$Port = 8080,
    [int]$Workers = 4
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

function Require-Command($name, $hint) {
    if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
        Write-Host "Missing '$name'. $hint" -ForegroundColor Red
        exit 1
    }
}

Require-Command "go" "Install Go from https://go.dev/dl/"
Require-Command "npm" "Install Node.js from https://nodejs.org/"

# 1. Frontend dependencies
if (-not (Test-Path "web/node_modules")) {
    Write-Host "==> Installing frontend dependencies..." -ForegroundColor Cyan
    Push-Location web
    npm install
    if ($LASTEXITCODE -ne 0) { Pop-Location; exit 1 }
    Pop-Location
}

# 2. Frontend build - skipped when dist is newer than every source file
$needsBuild = $Fresh -or -not (Test-Path "web/dist/index.html")
if (-not $needsBuild) {
    $built = (Get-Item "web/dist/index.html").LastWriteTime
    $newestSource = Get-ChildItem "web/src", "web/index.html", "web/tailwind.config.ts", "web/vite.config.ts" -Recurse -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($newestSource -and $newestSource.LastWriteTime -gt $built) { $needsBuild = $true }
}

if ($needsBuild) {
    Write-Host "==> Building frontend..." -ForegroundColor Cyan
    Push-Location web
    npm run build
    if ($LASTEXITCODE -ne 0) { Pop-Location; exit 1 }
    Pop-Location
} else {
    Write-Host "==> Frontend already up to date (use -Fresh to force a rebuild)" -ForegroundColor DarkGray
}

# 3. Server
$env:GOTASK_PORT = "$Port"
$env:GOTASK_WORKER_COUNT = "$Workers"

if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue)) {
    Write-Host "note: ffmpeg not found - video jobs will run but produce simulated output" -ForegroundColor DarkYellow
}

Write-Host "==> PixelForge running at http://localhost:$Port  (Ctrl+C to stop)" -ForegroundColor Green
go run ./cmd/server
