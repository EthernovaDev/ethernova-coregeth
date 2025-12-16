# verify-mainnet.ps1 (PowerShell 5.1 compatible)
[CmdletBinding()]
param(
  [string]$GenesisPath = "",
  [string]$Endpoint = "http://127.0.0.1:8545",
  [string]$ExpectedGenesisHash = "0xc67bd6160c1439360ab14abf7414e8f07186f3bed095121df3f3b66fdc6c2183"
)

$ErrorActionPreference = "Stop"

function Normalize-Hex([string]$v) {
  if (-not $v) { return "" }
  $s = $v.Trim().Trim('"').ToLower()
  if (-not $s.StartsWith("0x")) { $s = "0x$s" }
  return $s
}

function HexToBytes([string]$hex) {
  if (-not $hex) { return @() }
  $h = $hex.Trim()
  if ($h.StartsWith("0x")) { $h = $h.Substring(2) }
  if ($h.Length % 2 -ne 0) { $h = "0$h" }
  $bytes = New-Object byte[] ($h.Length / 2)
  for ($i = 0; $i -lt $bytes.Length; $i++) {
    $bytes[$i] = [Convert]::ToByte($h.Substring($i * 2, 2), 16)
  }
  return $bytes
}

function HexToUtf8([string]$hex) {
  try {
    $b = HexToBytes $hex
    if ($b.Length -eq 0) { return "" }
    $text = -join ($b | ForEach-Object { [char]$_ })
    return $text.Trim([char]0)
  } catch { return "" }
}

function Parse-BigInt([string]$s) {
  if (-not $s) { return $null }
  $t = $s.Trim()
  if ($t.StartsWith("0x")) {
    $be = HexToBytes $t            # big-endian bytes
    if ($be.Length -eq 0) { return $null }
    [Array]::Reverse($be)          # little-endian for BigInteger

    # Add 0x00 to force unsigned
    $le = New-Object byte[] ($be.Length + 1)
    [Array]::Copy($be, 0, $le, 0, $be.Length)
    $le[$be.Length] = 0

    return [System.Numerics.BigInteger]::new($le)
  }

  return [System.Numerics.BigInteger]::Parse($t, [System.Globalization.CultureInfo]::InvariantCulture)
}

# Simple JSON-RPC caller (HTTP/HTTPS)
function Call-Rpc([string]$method, [object[]]$params) {
  $payload = @{
    jsonrpc = "2.0"
    method  = $method
    params  = $params
    id      = 1
  } | ConvertTo-Json -Compress
  try {
    $resp = Invoke-RestMethod -Method Post -Uri $Endpoint -Body $payload -ContentType "application/json"
    if ($resp -and $resp.result) { return $resp.result }
    return $null
  } catch {
    return $null
  }
}

# Resolve repo root + defaults
$repoRoot = Split-Path -Parent $PSScriptRoot

if ([string]::IsNullOrWhiteSpace($GenesisPath)) {
  $GenesisPath = Join-Path $repoRoot "genesis-mainnet.json"
}

Write-Host "== Ethernova Mainnet Fingerprint Verify ==" -ForegroundColor Cyan
Write-Host "GenesisPath: $GenesisPath"
Write-Host "Endpoint:    $Endpoint"
Write-Host ""

if (-not (Test-Path $GenesisPath)) {
  throw "Genesis not found: $GenesisPath"
}

# Read expected from genesis JSON
$gen = Get-Content -Raw -Path $GenesisPath | ConvertFrom-Json
$cfg = $gen.config

$expected = [ordered]@{
  "ChainId"          = [string]$cfg.chainId
  "NetworkId"        = [string]$cfg.networkId
  "Consensus"        = "Ethash"
  "GenesisHash"      = (Normalize-Hex $ExpectedGenesisHash)
  "BaseFeeVault"     = (Normalize-Hex $cfg.baseFeeVault)
  "GasLimit"         = (Normalize-Hex $gen.gasLimit)
  "Difficulty"       = (Normalize-Hex $gen.difficulty)
  "BaseFeePerGas"    = (Normalize-Hex $gen.baseFeePerGas)
  "ExtraDataHex"     = (Normalize-Hex $gen.extraData)
  "ExtraDataText"    = (HexToUtf8 $gen.extraData)
}

# Read runtime from node (via JSON-RPC)
$runtime = [ordered]@{}
$missing = New-Object System.Collections.Generic.List[string]

$runtime["ChainId"]   = Call-Rpc "eth_chainId" @()
$runtime["NetworkId"] = Call-Rpc "net_version" @()
$block0              = Call-Rpc "eth_getBlockByNumber" @("0x0",$false)
if ($block0) {
  $runtime["GenesisHash"]   = $block0.hash
  $runtime["GasLimit"]      = $block0.gasLimit
  $runtime["Difficulty"]    = $block0.difficulty
  $runtime["BaseFeePerGas"] = $block0.baseFeePerGas
  $runtime["ExtraDataHex"]  = $block0.extraData
  $runtime["ExtraDataText"] = HexToUtf8 $runtime["ExtraDataHex"]
} else {
  $missing.Add("Block 0 via eth_getBlockByNumber") | Out-Null
}
# baseFeeVault not exposed on public RPC; mark as skipped
$runtime["BaseFeeVault"] = "SKIP (not available via public RPC)"

# Compare
$diffs = New-Object System.Collections.Generic.List[string]
$missing = New-Object System.Collections.Generic.List[string]

