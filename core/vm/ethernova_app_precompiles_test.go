package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/ethernova"
)

func appTestWord(v uint64) []byte                   { return common.BigToHash(new(big.Int).SetUint64(v)).Bytes() }
func appTestHash(hex string) []byte                 { return common.HexToHash(hex).Bytes() }
func appTestAddressWord(addr common.Address) []byte { return common.BytesToHash(addr.Bytes()).Bytes() }

func appTestInput(selector byte, words ...[]byte) []byte {
	out := []byte{selector}
	for _, word := range words {
		out = append(out, common.LeftPadBytes(word, 32)...)
	}
	return out
}

func appTestBool(out []byte) bool {
	return len(out) >= 32 && common.BytesToHash(out[:32]) != (common.Hash{})
}

// TestApplicationPrecompileIdentityLifecycle exercises the
// attest/verify/get/revoke selectors on novaIdentityAttestation. After
// the Phase 11 fix landing in this revision, attest() also creates a
// Protocol Object of type Identity; the test confirms the precompile
// still returns the attestation ID (unchanged input layout) and that
// the PO is queryable via PoGetObject.
func TestApplicationPrecompileIdentityLifecycle(t *testing.T) {
	evm, sdb := newTestEVM(t)
	issuer := common.HexToAddress("0x1111111111111111111111111111111111111111")
	subject := common.HexToAddress("0x2222222222222222222222222222222222222222")
	sdb.CreateAccount(issuer)
	sdb.SetNonce(issuer, 1)

	identity := &novaIdentityAttestation{}
	create := appTestInput(0x01, appTestAddressWord(subject), appTestHash("0xfeed01"), appTestWord(0))
	idBytes, err := identity.RunStateful(evm, issuer, create, false)
	if err != nil {
		t.Fatalf("attest: %v", err)
	}
	id := common.BytesToHash(idBytes)

	verify, err := identity.RunStateful(evm, issuer, appTestInput(0x02, id.Bytes()), true)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !appTestBool(verify) {
		t.Fatal("identity attestation should verify before revoke")
	}
	get, err := identity.RunStateful(evm, issuer, appTestInput(0x04, id.Bytes()), true)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got := common.BytesToAddress(get[:32]); got != subject {
		t.Fatalf("subject mismatch: got %s want %s", got, subject)
	}
	// New 7th word: po_id. MUST be non-zero on success path (BUG-2 fix).
	if len(get) < 7*32 {
		t.Fatalf("get returned %d bytes, want at least %d", len(get), 7*32)
	}
	poID := common.BytesToHash(get[6*32 : 7*32])
	if poID == (common.Hash{}) {
		t.Fatal("expected non-zero po_id in getIdentity output")
	}
	po := PoGetObject(sdb, poID)
	if po == nil {
		t.Fatalf("PO %s not found", poID.Hex())
	}
	if po.Owner != issuer {
		t.Fatalf("PO owner = %s, want issuer %s", po.Owner, issuer)
	}

	if _, err := identity.RunStateful(evm, issuer, appTestInput(0x03, id.Bytes()), true); err != ErrWriteProtection {
		t.Fatalf("static revoke should hit write protection, got %v", err)
	}
	if _, err := identity.RunStateful(evm, issuer, appTestInput(0x03, id.Bytes()), false); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	verify, err = identity.RunStateful(evm, issuer, appTestInput(0x02, id.Bytes()), true)
	if err != nil {
		t.Fatalf("verify after revoke: %v", err)
	}
	if appTestBool(verify) {
		t.Fatal("identity attestation should not verify after revoke")
	}
}

