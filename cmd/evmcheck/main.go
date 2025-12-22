package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	defaultForkBlock uint64 = 60000
	deployGasLimit   uint64 = 300000
	callGasLimit     uint64 = 300000
)

const (
	chainIDInitHex        = "0x6009600c60003960096000f34660005260206000f3"
	childInitHex          = "0x600a600c600039600a6000f3602a60005260206000f3"
	deployerInitPrefixHex = "0x602e600c600039602e6000f3601660186000396000600060166001f560005260206000f3"
)

const (
	push0RuntimeHex  = "0x5f5f5260205ff3"
	mcopyExpectedHex = "0x11223344556677889900aabbccddeeff00112233445566778899aabbccddeeff"
	mcopyRuntimeHex  = "0x7f11223344556677889900aabbccddeeff00112233445566778899aabbccddeeff5f5260205f60205e60206020f3"
	tstoreRuntimeHex = "0x602a5f5d5f5c5f5260205ff3"
)

func main() {
	rpcURL := flag.String("rpc", "", "RPC endpoint (e.g. http://HOST:8545)")
	pkHex := flag.String("pk", "", "hex private key (0x...)")
	chainIDFlag := flag.Uint64("chainid", 0, "expected chain ID (for CHAINID and tx signing)")
	forkBlock := flag.Uint64("forkblock", defaultForkBlock, "fork block height")
	flag.Parse()

	if *rpcURL == "" || *pkHex == "" || *chainIDFlag == 0 {
		fmt.Fprintln(os.Stderr, "Usage: evmcheck.exe --rpc http://HOST:8545 --pk 0xHEX --chainid 77777 --forkblock 60000")
		flag.PrintDefaults()
		os.Exit(1)
	}

	privKey, err := parsePrivateKey(*pkHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid private key: %v\n", err)
		os.Exit(1)
	}

	client, err := ethclient.Dial(*rpcURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to RPC: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get latest block: %v\n", err)
		os.Exit(1)
	}

	currentBlock := header.Number.Uint64()
	preFork := currentBlock < *forkBlock

	fmt.Printf("Current block: %d\n", currentBlock)
	fmt.Printf("Fork block: %d\n", *forkBlock)
	fmt.Printf("Pre-fork: %v\n", preFork)

	chainID := new(big.Int).SetUint64(*chainIDFlag)

	chainIDPass, chainIDMsg := checkChainID(ctx, client, privKey, chainID, preFork)
	printCheck("CHAINID opcode", chainIDPass, chainIDMsg)

	create2Pass, create2Msg := checkCreate2(ctx, client, privKey, chainID, preFork)
	printCheck("CREATE2 opcode", create2Pass, create2Msg)

	push0Pass, push0Msg := checkPush0(ctx, client, privKey, chainID, preFork)
	printCheck("PUSH0 opcode", push0Pass, push0Msg)

	mcopyPass, mcopyMsg := checkMCopy(ctx, client, privKey, chainID, preFork)
	printCheck("MCOPY opcode", mcopyPass, mcopyMsg)

	tstorePass, tstoreMsg := checkTStore(ctx, client, privKey, chainID, preFork)
	printCheck("TSTORE/TLOAD opcodes", tstorePass, tstoreMsg)

	selfdestructPass, selfdestructMsg := checkSelfdestruct(ctx, client, privKey, chainID, preFork)
	printCheck("SELFDESTRUCT (EIP-6780)", selfdestructPass, selfdestructMsg)

	if chainIDPass && create2Pass && push0Pass && mcopyPass && tstorePass && selfdestructPass {
		fmt.Println("EVM upgrade check: PASS")
		os.Exit(0)
	}
	fmt.Println("EVM upgrade check: FAIL")
	os.Exit(1)
}

func parsePrivateKey(pkHex string) (*ecdsa.PrivateKey, error) {
	pkHex = strings.TrimSpace(pkHex)
	pkHex = strings.TrimPrefix(pkHex, "0x")
	if pkHex == "" {
		return nil, fmt.Errorf("empty private key")
	}
	return crypto.HexToECDSA(pkHex)
}

