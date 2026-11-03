# REPORT-AUDIT-MINT-FIX — blind-аудит темы mint-fix (фикс окна челленджа M7, вариант A)

Ветка аудита: `audit-mint-fix` от tip `mint-fix` (`2b4f0ac`); база ветки — `d5ab36e`
(merge-base с `main`; `main` ушёл вперёд на 2 коммита pipeline-run, к теме не относятся).
Дата: 03.11.2026. Сессия: `audit-mint-fix-01`.
Код tnp — только чтение (правки только в `reports/`); репо eq (`/home/projects/anti-farm-simulation`)
— только чтение, источник описания дефекта: `reports/REPORT-A9-SIMULATION.md` (main, `19ce812`).
Живых сетей нет. CI в сессии не ждался — CI pending.

Хост аудита: i7-1065G7 (8 ядер), Go 1.27.3, 15,14 ГиБ ОЗУ, GOMAXPROCS=6.

**Вердикт (кратко): «готова с замечаниями».** Блокеров нет: фикс ядра верен, свойство окна
доказано и подтверждено независимой проверкой по реальному коду, тесты (a)–(h) на месте и
зелёные, полный набор и `-race` чистые, дифф — только предмет темы. Замечания — P2/P3:
устаревшие `spec/` (C11 описывает дофиксную семантику), отсутствие восстановления по
`accrual_window` в mobile-мосте, инструменты живого прогона (`slots`/`findlabels`) считают
достижимость по старой формуле, и формулировка отчёта «поглощающее состояние исчезает»
нуждается в уточнении (замер ниже).

---

## 0. Слепая схема

Пункты 1–5 выполнены по коду до открытия `reports/REPORT-MINT-FIX.md` (отчёт исполнителя
и `FINDING-MINT-FIX-0*.txt` прочитаны только на пункте 6). Независимая проверка — программа
`reports/audit-mint-fix/verify-main.go.txt` (модуль в `/tmp` с `replace` на рабочий клон,
код tnp не менялся): режимы `prop`, `e2e`, `replay`, `retro`, `recover`, `stats`.

---

## 1. Пункт 1 — фикс по коду (не по отчёту)

**Проверка M7** — `core/ledger/validate.go:164-167`:

```go
if tx.Type == wire.Mint && !l.Opts.SkipMintChallengeWindow {
    if !l.CheckMintChallengeWindow(tx.Wallet, l.tt.period, l.tt.now) {
        return reject("E_POW_HARD")
    }
}
```

- Якорь — **текущий период выпуска**: `l.tt.period` (`validate.go:165`), а не заявленный
  `mp.Period`. Подтверждено: в `applySigPowTime` переменная `mp` больше не извлекается
  (в `main` было `mp := tx.Payload.(*wire.MintPayload)` — `git show main:core/ledger/validate.go:156-157`).
- `l.tt` вычисляется в `Apply` до валидации: `l.tt = txTime{now: txNow, epoch: epochAt(txNow),
  period: periodAt(txNow)}` — `validate.go:229`, где `txNow = tx.TimestampClaim`
  (`validate.go:228`). `periodAt` — арифметически от genesis (`core/ledger/ledger.go:169-174`),
  та же конвенция, что `net.RealClock.Period()` (`net/clock.go:46-52`).
- **Механизм txNow не сломан**: `applySigPowTime` вызывается только из `applyChecked`
  (`validate.go:256` — конфликтная ветка, `validate.go:272` — обычная), а `applyChecked`
  вызывается только из `Apply` после установки `l.tt` (`validate.go:229-242`). Часы узла
  ядро не читает; отклонение откатывает сеттлемент (`validate.go:243-245`).
- Слот не менялся: `mintChallengeSlot` — `period_start + hashThree(addr, period) mod period_secs`
  (`validate.go:206-210`), `hashThree` — `SHA-256(addr‖period BE64)` (`core/ledger/ledger.go:130-145`).
  Чекпоинт-хеша в формуле нет (PENDING, п. 7).

