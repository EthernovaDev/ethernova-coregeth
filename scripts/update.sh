#!/usr/bin/env bash
set -euo pipefail

RELEASE_VERSION="${RELEASE_VERSION:-v1.2.4}"
RELEASE_URL_BASE="${RELEASE_URL_BASE:-}"
GITHUB_REPO="${GITHUB_REPO:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$SCRIPT_DIR"
if [[ "$(basename "$ROOT_DIR")" == "scripts" ]]; then
  ROOT_DIR="$(cd "$ROOT_DIR/.." && pwd)"
fi

detect_repo() {
  local url repo
  if [[ -n "$GITHUB_REPO" ]]; then
    echo "$GITHUB_REPO"
    return
  fi
  if command -v git >/dev/null 2>&1; then
    if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      url="$(git -C "$ROOT_DIR" remote get-url origin 2>/dev/null || true)"
      if [[ "$url" =~ github.com[:/]+([^/]+)/([^/.]+) ]]; then
        repo="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
        echo "$repo"
        return
      fi
    fi
  fi
  echo "EthernovaDev/ethernova-coregeth"
}

REPO="$(detect_repo)"
if [[ -z "$RELEASE_URL_BASE" ]]; then
  RELEASE_URL_BASE="https://github.com/$REPO/releases/download"
fi

TARBALL="ethernova-linux-amd64-$RELEASE_VERSION.tar.gz"
TARBALL_URL="$RELEASE_URL_BASE/$RELEASE_VERSION/$TARBALL"
CHECKSUMS_URL="$RELEASE_URL_BASE/$RELEASE_VERSION/checksums-sha256.txt"

TEMP_DIR="$ROOT_DIR/update-temp"
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR"

echo "Using GitHub repo: $REPO"
echo "Downloading $TARBALL_URL"
curl -fL "$TARBALL_URL" -o "$TEMP_DIR/$TARBALL"

echo "Downloading $CHECKSUMS_URL"
if curl -fL "$CHECKSUMS_URL" -o "$TEMP_DIR/checksums-sha256.txt"; then
  expected="$(tr -d '\r' < "$TEMP_DIR/checksums-sha256.txt" | grep -m1 " $TARBALL\$" | awk '{print $1}' || true)"
  if [[ -n "$expected" ]]; then
    actual="$(sha256sum "$TEMP_DIR/$TARBALL" | awk '{print $1}')"
    if [[ "$expected" != "$actual" ]]; then
      echo "ERROR: checksum mismatch for $TARBALL" >&2
      exit 1
    fi
    echo "Checksum OK."
  else
    echo "Checksum entry not found; skipping verification."
  fi
else
  echo "Checksums file not available; skipping verification."
fi

if [[ "${DRY_RUN:-}" == "1" ]]; then
  echo "DRY_RUN=1 set. Skipping extraction and install."
  exit 0
fi

echo "Extracting update..."
tar -xzf "$TEMP_DIR/$TARBALL" -C "$TEMP_DIR"

new_bin="$(find "$TEMP_DIR" -type f -name ethernova | head -n 1 || true)"
if [[ -z "$new_bin" ]]; then
  echo "ERROR: ethernova binary not found in update package." >&2
  exit 1
fi

dest_bin_dir="$ROOT_DIR/bin"
mkdir -p "$dest_bin_dir"
cp "$new_bin" "$dest_bin_dir/ethernova"
chmod +x "$dest_bin_dir/ethernova"
echo "Updated bin/ethernova"

genesis_dir="$ROOT_DIR/genesis"
mkdir -p "$genesis_dir"
for name in genesis-mainnet.json genesis-upgrade-60000.json genesis-upgrade-70000.json; do
  src="$(find "$TEMP_DIR" -type f -name "$name" | head -n 1 || true)"
  if [[ -n "$src" ]]; then
    cp "$src" "$genesis_dir/$name"
    echo "Updated genesis/$name"
  fi
done

if command -v systemctl >/dev/null 2>&1; then
  if systemctl list-unit-files | grep -q "^ethernova.service"; then
    if [[ "$(id -u)" -eq 0 ]]; then
      systemctl restart ethernova
      echo "Restarted systemd service: ethernova"
    else
      echo "systemd service detected. Restart with: sudo systemctl restart ethernova"
    fi
  fi
fi

echo "Update complete."
