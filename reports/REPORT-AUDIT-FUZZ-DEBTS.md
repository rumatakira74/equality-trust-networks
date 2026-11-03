# Отчёт blind-аудита темы «Fuzz-долги» (ветка fuzz-debts)

Аудит: ветка `audit-fuzz-debts` от tip `fuzz-debts` `ef06af9` (CI 8/8 зелёные,
CI-verify от 03.11.2026). Мандат аудита: `prompts/audit-fuzz-debts-prompt.md`
(main `abcae89`); мандат темы: `prompts/fuzz-debts-prompt.md`. Режим соблюдён:
только чтение и `reports/`; код, тесты, спека, промпты не менялись; находки
не правились. Пункты 1–5 выполнены до открытия отчёта исполнителя.

**Оговорка о слепоте.** Сессия аудита ранее выполняла CI-verify той же ветки
и видела фрагменты `REPORT-FUZZ-DEBTS.md` (таблицу инвентаризации раздела 1
и раздел 10 CI). Пункты 1–5 тем не менее выполнялись по факту кода, все
выводы ниже подтверждены независимо; расхождение с отчётом искалось, а не
списывалось. Оговорка фиксируется для честности протокола.

## 1. Полнота мандата (7 задач)

| Задача мандата | Статус | Доказательство | Замечание |
|---|---|---|---|
| 1. Инвентаризация недоверенного ввода | выполнена (минимум мандата) | `REPORT-FUZZ-DEBTS.md` раздел 1: таблица «функция → источник → лимиты» по `core/wire` (все `DecodeXxx`), `net/codec.go` (`readFrame`, `dec`, `decodeHeaders`, `decodeProof`, `decodePeers`, `decodeTxsMsg`), `core/attest` (`parseKeymint`, `scanAuthorizationList`, `derScanner`, `parseExtension`, `DecodeChain`, `Verify`), genesis-декодеры | все точки минимум-списка мандата покрыты; точки вне минимум-списка — FINDING-1 |
| 2. Fuzz-цели на группы декодеров | выполнена | 6 целей: `core/wire/fuzz_test.go` (`FuzzDecodeTransaction`:23, `FuzzDecodeGenesisPackage`:105, `FuzzDecodeStateLeaf`:127), `net/fuzz_test.go` (`FuzzNetFrame`:14), `core/attest/fuzz_test.go` (`FuzzAttestVerify`:16, `FuzzParseKeymint`:47) | группы wire/net/attest/genesis покрыты; seed-корпус в `f.Add` (67 векторов суммарно) |
| 3. Локальные прогоны с `-fuzztime` | выполнена (заявление; независимо воспроизведено выборочно) | отчёт: 30 м суммарно, бюджеты 10/4/3/6/2/5 м, ~66,5 млн execs, 0 крашей; аудит: smoke 6×60 с, ~13,9 млн execs, 0 крашей, 0 зависаний; скорости согласуются с заявленными (см. раздел 4) | краш-файлов в `testdata/fuzz/` нет — согласуется с «0 крашей» |
| 4. Property-тесты инвариантов | выполнена | `core/ledger/property_debts_test.go`: `TestPropertyDemurrageRandomStream` (300 шагов, `CheckInvariants` после каждого шага :112, сеттлемент на границах периодов :120, `FullCheck` на границах эпох :127, `checkIndexConsistency` после каждого шага), `TestPropertyDemurrageRules` (формула/floor/идемпотентность) | демередж включён (1%/период, floor 1000); дублирования с `property_test.go` нет — закрывает пробел (базовый тест демередж выключает) |
| 5. Тесты core/store | выполнена | `core/store/sqlite_test.go`: `TestStoreTxRoundTrip`:39, `TestStoreFingerprints`:68, `TestStoreBlacklist`:97, `TestStoreCheckpoints`:118 (headers + supply), `TestStoreMeta`:163, `TestStoreReopen`:180 (все таблицы после `Close`/`Open`) | до темы в пакете были только bench |
| 6. Proto-сверка | выполнена | `core/wire/proto_conformance_test.go`: разбор `.proto` в тесте, сверка (номер, wire-тип) в обе стороны + `TestProtoEnums`; аудит воспроизвёл санити-мутацию: `Transaction.seq 3→20` в `transactions.proto` → FAIL в обе стороны («поле seq (30) из .proto не кодируется» + «кодек кодирует поле 3, которого нет в .proto»), мутация откачена, дерево чисто, тест снова зелёный | механика ловит дрейф подтверждена |
| 7. Отчёт | выполнен | `reports/REPORT-FUZZ-DEBTS.md` (204 строки): инвентаризация, цели/seed-стратегия, лимиты с обоснованиями, бюджеты/результаты прогонов, находки, результаты property/store/proto, изменённые файлы, «CI pending» | CI на момент аудита 8/8 зелёные (core, net, cons, flutter, android на `ef06af9`) |

