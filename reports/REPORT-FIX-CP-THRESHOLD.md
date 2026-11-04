# REPORT-FIX-CP-THRESHOLD — C1 аудита Sonnet-2: подделка чекпоинта

Ветка `fix-cp-threshold` от свежего `main` (eba8522). Мандат:
`prompts/fix-cp-threshold-prompt.md` (коммит 36b515e). Источник находки:
`reports/DIGEST-AUDIT-SONNET-2.md` (C1, критичный) и полный отчёт
`origin/audit-sonnet:reports/audit-sonnet/REPORT-AUDIT-SONNET-2.md` п. 4.1.
Красные тесты аудита: `origin/audit-sonnet:reports/audit-sonnet/go-tests/cons_forged_checkpoint_test.go.txt`.

**session-ID: `2026-11-05-fix-cp-threshold`** (метка сессии; прогоны — 04.11.2026, UTC).

Мержа нет (мандат: «пуш ветки, НЕ мержить»).

---

## 1. Долг (что было) — file:line на main (eba8522)

| Место | Дефект |
| --- | --- |
| `cons/checkpoint.go:168-170` | `Verify` берёт `cp.Participants`/`cp.Threshold` из самого чекпоинта; при `Threshold = 0` условие `valid(0) < 0` ложно — чекпоинт без единой подписи проходит |
| `cons/checkpoint.go:264` | `Collector.Ready` возвращает полную подпись при `len(shares) ≥ Threshold`, то есть сразу при `Threshold = 0` |
| `net/cons.go:426` | `cons.Participants(...)` вызывается только при **построении**; на приёме состав/порог с манифестом не сверяются |
| `net/cons.go:618-622`, `:664` | `onCpMessage` подписывает первое увиденное предложение своей эпохи, **не сверяя `state_root`** со своим реестром |
| `net/cons.go:703`, `:714`, `:732` | `stateRootMatches` применяется только к перепубликованному финализированному чекпоинту (ветка `received`), ветка `ready` (порог набран локально) её не проходит |
| `net/gossip.go:69` | `RegisterTopicValidator` — только для темы tx; тема чекпоинтов принимается без валидатора (`:103` — «заготовка, приём без проверки подписи») |
| `net/cons.go:612-674` | нет проверки «эпоха наступила» и принадлежности предложения строителю эпохи: кто опубликовал раньше, тот и выиграл эпоху |

Итог по аудиту: произвольные `StateRoot`/`TotalSupply`/`NetworkTime`/`FinalizedTxIDs`
→ выбор ветки эквивокации (двойная трата).

## 2. Консенсус-оформление (по образцу 3.2 отчёта аудита)

Правило меняется, поэтому — **новое поле профиля, не `protocol_version`**:

- **поле 62 `cp_set_check`** (`core/wire/genesis.go:285-295`, `:346`;
  `core/wire/genesis_decode.go:362`; `proto/genesis.proto:169-174`):
  `0` — наследие (состав и порог из чекпоинта), `1` — строгая сверка
  `participants`/`threshold` с `cons.Participants(...)` по режиму манифеста.
- **Старые сети не ломаются**: канонический энкодер опускает нулевые
  значения (`appendU64Field`), поэтому манифест прежней сети кодируется
  байт-в-байт как раньше — `network_id` не меняется, `protocol_version`
  остаётся 3 (формат транзакций не изменился).
- **Старые узлы на новом манифесте отказывают закрыто**: неизвестное поле
  профиля → `errUnknownField` (`core/wire/genesis_decode.go:361-362`).
- **Новые сети**: `configure` ставит `cp_set_check = 1`
  (`cmd/tnp-testnet/configure.go:84`, вывод — `:363`).
- **Валидация значения**: `cp_set_check ∈ {0,1}` — genesis
  (`core/ledger/ledger.go:107-109`, п. 2.6 (8)) и профиль
  (`core/profiles/profiles.go:268-270`).
- Спека: `spec/02-genesis-manifest.md` (п. 2.2.2, поле в списке [P]),
  `spec/07-checkpoint-threshold.md` (7.2.1), решение — `QUESTIONS.md` №48.

## 3. Что сделано по шагам мандата

### Шаг 1. Принимающая сторона сверяет состав/порог с манифестом; порог 0 отвергается

- `cons/checkpoint.go:137-160` (`Verify`): безусловно отвергаются пустой
  состав, порог `0` и порог больше числа участников — легального
  чекпоинта с такими значениями не бывает ни в одном режиме
  (`modes.go`: `N ≥ 2`, `N ≤ M`), поэтому обратная совместимость не
  страдает; сигнатура `Verify` сохранена (тесты аудита компилируются как есть).
- `cons/checkpoint.go:281-290` (`Collector.Ready`): при пороге `0`
  готовность не наступает.
- `cons/modes.go:65-86` (`CheckParticipants`): сверка состава и порога с
  вычисленными по режиму; **порядок значим** — очередь строителей
  (№42 (A)) индексирует состав по номеру окна.
