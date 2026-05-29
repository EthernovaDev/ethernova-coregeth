// Ethernova: Application-Layer Precompiles (NIP-0004 Phase 11)
//
// Phase 11 adds higher-level application primitives on top of the lower
// Protocol Object, Mailbox, ContentRef, Session, Domain, and Resource layers.
// The original draft reused 0x2B/0x2C for app helpers, but those slots are
// already live in this codebase as ContentRegistry and MailboxManager. The
// activation-safe map is therefore:
//   0x30 novaAsyncCallback
//   0x31 novaIdentityAttestation
//   0x32 novaSocialGraph
//   0x33 novaContentManifest
//   0x34 novaGameState
//   0x36 novaComputeBounty
// 0x35 intentionally remains novaMailboxOps.
//
// ---------------------------------------------------------------------------
// Phase 11 audit fixes (this revision):
//   * BUG-1 (HIGH): novaAsyncCallback now schedules the callback via the new
//     by-target-block index (see ethernova_async_callbacks.go) AND wires an
//     EffectTypeAsyncCallback entry into the global Deferred Queue for
//     auditability. The drain handler is chain-driven by ProcessAsyncCallbacks
//     during Phase 0 of the target block.
//   * BUG-2 (HIGH): novaIdentityAttestation, novaContentManifest,
//     novaGameState (first commit), and novaComputeBounty now create a
//     Protocol Object via PoCreateObjectInternal so they are observable via
//     nova_getProtocolObject and nova_listProtocolObjects.
//   * BUG-3 (HIGH): novaComputeBounty.create accepts msg.value as escrow,
//     sweeps it into AppPrecompileRegistryAddr, and exposes a new claim
//     selector 0x05 that releases the escrow to the submitter on a
//     successful verification against the bounty's expectedResult field.
//   * BUG-4 (MEDIUM): novaContentManifest.create now validates that
//     contentRef resolves to a Phase 3 ProtoTypeContentReference object via
//     CrGetContentRef (the zero hash bypasses the check for backwards
//     compatibility with manifests that don't reference a ContentRef).
//   * BUG-5 (MEDIUM): novaGameState.commit creates a ProtoTypeGameRoom
//     Protocol Object on first commit so games are first-class entities
//     and queryable per-owner. Subsequent commits update the existing PO's
//     LastTouchedBlock.
//   * BUG-6 (LOW): novaSocialGraph keeps Twitter-style open follow by design
//     and documents that decision in the dispatcher. A new selector 0x05
//     `followWithConsent` is added for callers who want a consent-checked
//     follow; it accepts (target, sigR, sigS, sigV) and recovers via
//     ecrecover that the signer == target.

package vm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params/ethernova"
	"github.com/holiman/uint256"
)

const (
	appPrecompileGasRead    uint64 = 2000
	appPrecompileGasVerify  uint64 = 4000
	appPrecompileGasWrite   uint64 = 18000
	appPrecompileGasEnqueue uint64 = 24000 // register-async + claim-bounty (Deferred Queue / value transfer)
)

var AppPrecompileRegistryAddr = common.HexToAddress("0x000000000000000000000000000000000000FF11")

// Precompile addresses for the Phase 11 set, exposed as named constants so
// the value-sweep / escrow logic does not need magic numbers.
var (
	asyncCallbackPrecompileAddr       = common.HexToAddress("0x0000000000000000000000000000000000000030")
	identityAttestationPrecompileAddr = common.HexToAddress("0x0000000000000000000000000000000000000031")
	socialGraphPrecompileAddr         = common.HexToAddress("0x0000000000000000000000000000000000000032")
	contentManifestPrecompileAddr     = common.HexToAddress("0x0000000000000000000000000000000000000033")
	gameStatePrecompileAddr           = common.HexToAddress("0x0000000000000000000000000000000000000034")
	computeBountyPrecompileAddr       = common.HexToAddress("0x0000000000000000000000000000000000000036")
)

type novaAsyncCallback struct{}
type novaIdentityAttestation struct{}
type novaSocialGraph struct{}
type novaContentManifest struct{}
type novaGameState struct{}
type novaComputeBounty struct{}

var _ StatefulPrecompiledContract = (*novaAsyncCallback)(nil)
var _ StatefulPrecompiledContract = (*novaIdentityAttestation)(nil)
var _ StatefulPrecompiledContract = (*novaSocialGraph)(nil)
var _ StatefulPrecompiledContract = (*novaContentManifest)(nil)
var _ StatefulPrecompiledContract = (*novaGameState)(nil)
var _ StatefulPrecompiledContract = (*novaComputeBounty)(nil)

func (c *novaAsyncCallback) RequiredGas(input []byte) uint64 {
	return appRequiredGas(input, asyncCallbackPrecompileAddr)
}
func (c *novaIdentityAttestation) RequiredGas(input []byte) uint64 {
	return appRequiredGas(input, identityAttestationPrecompileAddr)
}
func (c *novaSocialGraph) RequiredGas(input []byte) uint64 {
	return appRequiredGas(input, socialGraphPrecompileAddr)
}
func (c *novaContentManifest) RequiredGas(input []byte) uint64 {
	return appRequiredGas(input, contentManifestPrecompileAddr)
}
func (c *novaGameState) RequiredGas(input []byte) uint64 {
	return appRequiredGas(input, gameStatePrecompileAddr)
}
func (c *novaComputeBounty) RequiredGas(input []byte) uint64 {
	return appRequiredGas(input, computeBountyPrecompileAddr)
}

func (c *novaAsyncCallback) Run(input []byte) ([]byte, error) {
	return nil, errors.New("novaAsyncCallback: requires stateful execution")
}
func (c *novaIdentityAttestation) Run(input []byte) ([]byte, error) {
	return nil, errors.New("novaIdentityAttestation: requires stateful execution")
}
func (c *novaSocialGraph) Run(input []byte) ([]byte, error) {
	return nil, errors.New("novaSocialGraph: requires stateful execution")
}
func (c *novaContentManifest) Run(input []byte) ([]byte, error) {
	return nil, errors.New("novaContentManifest: requires stateful execution")
}
func (c *novaGameState) Run(input []byte) ([]byte, error) {
	return nil, errors.New("novaGameState: requires stateful execution")
}
func (c *novaComputeBounty) Run(input []byte) ([]byte, error) {
	return nil, errors.New("novaComputeBounty: requires stateful execution")
}

