# NIP-0004 Mainnet Activation Audit - Block 853400

Date: 2026-05-29
Branch: `nip0004-mainnet-853400`
Scope: port the completed NIP-0004 devnet implementation to mainnet source without changing live mainnet behavior before activation.

## Activation Decision

- Mainnet Chain ID / Network ID: `121525`
- Selected activation block: `853400`
- Live RPC snapshot used for final sanity check:
  - RPC: `https://rpc.ethnova.net`
  - Current block at check: `803581`
  - Remaining blocks: `49819`
  - Approx ETA at 11s/block: `2026-06-05T00:20:51Z`

This follows the requested `current_block + ~50000` rollout window so all operators can upgrade before the hard fork.

## Mainnet Parameters Applied

All NIP-0004 consensus feature gates are set to block `853400`:

- `ExecutionDomainForkBlock`
- `ProtocolObjectForkBlock`
- `DeferredExecForkBlock`
- `ContentRefForkBlock`
- `MailboxForkBlock`
- `StateLifecycleForkBlock`
- `LifecycleSloadSurchargeForkBlock`
- `SessionForkBlock`
- `ApplicationPrecompileForkBlock`
- `NovaOpcodeForkBlock`
- `ResourceMeteringForkBlock`

Phase 5 mainnet lifecycle parameters:

- `ActiveTierBlocks = 100000`
- `WarmTierBlocks = 1000000`
- `ColdTierBlocks = 10000000`
- `WarmingFeePerByte = 50`
- `MaxLifecycleSweepPerBlock = 2048`

Existing mainnet Ethernova v2.0 fork blocks were kept at their live mainnet values:

- `NovenForkBlock = 480000`
- `AdaptiveGasV2ForkBlock = 480000`
- `StateExpiryForkBlock = 480000`
- `StateExpiryPeriod = 900000`
- `TempoTxForkBlock = 480000`
- `FrameAAForkBlock = 480000`

## Mainnet Safety Fixes Added During Port

- Domain EF01/EF02 bytecode prefixes are interpreted only after `ExecutionDomainForkBlock`.
- Before activation, EF-prefixed runtime bytecode keeps legacy EVM behavior and EIP-3541 rejection.
- EOA create defaults to Domain 1 only after the domain fork; pre-fork create remains legacy.
- Capability enforcement is a no-op before the domain fork, so historical execution is not reinterpreted by Phase 6.
- Deferred processing tests now use a real post-fork block and also verify pre-fork no-op behavior with queued entries left untouched.
- Phase 10D consensus helper `consensus/misc.CalcNextResourcePrice` was ported with the resource-metering code.

## Validation Log

```text
$ go test ./params/ethernova ./core/types ./core/vm ./core/state ./consensus/ethash ./consensus/lyra2
ok   github.com/ethereum/go-ethereum/params/ethernova  (cached)
ok   github.com/ethereum/go-ethereum/core/types        (cached)
ok   github.com/ethereum/go-ethereum/core/vm           (cached)
ok   github.com/ethereum/go-ethereum/core/state        (cached)
ok   github.com/ethereum/go-ethereum/consensus/ethash  221.566s
ok   github.com/ethereum/go-ethereum/consensus/lyra2   0.609s

$ go test ./core/vm/runtime -run 'TestExecutionDomain|TestDomain|TestCreateDomain|TestEIP3541|TestCapabilities'
ok   github.com/ethereum/go-ethereum/core/vm/runtime   0.423s

$ go test ./core -run 'TestDeferredProcessing'
ok   github.com/ethereum/go-ethereum/core              0.575s

$ go test -run '^$' ./core/... ./eth/... ./internal/ethapi ./miner ./params/... ./consensus/... ./cmd/geth
ok/compile-only across core, eth, internal/ethapi, miner, params, consensus, cmd/geth

$ git diff --check
(no output)
```

## Build Artifacts Generated Locally

```text
62b6bd4ee578cc4d2b28b42133ab999b622c038f04f7f34d361c558a79f0de63  dist/ethernova-nip0004-mainnet-853400-darwin-arm64
2fe515b9e9cea3cdcc6684148c60a4d1a5848fcce70bf396405a3c4634e42d3b  dist/ethernova-nip0004-mainnet-853400-linux-amd64
b4f2effe3bce103a4972cf30c496ce53474583ad626f7251d01c569db8433e36  dist/ethernova-nip0004-mainnet-853400-windows-amd64.exe
```

Local native smoke:

```text
$ ./dist/ethernova-nip0004-mainnet-853400-darwin-arm64 version
Ethernova
Version: v2.0.0
Git Commit: nip0004-mainnet-853400
Git Commit Date: 20260529
Architecture: arm64
Go Version: go1.21.13
Operating System: darwin
```

## Operational Notes Before Mainnet Rollout

- Do not activate by deploying only one node. This is a mandatory hard fork; all five nodes plus RPC/archive VPS should upgrade before block `853400`.
- Public RPC/archive service must expose the NIP RPC namespaces needed after activation, including `ethernova` and `nova` where applicable.
- Keep the archive RPC node in `--gcmode archive`; `ethernova_getStateWitness` requires archive-quality data.
- Do not set any NIP-0004 fork block to `0` on mainnet.
- If the chain advances too close to `853400` before all operators are ready, choose a later activation block and rebuild.

## Status

Ready as a candidate mainnet activation patch, pending operator rollout coordination and any final external audit review from Noven.
