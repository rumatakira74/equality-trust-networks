#!/usr/bin/env bash
# Прогон 8.1-1: флот 48 headless-узлов без эмуляторов, эпоха 90 с (~35 мин).
# Связность mesh, throughput, финальность, чекпоинты, память (RSS),
# потеря/восстановление связей (kill 5 узлов + рестарт из БД).
#
# Артефакты: WORK (по умолчанию /tmp/tnp8/run1); в репозиторий копируются
# выборочно (metrics.csv, events.csv, fleet.json, журнал супервизора,
# сводка, genesis/manifest, выводы драйверов).
set -euo pipefail
REPO="$(cd "$(dirname "$0")/../../.." && pwd)"
WORK="${WORK:-/tmp/tnp8/run1}"
DUR="${DUR:-2100}"            # длительность флота, сек (35 мин)
EPOCH="${EPOCH:-90}"          # эпоха, сек (1–2 мин по промпту)
NODES="${NODES:-48}"
NET="${NET:-tnp-sim50-run1}"
BASEPORT="${BASEPORT:-14100}"
KILLAT="${KILLAT:-600}"       # момент потери связей (T0+10 мин)
RESTARTAT="${RESTARTAT:-900}" # момент восстановления (T0+15 мин)
KILLNODES="${KILLNODES:-5,15,25,35,45}"

say() { echo; echo "### $*"; }
ts() { date -u +%H:%M:%S; }

# --- шаг 0: чистота среды ---------------------------------------------------
say "шаг 0: проверка среды ($(ts))"
if pgrep -x tnp-node >/dev/null; then echo "ОСТАЛИСЬ tnp-node:"; pgrep -ax tnp-node; exit 1; fi
if pgrep -x tnp-testnet >/dev/null; then echo "ОСТАЛИСЬ tnp-testnet:"; pgrep -ax tnp-testnet; exit 1; fi
if pgrep -x qemu-system-x86_64 >/dev/null || pgrep -f "qemu-system" >/dev/null 2>&1; then
  echo "остались qemu-процессы:"; pgrep -af qemu-system | head; exit 1
fi
pkill -f GradleDaemon 2>/dev/null || true
free -h | head -2
mkdir -p "$WORK"
cd "$WORK"
BIN="$WORK/bin"
mkdir -p "$BIN"
(cd "$REPO" && CGO_ENABLED=0 go build -o "$BIN/tnp-node" ./cmd/tnp-node)
(cd "$REPO" && CGO_ENABLED=0 go build -o "$BIN/tnp-testnet" ./cmd/tnp-testnet)
echo "бинарники собраны"

# --- шаг 1: genesis (метки посевного круга — окна челленджа P=2 открыты) ----
say "шаг 1: поиск меток и генезис ($(ts))"
GEN_TIME=$(( $(date +%s) - 91*86400 ))
SEEDS=$("$BIN/tnp-testnet" findlabels -network-id "$NET" -genesis-time "$GEN_TIME" \
  -kind seed -count 20 -period 2 -band-hours 24 | cut -d' ' -f1 | paste -sd,)
echo "SEEDS=$SEEDS" | tee seeds.txt
"$BIN/tnp-testnet" generate -network-id "$NET" -out "$WORK/net" \
  -genesis-time "$GEN_TIME" -seeds "$SEEDS" \
  -epoch-seconds "$EPOCH" -small-pow -anchor-epoch-zero -fresh-window 0 \
  > "$WORK/generate-output.txt" 2>&1
tail -8 "$WORK/generate-output.txt"

# --- шаг 2: флот -------------------------------------------------------------
say "шаг 2: запуск флота $NODES узлов ($(ts))"
"$BIN/tnp-testnet" fleet -genesis "$WORK/net/genesis.bin" -out "$WORK/fleet" \
  -nodes "$NODES" -seeds "$SEEDS" -base-port "$BASEPORT" -bin "$BIN/tnp-node" \
  -duration "$DUR" -poll 5 -stagger 100 -signers-last \
  > "$WORK/fleet-sup.log" 2>&1 &
SUP=$!
# спавн 48 узлов (стаггер + exec) занимает ~1–2 мин — ждём fleet.json
for i in $(seq 1 240); do
  [ -f "$WORK/fleet/fleet.json" ] && break
  sleep 1
done
BOOT=$(python3 -c "import json;print(json.load(open('$WORK/fleet/fleet.json'))['bootstrap'])")
echo "SUP=$SUP BOOT=$BOOT"
sleep 30
python3 - "$WORK/fleet" <<'EOF'
import json, sys
d = sys.argv[1]
st = []
for i in range(1, 49):
    try:
        s = json.load(open(f"{d}/status/node{i:02d}.json"))
        st.append(s["peers"])
    except Exception:
        st.append(-1)
