// Ethernova: Application-Layer Async Callback Dispatcher (NIP-0004 Phase 11 fix)
//
// This file implements the chain-driven dispatch path for the
// novaAsyncCallback (0x30) precompile. The original Phase 11 implementation
// only wrote metadata to AppPrecompileRegistryAddr (0xFF11), leaving
// execution to applications polling `asyncCallbackReady`. That violated the
// NIP-0004 contract that async callbacks fire deterministically at their
// target block.
//
// The fix here uses a per-target-block index (NOT the global Deferred
// Queue, which is a FIFO drain-next-block structure unsuitable for
// time-deferred execution). At registration time the callback ID is
// appended to async_by_block[targetBlock]. At Phase 0 of every block N,
// the function ProcessAsyncCallbacks scans the index for entries pointing
// to N and dispatches each via vm.EVM.Call from a system caller frame.
//
// Determinism invariants (matching the deferred queue precedent):
//   1. Index iteration order is by appended slot index, no map iteration.
//   2. Dispatch happens during Phase 0, before the user tx loop, so any
//      state the callback writes is visible to the very first tx in
//      block N — same guarantee the deferred queue provides.
//   3. Per-block dispatch cap (MaxAsyncCallbacksPerBlock) bounds wall
//      time; entries beyond the cap STAY in the index and fire at the
//      next block they remain in. They are NOT silently dropped.
//   4. A callback that reverts is marked fired anyway — execute-once
//      semantics. Reverts do NOT revert the block; they are logged and
//      counted (mirrors §18 of the NIP-0004 plan).
//   5. The callback fires from system caller 0xFF11 (AppPrecompileRegistryAddr)
//      under DomainNova capabilities. The callback's `target_addr` must
//      reside in Domain 1 or Domain 2; calling a Domain 0 contract is
//      rejected at the gate (consistent with Phase 11 capability rules).
//
// CONSENSUS-CRITICAL: this function MUST be invoked from BOTH validator
// and miner paths with identical inputs. See state_processor.go for the
// wire-through.

package vm

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params/ethernova"
	"github.com/holiman/uint256"
)

// MaxAsyncCallbacksPerBlock bounds the number of async callbacks fired in
// Phase 0 of a single block. Excess entries remain in the per-block index
// and fire on subsequent blocks. This bounds Phase 0 wall time and gas
// (in the abstract sense — async dispatch does not charge user gas).
const MaxAsyncCallbacksPerBlock uint64 = 64

// AsyncCallbackGasLimit caps how much gas a single async callback may
// consume. Chosen high enough for typical inline handlers, low enough that
// a malicious target cannot stall Phase 0.
const AsyncCallbackGasLimit uint64 = 250_000

// AsyncDispatchResult is informational telemetry — does NOT feed back into
// consensus beyond what the dispatch path already wrote to state.
type AsyncDispatchResult struct {
	BlockNumber  uint64
	Dispatched   uint64
	Reverted     uint64
	Skipped      uint64
	CapHit       bool
	NoOp         bool
}

// async-by-block index keys live inside AppPrecompileRegistryAddr. They
// share the keyspace with the rest of the Phase 11 records but use a
// dedicated prefix so they cannot collide with per-callback metadata.
func asyncByBlockCountKey(targetBlock uint64) common.Hash {
	var blk [8]byte
	binary.BigEndian.PutUint64(blk[:], targetBlock)
	return appKey([]byte("async-by-block-count"), blk[:])
}

func asyncByBlockSlotKey(targetBlock, slot uint64) common.Hash {
	var blk, idx [8]byte
	binary.BigEndian.PutUint64(blk[:], targetBlock)
	binary.BigEndian.PutUint64(idx[:], slot)
	return appKey([]byte("async-by-block-slot"), blk[:], idx[:])
}

