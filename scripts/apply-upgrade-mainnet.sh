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

GENESIS_UPGRADE="$ROOT_DIR/genesis/genesis-upgrade-60000.json"
if [[ ! -f "$GENESIS_UPGRADE" ]]; then
  GENESIS_UPGRADE="$ROOT_DIR/genesis-upgrade-60000.json"
fi
if [[ ! -f "$GENESIS_UPGRADE" ]]; then
  echo "genesis-upgrade-60000.json not found." >&2
  exit 1
fi

DATA_DIR="${DATA_DIR:-$ROOT_DIR/data-mainnet}"

echo "Applying Fork60000 config upgrade..."
echo "NOTE: Do NOT replace the genesis file in your datadir."
echo "      This command updates the stored chain config in-place."

"$ETHERNOVA" --datadir "$DATA_DIR" init "$GENESIS_UPGRADE"
