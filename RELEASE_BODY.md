# Ethernova CoreGeth v1.2.6 (Mandatory)

MANDATORY UPDATE: v1.2.6 enforces the chainId split fix at block **138,396** and disconnects peers running older binaries.

Highlights:
- Fork enforcement block: **138,396** (>= 138,396)
- chainId/networkId enforced: **121525**
- Old binaries are rejected by the peer gate and cannot mine valid blocks after the fork.

Upgrade steps:
- Windows: use `dist/update-windows.ps1` to stop the service, back up the old binary, replace it, and restart.
- Linux: use `dist/update-linux.sh` to stop systemd, back up the old binary, replace it, daemon-reload if needed, and restart.

Verification:
- `eth_chainId` should return `0x1dab5`
- `net_version` should return `121525`
- `eth_getBlockByNumber(0).hash` should return `0xc67bd6160c1439360ab14abf7414e8f07186f3bed095121df3f3b66fdc6c2183`
- Proof pack: https://github.com/EthernovaDev/ethernova-coregeth/blob/v1.2.6/VERIFY_OLD_BINARIES.md

Artifacts:
- `ethernova-v1.2.6-windows-amd64.exe`
- `ethernova-v1.2.6-linux-amd64`
- `SHA256SUMS.txt`
- `update-windows.ps1`
- `update-linux.sh`