**Вывод: подтверждено по коду.** Проверка окна вынесена в `CheckMintChallengeWindow`
(`validate.go:186-193`), окно `±win` вокруг слота, `win = MintChallengeWindowHours × 3600`
(48 ч по умолчанию — `core/params/params.go:163`, `core/profiles/profiles.go:87`).

---

## 2. Пункт 2 — математика фикса

### 2.1. Свойство окна (доказательство)

Слот лежит внутри текущего периода C: `slot ∈ [start, start+P)`, `P = period_secs` (720 ч),
`W = 48 ч`. Пересечение `[slot−W, slot+W] ∩ [start, start+P)`:

| положение слота | границы пересечения | длина |
|---|---|---|
| `slot ∈ [start, start+W)` | `[start, slot+W]` | `W + (slot−start) ≥ W` |
| `slot ∈ [start+W, start+P−W]` | `[slot−W, slot+W]` | `2W ≥ W` |
| `slot ∈ (start+P−W, start+P)` | `[slot−W, start+P]` | `W + (start+P−slot) > W` |

Минимум достигается на границах периода и равен ровно `W = 48 ч`; условие `P ≥ W` выполнено
для всех реальных конфигураций (30 сут; тестовые сети — поле 61, окно задаётся полем 51 так,
чтобы накрывать период). Значит, **у каждого адреса в каждом периоде есть достижимый интервал
подачи ≥ 48 ч внутри текущего периода**.

### 2.2. Перебор по реальному коду

`verify prop` — 200 адресов × 12 периодов = 2400 случаев, слот сверяется с ядром
(`CheckMintChallengeWindow` открыт в самом слоте и закрыт за ±48 ч + 1 с):

```
PROP OK: адресов=200 периодов=12 случаев=2400
  min пересечение окна с периодом = 173048 с (48.07 ч)
  среднее пересечение = 92.62 ч; доля периода (случайный момент) = 0.1286
PROP OK: короткие периоды (120 с, 1 ч, 6 ч) — окно накрывает период целиком
```

### 2.3. Сквозные вердикты (`verify e2e`, реальный `ledger.Apply`)

```
E2E OK: достижимость — mint принят в каждом из периодов 2..10 (9 случаев)
E2E OK: вне окна (слот ±49 ч в том же периоде) — reject/E_POW_HARD (16 случаев)
E2E OK: накопление diff=3 в окне — accept, когорты=3000 (M6: 1000×3)
E2E OK: осознанный пропуск 7 периодов — reject/E_ACCRUAL_WINDOW (M5 не изменён)
E2E OK: вердикт по окну не зависит от заявленного периода: map[claim=3:true claim=4:true claim=5:true]
E2E OK: песочница (SkipMintChallengeWindow) — вне окна accept
E2E OK: tx с protocolVersion=2 -> reject/E_VER (версия сети 3)
```

### 2.4. Поглощение: что именно устранено (уточнение)

Замер (`verify stats`, 5000 адресов, период 720 ч, окно ±48 ч):

| | до фикса | после фикса |
|---|---|---|
| доля периодов с непустым достижимым окном | 7,02 % | **100 %** |
| средняя достижимая часть периода | 1,72 ч (0,239 %) | 92,73 ч (12,88 %) |
| P(6 промахов подряд) при подаче в произвольный момент | 0,9857 | **0,4372** |

Читается так: фикс устраняет **структурную недостижимость** (было 14/15 слотов, при которых
период нельзя заявить никаким действием; стало — можно всегда). Но он не устраняет промах
при подаче в произвольный момент: пользователь, не знающий слота, попадает в окно в 12,9 %
случаев, и шесть промахов подряд остаются вероятными (43,7 %). Устранение требует, чтобы
приложение показывало слот (слот публично вычислим — PENDING-2 отчёта исполнителя, вне мандата).
Компенсация M6 в пределах `accrual_window` работает (diff=3 → 3000), осознанный пропуск
6+ периодов по-прежнему даёт `E_ACCRUAL_WINDOW` (M5 не тронут: `validate.go:559-568`).

Дополнительно: у протокола есть путь восстановления после длинного пропуска — заявка
`last + accrual_window` вместо последнего завершённого периода (`verify recover`):

