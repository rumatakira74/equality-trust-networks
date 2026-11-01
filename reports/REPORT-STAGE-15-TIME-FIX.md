# Отчёт этапа 15 — фикс времени (txNow = timestamp_claim)

Ветка: `stage15-time-fix` (от `main` `232c698`, fast-forward до `7e75d17` выполнен).
Мандат: `prompts/stage-15-time-fix-prompt.md`. Спека: v9.13 RU+ENG.
Основание: `reports/stage12-testnet/FINDING-replay-live-divergence.txt` (live/replay-расхождение
mint у границы периода: живая валидация — часы узла на момент apply, replay — пинирование на
`timestamp_claim`) и решения владельца от 01.11.2026: одна tx — один детерминированный вердикт,
время валидации — claim самой tx. Это первый этап по правилу CI-verify: статусы CI в сессии
не ждём.

## 1. txNow в ядре (задача 1)

**Что.** Время валидации всех часозависимых правил — txNow = `tx.TimestampClaim`; часы узла ядро
не читает вообще.

**Где.** `core/ledger/ledger.go` (тип `txTime{now, epoch, period}`, `epochAt`/`periodAt` —
арифметически от `genesis_time`; `epochSecsOrDefault`), `core/ledger/validate.go` (`Apply`:
`l.tt` заполняется от claim; все `apply*` читают `l.tt`), `core/ledger/modules.go`,
`core/ledger/settle.go`, `core/ledger/iface.go` (интерфейс `EpochSource` и `FixedClock` удалены —
отдельного «режима часов» больше не существует), `core/present/present.go` (возраст — от
`State.NetTime` = max(txNow)).

**Как.** `Apply` вычисляет `txNow = tx.TimestampClaim`, `epoch = epochAt(txNow)`,
`period = periodAt(txNow)` и валидирует против состояния, осевшего к txNow. Конструктор
`ledger.New(gp, opts)` больше не принимает часы. Аттестация (срок листа по claim, этап 14) уже
была на claim — теперь это единая точка времени всех правил.

### Таблица правил, переведённых на txNow

| Правило | Старое (до v9.13) | Новое (v9.13) |
| --- | --- | --- |
| M2 (приостановка `flagged`) | `period < MintSuspendedUntilPeriod`, period от часов узла | тот же код, `period = periodAt(claim)` |
| M4 (завершённые периоды) | `mp.Period >= period(часы узла)` → `E_MINT_PERIOD` | `mp.Period >= periodAt(claim)` → `E_MINT_PERIOD`; сборка в N / публикация в N+1 — один вердикт |
| M5 (окно накопления) | diff от claim-независимого last_claimed, окно от часов | diff от `periodAt(claim)`; квота `quota_reset` — ровно один период |
| M7 (окно челленджа) | окно сверялось с `Clock.NetTime()` | окно сверяется с `txNow = claim` (слот — п. 4) |
| T3 (мораторий) | `now(часы) − RegisteredAt < D` → `E_MORATORIUM` | `txNow − RegisteredAt < D` |
| T6/X3 (отложенный перевод) | `Deadline = now(часы) + 48 ч` | `Deadline = txNow + 48 ч`; исполнение — от claim принятых tx |
| V2 (возраст поручителя) | `AgeDays(now(часы))` | `AgeDays(txNow)` |
| V8 (эпоха vouch) | `vp.Epoch != Epoch(часы/чекпоинты)` → `E_VOUCH_EPOCH` | `vp.Epoch != epochAt(claim)` |
| `ActivatedPeriod` | `Clock.Period()` на момент apply | `periodAt(claim)` vouch-а активации |
| recurring-сгорание (T4/T8, redeem) | граница P+1 от `Clock.Period()` | граница P+1 от `periodAt(claim)` |
| evidence (приостановка) | `period + N` от часов | `periodAt(claim) + N` |
| Гардианы: offer/смена/миграция (сроки, возраст, wards) | `Clock.NetTime()/Epoch()` | `txNow`/`epochAt(txNow)` |
| Модули (pool дедлайн, redeem-слот, contribution-периоды) | `Clock.Epoch()/Period()/NetTime()` | `epochAt/periodAt(txNow)`, `txNow` |
| Сеттлемент (`syncTime` от часов) | `Clock.NetTime()/Epoch()/Period()` | `settleTo(txNow)` — см. п. 3 |

## 2. C9 — слой допуска сети (задача 2)

**Что.** Допуск ±30 мин убран из ledger-валидации (строка C9 `E_TIME` удалена из
`applySigPowTime`); проверяется в `net` при приёме/распространении.

**Где.** `net/gossip.go` — `c9Admit` (допуск `timestamp_tolerance_seconds` от часов узла
`n.Clock`), вызывается из `cheapCheck` (валидатор gossip → ретрансляция, `handleTxFromPeer` →
приём) и из `SubmitTx` (локальная публикация: отказ с понятной диагностикой, код `E_TIME`).
`net/node.go` — часы `n.Clock` строятся для обоих уровней узла.