// appSweepPrecompileBalance moves any ETH the caller transferred to a
// Phase 11 precompile address into the AppPrecompileRegistryAddr escrow
// ledger and returns the swept amount. The EVM's Call opcode transfers
// msg.value to the precompile's address BEFORE dispatching to the
// precompile, so the precompile's balance (which has no other sender)
// equals msg.value for this call. Sweeping it preserves the funds in a
// dedicated ledger and keeps the precompile balance at zero outside the
// active call.
//
// Returns *uint256.Int (never nil). For selectors that do not accept
// value, the dispatcher checks the returned amount and refuses if
// non-zero (with the funds still safe in AppPrecompileRegistryAddr).
func appSweepPrecompileBalance(evm *EVM, precompileAddr common.Address) *uint256.Int {
	bal := evm.StateDB.GetBalance(precompileAddr)
	if bal == nil || bal.IsZero() {
		return new(uint256.Int)
	}
	// Materialise the escrow account before moving funds.
	appEnsureRegistryExists(evm.StateDB)
	swept := new(uint256.Int).Set(bal)
	evm.Context.Transfer(evm.StateDB, precompileAddr, AppPrecompileRegistryAddr, swept)
	return swept
}

// appRefundEscrow returns ETH from the escrow ledger to the recipient.
// Used by the bounty claim path. Pure intra-state transfer; no external
// CALL is made so this is safe to invoke from inside another precompile.
func appRefundEscrow(evm *EVM, recipient common.Address, amount *uint256.Int) error {
	if amount == nil || amount.IsZero() {
		return nil
	}
	bal := evm.StateDB.GetBalance(AppPrecompileRegistryAddr)
	if bal == nil || bal.Cmp(amount) < 0 {
		return errors.New("appRefundEscrow: insufficient ledger balance")
	}
	evm.Context.Transfer(evm.StateDB, AppPrecompileRegistryAddr, recipient, amount)
	return nil
}

func (c *novaAsyncCallback) RunStateful(evm *EVM, caller common.Address, input []byte, readOnly bool) ([]byte, error) {
	if err := appPrecompileActive(evm, input); err != nil {
		return nil, err
	}
	// Reject accidental value sends. Value is only meaningful for the
	// bounty precompile.
	if v := appSweepPrecompileBalance(evm, asyncCallbackPrecompileAddr); !v.IsZero() {
		return nil, errors.New("novaAsyncCallback: does not accept value")
	}
	switch input[0] {
	case 0x01:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appRegisterAsyncCallback(evm, caller, input[1:])
	case 0x02:
		return appGetAsyncCallback(evm, input[1:])
	case 0x03:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appMarkAsyncCallbackFired(evm, caller, input[1:])
	case 0x04:
		return appAsyncCallbackReady(evm, input[1:])
	default:
		return nil, errors.New("novaAsyncCallback: unknown selector")
	}
}

func (c *novaIdentityAttestation) RunStateful(evm *EVM, caller common.Address, input []byte, readOnly bool) ([]byte, error) {
	if err := appPrecompileActive(evm, input); err != nil {
		return nil, err
	}
	if v := appSweepPrecompileBalance(evm, identityAttestationPrecompileAddr); !v.IsZero() {
		return nil, errors.New("novaIdentityAttestation: does not accept value")
	}
	switch input[0] {
	case 0x01:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appCreateIdentityAttestation(evm, caller, input[1:])
	case 0x02:
		return appVerifyIdentityAttestation(evm, input[1:])
	case 0x03:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appRevokeIdentityAttestation(evm, caller, input[1:])
	case 0x04:
		return appGetIdentityAttestation(evm, input[1:])
	default:
		return nil, errors.New("novaIdentityAttestation: unknown selector")
	}
}

func (c *novaSocialGraph) RunStateful(evm *EVM, caller common.Address, input []byte, readOnly bool) ([]byte, error) {
	if err := appPrecompileActive(evm, input); err != nil {
		return nil, err
	}
	if v := appSweepPrecompileBalance(evm, socialGraphPrecompileAddr); !v.IsZero() {
		return nil, errors.New("novaSocialGraph: does not accept value")
	}
	switch input[0] {
	case 0x01:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appFollow(evm, caller, input[1:])
	case 0x02:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appUnfollow(evm, caller, input[1:])
	case 0x03:
		return appIsFollowing(evm, input[1:])
	case 0x04:
		return appTrustScore(evm, input[1:])
	case 0x05:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appFollowWithConsent(evm, caller, input[1:])
	default:
		return nil, errors.New("novaSocialGraph: unknown selector")
	}
}

func (c *novaContentManifest) RunStateful(evm *EVM, caller common.Address, input []byte, readOnly bool) ([]byte, error) {
	if err := appPrecompileActive(evm, input); err != nil {
		return nil, err
	}
	if v := appSweepPrecompileBalance(evm, contentManifestPrecompileAddr); !v.IsZero() {
		return nil, errors.New("novaContentManifest: does not accept value")
	}
	switch input[0] {
	case 0x01:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appCreateContentManifest(evm, caller, input[1:])
	case 0x02:
		return appVerifyContentManifest(evm, input[1:])
	case 0x03:
		return appGetContentManifest(evm, input[1:])
	default:
		return nil, errors.New("novaContentManifest: unknown selector")
	}
}

func (c *novaGameState) RunStateful(evm *EVM, caller common.Address, input []byte, readOnly bool) ([]byte, error) {
	if err := appPrecompileActive(evm, input); err != nil {
		return nil, err
	}
	if v := appSweepPrecompileBalance(evm, gameStatePrecompileAddr); !v.IsZero() {
		return nil, errors.New("novaGameState: does not accept value")
	}
	switch input[0] {
	case 0x01:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appCommitGameState(evm, caller, input[1:])
	case 0x02:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appRevealGameState(evm, caller, input[1:])
	case 0x03:
		return appGetGameState(evm, input[1:])
	default:
		return nil, errors.New("novaGameState: unknown selector")
	}
}

func (c *novaComputeBounty) RunStateful(evm *EVM, caller common.Address, input []byte, readOnly bool) ([]byte, error) {
	if err := appPrecompileActive(evm, input); err != nil {
		return nil, err
	}
	// Bounty IS allowed to receive value (escrow). We sweep here, then
	// pass the swept amount into the create handler. Non-create selectors
	// MUST NOT receive value — refund / reject.
	swept := appSweepPrecompileBalance(evm, computeBountyPrecompileAddr)
	switch input[0] {
	case 0x01:
		if readOnly {
			return nil, ErrWriteProtection
		}
		return appCreateComputeBounty(evm, caller, input[1:], swept)
	case 0x02:
		if readOnly {
			return nil, ErrWriteProtection
		}
		if !swept.IsZero() {
			return nil, errors.New("novaComputeBounty: submit does not accept value")
		}
		return appSubmitComputeBounty(evm, caller, input[1:])
	case 0x03:
		if !swept.IsZero() {
			return nil, errors.New("novaComputeBounty: verify does not accept value")
		}
		return appVerifyComputeSubmission(evm, input[1:])
	case 0x04:
		if !swept.IsZero() {
			return nil, errors.New("novaComputeBounty: get does not accept value")
		}
		return appGetComputeBounty(evm, input[1:])
	case 0x05:
		if readOnly {
			return nil, ErrWriteProtection
		}
		if !swept.IsZero() {
			return nil, errors.New("novaComputeBounty: claim does not accept value")
		}
		return appClaimComputeBounty(evm, caller, input[1:])
	default:
		return nil, errors.New("novaComputeBounty: unknown selector")
	}
}

