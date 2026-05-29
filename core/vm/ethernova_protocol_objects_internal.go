// Ethernova: Internal Protocol Object Create Helper (NIP-0004 Phase 11 fix)
//
// This file exposes a package-internal helper that Phase 11 application
// precompiles (and any future cross-precompile callers) can use to mint
// a Protocol Object without going through the 0x29 precompile's input
// encoding. The original Phase 11 audit (see Phase11 README.md, BUG-2)
// flagged that the types ProtoTypeIdentity, ProtoTypeSubscription, and
// ProtoTypeGameRoom were registered but never instantiated by any
// precompile; this helper closes that gap by giving each Phase 11
// create-path a single line of code to produce a PO.
//
// Determinism: the ID derivation uses (caller || blockNumber || global_nonce)
// — identical to novaProtocolObjectRegistry.createObject — so external
// observers cannot distinguish a PO minted via Phase 11 from one minted
// via the 0x29 precompile. The global_nonce is the SAME counter on the
// same registry account, so it advances monotonically across both paths
// and IDs never collide.

package vm

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// PoCreateObjectInternal creates a Protocol Object owned by `caller`,
// with the given type tag and state data. It returns the deterministic
// ID assigned to the object.
//
// This is the in-process counterpart to novaProtocolObjectRegistry's
// createObject selector — both go through the same write path so the
// ID space and storage layout are identical.
//
// The function does NOT validate that `caller` is allowed to mint a PO
// of the requested type. Such policy checks (e.g. "only the Phase 11
// IdentityAttestation precompile may mint a Protocol Object of type
// Identity") are the caller's responsibility.
//
// On consensus-critical paths this helper MUST be called by validator
// AND miner with identical inputs. The deferred queue precedent applies.
func PoCreateObjectInternal(evm *EVM, caller common.Address, typeTag uint8, stateData []byte, expiryBlock uint64, rentPrepay *big.Int) (common.Hash, error) {
	if !types.IsValidProtocolObjectType(typeTag) {
		return common.Hash{}, errors.New("PoCreateObjectInternal: invalid type tag")
	}
	if rentPrepay == nil {
		rentPrepay = new(big.Int)
	}
	sdb := evm.StateDB
	blockNum := evm.Context.BlockNumber.Uint64()

	// Same ID derivation as createObject: keccak256(caller || block_be || nonce_be).
	globalNonce := poReadUint64(sdb, poKeyGlobalNonce())
	var blockBuf, nonceBuf [8]byte
	binary.BigEndian.PutUint64(blockBuf[:], blockNum)
	binary.BigEndian.PutUint64(nonceBuf[:], globalNonce)
	idInput := make([]byte, 0, 20+8+8)
	idInput = append(idInput, caller.Bytes()...)
	idInput = append(idInput, blockBuf[:]...)
	idInput = append(idInput, nonceBuf[:]...)
	id := crypto.Keccak256Hash(idInput)
	poWriteUint64(sdb, poKeyGlobalNonce(), globalNonce+1)

	obj := &types.ProtocolObject{
		ID:               id,
		Owner:            caller,
		TypeTag:          typeTag,
		StateData:        stateData,
		ExpiryBlock:      expiryBlock,
		LastTouchedBlock: blockNum,
		RentBalance:      rentPrepay,
	}
	data, err := obj.EncodeRLP()
	if err != nil {
		return common.Hash{}, err
	}

	// Ensure the registry account is materialised. Cheap no-op after the
	// first invocation in the chain's lifetime.
	poEnsureRegistryExists(sdb)

	// Presence marker + RLP-encoded body.
	sdb.SetState(ProtocolObjectRegistryAddr, poKeyObject(id), common.BytesToHash([]byte{0x01}))
	poWriteRLP(sdb, id, data)

	// Owner index + counts. Same logic as createObject.
	slotsUsed := poReadUint64(sdb, poKeyOwnerSlotsUsed(caller))
	sdb.SetState(ProtocolObjectRegistryAddr, poKeyOwnerIndex(caller, slotsUsed), id)
	poWriteUint64(sdb, poKeyOwnerSlotOf(id), slotsUsed)
	poWriteUint64(sdb, poKeyOwnerSlotsUsed(caller), slotsUsed+1)
	ownerCount := poReadUint64(sdb, poKeyOwnerCount(caller))
	poWriteUint64(sdb, poKeyOwnerCount(caller), ownerCount+1)

	totalCount := PoGetObjectCount(sdb)
	poWriteUint64(sdb, poKeyTotalCount(), totalCount+1)
	typeCount := PoGetTypeCount(sdb, typeTag)
	poWriteUint64(sdb, poKeyTypeCount(typeTag), typeCount+1)

	return id, nil
}
