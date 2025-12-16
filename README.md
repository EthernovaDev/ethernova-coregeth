# Ethernova (CoreGeth fork)

[![Windows Build](https://github.com/EthernovaDev/ethernova-coregeth/actions/workflows/windows.yml/badge.svg)](https://github.com/EthernovaDev/ethernova-coregeth/actions/workflows/windows.yml)

**Ethernova is a Windows-focused fork of CoreGeth** built to run the Ethernova EVM network with **Ethash PoW**, Ethernova genesis files, and project-specific operational tooling (PowerShell scripts, launch docs, smoke tests).

> For node operators, pool operators (Miningcore), and devs who need an Ethernova-compatible execution client on Windows.

---

## What you get

- **Ethash PoW** execution client for Ethernova
- **Ethernova genesis** (mainnet + dev) and init tooling
- **Base fee vault redirection** (project feature; see docs)
- **Windows-first scripts** for build/run/verification/smoke tests
- **RPC smoke tests** for quick validation (chainId/genesis/getWork)

---

## Before you begin

### Requirements (Windows)
- Windows 10/11 x64
- Go 1.21 (per CI); install via `actions/setup-go` equivalent locally
- Build tools: MSYS2 mingw-w64 (mingw-w64-x86_64-gcc/make/pkgconf)
- Disk: dev is small; mainnet grows over time

> Toolchain specifics live in the PowerShell scripts and CI workflow; install MSYS2/mingw-w64 to match.

---

## Networks

| Network            | chainId | networkId | Consensus  | Genesis file            | Block 0 hash                                                |
|--------------------|--------:|----------:|------------|-------------------------|------------------------------------------------------------|
| Ethernova Mainnet  | 77777   | 77777     | Ethash PoW | `genesis-mainnet.json`  | `0xc67bd6160c1439360ab14abf7414e8f07186f3bed095121df3f3b66fdc6c2183` |
| Ethernova Dev      | 77778   | 77778     | Ethash PoW | `genesis-dev.json`      | (derive via verify script after init)                      |

---

## Quickstart (Windows)

### 1) Build
```powershell
powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1
```

### 2) Dev chain init + run (chainId 77778)
```powershell
powershell -ExecutionPolicy Bypass -File scripts/init-ethernova.ps1 -Mode dev
```

### 3) Mainnet init (with bootnodes)
```powershell
powershell -ExecutionPolicy Bypass -File scripts/init-ethernova.ps1 -Mode mainnet -Bootnodes "<enode://...>"
```

### 4) Verify mainnet fingerprint
```powershell
powershell -ExecutionPolicy Bypass -File scripts/verify-mainnet.ps1 -Endpoint http://127.0.0.1:8545
```

### 5) Smoke test fees (dev)
```powershell
powershell -ExecutionPolicy Bypass -File scripts/smoke-test-fees.ps1
```

---

## Run a local mainnet node (Miningcore-friendly)

RPC is bound to localhost; run Miningcore on the same host or via SSH tunnel.

### Start node (example)
```powershell
powershell -ExecutionPolicy Bypass -File scripts/run-mainnet-node.ps1 -Etherbase <POOL_ADDRESS> -Mine
```

### Test RPC
```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-rpc.ps1 -Endpoint http://127.0.0.1:8545
```
Expected:
- `eth_chainId` == 0x12fd1 (77777)
- Genesis/block0 matches fingerprint
- `eth_getWork` responds when mining/getWork is enabled

> Full pool-oriented walkthrough: see `docs/LAUNCH.md` (Miningcore quickstart).

---

## Default endpoints & ports
- HTTP RPC: `http://127.0.0.1:8545`
- WS RPC: `ws://127.0.0.1:8546` (or HttpPort+1)
- P2P: `30303`
- Data dirs: `data-mainnet\`, `data-dev\`
- Logs: `logs\` (see script outputs)

---

## Bootnodes
Mainnet bootnodes (enode): add stable entries in `networks/mainnet/bootnodes.txt` and `static-nodes.json`.
> Provide at least 2–5 stable bootnodes before launch.

---

## Documentation
- Launch & operations: `docs/LAUNCH.md`
- Dev workflow: `docs/DEV.md`
- Config reference: `docs/CONFIG.md`
- Key scripts: `scripts/run-mainnet-node.ps1`, `scripts/test-rpc.ps1`, `scripts/init-ethernova.ps1`, `scripts/verify-mainnet.ps1`, `scripts/smoke-test-fees.ps1`

---

## Troubleshooting
- RPC works but `eth_getWork` is null: ensure `-Mine`/getWork enabled; hit `http://127.0.0.1:8545`.
- Genesis mismatch: re-init with correct genesis; verify via `scripts/verify-mainnet.ps1`; avoid reusing wrong datadir.

---

## Contributing
PRs welcome for Ethernova-specific changes (scripts, docs, chain config, ops hardening). Keep upstream-friendly changes isolated.

---

## Upstream / Credits
Ethernova is a fork of CoreGeth, downstream of `ethereum/go-ethereum`.
- CoreGeth: https://github.com/etclabscore/core-geth
- go-ethereum: https://github.com/ethereum/go-ethereum

---

## Licensing
- Library code (outside `cmd/`): GNU LGPL-3.0-or-later
- Binaries under `cmd/`: GNU GPL-3.0-or-later

See `LICENSE`, `COPYING`, and `COPYING.LESSER`.
