$ErrorActionPreference = "Stop"

Write-Host "Building ethernova.exe (Windows)..."

Set-Location (Split-Path $PSScriptRoot -Parent)

$mingw = "C:\msys64\mingw64\bin"
if (-not (Test-Path $mingw)) {
    $mingw = "C:\ProgramData\chocolatey\lib\mingw\tools\install\mingw64\bin"
}

$env:PATH = "$mingw;$env:PATH"
$env:CC = Join-Path $mingw "gcc.exe"
$env:CGO_ENABLED = "1"

if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Force -Path "bin" | Out-Null }

go build -o "bin\ethernova.exe" .\cmd\geth

Write-Host "Built bin\ethernova.exe"