```
RECOVER: claim=11 (последний завершённый) -> reject/E_ACCRUAL_WINDOW
RECOVER: claim=last+6=7 (в том же окне) -> accept/OK, last_claimed=7
```

То есть «поглощающее состояние» в протоколе не абсолютно даже при пропуске 10 периодов;
но mobile-мост этот путь не предлагает (см. AUDIT-MINT-FIX-02).

---

## 3. Пункт 3 — protocolVersion-bump

- Версия поднята **2 → 3** в genesis-дефолте: `core/params/params.go:210` (`TestGenesis` —
  дефолт всех тестов и песочниц), `cmd/tnp-testnet/main.go:393` (`generate`),
  `cmd/tnp-testnet/configure.go:158` (`configure`).
- C2 в ядре: `core/ledger/validate.go:64-67` (`tx.ProtocolVersion != l.State.Net.ProtocolVersion
  → E_VER`); в сети — `net/gossip.go:335-336` (`cheapCheck`, `errForeignVersion`).
- Векторы: 24 вхождения `protocol_version` в 20 файлах `vectors/*.json` → 3; дифф векторов —
  **только** эта строка (проверено `git diff main..mint-fix -- vectors/`, иных правок нет).
- Фикстуры журнала: `core/store/bench_test.go:20`, `core/store/sqlite_test.go:22` → 3.
- Зашитых «2» в Go/JSON не осталось (`grep -rn "ProtocolVersion: *2\|protocol_version.*2"` —
  пусто). Android версию не хардкодит: `genesis.protocolVersion()`
  (`android/.../wallet/WalletFlow.kt:69,96`).
- Сознательно не подняты фикстуры `core/wire/proto_conformance_test.go:254,429,462,474`
  (версия 1 в тестах бинарного кодирования, к сети не привязаны) — отклонение обосновано
  в отчёте исполнителя §3, принимается.
- **Живых сетей нет** — миграция не нужна. Подтверждение: в репо нет production-genesis
  (`find . -name "*genesis*.json"` пусто), `cmd/tnp-node` строит песочницу
  (`prof.Sandbox = true`, `cmd/tnp-node/main.go:302-320`), тестовые сети пересоздаются
  (тот же аргумент, что при v9.13 → 2: `core/params/params.go:203-210`).

**Вывод: подтверждено.** Единственный остаток — документация `spec/` (AUDIT-MINT-FIX-01).

---

## 4. Пункт 4 — единообразие M7 на всех путях

| путь | точка применения | окно от txNow |
|---|---|---|
| headless-ядро | `validate.go:165` | да (`l.tt.period`, `l.tt.now`) |
| net gossip | `net/node.go:618` → `Ledger.Apply` | да (тот же `Apply`) |
| net sync | `net/sync.go:388` → `Ledger.Apply` | да (тот же `Apply`) |
| replay | `cons/replay.go:114-126` → `l.Apply` | да (тот же `Apply`) |
| mobile-мост | `mobile/query.go:937` → `net/query.go:216-223` (`CheckMintChallengeWindow`) | да (текущий период живых часов) |

- Сетевой узел создаёт ядро **без** обхода окна: `net/node.go:264`
  (`ledger.New(gp, ledger.Options{})`), комментарий обновлён (`net/node.go:255-262`).
  Обхода через sync-подачу нет: sync применяет tx тем же `Apply` (`net/sync.go:388`),
  отдельного ядра с `Skip` в сети не создаётся.
- `SkipMintChallengeWindow: true` остался только в `sim/node.go:112`, `sim/sim.go:206`
  (симулятор) и тестовых harness'ах (`cons/cons_test.go:53`, `cons/replay_strict_test.go:65`,
  `cmd/tnp-testnet/configure_stage12_2_test.go:50`, `core/ledger/vector_harness_test.go:265`,
  `core/ledger/txnow_regression_test.go:202,310`, `core/ledger/mint_fix_test.go` через
  `newMfixEnv(t, true)`). Сетевых путей среди них нет.
- Песочница гасит окно легитимным оверрайдом профиля: `mobile/mobile.go:304`
  (`MintChallengeWindowHours = 0`), `cmd/tnp-node/main.go:318` (то же, `prof.Sandbox = true`).