print(f"меш через 30с: peers min={min(st)} med={sorted(st)[24]} max={max(st)}, узлов без статуса: {st.count(-1)}")
EOF

TX() { "$BIN/tnp-testnet" "$@" -peer "$BOOT" -link-groups=false -wait 4 >> "$WORK/drv.log" 2>&1; }

# --- шаг 3: минты посевного круга в окнах челленджа -------------------------
say "шаг 3: 20 минтов P=2 в окнах ($(ts))"
IFS=',' read -ra SEEDARR <<< "$SEEDS"
MINTLOG="$WORK/mints.log"; : > "$MINTLOG"
for s in "${SEEDARR[@]}"; do
  if TX mint -genesis "$WORK/net/genesis.bin" -seed "$s" -period 2 -node-seed "drv-m-$s"; then
    echo "$(ts) mint $s OK" >> "$MINTLOG"
  else
    echo "$(ts) mint $s FAIL" >> "$MINTLOG"
  fi
done
grep -c OK "$MINTLOG" | xargs echo "минтов принято:"

# финальность: последний минт должен дойти до всех 48
python3 - "$WORK/fleet" 20 120 <<'EOF' | tee "$WORK/finality-mint.txt"
import json, sys, time
d, want, timeout = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
t0 = time.time()
while time.time() - t0 < timeout:
    ok = 0
    for i in range(1, 49):
        try:
            if json.load(open(f"{d}/status/node{i:02d}.json"))["journal_len"] >= want:
                ok += 1
        except Exception:
            pass
    if ok == 48:
        print(f"финальность минта #20: все 48 узлов за {time.time()-t0:.1f}с")
        sys.exit(0)
    time.sleep(0.5)
print(f"финальность минта #20: ТАЙМАУТ {timeout}с, дожало {ok}/48")
EOF

