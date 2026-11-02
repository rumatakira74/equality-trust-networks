# Отчёт темы «Fuzz-долги» (ветка fuzz-debts)

Тема: fuzzing и property-based тесты — самописный protobuf-кодек, ASN.1-парсеры,
core/store, сверка proto, инварианты ядра. Ветка `fuzz-debts` от `origin/main`
`0e44d83`. Промпт: `prompts/fuzz-debts-prompt.md`. Живых сетей нет;
`protocolVersion` не поднимался (wire-байты не менялись).

## 1. Инвентаризация недоверенного ввода (шаг 0)

Таблица «функция → источник недоверенных байтов → лимиты». «было» — до темы,
«стало» — после (добавленные лимиты — задача 2, обоснования в разделе 3).

| Функция (пакет) | Источник недоверенных байтов | Лимиты |
|---|---|---|
| `wire.DecodeTransaction` + `decodePayload` (все 16 ветвей) | кадры `msgTxs`/`msgJournal` от пиров (`net/sync.go decodeTxsMsg`, ≤4096 tx на кадр), raw из `core/store` | varint ≤63 бит, bytes — длина против остатка (нет аллокации по заявленной длине); **глубина вложенности evidence: не было → 16** |
| `wire.DecodeRegisterPayload` | payload register из tx пира | **personal_guardians: не было → 16** |
| `wire.DecodeMigratePropose` | payload migrate из tx пира | **approvals: не было → 16** |
| `wire.DecodeUpdateGuardiansPropose` | payload update_guardians из tx пира | **new_guardians/approvals: не было → 16** |
| `wire.DecodeEvidencePayload` | payload evidence из tx пира | вложенность через `decodeTransaction(depth)` |
| прочие `wire.DecodeXxx` payload'ы, `DecodePowProof`, `DecodeStateLeaf`, `DecodeWalletFlags` | tx пиров; `LeafData` Merkle-доказательства (`net decodeProof`) | поля фиксированы, повторов/вложенности нет — лимиты не требуются |
| `wire.DecodeManifest`, `DecodeProfileParams`, `DecodeSeedMember`, `DecodeSeedSunset`, `DecodeSlotTier`, `DecodePowParams`, `DecodeMutableBound` | genesis-пакет чужой сети (файл/пиры) | **seed_circle: не было → 1024; slot_tiers: → 16; mutable_bounds: → 64; providers: → 256; modules: → 16** |
| `wire.DecodeGenesisRecord`, `DecodeGenesisPackage` | genesis-пакет чужой сети | вложенность фиксирована (3 уровня), поля фиксированы |
| `net.readFrame` + `dec.u64/blob/str/tag/done` | все служебные кадры пиров (`readFrameStream`, кадр ≤16 МиБ) | длина кадра против остатка — было и есть |
| `net.decodeHeaders` | ответ `msgHeaders` | ≤4096 заголовков — было |
| `net.decodeProof` | ответ `msgProof` (лёгкий узел) | ≤4096 siblings — было |
| `net.decodePeers` | ответ `msgPeers` (peer exchange) | ≤1024 пиров, ≤64 адресов — было |
| `net.decodeTxsMsg` | ответы `msgTxs`/`msgJournal` | ≤4096 tx — было |
| `attest.DecodeChain` | `RegisterPayload.AttestationChain` — цепочка аттестации чужого устройства | префикс длины против остатка — было; **число сертификатов: не было → 16** |
| `attest.Verify` (x509-верификация) | там же | пул промежуточных ограничен лимитом цепочки |
| `attest.parseKeymint`, `scanAuthorizationList`, `derScanner` | DER-расширение аттестации OID 1.3.6.1.4.1.11129.2.1.17 из листа цепочки | длинная форма тега — до конца ввода; форма длины k≤4 байт; содержимое против остатка — было |
| `attest.parseExtension` | синтетическое расширение (песочница/этап 3) | `encoding/asn1` stdlib, целевая структура плоская |

