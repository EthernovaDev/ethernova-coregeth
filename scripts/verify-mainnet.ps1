$ErrorActionPreference = "Stop"

Param(
    [string]$GenesisPath = "",
    [string]$Binary = "",
    [string]$Rpc = "http://127.0.0.1:8545",
    [switch]$SkipRpc
)

$RepoRoot = Split-Path $PSScriptRoot -Parent
if (-not $GenesisPath) { $GenesisPath = Join-Path $RepoRoot "genesis-mainnet.json" }
if (-not $Binary) { $Binary = Join-Path $RepoRoot "bin\ethernova.exe" }

if (-not (Test-Path $GenesisPath)) { throw "Genesis not found at $GenesisPath" }

function Decode-HexAscii([string]$hex) {
    if ($hex.StartsWith("0x")) { $hex = $hex.Substring(2) }
    if ($hex.Length -eq 0) { return "" }
    $bytes = New-Object byte[] ($hex.Length / 2)
    for ($i = 0; $i -lt $bytes.Length; $i++) {
        $bytes[$i] = [Convert]::ToByte($hex.Substring($i*2,2),16)
    }
    return [System.Text.Encoding]::ASCII.GetString($bytes)
}

$gen = Get-Content -Raw -Path $GenesisPath | ConvertFrom-Json
$expectedHash = "0xc67bd6160c1439360ab14abf7414e8f07186f3bed095121df3f3b66fdc6c2183"
$expected = [ordered]@{
    chainId       = "$($gen.config.chainId)"
    networkId     = "$($gen.config.networkId)"
    baseFeeVault  = "$($gen.config.baseFeeVault)"
    gasLimit      = "$($gen.gasLimit)"
    difficulty    = "$($gen.difficulty)"
    baseFeePerGas = "$($gen.baseFeePerGas)"
    extraData     = (Decode-HexAscii $gen.extraData)
    genesisHash   = $expectedHash
}

Write-Host "Expected fingerprint (from genesis):"
$expected.GetEnumerator() | ForEach-Object { Write-Host ("  {0}: {1}" -f $_.Key, $_.Value) }

if ($SkipRpc) { return }
if (-not (Test-Path $Binary)) {
    Write-Warning "Binary not found at $Binary, skipping runtime verification."
    return
}

Write-Host "Checking runtime via $Rpc ..."
$expr = 'JSON.stringify({hash:eth.getBlock(0).hash,chainId:eth.chainId(),config:(admin.nodeInfo.protocols.eth.config||{})})'
$out = & $Binary attach --exec $expr $Rpc 2>&1
$jsonLine = ($out -split "`r?`n") | Where-Object { $_ -match '\{.*\}' } | Select-Object -Last 1
if (-not $jsonLine) {
    Write-Host "---- RAW OUTPUT ----"
    $out | ForEach-Object { Write-Host $_ }
    throw "Could not parse runtime JSON."
}

$jsonClean = $jsonLine
if ($jsonClean.StartsWith('"') -and $jsonClean.EndsWith('"')) { $jsonClean = $jsonClean.Trim('"') }
$jsonClean = $jsonClean -replace '\\\"','"'
$runtime = $jsonClean | ConvertFrom-Json

$mismatch = @()
if (("$($runtime.chainId)") -ne $expected.chainId) { $mismatch += "chainId runtime=$($runtime.chainId) expected=$($expected.chainId)" }
if ($runtime.hash -and ($runtime.hash.ToLower() -ne $expected.genesisHash.ToLower())) { $mismatch += "genesis hash runtime=$($runtime.hash) expected=$($expected.genesisHash)" }
if ($runtime.config -and $runtime.config.baseFeeVault) {
    if (($runtime.config.baseFeeVault.ToLower()) -ne ($expected.baseFeeVault.ToLower())) { $mismatch += "baseFeeVault runtime=$($runtime.config.baseFeeVault) expected=$($expected.baseFeeVault)" }
}

if ($mismatch.Count -eq 0) {
    Write-Host "OK: runtime matches genesis/mainnet fingerprint."
} else {
    Write-Warning "MISMATCH:"
    $mismatch | ForEach-Object { Write-Host " - $_" }
}
