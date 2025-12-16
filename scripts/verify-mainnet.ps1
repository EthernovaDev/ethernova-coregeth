# verify-mainnet.ps1 (PowerShell 5.1 compatible)
[CmdletBinding()]
param(
  [string]$GenesisPath = "",
  [string]$Endpoint = "\\.\\pipe\\ethernova-mainnet.ipc",
  [string]$Binary = "",
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
    return ([Text.Encoding]::UTF8.GetString($b)).Trim([char]0)
  } catch { return "" }
}

function Parse-BigInt([string]$s) {
  if (-not $s) { return $null }
  $t = $s.Trim()
  if ($t.StartsWith("0x")) {
    $be = HexToBytes $t            # big-endian bytes
    [Array]::Reverse($be)          # little-endian for BigInteger

    # Add 0x00 to force unsigned
    $le = New-Object byte[] ($be.Length + 1)
    [Array]::Copy($be, 0, $le, 0, $be.Length)
    $le[$be.Length] = 0

    return [System.Numerics.BigInteger]::new($le)
  }

  return [System.Numerics.BigInteger]::Parse($t, [System.Globalization.CultureInfo]::InvariantCulture)
}

function Attach-Exec([string]$expr) {
  $out = & $Binary attach --exec $expr $Endpoint 2>$null | Out-String
  $lines = $out -split "`r?`n" | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" -and $_ -notmatch "^(WARN|INFO)\b" }
  if ($lines.Count -eq 0) { return "" }
  return $lines[$lines.Count - 1].Trim()
}

# Resolve repo root + defaults
$repoRoot = Split-Path -Parent $PSScriptRoot

if ([string]::IsNullOrWhiteSpace($GenesisPath)) {
  $GenesisPath = Join-Path $repoRoot "genesis-mainnet.json"
}

if ([string]::IsNullOrWhiteSpace($Binary)) {
  $candidate = Join-Path $repoRoot "bin\ethernova.exe"
  if (Test-Path $candidate) { $Binary = $candidate } else { $Binary = "ethernova.exe" }
}

Write-Host "== Ethernova Mainnet Fingerprint Verify ==" -ForegroundColor Cyan
Write-Host "GenesisPath: $GenesisPath"
Write-Host "Binary:      $Binary"
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

# Read runtime from node (via attach)
$runtime = [ordered]@{}
try {
  $runtime["ChainId"]       = Attach-Exec "eth.chainId"
  $runtime["NetworkId"]     = Attach-Exec "net.version"
  $runtime["GenesisHash"]   = Attach-Exec "eth.getBlock(0).hash"
  $runtime["GasLimit"]      = Attach-Exec "eth.getBlock(0).gasLimit"
  $runtime["Difficulty"]    = Attach-Exec "eth.getBlock(0).difficulty"
  $runtime["BaseFeePerGas"] = Attach-Exec "eth.getBlock(0).baseFeePerGas"
  $runtime["ExtraDataHex"]  = Attach-Exec "eth.getBlock(0).extraData"
  $runtime["ExtraDataText"] = HexToUtf8 $runtime["ExtraDataHex"]

  # optional (may be undefined if admin.nodeInfo doesn't expose custom field)
  $runtime["BaseFeeVault"]  = Attach-Exec "admin.nodeInfo.protocols.eth.config.baseFeeVault"
} catch {
  Write-Host "WARN: Could not query node via attach. Is ethernova running and is Endpoint correct?" -ForegroundColor Yellow
  Write-Host $_.Exception.Message -ForegroundColor Yellow
}

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
Write-Host "Expected (from genesis-mainnet.json):" -ForegroundColor Cyan
$expected.GetEnumerator() | ForEach-Object { "{0,-14} {1}" -f $_.Key, $_.Value } | Write-Host

Write-Host ""
Write-Host "Runtime (from node):" -ForegroundColor Cyan
$runtime.GetEnumerator() | ForEach-Object { "{0,-14} {1}" -f $_.Key, $_.Value } | Write-Host

Write-Host ""
if ($diffs.Count -eq 0) {
  Write-Host "OK: Mainnet fingerprint matches." -ForegroundColor Green
  if ($missing.Count -gt 0) {
    Write-Host "Note: Skipped optional fields:" -ForegroundColor Yellow
    $missing | ForEach-Object { " - $_" } | Write-Host
  }
  exit 0
} else {
  Write-Host "MISMATCH: Differences found:" -ForegroundColor Red
  $diffs | ForEach-Object { " - $_" } | Write-Host
  if ($missing.Count -gt 0) {
    Write-Host "Missing (not compared):" -ForegroundColor Yellow
    $missing | ForEach-Object { " - $_" } | Write-Host
  }
  exit 2
}
