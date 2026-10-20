#!/usr/bin/env bash
# Прогон 8.1-2, фаза C (эра 2): сдвиг часов +30 сут, рестарт узлов из БД,
# выпуски участников P=3 в окнах, демередж, чекпоинты после seed_sunset,
# холодный старт наблюдателя, возврат часов.
#
# Требует: sudoers-правило на date + systemctl stop/start systemd-timesyncd
# (владелец выдал; см. отчёт). Эмуляторы сдвигаются отдельно (cmd alarm set-time).
set -euo pipefail
REPO="$(cd "$(dirname "$0")/../../.." && pwd)"
WORK="${WORK:-/tmp/tnp8/run2}"

cd "$WORK"
BIN="$WORK/bin"
BOOT=$(python3 -c "import json;print(json.load(open('fleet/fleet.json'))['bootstrap'])")
SEEDS=$(grep '^SEEDS=' labels.txt | cut -d= -f2)
WALLETS=$(grep '^WALLETS=' labels.txt | cut -d= -f2)
IFS=',' read -ra SEEDARR <<< "$SEEDS"
IFS=',' read -ra WALTARR <<< "$WALLETS"
LOG="$WORK/era2.log"; : > "$LOG"
say() { echo; echo "### $* ($(date -u +%H:%M:%S))"; }
TX() { "$BIN/tnp-testnet" "$@" -peer "$BOOT" -link-groups=false -wait 4 >> "$WORK/drv.log" 2>&1; }
STATE() { "$BIN/tnp-testnet" state -genesis net/genesis.bin -peer "$BOOT" -link-groups=false \
  -wait 3 -node-seed "drv-st-$RANDOM" "$@" 2>/dev/null; }

say "эра 2: сдвиг часов хоста +30 сут"
sudo -n systemctl stop systemd-timesyncd
REAL_NOW=$(date +%s)
SHIFTED=$(( REAL_NOW + 30*86400 ))
sudo -n date -s "@$SHIFTED"
date -u
echo "сдвинуто с $REAL_NOW на $SHIFTED (+30 сут) — возврат: sudo date -s @$REAL_NOW" | tee -a "$LOG"

say "эра 2: рестарт 48 узлов из БД (чистый consClock, демонстрация №40)"
# приложения эмуляторов: force-stop + relaunch + разблокировка PIN —
# вручную через adb (см. run2-app-notes.md), до выпусков ниже
pkill -x tnp-node || true
sleep 3
python3 - "$WORK" net/genesis.bin "$BIN" <<'EOF'
import json, subprocess, sys
w, genesis, bin_ = sys.argv[1:4]
d = f"{w}/fleet"
fleet = json.load(open(f"{d}/fleet.json"))["nodes"]
boot = json.load(open(f"{d}/fleet.json"))["bootstrap"]
for n in fleet:
    argv = [bin_ + "/tnp-node", "-genesis-file", genesis, "-node-key", n["key"],
            "-listen", f"/ip4/127.0.0.1/udp/{n['port']}/quic-v1",
            "-db", n["db"], "-link-groups=false",
            "-status-file", n["status"]]
    if n.get("wallet"):
        argv += ["-wallet", n["wallet"], "-wallet-domain", n["domain"]]
    if n["index"] > 1:
        argv += ["-peer", boot]
    lf = open(n["log"], "a")
    subprocess.Popen(argv, stdout=lf, stderr=lf)
print(f"перезапущено {len(fleet)} узлов")
EOF
sleep 45

say "эра 2: выпуски участников P=3 (окна челленджа, ±48 ч)"
for w in "${WALTARR[@]}"; do
  if TX mint -genesis net/genesis.bin -seed "$w" -key-domain wallet -period 3 -node-seed "drv-p3-$w"; then
    echo "$(date -u +%H:%M:%S) mint $w OK" >> "$LOG"
  else
    echo "$(date -u +%H:%M:%S) mint $w FAIL" >> "$LOG"
  fi
done
echo "выпусков участников: $(grep -c 'mint.*OK' "$LOG")/28"

say "эра 2: демередж и балансы после границы периода (settlePeriod)"
STATE -watch-seeds "${SEEDARR[0]},${SEEDARR[1]},${SEEDARR[2]},${SEEDARR[3]}" \
      -watch-wallets "${WALTARR[0]},${WALTARR[1]}" > "$WORK/state-era2.json"
cat "$WORK/state-era2.json"

say "эра 2: холодный старт наблюдателя (узел 49, без БД)"
"$BIN/tnp-node" -genesis-file net/genesis.bin -node-key fleet-49 \
  -listen /ip4/127.0.0.1/udp/15149/quic-v1 -peer "$BOOT" -link-groups=false \
  -status-file "$WORK/fleet/status/node49.json" > "$WORK/fleet/logs/node49.log" 2>&1 &
sleep 40
cat "$WORK/fleet/status/node49.json" 2>/dev/null || echo "статус наблюдателя не готов"

say "эра 2: возврат часов"
sudo -n date -s "@$REAL_NOW"
sudo -n systemctl start systemd-timesyncd
date -u
say "фаза C (эра 2) завершена"
