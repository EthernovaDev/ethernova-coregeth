// Ethernova: Application-Layer Async Callback Processor Wrapper
// (NIP-0004 Phase 11 fix, BUG-1)
//
// This file lives in package `core` (not `core/vm`) because it owns the
// construction of the per-block EVM environment used to dispatch
// application-layer async callbacks. The actual dispatch logic is in
// vm.ProcessAsyncCallbacks; this wrapper:
//
//   1. Builds a deterministic vm.EVM rooted at the current header.
//   2. Calls vm.ProcessAsyncCallbacks(evm, statedb, blockNum).
//   3. Returns the dispatch result for telemetry/logging.
//
// CONSENSUS-CRITICAL: this function MUST be invoked from BOTH validator
// (state_processor.go) and miner (miner/worker.go) paths with identical
// inputs. SafeTuner-style asymmetry between the two paths IS a consensus
// split (see comments at top of ethernova_deferred_processing.go).

package core

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params/ethernova"
	"github.com/ethereum/go-ethereum/params/types/ctypes"
)

// ProcessApplicationAsyncCallbacks runs the Phase 11 application-layer
// async-callback dispatch pass for the given block. Returns nil if the
// inputs are degenerate (nil header/statedb) — the result is informational
// only; consensus is anchored by the state writes made inside the EVM.
//
// Before ApplicationPrecompileForkBlock the function is a hard no-op.
func ProcessApplicationAsyncCallbacks(header *types.Header, statedb *state.StateDB, bc ChainContext, config ctypes.ChainConfigurator) *vm.AsyncDispatchResult {
	if header == nil || statedb == nil {
		return nil
	}
	blockNum := header.Number.Uint64()
	if blockNum < ethernova.ApplicationPrecompileForkBlock {
		return &vm.AsyncDispatchResult{BlockNumber: blockNum, NoOp: true}
	}

	// Build a dedicated EVM for the dispatch pass. Using a fresh EVM
	// (rather than borrowing the validator's `vmenv`) keeps the dispatch
	// frame independent of any tx-level state on `vmenv` — e.g. the
	// resource meter, reentrancy guard, and adaptive-gas counters.
	context := NewEVMBlockContext(header, bc, nil)
	evm := vm.NewEVM(context, vm.TxContext{}, statedb, config, vm.Config{})
	return vm.ProcessAsyncCallbacks(evm, statedb, blockNum)
}
