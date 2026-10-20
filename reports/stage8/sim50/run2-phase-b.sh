#!/usr/bin/env bash
# Прогон 8.1-2, фаза B (эра 1, реальное время): полный путь 50 участников.
# Аргументы: APP1 APP2 — адреса кошельков приложений эмуляторов (hex),
# зарегистрированных через UI до запуска этой фазы.
set -euo pipefail
REPO="$(cd "$(dirname "$0")/../../.." && pwd)"
WORK="${WORK:-/tmp/tnp8/run2}"
APP1="${1:?нужен адрес кошелька эмулятора 1}"
APP2="${2:?нужен адрес кошелька эмулятора 2}"

cd "$WORK"
BIN="$WORK/bin"
BOOT=$(python3 -c "import json;print(json.load(open('fleet/fleet.json'))['bootstrap'])")
SEEDS=$(grep '^SEEDS=' labels.txt | cut -d= -f2)
WALLETS=$(grep '^WALLETS=' labels.txt | cut -d= -f2)
IFS=',' read -ra SEEDARR <<< "$SEEDS"
IFS=',' read -ra WALTARR <<< "$WALLETS"
LOG="$WORK/era1.log"; : > "$LOG"
say() { echo; echo "### $* ($(date -u +%H:%M:%S))"; }
TX() { "$BIN/tnp-testnet" "$@" -peer "$BOOT" -link-groups=false -wait 4 >> "$WORK/drv.log" 2>&1; }
STATE() { "$BIN/tnp-testnet" state -genesis net/genesis.bin -peer "$BOOT" -link-groups=false \
  -wait 3 -node-seed "drv-st-$RANDOM" "$@" 2>/dev/null; }

say "регистрация 28 участников (гардианы S01,S02)"
for w in "${WALTARR[@]}"; do
  if TX register -genesis net/genesis.bin -wallet "$w" -guardians "${SEEDARR[0]},${SEEDARR[1]}" \
      -android-id "sim-$w" -node-seed "drv-r-$w"; then
    echo "$(date -u +%H:%M:%S) register $w OK" >> "$LOG"
  else
    echo "$(date -u +%H:%M:%S) register $w FAIL" >> "$LOG"
  fi
done
echo "зарегистрировано: $(grep -c 'register.*OK' "$LOG")"

# адреса участников и посевных — из состояния сети
STATE -watch-wallets "$WALLETS" -watch-seeds "$SEEDS" > "$WORK/state-after-register.json"
python3 - "$WORK/state-after-register.json" > "$WORK/addrs.sh" <<'EOF'
import json, sys
d = json.load(open(sys.argv[1]))
w = d["wallets"]
print("declare -A PADDR")
for label, v in w.items():
    print(f'PADDR[{label}]="{v["address"]}"')
EOF
source "$WORK/addrs.sh"

say "минты посевного круга P=2 (окна челленджа, ±48 ч, №39)"
for s in "${SEEDARR[@]}"; do
  if TX mint -genesis net/genesis.bin -seed "$s" -period 2 -node-seed "drv-m-$s"; then
    echo "$(date -u +%H:%M:%S) mint $s OK" >> "$LOG"
  else
    echo "$(date -u +%H:%M:%S) mint $s FAIL" >> "$LOG"
  fi
done
echo "минтов: $(grep -c 'mint.*OK' "$LOG")"

say "поручительства: 30 кандидатов × 3 посевных (слоты ≤ 10 на посевного)"
CANDS=()
for w in "${WALTARR[@]}"; do CANDS+=("${PADDR[$w]}"); done
CANDS+=("$APP1" "$APP2")
i=0
for c in "${CANDS[@]}"; do
  for k in 0 1 2; do
    s=$(( (3*i + k) % 20 ))
    if TX vouch -genesis net/genesis.bin -seed "${SEEDARR[$s]}" -candidate "$c" \
        -node-seed "drv-v-$i-$k"; then
      echo "$(date -u +%H:%M:%S) vouch ${SEEDARR[$s]} -> ${c:0:8} OK" >> "$LOG"
    else
      echo "$(date -u +%H:%M:%S) vouch ${SEEDARR[$s]} -> ${c:0:8} FAIL" >> "$LOG"
    fi
  done
  i=$((i+1))