- `net/cons.go:764-786` (`cpCheckParticipants`): при `cp_set_check = 1`
  состав вычисляется локально (`cons.Participants` по режиму, эпохе и
  `prev_hash`) и сверяется; вызывается на приёме
  (`net/cons.go:831-836`, перед `cons.Verify`) и перед подписью
  (`net/cons.go:712-716`).

### Шаг 2. Подписант подписывает только при совпадении StateRoot со своим реестром

- `net/cons.go:712-746` (`cpSignGate`): `stateRootMatches` (собственный
  реестр) — обязательное условие подписи; иначе доля не ставится.
- `net/cons.go:666-673`: ворота подписи вынесены **после** `n.mu` и
  применяются только к новым чужим предложениям — дорогие проверки
  (полное SMT-дерево) не выполняются на каждое сообщение темы.

### Шаг 3. «Эпоха наступила» и принадлежность предложения строителю эпохи

- `net/cons.go:748-762` (`cpEpochArrived`): сеть-время предложения не
  раньше детерминированного момента `due` (`prev.network_time +
  epoch_seconds`; без prev — `genesis + epoch_seconds`) **и** локальные
  часы дошли до `due` с запасом `max_time_shift` (9.2) — расхождение
  часов не отбрасывает предложение живого строителя.
- `net/cons.go:640-648` (`onCpMessage`): предложение не наступившей эпохи
  не регистрируется — закрыт сценарий аудита «опубликовать эпоху
  `last+1` раньше честного строителя, первое увиденное побеждает».
- `net/cons.go:736-739`: при собственном предложении эпохи чужой вариант
  той же эпохи не подписывается (принадлежность строителю; два варианта
  одной эпохи раскалывают набор подписантов — блокер 8.1).

### Шаг 4. Валидатор темы чекпоинтов

- `net/gossip.go:110-124`: `RegisterTopicValidator` для темы
  `checkpoint/<network_id>` (раньше — только для tx).
- `net/gossip.go:137-226` (`cpAdmit`): структура кадра, своя сеть, длины
  `state_root`/`prev_hash`/адресов/`finalized_txid`, порог `0` и порог
  больше состава, подписант из состава, размер сообщения. Отказ —
  `ValidationReject` + счёт спама пира (тот же механизм, что у темы tx).
- `net/gossip.go:254-271` (`checkpointLoop`): использует разобранное
  валидатором сообщение (`ValidatorData`), с прежним путём разбора как
  запасным.

## 4. Тесты

### Красные тесты аудита — красные на входе, зелёные после фикса

Файл аудита перенесён как `cons/forged_checkpoint_test.go` (ожидания не
менялись). На входе, до фикса:

```
$ go test -tags cpforgefix ./cons -run Forged -v
=== RUN   TestForgedCheckpoint_ZeroThresholdRejectedByVerify
    forged_checkpoint_test.go:40: чекпоинт без подписей и с порогом 0 принят Verify
--- FAIL: TestForgedCheckpoint_ZeroThresholdRejectedByVerify (0.00s)
=== RUN   TestForgedCheckpoint_ZeroThresholdCollectorNotReady
    forged_checkpoint_test.go:50: порог 0 без единой подписи считается достигнутым
--- FAIL: TestForgedCheckpoint_ZeroThresholdCollectorNotReady (0.00s)
FAIL	.../cons	0.002s
```

После фикса тег `cpforgefix` снят (как предписано README аудита) — тесты
входят в обычный прогон:

```
$ go test ./cons -run Forged -v
--- PASS: TestForgedCheckpoint_ZeroThresholdRejectedByVerify (0.00s)
--- PASS: TestForgedCheckpoint_ZeroThresholdCollectorNotReady (0.00s)
ok  	.../cons	0.003s
```

### Новые тесты

| Тест | Что фиксирует |
| --- | --- |
| `cons/cp_set_test.go:TestVerifyThresholdSanity` | порог > состава и пустой состав отвергаются `Verify` |
| `cons/cp_set_test.go:TestCheckParticipants` | расхождение порога/числа/адреса/порядка участников отвергается |
| `net/cp_forge_fix_test.go:TestCpSignGateForeignStateRoot` | подпись под чужим `StateRoot` не ставится; под своим — ставится |
| `net/cp_forge_fix_test.go:TestCpSignGateEpochNotArrived` | предложение не наступившей эпохи не регистрируется; наступившей — регистрируется |
| `net/cp_forge_fix_test.go:TestCpSignGateOwnVariant` | чужой вариант эпохи при своём предложении не подписывается |
| `net/cp_forge_fix_test.go:TestCpStrictParticipantsCheck` | строгий режим отвергает подмену состава и порога; наследие (0) — не отвергает |
| `net/cp_forge_fix_test.go:TestCpStrictParticipantsNetwork` | **регресс**: сеть со `cp_set_check = 1` финализирует валидные чекпоинты |
| `net/cp_forge_fix_test.go:TestCpTopicValidator` | валидатор темы: порог 0/порог > состава/чужая сеть/подписант вне состава/битый кадр отвергнуты, валидное предложение и доля проходят |

## 5. Прогоны (04.11.2026, UTC; Go 1.27.3, linux/amd64)