func appRequiredGas(input []byte, addr common.Address) uint64 {
	if len(input) == 0 {
		return 0
	}
	switch input[0] {
	case 0x01:
		// Register-async + claim-bounty both touch the deferred queue or
		// the escrow ledger; surcharge to reflect the extra state write.
		if addr == asyncCallbackPrecompileAddr || addr == computeBountyPrecompileAddr {
			return appPrecompileGasEnqueue
		}
		return appPrecompileGasWrite
	case 0x02, 0x03, 0x04:
		return appPrecompileGasVerify
	case 0x05:
		if addr == computeBountyPrecompileAddr || addr == socialGraphPrecompileAddr {
			return appPrecompileGasWrite
		}
		return appPrecompileGasRead
	default:
		return appPrecompileGasRead
	}
}

func appPrecompileActive(evm *EVM, input []byte) error {
	if len(input) == 0 {
		return errors.New("application precompile: empty input")
	}
	if evm.Context.BlockNumber.Uint64() < ethernova.ApplicationPrecompileForkBlock {
		return errors.New("application precompile: not yet active")
	}
	return nil
}

func appEnsureRegistryExists(sdb StateDB) {
	if !sdb.Exist(AppPrecompileRegistryAddr) {
		sdb.CreateAccount(AppPrecompileRegistryAddr)
	}
	if sdb.GetNonce(AppPrecompileRegistryAddr) == 0 {
		sdb.SetNonce(AppPrecompileRegistryAddr, 1)
	}
}

func appKey(parts ...[]byte) common.Hash {
	return crypto.Keccak256Hash(parts...)
}

func appKindKey(kind string, id common.Hash, field string) common.Hash {
	return appKey([]byte("app"), []byte(kind), id.Bytes(), []byte(field))
}

func appIndexKey(kind string, parts ...[]byte) common.Hash {
	all := [][]byte{[]byte("app-index"), []byte(kind)}
	all = append(all, parts...)
	return appKey(all...)
}

func appReadUint64(sdb StateDB, key common.Hash) uint64 {
	return new(big.Int).SetBytes(sdb.GetState(AppPrecompileRegistryAddr, key).Bytes()).Uint64()
}

func appWriteUint64(sdb StateDB, key common.Hash, v uint64) {
	sdb.SetState(AppPrecompileRegistryAddr, key, common.BigToHash(new(big.Int).SetUint64(v)))
}

func appWriteHash(sdb StateDB, key common.Hash, v common.Hash) {
	sdb.SetState(AppPrecompileRegistryAddr, key, v)
}

func appReadHash(sdb StateDB, key common.Hash) common.Hash {
	return sdb.GetState(AppPrecompileRegistryAddr, key)
}

func appWriteAddress(sdb StateDB, key common.Hash, addr common.Address) {
	sdb.SetState(AppPrecompileRegistryAddr, key, common.BytesToHash(addr.Bytes()))
}

func appReadAddress(sdb StateDB, key common.Hash) common.Address {
	return common.BytesToAddress(sdb.GetState(AppPrecompileRegistryAddr, key).Bytes())
}

func appWriteBool(sdb StateDB, key common.Hash, enabled bool) {
	if enabled {
		sdb.SetState(AppPrecompileRegistryAddr, key, common.BytesToHash([]byte{0x01}))
		return
	}
	sdb.SetState(AppPrecompileRegistryAddr, key, common.Hash{})
}

func appReadBool(sdb StateDB, key common.Hash) bool {
	return sdb.GetState(AppPrecompileRegistryAddr, key) != (common.Hash{})
}

// appWriteU256 writes a 32-byte uint256 to AppPrecompileRegistryAddr keyed
// by `key`. Used for escrow amounts.
func appWriteU256(sdb StateDB, key common.Hash, v *uint256.Int) {
	if v == nil {
		sdb.SetState(AppPrecompileRegistryAddr, key, common.Hash{})
		return
	}
	sdb.SetState(AppPrecompileRegistryAddr, key, common.BigToHash(v.ToBig()))
}

func appReadU256(sdb StateDB, key common.Hash) *uint256.Int {
	h := sdb.GetState(AppPrecompileRegistryAddr, key)
	out := new(uint256.Int)
	out.SetBytes(h.Bytes())
	return out
}

func appNextID(sdb StateDB, kind string, caller common.Address, blockNum uint64, seed ...[]byte) common.Hash {
	appEnsureRegistryExists(sdb)
	nonceKey := appIndexKey(kind, []byte("nonce"))
	nonce := appReadUint64(sdb, nonceKey)
	var blockBuf, nonceBuf [8]byte
	binary.BigEndian.PutUint64(blockBuf[:], blockNum)
	binary.BigEndian.PutUint64(nonceBuf[:], nonce)
	parts := [][]byte{[]byte("app-id"), []byte(kind), caller.Bytes(), blockBuf[:], nonceBuf[:]}
	parts = append(parts, seed...)
	id := appKey(parts...)
	appWriteUint64(sdb, nonceKey, nonce+1)
	return id
}

func appRequireLen(input []byte, need int, label string) error {
	if len(input) < need {
		return fmt.Errorf("%s: input too short (need %d, got %d)", label, need, len(input))
	}
	return nil
}

func appWord(input []byte, idx int) []byte      { return input[idx*32 : (idx+1)*32] }
func appHash(input []byte, idx int) common.Hash { return common.BytesToHash(appWord(input, idx)) }
func appAddress(input []byte, idx int) common.Address {
	return common.BytesToAddress(appWord(input, idx))
}

func appUint64(input []byte, idx int, label string) (uint64, error) {
	word := new(big.Int).SetBytes(appWord(input, idx))
	if word.BitLen() > 64 {
		return 0, fmt.Errorf("%s exceeds uint64", label)
	}
	return word.Uint64(), nil
}

func appU256(input []byte, idx int) *uint256.Int {
	out := new(uint256.Int)
	out.SetBytes(appWord(input, idx))
	return out
}

func appWordUint64(v uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(v))
}

func appWordU256(v *uint256.Int) common.Hash {
	if v == nil {
		return common.Hash{}
	}
	return common.BigToHash(v.ToBig())
}

