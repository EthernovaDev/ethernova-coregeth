# Ethernova Launch Guide (Mainnet)

Windows-only instructions to stand up and operate Ethernova mainnet nodes safely.

## Modes and chain IDs
- Mainnet: chainId/networkId **77777** (genesis-mainnet.json)
- Dev/Testnet: chainId/networkId **77778** (genesis-dev.json)
- Do not mix modes: scripts validate the chainId to avoid accidental mainnet use in dev and vice versa.

## Build the binary
```
pwsh -File scripts/build-windows.ps1
```
Outputs `bin\ethernova.exe`.

## Launch checklist (mainnet)
1) Pick genesis: `genesis-mainnet.json` (extraData "NOVA MAINNET").
2) Init + start:  
   `pwsh -File scripts/init-ethernova.ps1 -Mode mainnet -Bootnodes "<enode://...>"`
3) Confirm fingerprint (hash + config):  
   `pwsh -File scripts/print-genesis-fingerprint.ps1`
4) Share enode: `bin\ethernova.exe attach --exec "admin.nodeInfo.enode" http://127.0.0.1:8545`
5) Add bootnodes/static peers: edit `networks/mainnet/bootnodes.txt` and `networks/mainnet/static-nodes.json` then restart.
6) Optional: run a second node locally for peering sanity (see below).

## Genesis fingerprint (mainnet)
- ChainId/NetworkId: 77777
- BaseFeeVault: 0x3a38560b66205bb6a31decbcb245450b2f15d4fd
- GasLimit: 0x1c9c380
- Difficulty: 0x400000
- BaseFeePerGas: 0x3b9aca00 (1 gwei)
- Block 0 hash: 0xc67bd6160c1439360ab14abf7414e8f07186f3bed095121df3f3b66fdc6c2183 (from `scripts/print-genesis-fingerprint.ps1 -Endpoint \\.\pipe\ethernova-mainnet.ipc`)

## Bootnodes / static peers
- `networks/mainnet/bootnodes.txt`: enode URLs, one per line.
- `networks/mainnet/static-nodes.json`: JSON array of enode URLs copied into `data-mainnet/geth/static-nodes.json` by the init script.
- Bootnode helper: `pwsh -File scripts/run-bootnode.ps1` (prints enode, HTTP admin on 8550 by default).
- For local peering, replace the IP in the enode with `127.0.0.1` to avoid NAT detection getting in the way.
- Bootnodes oficiales (rellena con los reales antes del lanzamiento):
  | Nombre | Enode | Ubicación |
  |--------|-------|-----------|
  | bootnode-1 | enode://<pubkey>@<ip>:30303 | VPS recomendada (bahía segura) |
  | bootnode-2 | enode://<pubkey>@<ip>:30303 | VPS recomendada |
  - Procedimiento:
    1) En el VPS bootnode: `pwsh -File scripts/run-bootnode.ps1` (o equivalente Linux con flags: `geth --nodiscover --port 30303 --http --http.addr 0.0.0.0 --http.port 8550 --http.api net,admin --verbosity 3 --ipcdisable`).
    2) Obtener enode: `geth attach --exec "admin.nodeInfo.enode" http://<bootnode-ip>:8550`.
    3) Reemplazar enodes en `networks/mainnet/bootnodes.txt` y `networks/mainnet/static-nodes.json` y redistribuir.
    4) Verificar desde nodo fresh: `admin.addPeer("<enode>")` y comprobar `net.peerCount > 0`.

## Running a second node (local peering)
```
pwsh -File scripts/run-second-node.ps1 -Mode mainnet -Bootnodes "<enode://of-first-node>" -Port 30304 -HttpPort 8547 -WsPort 8548
pwsh -File scripts/check-peering.ps1 -RpcA http://127.0.0.1:8545 -RpcB http://127.0.0.1:8547
```
Expected: both nodes report `net.peerCount > 0`.

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
pwsh -File scripts/init-ethernova.ps1 -Mode dev
pwsh -File scripts/smoke-test-fees.ps1
```
- Datadir: `data-dev\`, easy mining (difficulty 0x1), txpool/miner gasprice 0.
- Use for local testing only; do not point wallets to mainnet RPC when using dev genesis.

## Explorer and wallets
- MetaMask (mainnet): RPC `http://127.0.0.1:8545`, Chain ID `77777`, Symbol `NOVA`, Explorer URL (set to your deployment).
- Suggested explorer: Blockscout. Provide env pointing to your RPC, chainId 77777, and publish the explorer URL alongside bootnodes.

## Release packaging
```
pwsh -File scripts/package-release.ps1
```
Produces `dist/ethernova-<version>-windows.zip` with binaries, genesis files, scripts, and checksums.
