[CmdletBinding()]
param(
  [string]$Endpoint = "http://127.0.0.1:8545"
)

$ErrorActionPreference = "Stop"

function Call-Rpc([string]$method, [object[]]$params) {
  $payload = @{
    jsonrpc = "2.0"
    method  = $method
    params  = $params
    id      = 1
  } | ConvertTo-Json -Compress -Depth 8

  $resp = Invoke-RestMethod -Method Post -Uri $Endpoint -Body $payload -ContentType "application/json"
  if ($resp -and $resp.error) {
    throw ("RPC error: {0}" -f ($resp.error | ConvertTo-Json -Compress))
  }
  return $resp.result
}

Write-Host "== Ethernova Fork Verify ==" -ForegroundColor Cyan
Write-Host ("endpoint={0}" -f $Endpoint)

$clientVersion = Call-Rpc "web3_clientVersion" @()
Write-Host ("clientVersion={0}" -f $clientVersion)
if (-not $clientVersion -or $clientVersion -notmatch "Ethernova/v1\.2\.8") {
  throw ("unexpected clientVersion: {0}" -f $clientVersion)
}

$blockNumber = Call-Rpc "eth_blockNumber" @()
Write-Host ("blockNumber={0}" -f $blockNumber)
if (-not $blockNumber) {
  throw "unexpected empty blockNumber"
}

$addr = "0x0000000000000000000000000000000000000001"
$code = "0x600160021b00" # PUSH1 1, PUSH1 2, SHL, STOP
$call = @{
  from = "0x0000000000000000000000000000000000000000"
  to   = $addr
  gas  = "0x2dc6c0"
  data = "0x"
}
$stateOverrides = @{
  $addr = @{ code = $code }
}

function Trace-SHL([string]$label, [string]$blockHex, [bool]$expectInvalid) {
  $config = @{
    tracer         = "callTracer"
    stateOverrides = $stateOverrides
    blockOverrides = @{ number = $blockHex }
  }
  $result = Call-Rpc "debug_traceCall" @($call, "latest", $config)
  if ($null -eq $result) {
    throw ("{0}: debug_traceCall returned null" -f $label)
  }
  $err = $null
  if ($result.PSObject.Properties.Name -contains "error") {
    $err = $result.error
  }
  if ($expectInvalid) {
    if (-not $err -or $err -notmatch "invalid opcode") {
      throw ("{0}: expected invalid opcode, got {1}" -f $label, ($result | ConvertTo-Json -Compress -Depth 8))
    }
  } else {
    if ($err) {
      throw ("{0}: expected success, got error {1}" -f $label, $err)
    }
  }
  Write-Host ("{0}: OK" -f $label)
}

Trace-SHL "pre-fork (block 104999)" "0x19a27" $true
Trace-SHL "post-fork (block 105000)" "0x19a28" $false

Write-Host "OK: fork verification passed." -ForegroundColor Green