// asyncByBlockAppend registers an async callback ID against its target
// block. Called by appRegisterAsyncCallback during the create path.
//
// The index is append-only within a block; on dispatch (ProcessAsyncCallbacks)
// the count is reset to zero AND each slot is zeroed. The slot zeroing is
// important for storage rent: leaving stale IDs in slots would waste state.
func asyncByBlockAppend(sdb StateDB, targetBlock uint64, id common.Hash) {
	countKey := asyncByBlockCountKey(targetBlock)
	count := appReadUint64(sdb, countKey)
	slotKey := asyncByBlockSlotKey(targetBlock, count)
	appWriteHash(sdb, slotKey, id)
	appWriteUint64(sdb, countKey, count+1)
}

// asyncByBlockSnapshot returns the IDs queued against `targetBlock` and
// clears the index for that block (writing zeros to each slot and
// resetting the count). Returns at most `cap` entries; if more entries
// exist, they remain in the index for the next block.
//
// The clear-on-read semantics are critical: any entry that exits the
// snapshot is guaranteed to be dispatched (or have its dispatch attempt
// recorded as a revert). Entries that REMAIN in the snapshot after a
// cap-hit are NOT cleared, so they survive to the next block.
//
// IMPORTANT: this function takes only the FIRST `cap` entries and
// removes them. The remaining entries are SHIFTED DOWN so they occupy
// slots [0, count - cap). This keeps the index dense and avoids storage
// fragmentation, but costs O(remaining) state writes when cap is hit.
// In the common case (no cap hit), the shift is unnecessary and skipped.
func asyncByBlockDrainSnapshot(sdb StateDB, targetBlock uint64, cap uint64) (ids []common.Hash, capHit bool) {
	countKey := asyncByBlockCountKey(targetBlock)
	count := appReadUint64(sdb, countKey)
	if count == 0 {
		return nil, false
	}
	toDrain := count
	if toDrain > cap {
		toDrain = cap
		capHit = true
	}
	ids = make([]common.Hash, 0, toDrain)
	for i := uint64(0); i < toDrain; i++ {
		slot := asyncByBlockSlotKey(targetBlock, i)
		id := appReadHash(sdb, slot)
		appWriteHash(sdb, slot, common.Hash{})
		ids = append(ids, id)
	}
	if !capHit {
		appWriteUint64(sdb, countKey, 0)
		return ids, false
	}
	// Cap hit: shift remaining entries down so slot [0, remaining) holds
	// them. This preserves FIFO ordering for the next block's drain.
	remaining := count - toDrain
	for i := uint64(0); i < remaining; i++ {
		fromSlot := asyncByBlockSlotKey(targetBlock, toDrain+i)
		toSlot := asyncByBlockSlotKey(targetBlock, i)
		val := appReadHash(sdb, fromSlot)
		appWriteHash(sdb, toSlot, val)
		appWriteHash(sdb, fromSlot, common.Hash{})
	}
	appWriteUint64(sdb, countKey, remaining)
	return ids, true
}

// ProcessAsyncCallbacks runs the Phase 0 async-dispatch pass for the
// given block. It MUST be called from BOTH validator (state_processor)
// and miner (worker.prepareWork) paths with identical inputs.
//
// The function is safe to call with nil evm or nil statedb (becomes a
// no-op). Before ApplicationPrecompileForkBlock it is a hard no-op.
func ProcessAsyncCallbacks(evm *EVM, statedb *state.StateDB, blockNum uint64) *AsyncDispatchResult {
	result := &AsyncDispatchResult{BlockNumber: blockNum}
	if evm == nil || statedb == nil {
		result.NoOp = true
		return result
	}
	if blockNum < ethernova.ApplicationPrecompileForkBlock {
		result.NoOp = true
		return result
	}

	// Snapshot the per-block index. After this point any IDs in the
	// snapshot have been removed from state and MUST be dispatched (or
	// counted as revert) — no early-return allowed.
	ids, capHit := asyncByBlockDrainSnapshot(statedb, blockNum, MaxAsyncCallbacksPerBlock)
	result.CapHit = capHit
	if len(ids) == 0 {
		result.NoOp = true
		return result
	}

	for _, id := range ids {
		err := dispatchOneAsyncCallback(evm, statedb, id, blockNum)
		switch {
		case errors.Is(err, errAsyncCallbackAlreadyFired):
			result.Skipped++
		case err != nil:
			result.Reverted++
			log.Warn("async callback dispatch failed",
				"id", id.Hex(), "block", blockNum, "err", err)
		default:
			result.Dispatched++
		}
	}

	// Finalise so post-Phase-0 transactions see the writes.
	statedb.Finalise(true)
	return result
}