## 2. Качество fuzz-целей

- Seed-корпус: валидные векторы всех 16 ветвей payload (построены из
  `sampleTx()` существующих тестов), всех типов net-кадров (зеркально
  диспетчеру `serveSync`), валидная синтетическая цепочка аттестации и
  реальный KeyMint-вектор (TEE и software-fallback). Граничные: пустой
  ввод, усечение (`0x0a`, `0x08`, `0x30`), переполнение varint
  (`0xff`×16), гигантские заявленные длины (`0x0a, 0xff×4, 0x0f`),
  unknown-поле 127, вложенность evidence 20 уровней, 64 однобайтных
  «сертификата», длинная DER-форма тега без конца.
- Ошибки декода — ошибкой, не паникой: во всех целях проверка
  «ошибка с ненулевым результатом» отсутствия паник; smoke-прогоны
  (~13,9 млн execs) паник не нашли.
- Лимиты добавлены и обоснованы в коде комментариями + в отчёте
  (раздел 3): `maxDecodeDepth=16` (wire.go:737), `maxPersonalGuardians=16`,
  `maxApprovals=16`, genesis: `maxSeedCircle=1024`, `maxSlotTiers=16`,
  `maxMutableBounds=64`, `maxProviders=256`, `maxModules=16`
  (genesis_decode.go:14), `maxChainCerts=16` (attest.go:103). Значения
  сверены с кодом — совпадают.

## 3. Покрытие точек недоверенного ввода

Минимум-список мандата покрыт полностью:

- `core/wire`: `DecodeTransaction` + все 16 ветвей payload (прямая цель;
  ветвление через диспетчер `decodePayload`), `DecodeStateLeaf`,
  `DecodeWalletFlags`, `DecodePowProof` (одна цель, три декодера),
  genesis-декодеры — транзитивно через `DecodeGenesisPackage` →
  `DecodeManifest` → `DecodeProfileParams` → `DecodeSlotTier`/
  `DecodeMutableBound`/`DecodePowParams`/`DecodeSeedMember`/
  `DecodeSeedSunset` и `DecodeGenesisRecord` (граф вызовов сверен).
- `net/codec.go`: `readFrame` + `dec.u64/blob/str/tag/done` +
  `decodeHeaders`/`decodeProof`/`decodePeers`/`decodeTxsMsg` — одна цель
  `FuzzNetFrame`, switch зеркален диспетчеру `serveSync` (сверено по
  `net/sync.go`).
- `core/attest`: `Verify` (→ `DecodeChain` + `parseExtension` +
  `parseKeymint` fallback) — `FuzzAttestVerify`; `parseKeymint` →
  `scanAuthorizationList` → `derScanner` — `FuzzParseKeymint`.

Точки вне минимум-списка — см. FINDING-1.

## 4. Сверка с отчётом исполнителя

| Заявление отчёта | Факт аудита | Вердикт |
|---|---|---|
| 6 fuzz-целей, seed в `f.Add`, кеш `GOCACHE` не коммитится, краш-файлов нет | 6 целей в коде; `f.Add` 40+11+16 векторов; `testdata/fuzz/` в ветке отсутствует; `git status` чист после smoke-прогонов (кеш ушёл в `GOCACHE`) | подтверждено |
| 9 лимитов со значениями и обоснованиями | все 9 констант в коде с теми же значениями и обоснованиями (wire.go:737, genesis_decode.go:14, attest.go:103) | подтверждено |
| Прогоны 30 м / ~66,5 млн execs / 0 крашей | напрямую не воспроизводимы (прошлые прогоны); smoke 6×60 с дал ~13,9 млн execs, 0 крашей; удельные скорости согласуются: Transaction ~46 тыс/с против заявленных ~35 тыс/с, NetFrame ~3,4 тыс/с против ~3,7 тыс/с, AttestVerify ~29 тыс/с против ~38 тыс/с (`-parallel=2` в обоих случаях) | правдоподобно, противоречий нет |
| Дрейф proto: 4 файла, перечень правок (RedeemPayload 3↔4, ContributionClaim +pool_id=1 и сдвиг 2–4, update_guardians=25, TxType=16, slug=7, SeedMember без address=1, ProfileParams 56–61, TRANSFER_MODE_PEER_REDEEM=4, AttestStatement) | `git diff main...fuzz-debts -- proto/` — совпадает построчно; кодек не менялся (в диффе `wire.go` нет изменений номеров полей — только лимиты), т.е. wire-байты не менялись, `protocolVersion` не поднимался | подтверждено |
| «Санити-проверка: мутация номера поля ловится в обе стороны» | воспроизведено: seq 3→20 → FAIL с двумя диагностиками, откат → зелёный | подтверждено |
| Property: 300 шагов, инварианты после каждого шага, FullCheck на границах эпох | `step < 300` (property_debts_test.go:81), `CheckInvariants` в теле цикла, `FullCheck` каждые 50 шагов | подтверждено |
| Store: round-trip, fingerprints, blacklist, headers/supply, meta, reopen | 6 тестов, все зелёные (`go test ./core/store/` 0,013 с) | подтверждено |
| Предкоммит: gofmt/vet чисто, `go test ./...` зелёный, `-race` по задетым пакетам зелёный | gofmt -l пусто; vet чисто; 5 пакетов зелёные; аудит прогнал `-race` по всем 5 задетым (`core/wire`, `core/attest`, `core/store`, `core/ledger` 254 с, `net` 134 с) — зелёные | подтверждено |
| «CI pending» | на момент аудита CI 8/8 зелёные (CI-verify 03.11.2026) | подтверждено (статус известен) |