**Как.** Replay/сверка (`SyncJournalFrom`, `ImportJournal`, рестарт из БД) C9 не применяют —
вердикт ядра полностью детерминирован от txNow. Sandbox/тестовые сети — допуск управляем:
проверка не применяется (короткие `period_seconds` и предсборка tx прогонов не ломаются).
Диагностика: `timestamp_claim %d вне допуска C9 (±%d с от часов узла %d)`.

## 3. Сеттлемент от max(txNow), монотонно (задача 3)

**Что.** Демередж, сгорание квот, границы периодов/эпох и сроки отложенных операций (переводы D,
миграции, смены гардианов, повышения P) оседают от max(txNow) по журналу принятых tx.

**Где.** `core/ledger/ledger.go` — `settleTo(now)`: монотонно продвигает `Epoch`/`CurrentPeriod`/
`NetTime` (только вперёд), `flushPending(NetTime)` идемпотентен (флаги `Executed`/`Effective`).
`core/ledger/validate.go` — `Apply`: сеттлемент к txNow выполняется ДО валидации (правила
проверяются против состояния, осевшего к моменту claim), но подтверждается только принятием:
отклонённая tx откатывает сеттлемент снапшотом (`State.Clone`, `core/ledger/state.go`) — иначе
узлы, видевшие отклонённые tx, расходились бы с узлами и replay, знающими только журнал.
`net/node.go` — `settleNow`/`settleLoop` (доведение до часов узла, PENDING №2 этапа 11.2)
удалены: оно расходило бы live с replay; зрелые отложенные операции применяет ближайшая
принятая tx с достаточным claim.

**Следствия.** Перемешивание порядка подачи tx не откатывает сеттлемент; повторные применения
идемпотентны (E_DUP + флаги); рестарт (replay журнала) воспроизводит то же состояние. Тесты:
`TestSettlementMonotoneFromMaxTxNow`, `TestAcceptedTxOrderDemurrageConvergence` (все
перестановки блоков кошельков дают одно итоговое состояние), `TestSettleMaturesOnNextAcceptedTx`,
`TestSettleAfterRestartMaturedOnTrigger`, `TestSettleAfterRestartImmature`,
`TestStoreReplayConvergesWithColdSync` (A=B=C).

## 4. Слот челленджа (задача 4)

**Что.** Слот `period_start + H(address ‖ period ‖ checkpoint_hash) mod period_seconds`
(7.6) вычисляется от claim: окно ±`mint_challenge_window_hours` (поле 51) сверяется с
`txNow = claim` — заявляемый период обязан быть достижим в claim-окне. Масштабирование слота
`period_seconds` (поле 61) сохранено (`mintChallengeSlot`).

**Где.** `core/ledger/validate.go` — `CheckMintChallengeWindow(..., l.tt.now)`.

**Тест (блокер 2 прогона 12-testnet).** `TestMintChallengeWindowShortPeriodsSeedReachability`
(`core/ledger/mint_window_test.go`): genesis в прошлом (95 сут), `period_seconds` = 120 с —
слот периода P строго внутри периода P (масштабирование полем 61); текущий период достижим с
окном ±48 ч; первый заявляемый период посевного круга (квота периода 1 у genesis, quota_reset)
недостижим с окном ±48 ч и достижим с окном 2400 ч (поле 51 — достижимость первых периодов);
claim вне окна слота отклоняется (окно — от claim, не от часов узла).

## 5. cons.Replay strict + диагностика (задача 5)

**Что.** Replay пинирует на claim тем же механизмом, что live, — отдельного режима часов нет:
`pinClock` удалён, `Replay` применяет журнал через `ledger.Apply` (ядро само валидирует от
txNow). Strict — без послаблений: любая отклонённая tx — ошибка пересчёта; конфликтная ветка
(не победитель) фиксируется как Conflict и остаётся в журнале хранения (INV-1c).

**Где.** `cons/replay.go`: `Replay` без подмены часов; `replayDiag` — диагностика расхождения:
какая tx (txid, тип, seq, claim), период валидации и эпоха (пинированы на claim — как в live),
вердикт, для mint — заявленный период. `net/cons.go`, `net/sync.go`, `net/node.go` —
`applyWithClock`/`ReplayClock` удалены, сверка и ImportJournal применяют `Apply` напрямую.