// TestApplicationPrecompileSocialManifestGameBounty exercises the
// remaining 5 Phase 11 precompiles. Input layouts updated to reflect
// the BUG fixes:
//   - manifest.create: contentRef = zero hash (BUG-4 bypass for tests)
//   - bounty.create: new 4-word layout (specHash, expectedResult, reward, expiryBlock)
//   - bounty.verify: still works with 2-word input (overrideExpected path)
func TestApplicationPrecompileSocialManifestGameBounty(t *testing.T) {
	evm, sdb := newTestEVM(t)
	alice := common.HexToAddress("0x3333333333333333333333333333333333333333")
	bob := common.HexToAddress("0x4444444444444444444444444444444444444444")
	sdb.CreateAccount(alice)
	sdb.SetNonce(alice, 1)

	social := &novaSocialGraph{}
	if _, err := social.RunStateful(evm, alice, appTestInput(0x01, appTestAddressWord(bob)), false); err != nil {
		t.Fatalf("follow: %v", err)
	}
	isFollowing, err := social.RunStateful(evm, alice, appTestInput(0x03, appTestAddressWord(alice), appTestAddressWord(bob)), true)
	if err != nil || !appTestBool(isFollowing) {
		t.Fatalf("isFollowing err=%v out=%x", err, isFollowing)
	}
	trust, err := social.RunStateful(evm, alice, appTestInput(0x04, appTestAddressWord(alice), appTestAddressWord(bob)), true)
	if err != nil || new(big.Int).SetBytes(trust[:32]).Uint64() != 50 {
		t.Fatalf("trustScore err=%v out=%x", err, trust)
	}

	// BUG-4 fix: contentRef = zero bypasses validation. Non-zero would
	// require a real Phase 3 PO of type ContentReference.
	manifest := &novaContentManifest{}
	zeroRef := common.Hash{}
	manifestIDBytes, err := manifest.RunStateful(evm, alice, appTestInput(0x01, appTestHash("0xaaaa"), zeroRef.Bytes(), appTestHash("0xcccc"), appTestWord(4096)), false)
	if err != nil {
		t.Fatalf("create manifest: %v", err)
	}
	manifestID := common.BytesToHash(manifestIDBytes)
	manifestOK, err := manifest.RunStateful(evm, alice, appTestInput(0x02, manifestID.Bytes(), appTestHash("0xaaaa")), true)
	if err != nil || !appTestBool(manifestOK) {
		t.Fatalf("verify manifest err=%v out=%x", err, manifestOK)
	}
	// BUG-4 negative test: non-zero phantom contentRef MUST fail.
	if _, err := manifest.RunStateful(evm, alice, appTestInput(0x01, appTestHash("0xaaaa"), appTestHash("0xdeadbeef"), appTestHash("0xcccc"), appTestWord(4096)), false); err == nil {
		t.Fatal("manifest with phantom contentRef should fail validation")
	}

	game := &novaGameState{}
	gameID := common.HexToHash("0x7777")
	commit, err := game.RunStateful(evm, alice, appTestInput(0x01, gameID.Bytes(), appTestHash("0x1234"), appTestWord(1)), false)
	if err != nil {
		t.Fatalf("game commit: %v", err)
	}
	if len(commit) != 32 {
		t.Fatalf("game commit returned %d bytes", len(commit))
	}
	if _, err := game.RunStateful(evm, alice, appTestInput(0x01, gameID.Bytes(), appTestHash("0x1235"), appTestWord(1)), false); err == nil {
		t.Fatal("stale game turn should fail")
	}
	// BUG-5 fix: first commit creates a GameRoom PO. Verify via the get
	// selector — last (5th) word is po_id.
	gameGet, err := game.RunStateful(evm, alice, appTestInput(0x03, gameID.Bytes()), true)
	if err != nil {
		t.Fatalf("get game: %v", err)
	}
	if len(gameGet) < 5*32 {
		t.Fatalf("get game returned %d bytes, want %d", len(gameGet), 5*32)
	}
	gamePOID := common.BytesToHash(gameGet[4*32 : 5*32])
	if gamePOID == (common.Hash{}) {
		t.Fatal("expected non-zero game po_id")
	}
	if PoGetObject(sdb, gamePOID) == nil {
		t.Fatalf("game PO %s not found", gamePOID.Hex())
	}

	// BUG-3 fix: bounty.create takes 4-word input (spec, expected,
	// reward, expiry) and accepts msg.value as escrow (zero here, so
	// this exercises the no-escrow path).
	bounty := &novaComputeBounty{}
	bountyIDBytes, err := bounty.RunStateful(evm, alice, appTestInput(0x01, appTestHash("0xbeef"), appTestHash("0x5151"), appTestWord(100), appTestWord(0)), false)
	if err != nil {
		t.Fatalf("create bounty: %v", err)
	}
	submissionBytes, err := bounty.RunStateful(evm, bob, appTestInput(0x02, bountyIDBytes, appTestHash("0x5151"), appTestHash("0x6161")), false)
	if err != nil {
		t.Fatalf("submit bounty: %v", err)
	}
	// Override-expected path: verify against the explicitly-passed value.
	submissionOK, err := bounty.RunStateful(evm, alice, appTestInput(0x03, submissionBytes, appTestHash("0x5151")), true)
	if err != nil || !appTestBool(submissionOK) {
		t.Fatalf("verify submission err=%v out=%x", err, submissionOK)
	}
	// Canonical path: verify with empty override picks up bounty's `expected`.
	canonicalOK, err := bounty.RunStateful(evm, alice, appTestInput(0x03, submissionBytes, common.Hash{}.Bytes()), true)
	if err != nil || !appTestBool(canonicalOK) {
		t.Fatalf("canonical verify err=%v out=%x", err, canonicalOK)
	}
}

