$ErrorActionPreference = "Stop"

Set-Location (Split-Path $PSScriptRoot -Parent)

$Version = Get-Content -Path "VERSION" -ErrorAction SilentlyContinue
if (-not $Version) { $Version = "dev" }

$OutDir = "dist"
if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Force -Path $OutDir | Out-Null }

$ZipName = "ethernova-$Version-windows.zip"
$ZipPath = Join-Path $OutDir $ZipName
$StageDir = Join-Path $OutDir "stage-windows"

if (Test-Path $StageDir) { Remove-Item -Recurse -Force $StageDir }
New-Item -ItemType Directory -Force -Path $StageDir | Out-Null

# Copy binaries to stage root
Copy-Item "bin\ethernova.exe" "$StageDir\ethernova.exe" -Force
Copy-Item "bin\EthernovaNode.exe" "$StageDir\EthernovaNode.exe" -Force

# Copy genesis files to stage root
Copy-Item "genesis-dev.json" "$StageDir\genesis-dev.json" -Force
Copy-Item "genesis-mainnet.json" "$StageDir\genesis-mainnet.json" -Force

# Copy supporting files/directories
Copy-Item "VERSION" "$StageDir\VERSION" -Force
Copy-Item "LICENSE" "$StageDir\LICENSE" -Force
Copy-Item "README.md" "$StageDir\README.md" -Force
Copy-Item "RELEASE-NOTES.md" "$StageDir\RELEASE-NOTES.md" -Force
Copy-Item "docs" "$StageDir\docs" -Recurse -Force
Copy-Item "scripts" "$StageDir\scripts" -Recurse -Force
Copy-Item "networks" "$StageDir\networks" -Recurse -Force

if (Test-Path $ZipPath) { Remove-Item $ZipPath -Force }
Compress-Archive -Path (Join-Path $StageDir "*") -DestinationPath $ZipPath -Force

$hash = Get-FileHash -Algorithm SHA256 -Path $ZipPath
$hashLine = "$($hash.Hash)  $ZipName"
$hashPath = Join-Path $OutDir "SHA256SUMS.txt"
$hashLine | Out-File -FilePath $hashPath -Encoding ascii

Write-Host "Package: $ZipPath"
Write-Host "SHA256: $hashLine"