- **Независимая проверка replay**: `verify replay` — журнал из 4 выпусков, применённый live и
  через `cons.Replay`, даёт одинаковый дайджест состояния; отклонённая по окну tx состояние
  не меняет:

```
REPLAY OK: live == cons.Replay по дайджесту (4 tx); вне окна — E_POW_HARD, состояние не меняется
```

**Остаточное свойство (не регресс темы, AUDIT-MINT-FIX-05).** Ядро валидирует от
`timestamp_claim`, а C9 (±30 мин) — слой сети, и при сверке журнала он **не применяется**
(`net/sync.go:357-359`; задокументировано: `docs/KNOWLEDGE-4.md:44`,
`reports/REPORT-STAGE-15-TIME-FIX.md:57`). Поэтому заявленное время подачи выбирает
отправитель, и окно M7 не является барьером живости для tx, пришедшей sync'ом (`verify retro`):

```
RETRO: NetTime=1726555402 (период 10), mint с claim=1708279034 (период 3): accept/OK
```

Свойство прежнее (v9.13), тема его не меняла и не ухудшала: сама проверка окна на sync
выполняется одинаково. Но пункт мандата «обход через sync-подачу закрыт» верен только в узком
смысле — закрыт обход *проверки окна*, а не выбор claim-времени.

---

## 5. Пункт 5 — тесты и гигиена

### 5.1. Полный локальный набор (GOMAXPROCS=6)

| прогон | команда | итог |
|---|---|---|
| формат | `gofmt -l .` | пусто (весь репозиторий) |
| vet | `go vet ./...` | exit 0, вывод пуст |
| тесты | `go test ./...` | exit 0, 1:31.95, max RSS 625 МБ |
| тесты без кэша + race | `go test -race -count=1 ./core/ledger ./net ./cons ./mobile ./cmd/tnp-testnet ./core/store` | exit 0, 4:38.45, max RSS 1,32 ГБ |
| тесты мандата | `go test -count=1 -v -run TestMintFix ./core/ledger` | 8/8 PASS, 1,31 с |

Память: доступно 5–7 ГиБ из 15,14 ГиБ на протяжении прогонов; стоп-условие (<2 ГиБ) не
достигалось. Процессы после прогонов: `pgrep -a -f "tnp-node|tnp-testnet|qemu|gradle|go test"`
— пусто.

### 5.2. Тесты мандата (a)–(h)

Все восемь существуют в `core/ledger/mint_fix_test.go` и проходят:

| # | тест | строка |
|---|---|---|
| (a) | `TestMintFixWindowIntersectionProperty` | 162 |
| (b) | `TestMintFixWindowInOut` | 217 |
| (c) | `TestMintFixAccumulationInWindow` | 249 |
| (d) | `TestMintFixAccrualWindowUnchanged` | 277 |
| (e) | `TestMintFixLiveReplayDeterminism` | 335 |
| (f) | `TestMintFixSandboxSkip` | 382 |
| (g) | `TestMintFixOldProtocolVersion` | 401 |
| (h) | `TestMintFixPreFixVector` | 418 |

Данные репродуцируемы: `crypto.DeterministicKey`, genesis фиксирован (1 700 000 000), адреса
подбираются детерминированным перебором seed'ов, PoW — малые таргеты. Тест (h) действительно
фиксирует смену семантики: для «позднего слота» дофиксная семантика давала accept
(`slotClaim ≥ end−24 ч` ⇒ `slotClaim+48 ч > txNow`), после фикса — `E_POW_HARD`.

### 5.3. Дифф и гигиена

- Дифф ветки — только предмет темы (40 файлов): ядро M7 + bump + тесты + спека 7.6 + пути
  mobile/net + векторы + отчёт/FINDINGS. Файл `prompts/pipeline-run-prompt.md` в диффе
  `main..mint-fix` выглядит удалённым, но это **не правка темы**: ветка базируется на `d5ab36e`,
  а два коммита pipeline-run появились в `main` позже; merge-base `d5ab36e`, при merge файл
  не удаляется (проверено `git merge-tree`).
