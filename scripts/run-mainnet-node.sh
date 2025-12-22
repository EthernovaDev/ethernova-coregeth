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
BOOTNODES="${BOOTNODES:-}"
BOOTNODES_FILE="${BOOTNODES_FILE:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bootnodes)
      BOOTNODES="${2:-}"
      shift 2
      ;;
    --bootnodes-file)
      BOOTNODES_FILE="${2:-}"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

if [[ -z "$BOOTNODES" ]]; then
  if [[ -z "$BOOTNODES_FILE" ]]; then
    BOOTNODES_FILE="$ROOT_DIR/network/bootnodes.txt"
  fi
  if [[ -f "$BOOTNODES_FILE" ]]; then
    BOOTNODES="$(grep -vE '^[[:space:]]*(#|$)' "$BOOTNODES_FILE" | tr -d '\r' | paste -sd, - || true)"
  fi
fi

mkdir -p "$DATA_DIR"

STATIC_NODES_SRC="$ROOT_DIR/network/static-nodes.json"
if [[ -f "$STATIC_NODES_SRC" ]]; then
  mkdir -p "$DATA_DIR/geth"
  STATIC_NODES_DST="$DATA_DIR/geth/static-nodes.json"
  cp "$STATIC_NODES_SRC" "$STATIC_NODES_DST"
  echo "Static nodes: $STATIC_NODES_DST"
fi

ARGS=(
  --datadir "$DATA_DIR"
  --networkid 77777
  --http --http.addr 127.0.0.1 --http.port "$HTTP_PORT" --http.api eth,net,web3,debug
  --ws --ws.addr 127.0.0.1 --ws.port "$WS_PORT" --ws.api eth,net,web3,debug
)

if [[ -n "$BOOTNODES" ]]; then
  echo "Bootnodes: $BOOTNODES"
  ARGS+=(--bootnodes "$BOOTNODES")
else
  echo "Bootnodes: (none)"
fi

if [[ "$MINE" == "1" ]]; then
  ARGS+=(--mine)
fi

"$ETHERNOVA" "${ARGS[@]}"
