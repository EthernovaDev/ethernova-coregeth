$ErrorActionPreference = "Stop"

Param(
    [string]$Datadir = ".\data-mainnet",
    [int]$Port = 30303,
    [int]$HttpPort = 8545,
    [string]$Etherbase = "<POOL_ADDRESS_PLACEHOLDER>",
    [switch]$Mine
)

$RepoRoot = Split-Path $PSScriptRoot -Parent
Set-Location $RepoRoot

$Binary = Join-Path $RepoRoot "bin\ethernova.exe"
if (-not (Test-Path $Binary)) { throw "Binary not found at $Binary. Build first with scripts\build-windows.ps1." }

$LogsDir = Join-Path $RepoRoot "logs"
if (-not (Test-Path $LogsDir)) { New-Item -ItemType Directory -Force -Path $LogsDir | Out-Null }
$NodeLog = Join-Path $LogsDir "mainnet-node.log"
$NodeErr = Join-Path $LogsDir "mainnet-node.err.log"

if (-not (Test-Path $Datadir)) { New-Item -ItemType Directory -Force -Path $Datadir | Out-Null }

$args = @(
    "--networkid", "77777",
    "--datadir", $Datadir,
    "--port", "$Port",
    "--http", "--http.addr", "127.0.0.1", "--http.port", "$HttpPort",
    "--http.api", "eth,net,web3,admin",
    "--ws", "--ws.addr", "127.0.0.1", "--ws.port", ($HttpPort + 1), "--ws.api", "eth,net,web3,admin",
    "--authrpc.addr", "127.0.0.1", "--authrpc.port", "8551",
    "--ipcpath", "\\.\pipe\ethernova-mainnet.ipc",
    "--verbosity", "4"
)

if ($Mine.IsPresent) {
    $args += @("--mine", "--miner.etherbase", $Etherbase)
    # miner.threads 0 => auto-detect if supported
    $args += @("--miner.threads", "0")
}

Write-Host "Starting mainnet node (RPC on http://127.0.0.1:$HttpPort) ..."
$startInfo = @{
    FilePath               = $Binary
    ArgumentList           = $args
    RedirectStandardOutput = $NodeLog
    RedirectStandardError  = $NodeErr
    NoNewWindow            = $true
}

$proc = Start-Process @startInfo -PassThru
Write-Host "PID: $($proc.Id)"
Write-Host "Logs: $NodeLog / $NodeErr"
Write-Host "Note: RPC is bound to localhost; keep Miningcore on the same host or tunnel securely."