- Файлов > 1 МБ в диффе нет (максимум — `cmd/tnp-testnet/main.go`, 47,8 КБ).
- Секретов нет: `git diff main..mint-fix | grep -iE "BEGIN .*PRIVATE|api[_-]?key|secret|password|token|passphrase"`
  — пусто.

---

## 6. Пункт 6 — сверка с отчётом исполнителя

Сверено по коду построчно. **Расхождений, меняющих выводы, нет**; мелкие неточности — ниже.

| заявление отчёта | вердикт | подтверждение |
|---|---|---|
| до фикса якорь — `mp.Period`, строка 157 | подтверждено | `git show main:core/ledger/validate.go:156-157` |
| после фикса — `l.tt.period`, `validate.go:165` | подтверждено | `validate.go:165` |
| `mintChallengeSlot` — `validate.go:206` | подтверждено | `validate.go:206-210` |
| `periodAt` — `ledger.go:169` | подтверждено | `ledger.go:169-174` |
| C2 — `validate.go:65`, дубль в `net/gossip.go:336` | подтверждено | `validate.go:65-66`, `net/gossip.go:335-336` |
| bump: `params.go:210`, `main.go:393`, `configure.go:158` | подтверждено | те же строки |
| векторы: 24 вхождения в 20 файлах | подтверждено | `grep -c` по `vectors/` |
| фикстуры: `bench_test.go:20`, `sqlite_test.go:22` | подтверждено | те же строки |
| `proto_conformance_test.go` (версия 1) не поднят | подтверждено, обоснование принято | `core/wire/proto_conformance_test.go:254,429,462,474` |
| net: `ledger.New(gp, ledger.Options{})`, `net/node.go:264` | подтверждено | `net/node.go:264` |
| sync/replay — `cons/replay.go:116` | подтверждено | `cons/replay.go:114-126`; плюс `net/sync.go:388` (в отчёте не назван) |
| mobile: `MintWindowJSON` передаёт текущий период, `mobile/query.go:937` | подтверждено | `mobile/query.go:918-937` |
| sim: `sim/node.go:112`, `sim/sim.go:206` | подтверждено | те же строки |
| «SkipMintChallengeWindow остался только в sim и тестовых harness'ах» | подтверждено; перечень неполон | не названы `core/ledger/txnow_regression_test.go:202,310`, `cons/replay_strict_test.go:65` |
| свойство окна: 64 адреса × 8 периодов = 512 | подтверждено | `mint_fix_test.go:162-207` |
| «короткие периоды … (24 проверки)» (§2) | **опровергнуто** | фактически 8 адресов × 5 периодов × 3 точки = 120 точечных проверок (`mint_fix_test.go:190-207`); в таблице §5 указано верно |
| тесты (a)–(h) PASS | подтверждено | прогон 5.1 |
| спека 7.6 RU/ENG хирургически, версия 9.14, история версий | подтверждено | `git diff main..mint-fix -- docs/` — один пункт в 7.6 + заголовок/история |
| `gofmt -l .` пусто, `go vet ./...` чисто | подтверждено | прогон 5.1 |
| `go test ./...` зелёные | подтверждено | прогон 5.1 (числа времени отличаются от отчёта — разная загрузка хоста) |
| `go test -race` по затронутым пакетам чисто | подтверждено | прогон 5.1 (ledger 277,6 с, net 184,6 с) |
| память «≥ 7,6 ГБ из 15,5 ГБ» | неточно | на хосте аудита 15,14 ГиБ всего, 5–7 ГиБ доступно; стоп-условие <2 ГиБ не достигалось |
| «поглощающее состояние исчезает» (§2) | **требует уточнения** | см. 2.4 и AUDIT-MINT-FIX-04 |
| «живых сетей нет, миграция не нужна» | подтверждено | п. 3 |
| FINDING-01 (sim без окна M7) | подтверждено | `sim/node.go:112`, `sim/sim.go:206`; прежнее поведение (`git show main:sim/node.go` — то же) |
| FINDING-02 (дубликат `challengeSlot`) | подтверждено | `cmd/tnp-testnet/main.go:656-663` vs `core/ledger/validate.go:206-210` |

