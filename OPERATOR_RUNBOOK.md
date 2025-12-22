# OPERATOR_RUNBOOK.md

## Scheduled Hard Fork Upgrade at Block 60000

This runbook describes how to safely upgrade your node to activate EIP-1014 (CREATE2), EIP-1344 (CHAINID), and companion EIPs at block 60000 without resetting chain history.

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
ethernova.exe --datadir <your-datadir> init genesis-upgrade-60000.json
```

**Linux:**
```
ethernova --datadir <your-datadir> init genesis-upgrade-60000.json
```

- This updates the stored chain config in-place without wiping chain data as long as the genesis hash is unchanged.
- The tool uses `core.SetupGenesisBlockWithOverride` to safely update the config.

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
- Use the `cmd/evmcheck` tool to verify CREATE2 and CHAINID activation.

---

## How to verify pre/post fork

**Pre-fork (block < 60000), expected FAIL:**
```
.\evmcheck.exe --rpc http://HOST:8545 --pk 0xHEX --chainid 77777 --forkblock 60000
```
Expected: `Pre-fork: true`, `CHAINID opcode: FAIL`, `CREATE2 opcode: FAIL`, `EVM upgrade check: FAIL` (exit code 1).

**Post-fork (block >= 60000), expected PASS:**
```
.\evmcheck.exe --rpc http://HOST:8545 --pk 0xHEX --chainid 77777 --forkblock 60000
```
Expected: `Pre-fork: false`, `CHAINID opcode: PASS`, `CREATE2 opcode: PASS`, `EVM upgrade check: PASS` (exit code 0).

---

**Note:** If you see a genesis hash mismatch, STOP and restore your backup. Do not proceed.
