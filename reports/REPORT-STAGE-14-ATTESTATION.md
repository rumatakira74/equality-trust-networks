# Отчёт этапа 14 — аттестация: привязка листа и корни Google

Ветка: `stage14-attestation` (от `main` @ `6f1a144`). Основание: аудит от 01.11.2026
(`reports/AUDIT-equality-trust-networks.md`), находки A1 (критическая), A2 (высокая), A12 (низкая).
Спека: v9.11 → v9.12 (перечень правок в шапке `docs/KNOWLEDGE.md` / `docs/KNOWLEDGE-ENG.md`).
`protocolVersion` НЕ поднимался: формат транзакций и genesis не меняется, меняется только валидация.

## 1. A1 — привязка листа (критическая)

**Что:** регистрируемый ключ обязан совпадать с ключом сертификата-листа цепочки аттестации.
До фикса цепочка, полученная для одного ключа, регистрировала любой другой ключ —
фингерпринт и адрес считались бы от неподтверждённого ключа.

**Где/как:** `core/ledger/validate.go`, `applyRegister` — новая проверка R4c после R2/R3/R4/R4b:
SPKI листа (`x509.MarshalPKIXPublicKey(chain.Leaf.PublicKey)`) сравнивается байт-в-байт с SPKI
регистрируемого `AttestedPublicKey`. Несовпадение — reject `E_ATTEST_KEY_MISMATCH` (новый код
протокола). Проверка выполняется у каждого узла при apply; replay воспроизводит тот же вердикт
(проверка детерминирована от данных транзакции, не от часов). Идентичность адресов не менялась:
fingerprint остаётся от ключа, совпадающего с листом.

**Мост и Kotlin:** экспортируемый API моста не менялся. Честный Android-путь не сломан:
`KeystoreWallet.generateWithAttestation` создаёт ключ и возвращает цепочку, лист которой содержит
тот же ключ, что `publicKey()` (uncompressed point P-256 → SPKI того же ключа);
`WalletFlow.buildRegisterTx` кладёт в `register` ровно тот ключ, для которого получена цепочка.
Сборка Kotlin-модуля зелёная (см. проверку). Прогон на живом устройстве не выполнялся
(в мандате этапа эмуляторного прогона нет) — DISCLAIMER сохраняется.

## 2. A2 — корни Google (высокая)

**Что:** прод-путь аттестации доверяет только настоящим корням Google hardware attestation —
встроены ОБА корня; детерминированный синтетический корень
`DeterministicKey("tnp-android-root:" + network_id)` убран из прод-пути.

**Где/как:**
- `core/attest/roots.go` + `core/attest/roots/*.pem` (go:embed): `attest.GoogleRoots()` — оба
  корня; цепочка валидна, если доходит до любого из них. Ротация корня — новый PEM-файл в
  `core/attest/roots/` и коммит.
- `mobile/mobile.go`: `Genesis.TrustedRootsDER()` возвращает синтетический корень ТОЛЬКО для
  genesis с `sandbox = true` (иначе nil); `attestRoots()` — общий выбор корней узла:
  песочница → синтетический корень genesis, не-sandbox (прод) → `attest.GoogleRoots()`
  (используется всеми тремя StartNode*).
- `cmd/tnp-node/main.go`: демо-genesis (`testGenesis`) стал `sandbox = true` — выравнивание с
  `mobile.SandboxGenesis` приложения (эмуляторная песочница); для не-sandbox genesis-файла узел
  берёт корни Google.
- `net/node.go`, `NewNode`: не-sandbox genesis с синтетическим корнем в `TrustedRoots` — узел
  не стартует (genesis-валидация конфигурации). Тестовый обход — `NodeConfig.AllowSyntheticRoots`
  (только unit-тесты и headless-утилиты; прод-путь моста не выставляет).
- `cmd/tnp-testnet/main.go`: закрытая тестовая сеть регистрирует кошельки синтетическими
  цепочками при `sandbox = false` намеренно (тестирует реальное поведение: мораторий,
  отложенные переводы, окно челленджа) — переведена на test-only внедрение
  (`AllowSyntheticRoots = true`), прод-валидация не ослаблена.
- `net/helpers_test.go`: тестовое окружение — test-only флаг (синтетика при sandbox = false).
- Синтетические корни распознаются по CN-префиксу «TNP Synthetic Root»
  (`attest.IsSyntheticRoot`) — единый признак для всех синтетических корней (песочница,
  векторы, утилиты).

**Сертификаты — только из официального источника** (страница Key Attestation, раздел
Root certificates; из памяти модели не воспроизводились):

