param(
    [string]$DataDir = "",
    [int]$HttpPort = 8545,
    [int]$WsPort = 8546,
    [switch]$Mine
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

if (-not $DataDir) {
    $DataDir = Join-Path $RepoRoot "data-mainnet"
}

if (-not (Test-Path $DataDir)) {
    New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
}

$Args = @(
    "--datadir", $DataDir,
    "--networkid", "77777",
    "--http", "--http.addr", "127.0.0.1", "--http.port", "$HttpPort",
    "--http.api", "eth,net,web3,debug",
    "--ws", "--ws.addr", "127.0.0.1", "--ws.port", "$WsPort",
    "--ws.api", "eth,net,web3,debug"
)

if ($Mine) {
    $Args += @("--mine")
}

Write-Command -Exe $Ethernova -CmdArgs $Args
& $Ethernova @Args
