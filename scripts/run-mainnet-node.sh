#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ETHERNOVA="$ROOT_DIR/bin/ethernova"
if [[ ! -x "$ETHERNOVA" ]]; then
  ETHERNOVA="$ROOT_DIR/ethernova"
fi
if [[ ! -x "$ETHERNOVA" ]]; then
  echo "ethernova not found (expected bin/ethernova or root)." >&2
  exit 1
fi

DATA_DIR="${DATA_DIR:-$ROOT_DIR/data-mainnet}"
HTTP_PORT="${HTTP_PORT:-8545}"
WS_PORT="${WS_PORT:-8546}"
MINE="${MINE:-0}"

mkdir -p "$DATA_DIR"

ARGS=(
  --datadir "$DATA_DIR"
  --networkid 77777
  --http --http.addr 127.0.0.1 --http.port "$HTTP_PORT" --http.api eth,net,web3,debug
  --ws --ws.addr 127.0.0.1 --ws.port "$WS_PORT" --ws.api eth,net,web3,debug
)

if [[ "$MINE" == "1" ]]; then
  ARGS+=(--mine)
fi

"$ETHERNOVA" "${ARGS[@]}"