---

## 7. Пункт 7 — спец-проверки

| проверка | результат |
|---|---|
| M3/M4/M5/M6 не изменены | **подтверждено**: дифф `validate.go` — три хунка (147–199), `applyMint` (M3–M6, строки 526–600) не тронут |
| двойной выпуск (evidence `DoubleMint`) совместим с новой семантикой | **подтверждено**: E6 требует лишь `ma.Period == mb.Period` (`validate.go:904-910`), окно в evidence-пути не участвует; два подписанных mint одного периода доказательство дают, как и раньше. Второй mint того же периода по-прежнему невозможен (M3: `mp.Period <= LastClaimedPeriod → E_MINT_DOUBLE`, `validate.go:540-543`) |
| genesis-поле 61 (`period_seconds`) не тронуто | **подтверждено**: `git diff main..mint-fix -- core/wire/` пусто |
| формула слота `hashThree(addr, period)` не менялась | **подтверждено**: `core/ledger/ledger.go` в диффе отсутствует; `mintChallengeSlot` — только комментарии |
| чекпоинт-хеш — PENDING, не реализован | **подтверждено**: `mintChallengeSlot` считает `hashThree(addr, period)`; чекпоинт-хеша в коде нет; в спеке 7.6 — примечание о PENDING |
| спека 7.6 RU/ENG хирургическая, версия +1, запись в истории | **подтверждено**: `docs/KNOWLEDGE-3.md:108` и `docs/KNOWLEDGE-ENG-3.md:109` — один пункт; версия 9.13 → 9.14 (`docs/KNOWLEDGE.md:8-9`, `KNOWLEDGE-ENG.md:11-12`); записи «Версия 9.14» в истории обеих версий (`docs/KNOWLEDGE.md:44`, `KNOWLEDGE-ENG.md:39`) |
| README/DISCLAIMER темой не трогались | **подтверждено**: `git diff main..mint-fix -- README.md README-ENG.md DISCLAIMER.md LICENSE*` пусто |

Замечание по спеке: помимо якоря окна и примечания о PENDING в пункт 7.6 добавлена фраза
«слот публично вычислим заранее — приложение может показать его пользователю». Мандат
(п. 7) это прямо разрешал («вычислим заранее приложением»), поэтому отклонением не считаю.

---

## 8. Находки

### AUDIT-MINT-FIX-01 — `spec/` описывает дофиксную семантику и старую версию протокола
**Приоритет: P2 (документация, не блокер).**
- `spec/03-validation.md:31` (C11): «для `MINT` — txNow в окне ±`mint_challenge_window_hours`
  … от слота челленджа `period_start + H(...) mod period_seconds` **заявляемого периода**
  (v9.13: окно — от claim)». После фикса это неверно: якорь — текущий период
  (`core/ledger/validate.go:165`). C11 — нормативная таблица валидации, по ней будут
  реализовывать старое правило.
- `spec/00-overview.md:46` и `spec/02-genesis-manifest.md:86` объявляют `protocol_version = 2`,
  тогда как дефолт и шаблоны — 3 (`core/params/params.go:210`, `cmd/tnp-testnet/main.go:393`,
  `configure.go:158`).
- Мандат (п. 7) ограничивал правку спеки разделом 7.6, поэтому это долг документации,
  а не нарушение мандата. Закрыть отдельным коммитом «spec: C11 и версия после mint-fix».

### AUDIT-MINT-FIX-02 — mobile-мост не даёт восстановления по `accrual_window`
**Приоритет: P2 (функциональный, не консенсусный).**
- `mobile/query.go:929` — `claim = period - 1` без ограничения окном накопления;
  `amount` считается по M6 от этого claim (`mobile/query.go:938-944`). Для кошелька,
  пропустившего ≥7 периодов, `diff > accrual_window`, и mint всегда отклоняется
  `E_ACCRUAL_WINDOW` — кнопка выпуска навсегда «отравлена» (проверено `verify recover`:
  claim=11 → reject/E_ACCRUAL_WINDOW).
