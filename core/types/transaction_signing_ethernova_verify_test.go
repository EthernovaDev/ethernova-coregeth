package types

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params/ethernova"
	coregeth "github.com/ethereum/go-ethereum/params/types/coregeth"
)

func TestVerifyChainIDGatePostFork(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	cfg := &coregeth.CoreGethChainConfig{
		ChainID:     big.NewInt(int64(ethernova.OldChainID)),
		EIP155Block: big.NewInt(0),
	}

	tx := NewTransaction(0, common.HexToAddress("0x1"), big.NewInt(1), 21000, big.NewInt(1), nil)
	oldSigned, err := SignTx(tx, NewEIP155Signer(new(big.Int).Set(ethernova.OldChainIDBig)), key)
	if err != nil {
		t.Fatalf("sign old chainId tx: %v", err)
	}
	newSigned, err := SignTx(tx, NewEIP155Signer(new(big.Int).Set(ethernova.NewChainIDBig)), key)
	if err != nil {
		t.Fatalf("sign new chainId tx: %v", err)
	}

	signer := MakeSigner(cfg, new(big.Int).Set(ethernova.SplitFixBlockBig), 0)
	if _, err := Sender(signer, oldSigned); !errors.Is(err, ErrInvalidChainId) {
		t.Fatalf("post-fork old chainId: expected ErrInvalidChainId, got %v", err)
	} else {
		t.Logf("VERIFY_CHAINID_GATE: block=%d old_chainId=%d err=%v", ethernova.SplitFixBlock, ethernova.OldChainID, err)
	}
	if from, err := Sender(signer, newSigned); err != nil || from != addr {
		t.Fatalf("post-fork new chainId: from=%s err=%v", from, err)
	} else {
		t.Logf("VERIFY_CHAINID_GATE: block=%d new_chainId=%d accepted_from=%s", ethernova.SplitFixBlock, ethernova.NewChainID, from.Hex())
	}
}
