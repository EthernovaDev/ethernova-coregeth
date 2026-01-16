# Ethernova v1.2.7 Release

## Overview
Ethernova v1.2.7 is a mandatory update that embeds the mainnet genesis snapshot and verifies it at startup. Nodes on older versions are rejected by the P2P version gate.

## Downloads
- ethernova-windows-amd64-v1.2.7.zip
- ethernova-linux-amd64-v1.2.7.tar.gz

Checksums: `checksums-sha256.txt`

## Windows
1) Extract the zip.
2) Update (no data wipe):
   `update.bat` or `update.ps1`
3) Start the node:
   `run-node.bat`

## Linux
1) Extract the tarball:
   `tar -xzf ethernova-linux-amd64-v1.2.7.tar.gz`
2) Update (no data wipe):
   `./update.sh` or `./update-1.2.7.sh`
3) Start the node:
   `./scripts/run-mainnet-node.sh`

## Notes
- If you see WRONG GENESIS, delete the datadir and re-init with `genesis-mainnet.json`.
- RPC binds to localhost by default.