- Протокол при этом допускает восстановление заявкой `last + accrual_window` (проверено:
  claim=7 → accept, `last_claimed=7`), но UI этого пути не предлагает; в Android обработки
  `accrual` нет вовсе (`grep -rn accrual android/` пусто).
- Дефект дотемы (не введён фиксом), но тема заявляет устранение поглощения и «UI и apply
  сходятся к одному вердикту» — на этом пути они расходятся. Минимальный фикс: в
  `MintWindowJSON` клампить `claim` до `last + accrual_window` (и считать `amount` от клампа).

### AUDIT-MINT-FIX-03 — инструменты живого прогона считают достижимость по старой формуле
**Приоритет: P3 (tooling).**
- `cmd/tnp-testnet/main.go:692` — заголовок вывода `slots`: «достижимо из периода P+1 только
  для поздних слотов, №39»; `:715-724` — достижимость `slot+win >= конец периода p`, т.е.
  дофиксное правило. После фикса достижимый интервал — пересечение окна с ТЕКУЩИМ периодом
  (≥48 ч, п. 2). Диагностика вводит оператора в заблуждение.
- `cmd/tnp-testnet/fleet.go:46-50` (`findlabels`): критерий «слот в последних N часах периода»
  и комментарий «якорь слота не меняется, №39» больше не гарантируют открытое окно в момент
  прогона для сети с окном ±48 ч. Фактические прогоны 12-testnet не ломались (окно 2400 ч,
  `cmd/tnp-testnet/configure_testnet_test.go:11-28`), но следующий прогон с окном 48 ч
  получит `E_POW_HARD` на mint.
- В отчёте исполнителя §4 сказано только «комментарий уточнён» — фактически уточнён лишь
  комментарий `challengeSlot`, а вывод и критерий отбора — нет.

### AUDIT-MINT-FIX-04 — «поглощающее состояние исчезает» требует уточнения
**Приоритет: P2 (формулировка, влияет на решение владельца).**
- Отчёт §2: «Поглощающее состояние исчезает». Замер (п. 2.4): устраняется структурная
  недостижимость (100 % периодов имеют окно), но при подаче в произвольный момент
  P(6 промахов подряд) = 0,437. Устранение требует показа слота приложением — это PENDING-2
  самого отчёта, т.е. вне темы.
- Формулировка спеки (`docs/KNOWLEDGE-3.md:108`: «у каждого кошелька каждый период есть
  достижимое окно») точна; формулировку отчёта стоит привести к ней же.

### AUDIT-MINT-FIX-05 — окно M7 не связывает claim-время при sync-подаче (прежнее свойство)
**Приоритет: P3 (наблюдение, вне темы, не регресс).**
- Ядро валидирует от `timestamp_claim` (`validate.go:228-229`), C9 — слой сети и при сверке
  не применяется (`net/sync.go:357-359`, `docs/KNOWLEDGE-4.md:44`,
  `reports/REPORT-STAGE-15-TIME-FIX.md:57`). Проверено (`verify retro`): при NetTime в
  периоде 10 mint с claim периода 3 принимается ядром.
- Следствие: окно M7 — не барьер живости для tx, доставленной sync'ом; пункт мандата
  «обход через sync-подачу закрыт» верен только для самой проверки окна. Если цель —
  именно живость, нужен отдельный слой допуска claim при сверке (тема этапа 15/17).

### AUDIT-MINT-FIX-06 — устаревший комментарий в headless-демо
**Приоритет: P3 (косметика).** `cmd/tnp-node/main.go:314-318`: «слот почти всегда вне окна,
№39» — после фикса окно всегда достижимо в текущем периоде; оверрайд `MintChallengeWindowHours = 0`
остаётся законным (песочница, демо-минт в произвольный момент), но обоснование нужно обновить.

### AUDIT-MINT-FIX-07 — числовые неточности в отчёте исполнителя
**Приоритет: P4 (info).**
- §2 «короткие периоды … (24 проверки)» — фактически 40 период-кейсов / 120 точечных проверок
  (`core/ledger/mint_fix_test.go:190-207`); в таблице §5 указано верно — внутреннее расхождение.
