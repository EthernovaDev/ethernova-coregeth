#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

go_bin="$(command -v go || true)"
if [[ -z "$go_bin" && -x "$repo_root/.tools/go/bin/go" ]]; then
  export GOROOT="$repo_root/.tools/go"
  export PATH="$GOROOT/bin:$PATH"
  go_bin="$GOROOT/bin/go"
fi

if [[ -z "$go_bin" ]]; then
  printf '%s\n' "FAIL: go not found. Install Go 1.21+ or place it at .tools/go."
  exit 1
fi

printf '%s\n' "== ChainId gate verification (post-fork) =="
output="$("$go_bin" test ./core/types -run TestVerifyChainIDGatePostFork -v 2>&1)"
printf '%s\n' "$output"

reject_line="$(printf '%s\n' "$output" | grep -F 'VERIFY_CHAINID_GATE: block=138396 old_chainId=77777' || true)"
accept_line="$(printf '%s\n' "$output" | grep -F 'VERIFY_CHAINID_GATE: block=138396 new_chainId=121525' || true)"

if [[ -n "$reject_line" ]] && printf '%s\n' "$reject_line" | grep -Fq 'invalid chain id for signer' \
  && [[ -n "$accept_line" ]] && printf '%s\n' "$accept_line" | grep -Fq 'accepted_from='; then
  printf '%s\n' "PASS: chainId gate at fork block"
  exit 0
fi

printf '%s\n' "FAIL: chainId gate at fork block"
exit 1
