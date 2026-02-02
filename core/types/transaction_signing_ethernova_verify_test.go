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

func TestVerifyChainIDGate(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	cfg := &coregeth.CoreGethChainConfig{
		ChainID:     new(big.Int).Set(ethernova.NewChainIDBig),
		EIP155Block: big.NewInt(0),
	}

	tx := NewTransaction(0, common.HexToAddress("0x1"), big.NewInt(1), 21000, big.NewInt(1), nil)
	wrongSigned, err := SignTx(tx, NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		t.Fatalf("sign wrong chainId tx: %v", err)
	}
	correctSigned, err := SignTx(tx, NewEIP155Signer(new(big.Int).Set(ethernova.NewChainIDBig)), key)
	if err != nil {
		t.Fatalf("sign correct chainId tx: %v", err)
	}

	signer := MakeSigner(cfg, big.NewInt(0), 0)
	if _, err := Sender(signer, wrongSigned); !errors.Is(err, ErrInvalidChainId) {
		t.Fatalf("wrong chainId: expected ErrInvalidChainId, got %v", err)
	}
	if from, err := Sender(signer, correctSigned); err != nil || from != addr {
		t.Fatalf("correct chainId: from=%s err=%v", from, err)
	}
}