| Корень | Источник (URL) | SHA-256(DER) | Срок |
| --- | --- | --- | --- |
| RSA-4096, serial `f92009e853b6b045` | https://developer.android.com/privacy-and-security/security-key-attestation | `cedb1cb6dc896ae5ec797348bce9286753c2b38ee71ce0fbe34a9a1248800dfc` | 2022-03-20 … 2042-03-15 |
| ECDSA P-384, CN `Key Attestation CA1` (Remote Key Provisioning) | https://developer.android.com/privacy-and-security/security-key-attestation | `6d9db4ce6c5c0b293166d08986e05774a8776ceb525d9e4329520de12ba4bcc0` | 2025-07-17 … 2035-07-15 |

Оба SHA-256 закреплены юнит-тестом (`core/attest/roots_test.go`): подмена PEM-файла
обнаруживается тестом. Примечание: официальная страница сообщает, что новый ECDSA-корень
начинает подписывать цепочки аттестации с 01.02.2026 (в мандате упоминалась дата 10.04.2026
для Remote Key Provisioning); встроены оба корня независимо от дат — расхождение дат
не влияет на валидацию.

## 3. Срок цепочки при replay

**Что:** валидность листа (NotBefore/NotAfter) проверяется на момент `tx.TimestampClaim`, а не
на часы узла: листы короткоживущие (~2 мес), replay старых регистраций обязан давать тот же
вердикт, что live-apply.

**Где/как:** `core/ledger/validate.go`, `applyRegister` — `attest.Verify` вызывается с
`now = time.Unix(tx.TimestampClaim)`. Этому же моменту остаётся привязанной проверка свежести
патча. `TimestampClaim` ограничен допуском ±30 мин (C9), поэтому live-проверка и replay
(пинирование часов на `timestamp_claim`, `cons.Replay`) сходятся. Более широкий txNow-рефакторинг
(A7) — предмет этапа фикса времени, НЕ тронут.

## 4. A12 — target = 0

**Что:** вне песочницы молчаливое отключение PoW запрещено: genesis с нулевым таргетом
(«выключено», `HashBelowTarget`) или без параметров PoW — невалиден. В песочнице допустимо
(симуляции).

**Где/как:** `core/ledger/ledger.go`, `New` — правило 2.6 (7): `!sandbox && (Pow == nil ||
TargetLight == 0 || TargetHard == 0)` → ошибка genesis. Спека: `spec/02-genesis-manifest.md`,
правило 6b.

## 5. Честная спека (v9.12, RU+ENG синхронно)

- `docs/KNOWLEDGE-3.md` / `docs/KNOWLEDGE-ENG-3.md` — 7.1: аттестация подтверждает настоящее
  устройство с защищённым железом, но НЕ уникальность устройства — после привязки листа одно
  устройство может выпустить несколько attested-ключей (несколько кошельков на устройстве
  возможны и после фикса), основной барьер — поручительства; 7.2: оба корня Google, привязка
  листа, срок по `timestamp_claim`, честный DISCLAIMER; 7.6: «ферма = ферма дорогих устройств»
  выполняется не автоматически — барьер держат поручительства и стоимость.
- `docs/KNOWLEDGE-1.md` / `docs/KNOWLEDGE-ENG-1.md` — 3.3: строгая аттестация теперь
  обеспечена механикой (корни Google + привязка листа), но без замера на живых устройствах
  уровень гарантий описывает проверенную механику, а не измеренный результат (DISCLAIMER остаётся).
- `docs/KNOWLEDGE.md` / `docs/KNOWLEDGE-ENG.md` — версия 9.12 + запись в шапке.
- `README.md` / `README-ENG.md` — только часть об аттестации (7.1/7.2).
- `spec/00-overview.md` (код `E_ATTEST_KEY_MISMATCH`), `spec/03-validation.md` (R2 — срок по
  `timestamp_claim`; новая строка R4c), `spec/02-genesis-manifest.md` (правило 6b) — таблицы
  валидации приведены к фактическому поведению.
- Поля 61/51 и семантика M4/M5 — НЕ в этом этапе (заходом с фиксом времени), не тронуты.

## Таблица новых проверок и ошибок протокола

| Проверка | Место | Результат нарушения |
| --- | --- | --- |
| R4c: SPKI листа = `attested_public_key` | `core/ledger/validate.go` (applyRegister) | `E_ATTEST_KEY_MISMATCH` (новый код) |
| Срок листа на момент `timestamp_claim` (NotBefore ≤ claim ≤ NotAfter) | `core/ledger/validate.go` (applyRegister → attest.Verify) | `E_ATTEST` (strict) / `E_ATTEST_POLICY` |
| Синтетический корень вне песочницы | `net/node.go` (NewNode) | узел не стартует (невалидная конфигурация) |
| Нулевой/отсутствующий PoW-таргет вне песочницы | `core/ledger/ledger.go` (New, 2.6 (7)) | genesis невалиден |
| Встроены оба корня Google | `core/attest/roots.go` | — (набор закреплён SHA-256 в тесте) |