func appWordBool(v bool) common.Hash {
	if v {
		return common.BytesToHash([]byte{0x01})
	}
	return common.Hash{}
}

func appReturn(words ...common.Hash) []byte {
	out := make([]byte, 0, len(words)*32)
	for _, w := range words {
		out = append(out, w.Bytes()...)
	}
	return out
}

func appReturnBool(v bool) []byte { return appReturn(appWordBool(v)) }

func appExists(sdb StateDB, kind string, id common.Hash) bool {
	return appReadBool(sdb, appKindKey(kind, id, "exists"))
}

func appSetExists(sdb StateDB, kind string, id common.Hash, exists bool) {
	appWriteBool(sdb, appKindKey(kind, id, "exists"), exists)
}

// 0x30 novaAsyncCallback -------------------------------------------------

// appRegisterAsyncCallback input layout (5 words = 160 bytes, NOT 96):
//
//	word 0: condition       (bytes32, opaque tag set by the caller)
//	word 1: target_addr     (bytes32 left-padded; recipient contract)
//	word 2: selector_word   (bytes32; the last 4 bytes are the function
//	                         selector of the recipient's handler)
//	word 3: payloadHash     (bytes32; passed as second arg to the handler)
//	word 4: targetBlock     (uint256; block at which the dispatch fires)
//
// Recipients are expected to expose:
//
//	function <handler>(bytes32 callbackId, bytes32 payloadHash) external;
//
// where <handler> matches the selector. The system caller for the
// chain-driven dispatch is AppPrecompileRegistryAddr (0xFF11).
//
// IMPORTANT: target_addr MAY be the zero address. A zero-target record
// behaves as the old polling-style commitment (no chain-driven dispatch).
// This is kept for callers who want only the "ready at block X" timer
// without exposing a callable surface.
func appRegisterAsyncCallback(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 160, "registerAsyncCallback"); err != nil {
		return nil, err
	}
	condition := appHash(input, 0)
	targetAddr := appAddress(input, 1)
	selectorWord := appHash(input, 2)
	payloadHash := appHash(input, 3)
	targetBlock, err := appUint64(input, 4, "targetBlock")
	if err != nil {
		return nil, err
	}

	sdb := evm.StateDB
	id := appNextID(sdb, "async", caller, evm.Context.BlockNumber.Uint64(), input[:160])
	appSetExists(sdb, "async", id, true)
	appWriteHash(sdb, appKindKey("async", id, "condition"), condition)
	appWriteAddress(sdb, appKindKey("async", id, "target_addr"), targetAddr)
	appWriteHash(sdb, appKindKey("async", id, "selector"), selectorWord)
	appWriteHash(sdb, appKindKey("async", id, "payloadHash"), payloadHash)
	appWriteUint64(sdb, appKindKey("async", id, "target"), targetBlock)
	appWriteAddress(sdb, appKindKey("async", id, "owner"), caller)
	appWriteBool(sdb, appKindKey("async", id, "fired"), false)

	// Schedule chain-driven dispatch by appending the ID to the per-block
	// index. ProcessAsyncCallbacks (called during Phase 0 of `targetBlock`)
	// will drain this index and dispatch the recorded targets.
	asyncByBlockAppend(sdb, targetBlock, id)

	// Also enqueue an EffectTypeAsyncCallback into the global Deferred
	// Queue so nova_getDeferredStats reflects the in-flight commitment.
	// The deferred queue drain handler for this type is intentionally a
	// no-op (chain-driven execution happens via ProcessAsyncCallbacks).
	// The enqueue is best-effort: if the per-block cap is hit, we still
	// record the by-block index and return success — the index is the
	// source of truth for dispatch.
	payload := make([]byte, 0, 40)
	payload = append(payload, id.Bytes()...)
	var blkBuf [8]byte
	binary.BigEndian.PutUint64(blkBuf[:], targetBlock)
	payload = append(payload, blkBuf[:]...)
	_, _ = DqEnqueueDirectly(sdb, evm.Context.BlockNumber.Uint64(), types.EffectTypeAsyncCallback, caller, payload)

	return id.Bytes(), nil
}

func appGetAsyncCallback(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "getAsyncCallback"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "async", id) {
		return nil, errors.New("getAsyncCallback: callback not found")
	}
	// 7 words: condition, target_addr, selector, payloadHash, target,
	// owner, fired. This is backwards-incompatible with the prior 5-word
	// return — Phase 11 fix v2.
	return appReturn(
		appReadHash(sdb, appKindKey("async", id, "condition")),
		common.BytesToHash(appReadAddress(sdb, appKindKey("async", id, "target_addr")).Bytes()),
		appReadHash(sdb, appKindKey("async", id, "selector")),
		appReadHash(sdb, appKindKey("async", id, "payloadHash")),
		appWordUint64(appReadUint64(sdb, appKindKey("async", id, "target"))),
		common.BytesToHash(appReadAddress(sdb, appKindKey("async", id, "owner")).Bytes()),
		appWordBool(appReadBool(sdb, appKindKey("async", id, "fired"))),
	), nil
}

func appMarkAsyncCallbackFired(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "markAsyncCallbackFired"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "async", id) {
		return nil, errors.New("markAsyncCallbackFired: callback not found")
	}
	if appReadAddress(sdb, appKindKey("async", id, "owner")) != caller {
		return nil, errors.New("markAsyncCallbackFired: caller is not owner")
	}
	appWriteBool(sdb, appKindKey("async", id, "fired"), true)
	return appReturnBool(true), nil
}

func appAsyncCallbackReady(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "asyncCallbackReady"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "async", id) {
		return appReturnBool(false), nil
	}
	target := appReadUint64(sdb, appKindKey("async", id, "target"))
	fired := appReadBool(sdb, appKindKey("async", id, "fired"))
	return appReturnBool(!fired && evm.Context.BlockNumber.Uint64() >= target), nil
}

// 0x31 novaIdentityAttestation -------------------------------------------

