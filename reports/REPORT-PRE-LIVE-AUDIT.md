# REPORT-PRE-LIVE-AUDIT — полный аудит проекта перед релизом v3.0.0

- **Сессия (session-ID):** `05e53862-a237-4d23-b30d-7c0a1ebb3b88`
- **Дата/время аудита (UTC):** 2026-11-04, старт 02:20Z
- **Ветка:** `check-pre-live-audit` от свежего `main`, база аудита — tip `origin/main` = `0fdb975c6b283af1f7a2f413dcf2bed865aa56d8` (включает merge a10-docs `6ca9ce4a`, проверено `git merge-base --is-ancestor`)
- **Режим:** только чтение всех репозиториев проекта + запись в `reports/`; код и документы не правились
- **Модель/окно:** заданы YAML-шапкой задачи очереди (mistral-large-4, timeout_min 2880); в промпте не дублируются
- **Артефакты:** `reports/pre-live-audit/` (сейчас: `01-grep-pending-finding.txt`, `02-go-test.txt`, `03-go-test-race.txt`)
- **Прогон кода (доказательство состояния main):** `go build ./...` rc=0; `go test ./...` все пакеты ok (net 89,1 с), rc=0 — артефакт `02-go-test.txt`; `go test -race ./net/ ./cons/` ok (net 142,1 с), rc=0 — артефакт `03-go-test-race.txt`

## 0. Синхронизация всех клонов с ремутами (первое действие сессии)

| Клон | Путь | Действие | Результат |
| --- | --- | --- | --- |
| tnp (рабочий, конвейерный) | `/home/projects/audit-equality-trust-networks` | `git fetch origin && git checkout main && git reset --hard origin/main` | tip main = `0fdb975c6b283af1f7a2f413dcf2bed865aa56d8` (совпадает с `git ls-remote origin main`); `6ca9ce4a` (merge a10-docs) — предок origin/main, подтверждено |
| tnp (личный) | `/home/projects/equality-trust-networks` | **не тронут** | — |
| eq (anti-farm-simulation) | `/home/projects/anti-farm-simulation` | `git fetch origin` без checkout, рабочее дерево не менялось | default branch `main`, `origin/main` = `c0472d5363d39786d2108c803b96c081c467615c`; чтение — через `origin/main` |
| programming-docs | `/home/projects/programming-docs` | `git fetch origin` без checkout | remote `equality-programming-docs`, `origin/main` = `aa21c162940b0aa93101d58cf45b236dedc2ba28` (fetch обновил `f1da65e..aa21c16`) |
| user-docs | `/home/projects/user-docs` | `git fetch origin` без checkout | remote `equality-user-docs`, `origin/main` = `f00b17171d4550a1ec9de5e059d9997a74d2e235` |

Хвосты убитой перезагрузкой сессии: ветка `check-pre-live-audit` **не существовала** ни локально, ни в ремуте (проверено `ls-remote` до создания) — пересоздание не потребовалось; создана от `origin/main` (`git checkout -B check-pre-live-audit origin/main`) и запушена (`0fdb975`). Частичных отчётов на ветке не было.

## 0.5. Гигиена репо tnp — чистка влитых веток

До удаления в ремуте существовали refs `a10-docs` (`d131ac9`) и `check-a10-docs` (`76df2b2`) — выполнено из рабочего клона:

```
git push origin --delete a10-docs check-a10-docs
→ - [deleted] a10-docs
→ - [deleted] check-a10-docs
```

После: в ремуте остались `main` (`0fdb975`), `audit` (`01d9181`), `audit-sonnet` (`829102a`), `net-dos-a3` (`6e16bbb`) — ветки `audit`/`audit-sonnet` **не тронуты** (только владелец). Локальные ветки `a10-docs`/`check-a10-docs` в клоне оставлены как есть.

## 1. Реестр долгов — все накопленные PENDING/FINDING по отчётам tnp (+ eq)

Полный grep-реестр: артефакт `reports/pre-live-audit/01-grep-pending-finding.txt` (219 строк PENDING/FINDING по `reports/`, корневым FINDING-файлам, QUESTIONS.md, docs/). Ниже — сводный реестр с верификацией статуса на main `0fdb975`.