Расхождений с отчётом не найдено.

## 5. Находки

| # | FINDING (файл/строка) | Значимость | Рекомендация (НЕ реализовано) |
|---|---|---|---|
| 1 | Точки недоверенного ввода вне минимум-списка мандата не инвентаризированы и не покрыты fuzz-целями: `net/gossip.go:183` `decodeCheckpoint` (кадры gossip от пиров; повторяющиеся поля `Participants`/`FinalizedTxIDs` без счётных лимитов — память ограничена только размером кадра ≤16 МиБ), `net/seedlist.go:137,165,193` `decodeThresholdSig`/`decodeSignatureShare`/`parseSeedListPayload` (байты seed-списка из внешних каналов; лимиты 32/1024/64 есть), `net/cons.go:80,145` протокол голосования (`msgVoteQuery`/`msgVoteResp`), клиентский разбор `msgTip` (`net/sync.go:442`, u64+blob) | средняя (не блокер): примитивы `dec` (tag/blob/u64/str) покрыты транзитивно `FuzzNetFrame`, у seedlist-парсеров лимиты стоят; но собственная логика этих парсеров не фаззится, а мандат задачи 1 требовал «полный список по факту кода», не только минимум | отдельной темой-долгом: добавить инвентаризацию этих точек и цели `FuzzNetCheckpoint`/`FuzzSeedList`/`FuzzVote`; счётные лимиты для `Participants`/`FinalizedTxIDs` |

Крипто-наблюдений вне мандата аудита не возникло (контур 2 не привлекался).

Замечание по процессу (не FINDING): параллельная ветка `fuzz-debts-ab`
(tip `8e88aee`) с той же темой от того же базиса `0e44d83` — не аудитировалась
(по мандату, «только база сверки»), но владельцу стоит решить её судьбу до
merge, чтобы не осталась живая дублирующая ветка.

## 6. Процесс и гигиена ветки

- `git diff main...fuzz-debts`: 14 файлов, +1740/−11 — только предмет темы
  (лимиты, fuzz-цели, property/store/proto-тесты, `proto/*.proto`
  синхронизация к факту кодека, отчёт). Правок вне мандата нет: net-DoS,
  UI, F5/F16, genesis-семантика, `protocolVersion`, README — не тронуты.
- Коммиты событийные: `8326d23` (лимиты + 6 целей) → `d768f4d` (store) →
  `8680717` (property) → `2a05cb2` (proto-сверка + дрейф) → `ef06af9`
  (отчёт); форматирование отдельным коммитом не требовалось (gofmt чист).
- Артефакты: отчёт 19 КБ; файлов >1 МБ нет; корпусной кеш не закоммичен.
- Ресурсные правила аудита соблюдены: `GOMAXPROCS=6`, `-parallel=2`,
  цели последовательно, свободная память ≥5 ГБ на всех контрольных точках;
  после прогонов `pgrep` чист (go-процессов, демонов, эмуляторов нет).

## 7. Вердикт

**Ветка `fuzz-debts` готова к merge с замечаниями.**

- Блокеров нет: все 7 задач мандата выполнены по факту кода; заявления
  отчёта подтверждены (расхождений нет); CI 8/8 зелёные; wire-байты и
  `protocolVersion` не менялись.
- Замечание одно — FINDING-1 (покрытие точек недоверенного ввода вне
  минимум-списка мандата: gossip/seedlist/cons/`msgTip`); рекомендуется
  закрыть отдельной темой-долгом, решение за владельцем.

Отчёт аудита: `reports/REPORT-AUDIT-FUZZ-DEBTS.md` (этот файл).
Артефакты `reports/audit-fuzz-debts/` не потребовались — выжимки прогонов
выше.