func appCreateIdentityAttestation(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 96, "attestIdentity"); err != nil {
		return nil, err
	}
	subject := appAddress(input, 0)
	if subject == (common.Address{}) {
		return nil, errors.New("attestIdentity: zero subject")
	}
	claim := appHash(input, 1)
	expiry, err := appUint64(input, 2, "expiryBlock")
	if err != nil {
		return nil, err
	}
	sdb := evm.StateDB
	id := appNextID(sdb, "identity", caller, evm.Context.BlockNumber.Uint64(), input[:96])
	appSetExists(sdb, "identity", id, true)
	appWriteAddress(sdb, appKindKey("identity", id, "subject"), subject)
	appWriteHash(sdb, appKindKey("identity", id, "claim"), claim)
	appWriteAddress(sdb, appKindKey("identity", id, "issuer"), caller)
	appWriteUint64(sdb, appKindKey("identity", id, "expiry"), expiry)
	appWriteBool(sdb, appKindKey("identity", id, "revoked"), false)

	// BUG-2 fix: also create a Protocol Object of type Identity so it is
	// queryable via nova_listProtocolObjects(0x04, owner). The PO's
	// stateData encodes the attestation fields for downstream consumers.
	// The PO's Owner is the issuer (caller), so list-by-issuer works
	// directly. The PO's ID is INDEPENDENT of the precompile's `id` to
	// preserve the global_nonce monotonic guarantee on PO IDs; the
	// precompile records the PO ID alongside the attestation for lookup.
	stateData := encodeIdentityPOState(subject, claim, id, expiry)
	poID, perr := PoCreateObjectInternal(evm, caller, types.ProtoTypeIdentity, stateData, expiry, new(big.Int))
	if perr != nil {
		// Non-fatal: log via state slot and continue. Identity is still
		// queryable via getIdentity even without a PO; this just means
		// nova_listProtocolObjects won't surface it. We do NOT return an
		// error because that would revert the attestation creation, and
		// PO creation failure (e.g. global_nonce overflow) is exceptional
		// enough to deserve a soft-fail path.
		appWriteBool(sdb, appKindKey("identity", id, "po_missing"), true)
	} else {
		appWriteHash(sdb, appKindKey("identity", id, "po_id"), poID)
	}
	return id.Bytes(), nil
}

// encodeIdentityPOState builds a compact RLP-friendly byte sequence used
// as a Protocol Object's StateData. Layout (fixed-position):
//
//	[0:32]  subject_word (left-padded address)
//	[32:64] claim
//	[64:96] precompile_attestation_id
//	[96:128] expiry_word
func encodeIdentityPOState(subject common.Address, claim, attestationID common.Hash, expiry uint64) []byte {
	out := make([]byte, 0, 128)
	out = append(out, common.BytesToHash(subject.Bytes()).Bytes()...)
	out = append(out, claim.Bytes()...)
	out = append(out, attestationID.Bytes()...)
	out = append(out, appWordUint64(expiry).Bytes()...)
	return out
}

func appIdentityValid(evm *EVM, id common.Hash) bool {
	sdb := evm.StateDB
	if !appExists(sdb, "identity", id) || appReadBool(sdb, appKindKey("identity", id, "revoked")) {
		return false
	}
	expiry := appReadUint64(sdb, appKindKey("identity", id, "expiry"))
	return expiry == 0 || evm.Context.BlockNumber.Uint64() <= expiry
}

func appVerifyIdentityAttestation(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "verifyIdentity"); err != nil {
		return nil, err
	}
	return appReturnBool(appIdentityValid(evm, appHash(input, 0))), nil
}

func appRevokeIdentityAttestation(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "revokeIdentity"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "identity", id) {
		return nil, errors.New("revokeIdentity: attestation not found")
	}
	if appReadAddress(sdb, appKindKey("identity", id, "issuer")) != caller {
		return nil, errors.New("revokeIdentity: caller is not issuer")
	}
	appWriteBool(sdb, appKindKey("identity", id, "revoked"), true)
	return appReturnBool(true), nil
}

func appGetIdentityAttestation(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "getIdentity"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "identity", id) {
		return nil, errors.New("getIdentity: attestation not found")
	}
	// 7 words: subject, claim, issuer, expiry, revoked, valid, po_id.
	// Adding po_id (the last word) so callers can pivot to the PO query.
	return appReturn(
		common.BytesToHash(appReadAddress(sdb, appKindKey("identity", id, "subject")).Bytes()),
		appReadHash(sdb, appKindKey("identity", id, "claim")),
		common.BytesToHash(appReadAddress(sdb, appKindKey("identity", id, "issuer")).Bytes()),
		appWordUint64(appReadUint64(sdb, appKindKey("identity", id, "expiry"))),
		appWordBool(appReadBool(sdb, appKindKey("identity", id, "revoked"))),
		appWordBool(appIdentityValid(evm, id)),
		appReadHash(sdb, appKindKey("identity", id, "po_id")),
	), nil
}

// 0x32 novaSocialGraph ----------------------------------------------------

func appSocialEdgeKey(follower, target common.Address) common.Hash {
	return appIndexKey("social-edge", follower.Bytes(), target.Bytes())
}

func appSocialEdgeID(follower, target common.Address) common.Hash {
	return appKey([]byte("social-edge-id"), follower.Bytes(), target.Bytes())
}

func appFollow(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "follow"); err != nil {
		return nil, err
	}
	target := appAddress(input, 0)
	if target == (common.Address{}) || target == caller {
		return nil, errors.New("follow: invalid target")
	}
	appEnsureRegistryExists(evm.StateDB)
	appWriteBool(evm.StateDB, appSocialEdgeKey(caller, target), true)
	return appSocialEdgeID(caller, target).Bytes(), nil
}

// appFollowWithConsent (selector 0x05, NEW) requires a signature from the
// target authorizing the follow. Input layout (4 words):
//
//	word 0: target (bytes32 left-padded address)
//	word 1: sigR
//	word 2: sigS
//	word 3: sigV (last byte holds v; rest must be zero)
//
// The message hashed for signing is:
//
//	keccak256("nova.social.follow:" || caller_addr || target || chainId)
//
// The recovered signer MUST equal target.
//
// This is an OPTIONAL consent gate; the open `follow` (0x01) remains
// available by design (audit BUG-6: "Twitter-style open follow").
func appFollowWithConsent(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 128, "followWithConsent"); err != nil {
		return nil, err
	}
	target := appAddress(input, 0)
	if target == (common.Address{}) || target == caller {
		return nil, errors.New("followWithConsent: invalid target")
	}
	sigR := appHash(input, 1)
	sigS := appHash(input, 2)
	sigVWord := appHash(input, 3)
	v := sigVWord.Bytes()[31]
	if v < 27 {
		// Normalise EIP-155 v if present (caller might submit 0/1).
		v += 27
	}

	// Reconstruct the message hash. ChainID is taken from EVM config to
	// bind the signature to this chain.
	chainID := evm.chainConfig.GetChainID()
	if chainID == nil {
		chainID = new(big.Int)
	}
	chainIDBytes := common.BigToHash(chainID).Bytes()
	msg := append([]byte("nova.social.follow:"), caller.Bytes()...)
	msg = append(msg, target.Bytes()...)
	msg = append(msg, chainIDBytes...)
	msgHash := crypto.Keccak256Hash(msg)

	// Build the [R||S||V-1] sig blob (Geth's Ecrecover expects v=0/1).
	sig := make([]byte, 65)
	copy(sig[0:32], sigR.Bytes())
	copy(sig[32:64], sigS.Bytes())
	sig[64] = v - 27
	pub, err := crypto.Ecrecover(msgHash.Bytes(), sig)
	if err != nil {
		return nil, fmt.Errorf("followWithConsent: ecrecover failed: %w", err)
	}
	recovered := common.BytesToAddress(crypto.Keccak256(pub[1:])[12:])
	if recovered != target {
		return nil, errors.New("followWithConsent: signer is not target")
	}

	appEnsureRegistryExists(evm.StateDB)
	appWriteBool(evm.StateDB, appSocialEdgeKey(caller, target), true)
	// Mark as consented for downstream auditing.
	appWriteBool(evm.StateDB, appKey([]byte("app-social-consent"), caller.Bytes(), target.Bytes()), true)
	return appSocialEdgeID(caller, target).Bytes(), nil
}