Вывод шага: паник в decode-путях нет (все ошибки — ошибкой); критические пробелы —
неограниченная вложенность evidence (O(n²) копирований: каждый уровень копирует
остаток блоба) и неограниченные счётчики повторяющихся полей; в `net/codec.go`
лимиты уже стояли. Закрыто лимитами (раздел 3) + проверено fuzz-прогонами (раздел 4).

## 2. Fuzz-цели и seed-стратегия

Нативный Go fuzzing. Seed-корпус задан в `f.Add` (прогоняется штатным `go test`
в CI на каждом запуске): валидные векторы всех ветвей из существующих тестов +
граничные (пустой ввод, усечение в середине тега/длины, переполнение varint,
гигантские заявленные длины полей, unknown-поля, вложенность 20 уровней,
много «сертификатов» по байту). Кеш корпусов (`GOCACHE`) не коммитится;
краш-файлы в `testdata/fuzz/` не появились (крашей нет).

| Цель | Пакет | Покрывает |
|---|---|---|
| `FuzzDecodeTransaction` | `core/wire` | конверт + все 16 ветвей payload + вложенность evidence + Pow |
| `FuzzDecodeGenesisPackage` | `core/wire` | genesis-пакет: манифест, профиль, посевной круг, запись |
| `FuzzDecodeStateLeaf` | `core/wire` | StateLeaf, WalletFlags, PowProof (лист состояния из доказательства) |
| `FuzzNetFrame` | `net` | `readFrame` + разбор тел всех типов сообщений, зеркально диспетчеру `serveSync` |
| `FuzzAttestVerify` | `core/attest` | `Verify` целиком: DecodeChain + синтетическое/KeyMint-расширение + x509 против синтетического корня |
| `FuzzParseKeymint` | `core/attest` | `parseKeymint`/`scanAuthorizationList`/`derScanner` (ASN.1, длинные теги 704/706) |

## 3. Добавленные лимиты и обоснования

Все лимиты — защитные пределы декодера с многократным запасом над семантическими
максимумами; легитимные сообщения отправителей не затрагивают (protocolVersion
не поднимался, приём валидных сообщений не изменился).

| Лимит | Значение | Обоснование |
|---|---|---|
| глубина вложенности `Transaction` | 16 | evidence содержит конфликтующие tx целиком (spec/01 1.5.4), легитимная глубина 2; без лимита вложенность даёт O(n²) копирований — зависание на ~1 МиБ вводе |
| `personal_guardians`, `new_guardians` | 16 | семантический максимум — 3 гардиана (`GuardianPersonalCount`, раздел 4) |
| `approvals` (migrate, update_guardians) | 16 | семантический максимум — число гардианов (3) |
| `seed_circle` | 1024 | рекомендация посевного круга ≥ 300 (7.7) |
| `slot_tiers` | 16 | в профилях 2 яруса (7.4) |
| `mutable_bounds` | 64 | изменяемых [M]-параметров порядка двух десятков (13) |
| `providers` | 256 | список поставщиков сети (6.2) |
| `modules` | 16 | закрытый перечень из 5 модулей (раздел 6) |
| сертификатов в цепочке аттестации | 16 | передаются лист + промежуточные, корни зашиты в APK (7.2); у Android 2–4; без лимита пул промежуточных раздувает перебор цепочки x509 |

## 4. Прогоны и результаты

Прогоны локальные, `-fuzztime` по критичности пути; после замечания владельца по
загрузке CPU — последовательно, `-parallel=2` (правило владельца: Go-прогоны
≤ 6 из 8 ядер; контроль `free` перед стартом и во время, доступно ≥ 8 ГБ на
всех контрольных точках). Сводка по целям:

