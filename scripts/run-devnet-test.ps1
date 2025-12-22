$ErrorActionPreference = "Stop"

param(
    [string]$DataDir = "",
    [switch]$KeepRunning
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

$Evmcheck = Resolve-FirstPath @(
    (Join-Path $RepoRoot "bin\\evmcheck.exe"),
    (Join-Path $RepoRoot "evmcheck.exe")
)
if (-not $Evmcheck) { throw "evmcheck.exe not found (expected bin\\evmcheck.exe or root)." }

$GenesisPath = Resolve-FirstPath @(
    (Join-Path $RepoRoot "genesis\\genesis-devnet-fork20.json"),
    (Join-Path $RepoRoot "genesis-devnet-fork20.json")
)
if (-not $GenesisPath) { throw "genesis-devnet-fork20.json not found." }

$KeyPath = Resolve-FirstPath @(
    (Join-Path $RepoRoot "genesis\\devnet-testkey.txt"),
    (Join-Path $RepoRoot "devnet-testkey.txt")
)
if (-not $KeyPath) { throw "devnet-testkey.txt not found." }

$KeyData = @{}
Get-Content $KeyPath | ForEach-Object {
    if ($_ -match '^\s*([A-Z_]+)\s*=\s*(.+)\s*$') {
        $KeyData[$matches[1]] = $matches[2].Trim()
    }
}

$PrivKey = $KeyData["PRIVATE_KEY"]
$DevAddr = $KeyData["ADDRESS"]
$ChainID = [uint64]$KeyData["CHAINID"]

if (-not $PrivKey -or -not $DevAddr -or -not $ChainID) {
    throw "devnet-testkey.txt missing PRIVATE_KEY, ADDRESS, or CHAINID."
}

if (-not $DataDir) {
    $DataDir = Join-Path $RepoRoot "data-devnet"
}

$RpcUrl = "http://127.0.0.1:8545"
$ForkBlock = 20

if (-not (Test-Path $DataDir)) {
    New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
}

$ChainData = Join-Path $DataDir "geth"
if (-not (Test-Path $ChainData)) {
    Write-Host "Initializing devnet datadir..."
    & $Ethernova --datadir $DataDir init $GenesisPath | Out-Null
}

$LogPath = Join-Path $DataDir "devnet.log"
$Args = @(
    "--datadir", $DataDir,
    "--http", "--http.addr", "127.0.0.1", "--http.port", "8545",
    "--http.api", "eth,net,web3,debug",
    "--ws", "--ws.addr", "127.0.0.1", "--ws.port", "8546",
    "--ws.api", "eth,net,web3,debug",
    "--nodiscover", "--maxpeers", "0",
    "--networkid", "$ChainID",
    "--mine", "--miner.etherbase", $DevAddr
)

Write-Host "Starting devnet node..."
$proc = Start-Process -FilePath $Ethernova -ArgumentList $Args -PassThru -RedirectStandardOutput $LogPath -RedirectStandardError $LogPath

function Get-BlockNumber {
    param([string]$Url)
    $body = '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}'
    try {
        $resp = Invoke-RestMethod -Method Post -Uri $Url -ContentType "application/json" -Body $body -TimeoutSec 5
        if ($resp.result) {
            return [Convert]::ToInt64($resp.result.Replace("0x", ""), 16)
        }
    } catch {
        return $null
    }
    return $null
}

$deadline = (Get-Date).AddMinutes(3)
try {
    Write-Host "Waiting for RPC..."
    do {
        $bn = Get-BlockNumber -Url $RpcUrl
        if ($bn -ne $null) { break }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)

    if ($bn -eq $null) { throw "RPC did not become ready in time. Check $LogPath." }

    Write-Host "Mining until block >= $ForkBlock..."
    do {
        Start-Sleep -Seconds 1
        $bn = Get-BlockNumber -Url $RpcUrl
    } while ($bn -lt $ForkBlock)

    Write-Host "Running evmcheck..."
    & $Evmcheck --rpc $RpcUrl --pk $PrivKey --chainid $ChainID --forkblock $ForkBlock
    $exitCode = $LASTEXITCODE
} finally {
    if (-not $KeepRunning) {
        Write-Host "Stopping devnet node..."
        try { Stop-Process -Id $proc.Id -Force } catch {}
    } else {
        Write-Host "Devnet left running. Logs: $LogPath"
    }
}

exit $exitCode
