param(
    [string]$DataDir = ""
)

$ErrorActionPreference = "Stop"

function Resolve-FirstPath {
    param([string[]]$Candidates)
    foreach ($path in $Candidates) {
        if (Test-Path $path) {
            return (Resolve-Path $path).Path
        }
    }
    return $null
}

function Write-Command {
    param([string]$Exe, [string[]]$CmdArgs)
    if ($CmdArgs) {
        Write-Host ("Running: {0} {1}" -f $Exe, ($CmdArgs -join " "))
    } else {
        Write-Host ("Running: {0}" -f $Exe)
    }
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir "..")).Path

$Ethernova = Resolve-FirstPath @(
    (Join-Path $RepoRoot "bin\\ethernova.exe"),
    (Join-Path $RepoRoot "ethernova.exe")
)
if (-not $Ethernova) { throw "ethernova.exe not found (expected bin\\ethernova.exe or root)." }

$GenesisUpgrade = Resolve-FirstPath @(
    (Join-Path $RepoRoot "genesis\\genesis-upgrade-70000.json"),
    (Join-Path $RepoRoot "genesis-upgrade-70000.json"),
    (Join-Path $RepoRoot "genesis\\genesis-upgrade-60000.json"),
    (Join-Path $RepoRoot "genesis-upgrade-60000.json")
)
if (-not $GenesisUpgrade) { throw "genesis-upgrade-70000.json or genesis-upgrade-60000.json not found." }

if (-not $DataDir) {
    $DataDir = Join-Path $RepoRoot "data-mainnet"
}

Write-Host ("Using upgrade genesis: {0}" -f $GenesisUpgrade)
Write-Host "Applying mainnet config upgrade..."
Write-Host "NOTE: Do NOT replace the genesis file in your datadir."
Write-Host "      This command updates the stored chain config in-place."

Write-Command -Exe $Ethernova -CmdArgs @("--datadir", $DataDir, "init", $GenesisUpgrade)
& $Ethernova --datadir $DataDir init $GenesisUpgrade
exit $LASTEXITCODE
