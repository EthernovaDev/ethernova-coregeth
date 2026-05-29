package core

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params/ethernova"
	"github.com/holiman/uint256"
)

const resourceTxPreforkGasLimit = uint64(21_000)

func newResourceTxPrecheck(t *testing.T, blockNumber uint64) *StateTransition {
	t.Helper()

	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		t.Fatalf("failed to create state db: %v", err)
	}

	from := common.HexToAddress("0x0000000000000000000000000000000000000abc")
	to := common.HexToAddress("0x0000000000000000000000000000000000000def")
	statedb.SetBalance(from, uint256.NewInt(1_000_000_000))

	msg := &Message{
		From:      from,
		To:        &to,
		GasLimit:  resourceTxPreforkGasLimit,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(0),
		Value:     big.NewInt(0),
		ResourceLimits: &types.ResourceLimits{
			Compute:     resourceTxPreforkGasLimit,
			StateRead:   resourceTxPreforkGasLimit,
			StateWrite:  resourceTxPreforkGasLimit,
			ProtocolOps: resourceTxPreforkGasLimit,
			ProofVerify: resourceTxPreforkGasLimit,
		},
	}

	cfg := makeTestConfig(nil, nil)
	blockCtx := vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.HexToAddress("0x0000000000000000000000000000000000000001"),
		GasLimit:    resourceTxPreforkGasLimit,
		BlockNumber: new(big.Int).SetUint64(blockNumber),
		Time:        0,
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(0),
	}
	evm := vm.NewEVM(blockCtx, vm.TxContext{Origin: from, GasPrice: msg.GasPrice}, statedb, cfg, vm.Config{})
	gp := new(GasPool).AddGas(resourceTxPreforkGasLimit)
	return NewStateTransition(evm, msg, gp)
}

func TestResourceTxRejectedBeforeResourceMeteringFork(t *testing.T) {
	st := newResourceTxPrecheck(t, ethernova.ResourceMeteringForkBlock-1)
	if err := st.preCheck(); !errors.Is(err, types.ErrTxTypeNotSupported) {
		t.Fatalf("expected resource tx to be rejected before fork, got %v", err)
	}
}

func TestResourceTxAcceptedAtResourceMeteringFork(t *testing.T) {
	st := newResourceTxPrecheck(t, ethernova.ResourceMeteringForkBlock)
	if err := st.preCheck(); err != nil {
		t.Fatalf("expected resource tx precheck at fork to pass, got %v", err)
	}
}