func printCheck(label string, pass bool, msg string) {
	if pass {
		fmt.Printf("%s: PASS\n", label)
		return
	}
	if msg == "" {
		fmt.Printf("%s: FAIL\n", label)
		return
	}
	fmt.Printf("%s: FAIL (%s)\n", label, msg)
}

func checkChainID(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, preFork bool) (bool, string) {
	fromAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return false, fmt.Sprintf("nonce error: %v", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return false, fmt.Sprintf("gas price error: %v", err)
	}

	chainIDInit := common.FromHex(chainIDInitHex)
	tx, err := signAndSendTx(ctx, client, privKey, chainID, nonce, nil, chainIDInit, deployGasLimit, gasPrice)
	if err != nil {
		return false, fmt.Sprintf("deploy tx error: %v", err)
	}
	receipt, err := waitMined(ctx, client, tx)
	if err != nil {
		return false, fmt.Sprintf("deploy receipt error: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return false, fmt.Sprintf("deploy tx status %d", receipt.Status)
	}
	if receipt.ContractAddress == (common.Address{}) {
		return false, "deploy tx missing contract address"
	}

	callMsg := ethereum.CallMsg{
		From: fromAddr,
		To:   &receipt.ContractAddress,
		Gas:  callGasLimit,
	}
	out, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		if preFork {
			return false, fmt.Sprintf("expected pre-fork failure: %v", err)
		}
		return false, fmt.Sprintf("call error: %v", err)
	}
	if len(out) < 32 {
		if preFork {
			return false, fmt.Sprintf("unexpected pre-fork success: short output (%d bytes)", len(out))
		}
		return false, fmt.Sprintf("short output (%d bytes)", len(out))
	}

	gotChainID := new(big.Int).SetBytes(out)
	if gotChainID.Cmp(chainID) != 0 {
		if preFork {
			return false, fmt.Sprintf("unexpected pre-fork success: got chainid %s", gotChainID.String())
		}
		return false, fmt.Sprintf("chainid mismatch got %s want %s", gotChainID.String(), chainID.String())
	}

	if preFork {
		return false, fmt.Sprintf("unexpected pre-fork success: got chainid %s", gotChainID.String())
	}
	return true, ""
}

func checkPush0(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, preFork bool) (bool, string) {
	runtime := common.FromHex(push0RuntimeHex)
	initCode, err := buildInitCode(runtime)
	if err != nil {
		return false, fmt.Sprintf("initcode error: %v", err)
	}
	expected := make([]byte, 32)
	return checkOpcodeCall(ctx, client, privKey, chainID, preFork, initCode, expected)
}

func checkMCopy(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, preFork bool) (bool, string) {
	runtime := common.FromHex(mcopyRuntimeHex)
	initCode, err := buildInitCode(runtime)
	if err != nil {
		return false, fmt.Sprintf("initcode error: %v", err)
	}
	expected := common.FromHex(mcopyExpectedHex)
	if len(expected) != 32 {
		return false, fmt.Sprintf("expected output length %d", len(expected))
	}
	return checkOpcodeCall(ctx, client, privKey, chainID, preFork, initCode, expected)
}

func checkTStore(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, preFork bool) (bool, string) {
	runtime := common.FromHex(tstoreRuntimeHex)
	initCode, err := buildInitCode(runtime)
	if err != nil {
		return false, fmt.Sprintf("initcode error: %v", err)
	}
	expected := common.LeftPadBytes([]byte{0x2a}, 32)
	return checkOpcodeCall(ctx, client, privKey, chainID, preFork, initCode, expected)
}