done
echo "vouch: $(grep -c 'vouch.*OK' "$LOG")/90"

say "переводы: обычные, в приложения, отложенный с отменой"
# обычный: S01 -> P01 100 (комиссия 1, сжигается)
TX transfer -genesis net/genesis.bin -seed "${SEEDARR[0]}" -to "${PADDR[${WALTARR[0]}]}" -amount 100 -node-seed drv-tr1 \
  && echo "$(date -u +%H:%M:%S) transfer S01->P01 100 OK" >> "$LOG" || echo "transfer S01->P01 FAIL" >> "$LOG"
# в приложения: S02 -> app1/app2 по 100
TX transfer -genesis net/genesis.bin -seed "${SEEDARR[1]}" -to "$APP1" -amount 100 -node-seed drv-tr2 \
  && echo "transfer S02->app1 100 OK" >> "$LOG" || echo "transfer S02->app1 FAIL" >> "$LOG"
TX transfer -genesis net/genesis.bin -seed "${SEEDARR[1]}" -to "$APP2" -amount 100 -node-seed drv-tr3 \
  && echo "transfer S02->app2 100 OK" >> "$LOG" || echo "transfer S02->app2 FAIL" >> "$LOG"
# отложенный: S03 -> S04 3500 (> P=3000: locked 48 ч), затем отмена
# seq S03 после минта (1): pending-seq = 2, отложенный = seq 2
TX transfer -genesis net/genesis.bin -seed "${SEEDARR[2]}" -to "${PADDR[${SEEDARR[3]}]}" -amount 3500 -node-seed drv-tr4 \
  && echo "transfer S03->S04 3500 (отложенный, комиссия 35) OK" >> "$LOG" || echo "transfer отложенный FAIL" >> "$LOG"
TX cancel -genesis net/genesis.bin -seed "${SEEDARR[2]}" -pending-seq 2 -node-seed drv-tr5 \
  && echo "cancel отложенного (возврат 3500) OK" >> "$LOG" || echo "cancel FAIL" >> "$LOG"
# концентрация баланса для демереджа эры 2: S04 -> S03 2000 (S03 = 5000 > floor 3000)
TX transfer -genesis net/genesis.bin -seed "${SEEDARR[3]}" -to "${PADDR[${SEEDARR[2]}]}" -amount 2000 -node-seed drv-tr6 \
  && echo "transfer S04->S03 2000 (демередж-кандидат 5000) OK" >> "$LOG" || echo "transfer S04->S03 FAIL" >> "$LOG"

say "миграция: P01 propose (гардианы S01,S02) + cancel"
TX migrate -mode propose -new "${WALTARR[1]}" -old "${PADDR[${WALTARR[0]}]}" \
   -guardians "${SEEDARR[0]},${SEEDARR[1]}" -nonce 424242 -genesis net/genesis.bin -node-seed drv-mig1 \
   && echo "migrate propose OK" >> "$LOG" || echo "migrate propose FAIL" >> "$LOG"
# propose занял seq 2 у P01 (register = seq 1); cancel старым ключом
TX migrate -mode cancel -old "${WALTARR[0]}" -new "${PADDR[${WALTARR[1]}]}" -propose-seq 2 \
   -genesis net/genesis.bin -node-seed drv-mig2 \
   && echo "migrate cancel OK" >> "$LOG" || echo "migrate cancel FAIL" >> "$LOG"

say "состояние после эры 1 (драйвер)"
STATE -watch-seeds "${SEEDARR[0]},${SEEDARR[1]},${SEEDARR[2]},${SEEDARR[3]}" \
      -watch-wallets "${WALTARR[0]},${WALTARR[1]}" > "$WORK/state-era1.json"
cat "$WORK/state-era1.json"
say "фаза B (эра 1) завершена"
