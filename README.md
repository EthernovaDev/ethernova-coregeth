<p align="center">
  <img src="https://ethnova.net/favicon.png" width="80" alt="Ethernova">
</p>

<h1 align="center">Ethernova</h1>

<p align="center">
  <strong>A smarter Ethereum. Same compatibility, better economics.</strong>
</p>

<p align="center">
  <a href="https://github.com/EthernovaDev/ethernova-coregeth/releases/latest"><img src="https://img.shields.io/github/v/release/EthernovaDev/ethernova-coregeth?label=latest&color=blue" alt="Latest Release"></a>
  <a href="https://ethnova.net"><img src="https://img.shields.io/badge/website-ethnova.net-purple" alt="Website"></a>
  <a href="https://explorer.ethnova.net"><img src="https://img.shields.io/badge/explorer-live-green" alt="Explorer"></a>
</p>

---

## What is Ethernova?

Ethernova is an **Ethereum-compatible Layer 1 blockchain** that keeps everything developers love about Ethereum (Solidity, EVM, existing tools) while fixing what they don't: **unfair gas pricing, no reentrancy protection, no native account abstraction, and no built-in state management**.

Every smart contract deployed on Ethereum works on Ethernova with zero modifications. But on Ethernova, simple operations cost less, complex operations are priced fairly, and the protocol does more for you at Layer 1.

**Chain ID:** `121525` | **Consensus:** Ethash PoW | **Block time:** ~11s | **Currency:** NOVA

---

## Why Ethernova over vanilla Ethereum?

### The problem with Ethereum's gas model

On Ethereum, a simple token transfer and a complex DEX swap with 10 storage writes pay gas based only on raw opcode costs. There's no distinction between a lightweight `view` function and a storage-heavy arbitrage bot. Validators don't benefit from processing simple transactions, and users don't benefit from writing efficient contracts.

### How Ethernova fixes it

| Feature | Ethereum | Ethernova |
|---------|----------|-----------|
| **Gas pricing** | Flat — same cost regardless of complexity | **Adaptive** — simple contracts get up to 25% discount, heavy contracts pay up to 10% more |
| **Reentrancy protection** | None at protocol level (devs must add their own) | **Built-in** — native per-call reentrancy guard prevents the #1 exploit in DeFi |
| **Failed transaction cost** | You pay full gas even when tx reverts | **90% refund** on revert for simple transactions (anti-DoS protected) |
| **Account abstraction** | Requires ERC-4337 infrastructure + bundlers | **Native** — Frame-style AA at protocol level, no external infrastructure |
| **Multi-token support** | Every token is a separate contract | **Native precompile** — create and transfer tokens at protocol speed |
| **State management** | State grows forever, no cleanup | **State expiry** — inactive contracts archived after ~115 days, recoverable with proof |
| **Batch operations** | One operation per transaction | **Tempo transactions** — atomic batching, fee delegation, scheduled execution |
| **MEV protection** | Miners can reorder for profit | **Fair ordering** — FIFO arrival + rate limiting prevents sandwich attacks |
| **Oracle support** | External services only (Chainlink, etc.) | **Native oracle precompile** — protocol-level price feeds with TWAP |
| **Contract upgrades** | Proxy pattern (complex, error-prone) | **Native upgrade precompile** — built-in timelock upgrades without proxies |

---

## Ethernova 2.0 — The Noven Fork

