Ethernova v1.2.4 - Windows Bundle

Quick start
1) Extract the zip to a folder.
2) (One-time) Update mainnet config:
   - Run: scripts\apply-upgrade-mainnet.bat
   - This updates the stored chain config in-place (no chain reset).
3) Start the node:
   - Double-click: run-node.bat

Defaults
- Data dir: data-mainnet
- HTTP RPC: 127.0.0.1:8545
- WS RPC: 127.0.0.1:8546
- Logs: node.log

Update (no data wipe)
- Double-click: update.bat
- This replaces ethernova.exe and genesis upgrade files only.

Important
- Upgrade BEFORE block 70000.
- Do NOT replace the genesis file inside your datadir.
- Bootnodes can be set in network\bootnodes.txt
