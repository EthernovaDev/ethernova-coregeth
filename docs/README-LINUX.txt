Ethernova v1.2.7 - Linux Bundle

Quick start
1) Extract the tarball:
   tar -xzf ethernova-linux-amd64-v1.2.7.tar.gz
2) Start the node:
   ./scripts/run-mainnet-node.sh

Defaults
- Data dir: data-mainnet
- HTTP RPC: 127.0.0.1:8545
- WS RPC: 127.0.0.1:8546
- Logs: node.log

Update (no data wipe)
- ./update.sh or ./update-1.2.7.sh
- This replaces the ethernova binary, scripts, and upgrade genesis files only.

Systemd (optional)
- sudo ./install.sh
- sudo systemctl enable --now ethernova

Important
- Upgrade BEFORE block 138396 (chainId enforcement).
- Do NOT replace the genesis file inside your datadir.
- Bootnodes can be set in network/bootnodes.txt