| Цель | Бюджет | Execs | Краши |
|---|---|---|---|
| `FuzzDecodeTransaction` | 10 м | ~21,1 млн | 0 |
| `FuzzDecodeGenesisPackage` | 4 м | ~16,4 млн | 0 |
| `FuzzDecodeStateLeaf` | 3 м | ~9,9 млн | 0 |
| `FuzzAttestVerify` | 6 м | ~13,7 млн | 0 |
| `FuzzParseKeymint` | 2 м | ~4,2 млн | 0 |
| `FuzzNetFrame` | 5 м | ~1,1 млн | 0 |
| **Итого** | **30 м** | **~66,5 млн** | **0** |

Паник, зависаний и неограниченных аллокаций ни одна цель не нашла: decode-пути
возвращают ошибки; добавленные лимиты срабатывают ошибкой (не паникой), сид
20-уровневой вложенности разбирается с ошибкой глубины.

## 5. Находки и фиксы

**Краши fuzzing: 0.** Регресс-сиды не потребовались.

**Находка 1 (не краш, дрейф документации wire-формата): `proto/*.proto`
расходились с ручным кодеком.** Найдено при написании теста сверки (задача 6),
закрыто правкой `.proto` (кодек — канон: детерминизм байтов и хеши считаются по
нему; wire-байты и `protocolVersion` не менялись, живых сетей нет):

- `transactions.proto`: `RedeemPayload` — amount=3/slot=4 (в .proto было
  slot=3/amount=4); `ContributionClaimPayload` — pool_id=1,
  description_hash=2, size=3, period=4 (в .proto была нумерация 1–3 без
  pool_id); oneof payload — добавлена ветвь `update_guardians = 25`; добавлены
  сообщения `UpdateGuardiansPayload`/`UpdateGuardiansPropose`/
  `UpdateGuardiansCancel`, `GuardianChangeStatement`.
- `core.proto`: `TxType` — добавлен `UPDATE_GUARDIANS = 16`.
- `genesis.proto`: `Manifest` — добавлен `slug = 7` (v9.11); `SeedMember` —
  удалён `address = 1` (v9.11 fingerprint-манифест, старые бинарники не
  декодируются, живых сетей нет); `ProfileParams` — добавлены поля 56–61
  (cp_builder_queue, fee_adjust_epochs, fee_target_growth_micros,
  fee_step_micros, recurring, period_seconds); `TransferMode` — добавлен
  `TRANSFER_MODE_PEER_REDEEM = 4` (v9.10).
- `attest.proto`: добавлено сообщение `AttestStatement` (подписываемое
  утверждение предъявления; в .proto был только конверт `Attestation`).

Спека (docs/KNOWLEDGE-*.md) не трогалась: семантика wire-формата не менялась,
изменена только документация `.proto` к факту кодека.

**Замечание (вне сверки):** `wire.MerkleProof` (checkpoint.proto) не имеет
Encode/Decode в кодеке (доказательства ходят кадром `msgProof` в компактном
формате `net/codec.go`) — в сверке не участвует, тегов не имеет.

## 6. Property-тесты на инварианты ядра

Сверено с существующим `core/ledger/property_test.go` (случайный поток без
демереджа + метаморфный тест) — новые тесты закрывают пробелы, не дублируя:

- `TestPropertyDemurrageRandomStream` — рандомизированный поток tx с
  ВКЛЮЧЁННЫМ демереджем (1%/период, floor 1000): `CheckInvariants` после
  каждого шага и после сеттлемента на границах периодов, `FullCheck` на
  границах эпох, консистентность индексов после каждого шага (ключи TxIndex =
  txid; tx только известных кошельков; ровно один tx на seq 2..Seq;
  фингерпринт-индекс ↔ кошельки; burned ≤ minted — нет underflow-обёртки).
- `TestPropertyDemurrageRules` — детерминированные правила 10.3/№31:
  списание ровно `(base − floor) × rate` за период, base = balance − locked,
  floor и ниже не трогаются, идемпотентность сеттлемента внутри периода,
  сгоревшее попадает в `TotalBurned` (INV-8 сохранение предложения).