# --- шаг 4: throughput — 40 переводов ---------------------------------------
say "шаг 4: 40 переводов ($(ts))"
ADDRS=$(python3 -c "
import json
m = json.load(open('$WORK/net/manifest.json'))
print(','.join(s['address'] for s in m['seed_circle']))")
IFS=',' read -ra A <<< "$ADDRS"
TRLOG="$WORK/transfers.log"; : > "$TRLOG"
for i in $(seq 0 39); do
  from=$(( i % 20 )); to=$(( (i + 1) % 20 ))
  if TX transfer -genesis "$WORK/net/genesis.bin" -seed "${SEEDARR[$from]}" \
      -to "${A[$to]}" -amount 50 -node-seed "drv-t$i"; then
    echo "$(ts) transfer $i ${SEEDARR[$from]}->${SEEDARR[$to]} OK" >> "$TRLOG"
  else
    echo "$(ts) transfer $i FAIL" >> "$TRLOG"
  fi
done
echo "переводов принято: $(grep -c OK "$TRLOG")/40"

# --- шаг 5: потеря связей ----------------------------------------------------
say "шаг 5: kill узлов $KILLNODES ($(ts))"
python3 - "$WORK/fleet" "$KILLNODES" <<'EOF'
import json, sys, os, signal
d, nodes = sys.argv[1], sys.argv[2]
fleet = {n["index"]: n for n in json.load(open(f"{d}/fleet.json"))["nodes"]}
for x in nodes.split(","):
    n = fleet[int(x)]
    os.kill(n["pid"], signal.SIGKILL)
    print(f"узел {n['index']} (pid {n['pid']}) убит")
EOF
sleep 60
python3 - "$WORK/fleet" <<'EOF'
import json, sys
d = sys.argv[1]
st = []
for i in range(1, 49):
    try:
        st.append(json.load(open(f"{d}/status/node{i:02d}.json"))["peers"])
    except Exception:
        st.append(-1)
print(f"меш через 60с после kill: peers min={min(st)} med={sorted(st)[24]} max={max(st)}")
EOF

# --- шаг 6: восстановление (рестарт из БД) -----------------------------------
say "шаг 6: рестарт убитых узлов ($(ts))"
python3 - "$WORK/fleet" "$KILLNODES" "$WORK/net/genesis.bin" "$BIN" <<'EOF'
import json, sys, subprocess
d, nodes, genesis, bin_ = sys.argv[1:5]
fleet = {n["index"]: n for n in json.load(open(f"{d}/fleet.json"))["nodes"]}
boot = json.load(open(f"{d}/fleet.json"))["bootstrap"]
for x in nodes.split(","):
    n = fleet[int(x)]
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
    print(f"узел {n['index']} перезапущен из {n['db']}")
EOF
sleep 90
python3 - "$WORK/fleet" "$WORK" <<'EOF'
import json, sys
d, w = sys.argv[1:3]
st, jl = [], []
for i in range(1, 49):
    try:
        s = json.load(open(f"{d}/status/node{i:02d}.json"))
        st.append(s["peers"]); jl.append(s["journal_len"])
    except Exception:
        st.append(-1); jl.append(-1)
print(f"меш после рестарта: peers min={min(st)} med={sorted(st)[24]} max={max(st)}")
print(f"журнал: min={min(jl)} med={sorted(jl)[24]} max={max(jl)}")
EOF

# --- шаг 7: догон — остаток длительности -------------------------------------
LEFT=$(( DUR - SECONDS - 60 ))
if [ "$LEFT" -gt 60 ]; then
  say "шаг 7: прогон ещё $LEFT с (чекпоинты, seed_sunset на эпохе 24) ($(ts))"
  sleep "$LEFT"
fi

# --- шаг 8: стоп и сводка ----------------------------------------------------
say "шаг 8: остановка флота ($(ts))"
kill -TERM "$SUP" 2>/dev/null || true
for i in $(seq 1 60); do kill -0 "$SUP" 2>/dev/null || break; sleep 1; done
kill -KILL "$SUP" 2>/dev/null || true
sleep 2
if pgrep -x tnp-node >/dev/null; then pgrep -ax tnp-node; pkill -x tnp-node; sleep 2; fi
echo "процессов tnp-node осталось: $(pgrep -cx tnp-node || true)"

python3 - "$WORK" "$EPOCH" <<'EOF' | tee "$WORK/summary.txt"
import csv, json, sys, statistics
w, epoch = sys.argv[1], int(sys.argv[2])
rows = list(csv.DictReader(open(f"{w}/fleet/metrics.csv")))
by = {}
for r in rows:
    by.setdefault(r["node"], []).append(r)
def col(node, name):
    out = []
    for r in by.get(node, []):
        if r[name]:
            out.append(int(r[name]))
    return out
# финальный снимок — из статус-файлов (перезапущенные узлы живут
# под новыми pid, супервизор их не видит)
snap = {}
for i in range(1, 49):
    try:
        snap[i] = json.load(open(f"{w}/fleet/status/node{i:02d}.json"))
    except Exception:
        snap[i] = None
alive = [i for i, s in snap.items() if s]
rss0, rss1 = [], []
for n in by:
    rs = col(n, "rss_kb")
    if rs:
        rss0.append(rs[0]); rss1.append(rs[-1])
print(f"узлов живых к концу (статус-файлы): {len(alive)}/48")
if rss0:
    print(f"RSS старт: min={min(rss0)} КБ med={statistics.median(rss0):.0f} max={max(rss0)}")
    print(f"RSS конец: min={min(rss1)} КБ med={statistics.median(rss1):.0f} max={max(rss1)}")
peers_end = [s["peers"] for s in alive if s]
jl_end = [s["journal_len"] for s in alive if s]
if peers_end:
    print(f"peers конец: min={min(peers_end)} med={statistics.median(peers_end):.0f} max={max(peers_end)}")
    print(f"journal_len конец: min={min(jl_end)} med={statistics.median(jl_end):.0f} max={max(jl_end)}")
roots = set(s["state_root"][:12] for s in alive if s)
print(f"state_root в конце: {len(roots)} вариант(а): {sorted(roots)[:4]}")
cps = [s["checkpoints"] for s in alive if s]
if cps:
    print(f"чекпоинтов на узле: min={min(cps)} med={statistics.median(cps):.0f} max={max(cps)}, узлов с 0: {cps.count(0)}")
ev = list(csv.DictReader(open(f"{w}/fleet/events.csv")))
fin = {}
for e in ev:
    if "финализирован" in e["event"]:
        ep = e["event"].split("эпохи ")[1].split(" ")[0]
        fin.setdefault(ep, []).append(int(e["ts"]))
if fin:
    ts0 = int(rows[0]["ts"])
    print("чекпоинты по эпохам (первая финализация, с от старта):")
    for ep in sorted(fin, key=int):
        print(f"  эпоха {ep}: +{min(fin[ep])-ts0}с, узлов отписались: {len(fin[ep])}")
spam = sum(s["spam_rejected"] for s in alive if s)
conf = sum(s["conflicts"] for s in alive if s)
print(f"spam_rejected всего: {spam}, conflicts всего: {conf}")
EOF
say "прогон 1 завершён ($(ts))"