**Регресс-тест (материал находки).** `cons/replay_strict_test.go`:
`TestReplayStrictReproducesLiveRoot` — журнал сети с короткими периодами (120 с, genesis в
прошлом), квоты периода 1 с claim в периоде 2 (паттерн «сборка в N, публикация в N+1»,
`FINDING-replay-live-divergence`): live-узел и свежий пересчёт `cons.Replay`-ом дают
одинаковый `state_root` — расхождение находки закрыто по построению. Живой журнал сети A
прогона 12-testnet (`/tmp/stage12-testnet`, вне репо) собран под `protocolVersion` 1 — по
правилу «миграции нет, тестовые сети пересоздаются» материалом регресс-теста служит
реконструкция паттерна; сами данные прогона сохранены для сверки владельцем.
`TestReplayStrictRejectsCorruptedJournal` — strict: двойной выпуск периода в журнале — ошибка
с диагностикой (`cons: replay: tx … (type=3 seq=1 claim=…): период валидации 29, эпоха 0,
вердикт E_MINT_DOUBLE; заявленный период mint 1`).

## 6. protocolVersion+1 (задача 6)

`protocolVersion` поднят с 1 до 2: семантика валидации/сеттлемента изменилась (txNow, C9 — слой
сети, сеттлемент от max(txNow)). Где: `core/params/params.go` (TestGenesis),
`cmd/tnp-testnet/main.go`, `cmd/tnp-testnet/configure.go`, векторы (`protocol_version: 2`).
Мобильное приложение читает версию из genesis (мост `mobile`), хардкода нет. Миграция не
пишется: живых сетей нет, тестовые пересоздаются (genesis v1 узлами v2 отклоняется, `E_VER`).

## 7. Спека v9.13 RU+ENG (задача 7)

Синхронно, перечень правок в шапке (история версий титула):

- `docs/KNOWLEDGE.md`, `docs/KNOWLEDGE-ENG.md` — версия документа 9.13, запись «Версия 9.13»
  (полное содержание правок).
- `docs/KNOWLEDGE-4.md`, `docs/KNOWLEDGE-ENG-4.md` — 9.2 (время валидации txNow; C9 — слой
  допуска сети; периоды арифметически от genesis, поле 61; сеттлемент монотонно от max(txNow)),
  10.1 (момент валидации M4/M5 — claim, не публикация), 10.3 (демередж на границах от max(txNow)).
- `docs/KNOWLEDGE-3.md`, `docs/KNOWLEDGE-ENG-3.md` — 7.6: слот от claim, формула с
  `mod period_seconds`, поля 61/51, достижимость первых заявляемых периодов посевного круга.
- `docs/KNOWLEDGE-2.md`, `docs/KNOWLEDGE-ENG-2.md` — 6.2: std-семантика «потронутых» единиц
  (при `fresh_transfer_window = 0` первое списание переводит все когорты в Liquid; сгорание
  сбросом действует только на нетронутые когорты — расхождение №6 `REPORT-STAGE-12.md`,
  подтверждение `REPORT-STAGE-12-TESTNET.md`); recurring-граница от txNow.
- `spec/00-overview.md` — время (txNow, C9 — слой сети, сеттлемент от max(txNow)),
  `protocol_version = 2`, код `E_TIME` — уровень сети.
- `spec/02-genesis-manifest.md` — раздел 2.2.1: поле 61 `period_seconds` (0 — конвенция 30 сут;
  масштабирует часы узла, replay и слот), поле 51 `mint_challenge_window_hours` (достижимость
  первых периодов при коротких `period_seconds`), `protocol_version` = 2.
- `spec/03-validation.md` — обозначения (txNow, `periodAt`/`epochAt`), C9 — не проверка ядра,
  M4/M5 (период валидации от claim; вопрос №24 решён), C11 (окно от claim, поля 61/51),
  T3/T6/X3/N2 — от txNow, 3.10 — сеттлемент от max(txNow) по журналу.
- `spec/04-invariants.md` — INV-3: детерминизм полный (replay без режима часов, strict),
  INV-10 от txNow. `spec/06-wallet-statemachine.md` — возраст от txNow.

## Тесты (обязательные)

Пять регресс-тестов фикса:

1. `TestTxNowVerdictsFromClaim` (`core/ledger/txnow_regression_test.go`) — вердикты всех
   часозависимых правил от claim: M2, M4 (кейс находки: сборка в периоде N, публикация в N+1 —
   `E_MINT_PERIOD` на двух независимых ядрах с одинаковым вердиктом), T3, V2,
   `ActivatedPeriod` = `periodAt(claim)`, evidence-сроки, модули (дедлайн пула от
   `epochAt(claim)`), recurring-граница P+1.
2. `TestSettlementMonotoneFromMaxTxNow` — демередж монотонно оседает от claim принятых tx;
   повторные применения (E_DUP) не меняют состояние; tx с ранним claim не откатывает сеттлемент;
   рестарт (пересчёт журнала) воспроизводит состояние.
