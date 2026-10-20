#!/usr/bin/env bash
# Эмуляторный прогон закрытой тестовой сети (этап 8, задача 4).
# Полный путь нескольких узлов на genesis-пакете утилиты cmd/tnp-testnet:
# выпуск в окне челленджа (позитив/негатив), эквивокация, отложенный перевод
# с отменой, комиссия, поручительства, evidence, миграция, холодный старт.
#
# Требования: эмулятор x86_64 (API 35/36) с adb; бинарники tnp-node и
# tnp-testnet (linux/amd64, CGO_ENABLED=0) и genesis.bin в /data/local/tmp/tnp.
# Часы эмулятора сдвигаются cmd alarm set-time (все узлы в эмуляторе делят
# одни часы; после каждого сдвига узлы перезапускаются — TLS-сертификаты
# libp2p генерируются на старте и просрочены сдвигом назад).
set -u
ADB="${ADB:-adb}"
DIR=/data/local/tmp/tnp
PEER_A="/ip4/127.0.0.1/udp/4001/quic-v1/p2p/QmaTWNtyLm8R4bzSgXWr887tdmDTXx7Xb25HeL4W4VzoPC"
V1=0deb1039805d90679d7799fb9a72eec1ecea1b00ecb91ad622a815ca40d4fef5
V2=c30983ac21869edf9e203b96d283539e29e4167be19306f096b12c6ded64fdfc
V3=c31bb485f2ccbe0c2677979f1da3b40e5847bb06ff822f8acafb95a184a382ee

# автосинхронизация часов отключается: иначе время соскакивает к хосту
# во время усиленного PoW (десятки секунд) и окно челленджа закрывается
$ADB shell "settings put global auto_time 0" >/dev/null
$ADB shell "settings put global auto_time_zone 0" >/dev/null

say() { echo; echo "### $*"; }
status() { $ADB shell cat $DIR/$1.status.json 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin); print('$1'.upper(), 'journal:', d['journal_len'], 'conflicts:', d['conflicts'], 'peers:', d['peers'], 'root:', d['state_root'][:12])"; }
setclock() { $ADB shell "cmd alarm set-time $1" >/dev/null; sleep 2; $ADB shell date; }
start_nodes() {
  $ADB shell "pkill -x tnp-node" 2>/dev/null; sleep 1
  nohup $ADB shell "cd $DIR && ./tnp-node -genesis-file genesis.bin -node-key testnet-a -listen /ip4/127.0.0.1/udp/4001/quic-v1 -db $DIR/a.db -quiet -status-file $DIR/a.status.json" </dev/null >/tmp/tnp-a.log 2>&1 &
  sleep 5
  nohup $ADB shell "cd $DIR && ./tnp-node -genesis-file genesis.bin -node-key testnet-b -listen /ip4/127.0.0.1/udp/4002/quic-v1 -peer $PEER_A -db $DIR/b.db -quiet -status-file $DIR/b.status.json" </dev/null >/tmp/tnp-b.log 2>&1 &
  sleep 6
  status a; status b
}
tx() { $ADB shell "cd $DIR && timeout 200 ./tnp-testnet $*" 2>/dev/null; }

say "ЭРА 1 (2025-12-08): окно челленджа V3/период 1 открыто"
setclock 1765195200000
start_nodes
tx mint -genesis genesis.bin -seed V3 -period 1 -peer $PEER_A -node-seed drv1 -wait 8
tx mint -genesis genesis.bin -seed V3 -period 2 -peer $PEER_A -node-seed drv2 -wait 5   # негатив: окно закрыто

say "ЭРА 2 (2026-03-08): окно V3/период 4; эквивокация"
setclock 1772990400000
start_nodes
tx mint -genesis genesis.bin -seed V3 -period 4 -peer $PEER_A -node-seed drv3 -no-submit -save $DIR/tx-a.bin -wait 2
tx mint -genesis genesis.bin -seed V3 -period 4 -peer $PEER_A -node-seed drv4 -no-submit -save $DIR/tx-b.bin -wait 2
tx publish -genesis genesis.bin -tx $DIR/tx-a.bin -peer $PEER_A -node-seed drv5 -wait 8
tx publish -genesis genesis.bin -tx $DIR/tx-b.bin -peer $PEER_A -node-seed drv6 -wait 8   # конфликтная ветка

say "ЭРА 3 (2026-05-08, период 7): переводы"
setclock 1778147600000
start_nodes
tx transfer -genesis genesis.bin -seed V3 -to $V1 -amount 3500 -peer $PEER_A -node-seed drv7 -wait 8   # > P=3000: отложенный
tx cancel -genesis genesis.bin -seed V3 -pending-seq 3 -peer $PEER_A -node-seed drv8 -wait 8
tx transfer -genesis genesis.bin -seed V3 -to $V2 -amount 100 -peer $PEER_A -node-seed drv9 -wait 8    # обычный + комиссия

say "ЭРА 4 (реальное время +1 ч): регистрация, поручительства, evidence, миграция"
NOW_MS=$(($(date +%s)*1000+3600000))
setclock $NOW_MS
start_nodes
W1=$(tx register -genesis genesis.bin -wallet W1 -guardians $V2,$V3 -android-id testnet-w1 -peer $PEER_A -node-seed drv12 -wait 8 | grep -o '[0-9a-f]\{64\}')
W2=$(tx register -genesis genesis.bin -wallet W2 -android-id testnet-w2 -peer $PEER_A -node-seed drv19 -wait 8 | grep -o '[0-9a-f]\{64\}')
echo "W1=$W1 W2=$W2"
tx vouch -genesis genesis.bin -seed V3 -candidate $W1 -peer $PEER_A -node-seed drv13 -wait 8
tx vouch -genesis genesis.bin -seed V1 -candidate $W1 -peer $PEER_A -node-seed drv14 -wait 5   # негатив: 0 выпусков
tx evidence -genesis genesis.bin -seed V2 -txa $DIR/tx-a.bin -txb $DIR/tx-b.bin -peer $PEER_A -node-seed drv15 -wait 8
tx vouch -genesis genesis.bin -seed V3 -candidate $W1 -peer $PEER_A -node-seed drv16 -wait 5   # негатив: flagged
tx migrate -mode propose -new W2 -old $W1 -guardians V2,V3 -nonce 424242 -genesis genesis.bin -peer $PEER_A -node-seed drv20 -wait 8
tx migrate -mode cancel -old W1 -new $W2 -propose-seq 2 -genesis genesis.bin -peer $PEER_A -node-seed drv22 -wait 8
status a; status b

say "ХОЛОДНЫЙ СТАРТ: свежий узел C без БД"
nohup $ADB shell "cd $DIR && ./tnp-node -genesis-file genesis.bin -node-key testnet-c -listen /ip4/127.0.0.1/udp/4003/quic-v1 -peer $PEER_A -quiet -status-file $DIR/c.status.json" </dev/null >/tmp/tnp-c.log 2>&1 &
sleep 14
status c
say "ИТОГ"
status a; status b; status c