Результат: оба зелёные (300 шагов потока; INV-1d/2a/8/11a/13a + INV-1/9/10
FullCheck не нарушались ни на одном шаге).

## 7. Тесты core/store

`core/store/sqlite_test.go` (до темы — только bench): round-trip
`SaveTx`/`LoadTxs` (canonical-байты и txid стабильны; повторная запись —
`INSERT OR IGNORE` без дубликата), fingerprints (`INSERT OR REPLACE`),
blacklist (`INSERT OR IGNORE`), checkpoint headers (`LatestCheckpointHeader`
по максимальной эпохе) и supply (ряд по возрастанию эпох), meta
(REPLACE/отсутствующий ключ), восстановление всех данных после `Close`/`Open`.
Результат: зелёные.

## 8. Сверка proto ↔ ручной кодек

Механика: разбор `.proto` в тесте (`core/wire/proto_conformance_test.go`).
Выбор перед golden-векторами обоснован: разбор .proto падает при изменении
.proto напрямую (golden-векторы — нет, они фиксируют байты кодека).
`TestProtoConformance`: для каждого сообщения кодека — полностью населённый
экземпляр, кодирование, фактические (номер, wire-тип) пары против .proto в обе
стороны (каждое закодированное поле есть в .proto с тем же wire-типом; каждое
не-oneof поле .proto кодируется; oneof — не более одной ветви).
`TestProtoEnums`: номера значений всех enum'ов кодека против .proto в обе
стороны. Санити-проверка: мутация номера поля в .proto ловится в обе стороны.
Результат: зелёные после закрытия дрейфа (раздел 5).

## 9. Предкоммитная проверка

- `gofmt -l .` — пусто; `go vet ./...` — чисто.
- `go test ./...` — зелёный (fuzz-цели штатно прогоняют seed-корпуса).
- `-race` по задетым пакетам (`core/wire`, `core/attest`, `core/ledger`,
  `core/store`, `net`) — зелёный.
- Сборка `cmd/...` — проходит.
- Flutter/Kotlin-код темы не менялся — flutter/android-проверки не требуются
  (по промпту).
- Уборка: go-прогоны завершены, Gradle-демоны не запускались, `pgrep` по
  tnp-node/tnp-testnet/gradle — чисто.

## 10. CI

**CI pending** — номера прогонов workflows (core, net, cons, flutter, android)
будут известны после пуша ветки; сессия CI-статусы не ждёт (правило мандата).
Merge в main — только после CI-verify владельцем (`prompts/ci-verify-prompt.md`)
и подтверждения владельца.

## 11. Изменённые файлы

- `core/wire/wire.go` — лимиты декодера (глубина 16, guardians/approvals 16).
- `core/wire/genesis_decode.go` — лимиты genesis-декодеров (seed_circle 1024,
  slot_tiers 16, mutable_bounds 64, providers 256, modules 16).
- `core/attest/attest.go` — лимит цепочки аттестации (16 сертификатов).
- `core/wire/fuzz_test.go` (новый) — FuzzDecodeTransaction,
  FuzzDecodeGenesisPackage, FuzzDecodeStateLeaf.
- `net/fuzz_test.go` (новый) — FuzzNetFrame.
- `core/attest/fuzz_test.go` (новый) — FuzzAttestVerify, FuzzParseKeymint.
- `core/ledger/property_debts_test.go` (новый) — property-тесты с демереджем
  и консистентностью индексов.
- `core/store/sqlite_test.go` (новый) — юнит-тесты хранилища.
- `core/wire/proto_conformance_test.go` (новый) — сверка proto ↔ кодек.
- `proto/core.proto`, `proto/transactions.proto`, `proto/genesis.proto`,
  `proto/attest.proto` — синхронизация с кодеком (дрейф, раздел 5).
- `reports/REPORT-FUZZ-DEBTS.md` — этот отчёт.
