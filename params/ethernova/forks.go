package ethernova

const (
	// EVMCompatibilityForkBlock enables Constantinople + Petersburg + Istanbul.
	EVMCompatibilityForkBlock uint64 = 105000
	// EIP658ForkBlock enables receipt status (EIP-658).
	EIP658ForkBlock uint64 = 110500
	// MegaForkBlock enables missing historical EVM forks for compatibility.
	MegaForkBlock uint64 = 118200

	// ============================================================
	// NOVEN FORK — Ethernova 2.0
	// Named after community developer Noven who built the adaptive
	// gas system and parallel execution classifier.
	// All features below activate at NovenForkBlock.
	// ============================================================

	// NovenForkBlock activates ALL Noven Fork features simultaneously:
	//   - Adaptive Gas V2 (trace-based post-execution adjustment)
	//   - Per-EVM reentrancy guard
	//   - Gas refund on revert (90% execution gas)
	//   - Native precompiles (0x20-0x28)
	//   - State expiry (contract garbage collection)
	//   - Tempo transactions (atomic batching)
	//   - Frame Account Abstraction
	//
	// ALL mainnet nodes MUST upgrade to v2.0.0 BEFORE this block.
	// Nodes running v1.3.x will refuse to start past this block
	// (enforced by ethernovaPatchConfigIfNeeded).
	//
	// Set to 460,000 (~2 days from block 445,183 at 11s/block)
	NovenForkBlock uint64 = 480000

	// AdaptiveGasV2ForkBlock activates trace-based adaptive gas pricing.
	// Replaces the v1 bytecode-based system that caused consensus splits.
	// Pure computation contracts get up to -25% gas discount.
	// Storage-heavy contracts get up to +10% gas penalty.
	// CONSENSUS RULE — all nodes MUST apply the same adjustment.
	AdaptiveGasV2ForkBlock uint64 = 480000

	// StateExpiryForkBlock activates the state expiry garbage collector.
	// Contracts with no activity for StateExpiryPeriod blocks get archived.
	// EOA wallets are NEVER expired. Archived state can be restored.
	StateExpiryForkBlock uint64 = 480000
	// StateExpiryPeriod is the number of blocks of inactivity before archival.
	// ~115 days at 11s/block. Much longer than devnet for safety.
	StateExpiryPeriod uint64 = 900000

	// TempoTxForkBlock activates Tempo-style smart transactions.
	// Enables: atomic batching, fee delegation, scheduled transactions.
	TempoTxForkBlock uint64 = 480000

	// FrameAAForkBlock activates Frame-style Account Abstraction.
	// Precompiles 0x23 (novaFrameApprove) and 0x24 (novaFrameIntrospect).
	FrameAAForkBlock uint64 = 480000
)
