Param(
    [string]$Endpoint = "http://127.0.0.1:8545"
)

$ErrorActionPreference = "Stop"

function Call-Rpc([string]$method, [object[]]$params=@()) {
    $payload = @{
        jsonrpc = "2.0"
        id      = 1
        method  = $method
        params  = $params
    } | ConvertTo-Json -Compress
    try {
        return Invoke-RestMethod -Method Post -Uri $Endpoint -Body $payload -ContentType "application/json"
    } catch {
        return $null
    }
}

function Print-Result($name, $ok, $extra="") {
    if ($ok) {
        Write-Host "OK   $name $extra"
    } else {
        Write-Host "FAIL $name $extra" -ForegroundColor Red
    }
}

$chain = Call-Rpc "eth_chainId"
$okChain = $chain -and $chain.result
Print-Result "eth_chainId" $okChain "($($chain.result))"

$block0 = Call-Rpc "eth_getBlockByNumber" @("0x0",$false)
$okBlock = $block0 -and $block0.result -and $block0.result.hash
Print-Result "eth_getBlockByNumber(0x0,false)" $okBlock "hash=$($block0.result.hash)"

$work = Call-Rpc "eth_getWork"
$okWork = $work -and $work.result -and $work.result.Count -ge 3
$workHint = if ($okWork) { "" } else { "(start mining or enable getWork)" }
Print-Result "eth_getWork" $okWork $workHint
