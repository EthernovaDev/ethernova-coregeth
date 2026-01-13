package ethernova

import "math/big"

const (
	OldChainID         uint64 = 77777
	NewChainID         uint64 = 121525
	SplitFixBlock      uint64 = 138396
	ChainIDSwitchBlock uint64 = SplitFixBlock
)

var (
	OldChainIDBig         = new(big.Int).SetUint64(OldChainID)
	NewChainIDBig         = new(big.Int).SetUint64(NewChainID)
	SplitFixBlockBig      = new(big.Int).SetUint64(SplitFixBlock)
	ChainIDSwitchBlockBig = SplitFixBlockBig
)

func IsPreSwitch(number *big.Int) bool {
	if number == nil {
		return true
	}
	return number.Cmp(SplitFixBlockBig) < 0
}

func ChainIDForBlock(number *big.Int) *big.Int {
	if IsPreSwitch(number) {
		return new(big.Int).Set(OldChainIDBig)
	}
	return new(big.Int).Set(NewChainIDBig)
}

func IsEthernovaChainID(chainID *big.Int) bool {
	if chainID == nil {
		return false
	}
	return chainID.Cmp(OldChainIDBig) == 0 || chainID.Cmp(NewChainIDBig) == 0
}