func checkSelfdestruct(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, preFork bool) (bool, string) {
	fromAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return false, fmt.Sprintf("gas price error: %v", err)
	}

	runtime := buildSelfdestructRuntime(fromAddr)
	initCode, err := buildInitCode(runtime)
	if err != nil {
		return false, fmt.Sprintf("initcode error: %v", err)
	}

	deployReceipt, err := deployContract(ctx, client, privKey, chainID, initCode, gasPrice)
	if err != nil {
		return false, fmt.Sprintf("deploy error: %v", err)
	}
	if deployReceipt.Status != types.ReceiptStatusSuccessful {
		return false, fmt.Sprintf("deploy tx status %d", deployReceipt.Status)
	}
	if deployReceipt.ContractAddress == (common.Address{}) {
		return false, "deploy missing contract address"
	}

	codeBefore, err := client.CodeAt(ctx, deployReceipt.ContractAddress, nil)
	if err != nil {
		return false, fmt.Sprintf("getCode error: %v", err)
	}
	if len(codeBefore) == 0 {
		return false, "deployed code is empty"
	}

	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return false, fmt.Sprintf("nonce error: %v", err)
	}
	callTx, err := signAndSendTx(ctx, client, privKey, chainID, nonce, &deployReceipt.ContractAddress, nil, callGasLimit, gasPrice)
	if err != nil {
		return false, fmt.Sprintf("selfdestruct tx error: %v", err)
	}
	callReceipt, err := waitMined(ctx, client, callTx)
	if err != nil {
		return false, fmt.Sprintf("selfdestruct receipt error: %v", err)
	}
	if callReceipt.Status != types.ReceiptStatusSuccessful {
		return false, fmt.Sprintf("selfdestruct tx status %d", callReceipt.Status)
	}

	codeAfter, err := client.CodeAt(ctx, deployReceipt.ContractAddress, nil)
	if err != nil {
		return false, fmt.Sprintf("getCode error: %v", err)
	}

	if preFork {
		if len(codeAfter) == 0 {
			return false, "expected pre-fork behavior: code deleted"
		}
		return false, "unexpected pre-fork behavior: code preserved"
	}

	if len(codeAfter) == 0 {
		return false, "code deleted (EIP-6780 not active)"
	}
	return true, ""
}

func checkCreate2(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, preFork bool) (bool, string) {
	fromAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return false, fmt.Sprintf("gas price error: %v", err)
	}

	deployerInitHex := deployerInitPrefixHex + strings.TrimPrefix(childInitHex, "0x")
	deployerInit := common.FromHex(deployerInitHex)

	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return false, fmt.Sprintf("nonce error: %v", err)
	}
	deployTx, err := signAndSendTx(ctx, client, privKey, chainID, nonce, nil, deployerInit, deployGasLimit, gasPrice)
	if err != nil {
		return false, fmt.Sprintf("deployer deploy error: %v", err)
	}
	deployReceipt, err := waitMined(ctx, client, deployTx)
	if err != nil {
		return false, fmt.Sprintf("deployer receipt error: %v", err)
	}
	if deployReceipt.Status != types.ReceiptStatusSuccessful {
		return false, fmt.Sprintf("deployer tx status %d", deployReceipt.Status)
	}
	if deployReceipt.ContractAddress == (common.Address{}) {
		return false, "deployer missing contract address"
	}

	nonce, err = client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return false, fmt.Sprintf("nonce error: %v", err)
	}
	create2Tx, err := signAndSendTx(ctx, client, privKey, chainID, nonce, &deployReceipt.ContractAddress, nil, callGasLimit, gasPrice)
	if err != nil {
		return false, fmt.Sprintf("create2 tx error: %v", err)
	}
	create2Receipt, err := waitMined(ctx, client, create2Tx)
	if err != nil {
		return false, fmt.Sprintf("create2 receipt error: %v", err)
	}

	childInit := common.FromHex(childInitHex)
	childInitHash := crypto.Keccak256Hash(childInit)
	salt := common.BigToHash(big.NewInt(1))
	childAddr := crypto.CreateAddress2(deployReceipt.ContractAddress, salt, childInitHash.Bytes())

	code, err := client.CodeAt(ctx, childAddr, nil)
	if err != nil {
		return false, fmt.Sprintf("getCode error: %v", err)
	}

	if preFork {
		if create2Receipt.Status == types.ReceiptStatusSuccessful && len(code) > 0 {
			return false, "unexpected CREATE2 success before fork"
		}
		if create2Receipt.Status == types.ReceiptStatusSuccessful && len(code) == 0 {
			return false, "unexpected CREATE2 success before fork (child code empty)"
		}
		return false, fmt.Sprintf("expected pre-fork failure: receipt status %d", create2Receipt.Status)
	}

	if create2Receipt.Status != types.ReceiptStatusSuccessful {
		return false, fmt.Sprintf("create2 tx status %d", create2Receipt.Status)
	}
	if len(code) == 0 {
		return false, "child code is empty"
	}

	return true, ""
}