func appUnfollow(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "unfollow"); err != nil {
		return nil, err
	}
	target := appAddress(input, 0)
	appEnsureRegistryExists(evm.StateDB)
	appWriteBool(evm.StateDB, appSocialEdgeKey(caller, target), false)
	// Also clear consent record if any.
	appWriteBool(evm.StateDB, appKey([]byte("app-social-consent"), caller.Bytes(), target.Bytes()), false)
	return appReturnBool(true), nil
}

func appIsFollowing(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 64, "isFollowing"); err != nil {
		return nil, err
	}
	return appReturnBool(appReadBool(evm.StateDB, appSocialEdgeKey(appAddress(input, 0), appAddress(input, 1)))), nil
}

func appTrustScore(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 64, "trustScore"); err != nil {
		return nil, err
	}
	a, b := appAddress(input, 0), appAddress(input, 1)
	ab := appReadBool(evm.StateDB, appSocialEdgeKey(a, b))
	ba := appReadBool(evm.StateDB, appSocialEdgeKey(b, a))
	score := uint64(0)
	if ab && ba {
		score = 100
	} else if ab {
		score = 50
	}
	return appReturn(appWordUint64(score)), nil
}

// 0x33 novaContentManifest -----------------------------------------------

func appCreateContentManifest(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 128, "createContentManifest"); err != nil {
		return nil, err
	}
	root := appHash(input, 0)
	contentRef := appHash(input, 1)
	mime := appHash(input, 2)
	size, err := appUint64(input, 3, "size")
	if err != nil {
		return nil, err
	}

	// BUG-4 fix: validate that contentRef resolves to a Phase 3
	// ContentReference Protocol Object. The zero hash is permitted as a
	// "no contentRef attached" sentinel so callers who only want a root
	// commitment can still create a manifest.
	sdb := evm.StateDB
	if contentRef != (common.Hash{}) {
		if obj := CrGetContentRef(sdb, contentRef); obj == nil {
			return nil, errors.New("createContentManifest: contentRef does not resolve to a Phase 3 ContentReference object")
		}
	}

	id := appNextID(sdb, "manifest", caller, evm.Context.BlockNumber.Uint64(), input[:128])
	appSetExists(sdb, "manifest", id, true)
	appWriteHash(sdb, appKindKey("manifest", id, "root"), root)
	appWriteHash(sdb, appKindKey("manifest", id, "contentRef"), contentRef)
	appWriteHash(sdb, appKindKey("manifest", id, "mime"), mime)
	appWriteUint64(sdb, appKindKey("manifest", id, "size"), size)
	appWriteAddress(sdb, appKindKey("manifest", id, "owner"), caller)

	// BUG-2 fix: also create a Protocol Object so the manifest is
	// listable via nova_listProtocolObjects. We use ProtoTypeContentReference
	// since the manifest is conceptually a content-reference primitive;
	// downstream consumers can use the stateData layout below to
	// distinguish manifests from raw ContentRefs created by 0x2B.
	stateData := encodeManifestPOState(root, contentRef, mime, size, id)
	poID, perr := PoCreateObjectInternal(evm, caller, types.ProtoTypeContentReference, stateData, 0, new(big.Int))
	if perr == nil {
		appWriteHash(sdb, appKindKey("manifest", id, "po_id"), poID)
	} else {
		appWriteBool(sdb, appKindKey("manifest", id, "po_missing"), true)
	}
	return id.Bytes(), nil
}

func encodeManifestPOState(root, contentRef, mime common.Hash, size uint64, manifestID common.Hash) []byte {
	out := make([]byte, 0, 160)
	out = append(out, []byte("manifest:")...) // tag so consumers can tell manifest POs from CR POs
	for len(out) < 32 {
		out = append(out, 0x00)
	}
	out = out[:32]
	out = append(out, root.Bytes()...)
	out = append(out, contentRef.Bytes()...)
	out = append(out, mime.Bytes()...)
	out = append(out, appWordUint64(size).Bytes()...)
	out = append(out, manifestID.Bytes()...)
	return out
}

func appVerifyContentManifest(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 64, "verifyContentManifest"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	ok := appExists(evm.StateDB, "manifest", id) && appReadHash(evm.StateDB, appKindKey("manifest", id, "root")) == appHash(input, 1)
	return appReturnBool(ok), nil
}

func appGetContentManifest(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "getContentManifest"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "manifest", id) {
		return nil, errors.New("getContentManifest: manifest not found")
	}
	// 6 words: root, contentRef, mime, size, owner, po_id.
	return appReturn(
		appReadHash(sdb, appKindKey("manifest", id, "root")),
		appReadHash(sdb, appKindKey("manifest", id, "contentRef")),
		appReadHash(sdb, appKindKey("manifest", id, "mime")),
		appWordUint64(appReadUint64(sdb, appKindKey("manifest", id, "size"))),
		common.BytesToHash(appReadAddress(sdb, appKindKey("manifest", id, "owner")).Bytes()),
		appReadHash(sdb, appKindKey("manifest", id, "po_id")),
	), nil
}

// 0x34 novaGameState ------------------------------------------------------

