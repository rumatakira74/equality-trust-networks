#!/usr/bin/env bash
# Прогон 8.1-2, фаза A: генезис полного прогона (эпоха 300 с) + флот 48 узлов.
# 20 посевных (S*, окна P=2) + 28 участников (P*, окна P=3 — эра 2) + 2 эмулятора.
set -euo pipefail
REPO="$(cd "$(dirname "$0")/../../.." && pwd)"
WORK="${WORK:-/tmp/tnp8/run2}"
EPOCH="${EPOCH:-300}"
NODES="${NODES:-48}"
NET="${NET:-tnp-sim50-run2}"
BASEPORT="${BASEPORT:-15100}"
DUR="${DUR:-9000}"   # супервизор живёт всю эру 1 и эру 2 (перезапуск узлов — вручную)

say() { echo; echo "### $*"; }

if pgrep -x tnp-node >/dev/null; then echo "ОСТАЛИСЬ tnp-node"; pgrep -ax tnp-node; exit 1; fi
if pgrep -x tnp-testnet >/dev/null; then echo "ОСТАЛИСЬ tnp-testnet"; pgrep -ax tnp-testnet; exit 1; fi
pkill -f GradleDaemon 2>/dev/null || true
mkdir -p "$WORK"; cd "$WORK"
BIN="$WORK/bin"; mkdir -p "$BIN"
(cd "$REPO" && CGO_ENABLED=0 go build -o "$BIN/tnp-node" ./cmd/tnp-node)
(cd "$REPO" && CGO_ENABLED=0 go build -o "$BIN/tnp-testnet" ./cmd/tnp-testnet)

say "фаза A: метки и генезис"
GEN_TIME=$(( $(date +%s) - 91*86400 ))
SEEDS=$("$BIN/tnp-testnet" findlabels -network-id "$NET" -genesis-time "$GEN_TIME" \
  -kind seed -count 20 -period 2 -band-hours 24 | cut -d' ' -f1 | paste -sd,)
WALLETS=$("$BIN/tnp-testnet" findlabels -network-id "$NET" -genesis-time "$GEN_TIME" \
  -kind wallet -count 28 -period 3 -band-hours 24 | cut -d' ' -f1 | paste -sd,)
echo "SEEDS=$SEEDS" | tee labels.txt
echo "WALLETS=$WALLETS" | tee -a labels.txt
"$BIN/tnp-testnet" generate -network-id "$NET" -out "$WORK/net" \
  -genesis-time "$GEN_TIME" -seeds "$SEEDS" \
  -epoch-seconds "$EPOCH" -small-pow -anchor-epoch-zero -fresh-window 0 \
  > "$WORK/generate-output.txt" 2>&1
grep -E "genesis_time|epoch_seconds|epoch_zero|seed_sunset|manifest_hash" "$WORK/generate-output.txt"

say "фаза A: флот $NODES узлов"
"$BIN/tnp-testnet" fleet -genesis "$WORK/net/genesis.bin" -out "$WORK/fleet" \
  -nodes "$NODES" -seeds "$SEEDS" -wallets "$WALLETS" -base-port "$BASEPORT" \
  -bin "$BIN/tnp-node" -duration "$DUR" -poll 5 -stagger 100 -signers-last \
  > "$WORK/fleet-sup.log" 2>&1 &
echo "SUP=$!" | tee "$WORK/sup.pid"
for i in $(seq 1 240); do [ -f "$WORK/fleet/fleet.json" ] && break; sleep 1; done
python3 - "$WORK/fleet" <<'EOF'
import json, sys
d = sys.argv[1]
boot = json.load(open(f"{d}/fleet.json"))["bootstrap"]
print("BOOT=" + boot)
st = []
for i in range(1, 49):
    try:
        st.append(json.load(open(f"{d}/status/node{i:02d}.json"))["peers"])
    except Exception:
        st.append(-1)
print(f"меш: peers min={min(st)} med={sorted(st)[24]} max={max(st)}")
EOF
say "фаза A готова: genesis $WORK/net/genesis.bin, флот поднят"