func checkOpcodeCall(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, preFork bool, initCode []byte, expected []byte) (bool, string) {
	fromAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return false, fmt.Sprintf("gas price error: %v", err)
	}

	receipt, err := deployContract(ctx, client, privKey, chainID, initCode, gasPrice)
	if err != nil {
		return false, fmt.Sprintf("deploy error: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		if preFork {
			return false, fmt.Sprintf("expected pre-fork failure: deploy status %d", receipt.Status)
		}
		return false, fmt.Sprintf("deploy tx status %d", receipt.Status)
	}
	if receipt.ContractAddress == (common.Address{}) {
		return false, "deploy missing contract address"
	}

	callMsg := ethereum.CallMsg{
		From: fromAddr,
		To:   &receipt.ContractAddress,
		Gas:  callGasLimit,
	}
	out, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		if preFork {
			return false, fmt.Sprintf("expected pre-fork failure: %v", err)
		}
		return false, fmt.Sprintf("call error: %v", err)
	}
	if len(out) != len(expected) {
		if preFork {
			return false, fmt.Sprintf("unexpected pre-fork success: output len %d", len(out))
		}
		return false, fmt.Sprintf("unexpected output length %d", len(out))
	}
	if !bytes.Equal(out, expected) {
		if preFork {
			return false, fmt.Sprintf("unexpected pre-fork success: output 0x%s", common.Bytes2Hex(out))
		}
		return false, fmt.Sprintf("output mismatch got 0x%s", common.Bytes2Hex(out))
	}
	if preFork {
		return false, fmt.Sprintf("unexpected pre-fork success: output 0x%s", common.Bytes2Hex(out))
	}
	return true, ""
}

func buildInitCode(runtime []byte) ([]byte, error) {
	if len(runtime) > 0xff {
		return nil, fmt.Errorf("runtime too long: %d", len(runtime))
	}
	length := byte(len(runtime))
	init := []byte{
		0x60, length,
		0x60, 0x0c,
		0x60, 0x00,
		0x39,
		0x60, length,
		0x60, 0x00,
		0xf3,
	}
	return append(init, runtime...), nil
}

func buildSelfdestructRuntime(beneficiary common.Address) []byte {
	runtime := make([]byte, 0, 22)
	runtime = append(runtime, 0x73)
	runtime = append(runtime, beneficiary.Bytes()...)
	runtime = append(runtime, 0xff)
	return runtime
}

func deployContract(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, initCode []byte, gasPrice *big.Int) (*types.Receipt, error) {
	fromAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return nil, err
	}
	tx, err := signAndSendTx(ctx, client, privKey, chainID, nonce, nil, initCode, deployGasLimit, gasPrice)
	if err != nil {
		return nil, err
	}
	return waitMined(ctx, client, tx)
}

func signAndSendTx(ctx context.Context, client *ethclient.Client, privKey *ecdsa.PrivateKey, chainID *big.Int, nonce uint64, to *common.Address, data []byte, gasLimit uint64, gasPrice *big.Int) (*types.Transaction, error) {
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privKey)
	if err != nil {
		return nil, err
	}
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return nil, err
	}
	return signedTx, nil
}

func waitMined(ctx context.Context, client *ethclient.Client, tx *types.Transaction) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, err
	}
	if receipt == nil {
		return nil, fmt.Errorf("receipt not found")
	}
	return receipt, nil
}
