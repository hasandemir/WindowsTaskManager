# Windows Task Manager — production build script
# Produces a single optimized exe with no console window.

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location $Root

$Out = Join-Path $Root "wtm.exe"
$Module = "./cmd/wtm"

Write-Host "==> tidying modules"
go mod tidy

Write-Host "==> formatting"
go fmt ./...

Write-Host "==> vet"
go vet ./...

Write-Host "==> building $Out"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

$ldflags = "-s -w -H windowsgui -X main.version=1.0.0"
go build -trimpath -ldflags $ldflags -o $Out $Module

if (Test-Path $Out) {
    $size = (Get-Item $Out).Length / 1MB
    Write-Host ("==> done: {0} ({1:N1} MB)" -f $Out, $size)
} else {
    Write-Error "build failed"
    exit 1
}
