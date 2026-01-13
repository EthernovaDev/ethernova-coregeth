param()

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path $PSScriptRoot -Parent

function Resolve-Go {
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if ($goCmd) {
        return $goCmd.Path
    }
    $localGo = Join-Path $RepoRoot ".tools\go\bin\go.exe"
    if (Test-Path $localGo) {
        $env:GOROOT = Join-Path $RepoRoot ".tools\go"
        $env:PATH = "$env:GOROOT\bin;$env:PATH"
        return $localGo
    }
    return $null
}

$go = Resolve-Go
if (-not $go) {
    Write-Host "FAIL: go not found. Install Go 1.21+ or place it at .tools\go."
    exit 1
}

Write-Host "== ChainId gate verification (post-fork) =="
$output = & $go test ./core/types -run TestVerifyChainIDGatePostFork -v 2>&1
$output | ForEach-Object { Write-Host $_ }

$rejectLine = $output | Select-String -SimpleMatch "VERIFY_CHAINID_GATE: block=138396 old_chainId=77777"
$acceptLine = $output | Select-String -SimpleMatch "VERIFY_CHAINID_GATE: block=138396 new_chainId=121525"
$rejectOk = $rejectLine -and ($rejectLine -match "invalid chain id for signer")
$acceptOk = $acceptLine -and ($acceptLine -match "accepted_from=")

if ($rejectOk -and $acceptOk) {
    Write-Host "PASS: chainId gate at fork block"
    exit 0
}

Write-Host "FAIL: chainId gate at fork block"
exit 1