- §4 перечень мест `SkipMintChallengeWindow` неполон (не названы
  `core/ledger/txnow_regression_test.go:202,310`, `cons/replay_strict_test.go:65`); вывод
  «только sim + тестовые harness'ы» при этом верен.
- §9 «≥ 7,6 ГБ доступной из 15,5 ГБ» — на хосте 15,14 ГиБ всего, 5–7 ГиБ доступно в прогонах
  аудита; стоп-условие <2 ГиБ не достигалось.

---

## 9. PENDING (за владельцем)

1. **Чекпоинт-хеш в формуле слота** — открытый вопрос (мандат п. 5 запрещал менять формулу;
   код считает `H(address‖period)`, спека приведена к факту с примечанием). Не реализовано — верно.
2. **UX-показ слота пользователю** — вне мандата; без него выигрыш от фикса ограничен
   (см. AUDIT-MINT-FIX-04).
3. **Включение окна M7 в sim** — FINDING-01 исполнителя; после фикса это стало бы безопасно,
   но меняет результаты симуляции 8.1 — решение за владельцем.
4. **Экспорт слот-функции из ядра** — FINDING-02 исполнителя (устранит дубликат в
   `cmd/tnp-testnet`).

---

## 10. Вердикт

**«Готова с замечаниями».** Merge не блокируется.

- **Блокеры: нет.** Фикс ядра минимален и верен; свойство окна доказано аналитически и
  подтверждено независимым перебором по реальному коду (2400 случаев, минимум 48,07 ч);
  вердикты сквозных сценариев совпадают с ожидаемыми; live == replay; песочница изолирована;
  bump версии полный; тесты (a)–(h) на месте; полный набор, `vet`, `gofmt` и `-race` чистые;
  дифф — только предмет темы; секретов и файлов >1 МБ нет.
- **Перед merge желательно (P2):** AUDIT-MINT-FIX-01 (spec C11 + версия), AUDIT-MINT-FIX-04
  (формулировка «поглощение»), AUDIT-MINT-FIX-02 (кламп claim в mobile-мосте).
- **После merge / отдельными темами (P3–P4):** AUDIT-MINT-FIX-03 (tooling прогона),
  AUDIT-MINT-FIX-05 (claim при sync), AUDIT-MINT-FIX-06, AUDIT-MINT-FIX-07.

---

## 11. Воспроизводимость

```bash
# ветка и база
git fetch origin && git checkout -b audit-mint-fix mint-fix   # tip 2b4f0ac, merge-base d5ab36e

# гигиена
gofmt -l .
GOMAXPROCS=6 go vet ./...
GOMAXPROCS=6 /usr/bin/time -v go test ./...
GOMAXPROCS=6 /usr/bin/time -v go test -race -count=1 \
  ./core/ledger ./net ./cons ./mobile ./cmd/tnp-testnet ./core/store
GOMAXPROCS=6 go test -count=1 -v -run TestMintFix ./core/ledger

# независимая проверка по реальному коду (модуль в /tmp с replace на клон tnp)
mkdir -p /tmp/audit-mint-fix/verify && cd /tmp/audit-mint-fix/verify
cp reports/audit-mint-fix/verify-main.go.txt main.go   # из этого репозитория
cat > go.mod <<'EOF'
module auditmintfix
go 1.27
require github.com/rumatakira74/equality-trust-networks v0.0.0
replace github.com/rumatakira74/equality-trust-networks => /path/to/tnp
EOF
cp /path/to/tnp/go.sum .
GOFLAGS=-mod=mod GOPROXY=off go build -o verify .
./verify prop && ./verify e2e && ./verify replay && ./verify retro && ./verify recover && ./verify stats
```

Артефакты — `reports/audit-mint-fix/`: `verify-main.go.txt` (исходник проверки),
`prop.txt`, `e2e.txt`, `replay.txt`, `retro.txt`, `recover.txt`, `stats.txt`,
`tests-full.txt`, `tests-race.txt`, `tests-mint-fix.txt`, `vet-gofmt.txt`, `diff-files.txt`,
`env.txt`.
