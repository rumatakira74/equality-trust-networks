# Прогон 8.1-2: управление приложениями эмуляторов (adb)

Оба эмулятора: Pixel_9 (API 36) и tnp_api35 (API 35), headless
(`-no-window -no-audio -gpu swiftshader_indirect`). APK — release
(163,5 МБ); если `adb push` в Android/data недоступен на
production-образе (находка этапа 8) — ставить debug-APK и класть genesis
через `run-as` (зафиксировать в отчёте, какой путь сработал).

## Подготовка (до первого запуска приложения)

```bash
ADB=~/Android/Sdk/platform-tools/adb
EM=~/Android/Sdk/emulator/emulator
$EM -avd Pixel_9 -no-window -no-audio -no-boot-anim -gpu swiftshader_indirect &
$EM -avd tnp_api35 -no-window -no-audio -no-boot-anim -gpu swiftshader_indirect &
$ADB devices                       # ждать device
$ADB -s <ID> install -r app/build/app/outputs/flutter-apk/app-release.apk
# genesis + пиры (путь — по находке этапа 8):
$ADB -s <ID> shell mkdir -p /storage/emulated/0/Android/data/platform.tnp.tnp_app/files/networks
$ADB -s <ID> push net/genesis.bin /storage/emulated/0/Android/data/platform.tnp.tnp_app/files/networks/
$ADB -s <ID> push peers.txt /storage/emulated/0/Android/data/platform.tnp.tnp_app/files/networks/
# если push в Android/data запрещён (API 36 production):
$ADB -s <ID> install -r app-debug.apk
$ADB -s <ID> push net/genesis.bin /data/local/tmp/genesis.bin
$ADB -s <ID> shell "run-as platform.tnp.tnp_app sh -c 'mkdir -p files/networks && cp /data/local/tmp/genesis.bin files/networks/'"
$ADB -s <ID> shell "run-as platform.tnp.tnp_app sh -c 'cp /data/local/tmp/peers.txt files/networks/'"
```

peers.txt — multiaddr узлов флота с адресом 10.0.2.2 (slirp-алиас хоста):
`/ip4/10.0.2.2/udp/<порт>/quic-v1/p2p/<peer-id>` (узлы 1–3).

## Онбординг (UI, adb input)

1. Энролл отпечатка: Настройки → Безопасность → отпечаток (или
   `adb -s <ID> shell settings ...`; на эмуляторе — Fingerprint →
   Touch sensor, палец = `adb -s <ID> emu finger touch 1`).
2. Запуск приложения → биометрия → PIN (запомнить!) → тревожный PIN
   (запомнить!) → гардианы (2 из реестра — посевные кошельки) →
   регистрация (усиленный PoW, секунды на small-pow).
3. Адрес своего кошелька: экран настроек/кошелька (uiautomator dump +
   grep hex) — передать в фазу B для vouch.

## Эра 1

- Регистрация через UI (см. выше) — до запуска фазы B.
- Тревожный PIN: заблокировать (выйти), войти тревожным → режим-приманка
  (скриншот), выйти, войти обычным → реальный кошелёк (скриншот).

## Эра 2 (после сдвига часов +30 сут и рестарта узлов)

- Эмуляторы: `adb -s <ID> shell cmd alarm set-time <ms+30сут>` (auto_time=0
  на время сдвига; вернуть после прогона).
- Приложение: force-stop + relaunch + разблокировка PIN (свежий узел,
  холодный синк журнала из толпы).
- Выпуск (mint): экран кошелька → выпуск → ожидание accept в окне
  (№39: слот кошелька приложения случаен — вероятен E_POW_HARD,
  зафиксировать результат).
- Перевод: экран перевода → адрес получателя (участник P02) → сумма 50
  (баланс приложения: 100 из эры 1) → отправка.
