Ethernova v1.2.6 - Linux Bundle

Quick start
1) Extract the tarball:
   tar -xzf ethernova-linux-amd64-v1.2.6.tar.gz
2) (One-time) Update mainnet config:
   ./scripts/apply-upgrade-mainnet.sh
3) Start the node:
   ./scripts/run-mainnet-node.sh

Defaults
- Data dir: data-mainnet
- HTTP RPC: 127.0.0.1:8545
- WS RPC: 127.0.0.1:8546
- Logs: node.log

Update (no data wipe)
- ./update-linux.sh
- This replaces the ethernova binary, scripts, and upgrade genesis files only.

Systemd (optional)
- sudo ./install.sh
- sudo systemctl enable --now ethernova

Important
- Upgrade BEFORE block 138396 (chainId enforcement).
- If not done yet, run ./scripts/apply-upgrade-mainnet.sh (fork 70000 config update).
- Do NOT replace the genesis file inside your datadir.
- Bootnodes can be set in network/bootnodes.txt
