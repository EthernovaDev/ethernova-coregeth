package eth

import (
	"github.com/ethereum/go-ethereum/params/ethernova"
)

type ApplicationPrecompileInfo struct {
	Address     string   `json:"address"`
	Name        string   `json:"name"`
	Selectors   []string `json:"selectors"`
	Description string   `json:"description"`
}

type NovaOpcodeInfo struct {
	Opcode      string `json:"opcode"`
	Name        string `json:"name"`
	Bridge      string `json:"bridge"`
	Description string `json:"description"`
}

// ApplicationPrecompiles exposes the NIP-0004 Phase 11 app-layer precompile
// map for SDKs, explorers, and external test harnesses. Wire names:
// ethernova_applicationPrecompiles and nova_applicationPrecompiles.
func (api *EthernovaAPI) ApplicationPrecompiles() map[string]interface{} {
	head := api.e.blockchain.CurrentBlock().Number.Uint64()
	return map[string]interface{}{
		"phase":         11,
		"active":        head >= ethernova.ApplicationPrecompileForkBlock,
		"forkBlock":     ethernova.ApplicationPrecompileForkBlock,
		"currentBlock":  head,
		"addressPolicy": "0x2B/0x2C are already ContentRegistry/MailboxManager and 0x35 remains MailboxOps; Phase 11 uses free slots 0x30-0x34 and 0x36.",
		"precompiles": []ApplicationPrecompileInfo{
			{
				Address:     "0x30",
				Name:        "novaAsyncCallback",
				Selectors:   []string{"0x01 register", "0x02 get", "0x03 markFired", "0x04 ready"},
				Description: "Register deterministic callback commitments with chain-driven dispatch at the target block via a per-target-block index (see core.ProcessApplicationAsyncCallbacks). Register input is 5 words (condition, target_addr, selector_word, payloadHash, targetBlock). Also enqueues an EffectTypeAsyncCallback for nova_getDeferredStats visibility.",
			},
			{
				Address:     "0x31",
				Name:        "novaIdentityAttestation",
				Selectors:   []string{"0x01 attest", "0x02 verify", "0x03 revoke", "0x04 get"},
				Description: "Issuer-owned subject/claim attestations with expiry and revocation. Each attestation also creates a ProtoTypeIdentity Protocol Object owned by the issuer, queryable via nova_listProtocolObjects(0x04, owner). get() returns 7 words including po_id at the last slot.",
			},
			{
				Address:     "0x32",
				Name:        "novaSocialGraph",
				Selectors:   []string{"0x01 follow", "0x02 unfollow", "0x03 isFollowing", "0x04 trustScore", "0x05 followWithConsent"},
				Description: "Deterministic follow edges and a small trust-score primitive. 0x01 is Twitter-style open follow (no consent required). 0x05 followWithConsent requires a target-signed message keccak256('nova.social.follow:' || caller || target || chainId) verified via ecrecover; signer MUST equal target.",
			},
			{
				Address:     "0x33",
				Name:        "novaContentManifest",
				Selectors:   []string{"0x01 create", "0x02 verify", "0x03 get"},
				Description: "Root-hash manifests that compose with Phase 3 ContentRef IDs. Non-zero contentRef MUST resolve to a Phase 3 ProtoTypeContentReference (validated via CrGetContentRef); zero contentRef bypasses the check. Each create also mints a ProtoTypeContentReference PO tagged 'manifest:' in stateData. get() returns 6 words including po_id.",
			},
			{
				Address:     "0x34",
				Name:        "novaGameState",
				Selectors:   []string{"0x01 commit", "0x02 reveal", "0x03 get"},
				Description: "Turn-ordered commit/reveal state checkpoints for game/session apps. First commit on a gameID mints a ProtoTypeGameRoom Protocol Object owned by the first committer. get() returns 5 words including po_id.",
			},
			{
				Address:     "0x36",
				Name:        "novaComputeBounty",
				Selectors:   []string{"0x01 create", "0x02 submit", "0x03 verify", "0x04 get", "0x05 claim"},
				Description: "Off-chain compute bounty commitments with result/proof submissions. 0x01 create takes 4 words (specHash, expectedResult, reward, expiryBlock) and accepts msg.value as escrow (swept into 0xFF11). 0x03 verify compares submission.result against bounty.expected by default (override via second input word). 0x05 claim (owner-only) releases escrow to the verified submitter and closes the bounty. Each create also mints a ProtoTypeSubscription PO. get() returns 10 words.",
			},
		},
	}
}

// OpcodeConfig exposes the NIP-0004 Phase 12 opcode bridge metadata. Wire
// names: ethernova_opcodeConfig and nova_opcodeConfig.
func (api *EthernovaAPI) OpcodeConfig() map[string]interface{} {
	head := api.e.blockchain.CurrentBlock().Number.Uint64()
	return map[string]interface{}{
		"phase":                   12,
		"active":                  head >= ethernova.NovaOpcodeForkBlock,
		"forkBlock":               ethernova.NovaOpcodeForkBlock,
		"currentBlock":            head,
		"opcodeRange":             "0xD0-0xD8",
		"legacyDraftRangeAvoided": "0xF6-0xFE was rejected because it collides with canonical EVM opcodes like STATICCALL, REVERT, INVALID, and SELFDESTRUCT.",
		"opcodes": []NovaOpcodeInfo{
			{"0xD0", "MSEND", "0x35 selector 0x01", "Send a mailbox payload hash through the deferred queue."},
			{"0xD1", "MRECV", "0x35 selector 0x02", "Receive and dequeue the next mailbox message."},
			{"0xD2", "MPEEK", "0x35 selector 0x03", "Peek the next mailbox message without dequeuing."},
			{"0xD3", "MCOUNT", "0x35 selector 0x04", "Read unread mailbox message count."},
			{"0xD4", "CREF", "0x2B selector 0x01", "Create a compact ContentRef from stack words."},
			{"0xD5", "CVERIFY", "0x2B selector 0x03", "Verify ContentRef validity."},
			{"0xD6", "SOPEN", "0x2D selector 0x01", "Open a Domain 2 session/channel."},
			{"0xD7", "SCOMMIT", "0x2D selector 0x02", "Commit a signed session state using memory-carried signature tail."},
			{"0xD8", "SCLOSE", "0x2D selector 0x03", "Close a session using memory-carried signature tail."},
		},
	}
}