# Numeric comparisons (accept hex/dec)
function Compare-Num([string]$name, [string]$expHex, [string]$runVal) {
  $e = Parse-BigInt $expHex
  $r = Parse-BigInt $runVal
  if ($e -ne $null -and $r -ne $null -and $e -ne $r) {
    return "$name expected=$expHex runtime=$runVal"
  }
  return $null
}

# chainId/networkId
if ((Parse-BigInt $expected["ChainId"]) -ne (Parse-BigInt $runtime["ChainId"])) {
  $diffs.Add("ChainId expected=$($expected["ChainId"]) runtime=$($runtime["ChainId"])") | Out-Null
}
if ((Parse-BigInt $expected["NetworkId"]) -ne (Parse-BigInt $runtime["NetworkId"])) {
  $diffs.Add("NetworkId expected=$($expected["NetworkId"]) runtime=$($runtime["NetworkId"])") | Out-Null
}

# hash + strings
if ((Normalize-Hex $runtime["GenesisHash"]) -ne $expected["GenesisHash"]) {
  $diffs.Add("GenesisHash expected=$($expected["GenesisHash"]) runtime=$($runtime["GenesisHash"])") | Out-Null
}
if ((Normalize-Hex $runtime["ExtraDataHex"]) -ne $expected["ExtraDataHex"]) {
  $diffs.Add("ExtraData expected=$($expected["ExtraDataHex"]) runtime=$($runtime["ExtraDataHex"])") | Out-Null
}

# numeric fields from block0
$check = Compare-Num "GasLimit" $expected["GasLimit"] $runtime["GasLimit"]; if ($check) { $diffs.Add($check) | Out-Null }
$check = Compare-Num "Difficulty" $expected["Difficulty"] $runtime["Difficulty"]; if ($check) { $diffs.Add($check) | Out-Null }
$check = Compare-Num "BaseFeePerGas" $expected["BaseFeePerGas"] $runtime["BaseFeePerGas"]; if ($check) { $diffs.Add($check) | Out-Null }

# baseFeeVault (optional)
if (-not [string]::IsNullOrWhiteSpace($runtime["BaseFeeVault"])) {
  if ((Normalize-Hex $runtime["BaseFeeVault"]) -ne $expected["BaseFeeVault"]) {
    $diffs.Add("BaseFeeVault expected=$($expected["BaseFeeVault"]) runtime=$($runtime["BaseFeeVault"])") | Out-Null
  }
} else {
  $missing.Add("BaseFeeVault (admin.api may be disabled)") | Out-Null
}

Write-Host ""
Write-Host ("{0,-14} {1,-24} {2,-24} {3}" -f "Field","Expected","Runtime","Status")
Write-Host ("{0,-14} {1,-24} {2,-24} {3}" -f "-----","--------","-------","------")

function Print-Row([string]$name, [string]$expVal, [string]$runVal, [bool]$numeric=$false) {
  $status = "OK"
  if ([string]::IsNullOrWhiteSpace($runVal) -or $runVal -like "SKIP*") {
    $status = "SKIP"
    $missing.Add($name) | Out-Null
  } elseif ($numeric) {
    if ((Parse-BigInt $expVal) -ne (Parse-BigInt $runVal)) { $status = "MISMATCH" }
  } else {
    if ((Normalize-Hex $expVal) -ne (Normalize-Hex $runVal)) { $status = "MISMATCH" }
  }
  Write-Host ("{0,-14} {1,-24} {2,-24} {3}" -f $name, $expVal, $runVal, $status)
  if ($status -eq "MISMATCH") {
    $diffs.Add("$name expected=$expVal runtime=$runVal") | Out-Null
  }
}

Print-Row "ChainId"       $expected["ChainId"]       $runtime["ChainId"]       $true
Print-Row "NetworkId"     $expected["NetworkId"]     $runtime["NetworkId"]     $true
Print-Row "GenesisHash"   $expected["GenesisHash"]   $runtime["GenesisHash"]
Print-Row "GasLimit"      $expected["GasLimit"]      $runtime["GasLimit"]      $true
Print-Row "Difficulty"    $expected["Difficulty"]    $runtime["Difficulty"]    $true
Print-Row "BaseFeePerGas" $expected["BaseFeePerGas"] $runtime["BaseFeePerGas"] $true
Print-Row "ExtraDataHex"  $expected["ExtraDataHex"]  $runtime["ExtraDataHex"]
Print-Row "ExtraDataText" $expected["ExtraDataText"] $runtime["ExtraDataText"]
Print-Row "BaseFeeVault"  $expected["BaseFeeVault"]  $runtime["BaseFeeVault"]

Write-Host ""
if ($diffs.Count -eq 0) {
  Write-Host "OK: Mainnet fingerprint matches." -ForegroundColor Green
  if ($missing.Count -gt 0) {
    Write-Host "Note: Skipped fields:" -ForegroundColor Yellow
    $missing | Sort-Object -Unique | ForEach-Object { " - $_" } | Write-Host
  }
  exit 0
} else {
  Write-Host "MISMATCH: Differences found:" -ForegroundColor Red
  $diffs | ForEach-Object { " - $_" } | Write-Host
  if ($missing.Count -gt 0) {
    Write-Host "Skipped (not compared):" -ForegroundColor Yellow
    $missing | Sort-Object -Unique | ForEach-Object { " - $_" } | Write-Host
  }
  exit 2
}