Version 2.0 introduces the **Noven Fork**, named after community developer [Noven](https://github.com/novenrizkia) who designed and built the adaptive gas system. It activates at **mainnet block 480,000** and brings:

### Adaptive Gas V2 (Consensus Rule)

The core innovation. After every contract execution, Ethernova classifies the transaction based on what it *actually did* (not what the bytecode contains):

```
Pure computation (math, hashing)     →  up to -25% gas discount
Light operations (reads, view calls) →  up to -15% gas discount
Mixed operations (moderate storage)  →  no adjustment
Complex operations (heavy storage)   →  up to +10% gas penalty
```

This is **fully deterministic** — the classification is a pure function of execution trace counters (SSTORE, SLOAD, CALL counts). Same transaction always produces the same adjustment on every node. No randomness, no floating point, no global state.

**Why this matters:** Developers who write efficient contracts are rewarded. Users calling simple functions pay less. The network naturally incentivizes gas-efficient code.

### 9 Native Precompiles (0x20-0x28)

Protocol-level operations that would be expensive or impossible as smart contracts:

| Address | Name | What it does | Gas |
|---------|------|-------------|-----|
| `0x20` | BatchHash | Batch keccak256 — hash multiple items in one call | 30/item |
| `0x21` | BatchVerify | Batch ecrecover — verify multiple signatures at once | 2000/sig |
| `0x22` | AccountManager | Native key rotation + guardian recovery for wallets | varies |
| `0x23` | FrameApprove | Frame-style account abstraction approval | 5000 |
| `0x24` | FrameIntrospect | Cross-frame transaction inspection | 2000 |
| `0x25` | TokenManager | Create and transfer native multi-tokens | varies |
| `0x26` | ShieldedPool | Optional privacy — shield/unshield NOVA | 50k-100k |
| `0x27` | ContractUpgrade | Native contract upgrades with timelock | 50000 |
| `0x28` | Oracle | Protocol-level price feeds with TWAP | 2000-5000 |

### Per-EVM Reentrancy Guard

Every contract call is protected against self-reentrancy at the protocol level. No more `ReentrancyGuard` modifiers needed — the EVM itself blocks a contract from calling itself while it's already executing. Cross-contract calls (A calls B calls C) work normally.

### State Expiry

Contracts and tokens with no activity for ~115 days (900,000 blocks) are automatically archived. EOA wallets are **never** expired. Archived state can be restored with a Merkle proof. This prevents the state from growing unbounded — the #1 scalability concern for all EVM chains.

### Tempo Transactions

Batch multiple operations into a single atomic transaction:
- **Atomic batching** — approve + swap in one tx (both succeed or both revert)
- **Fee delegation** — someone else pays your gas
- **Scheduled execution** — execute at a future block
- Up to 16 calls per batch, all-or-nothing execution

### Anti-MEV Fair Ordering

Transactions are ordered by arrival time (FIFO), not by gas price bidding. Rate limiting prevents spam flooding while maintaining fairness. This eliminates sandwich attacks and front-running at the protocol level.

---

## Full Ethereum Compatibility

Ethernova is **not a sidechain or L2** — it's an independent Layer 1 with full EVM compatibility:

- Deploy any Solidity/Vyper contract — zero modifications needed
- Use MetaMask, Hardhat, Foundry, Remix, ethers.js, web3.py — all work out of the box
- Same RPC API as Ethereum (`eth_*`, `net_*`, `web3_*`) plus `ethernova_*` extensions
- Same transaction format, same signing, same address derivation
- Run existing DeFi protocols (Uniswap, Aave, etc.) — they just work

**What you add:** 30+ new RPC endpoints under the `ethernova_*` namespace for adaptive gas stats, execution profiling, parallel analysis, and protocol configuration.

---

## Network Info

| Property | Value |
|----------|-------|
| Chain ID | `121525` (0x1DAB5) |
| Currency | NOVA |
| Consensus | Ethash PoW |
| Block time | ~11 seconds |
| Block reward | 10 NOVA (halving every ~2.1M blocks) |
| EIP-1559 | Active from genesis |
| RPC (official) | `https://rpc.ethnova.net` |
| RPC (XBiNodes) | `https://nova-rpc.xbinodes.com` |
| Explorer | [explorer.ethnova.net](https://explorer.ethnova.net) |
| Website | [ethnova.net](https://ethnova.net) |

### Fork Schedule

| Block | Fork | Features |
|-------|------|----------|
| 0 | Genesis | EIP-1559, EIP-155, Berlin suite |
| 105,000 | Constantinople | SHL, SHR, CREATE2, CHAINID |
| 110,500 | EIP-658 | Receipt status field |
| 118,200 | Mega Fork | Full EVM compatibility |
| **480,000** | **Noven Fork** | **Adaptive Gas V2, Precompiles, State Expiry, Tempo TX, Frame AA** |

---

## Quick Start

### Download

Get the latest binary from [Releases](https://github.com/EthernovaDev/ethernova-coregeth/releases/latest).

### Run a node

**Windows:**
```
ethernova.exe --networkid 121525 --http --http.api eth,net,web3,ethernova
```

**Linux:**
```
./ethernova --networkid 121525 --http --http.api eth,net,web3,ethernova
```

Genesis is embedded and verified automatically. No manual init required.

### Connect MetaMask

| Field | Value |
|-------|-------|
| Network Name | Ethernova |
| RPC URL | `https://rpc.ethnova.net` |
| Chain ID | `121525` |
| Currency Symbol | NOVA |
| Explorer | `https://explorer.ethnova.net` |

### Mining

```
ethernova --networkid 121525 --mine --miner.etherbase 0xYOUR_ADDRESS
```

For GPU mining, use a stratum proxy between your miner (T-Rex, lolMiner, etc.) and the node.

---

## RPC Extensions

Ethernova adds 30+ endpoints under the `ethernova` namespace:

```
ethernova_forkStatus        — Fork activation status
ethernova_adaptiveGasV2     — Current gas adjustment stats
ethernova_executionMode     — Execution mode info
ethernova_parallelStats     — Transaction parallelism analysis
ethernova_precompiles       — List all native precompiles
ethernova_stateExpiry       — State expiry configuration
ethernova_nodeHealth        — Node health metrics
```

Enable with `--http.api eth,net,web3,ethernova`.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Ethernova Node                     │
├─────────────────────────────────────────────────────┤
│  EVM (full Ethereum compatibility)                   │
│  ├── TraceCounters (per-tx opcode recording)         │
│  ├── Per-EVM Reentrancy Guard                        │
│  └── 9 Native Precompiles (0x20-0x28)               │
├─────────────────────────────────────────────────────┤
│  Adaptive Gas V2 (post-execution adjustment)         │
│  ├── Classification: Pure / Light / Mixed / Complex  │
│  ├── Discount: up to -25% for pure computation       │
│  └── Penalty: up to +10% for heavy storage           │
├─────────────────────────────────────────────────────┤
│  Tempo Transactions (atomic batching)                │
│  State Expiry (contract garbage collection)          │
│  Anti-MEV Fair Ordering (FIFO + rate limiting)       │
├─────────────────────────────────────────────────────┤
│  Ethash PoW Consensus                                │
│  EIP-1559 Fee Market                                 │
│  Base Fee Vault (protocol treasury)                  │
└─────────────────────────────────────────────────────┘
```

---

## Building from Source

```bash
# Requirements: Go 1.21+, GCC (for CGO)
git clone https://github.com/EthernovaDev/ethernova-coregeth.git
cd ethernova-coregeth
make geth
# Binary at ./build/bin/geth
```

---

## Documentation

- [Noven Fork details](docs/NOVEN-FORK.md)
- [Upgrade guide](docs/UPGRADE-v2.0.0.md)
- [Configuration reference](docs/CONFIG.md)
- [RPC API reference](docs/RPC-API.md)

---

## Community

- Website: [ethnova.net](https://ethnova.net)
- Explorer: [explorer.ethnova.net](https://explorer.ethnova.net)
- GitHub: [EthernovaDev](https://github.com/EthernovaDev)

---

## Credits

Ethernova 2.0 was built by the Ethernova team with major contributions from:
- **Noven** ([novenrizkia](https://github.com/novenrizkia)) — Adaptive Gas V2, parallel execution classifier, consensus fixes
- **XBiNodes** ([xbinodes.com](https://xbinodes.com)) — Infrastructure partner, public RPC node

Based on [CoreGeth](https://github.com/etclabscore/core-geth) (downstream of [go-ethereum](https://github.com/ethereum/go-ethereum)).

---

## License

- Library code: GNU LGPL-3.0-or-later
- Binaries: GNU GPL-3.0-or-later

See `LICENSE`, `COPYING`, and `COPYING.LESSER`.