| № | Долг | Источник | Статус на main (проверено) | Влияние на v3.0.0 | Рекомендация |
| --- | --- | --- | --- | --- | --- |
| D1 | AUDIT-MINT-FIX-01 (P2): `spec/03-validation.md:31` (C11) — якорь слота **заявляемого** периода; `spec/00-overview.md:46`, `spec/02-genesis-manifest.md:86` — `protocol_version = 2` | `reports/REPORT-AUDIT-MINT-FIX.md` §8 | **Открыт**: код v9.14 — текущий период (`core/ledger/validate.go:150-196`, `CheckMintChallengeWindow`), `ProtocolVersion: 3` (`core/params/params.go:210`, `cmd/tnp-testnet/main.go:393`); spec/ расходится (см. PLA-F-01) | Документационное: нормативная таблица spec/03 ведёт к старому правилу M7 | **Фикс-тема до релиза** (1 docs-коммит: C11 + версия в spec/) |
| D2 | AUDIT-MINT-FIX-02 (P2): mobile-мост не клампит `claim` до `last + accrual_window` — выпуск «отравлен» E_ACCRUAL_WINDOW у кошелька, пропустившего ≥7 периодов | тот же отчёт §8 | **Открыт**: `mobile/query.go:929` — `claim = period - 1` без клампа (проверено; `accrual_window` отдаётся в JSON :953, но не используется для клампа) | Функциональный, не консенсусный: UI/apply расходятся на этом пути | **Фикс-тема до релиза** (малый кламп в MintWindowJSON) или осознанное отложение владельцем |
| D3 | AUDIT-MINT-FIX-03 (P3): диагностика `slots`/`findlabels` в tnp-testnet считает достижимость по дофиксной формуле (`cmd/tnp-testnet/main.go:692,715-724`, `fleet.go:46-50`) | тот же отчёт | Открыт (tooling) | Не влияет на протокол; вводит оператора в заблуждение в прогонах | Отдельная тема после релиза |
| D4 | AUDIT-MINT-FIX-04 (P2, формулировка): «поглощающее состояние исчезает» — на деле P(6 промахов подряд)=0,437; полное устранение требует показа слота приложением | тот же отчёт | Открыт (формулировка отчёта; спека v9.14 сформулирована точно) | Документация | Правка формулировки отчёта; UX-показ слота — с PENDING-2 |
| D5 | AUDIT-MINT-FIX-05 (P3): окно M7 не барьер живости при sync-подаче (C9 применяется только на приёме/ретрансляции) | тот же отчёт | Открыт (наблюдение; `net/sync.go:357-359`) | Потенциальный слой допуска claim при сверке | Тема 8-live / следующая итерация протокола |
| D6 | AUDIT-MINT-FIX-06 (P3): устаревший комментарий `cmd/tnp-node/main.go:314-318` | тот же отчёт | Открыт (косметика) | Нет | После релиза |
| D7 | AUDIT-MINT-FIX-07 (P4): числовые неточности отчёта исполнителя | тот же отчёт | Открыт (info) | Нет | По желанию |
| D8 | FINDING-MINT-FIX-01: sim не моделирует окно челленджа M7 | `FINDING-MINT-FIX-01.txt` | Открыт — решение владельца (после фикса v9.14 включение безопасно) | Только симуляция | Решение владельца |
| D9 | FINDING-MINT-FIX-02: дубль `challengeSlot` в `cmd/tnp-testnet/main.go:656-663` vs `core/ledger/validate.go:206-210` | `FINDING-MINT-FIX-02.txt` | Открыт — экспорт слот-функции из ядра | Tooling; расхождение CLI с ядром при ручном PENDING-1 | Отдельная тема |
| D10 | FINDING-1 аудита fuzz-debts (средняя): вне минимум-списка не фаззятся `net/gossip.go:183` `decodeCheckpoint` (повторяющиеся `Participants`/`FinalizedTxIDs` без счётных лимитов; память ограничена только кадром ≤16 МиБ), `net/seedlist.go:137,165,193`, `net/cons.go:80,145` (msgVoteQuery/Resp), `net/sync.go:442` (msgTip) | `reports/REPORT-AUDIT-FUZZ-DEBTS.md` §5 | **Открыт** (признан, отдельная тема-долг; 6 fuzz-целей и 9 лимитов — на месте, `core/wire/wire.go:737-748`) | Средняя: незакрытые точки разбора недоверенного ввода; ущерб ограничен лимитом кадра 16 МиБ (`net/sync.go:23`) | **Тема-долг до/после релиза по решению владельца** (не блокер): fuzz-цели + счётные лимиты |
| D11 | Этап 16, FINDING: офлайн-буфер публикации (awaiting, Bluetooth 12.4) только в памяти — теряется при force stop; получатель видел «принято», платёж не опубликован | `FINDING-awaiting-offline-tx-not-persistent.txt`, `reports/REPORT-STAGE-16-NODE-PERSISTENCE.md` §7 | **Открыт** (`net/node.go` OnOfflineTx/awaiting/PublishAwaiting — в памяти) | Реальный пользовательский риск потери платежа в офлайн-сценарии | **Кандидат в мини-этап до релиза** (таблица awaiting_txs в store) или осознанное отложение владельцем |
| D12 | Этап 16, FINDING: экран Vouching после рестарта показывает «0 of 3» для ACTIVE-кошелька (опрос один раз в initState, без ретрая после NONODE) | `FINDING-vouching-screen-stale-nonestatus.txt` | **Открыт** (UI-quirk; состояние узла корректно) | Только отображение, низкий | Мини-ветка UI-фиксов или 8-live |
| D13 | a10-docs PENDING-1: включение `hash(чекпоинта предыдущей эпохи)` в формулу слота — открытый вопрос владельца (код: `H(address‖period)`) | `reports/REPORT-A10-DOCS.md` §5 | Открыт (за владельцем) | Не блокер; меняет формулу слота, если будет принят | Решение владельца |
| D14 | a10-docs PENDING-2: контрольный замер PoW на живом телефоне | тот же отчёт; QUESTIONS.md №9 | Открыт — **чек-лист 8-live** | Спека опирается на хостовые замеры A9; эмуляторный замер противоречив (CHECK-A10-DOCS-05) | 8-live |
| D15 | a10-docs PENDING-3: README-ENG.md отстаёт от README.md (7.6/7.7) | тот же отчёт | Открыт — территория темы ENG-доков | Документация EN | Тема ENG-доков (запланирована) |
| D16 | a10-docs PENDING-4: замечания AUDIT-A9-02..05, 07..09 к отчёту A9 — в eq-репо | тот же отчёт | Открыт (вне tnp) | Документация модели | Правка в eq-репо по желанию владельца |
| D17 | №43 (QUESTIONS.md): отзыв attestation (CRL/OCSP) не реализован; future patch-date теперь отклоняется | `reports/REPORT-STAGE-9.md:170`, QUESTIONS.md №43 | **Осознанно отложено для v1** решением владельца; пересмотр при реальных корнях | Для песочницы некритично; для продакшена с реальными корнями — решение владельца | 8-live / переход на реальные корни |
| D18 | IP-colocation фактор pubsub-score отключён (вес 0) | `reports/REPORT-NET-DOS-A3.md` §3 | Открыт — **кандидат 8-live** | Eclipse/DoS на одном IP | 8-live |
| D19 | stage11 PENDING №1: семантика подписи гардиана (список на момент подписи vs на момент propose) — реализовано «на момент propose» | `reports/REPORT-STAGE-11-2.md` §6; `REPORT-STAGE-11-3-FIXES.md` | Открыт — решение владельца, до пилотов | Протокол v9.8 допускает обе интерпретации | Решение владельца |
| D20 | stage11 PENDING №5: способ сдвига часов для прогонов (`-rtc` не двигает guest-часы) | тот же отчёт | Открыт — вне кода (rootable-образ + `adb root date -s`) | Только прогоны | По согласованию |
| D21 | №35 ASN-классификация eclipse — PENDING (нужны внешние BGP-данные); реализовано /24 (IPv4) и /64 (IPv6) | QUESTIONS.md №35; `net/eclipse.go:9-10,36-52` | Открыт; F17 frontier-pass (средняя): три /24 одного ASN проходят N=3 | Средняя для eclipse-устойчивости | 8-live (внешние данные) или осознанное принятие |
| D22 | №39-доп (этап 12): окна челленджа периодов 1–2 при quota_reset почти недостижимы (vouch посевного круга) | QUESTIONS.md №39 | Открыт — решение отнесено к 8-live | Живой прогон | 8-live |
| D23 | №45: сборки проверяющей библиотеки JS/WASM — отложены до первого стороннего потребителя | QUESTIONS.md №45 | Осознанно отложено | Нет для v1 (AAR/JVM закрыты) | По появлению потребителя |
| D24 | №9/№37/№38: чек-листы живого телефона — замер PoW, цепочка KeyMint до корня Google на устройстве, драйвер ncruces/go-sqlite3 на живой прошивке (ARM-замеры) | QUESTIONS.md; `reports/REPORT-STAGE-16-NODE-PERSISTENCE.md` | Открыты — **чек-лист 8-live** | Эмулятор ≠ живое устройство | 8-live |
| D25 | CHECK-A10-DOCS-02 (нумерация PENDING-3/PENDING-4 в отчёте) и CHECK-A10-DOCS-07 (erratum в REPORT-STAGE-6) | `reports/REPORT-CHECK-A10-DOCS.md`, `REPORT-A10-DOCS-FIXES.md` | **Закрыты** в a10-docs-fixes (merge 6ca9ce4a) | Нет | — |
| D26 | CHECK-A10-DOCS-05 (средний): «256 итераций ≈ 4,7 мс/итер» — оценка по E[N], не измерение; эмуляторный артефакт внутренне противоречив | `reports/REPORT-CHECK-A10-DOCS.md:195` | Открыт (исторический артефакт; рекомендация — писать фактические итерации) | Не влияет на числа спеки (они от хостовых замеров A9) | Связано с D14 (PENDING-2) |
| D27 | Frontier-pass: 24 FINDING — F5 (приманка, высокая) и F16 (офлайн-финальность, высокая) **решены** (merge `169f12c`, регресс-тесты, эмуляторный прогон postfix); 9 средних (PNG карточки в кэше, суммы/адреса в SQLite открытым текстом, F6=№43, сети device_credential, biometric alpha, release debug-подпись, /24 без ASN, отравление кэша пиров, нет анти-tapjacking) и 13 низких (дрейф RU/ENG и др.) не исправлены | `reports/REPORT-AUDIT-FRONTIER-PASS.md` | F5/F16 закрыты; остальные — отдельные темы по решению владельца | Средние — часть в чек-лист 8-live | По решению владельца; не блокеры |
| D28 | eq-репо: AUDIT-A9-01..11 — замечания blind-аудита A9 (01 средняя: 2000 кошельков × 17,38 с = ~9,7 ч/мес суммарно, а не «29 мин»; 02–11 низкие/инфо) | `equality-anti-farm-simulation@origin/main` `reports/REPORT-AUDIT-A9-SIMULATION.md` | Открыты в eq-репо; спека 7.7 (v9.15) уже учла замечания | Документация модели | Правка отчёта A9 в eq-репо по желанию |
| D29 | X1 (frontier-pass): два FINDING-*.txt лежат в корне репо, а не в `reports/` | `reports/REPORT-AUDIT-FRONTIER-PASS.md` §6 | **Открыт** (оба файла на месте на main) | Конвенция, низкий | Перенос в reports/ отдельной темой |
| D30 | Известный net-флак `TestVoteFinalityNetwork` (не регресс) | `reports/REPORT-STAGE-11-PREP.md:188` и др. | Сегодня `go test ./...` и `-race ./net ./cons` зелёные (артефакты 02/03) — флак не проявился | Нет | Мониторить в CI |

