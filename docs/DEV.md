# Ethernova Dev Workflow (Windows)

## Build
```
powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1
```
Output: `bin/ethernova.exe`

## Dev genesis and fast mining (chainId 77778)
Use `genesis-dev.json` (difficulty=0x1, forks at block 0, block reward halving baked in, permissive txpool).

## Init + start (dev mode)
```
powershell -ExecutionPolicy Bypass -File scripts/init-ethernova.ps1 -Mode dev
```
- Datadir: `data-dev\`
- Logs: `logs\node.log` and `logs\node.err`
- RPC: `http://127.0.0.1:8545`, `ws://127.0.0.1:8546`
- APIs: `eth,net,web3,personal,miner,txpool,admin,debug`
- txpool/miner gasprice set to 0 for easy inclusion.

## Smoke test (baseFeeVault)
```
powershell -ExecutionPolicy Bypass -File scripts/smoke-test-fees.ps1 -Rpc http://127.0.0.1:8545 -Pass "nova-smoke"
```
Pass criteria: type-2 tx mined, gasUsed>0, vault delta == baseFeePerGas * gasUsed.
> Tip: If you have PowerShell 7 installed, `pwsh` works too; examples above assume Windows PowerShell with `powershell -ExecutionPolicy Bypass -File ...`.
## Useful attach commands
```
bin\ethernova.exe attach --exec "eth.blockNumber" http://127.0.0.1:8545
bin\ethernova.exe attach --exec "txpool.status"   http://127.0.0.1:8545
bin\ethernova.exe attach --exec "admin.nodeInfo.protocols.eth.config" http://127.0.0.1:8545
```

## Keystore safety
- Scripts back up `data-*/keystore` to `backups/` before wiping.
- Never delete keystore without a backup. Keep passwords secure.

## Test suite
```
set PATH=C:\msys64\mingw64\bin;%PATH%
set CC=C:\msys64\mingw64\bin\gcc.exe
set CGO_ENABLED=1
go test (go list ./... excluding cmd/geth, consensus/ethash, tests)
```
Focus tests: `core/basefee_vault_test.go`, `params/types/ctypes/ethash_reward_test.go`.

## CI policy
- CI runs the fast subset above (excludes `cmd/geth`, `consensus/ethash`, `tests`).
- To run the full integration suite locally: `go test ./...` (expect longer time and ensure no port conflicts with authrpc/HTTP).