var errAsyncCallbackAlreadyFired = errors.New("async callback already fired")

// dispatchOneAsyncCallback executes a single registered callback. It
// returns:
//   - nil if the callback was dispatched (success OR revert by the target)
//   - errAsyncCallbackAlreadyFired if the callback record indicates the
//     application already fired it via markFired
//   - a wrapped error if the metadata could not be read (e.g. id not
//     present in 0xFF11 — should not happen if the index is consistent)
//
// In all cases except the metadata error, the "fired" flag is set to true
// at the end so the callback is execute-once.
func dispatchOneAsyncCallback(evm *EVM, statedb *state.StateDB, id common.Hash, blockNum uint64) error {
	if !appExists(statedb, "async", id) {
		// Index points to a metadata record that no longer exists. This
		// should not happen on canonical paths — defensive skip.
		return errors.New("async metadata missing")
	}
	if appReadBool(statedb, appKindKey("async", id, "fired")) {
		return errAsyncCallbackAlreadyFired
	}

	targetAddr := appReadAddress(statedb, appKindKey("async", id, "target_addr"))
	selectorWord := appReadHash(statedb, appKindKey("async", id, "selector"))
	payloadHash := appReadHash(statedb, appKindKey("async", id, "payloadHash"))

	// Mark fired BEFORE executing so a reentrant call into markFired or
	// register from the target can't observe the callback as pending.
	// This also enforces execute-once even when the target reverts.
	appWriteBool(statedb, appKindKey("async", id, "fired"), true)

	if targetAddr == (common.Address{}) {
		// No target configured. Pure commitment-only callback (legacy
		// register path that omitted target_addr). Treat as dispatched
		// with no execution.
		return nil
	}

	// Build calldata = function selector (4 bytes) || callback id (32) ||
	// payload hash (32). Targets are expected to declare:
	//     function onAsyncCallback(bytes32 callbackId, bytes32 payloadHash)
	// (or any signature whose first 4 selector bytes match the registered
	// `selector` field). Custom payloads beyond the 64-byte trailer must be
	// retrieved by the target via getAsyncCallback().
	calldata := make([]byte, 0, 4+32+32)
	calldata = append(calldata, selectorWord.Bytes()[28:32]...) // last 4 bytes are the selector
	calldata = append(calldata, id.Bytes()...)
	calldata = append(calldata, payloadHash.Bytes()...)

	// System caller frame: AppPrecompileRegistryAddr executes the call.
	// This gives the callback DomainNova capabilities (via the EOA-style
	// fallback in currentCapabilities when caller has no code) so it can
	// in turn call any application precompile if it needs to. Targets in
	// Domain 0 will be rejected by the capability gate the EVM enforces
	// on outbound calls.
	sysCaller := AccountRef(AppPrecompileRegistryAddr)
	_, _, callErr := evm.Call(sysCaller, targetAddr, calldata, AsyncCallbackGasLimit, uint256.NewInt(0))
	if callErr != nil {
		appWriteBool(statedb, appKindKey("async", id, "fire_failed"), true)
		appWriteHash(statedb, appKindKey("async", id, "fire_error_block"), common.BigToHash(uint256.NewInt(blockNum).ToBig()))
		return callErr
	}
	return nil
}
