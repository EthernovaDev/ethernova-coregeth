# OPERATOR_RUNBOOK.md

## Scheduled Hard Fork Upgrade at Blocks 60000 and 70000

This runbook describes how to safely upgrade your node to activate new EVM features at blocks 60000 and 70000 without resetting chain history.

---

## Fork Options

**Minimal DEX fork**
- Enables only the minimum opcodes and gas changes needed for UniswapV2-style deployments.
- EIP-1014 (CREATE2), EIP-1344 (CHAINID), plus Istanbul companion EIPs (145/1052/152/1108/1884/2028/2200).

**Modern EVM fork**
- Includes the minimal DEX fork plus Shanghai/Cancun opcode upgrades commonly required by Solidity 0.8.20+.
- EIP-3198 (BASEFEE opcode), EIP-3651 (warm COINBASE), EIP-3855 (PUSH0), EIP-3860 (initcode limits), EIP-1153 (transient storage), EIP-5656 (MCOPY), EIP-6780 (SELFDESTRUCT changes).
- Does not enable PoS/Merge features or PoS-specific upgrades (no TTD, no EIP-4895 withdrawals, no EIP-4788, no EIP-4844/7516 blobs).
- Fork 70000 adds the missing Byzantium base package (EIP-214 STATICCALL) to fix contract-to-contract view calls.

---

## 1. Stop the Node

**Windows:**
```
Stop-Process -Name ethernova
```
Or close the terminal/window running the node.

**Linux:**
```
killall ethernova
```
Or use your process manager (systemd, supervisor, etc).

---

## 2. Backup the Data Directory

**Windows:**
```
robocopy C:\path\to\datadir C:\path\to\backup\datadir /MIR
```

**Linux:**
```
cp -a /path/to/datadir /path/to/backup/datadir
```

---

## 3. Apply the Genesis Config Upgrade

**Windows:**
```
ethernova.exe --datadir <your-datadir> init genesis-upgrade-70000.json
```

**Linux:**
```
ethernova --datadir <your-datadir> init genesis-upgrade-70000.json
```

- Do NOT replace the genesis file in your datadir. The `init` command updates the stored chain config in-place and preserves the existing genesis hash.
- This updates the stored chain config in-place without wiping chain data as long as the genesis hash is unchanged.
- The tool uses `core.SetupGenesisBlockWithOverride` to safely update the config.
 - The run-mainnet-node scripts also apply the latest upgrade config if present (idempotent).

---

## Mainnet config update (no wipe)

To update the live mainnet config for the Fork70000 schedule without wiping chain data, use:
```
ethernova --datadir <your-datadir> init genesis-upgrade-70000.json
```
This updates the chain config stored in the DB while preserving the genesis hash and full history.

---

## 4. Restart the Node

**Windows:**
```
ethernova.exe --datadir <your-datadir> --networkid 77777 --mine ...
```

**Linux:**
```
ethernova --datadir <your-datadir> --networkid 77777 --mine ...
```

---

## 5. Verify the Upgrade

- Check logs for config update confirmation.
- Use the `cmd/evmcheck` tool to verify CREATE2, CHAINID, and Shanghai/Cancun opcode activation.

---

## What happens if you do not upgrade

- Nodes that skip the fork will reject post-fork blocks and follow a minority chain.
- That causes a chain split between upgraded and non-upgraded nodes.

---

## How to verify pre/post fork

**Pre-fork (block < 60000), expected FAIL:**
```
.\evmcheck.exe --rpc http://HOST:8545 --pk 0xHEX --chainid 77777 --forkblock 60000
```
Expected: `Pre-fork: true`, `CHAINID opcode: FAIL`, `CREATE2 opcode: FAIL`, `PUSH0 opcode: FAIL`, `MCOPY opcode: FAIL`, `TSTORE/TLOAD opcodes: FAIL`, `SELFDESTRUCT (EIP-6780): FAIL`, `EVM upgrade check: FAIL` (exit code 1).

**Post-fork (block >= 60000), expected PASS:**
```
.\evmcheck.exe --rpc http://HOST:8545 --pk 0xHEX --chainid 77777 --forkblock 60000
```
Expected: `Pre-fork: false`, `CHAINID opcode: PASS`, `CREATE2 opcode: PASS`, `PUSH0 opcode: PASS`, `MCOPY opcode: PASS`, `TSTORE/TLOAD opcodes: PASS`, `SELFDESTRUCT (EIP-6780): PASS`, `EVM upgrade check: PASS` (exit code 0).

---

## STATICCALL check (fork 70000)

After upgrading, you can validate STATICCALL activation with the built-in test:
```
go test ./core/vm -run TestStaticcallFork
```

---

**Note:** If you see a genesis hash mismatch, STOP and restore your backup. Do not proceed.
