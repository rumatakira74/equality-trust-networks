#!/bin/bash
set -u
cd /tmp/tnp8/run2
BIN=bin
BOOT=$(python3 -c "import json;print(json.load(open('fleet/fleet.json'))['bootstrap'])")
WALLETS=$(grep '^WALLETS=' labels.txt | cut -d= -f2)
IFS=',' read -ra WALTARR <<< "$WALLETS"
LOG=era2.log
for w in "${WALTARR[@]}"; do
  if "$BIN/tnp-testnet" mint -genesis net/genesis.bin -seed "$w" -key-domain wallet -period 3 -peer "$BOOT" -link-groups=false -wait 4 -node-seed "drv-p3-$w" >> drv.log 2>&1; then
    echo "$(date -u +%H:%M:%S) mint $w OK" >> "$LOG"
  else
    echo "$(date -u +%H:%M:%S) mint $w FAIL" >> "$LOG"
  fi
done
echo "выпусков участников: $(grep -c 'mint.*OK' "$LOG")/28 (эта фаза)"