3. `TestMintChallengeWindowShortPeriodsSeedReachability` — слот от claim внутри claim-окна при
   коротких `period_seconds`, генезис в прошлом (кейс прогона 12-testnet, блокер 2).
4. `TestReplayStrictReproducesLiveRoot` + `TestReplayStrictRejectsCorruptedJournal`
   (`cons/replay_strict_test.go`) — живой журнал (сборка N, публикация N+1) даёт state_root
   живых узлов; strict-диагностика повреждённого журнала.
5. `TestC9AdmissionLayer` + `TestC9SandboxManaged` (`net/c9_test.go`) — claim вне ±30 мин не
   принимается и не ретранслируется сетью (cheapCheck/handleTxFromPeer/SubmitTx); ledger и
   replay валидируют без C9 (та же tx применяется ядром); sandbox — допуск управляем.

Тест порядка AcceptedTxs с демереджем: `TestAcceptedTxOrderDemurrageConvergence` — подача tx
журнала во всех перестановках блоков кошельков даёт одно итоговое состояние (сеттлемент к
max(txNow) оседает на первой принятой tx, до любых эффектов).

Перегон под protocolVersion 2 и txNow: векторы `vectors/*.json` (20 файлов: `protocol_version`
2; в network.params добавлены `genesis_time`, `period_seconds`, `epoch_seconds`, согласованные
с pre_state так, что `periodAt/epochAt(claim)` = периоду/эпохе pre_state; ожидания
`registered_at`/`execute_not_before` — от claim), harness `vector_harness_test.go` (часы
удалены, читает genesis_time/61/51). Существующие тесты переведены на claim/txNow без ослабления
assertions: e2e, property (мутация «вне допуска времени» удалена — C9 больше не правило ядра,
покрыт тестом 5 в net), settle/settle_now (зрелость — от claim принятой tx), modules,
recurring, fingerprints, update_guardians, measure, attest, cons, present, net (рестарты,
сверка, интеграция), sim (периоды арифметические), cmd/tnp-testnet.

## Предкоммитная проверка

- `gofmt -l .` — пусто; `go vet ./...` — чисто.
- `go test ./...` — зелёные (core/ledger 27–30 с, cons, net 45–50 с, sim ~110–130 с,
  cmd/tnp-testnet, core/present, mobile, остальные).
- `go test -race` по задетым пакетам: core/ledger, cons, net — зелёные; sim, cmd/tnp-testnet,
  core/present, mobile — зелёные (лог прогона в сессии).
- `flutter analyze` — No issues found; `flutter test` — 43 passed.
- Kotlin: `:app:compileDebugKotlin` — BUILD SUCCESSFUL; Gradle-демоны остановлены, `pgrep`
  чист (gradle/tnp-node/tnp-testnet/qemu — нет процессов).

## Изменённые файлы

Ядро: `core/ledger/{ledger,validate,settle,state,iface,modules}.go`, `core/present/present.go`,
`core/params/params.go`. Консенсус/сеть: `cons/{replay,time}.go`, `net/{node,gossip,sync,clock}.go`.
Утилиты: `cmd/tnp-testnet/{main,configure}.go`, `sim/{sim,node,scenarios}.go`. Тесты: 14 файлов
core/ledger, cons/cons_test, cons/replay_strict_test, core/present, net (10 файлов), sim,
cmd/tnp-testnet. Векторы: 20 JSON. Спека: docs/KNOWLEDGE{,-ENG}{,-2,-3,-4}.md,
spec/{00,02,03,04,06}-*.md.

## CI

Правило CI-verify: пуш ветки — без ожидания статусов в сессии. Прогоны workflows на ветке
`stage15-time-fix` (номера на момент пуша):

| Workflow | Прогон | Статус на момент пуша |
| --- | --- | --- |
| core (core.yml) | см. сводку ниже | CI pending |
| net (net.yml) | см. сводку ниже | CI pending |
| cons (cons.yml) | см. сводку ниже | CI pending |
| flutter (flutter.yml) | см. сводку ниже | CI pending |
| android (android.yml) | см. сводку ниже | CI pending |

Точные номера прогонов (run numbers) — в сводке сессии после пуша; владелец запускает
CI-verify по `prompts/ci-verify-prompt.md`, подставив имя ветки `stage15-time-fix`.

## Сводка

- tip ветки `stage15-time-fix`: см. `git log -1` после финального коммита.
- protocolVersion: 2.
- Правила на txNow: M2, M4, M5, M7, T3, T6/X3, V2, V8, ActivatedPeriod, recurring-сгорание
  (T4/T8, redeem), evidence, гардианы (offer/смена/миграция), модули (pool/redeem/
  contribution), сеттлемент (демередж, сгорание, границы, сроки).
- CI: все 5 workflows — CI pending (номера прогонов в сводке).
- Отчёт: `reports/REPORT-STAGE-15-TIME-FIX.md`.