| Команда | Результат |
| --- | --- |
| `go build ./...` | rc=0 |
| `go test ./... -count=1` | rc=0 (все пакеты `ok`; `net` 67,5 с, `core/ledger` 56,6 с, `sim` 140,1 с) |
| `go test -race ./cons/ -count=1` | rc=0 (`ok` 2,5 с) |
| `go test -race ./net/ -count=1` | rc=0 (`ok` 114,9 с) |
| `go test ./cons -run Forged` | `ok` (после снятия тега) |

Регресс существующих чекпоинт-тестов: `TestCheckpointTransitionalNetwork`,
`TestCpBuilderQueueRotation`, `TestCpSkipProposal`, `TestCpSkipSignerGate`,
`TestLightNodeCheckpointProof`, `TestF16OfflineDoubleSpend`,
`TestVoteFinalityNetwork` — зелёные (входят в полный прогон `./net/`).

## 6. Обратная совместимость

| Кто | Что видит | Поведение |
| --- | --- | --- |
| Сеть со старым манифестом (поле 62 отсутствует = 0) | байт-в-байт прежний манифест, прежний `network_id` | состав/порог по-прежнему берутся из чекпоинта; работают как раньше |
| Новый узел на старом манифесте | `cp_set_check = 0` | сверки состава нет; **но** порог 0/пустой состав отвергаются, подпись под чужим `StateRoot` не ставится (локальная политика, валидность не меняет) |
| Старый узел на новом манифесте | неизвестное поле 62 | отказ закрыто (`errUnknownField`), тихого применения другой семантики нет |
| Новая сеть (`configure`) | `cp_set_check = 1` | строгая сверка состава и порога |

Порог 0, пустой состав и порог > числа участников отвергаются
**безусловно**: легального чекпоинта с такими значениями не бывает ни в
одном режиме (переходный: `N = max(2, 2/3 M)`; `decentralized`/
`members_quorum`: `N = ceil(2/3 M)`), поэтому правило не может отвергнуть
валидный чекпоинт прежней сети.

## 7. Остаточные риски и PENDING

1. **Живучесть (не безопасность).** Предложение с чужим `StateRoot`,
   опубликованное ровно в момент `due`, регистрируется («первое
   увиденное побеждает») и блокирует построение своего предложения
   строителем окна; подписей под ним не будет, эпоха закрывается
   пропуском (D) — стагнация ≤ 1 эпохи. Закрытие потребовало бы разрешить
   строителю окна строить свой вариант при чужом предложении эпохи
   (правка правила одного предложения, блокер 8.1) — вне минимального
   плана C1, **PENDING: решение владельца**.
2. **Подписант временно не подписывает, пока не дошёл до того же
   состояния** (следствие шага 2, отмечено в отчёте аудита 4.1). Это
   нормальное поведение, но при систематическом отставании узла его доля
   выпадает; порог 2/3 это переживает.
3. **Сверка состава в `decentralized` зависит от локального состояния**
   (`active`-кошельки): узел, отставший по применению tx, вычислит другой
   состав и отвергнет чекпоинт. Для новых сетей (`cp_set_check = 1`) это
   осознанный выбор «отказ закрыто»; переходный режим (посевной круг)
   детерминирован манифестом и от состояния не зависит.
4. **Ветка `ready` (`col.Ready()`, `net/cons.go:703`) по-прежнему не
   сверяет `StateRoot`**: при шаге 2 порог из честных подписантов под
   чужим корнем не собирается, поэтому отдельная проверка не добавлялась
   (добавление изменило бы приём финализированных чекпоинтов для узлов,
   ушедших вперёд по состоянию, — риск живучести без выигрыша в
   безопасности). **PENDING: решение владельца** при следующей правке
   приёма чекпоинтов.
5. **V1 аудита (сибиль `PollRound` по пирам)** — вне темы C1, не трогался.

## 8. Файлы и коммиты

```
core/wire/genesis.go              поле 62 CpSetCheck + Encode
core/wire/genesis_decode.go       Decode поля 62
proto/genesis.proto               поле 62
core/ledger/ledger.go             genesis 2.6 (8): cp_set_check ∈ {0,1}
core/profiles/profiles.go         валидация профиля
cmd/tnp-testnet/configure.go      новые сети: cp_set_check = 1
cons/checkpoint.go                Verify: порог/состав; Collector.Ready
cons/modes.go                     CheckParticipants
net/cons.go                       cpCheckParticipants, cpSignGate, cpEpochArrived,
                                  регистрация/подпись, приём
net/gossip.go                     валидатор темы чекпоинтов (cpAdmit)
cons/forged_checkpoint_test.go    тесты аудита (тег снят)
cons/cp_set_test.go               новые тесты cons
net/cp_forge_fix_test.go          новые тесты net
spec/02-genesis-manifest.md       п. 2.2.2, поле 62 в списке [P]
spec/07-checkpoint-threshold.md   7.2.1: правило валидности и подписи
QUESTIONS.md                      №48 (решение по оформлению и остаткам)
```

Коммиты ветки: `a2563e2` (поле 62 + `cons`), `e580ff3` (сеть: шаги 1–4),
далее — отчёт и спека.
