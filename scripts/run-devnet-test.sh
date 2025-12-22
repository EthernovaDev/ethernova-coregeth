#!/usr/bin/env bash
set -euo pipefail

KEEP_RUNNING=0
if [[ "${1:-}" == "--keep-running" ]]; then
  KEEP_RUNNING=1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ETHERNOVA="$ROOT_DIR/bin/ethernova"
if [[ ! -x "$ETHERNOVA" ]]; then
  ETHERNOVA="$ROOT_DIR/ethernova"
fi
if [[ ! -x "$ETHERNOVA" ]]; then
  echo "ethernova not found (expected bin/ethernova or root)." >&2
  exit 1
fi

EVMCHECK="$ROOT_DIR/bin/evmcheck"
if [[ ! -x "$EVMCHECK" ]]; then
  EVMCHECK="$ROOT_DIR/evmcheck"
fi
if [[ ! -x "$EVMCHECK" ]]; then
  echo "evmcheck not found (expected bin/evmcheck or root)." >&2
  exit 1
fi

GENESIS="$ROOT_DIR/genesis/genesis-devnet-fork20.json"
if [[ ! -f "$GENESIS" ]]; then
  GENESIS="$ROOT_DIR/genesis-devnet-fork20.json"
fi
if [[ ! -f "$GENESIS" ]]; then
  echo "genesis-devnet-fork20.json not found." >&2
  exit 1
fi

KEY_FILE="$ROOT_DIR/genesis/devnet-testkey.txt"
if [[ ! -f "$KEY_FILE" ]]; then
  KEY_FILE="$ROOT_DIR/devnet-testkey.txt"
fi
if [[ ! -f "$KEY_FILE" ]]; then
  echo "devnet-testkey.txt not found." >&2
  exit 1
fi

PRIV_KEY="$(grep -E '^PRIVATE_KEY=' "$KEY_FILE" | cut -d= -f2-)"
DEV_ADDR="$(grep -E '^ADDRESS=' "$KEY_FILE" | cut -d= -f2-)"
CHAINID="$(grep -E '^CHAINID=' "$KEY_FILE" | cut -d= -f2-)"

if [[ -z "$PRIV_KEY" || -z "$DEV_ADDR" || -z "$CHAINID" ]]; then
  echo "devnet-testkey.txt missing PRIVATE_KEY, ADDRESS, or CHAINID." >&2
  exit 1
fi

DATA_DIR="$ROOT_DIR/data-devnet"
RPC_URL="http://127.0.0.1:8545"
FORK_BLOCK=20
LOG_PATH="$DATA_DIR/devnet.log"

mkdir -p "$DATA_DIR"
if [[ ! -d "$DATA_DIR/geth" ]]; then
  echo "Initializing devnet datadir..."
  "$ETHERNOVA" --datadir "$DATA_DIR" init "$GENESIS" >/dev/null
fi

echo "Starting devnet node..."
"$ETHERNOVA" \
  --datadir "$DATA_DIR" \
  --http --http.addr 127.0.0.1 --http.port 8545 --http.api eth,net,web3,debug \
  --ws --ws.addr 127.0.0.1 --ws.port 8546 --ws.api eth,net,web3,debug \
  --nodiscover --maxpeers 0 \
  --networkid "$CHAINID" \
  --mine --miner.etherbase "$DEV_ADDR" \
  >"$LOG_PATH" 2>&1 &

NODE_PID=$!

cleanup() {
  if [[ "$KEEP_RUNNING" -eq 0 ]]; then
    echo "Stopping devnet node..."
    kill "$NODE_PID" >/dev/null 2>&1 || true
  else
    echo "Devnet left running. Logs: $LOG_PATH"
  fi
}
trap cleanup EXIT

get_block_number() {
  local resp hex
  resp="$(curl -s -X POST "$RPC_URL" -H 'Content-Type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}')"
  hex="$(echo "$resp" | sed -n 's/.*"result":"\(0x[0-9a-fA-F]*\)".*/\1/p')"
  if [[ -n "$hex" ]]; then
    echo $((16#${hex#0x}))
  else
    echo ""
  fi
}

echo "Waiting for RPC..."
deadline=$((SECONDS+180))
bn=""
while [[ $SECONDS -lt $deadline ]]; do
  bn="$(get_block_number || true)"
  if [[ -n "$bn" ]]; then
    break
  fi
  sleep 1
done

if [[ -z "$bn" ]]; then
  echo "RPC did not become ready in time. Check $LOG_PATH." >&2
  exit 1
fi

echo "Mining until block >= $FORK_BLOCK..."
while true; do
  sleep 1
  bn="$(get_block_number || true)"
  if [[ -n "$bn" && "$bn" -ge "$FORK_BLOCK" ]]; then
    break
  fi
done

echo "Running evmcheck..."
set +e
"$EVMCHECK" --rpc "$RPC_URL" --pk "$PRIV_KEY" --chainid "$CHAINID" --forkblock "$FORK_BLOCK"
EVMCHECK_EXIT=$?
set -e

exit "$EVMCHECK_EXIT"