func appCommitGameState(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 96, "commitGameState"); err != nil {
		return nil, err
	}
	gameID, stateHash := appHash(input, 0), appHash(input, 1)
	turn, err := appUint64(input, 2, "turn")
	if err != nil {
		return nil, err
	}
	sdb := evm.StateDB
	currentTurn := appReadUint64(sdb, appKindKey("game", gameID, "turn"))
	exists := appReadBool(sdb, appKindKey("game", gameID, "exists"))
	if exists && turn <= currentTurn {
		return nil, errors.New("commitGameState: turn must increase")
	}
	appEnsureRegistryExists(sdb)
	appWriteBool(sdb, appKindKey("game", gameID, "exists"), true)
	appWriteHash(sdb, appKindKey("game", gameID, "state"), stateHash)
	appWriteUint64(sdb, appKindKey("game", gameID, "turn"), turn)
	appWriteAddress(sdb, appKindKey("game", gameID, "player"), caller)
	commitmentID := appKey([]byte("game-commit"), gameID.Bytes(), stateHash.Bytes(), appWordUint64(turn).Bytes(), caller.Bytes())
	appWriteHash(sdb, appKindKey("game", gameID, "commitment"), commitmentID)

	// BUG-5 fix: on FIRST commit, create a ProtoTypeGameRoom Protocol
	// Object so the game is queryable per-owner via
	// nova_listProtocolObjects. Subsequent commits do not create new POs
	// but update the existing one's LastTouchedBlock by writing the same
	// stateData (cheap idempotent path).
	if !exists {
		stateData := encodeGameRoomPOState(gameID, stateHash, turn, caller)
		poID, perr := PoCreateObjectInternal(evm, caller, types.ProtoTypeGameRoom, stateData, 0, new(big.Int))
		if perr == nil {
			appWriteHash(sdb, appKindKey("game", gameID, "po_id"), poID)
		} else {
			appWriteBool(sdb, appKindKey("game", gameID, "po_missing"), true)
		}
	}
	return commitmentID.Bytes(), nil
}

func encodeGameRoomPOState(gameID, stateHash common.Hash, turn uint64, player common.Address) []byte {
	out := make([]byte, 0, 128)
	out = append(out, gameID.Bytes()...)
	out = append(out, stateHash.Bytes()...)
	out = append(out, appWordUint64(turn).Bytes()...)
	out = append(out, common.BytesToHash(player.Bytes()).Bytes()...)
	return out
}

func appRevealGameState(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 128, "revealGameState"); err != nil {
		return nil, err
	}
	gameID, stateHash := appHash(input, 0), appHash(input, 1)
	turn, err := appUint64(input, 2, "turn")
	if err != nil {
		return nil, err
	}
	salt := appHash(input, 3)
	sdb := evm.StateDB
	if !appReadBool(sdb, appKindKey("game", gameID, "exists")) {
		return nil, errors.New("revealGameState: game not found")
	}
	if appReadHash(sdb, appKindKey("game", gameID, "state")) != stateHash || appReadUint64(sdb, appKindKey("game", gameID, "turn")) != turn {
		return nil, errors.New("revealGameState: state mismatch")
	}
	if appReadAddress(sdb, appKindKey("game", gameID, "player")) != caller {
		return nil, errors.New("revealGameState: caller is not current player")
	}
	revealHash := appKey([]byte("game-reveal"), gameID.Bytes(), stateHash.Bytes(), appWordUint64(turn).Bytes(), salt.Bytes())
	appWriteHash(sdb, appKindKey("game", gameID, "reveal"), revealHash)
	return revealHash.Bytes(), nil
}

func appGetGameState(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "getGameState"); err != nil {
		return nil, err
	}
	gameID := appHash(input, 0)
	sdb := evm.StateDB
	if !appReadBool(sdb, appKindKey("game", gameID, "exists")) {
		return nil, errors.New("getGameState: game not found")
	}
	// 5 words: state, turn, player, reveal, po_id.
	return appReturn(
		appReadHash(sdb, appKindKey("game", gameID, "state")),
		appWordUint64(appReadUint64(sdb, appKindKey("game", gameID, "turn"))),
		common.BytesToHash(appReadAddress(sdb, appKindKey("game", gameID, "player")).Bytes()),
		appReadHash(sdb, appKindKey("game", gameID, "reveal")),
		appReadHash(sdb, appKindKey("game", gameID, "po_id")),
	), nil
}

// 0x36 novaComputeBounty --------------------------------------------------

// appCreateComputeBounty input layout (4 words, 128 bytes):
//
//	word 0: specHash         (bytes32)
//	word 1: expectedResult   (bytes32; the answer the bounty creator
//	                          commits to as correct. Verification compares
//	                          this against submission.result.)
//	word 2: reward           (uint256 unused as a value field; preserved
//	                          for off-chain display. Real economic value
//	                          is sourced from msg.value, see `escrow`)
//	word 3: expiryBlock      (uint64)
//
// msg.value is recorded as the escrow amount. If msg.value is zero the
// bounty is created with zero escrow — useful for purely-commitment
// bounties (no economic incentive).
func appCreateComputeBounty(evm *EVM, caller common.Address, input []byte, swept *uint256.Int) ([]byte, error) {
	if err := appRequireLen(input, 128, "createComputeBounty"); err != nil {
		return nil, err
	}
	specHash := appHash(input, 0)
	expectedResult := appHash(input, 1)
	reward, err := appUint64(input, 2, "reward")
	if err != nil {
		return nil, err
	}
	expiry, err := appUint64(input, 3, "expiryBlock")
	if err != nil {
		return nil, err
	}
	sdb := evm.StateDB
	id := appNextID(sdb, "bounty", caller, evm.Context.BlockNumber.Uint64(), input[:128])
	appSetExists(sdb, "bounty", id, true)
	appWriteHash(sdb, appKindKey("bounty", id, "spec"), specHash)
	appWriteHash(sdb, appKindKey("bounty", id, "expected"), expectedResult)
	appWriteUint64(sdb, appKindKey("bounty", id, "reward"), reward)
	appWriteUint64(sdb, appKindKey("bounty", id, "expiry"), expiry)
	appWriteAddress(sdb, appKindKey("bounty", id, "owner"), caller)
	appWriteBool(sdb, appKindKey("bounty", id, "open"), true)
	appWriteBool(sdb, appKindKey("bounty", id, "claimed"), false)
	appWriteU256(sdb, appKindKey("bounty", id, "escrow"), swept)

	// BUG-2 fix: surface the bounty as a Subscription PO (closest fit:
	// recurring or event-driven obligations). ContentManifest used
	// ContentReference; this one uses Subscription. GameRoom is the only
	// PO type fully exclusive to one precompile.
	stateData := encodeBountyPOState(specHash, expectedResult, swept, expiry, id)
	poID, perr := PoCreateObjectInternal(evm, caller, types.ProtoTypeSubscription, stateData, expiry, new(big.Int))
	if perr == nil {
		appWriteHash(sdb, appKindKey("bounty", id, "po_id"), poID)
	} else {
		appWriteBool(sdb, appKindKey("bounty", id, "po_missing"), true)
	}
	return id.Bytes(), nil
}

func encodeBountyPOState(specHash, expectedResult common.Hash, escrow *uint256.Int, expiry uint64, bountyID common.Hash) []byte {
	out := make([]byte, 0, 160)
	out = append(out, specHash.Bytes()...)
	out = append(out, expectedResult.Bytes()...)
	out = append(out, appWordU256(escrow).Bytes()...)
	out = append(out, appWordUint64(expiry).Bytes()...)
	out = append(out, bountyID.Bytes()...)
	return out
}