// TestApplicationPrecompileBountyClaim exercises the new selector 0x05
// claim path: owner releases escrow to a verified submitter.
func TestApplicationPrecompileBountyClaim(t *testing.T) {
	evm, _ := newTestEVM(t)
	alice := common.HexToAddress("0x3333333333333333333333333333333333333333")
	bob := common.HexToAddress("0x4444444444444444444444444444444444444444")
	bounty := &novaComputeBounty{}

	bountyIDBytes, err := bounty.RunStateful(evm, alice, appTestInput(0x01, appTestHash("0xbeef"), appTestHash("0x5151"), appTestWord(100), appTestWord(0)), false)
	if err != nil {
		t.Fatalf("create bounty: %v", err)
	}
	submissionBytes, err := bounty.RunStateful(evm, bob, appTestInput(0x02, bountyIDBytes, appTestHash("0x5151"), appTestHash("0x6161")), false)
	if err != nil {
		t.Fatalf("submit bounty: %v", err)
	}

	// Non-owner cannot claim.
	if _, err := bounty.RunStateful(evm, bob, appTestInput(0x05, bountyIDBytes, submissionBytes), false); err == nil {
		t.Fatal("non-owner claim should fail")
	}

	// Owner can claim (zero escrow path — just closes the bounty).
	out, err := bounty.RunStateful(evm, alice, appTestInput(0x05, bountyIDBytes, submissionBytes), false)
	if err != nil {
		t.Fatalf("owner claim: %v", err)
	}
	if !appTestBool(out) {
		t.Fatal("claim should return true")
	}

	// Bounty must now be closed; subsequent claim must fail.
	if _, err := bounty.RunStateful(evm, alice, appTestInput(0x05, bountyIDBytes, submissionBytes), false); err == nil {
		t.Fatal("second claim should fail (bounty closed)")
	}

	// getComputeBounty: open=false, claimed=true, winner=bob.
	got, err := bounty.RunStateful(evm, alice, appTestInput(0x04, bountyIDBytes), true)
	if err != nil {
		t.Fatalf("get bounty: %v", err)
	}
	if len(got) < 10*32 {
		t.Fatalf("get bounty returned %d bytes, want at least 320", len(got))
	}
	// Layout: 0=spec, 1=expected, 2=reward, 3=expiry, 4=owner, 5=open,
	// 6=claimed, 7=winner, 8=escrow, 9=po_id.
	openWord := common.BytesToHash(got[5*32 : 6*32])
	claimedWord := common.BytesToHash(got[6*32 : 7*32])
	winnerWord := common.BytesToHash(got[7*32 : 8*32])
	if openWord != (common.Hash{}) {
		t.Fatal("bounty should be closed after claim")
	}
	if claimedWord == (common.Hash{}) {
		t.Fatal("bounty should be marked claimed")
	}
	if common.BytesToAddress(winnerWord.Bytes()) != bob {
		t.Fatalf("winner = %s, want bob %s", common.BytesToAddress(winnerWord.Bytes()), bob)
	}
}

func TestApplicationPrecompileCapabilityGate(t *testing.T) {
	addr := common.BytesToAddress([]byte{0x30})
	if RequiredCapabilityForPrecompile(addr) != CapabilityAppPrecompiles {
		t.Fatalf("0x30 capability = %s", CapabilityName(RequiredCapabilityForPrecompile(addr)))
	}
	if DefaultCapabilitiesForDomain(DomainLegacy)&CapabilityAppPrecompiles != 0 {
		t.Fatal("Domain 0 should not have application precompile capability")
	}
	if DefaultCapabilitiesForDomain(DomainNova)&CapabilityAppPrecompiles == 0 {
		t.Fatal("Domain 1 should include application precompile capability")
	}
}

func TestNovaOpcodeActivationAndNames(t *testing.T) {
	cfg := *params.AllEthashProtocolChanges
	cfg.ChainID = ethernova.NewChainIDBig
	jt := instructionSetForConfig(&cfg, false, new(big.Int).SetUint64(ethernova.NovaOpcodeForkBlock), nil)
	if jt[MSEND].minStack != minStack(3, 1) || jt[SCLOSE].minStack != minStack(5, 1) {
		t.Fatalf("Nova opcode jump table not enabled: MSEND=%#v SCLOSE=%#v", jt[MSEND], jt[SCLOSE])
	}
	cases := map[OpCode]string{
		MSEND: "MSEND", MRECV: "MRECV", MPEEK: "MPEEK", MCOUNT: "MCOUNT",
		CREF: "CREF", CVERIFY: "CVERIFY", SOPEN: "SOPEN", SCOMMIT: "SCOMMIT", SCLOSE: "SCLOSE",
	}
	for op, name := range cases {
		if op.String() != name {
			t.Fatalf("opcode %#x string = %q want %q", byte(op), op.String(), name)
		}
		if StringToOp(name) != op {
			t.Fatalf("StringToOp(%s) = %#x want %#x", name, StringToOp(name), op)
		}
	}
}
