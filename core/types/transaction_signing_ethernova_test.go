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

func TestEthernovaChainIDSwitch(t *testing.T) {
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

	preBlock := new(big.Int).Sub(ethernova.SplitFixBlockBig, big.NewInt(1))
	preSigner := MakeSigner(cfg, preBlock, 0)
	if from, err := Sender(preSigner, oldSigned); err != nil || from != addr {
		t.Fatalf("pre-switch old chainId: from=%s err=%v", from, err)
	}
	if from, err := Sender(preSigner, newSigned); err != nil || from != addr {
		t.Fatalf("pre-switch new chainId: from=%s err=%v", from, err)
	}

	postSigner := MakeSigner(cfg, new(big.Int).Set(ethernova.SplitFixBlockBig), 0)
	if _, err := Sender(postSigner, oldSigned); !errors.Is(err, ErrInvalidChainId) {
		t.Fatalf("post-switch old chainId: expected ErrInvalidChainId, got %v", err)
	}
	if from, err := Sender(postSigner, newSigned); err != nil || from != addr {
		t.Fatalf("post-switch new chainId: from=%s err=%v", from, err)
	}
}
