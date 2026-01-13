# VERIFY_OLD_BINARIES.md

Purpose: prove that old binaries (< v1.2.6) cannot keep the network alive after the chainId split fix at block **138,396**.

Prereqs:
- Go 1.21+ in PATH, or `C:\dev\core-geth-src\.tools\go\bin\go.exe` (scripts auto-detect).
- Repo root: `C:\dev\core-geth-src`.

Constants:
- Fork enforcement block: **138,396** (>= 138396)
- chainId/networkId enforced: **121525**
- Genesis hash (block 0): **0xc67bd6160c1439360ab14abf7414e8f07186f3bed095121df3f3b66fdc6c2183**

Checklist (PASS/FAIL):
- [ ] PASS: P2P gate rejects `CoreGeth/v1.2.5...` (old client).
- [ ] PASS: P2P gate accepts `CoreGeth/v1.2.6...` (new client).
- [ ] PASS: Post-fork block 138,396 rejects chainId **77777** (invalid signer).
- [ ] PASS: Post-fork block 138,396 accepts chainId **121525**.
- [ ] PASS: Genesis hash matches `0xc67bd6...6c2183`.
- [ ] PASS: `scripts/verify_all.ps1` or `scripts/verify_all.sh` returns PASS.

---

## A) P2P Proof (peer gating)

Goal: show that a node < v1.2.6 is rejected during handshake and cannot stay connected.

Windows:
```
powershell -ExecutionPolicy Bypass -File scripts\verify_p2p_gate.ps1
```

Linux:
```
./scripts/verify_p2p_gate.sh
```

Expected test harness output (from `go test -v`):
```
VERIFY_P2P_GATE: name="CoreGeth/v1.2.5/windows-amd64/go1.20" rejected err=peer version 1.2.5 < required 1.2.6
VERIFY_P2P_GATE: name="CoreGeth/v1.2.6/windows-amd64/go1.20" accepted
PASS: P2P version gate
```

Expected node log line (runtime handshake):
```
Rejected peer due to client version gate name=CoreGeth/v1.2.5/... err="peer version 1.2.5 < required 1.2.6"
```

---

## B) Consensus Proof (post-fork chainId rejection)

Goal: show that after block 138,396, a tx signed with chainId **77777** is invalid and cannot be mined into the canonical chain.

Windows:
```
powershell -ExecutionPolicy Bypass -File scripts\verify_chainid_gate.ps1
```

Linux:
```
./scripts/verify_chainid_gate.sh
```

Expected test harness output (from `go test -v`):
```
VERIFY_CHAINID_GATE: block=138396 old_chainId=77777 err=invalid chain id for signer: have 77777 want 121525
VERIFY_CHAINID_GATE: block=138396 new_chainId=121525 accepted_from=0x...
PASS: chainId gate at fork block
```

---

## C) One-command summary

Windows:
```
powershell -ExecutionPolicy Bypass -File scripts\verify_all.ps1
```

Linux:
```
./scripts/verify_all.sh
```

Expected final line:
```
VERIFY_ALL: PASS
```
