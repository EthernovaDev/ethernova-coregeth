$ErrorActionPreference = "Stop"

param(
    [string]$DataDir = ""
)

function Resolve-FirstPath {
    param([string[]]$Candidates)
    foreach ($path in $Candidates) {
        if (Test-Path $path) {
            return (Resolve-Path $path).Path
        }
    }
    return $null
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir "..")).Path

$Ethernova = Resolve-FirstPath @(
    (Join-Path $RepoRoot "bin\\ethernova.exe"),
    (Join-Path $RepoRoot "ethernova.exe")
)
if (-not $Ethernova) { throw "ethernova.exe not found (expected bin\\ethernova.exe or root)." }

$GenesisUpgrade = Resolve-FirstPath @(
    (Join-Path $RepoRoot "genesis\\genesis-upgrade-60000.json"),
    (Join-Path $RepoRoot "genesis-upgrade-60000.json")
)
if (-not $GenesisUpgrade) { throw "genesis-upgrade-60000.json not found." }

if (-not $DataDir) {
    $DataDir = Join-Path $RepoRoot "data-mainnet"
}

Write-Host "Applying Fork60000 config upgrade..."
Write-Host "NOTE: Do NOT replace the genesis file in your datadir."
Write-Host "      This command updates the stored chain config in-place."

& $Ethernova --datadir $DataDir init $GenesisUpgrade
exit $LASTEXITCODE
