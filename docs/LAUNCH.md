# Ethernova Launch Guide (Mainnet)

Windows-only instructions to stand up and operate Ethernova mainnet nodes safely.

## Modes and chain IDs
- Mainnet: chainId/networkId **77777** (`genesis-mainnet.json`)
- Dev/Testnet: chainId/networkId **77778** (`genesis-dev.json`)
- Scripts validate the chainId to avoid accidental cross-use.

## Build the binary
```
powershell -ExecutionPolicy Bypass -File scripts/build-windows.ps1
```
Outputs `bin\ethernova.exe`. (If you use PowerShell 7, `pwsh` also works.)

## Launch checklist (mainnet)
1) Pick genesis: `genesis-mainnet.json` (extraData "NOVA MAINNET").
2) Init + start:  
   `powershell -ExecutionPolicy Bypass -File scripts/init-ethernova.ps1 -Mode mainnet -Bootnodes "<enode://...>"`
3) Confirm fingerprint (hash + config):  
   `powershell -ExecutionPolicy Bypass -File scripts/verify-mainnet.ps1`
4) Share enode: `bin\ethernova.exe attach --exec "admin.nodeInfo.enode" http://127.0.0.1:8545`
5) Add bootnodes/static peers: edit `networks/mainnet/bootnodes.txt` and `networks/mainnet/static-nodes.json` then restart.
6) Optional: run a second node locally for peering sanity (see below).

## Mainnet Genesis Fingerprint
| Field              | Value                                                               |
|--------------------|---------------------------------------------------------------------|
| ChainId/NetworkId  | 77777                                                               |
| Consensus          | Ethash                                                              |
| Genesis Block Hash | 0xc67bd6160c1439360ab14abf7414e8f07186f3bed095121df3f3b66fdc6c2183  |
| BaseFeeVault       | 0x3a38560b66205bb6a31decbcb245450b2f15d4fd                          |
| GasLimit           | 0x1c9c380                                                           |
| Difficulty         | 0x400000                                                            |
| BaseFeePerGas      | 0x3b9aca00 (1 gwei)                                                 |
| extraData          | "NOVA MAINNET"                                                      |

## Bootnodes / static peers
- `networks/mainnet/bootnodes.txt`: enode URLs, one per line (placeholder until you replace them).
- `networks/mainnet/static-nodes.json`: JSON array of enode URLs copied into `data-mainnet/geth/static-nodes.json` by the init script.
- Bootnode helper: `powershell -ExecutionPolicy Bypass -File scripts/run-bootnode.ps1` (prints enode, HTTP admin on 8550 by default).
- For local peering, replace the IP in the enode with `127.0.0.1` to avoid NAT detection getting in the way.

### How to replace placeholders with real enodes
1) Generate a nodekey  
   - Windows: `powershell -ExecutionPolicy Bypass -Command "[IO.File]::WriteAllBytes('boot.key',(New-Guid).ToByteArray())"`  
   - Linux: `openssl rand -hex 32 > boot.key`
2) Start a bootnode on the VPS  
   - Windows (PowerShell): `powershell -ExecutionPolicy Bypass -File scripts/run-bootnode.ps1 -NodeKeyPath .\boot.key -Port 30303 -HttpPort 8550 -HttpAddr 0.0.0.0`  
   - Linux: `./ethernova --nodiscover --nodekey boot.key --port 30303 --http --http.addr 0.0.0.0 --http.port 8550 --http.api net,admin --verbosity 3 --ipcdisable`
3) Read the enode from the VPS: `ethernova attach --exec "admin.nodeInfo.enode" http://<bootnode-ip>:8550`
4) Paste the enode into `networks/mainnet/bootnodes.txt` and `networks/mainnet/static-nodes.json` (JSON array) and redistribute both files.
5) Verify peering from a fresh node:  
   `powershell -ExecutionPolicy Bypass -File scripts/check-peering.ps1 -RpcA http://127.0.0.1:8545`  
   Expect `net.peerCount > 0`.

Minimum checklist before publishing bootnodes:
- Ports 30303 TCP/UDP open.
- External IP / NAT is correct (`--nat extip:<ip>` if needed).
- VPS clock synced (NTP).

## Running a second node (local peering)
```
powershell -ExecutionPolicy Bypass -File scripts/run-second-node.ps1 -Mode mainnet -Bootnodes "<enode://of-first-node>" -Port 30304 -HttpPort 8547 -WsPort 8548
powershell -ExecutionPolicy Bypass -File scripts/check-peering.ps1 -RpcA http://127.0.0.1:8545 -RpcB http://127.0.0.1:8547
```
Expected: both nodes report `net.peerCount > 0`.

