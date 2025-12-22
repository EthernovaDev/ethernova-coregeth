# README_QUICKSTART.md

## Ethernova v1.2.0 Quickstart (Windows + Linux)

This bundle is plug-and-play. You can run a devnet test or start a mainnet node without manual setup.

---

## Devnet One-Click Test (fast fork at block 20)

**Windows**
1) Double-click `scripts\run-devnet-test.bat`
2) It will:
   - init a local devnet
   - mine past the fork
   - run `evmcheck`
   - print PASS/FAIL
3) To keep the devnet running:
   - `powershell -ExecutionPolicy Bypass -File scripts\run-devnet-test.ps1 -KeepRunning`

**Linux**
```bash
./scripts/run-devnet-test.sh
```
To keep the devnet running:
```bash
./scripts/run-devnet-test.sh --keep-running
```

**Devnet warning**
- The devnet private key is public. Never use it on mainnet.
- Key file: `devnet-testkey.txt`
- Devnet chainId/networkId: `177777`

---

## Mainnet Upgrade (config update, no chain reset)

**Windows**
```bat
scripts\apply-upgrade-mainnet.bat
```

**Linux**
```bash
./scripts/apply-upgrade-mainnet.sh
```

This runs:
```
ethernova --datadir <your-datadir> init genesis-upgrade-60000.json
```

Do NOT replace the genesis file in your datadir. The init command updates the stored chain config in-place and preserves the genesis hash.

---

## Run a Mainnet Node

**Windows**
```bat
scripts\run-mainnet-node.bat
```

**Linux**
```bash
./scripts/run-mainnet-node.sh
```

Defaults:
- Data dir: `data-mainnet`
- HTTP RPC: `127.0.0.1:8545`
- WS: `127.0.0.1:8546`

You can edit the scripts to change datadir or ports.
