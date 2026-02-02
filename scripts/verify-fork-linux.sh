#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="${1:-http://127.0.0.1:8545}"

rpc() {
  local method="$1"
  local params="$2"
  curl -s -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"${method}\",\"params\":${params}}" \
    "$ENDPOINT"
}

echo "== Ethernova Fork Verify =="
echo "endpoint=${ENDPOINT}"

client=$(rpc web3_clientVersion "[]")
echo "clientVersion=${client}"
if ! echo "${client}" | grep -q "Ethernova/v1.2.8"; then
  echo "FAIL: unexpected clientVersion" >&2
  exit 1
fi

blocknum=$(rpc eth_blockNumber "[]")
echo "blockNumber=${blocknum}"
if ! echo "${blocknum}" | grep -q "\"result\""; then
  echo "FAIL: unexpected blockNumber response" >&2
  exit 1
fi

addr="0x0000000000000000000000000000000000000001"
code="0x600160021b00"
call="{\"from\":\"0x0000000000000000000000000000000000000000\",\"to\":\"${addr}\",\"gas\":\"0x2dc6c0\",\"data\":\"0x\"}"
stateOverrides="{\"${addr}\":{\"code\":\"${code}\"}}"

trace_shl() {
  local label="$1"
  local blockhex="$2"
  local expect_invalid="$3"
  local config="{\"tracer\":\"callTracer\",\"stateOverrides\":${stateOverrides},\"blockOverrides\":{\"number\":\"${blockhex}\"}}"
  local resp
  resp=$(rpc debug_traceCall "[${call},\"latest\",${config}]")
  echo "${label}: ${resp}"
  if [[ "${expect_invalid}" == "true" ]]; then
    if ! echo "${resp}" | grep -qi "invalid opcode"; then
      echo "FAIL: ${label} expected invalid opcode" >&2
      exit 1
    fi
  else
    if echo "${resp}" | grep -qi "\"error\""; then
      echo "FAIL: ${label} expected success" >&2
      exit 1
    fi
  fi
  echo "${label}: OK"
}

trace_shl "pre-fork (block 104999)" "0x19a27" "true"
trace_shl "post-fork (block 105000)" "0x19a28" "false"

echo "OK: fork verification passed."