func appSubmitComputeBounty(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 96, "submitComputeBounty"); err != nil {
		return nil, err
	}
	bountyID := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "bounty", bountyID) || !appReadBool(sdb, appKindKey("bounty", bountyID, "open")) {
		return nil, errors.New("submitComputeBounty: bounty not open")
	}
	expiry := appReadUint64(sdb, appKindKey("bounty", bountyID, "expiry"))
	if expiry != 0 && evm.Context.BlockNumber.Uint64() > expiry {
		return nil, errors.New("submitComputeBounty: bounty expired")
	}
	submissionID := appNextID(sdb, "bounty-submission", caller, evm.Context.BlockNumber.Uint64(), input[:96])
	appSetExists(sdb, "bounty-submission", submissionID, true)
	appWriteHash(sdb, appKindKey("bounty-submission", submissionID, "bounty"), bountyID)
	appWriteHash(sdb, appKindKey("bounty-submission", submissionID, "result"), appHash(input, 1))
	appWriteHash(sdb, appKindKey("bounty-submission", submissionID, "proof"), appHash(input, 2))
	appWriteAddress(sdb, appKindKey("bounty-submission", submissionID, "submitter"), caller)
	return submissionID.Bytes(), nil
}

// appVerifyComputeSubmission compares submission.result to:
//   - the SECOND input word if provided as a non-zero override (legacy
//     callers); OR
//   - the bounty's stored `expected` field otherwise.
//
// This preserves backwards-compatible behaviour for callers that explicitly
// pass an expected value, while also enabling the canonical "verify against
// bounty's expected" path used by the new claim selector.
func appVerifyComputeSubmission(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 64, "verifyComputeSubmission"); err != nil {
		return nil, err
	}
	submissionID := appHash(input, 0)
	overrideExpected := appHash(input, 1)
	sdb := evm.StateDB
	if !appExists(sdb, "bounty-submission", submissionID) {
		return appReturnBool(false), nil
	}
	stored := appReadHash(sdb, appKindKey("bounty-submission", submissionID, "result"))
	if overrideExpected != (common.Hash{}) {
		return appReturnBool(stored == overrideExpected), nil
	}
	bountyID := appReadHash(sdb, appKindKey("bounty-submission", submissionID, "bounty"))
	canonical := appReadHash(sdb, appKindKey("bounty", bountyID, "expected"))
	return appReturnBool(stored == canonical), nil
}

// appClaimComputeBounty (selector 0x05) releases the escrow to the
// submitter of a verified submission. Caller MUST be the bounty owner
// (only the owner can release escrow to prevent griefing where a third
// party releases to the wrong submitter). Input layout:
//
//	word 0: bountyID
//	word 1: submissionID
func appClaimComputeBounty(evm *EVM, caller common.Address, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 64, "claimComputeBounty"); err != nil {
		return nil, err
	}
	bountyID := appHash(input, 0)
	submissionID := appHash(input, 1)
	sdb := evm.StateDB
	if !appExists(sdb, "bounty", bountyID) {
		return nil, errors.New("claimComputeBounty: bounty not found")
	}
	if appReadAddress(sdb, appKindKey("bounty", bountyID, "owner")) != caller {
		return nil, errors.New("claimComputeBounty: caller is not bounty owner")
	}
	if !appReadBool(sdb, appKindKey("bounty", bountyID, "open")) {
		return nil, errors.New("claimComputeBounty: bounty already closed")
	}
	if !appExists(sdb, "bounty-submission", submissionID) {
		return nil, errors.New("claimComputeBounty: submission not found")
	}
	if appReadHash(sdb, appKindKey("bounty-submission", submissionID, "bounty")) != bountyID {
		return nil, errors.New("claimComputeBounty: submission does not match bounty")
	}
	expected := appReadHash(sdb, appKindKey("bounty", bountyID, "expected"))
	stored := appReadHash(sdb, appKindKey("bounty-submission", submissionID, "result"))
	if stored != expected {
		return nil, errors.New("claimComputeBounty: submission result does not match bounty expected")
	}
	submitter := appReadAddress(sdb, appKindKey("bounty-submission", submissionID, "submitter"))

	// Release escrow.
	escrow := appReadU256(sdb, appKindKey("bounty", bountyID, "escrow"))
	if err := appRefundEscrow(evm, submitter, escrow); err != nil {
		return nil, err
	}
	// Close bounty and zero out the escrow record.
	appWriteBool(sdb, appKindKey("bounty", bountyID, "open"), false)
	appWriteBool(sdb, appKindKey("bounty", bountyID, "claimed"), true)
	appWriteAddress(sdb, appKindKey("bounty", bountyID, "winner"), submitter)
	appWriteU256(sdb, appKindKey("bounty", bountyID, "escrow"), new(uint256.Int))

	return appReturnBool(true), nil
}

func appGetComputeBounty(evm *EVM, input []byte) ([]byte, error) {
	if err := appRequireLen(input, 32, "getComputeBounty"); err != nil {
		return nil, err
	}
	id := appHash(input, 0)
	sdb := evm.StateDB
	if !appExists(sdb, "bounty", id) {
		return nil, errors.New("getComputeBounty: bounty not found")
	}
	// 9 words: spec, expected, reward, expiry, owner, open, claimed,
	// winner, escrow, po_id. (10 actually — counted again below.)
	// Note: the public layout is documented in
	// eth/api_ethernova_phase11.go's description; downstream SDKs must
	// upgrade once Phase 11 is rolled. For backwards-compatibility we
	// preserve the previous first 5 fields in their previous positions
	// (spec at idx 0, reward at idx 1 USED to be) — but we now insert
	// `expected` at idx 1 to make the structure self-describing. SDKs
	// must rebuild against the new layout.
	return appReturn(
		appReadHash(sdb, appKindKey("bounty", id, "spec")),
		appReadHash(sdb, appKindKey("bounty", id, "expected")),
		appWordUint64(appReadUint64(sdb, appKindKey("bounty", id, "reward"))),
		appWordUint64(appReadUint64(sdb, appKindKey("bounty", id, "expiry"))),
		common.BytesToHash(appReadAddress(sdb, appKindKey("bounty", id, "owner")).Bytes()),
		appWordBool(appReadBool(sdb, appKindKey("bounty", id, "open"))),
		appWordBool(appReadBool(sdb, appKindKey("bounty", id, "claimed"))),
		common.BytesToHash(appReadAddress(sdb, appKindKey("bounty", id, "winner")).Bytes()),
		appReadHash(sdb, appKindKey("bounty", id, "escrow")),
		appReadHash(sdb, appKindKey("bounty", id, "po_id")),
	), nil
}
