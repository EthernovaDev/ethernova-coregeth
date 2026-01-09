Ethernova v1.2.5 - Windows Bundle

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
- Double-click: update-1.2.5.bat (or update.bat)
- This replaces ethernova.exe, scripts, and upgrade genesis files only.

Important
- Upgrade BEFORE block 138392 (chainId switch).
- If not done yet, run scripts\apply-upgrade-mainnet.bat (fork 70000 config update).
- Do NOT replace the genesis file inside your datadir.
- Bootnodes can be set in network\bootnodes.txt