## Тесты

Негативные (все зелёные):
- валидная цепочка + `AttestedPublicKey` ≠ SPKI листа → `E_ATTEST_KEY_MISMATCH`
  (`core/ledger/attest_stage14_test.go`, TestRegisterLeafKeyMismatch);
- лист просрочен / ещё не действует на момент `timestamp_claim` → reject
  (TestRegisterLeafValidityAtTimestampClaim);
- не-sandbox genesis с `target_light = 0` / `target_hard = 0` / без параметров PoW →
  genesis невалиден (TestGenesisZeroPowTarget);
- не-sandbox genesis с синтетическим корнем → узел не стартует
  (`net/synthetic_roots_test.go`, TestSyntheticRootOutsideSandbox).

Положительные:
- регистрация с совпадающим листом принимается (TestRegisterLeafKeyMatchAccepted);
- лист, действующий на момент `timestamp_claim`, принимается;
- sandbox-сеть с синтетикой работает: sandbox genesis + синтетический корень — узел
  стартует; `AllowSyntheticRoots` (test-only) работает;
- все существующие векторы и тесты регистрации зелёные (векторы строят цепочки от ключа
  кошелька — привязка листа выполняется ими по построению).

Регресс: `go test ./...` — все пакеты зелёные (core, net, cons, sim, mobile, cmd);
replay-изоляция сетей и векторы (`vectors/`) не падают.

## Предкоммитная проверка (локально)

- `gofmt -l .` — пусто; `go vet ./...` — чисто.
- `go test ./...` — зелёные; `go test -race ./...` — зелёные (все пакеты).
- `flutter analyze` — No issues found; `flutter test` — 43 passed.
- Kotlin-модуль (`android/`): `:app:testDebugUnitTest` + `:app:assembleDebug` —
  BUILD SUCCESSFUL (Gradle 8.11.1, JDK 21).
- После прогона: Gradle-демоны остановлены (`gradlew --stop` — «No Gradle daemons are
  running»), `pgrep` по gradle/tnp-node/tnp-testnet/qemu — пусто.

## CI (ветка stage14-attestation, коммит c9fcdbf)

Все 8 jobs пяти workflows зелёные (GitHub Actions, прогон от 01.11.2026):

| Workflow | Run | Статус | Время |
| --- | --- | --- | --- |
| core.yml (go test + -race) | core #30 | success | 2m 46s |
| net.yml | net #39 | success | 3m 47s |
| cons.yml | cons #39 | success | 3m 2s |
| flutter.yml (analyze + test) | flutter #38 | success | 11m 8s |
| android.yml (AAR gomobile bind + :app:testDebugUnitTest + assembleDebug/Release) | android #38 | success | 7m 25s |

AAR в CI собирается из исходников (в git не коммитится), поэтому проверки A1/A2
попадают в сборку CI автоматически.

## Изменённые файлы

Код: `core/attest/roots.go` (новый), `core/attest/roots/*.pem` (новые, 2),
`core/attest/attest.go` (BuildLeafValidity), `core/ledger/validate.go`,
`core/ledger/ledger.go`, `net/config.go`, `net/node.go`, `mobile/mobile.go`,
`cmd/tnp-node/main.go`, `cmd/tnp-testnet/main.go`.
Тесты: `core/attest/roots_test.go` (новый), `core/ledger/attest_stage14_test.go` (новый),
`net/synthetic_roots_test.go` (новый), `net/helpers_test.go`.
Спека: `docs/KNOWLEDGE.md`, `docs/KNOWLEDGE-ENG.md`, `docs/KNOWLEDGE-1.md`,
`docs/KNOWLEDGE-ENG-1.md`, `docs/KNOWLEDGE-3.md`, `docs/KNOWLEDGE-ENG-3.md`,
`README.md`, `README-ENG.md`, `spec/00-overview.md`, `spec/02-genesis-manifest.md`,
`spec/03-validation.md`.

## PENDING / примечания

- Прогон честного пути на живом устройстве (цепочка KeyMint до корней Google) не выполнялся —
  в мандате этапа его нет; DISCLAIMER о замере сохранён в спеке.
- `protocolVersion` не поднимался (формат транзакций/genesis не изменился) — по мандату;
  предложений об изменении нет.
- Демо-genesis `cmd/tnp-node` стал `sandbox = true` (выравнивание с песочницей приложения);
  genesis-файлы прошлых прогонов несовместимы по manifest_hash только если пересобирались
  с изменённым профилем — `tnp-testnet` genesis не менялся.
- A4 (таргеты PoW), A5 (окно 30 с), A6 (приманка), A7 (txNow вне аттестации), A3 (net-DoS),
  A8 (ключи APK), A9 (пересчёт симуляции), A10 (README-эталон) — не тронуты, по правилам этапа.
