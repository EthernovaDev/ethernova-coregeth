$ErrorActionPreference = "Stop"

Set-Location (Split-Path $PSScriptRoot -Parent)

$Version = Get-Content -Path "VERSION" -ErrorAction SilentlyContinue
if (-not $Version) { $Version = "dev" }

$OutDir = "dist"
if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Force -Path $OutDir | Out-Null }

$ZipName = "ethernova-$Version-windows.zip"
$ZipPath = Join-Path $OutDir $ZipName

$items = @(
    "bin\ethernova.exe",
    "bin\EthernovaNode.exe",
    "genesis-dev.json",
    "genesis-mainnet.json",
    "VERSION",
    "scripts\init-ethernova.ps1",
    "scripts\smoke-test-fees.ps1",
    "scripts\smoke-test-fees.js",
    "scripts\run-bootnode.ps1",
    "scripts\run-second-node.ps1",
    "scripts\check-peering.ps1",
    "scripts\print-genesis-fingerprint.ps1",
    "networks\mainnet\bootnodes.txt",
    "networks\mainnet\static-nodes.json",
    "networks\dev\bootnodes.txt",
    "networks\dev\static-nodes.json",
    "docs\LAUNCH.md",
    "docs\DEV.md",
    "docs\CONFIG.md",
    "LICENSE",
    "README.md",
    "RELEASE-NOTES.md"
)

Compress-Archive -Path $items -DestinationPath $ZipPath -Force

$hash = Get-FileHash -Algorithm SHA256 -Path $ZipPath
$hashLine = "$($hash.Hash)  $ZipName"
$hashPath = Join-Path $OutDir "SHA256SUMS.txt"
$hashLine | Out-File -FilePath $hashPath -Encoding ascii

Write-Host "Package: $ZipPath"
Write-Host "SHA256: $hashLine"
