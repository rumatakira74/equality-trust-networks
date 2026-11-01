# Отчёт: этап 16 — персистентность узла приложения (№38: SQLite/SIGSYS)

Ветка: `stage16-node-persistence` (от `main`, 29596ee). Мандат:
`prompts/stage-16-node-persistence-prompt.md`. Основание: №38 QUESTIONS.md
+ находка аудита (узел приложения в памяти — баланс/история терялись при
рестарте). Этап 15 (txNow, cons.Replay strict) снял блокер №40 —
store-replay детерминирован и сходится с холодной синхронизацией.

## 1. Диагноз №38 на актуальных версиях

Обновлены до последних: `modernc.org/sqlite` v1.60.1 → **v1.61.1**,
`modernc.org/libc` v1.79.0 → **v1.79.1**.

**Вердикт: SIGSYS на x86_64-эмуляторе остался.** Эмпирика: приложение с
файловой БД (`inMemoryDb` снят) на эмуляторе API 36 (google_apis x86_64,
AVD stage12ga) падает при старте узла:

```
Fatal signal 31 (SIGSYS), code 1 (SYS_SECCOMP), syscall 6
  in tid 3788 (tnp-node-io), pid 3377
```

Tombstone: `reports/stage16/01-sigsys-tombstone-modernc.txt`
(syscall 6 = lstat на x86_64; в памяти потока — путь БД `/data/...`).
Версии в AAR подтверждены `go version -m libgojni.so`: sqlite v1.61.1,
libc v1.79.1.

