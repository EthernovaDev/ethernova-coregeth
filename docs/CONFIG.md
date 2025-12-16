# Ethernova Node Configuration

## RPC
- Default bind: `127.0.0.1`.
- HTTP APIs:
  - Dev: `eth,net,web3,personal,miner,txpool,admin,debug`.
  - Mainnet: `eth,net,web3` (expand only if necessary).
- Do not expose RPC publicly without authentication/proxy.

## Ports
- p2p: 30303 (UDP/TCP)
- HTTP: 8545 (localhost)
- WS: 8546 (localhost)

## Data and logs
- Datadirs (default): `data-dev\` for dev, `data-mainnet\` for mainnet (under repo root unless `-Root` is provided).
- Logs: `logs\node.log`, `logs\node.err.log`

## IPC / Auth RPC
- IPC paths: `\\.\pipe\ethernova-dev.ipc` (dev), `\\.\pipe\ethernova-mainnet.ipc` (mainnet), `\\.\pipe\ethernova-node2-*.ipc` for secondary nodes.
- Auth RPC: main node binds `127.0.0.1:8551`; second node uses `127.0.0.1:8552` (see scripts).

## Mining
- Etherbase configured in `scripts/init-ethernova.ps1` (`$Miner`).
- Dev mode: gasprice 0, txpool pricelimit 0.
- Mainnet mode: gasprice default 1 gwei; txpool pricelimit default (non-zero).

## Fees
- EIP-1559 baseFee is redirected to the configured `baseFeeVault`; tips remain with the miner.

## Peering
- Bootnodes: `networks/mainnet/bootnodes.txt` (enodes, one per line).
- Static peers: `networks/mainnet/static-nodes.json` (JSON array). Copied to `data/geth/static-nodes.json` if present.
- Discover peers: `admin.nodeInfo.enode` to share your enode.

## Genesis files
- `genesis-dev.json`: difficulty=0x1, forks at block 0, baseFeeVault set, chainId/networkId 77778 (dev/testnet).
- `genesis-mainnet.json`: same forks, higher difficulty (0x400000), baseFeeVault set, chainId/networkId 77777.
- Block reward schedule (both): 10 -> 5 -> 2.5 -> 1.25 -> floor 1 NOVA, halving every ~2,102,400 blocks (~1 year at ~15s/block).

## Security
- Avoid `--allow-insecure-unlock` on anything except isolated dev.
- Keep keystore backed up; scripts auto-backup before wiping datadir.
- Use firewall rules to restrict RPC/WS to localhost if running on shared hosts.