## Miningcore quickstart (solo mining / pool daemon)
- Keep RPC on localhost; run Miningcore on the same host or via SSH tunnel (do NOT expose 0.0.0.0).
- Start the RPC daemon (no datadir wipe):  
  `powershell -ExecutionPolicy Bypass -File scripts\run-mainnet-node.ps1 -Etherbase <POOL_ADDRESS> -Mine`
- Verify RPC responses for chainId/block0/getWork:  
  `powershell -ExecutionPolicy Bypass -File scripts\test-rpc.ps1 -Endpoint http://127.0.0.1:8545`
- Point Miningcore to `http://127.0.0.1:8545` with Ethash getWork; keep auth/whitelist per Miningcore docs.

## One-click mainnet node (portable)
- Download the Windows release ZIP, extract anywhere.
- Double-click `EthernovaNode.exe`:
  - Uses local folder: `.\data-mainnet` for state, `.\logs\mainnet-node.log` for logs.
  - Requires `ethernova.exe` and `genesis-mainnet.json` in the same folder.
  - RPC binds to localhost (`http://127.0.0.1:8545`, WS `8546`). If 8545/8546 are busy, it falls back to 8547/8548.
  - No datadir wipe; if not initialized, it runs genesis init once and logs to `logs\init.log` / `logs\init.err.log`.
- Stop by pressing Enter in the launcher console.
- For mining, prefer `scripts\run-mainnet-node.ps1 -Mine` (launcher defaults to non-mining for safety).

## Miningcore Pool Node (Windows Portable)
1) Download the Windows release ZIP and extract.
2) Run `Start-PoolNode.cmd 0xPOOLADDRESS` (or double-click and enter the etherbase when prompted).
   - Keeps datadir in `.\data-mainnet`, logs in `.\logs\pool-node.log` / `pool-node.err.log`.
   - RPC: `http://127.0.0.1:8545` (fallback 8547 if busy), WS: `127.0.0.1:8546` (fallback 8548).
   - Initializes with `genesis-mainnet.json` if needed; will not wipe data.
3) Point Miningcore daemon URL to `http://127.0.0.1:8545` (Ethash getWork). For remote Miningcore, use SSH tunnel—do not expose RPC publicly.
4) Verify node before attaching the pool:
   - `powershell -ExecutionPolicy Bypass -File scripts\test-rpc.ps1 -Endpoint http://127.0.0.1:8545`
   - `powershell -ExecutionPolicy Bypass -File scripts\verify-mainnet.ps1 -Endpoint http://127.0.0.1:8545`
   - Expected: chainId 0x12fd1, genesis hash `0xc67bd6...6c2183`, `eth_getWork` returns values when mining is on.

## Security defaults (mainnet)
- RPC binds to `127.0.0.1` only.
- HTTP APIs: `eth,net,web3` only (no `personal`, `admin`, `debug`, `txpool`).
- No `--allow-insecure-unlock`.
- Keep keystore backed up; scripts create `backups/` before wiping datadir.

## Economics
- Consensus: Ethash PoW.
- Base fee: redirected to treasury vault `0x3a38560b66205bb6a31decbcb245450b2f15d4fd`; tips stay with miner.
- Block reward: starts at 10 NOVA, halves every ~2,102,400 blocks (~1 year @15s) with a 1 NOVA floor. Schedule baked into genesis.

## Dev/Testnet mode (chainId 77778)
```
powershell -ExecutionPolicy Bypass -File scripts/init-ethernova.ps1 -Mode dev
powershell -ExecutionPolicy Bypass -File scripts/smoke-test-fees.ps1
```
- Datadir: `data-dev\`, easy mining (difficulty 0x1), txpool/miner gasprice 0.
- Use for local testing only; do not point wallets to mainnet RPC when using dev genesis.

## Explorer and wallets
- MetaMask (mainnet): RPC `http://127.0.0.1:8545`, Chain ID `77777`, Symbol `NOVA`, Explorer URL (set to your deployment).
- Suggested explorer: Blockscout. Provide env pointing to your RPC, chainId 77777, and publish the explorer URL alongside bootnodes.

## Release packaging
```
powershell -ExecutionPolicy Bypass -File scripts/package-release.ps1
```
Produces `dist/ethernova-<version>-windows.zip` with binaries, genesis files, scripts, and checksums.
