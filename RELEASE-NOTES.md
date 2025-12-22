# Ethernova v1.2.0

- Fork60000 upgrade files (mainnet config update + devnet fork-20).
- Modern EVM opcode subset for 2025 contracts (Shanghai/Cancun: PUSH0, MCOPY, TSTORE/TLOAD, SELFDESTRUCT changes, initcode limits, warm coinbase).
- Expanded `evmcheck` with on-chain opcode verification and PASS/FAIL output.
- Operator runbook updates and fork-specific release notes.
- One-click devnet test/run scripts for Windows and Linux bundles.

# Ethernova v1.0.0-nova

- Ethash PoW chain with EIP-1559 baseFee redirected to treasury vault `0x3a38560b66205bb6a31decbcb245450b2f15d4fd`; tips remain with miners.
- Block reward schedule: starts at 10 NOVA, halves every ~2,102,400 blocks (~1 year @15s) with a 1 NOVA floor (encoded in genesis).
- Forks active from block 0: Berlin/London (type-2 tx, baseFee active).
- Genesis files: `genesis-mainnet.json` (chainId 77777, difficulty 0x400000, extraData "NOVA MAINNET") and `genesis-dev.json` (chainId 77778, difficulty 0x1).
- Windows scripts: build, init, smoke test, bootnode, second node, peering check, genesis fingerprint, release packaging.
