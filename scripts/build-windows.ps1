$ErrorActionPreference = "Stop"

Write-Host "Building ethernova.exe (Windows amd64, CGO disabled)..."

Set-Location (Split-Path $PSScriptRoot -Parent)

$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Force -Path "bin" | Out-Null }

go build -o "bin\ethernova.exe" .\cmd\geth
go build -o "bin\evmcheck.exe" .\cmd\evmcheck

Write-Host "Built bin\ethernova.exe and bin\evmcheck.exe"