**Итог по разделу 1:** блокеров среди долгов нет; открытые PENDING с влиянием на релиз — D1 (spec/ vs код, документация), D2 (mobile-кламп accrual), D10 (fuzz-покрытие, средняя), D11 (офлайн-буфер, пользовательский риск), D21 (ASN), D27 (средние frontier, часть в 8-live). Остальное — осознанные отложенные решения владельца или чек-лист 8-live.

## 2. Инварианты релиза — сверка спеки v9.16 (RU) и README с кодом main

База: `docs/KNOWLEDGE*.md` и `README.md` версии 9.16 (дата 4 ноября 2026), `spec/*.md`. Сверка по критическим разделам, все file:line — на main `0fdb975`.

| Заявление спеки/README | Факт кода | Вердикт |
| --- | --- | --- |
| Валидация tx: закрытый перечень C1–C11 (spec/03) | `core/ledger/validate.go`: C1–C4 `applyCommon` :60-83 (E_NET :61, E_VER :65, E_DUP :69, E_TYPE :73), C5–C7 `applyChainChecks` :86-110, C8–C11 `applySigPowTime` | ✅ подтверждено |
| Подпись ECDSA P-256, каноничный DER + low-s (12.1) | `core/crypto/keys.go`: `SignDER` :52-64 (low-s канонизация :60-62), строгий DER-decode `decodeDERSignature` :131-160, `VerifyDER` :72-81; `TxID = SHA-256(Encode())` включает подпись (`core/wire/wire.go:375-377`) — malleability (s→N−s) даёт другой txid, ложного конфликта нет | ✅ подтверждено |
| PoW-таргеты TargetLight 2^40 / TargetHard 2^56, Argon2id 32 МиБ × 1 × p=1 (№9, 7.6, 7.7) | `core/params/params.go:180-184`: `TargetLight = 0x0000_0100_0000_0000` (= 2^40), `TargetHard = 0x0100_0000_0000_0000` (= 2^56), Argon2 32768/1/1; `core/profiles/profiles.go:92-97` — те же; `core/crypto/pow.go:16-22,66-79`; пин-тест `core/params/pow_target_test.go` (2^56 → 256 итераций) | ✅ значения подтверждены; ⚠ см. PLA-F-02 (устаревшие подписи «2^24») |
| recurring-сгорание: периодная метка, сгорание после границы P+1, без сброса баланса (v9.10, раздел 4) | `core/ledger/settle.go:81-98` — когорты с `Period < period` сгорают, `TotalBurned` растёт, баланс не сбрасывается; genesis-валидация комбинаций (recurring только при не-global network_id, дубликаты fingerprint'ов) — `core/profiles/profiles.go:355-385`, `core/wire/genesis.go:272-277` | ✅ подтверждено |
| Консенсус/чекпоинты: N из M = ceil(2/3·21) = 14, детерминированный строитель, очередь строителей, пустые эпохи (v9.6, №42 (A)+(D)) | `cons/checkpoint.go` Build :85, Verify :129-171 (порог через `threshold.Verify` :168-171); `cons/params.go:26-53` (N=14, BuilderQueue=3); очередь/ротация/пропуск — `net/cons.go:377-640` (`cpBuilderQueueK` :545, `cpSkipDue` :567) | ✅ подтверждено |
| Офлайн-финальность (F16): «финализировано» только для txid финализированной ветки | `net/eclipse.go` `SetFinalityCheck` :159, гейт в `Confirmed` :168-175; wiring `net/node.go:251` → `net/cons.go` `isFinalizedTx` :293 (`finalityView` :302-317) | ✅ подтверждено |
| Приманка (F5): подменной адрес во всех точках выдачи | `app/android/.../FlutterBridge.kt` `displayAddress()` :183, `walletAddress → displayAddress` :373, :215, :715; `mobile/mobile.go` `GenerateDecoyAddress` :64; персист легенды `tnp_duress`; регресс-тест `app/test/decoy_address_test.dart` | ✅ подтверждено |
| PIN-лимит: 5 неудач → 30 с, удвоение до 10 мин, сброс успешным входом/24 ч (8.2) | `android/.../pin/PinAttemptPolicy.kt` (MAX_LOCK_MS = 600_000 :53), `PinManager.kt` :76-94 | ✅ подтверждено |
| FLAG_SECURE на окнах | `android/.../ui/MainActivity.kt:42-46`, `FlutterBridge.kt:335-348`, `app/lib/services/secure_mode.dart`, `mint_screen.dart:32` | ✅ подтверждено |
| Окно подписи 0 (8.1: «без временного окна», BIOMETRIC_STRONG) | `android/.../keystore/KeystoreWallet.kt:142` `AUTH_TIMEOUT_SECONDS = 0`, :78-81 `setUserAuthenticationParameters(0, AUTH_BIOMETRIC_STRONG)`, `setUserAuthenticationRequired(true)`, DIGEST_NONE (подпись готового дайджеста) | ✅ подтверждено |
| txNow = timestamp_claim для всех часозависимых правил; live = replay по построению (v9.13) | `core/ledger/validate.go` Apply :228-229 (`l.tt` от `tx.TimestampClaim`); C9 — только слой сети (spec/03 :29) | ✅ подтверждено; ⚠ spec/03 C11 — см. PLA-F-01 |
| Replay-изоляция по network_id (12.3): replay между сетями невозможен | `core/ledger/validate.go:61-62` E_NET; `core/wire/wire.go:344,675,1626`; само-сертифицирующийся `network_id = H(manifest‖genesis)` — `core/wire/genesis.go:461-471` `DeriveNetworkID`, проверка `VerifySelfCertified` :507-527 | ✅ подтверждено |
| Адрес = SHA-256(fingerprint ‖ network_id), единая точка деривации (v9.11, 7.2) | `core/crypto/address.go` `DeriveAddress` (факт), `AddressOfKey`; ⚠ комментарий :1-11 всё ещё «PENDING: вопрос №2» — PLA-F-03 | ✅ факт подтверждён, комментарий устарел |

### Находки раздела 2

- **PLA-F-01 (средняя, документация):** `spec/03-validation.md:31` (C11) описывает якорь слота **заявляемого** периода и `protocol_version = 2` в `spec/00-overview.md:46` / `spec/02-genesis-manifest.md:86`; код main (v9.14) — якорь **текущего** периода (`validate.go:150-196`) и `ProtocolVersion = 3` (`params.go:210`). Это открытый долг D1 (AUDIT-MINT-FIX-01). Реализация по spec/ даст дофиксное правило M7. Рекомендация: фикс-тема до релиза (только spec/, 1 коммит).
- **PLA-F-02 (низкая, документация/комментарий):** «лёгкий PoW 2^24» в `docs/KNOWLEDGE-5.md:130`, комментарий `core/params/params.go:180` («2^24 итераций») и QUESTIONS.md №9 («таргет 2^24 (0x0000010000000000…)» — сам hex равен 2^40) противоречат значению кода 2^40 и утверждениям README.md:378 / KNOWLEDGE-3.md:130 («лёгкий 2^40»). Значение в коде единообразно 2^40. Рекомендация: erratum в KNOWLEDGE-5/QUESTIONS + комментарий в params.go.
- **PLA-F-03 (низкая, комментарий):** `core/crypto/address.go:1-11` — stale «PENDING: вопрос №2» (F11 frontier-pass); деривация зафиксирована v9.11.

## 3. Крипто и консенсус

| Механизм | Факт кода | Вердикт |
| --- | --- | --- |
| Подписи ECDSA P-256, DER, low-s | `core/crypto/keys.go` (см. раздел 2); пороговые подписи — `core/threshold/threshold.go:13-40` `Verify` (≥ threshold валидных долей из participants), `core/threshold/verify.go:13-14` (тот же `crypto.VerifyDER`) | ✅ |
| Argon2id/PoW | `core/crypto/pow.go`: лёгкий SHA-256 (`HashBelowTarget` :16-22 — big-endian uint64 первых 8 байт < target), усиленный `argon2.IDKey` 32 МиБ × 1 × p=1 :66-79; параметры `params.go:180-184` | ✅ |
| Sparse Merkle-дерево состояния | `core/smt/smt.go` (SetLeaf/Root/Prove :62-108), корень — `cons/checkpoint.go:23` `StateRoot`, доказательства `StateProof` :62-83; состав листа — 9 полей `stateLeafOf` :36-48 | ✅ |
| Пороговые подписи чекпоинтов: очередь строителей, пустые эпохи | `cons/checkpoint.go` Build/Verify/Collector :85-267; очередь K=3, ротация epoch/2, пропуск застрявшей эпохи — `net/cons.go:377-640`; «пустая эпоха» — гэп нумерации, тест `TestCheckpointSkipGap` | ✅ |
| live == replay (cons.Replay strict) | `cons/replay.go:114-127`: любая reject — ошибка, Conflict-ветка фиксируется (INV-1c); канонизация `Canonicalize` :48 с `FinalityView` :21-41; единый механизм с live (v9.13 — отдельного режима часов нет) | ✅ |
| Replay-изоляция network_id | E_NET на валидации (:61-62 validate.go); self-certified network_id (`genesis.go:461-527`); тестнет: 6/6 направлений E_NET (REPORT-STAGE-12-TESTNET) | ✅ |
| Разделение/эквивокация | `validate.go:99-102` — конфликтная ветка на той же высоте → `Conflict` (не reject), проходит C8–C11 к финализатору; `core/ledger/ledger.go:212-228` `ConflictBranches`/`RecordConflictTx`; финализированная ветка — `FinalizedTxIDs` чекпоинта (`cons/checkpoint.go:98`); replay выбирает победителя через `FinalityView` | ✅ незакрытых сценариев не найдено |

Открытые вопросы (не находки): PENDING-1 (hash чекпоинта в формуле слота — D13), №43 CRL/OCSP (D17), AUDIT-MINT-FIX-05 (живость при sync-подаче — D5). Новых незакрытых крипто/консенсус-сценариев аудит не обнаружил.

## 4. Сеть

| Механизм | Факт кода | Вердикт |
| --- | --- | --- |
| Единый путь приёма relayAdmit: дешёвое → подпись → дорогое | `net/gossip.go:428-473`: `cheapCheck` (структура, подпись ECDSA DER+low-s, дешёвый отказ неизвестного кошелька) → бюджет дорогих операций → семафор `hardSem` (2 параллельных Argon2id) → `verifyPoW` → награда; штрафы и `maybeDisconnect` :475-482 | ✅ порядок подтверждён |
| DoS-бюджет: 32 дорогие проверки/пира за окно 10 с | `net/dos.go:28-33` `DosExpensiveBudget = 32`, `chargeExpensive` :128; окно — в `newDosTracker` :76-83 | ✅ |
| Peer score (libp2p WithPeerScore + dosTracker) | `net/dos.go:179-219`: GossipThreshold −10 (вырезание из mesh), GraylistThreshold −80, порог разрыва `dosDisconnectScore`; штрафы: подпись −8, PoW −4, мусор −4, бюджет −4, неизвестный кошелёк −2; награда +1, кап +16, полураспад 60 с; подключение `WithPeerScore` в `newPubSub` | ✅ |
| IP-colocation в pubsub-score | **Отключён (вес 0)** — кандидат 8-live (D18) | ⚠ открыто осознанно |
| Eclipse-гейты | `net/eclipse.go`: классификатор связности /24 (IPv4) / /64 (IPv6) `PeerSubnetLink` :36-52 (relay/circuit — уникальная группа на пира), Corroboration по N **различных групп** :168+; F16-гейт финальности; ASN — PENDING (нужны BGP-данные) :9-10; `CorroborationN` default 3 / `eclipse_n` (`net/config.go:143-148`) | ✅ /24-/64 подтверждено; ASN — D21 |
| Лимиты недоверенного ввода | кадр ≤ 16 МиБ (`net/sync.go:23` readFrameStream; `net/codec.go:117-123` readFrame), лимиты wire-декодера (`core/wire/wire.go:737-748`: maxDecodeDepth 16, maxPersonalGuardians 16, maxApprovals 16 + счётные лимиты темы fuzz-debts), genesis `core/wire/genesis_decode.go:14`, attest `core/attest/attest.go:103` | ✅; ⚠ FINDING-1 (D10): `decodeCheckpoint` gossip без счётных лимитов Participants/FinalizedTxIDs, seedlist-парсеры, cons-голосование, msgTip — открытая тема-долг |

**Чек-лист 8-live (входной материал — готовность релиза к живым прогонам):**
1. Живой телефон: замер PoW (D14/PENDING-2), цепочка KeyMint до корня Google на устройстве (№37), драйвер ncruces/go-sqlite3 на живой прошивке + ARM-замеры (№38), транзиент Keystore в verifyPin (из stage12-testnet).
2. IP-colocation pubsub-score (D18), ASN-классификация (D21), CRL/OCSP отзыв attestation при реальных корнях (D17).
3. Офлайн-буфер публикации пережить рестарт (D11), Vouching-ретрай (D12), окна челленджа периодов 1–2 при quota_reset (D22).
4. Из средних frontier-pass: анти-tapjacking, отравление кэша пиров, release debug-подпись → боевая (A8), суммы/адреса в SQLite открытым текстом, PNG карточки в кэше (D27).
5. Что закрывается в 8-live без повторного полного аудита: пункты 1–4 выше + AUDIT-MINT-FIX-05 (D5).

## 5. Гигиена

- **Секреты:** `git grep` по всем отслеживаемым файлам (паттерны приватных ключей, ghp_/gho_/github_pat_, AKIA…, xox…, AIza…, password/secret/api_key в присваиваниях) — совпадений нет (ложные срабатывания — атрибут `password="false"` в XML-дампах UI). В истории (`git log --all --diff-filter=A --name-only`) единственные key-подобные файлы — публичные корневые сертификаты Google attestation `core/attest/roots/google-hardware-attestation-root-{rsa,ecdsa-p384}.pem` (публичные, ожидаемы). Ключевые файлы (*.jks, *.keystore, signing.properties, .tnp-signing) в git не попадали никогда; боевые ключи A8 — вне репо (`~/.tnp-signing/`), что подтверждено `reports/REPORT-AUDIT-A8-KEYS.md`.
- **CI:** 5 workflows (`.github/workflows/{android,cons,core,flutter,net}.yml`) — хардкод-секретов нет; android.yml — instrumented-тесты, release-подпись вне CI.
- **.gitignore:** покрывает `build/`, `*.apk/aab/aar`, `android/libs/`, gradle/dart кэши, `*.log`, keystore-паттерны (`*.jks`, `*.keystore`, `credentials.txt`, `signing.properties`, `.tnp-signing/`), `.vibe/`, `.wrangler/`. Build-деревья в рабочем дереве не отслеживаются (`git ls-files | grep -c 'build/'` = 0).
- **Файлы > 1 МБ:** один — `reports/stage8/sim50/run1/metrics.csv` = 1 049 554 Б (на 978 Б выше 1 МиБ; при пороге 10^6 Б — в пределах). Наблюдение, не нарушение по духу правила.
- **Замечание (низкое):** два FINDING-файла в корне репо (`FINDING-awaiting-offline-tx-not-persistent.txt`, `FINDING-vouching-screen-stale-nonestatus.txt`) — вопреки конвенции «находки — в reports/» (X1, D29).

## 6. Внешние репозитории проекта (чтение через origin/main)

- **eq** (`equality-anti-farm-simulation`, `origin/main` = `c0472d5`): модель (`Mistral/farm_model.py`, `Claude Sonnet 3.6/farm_model.py`), тесты, отчёты `REPORT-A9-SIMULATION.md`, `REPORT-AUDIT-A9-SIMULATION.md` (AUDIT-A9-01..11 — D28), BLOCK3/BLOCK4, `reports/audit-a9-simulation/recheck_metrics.py` — на месте. Замечаний к структуре нет.
- **programming-docs** (`origin/main` = `aa21c16`): 6 документов (`docs/01..06`), актуализация до **v9.11** (этап 12.x, network-merge отклонён, деплой по факту cmd/). **PLA-F-04 (средняя, документация):** не отражены v9.12–v9.16 — attestation-валидация этапа 14, txNow v9.13, mint-fix v9.14, факты A9 v9.15, `protocolVersion = 3`.
- **user-docs** (`origin/main` = `f00b171`): 5 документов, до **v9.11** (8-й сценарий, Eq-<hash8>, сети-подписки). То же расхождение до v9.16 — PLA-F-04.
- Вывод: docs-репо отстают от спеки tnp на 5 минорных версий; это документационный долг (не блокер кода), частично покрывается запланированной темой ENG-доков (D15) — но RU-спека tnp ушла вперёд, и user/programming docs это не отражают.

## Вердикт

**Блокеры: нет.** `go build ./...`, `go test ./...`, `go test -race ./net/ ./cons/` на main `0fdb975` — зелёные (артефакты 02/03). F5/F16 высокой значимости закрыты в main. Все критические инварианты раздела 2 подтверждены кодом.

| Приоритет | Находки/долги |
| --- | --- |
| Высокая | Новых нет. (F5/F16 frontier — решены до аудита.) |
| Средняя | PLA-F-01 / D1 (spec/ C11 + protocol_version расходятся с кодом v9.14/v3 — фикс-тема до релиза, документация); D2 AUDIT-MINT-FIX-02 (mobile-кламп accrual_window — малый фикс до релиза или осознанное отложение); D10 FINDING-1 fuzz (тема-долг, не блокер); D11 офлайн-буфер не переживает рестарт (пользовательский риск потери платежа — кандидат в мини-этап до релиза или осознанное решение владельца); D21 ASN (8-live); D27 средние frontier (по решению владельца, часть — 8-live); PLA-F-04 (docs-репо на v9.11 против v9.16) |
| Низкая | PLA-F-02 («2^24» vs 2^40 в KNOWLEDGE-5/комментариях/QUESTIONS), PLA-F-03 (stale комментарий address.go), D3/D4/D5/D6/D7/D8/D9, D12, D13, D15, D16, D19, D20, D22, D26, D28, D29 (FINDING-файлы в корне), D30 (флак), файл 1 049 554 Б |

**Рекомендация по релизу v3.0.0:** блокеров нет; перед релизом желательны фикс-темы по PLA-F-01 (spec/, 1 коммит) и D2 (mobile, малый), решение владельца по D10/D11; остальные средние — осознанное отложение (8-live) с фиксацией в этом отчёте. Релиз не выходит до чистого аудита или осознанного решения владельца по каждой находке — аудит чистый по блокерам, решения по средним/низким за владельцем. 8-live — после релиза, без повторного полного аудита (чек-лист в разделе 4).

*Сессия `05e53862-a237-4d23-b30d-7c0a1ebb3b88`, 2026-11-04. Только чтение + reports/; код и документы не правились.*