**Корень (анализ исходников модулей).** Upstream-фикс «fstatat вместо
lstat» частично состоялся: `Xlstat`/`Xstat` в `ccgo_linux_amd64.go`
делегируют в `Xfstatat(AT_FDCWD, …, AT_SYMLINK_NOFOLLOW)`. Но musl-путь
`_fstatat_kstat` (тот же файл, libc v1.79.1) содержит быстрый путь
`fd == AT_FDCWD && flag == AT_SYMLINK_NOFOLLOW → raw SYS_lstat` —
«фикс» на linux/amd64 иллюзорен. На linux/**arm64** modernc безопасен:
ядро arm64 не реализует legacy-сисколлы вовсе (lstat/stat/access), весь
путь идёт через `*at`. Т.е. №38 на реальных arm64-устройствах не
воспроизводится и на старых версиях, но эмуляторная разработка/CI
(x86_64) требует решения.

Upstream-контекст (класс проблемы, не закрыт upstream для amd64):
golang/go#27797 (x/mobile SIGSYS; Bionic реализует lstat через
fstatat, app-seccomp блокирует legacy-сисколлы);
lightninglabs/wavelength#1351 и PR #1352 (тот же SIGSYS от modernc/libc
на x86_64-эмуляторе; решение — смена драйвера).

## 2. Решение по драйверу

Порядок предпочтения мандата: (1) upstream-фикс — недоступен (см. выше);
(2) смена драйвера. Кандидаты и критерии:

| Критерий | ncruces/go-sqlite3 (выбран) | mattn/go-sqlite3 |
| --- | --- | --- |
| seccomp-совместимость (главный) | SQLite → WASM через wazero (pure-Go): файловый I/O только через пакет os Go (openat/fstatat), raw legacy-сисколлы не излучаются НИ на одной архитектуре — безопасен по построению | bionic через NDK: lstat → fstatat, безопасен; но тянет cgo-рантайм в AAR |
| Размер | AAR 104,7 → 117,9 МБ (+13,2 МБ на 4 ABI, ~3,3 МБ/ABI; wazero + sqlite.wasm) | ~+1 МБ/ABI (нативный sqlite) |
| Скорость (замер, host amd64, 1000 tx журнала; `go test ./core/store/ -bench Store`) | вставка 107 мс (1,13×), чтение 8,8 мс (5,1× modernc — единовременно на рестарте, off-main) | быстрее на ARM (нативный), но |
| Риск сборки gomobile bind | нулевой: pure-Go, сборка AAR не меняется | высокий: cgo + NDK на 4 ABI, пересборка цепочки CI |

**Выбор: `ncruces/go-sqlite3` v0.36.0** (SQLite 3.45.1 → WASM,
wazero v1.12.0). Решающий критерий — seccmp-совместимость по построению
на ВСЕХ архитектурах (эмулятор x86_64 + устройства arm64); далее —
нулевой риск сборки gomobile (pure-Go, CI не меняется). Замедление
чтения журнала (5× на 1000 tx ≈ +7 мс на рестарте, off-main) для
объёмов приложения несущественно; вставка сопоставима (1,13×). Замеры
на ARM (живой телефон) — чек-лист 8-live; wazero компилирует WASM в
нативный код и на arm64, кратного замедления архитектуры нет.

Интерфейс `core/store` и схема не меняются; протокол не тронут,
`protocolVersion` не поднимался.

## 3. Персистентность узла приложения

- **Файл БД — приватный каталог**: `context.getDatabasePath("tnp-node.db")`
  (+ `mkdirs` родителя), никаких внешних хранилищ. Подтверждено в
  прогоне: `/data/data/platform.tnp.tnp_app/databases/tnp-node.db`
  (артефакт 18).
- **Весь I/O off-main**: без изменений — старт/стоп узла и все вызовы
  моста на однопоточном executor `tnp-node-io` (ANR 8.1); готовность —
  опрос `nodeStats`.
- **Флаг `inMemoryDb` удалён** (обход этапа 7 больше не нужен);
  FlutterBridge не выключает персистентность.
- **Рестарт полного узла**: журнал из БД → `cons.Replay strict`
  (пинирование на claim, этап 15) → инкрементальная сверка (якорь №41:
  первая сверка с пиром полная — corroboration, далее только новые).
  Существующий механизм `net.initLedger` — без изменений.
- **Light-узел**: `restoreLight` — своя цепочка из БД при старте
  (сортировка по seq, tip для `lightCheckOwn`); входящие НЕ
  персистентятся сознательно: неполный журнал сломал бы
  `cons.Replay strict` при смене уровня на full (12.2).
- **Дедуп txid до apply**: gossip-путь — существующий механизм 8.1
  (`indexTx` наполняет `seen` из replay-журнала, `handleTxFromPeer`
  отсеивает до cheapCheck/Argon2). Sync-путь: повторное применение
  известной tx отсеивается первой проверкой apply (C3 E_DUP, до PoW) —
  дёшево, шторма нет (тест 6).
- **Bootstrap при старте без пиров — не фатален** (правка этапа):
  раньше `net.Start` падал («ни один пир не доступен») и узел вообще
  не поднимался — рестарт без сети терял ВСЁ. Теперь узел поднимается
  (replay из БД, входящие подключения, сверка по identify), 12.3.
- **`state_root` в `StatsText`** (мост): сверка узла приложения с пиром
  видна в logcat/журнале пиров (критерий прогона).

## 4. Тесты (обязательные, мандат п.4)

| # | Тест (мандат) | Где | Результат |
| --- | --- | --- | --- |
| 1 | Рестарт из БД: store-replay = холодная синхронизация, state_root идентичен | эталон `TestStoreReplayConvergesWithColdSync` + `TestStoreReplayRestartMultiEpochVouch` (№40, этап 15, не менялись) + новый `TestSettlementReplayIdempotent` (двойной replay) | PASS |
| 2 | После жёсткого рестарта без пиров: баланс/история из локальной БД; после появления пира сверка доходит до сети | `net/persist_restart_test.go`: `TestRestartWithoutPeersThenConverge` | PASS |
| 3 | Отсутствие SIGSYS: приложение с файловой БД живёт в эмуляторе API 35/36 | эмуляторный прогон (ниже), артефакты 01 (modernc, есть SIGSYS) и 18 (ncruces, 0 SIGSYS, процесс жив) | PASS |
| 4 | Идемпотентность сеттлемента после рестарта: повторное применение журнала не двигает демередж | `TestSettlementReplayIdempotent` (два рестарта из одной БД: burn/netTime/state_root не меняются) | PASS |
| 5 | Инкрементальный якорь после рестарта: повторная сверка не пересылает весь журнал | `TestSyncAnchorIncrementalAfterRestart` (первая сверка полная — якорь на границе; новая tx у пира → continuation k=1, pos=4; повтор без изменений k=0) | PASS |
| 6 | E_DUP-шторм отсутствует | `TestNoDupStormAfterRestart` (seen наполнен из replay; повторная доставка gossip известной tx — 0 спама, журнал/корень не меняются; сверка-повтор k=0) | PASS |
| + | Light-узел: своя цепочка из БД после рестарта | `TestLightRestoreFromDB` | PASS |

Прогоны: `go test ./...` — зелёные; `go test -race -count=1 ./net/
./mobile/ ./core/store/` — зелёные (один ранний `-race ./net/` упал на
известном флаке multiproc-тестов под нагрузкой, повтор и `-count=1` —
зелёные). `flutter analyze` — чисто; `flutter test` — 43/43; сборка
модуля Kotlin (APK debug) — проходит.

## 5. Эмуляторный прогон (артефакты `reports/stage16/`)

Среда: AVD stage12ga, API 36 google_apis x86_64, Xvfb :99 + `-gpu host`
(этап 12), wipe-data; пиры песочницы — `cmd/tnp-sandbox-peer` ВНУТРИ
эмулятора (127.0.0.1:4001/4002, детерминированные peer ID как в
FlutterBridge), БД пиров — `/data/local/tmp/peer{1,2}.db`.

| Шаг | Артефакт | Результат |
| --- | --- | --- |
| Диагноз: modernc, файловая БД | 01 | SIGSYS syscall 6, процесс убит (tombstone) |
| Биометрия (enroll + эмуляция пальца), PIN, гардианы | 02–07 | отпечаток BIOMETRIC_STRONG, PIN, выбраны 2 гардиана из реестра (синхронизирован с пиром) |
| Регистрация (PoW Argon2id + биометрия) | 08–09 | accept; авто-vouch V1..V3 → ACTIVE |
| Кошелёк | 10–11 | ACTIVE, история: регистрация; узел: peers 2, журнал 10, state_root `256dca94…` — идентичен обоим пирам |
| Выпуск: экран Mint | 12 | «Period to claim: 11 (M4), Amount to mint: 0» — честно: без сдвига часов заявляемый период недостижим (см. отклонение) |
| **Жёсткий рестарт без пиров** (пиры убиты, force stop + relaunch) | 13, 15 | узел поднялся (bootstrap не фатален), **журнал 10 tx из локальной БД, state_root `256dca94…` идентичен**, кошелёк ACTIVE, история жива, peers: 0 |
| Появление пира + **повторный рестарт** | 14–17 | peers: 2, сверка: первая — полный проход (applied=10, всё известно, дёшево), **повторная — applied=0** (якорь продолжает, журнал не пересылается); state_root `256dca94…` у приложения и обоих пиров идентичен |
| Отсутствие SIGSYS (ncruces) | 18 | 0 SIGSYS в logcat за весь прогон с несколькими force-stop; новых tombstones нет; БД в приватном каталоге |
| Поручительства после рестарта | 19 | вкладка Vouching (см. FINDING про stale-опрос) |

**Отклонение от сценария (зафиксировано, не додумано):** «выпуск/перевод»
в приложении без сдвига часов невозможны: M4 требует завершённого
периода ≥ периода активации (12), перевод требует баланса (0 без
выпуска). Мандат прямо исключает сдвиг часов («Сдвиг часов не
требуется»), поэтому выпуски в журнале — посевные (периоды 1–2,
`cmd/tnp-sandbox-peer`), собственный выпуск приложения показан честным
состоянием экрана (12). Инкрементальность сверки и идентичность
state_root от этого не зависят; сходимость путей пересчёта с демереджем
покрыта тестами 1/4.

**Блокер прогона, устранён в рамках этапа:** `mintFor` в
`cmd/tnp-sandbox-peer` подавал 2000 за период 1 (устарело с этапа 12:
`ActivatedPeriod` посевного круга 0 → 1, diff 2 → 1) — все посевные
выпуски падали E_MINT_AMOUNT; исправлено на 1000 (= issuance × diff).

## 6. Изменённые файлы

| Файл | Что |
| --- | --- |
| `go.mod`, `go.sum` | modernc.org/sqlite v1.61.1, libc v1.79.1 (диагноз); затем ncruces/go-sqlite3 v0.36.0 (+go-sqlite3-wasm v6, julianday; modernc удалён) |
| `core/store/sqlite.go` | драйвер ncruces (импорт + имя «sqlite3»), комментарий-обоснование |
| `core/store/bench_test.go` | бенчмарки вставки/чтения журнала (замеры драйвера) |
| `net/node.go` | `restoreLight` (light-цепочка из БД); bootstrap при старте — не фатален (12.3, №38) |
| `net/persist_restart_test.go` | 5 тестов мандата (таблица выше) |
| `mobile/mobile.go` | `state_root` в `StatsText` (сверка узла приложения с пиром) |
| `cmd/tnp-sandbox-peer/main.go` | mintFor: сумма посевного выпуска (устарела с этапа 12) |
| `android/.../NodeController.kt` | флаг `inMemoryDb` удалён; файл БД в приватном каталоге + mkdirs; персистентность |
| `app/.../FlutterBridge.kt` | принудительный in-memory снят |
| `QUESTIONS.md` | №38 → «решено (этап 16)» с вариантом и обоснованием |
| `README.md` | 12.5: персистентность узла приложения заявлена честно (драйвер, рестарт, инкрементальная сверка) |
| `DISCLAIMER.md` | п.6 RU/EN: локальная БД узла + честные границы (удаление приложения/утрата устройства) |
| `reports/REPORT-STAGE-7.md` | чек-лист 8-live: строка lstat/SIGSYS → проверка ncruces-драйвера на живой прошивке |
| `FINDING-awaiting-offline-tx-not-persistent.txt` | вне мандата: офлайн-буфер публикации (12.4) не переживает рестарт |
| `FINDING-vouching-screen-stale-nonestatus.txt` | вне мандата: вкладка Vouching не перезапрашивает после NONODE |

Спека не менялась (12.5 уже обещал SQLite-хранилище — реализация
догоняет документацию); `protocolVersion` не поднимался; миграций нет
(живых сетей нет).

## 7. Находки вне мандата

- `FINDING-awaiting-offline-tx-not-persistent.txt` — ожидающие
  публикации Bluetooth-транзакции (12.4) теряются при рестарте
  (буфер только в памяти); кандидат в мини-этап.
- `FINDING-vouching-screen-stale-nonestatus.txt` — UI-quirk: вкладка
  Vouching после рестарта показывает «0 of 3» для ACTIVE-кошелька
  (опрос до готовности узла, ретрая нет).

## 8. CI

Ветка запушена (tip `3de013a`); прогоны CI на момент фиксации — **CI
pending** (правило CI-verify: сессия этапа статусы не ждёт). Номера
прогонов workflows (push `3de013a`):

| workflow | прогон | статус на момент фиксации |
| --- | --- | --- |
| core | 41364226339 | CI pending |
| net | 41364226347 | CI pending |
| cons | 41364226351 | CI pending |
| android | 41364226353 | CI pending |
| flutter | 41364226370 | CI pending |

## 9. Уборка

Эмулятор (`adb emu kill`), пиры в эмуляторе (pkill), Gradle-демоны
(`gradlew --stop`), Xvfb — остановлены; `pgrep` по qemu/java/tnp-node/
tnp-testnet/Xvfb/tnp-sandbox-peer — пусто. Бинарники сборок — вне
репо (/tmp/stage16, android/libs/ в .gitignore).
