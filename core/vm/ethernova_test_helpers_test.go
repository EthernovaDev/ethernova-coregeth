package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/ethernova"
	"github.com/holiman/uint256"
)

// newTestEVM wires up an in-memory StateDB and an EVM suitable for invoking
// stateful Ethernova precompiles directly. BlockNumber is set high enough to
// clear every NIP-0004 fork block.
func newTestEVM(t *testing.T) (*EVM, *state.StateDB) {
	t.Helper()
	sdb, err := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	ctx := BlockContext{
		BlockNumber: big.NewInt(1_000_000),
		Transfer:    func(StateDB, common.Address, common.Address, *uint256.Int) {},
	}
	cfg := *params.AllEthashProtocolChanges
	cfg.ChainID = ethernova.NewChainIDBig
	evm := NewEVM(ctx, TxContext{}, sdb, &cfg, Config{})
	return evm, sdb
}
