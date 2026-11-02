# Отчёт: аудиторский прогон контура 1 (ширина)

Ветка `audit-frontier-pass` от `main` `0ca176b`, промпт
`prompts/audit-frontier-pass-prompt.md`, режим — **только чтение +
`reports/`**. Код, спека, README, промпты не менялись; находки не
правились. Живых сетей нет, сеть не использовалась (CVE-сверка —
контур 2).

Предусловие выполнено: тема UI-защиты владельца слита в `main`
(merge `84bf387`), правки темы видны в аудите.

**Итог.** 24 наблюдения-расхождения (FINDING), из них 2 высокой
значимости, 9 средней, 13 низкой. По пунктам: privacy/следы — 4,
спека-vs-код — 8, supply-chain — 4, red-team APK — 3, eclipse/sybil — 5;
плюс 1 находка вне мандата (X1). Из 27 проверенных заявлений README/спеки
**22 подтверждены**, 5 расходятся с кодом (таблицы ниже).

---

## 1. Privacy / следы на устройстве

### 1a. Что лежит на диске в открытом виде (SQLite ядра)

| Что | Файл/строка | Оценка | Рекомендация (не реализовывать) |
| --- | --- | --- | --- |
| Полные canonical-байты каждой принятой tx: `wallet`, `seq`, `type`, payload (суммы, получатель), подпись, PoW — колонка `raw` | `core/store/sqlite.go:48-56` (`CREATE TABLE txs`, `raw BLOB NOT NULL`), запись — `net/node.go:731` (`SaveTx`) | средняя | Шифрование БД (SQLCipher-эквивалент) или шифрованный контейнер; сейчас защита — только песочница ОС |
| Индекс «фингерпринт → адрес» (связка «устройство → кошелёк») | `core/store/sqlite.go:57-60`, запись — `net/node.go:735` | средняя | Тот же контейнер; индекс не нужен после активации — кандидат на обрезку |
| Чёрный список фингерпринтов, заголовки чекпоинтов, `total_supply` по эпохам | `core/store/sqlite.go:61-75` | низкая | Публичные данные сети |
| `meta["peer_cache"]` — до 64 адресов пиров | `net/bootstrap.go:229-268` | низкая | См. п.5 (отравление кэша) — это не приватность, а целостность |
| Путь БД | `android/.../node/NodeController.kt:84` → `context.getDatabasePath("tnp-node.db")` = `/data/data/<pkg>/databases/tnp-node.db` | — | Приватный каталог приложения; `allowBackup=false` |
| PIN: `salt + PBKDF2-хеш(обычный) + PBKDF2-хеш(тревожный)`, AES-GCM ключом Keystore | `android/.../pin/PinManager.kt:32` (`File(context.filesDir, "pin_state.bin")`), шифрование — `PinManager.kt:120-145` | низкая | Уже шифровано Keystore-ключом; PBKDF2 120k (`PinHash.kt:12-13`) |
| Легенда приманки (остаток + история) — SharedPreferences `tnp_duress`, **открытым текстом** | `android/.../pin/DuressState.kt:17-19, 38-44` | низкая | Это поддельные данные; реального секрета нет |
| Счётчик неудачных попыток — SharedPreferences `tnp_pin_attempts` | `android/.../pin/PinAttemptPolicy.kt:106-112` | низкая | Не секрет (обосновано в комментарии) |
| Выбор языка — SharedPreferences `ui_locale` | `app/lib/state/app_state.dart:51-60` | низкая | Не секрет |

**Вывод 1a.** Реальные суммы/адреса лежат на диске открытым текстом в
БД ядра; шифрования БД нет ни в каком виде. Защита — приватный каталог
+ `allowBackup=false` + отсутствие `run-as` на release-сборке. Это
соответствует README 12.5 (там заявлено только «SQLite в приватном
каталоге») — расхождения со спекой нет, но канал зафиксирован.

### 1b. Logcat

Полная инвентаризация — `reports/audit-frontier-pass/logcat-inventory.txt`
(63 строки). Суммарно 13 точек `android.util.Log.*` в Kotlin, 0 в Dart,
и журнал ядра `logf → cfg.Log → Log.d("TnpNode", …)`.

| Канал | Файл/строка | Что попадает | Оценка |
| --- | --- | --- | --- |
| Журнал ядра в logcat | `app/.../flutterbridge/FlutterBridge.kt:116` (`Log.d("TnpNode", line)`), источник — `net/node.go:134-138` | peer_id (ShortString), txid (12 hex), первые 12 hex адреса подписанта (`net/cons.go:171`), корень чекпоинта (`net/cons.go:476`), `state_root` (`mobile/mobile.go:629`) | низкая: **сумм нет** (проверено: ни одна `logf`-строка в `net/*.go` не содержит amount/баланс) |
| Статистика узла | `FlutterBridge.kt:745,748` (`sync:`, `stats:`) | peer_id, peers, journal_len, state_root | низкая |
| Старт узла | `android/.../node/NodeController.kt:96` | тот же `statsText()` | низкая |
| Метод моста | `FlutterBridge.kt:257,870,872` | имя метода (`signTx`, `history`, …), текст ошибки | низкая: имена методов раскрывают сценарий, не данные |
| Биометрия | `android/.../keystore/BiometricGate.kt:129,132,133,142` | факт «CryptoObject/plain», текст исключения | низкая: **полезно для атакующего** — различает пути промпта (см. п.4, R4) |
| Подпись | `FlutterBridge.kt:627` | `sig=true/false` | низкая |
| Dart | — | `print`/`debugPrint` в `app/lib` — 0 совпадений | — |

**Вывод 1b.** Сумм в logcat нет — это лучше, чем можно было ожидать.
Утечка ограничена псевдонимами (peer_id, txid, 6 байт адреса) и
публичным `state_root`. Уровень `Log.d` в release-сборке остаётся
активным (R8 не вырезает `android.util.Log` без правила
`-assumenosideeffects`; в `proguard-rules.pro` такого правила нет).

### 1c. FLAG_SECURE — покрытие экранов

| Экран | Пометка | Файл/строка |
| --- | --- | --- |
| Вкладка «Кошелёк» (баланс, история, отложенные) | есть | `app/lib/screens/home_shell.dart:28-34, 58-61` |
| Перевод | есть | `app/lib/screens/wallet/transfer_screen.dart:40-45` |
| Выпуск | есть | `app/lib/screens/wallet/mint_screen.dart:34-40` |
| Отложенные | есть | `app/lib/screens/wallet/pending_screen.dart:33-39` |
| Kotlin-экран этапа 6 (баланс/статус) | есть | `android/.../ui/MainActivity.kt:42-47` |
| Экран блокировки (PIN/отпечаток) | нет | `app/lib/screens/lock_screen.dart` — данных нет, ок |
| Поручительства / Реестр / Миграция / Настройки / Гардианы | нет | публичные адреса, заявлено темой UI |
| **Карточка восстановления** | **нет** | `app/lib/screens/recovery/recovery_card_screen.dart` — `SecureMode.set` отсутствует; входы: `settings_screen.dart:135`, `migration_screen.dart:116`, `onboarding_flow.dart:58` |
| Диалог PIN деструктивной операции | нет | `app/lib/screens/pin_gate.dart` — поверх уже помеченного окна, но при вызове из «Настроек»/«Реестра» окно не помечено |

**Вывод 1c.** Перечень темы UI-защиты полон для экранов с суммами, но
карточка восстановления (адрес кошелька + адреса гардианов +
`network_id` + QR) в перечне отсутствует вовсе — ни в «помечены», ни в
«не помечены» (`reports/REPORT-UI-OWNER-PROTECTION.md:49-58`).
Карточка по замыслу экспортируемая (её печатают/шарят), поэтому это
вопрос политики, а не дыра, — но инвентаризация неполна (F2).

### 1d. allowBackup / debuggable

| Приложение | `allowBackup` | Правила исключения | `debuggable` |
| --- | --- | --- | --- |
| `android/` (этап 6) | `false` (`android/app/src/main/AndroidManifest.xml:23`) | `backup_rules.xml`, `data_extraction_rules.xml` — исключены `root`, `database`, `sharedpref`, `file` (cloud-backup **и** device-transfer) | не задан: debug-сборка — `true` (по умолчанию AGP), release — `false` |
| `app/` (Flutter, этап 7) | `false` (`app/android/app/src/main/AndroidManifest.xml:25`) | те же 4 домена в обоих файлах | то же |

**Вывод 1d.** Соответствует README 12.x и заявлению темы UI. Обе
сборки закрыты для `adb backup`/`bmgr`; `debuggable` в release не
включён. Единственный побочный эффект — debug-сборка доступна через
`run-as` (это ожидаемо и используется прогонами).

### 1e. Временные файлы / кэш

| Что | Файл/строка | Оценка |
| --- | --- | --- |
| PNG карточки восстановления при шаринге | `app/lib/screens/recovery/recovery_card_screen.dart:64-69` (`XFile.fromData` → `SharePlus.instance.share`); плагин пишет файл: `share_plus_platform_interface-6.1.0/lib/method_channel/method_channel_share.dart:96,125` → `<cache>/<uuid>/recovery-card.png`, **не удаляет** | средняя |
| Временные файлы ядра/Android | `os.CreateTemp`/`MkdirTemp`/`cacheDir`/`getCacheDir` — **0 совпадений** во всём репозитории | — |
| Кэш пиров | `meta["peer_cache"]` в БД (см. 1a) | низкая |

**Вывод 1e.** Единственный временный артефакт с чувствительным
содержимым — PNG карточки в кэше приложения. Он переживает шаринг и
живёт до вытеснения кэша ОС (F1).

---

## 2. Спека-vs-код

### 2a. Заявления README / спеки против кода

| # | Заявление (цитата, файл/строка) | Код (файл/строка) | Статус |
| --- | --- | --- | --- |
| 1 | «`setUserAuthenticationRequired(true)` с аутентификацией **на каждую операцию** (CryptoObject), **без временного окна**» — README 8.1:388, KNOWLEDGE-3 8.1:140 | `KeystoreWallet.kt:70-85`: `setUserAuthenticationRequired(true)` + `setUserAuthenticationParameters(AUTH_TIMEOUT_SECONDS=0, AUTH_BIOMETRIC_STRONG)`; `BiometricGate.kt:105-140` — `CryptoObject` на вызов, plain-фолбэк без окна | **подтверждено** |
| 2 | «при `signing_auth = biometric_strong` допустим только `BIOMETRIC_STRONG` (без PIN/пароля устройства)» — README 8.1:389 | `BiometricGate.kt:54,95` — `setAllowedAuthenticators(BIOMETRIC_STRONG)`; `KeystoreWallet.kt:78-81` — `AUTH_BIOMETRIC_STRONG` | **подтверждено** |
| 3 | «`setInvalidatedByBiometricEnrollment(true)`: добавление нового отпечатка инвалидирует ключ» — README 8.1:390 | `KeystoreWallet.kt:83` | **подтверждено** |
| 4 | «В сетях с `signing_auth = device_credential` … подпись допускается по PIN/паролю устройства» — README 8.1:391 | `SigningAuthDeviceCredential` есть только в wire/профилях (`core/wire/genesis.go:80`, `core/profiles/profiles.go:120`); в платформенном слое ветки нет — всегда `BIOMETRIC_STRONG` | **расходится** (F7) |
| 5 | «Устройство без настроенной биометрии: регистрация не разрешается; приложение предлагает системный экран добавления» — README 8.1:393 | `BiometricGate.kt:22-40` (`canUseStrongBiometrics`, `enrollIntent`); Flutter: `onboarding_flow.dart:88,133` (`biometricsAvailable`, `openBiometricsEnroll`) | **подтверждено** |
| 6 | Приманка: «интерфейс тот же, но показывается правдоподобный небольшой остаток и история» — README 8.2:397 | Мост подменяет данные: `FlutterBridge.kt:784-796` (`walletInfo`→`decoyWalletJson`), `813-825` (`history`→`decoyHistoryJson`), `773-782` (`balanceWithProof`), `762` (`awaiting`), `826` (`pendingTransfers`), `840` (`pendingMigrations`) | **подтверждено** |
| 7 | Приманка: «биометрический промпт показывается тот же (plain, без ключа)» — README 8.2:399 | `FlutterBridge.kt:617-627` (`signTx` → `BiometricGate.authenticatePlain`), `501,581` (миграция/гардианы) | **подтверждено** |
| 8 | Приманка: «отправка завершается правдоподобным „ожидает подтверждения“ … без реальной публикации» — README 8.2:399 | `FlutterBridge.kt:636-660` (`submitTx` в приманке → правдоподобный ответ, публикации нет); прогон темы A6 (`reports/REPORT-UI-OWNER-PROTECTION.md`) | **подтверждено** |
| 9 | Приманка: «уведомления не показывают суммы» — README 8.2:399 | `android/.../node/NodeService.kt:50-54` — литерал «Узел сети доверия работает» | **подтверждено** |
| 10 | «Приманка проверяется: **адрес в режиме-приманке по сети не совпадёт с реальным**» — README 8.2:404 | `FlutterBridge.kt:354` — `walletAddress` возвращает реальный адрес **без проверки** `duress.active`; карточка восстановления показывает `app.address` (`recovery_card_screen.dart:109`, `app_state.dart:84,136`) | **расходится** (F5) → **Решено** в теме fix-f5-f16 |
| 11 | «Оба PIN неразличимы: одинаковое время открытия, интерфейс и ограничение числа попыток (5 неудач → 30 с, удвоение до 10 мин, сброс 24 ч)» — README 8.2:401 | `PinAttemptPolicy.kt:27-60` (5/30 с/×2/кап 600 с/24 ч), `PinManager.kt:69-102` (один код на оба хеша, PBKDF2 на оба), `lock_screen.dart:52-56` (задержка 350 мс на любой исход) | **подтверждено** |
| 12 | «Ключ кошелька в Android Keystore, экспорт невозможен» — README 8.1:386 | `KeystoreWallet.kt:38-53` — только публичный ключ и цепочка; `getKey` используется лишь для `initSign` | **подтверждено** |
| 13 | «Цепочка аттестации проверяется локально каждым узлом по корневым сертификатам Google, зашитым в APK» — README 7.2:273 | `core/attest/roots.go:22-60` (`//go:embed roots/*.pem`), `core/attest/roots/google-hardware-attestation-root-rsa.pem` + `…-ecdsa-p384.pem`; проверка — `core/attest/attest.go:130-197` | **подтверждено** |
| 14 | «Проверяются: аппаратный уровень (StrongBox/TEE), `verifiedBootState = Verified`, `deviceLocked = true`, не просроченный уровень патча» — README 7.2:273 | `core/attest/attest.go:178-196` (`SecurityLevel < minLevel`, `BootVerified`, `DeviceLocked`, `patch.After(now)`, `months > maxPatchAgeMonths`) | **подтверждено** |
| 15 | «Привязка листа: SPKI листа равен регистрируемому ключу (`E_ATTEST_KEY_MISMATCH`), срок листа — на момент `timestamp_claim`» — README 7.2:274 | `core/ledger/validate.go:362,366` (`E_ATTEST_KEY_MISMATCH`), `core/attest/attest.go:175` (`CurrentTime: now`) | **подтверждено** |
| 16 | «Список отозванных аттестационных сертификатов распространяется как подписанные данные через сеть» — README 7.2:275 | `grep -rni 'revocation\|revoked\|crl'` по `*.go/*.kt/*.dart` вне `reports/` — **0 совпадений** | **расходится** (F6) |
| 17 | «Один фингерпринт = один адрес; повторная `register` с тем же фингерпринтом и другим адресом отвергается. Индекс узла: `фингерпринт → адрес`» — README 7.2:276 | `core/ledger/validate.go:330-334` (проверка `FingerprintIndex`/`FingerprintBlacklist`), `core/store/sqlite.go:57-60, 119-135` | **подтверждено** |
| 18 | «Запрет второго кошелька на уровне приложения» — README 7.2:279 | `core/ledger/invariants.go:77-84` (один фингерпринт — один адрес); на уровне APK — `walletRegistered` (`FlutterBridge.kt:355`) | **подтверждено** (с оговоркой 7.1: «стоимостной барьер», не абсолютный запрет) |
| 19 | Типы транзакций ядра: `register, vouch, mint, transfer, migrate, evidence, guardian_offer, update_guardians` — README 9.3:451 / KNOWLEDGE-4 9.3:51 | `core/wire/wire.go:24-42` — ровно эти 8 (+ модули 8–15) | **подтверждено** |
| 20 | v9.11: «манифест хранит публичные ключи; `address = SHA-256(fingerprint ‖ network_id)`; адреса в манифесте не хранятся» — KNOWLEDGE.md (v9.11), KNOWLEDGE-1 3.3 | `core/crypto/address.go:26-31` (`AddressOfKey`), `core/crypto/keys.go:106-112` (`Fingerprint`), `core/profiles/profiles.go:348-372` (дубликаты/пустые ключи) | **подтверждено** |
| 21 | «Жёсткий рестарт восстанавливает состояние replay-ом журнала из БД (`cons.Replay strict`), сверка после рестарта — инкрементальная (якорь №41)» — README 12.5:545 | `net/node.go:263-272` (`cons.Replay` при старте), `cons/replay.go:110-129` (strict — любая отклонённая tx = ошибка), `net/sync.go:57-61, 144-160, 310-400` (позиционный якорь, выравнивание с начала при расхождении) | **подтверждено** |
| 22 | «Drift/SQLite для цепочек и индексов» — KNOWLEDGE-4 12.5:149 | Drift отсутствует: нет в `app/pubspec.yaml` и `app/pubspec.lock`; хранилище — Go SQLite `ncruces/go-sqlite3` (`core/store/sqlite.go:1-25`) | **расходится** (F8) |
| 23 | «Лёгкий узел хранит собственную цепочку, **свои входящие** и заголовки чекпоинтов» — KNOWLEDGE-4 12.5:151 | `net/node.go:229-238`: «Входящие НЕ сохраняются: неполный журнал сломал бы `cons.Replay strict`»; `LightState.Incoming` (`net/node.go:127-132`) — только память | **расходится** (F9) |
| 24 | «Собственная цепочка лёгкого узла персистентна (рестарт восстанавливает её из БД)» — README 12.5:547 | `net/node.go:229-238` (`restoreLight`), `lightStore` → `SaveTx` (`net/node.go:798-830`) | **подтверждено** |
| 25 | «Правила безопасности UI: деструктивные операции и опасные настройки — только явный ввод PIN, не биометрия» — KNOWLEDGE-3 8.2:155-159 | `app/lib/screens/pin_gate.dart` (`confirmPin`), вызовы в `transfer_screen.dart`, `mint_screen.dart`, `migration_screen.dart`, `guardians_screen.dart` | **подтверждено** |
| 26 | «P = 3000, `P_max` = 30000» — KNOWLEDGE-3 8.3:169 | `core/params/params.go:146-147`, `FlutterBridge.kt:212` (`threshold_p: 3000`) | **подтверждено** |
| 27 | «Корни Google встроены — оба (RSA и ECDSA P-384); синтетические — только песочница» — README 7.2:274 | `core/attest/roots.go:1-12`, `net/node.go` (проверка `AllowSyntheticRoots`/`sandbox`), `net/config.go:111-114` | **подтверждено** |

### 2b. Дрейф RU/ENG спеки

| Раздел | RU (оригинал) | ENG (перевод) | Дрейф |
| --- | --- | --- | --- |
| История версий | 9.4, 9.5, 9.6, 9.7 присутствуют (`docs/KNOWLEDGE.md:24-30`) | пропущены: 9.3 → 9.8 (`docs/KNOWLEDGE-ENG.md:25→27`), с пометкой «The English translation of versions 9.4–9.7 is pending» | **отстаёт** |
| 8.2 «Правила безопасности UI (этап 9)» | есть (`docs/KNOWLEDGE-3.md:155-159`) | **отсутствует** (`docs/KNOWLEDGE-ENG-3.md:148-158`) | **отстаёт** |
| 8.3 «P = 3000, `P_max` = 30000 ✅» | есть (`docs/KNOWLEDGE-3.md:169`) | «❓ The default value of P and `P_max`» (`docs/KNOWLEDGE-ENG-3.md:165`) | **отстаёт** |
| 11 «Карточка восстановления» | есть (`docs/KNOWLEDGE-4.md:95`) | **отсутствует** (`docs/KNOWLEDGE-ENG-4.md:88-100`) | **отстаёт** |
| 9.2 v9.13 (txNow, C9) | есть (`docs/KNOWLEDGE-4.md:39-47`) | есть (`docs/KNOWLEDGE-ENG-4.md:34-42`) | синхронно |
| 9.3 `update_guardians` | есть | есть | синхронно |
| 12.5 | «Drift/SQLite» (`:149`) | «Drift/SQLite» (`:141`) | синхронно, но оба расходятся с кодом (F8) |

### 2c. Внутренние противоречия спеки

| Противоречие | Где |
| --- | --- |
| 12.5 «Drift/SQLite» против README 12.5:545 «драйвер `ncruces/go-sqlite3` … №38 решён на этапе 16» — два разных утверждения об одном хранилище в одном документе (спека + README) | `docs/KNOWLEDGE-4.md:149` vs `README.md:545` |
| 12.5 «лёгкий узел хранит … свои входящие» против комментария кода «Входящие НЕ сохраняются» и против 12.2 (смена уровня light→full) | `docs/KNOWLEDGE-4.md:151` vs `net/node.go:229-238` |
| 8.2 «адрес в режиме-приманке по сети не совпадёт с реальным» против реализации, где адрес не подменяется | `README.md:404` vs `FlutterBridge.kt:354` |
| 7.2 «список отозванных … через сеть» против отсутствия кода и против 14 «модель угроз» (там отзыв не упомянут) | `README.md:275` vs `grep` = 0 |
| 8.1 «в сетях `device_credential` подпись по PIN/паролю» против единственного реализованного пути `BIOMETRIC_STRONG` | `README.md:391` vs `BiometricGate.kt:54,95` |

---

## 3. Supply-chain инвентаризация

**SBOM:** `reports/audit-frontier-pass/sbom-cyclonedx.json` (CycloneDX 1.5,
209 компонентов: Go 115, pub 78, Maven 16; прямых — 9 / 10 / 16).
Сеть не использовалась: пометки «требует CVE-сверки
govulncheck/osv-scanner — контур 2» относятся ко всем строкам.

### 3a. Go (go.mod / go.sum)

| Модуль | Версия | Закрепление | Замечание (по знаниям модели, без сети) |
| --- | --- | --- | --- |
| `github.com/libp2p/go-libp2p` | v0.50.0 | semver-тег + `go.sum` (h1) | Ядро сетевого стека; исторически — череда DoS-замечаний (identify, Yamux, QUIC-транспорт). Требует CVE-сверки |
| `github.com/libp2p/go-libp2p-pubsub` | v0.18.0 | semver + `go.sum` | Валидаторы/скоринг — исторический класс DoS (усилие на валидацию); тема A3 уже закрыла бюджет дорогих операций на уровне узла |
| `github.com/libp2p/go-libp2p-kad-dht` | v0.42.2 | semver + `go.sum` | DHT-запросы — класс «усилие amplification»; требует сверки |
| `github.com/multiformats/go-multiaddr` | v0.16.1 | semver + `go.sum` | Парсер адресов — класс «некорректный ввод» |
| `github.com/multiformats/go-multihash` | v0.2.3 | semver + `go.sum` | — |
| `github.com/ipfs/go-cid` | v0.6.2 | semver + `go.sum` | — |
| `github.com/ncruces/go-sqlite3` | v0.36.0 | semver + `go.sum` | SQLite в WASM (`go-sqlite3-wasm/v6 v6.4.35401`); CVE SQLite приходят через встроенный WASM-блоб — сверять отдельно от Go-модуля |
| `golang.org/x/crypto` | v0.58.0 | semver + `go.sum` | Argon2id/ECDSA — критичный путь PoW и подписей |
| `golang.org/x/mobile` | v0.0.0-20261011034528-197770444867 | **pseudo-version (коммит, не тег)** | `gomobile bind` — сборка AAR; пиннинг по хешу коммита, обновление только вручную |

Транзитивных модулей в `go.sum` — 147 (115 в SBOM после дедупликации
по версии). `vendor/` нет; целостность — только `go.sum`.
`go 1.27` в `go.mod`, версия Go в CI — `1.27`
(`.github/workflows/android.yml:18`, `flutter.yml:36`).

### 3b. Flutter (pubspec.yaml / pubspec.lock)

| Пакет | Версия (lock) | Объявление | Замечание |
| --- | --- | --- | --- |
| `provider` | 6.1.5+1 | `^6.1.2` | Состояние UI; стабильный |
| `qr_flutter` | 4.1.0 | `^4.1.0` | Рендер QR карточки восстановления |
| `shared_preferences` | 2.5.6 | `^2.5.0` | Хранит `ui_locale`; в Kotlin-слое — `tnp_duress`, `tnp_pin_attempts` |
| `share_plus` | 11.1.0 | `^11.0.0` | **Пишет PNG карточки в кэш** (F1) |
| `intl` | 0.20.3 | `any` | `any` — не пиннинг: версия фиксируется только lock-файлом |
| `cupertino_icons` | 1.0.9 | `^1.0.8` | — |
| `flutter_lints` | 6.0.0 | `^6.0.0` | dev |
| `flutter`, `flutter_localizations`, `flutter_test` | sdk | `sdk: flutter` | Версия SDK — `^3.13.5` (`pubspec.yaml:9`), в CI — `3.47.9` (`flutter.yml:19`) |

`pubspec.lock` закоммичен, у каждого пакета — `sha256` (проверено: 78
пакетов, у всех есть `sha256`). SDK-констрейнт `^3.13.5` в репозитории
и `3.47.9` в CI — расхождение фиксируется как наблюдение (сборка
воспроизводится по CI-пиннингу, локально — по установленному SDK).

### 3c. Gradle / Android

| Что | Версия | Где | Замечание |
| --- | --- | --- | --- |
| AGP (`android/`) | 8.9.1 | `android/build.gradle.kts:6` | **Два разных тулчейна в репозитории** (F14) |
| Kotlin (`android/`) | 2.0.20 | `android/build.gradle.kts:7` | |
| Gradle (`android/`) | 8.11.1 | `android/gradle/wrapper/gradle-wrapper.properties:3` | `distributionSha256Sum` задан — **хорошо** |
| AGP (`app/android/`) | 9.1.0 | `app/android/settings.gradle.kts:24` | |
| Kotlin (`app/android/`) | 2.4.0 | `app/android/settings.gradle.kts:25` | |
| Gradle (`app/android/`) | 9.3.1 | `app/android/gradle/wrapper/gradle-wrapper.properties:5` | `distributionSha256Sum` задан — **хорошо** |
| `compileSdk` / `targetSdk` / `minSdk` | 36 / 36 / 26 | `android/app/build.gradle.kts:10,14,15`; `app/android/app/build.gradle.kts:13,31,32` | Согласовано |
| `androidx.biometric:biometric` | **1.4.0-alpha07** | `android/app/build.gradle.kts:46`, `app/android/app/build.gradle.kts:84` | **alpha в критичном пути** (F12) |
| `androidx.core:core-ktx` | 1.13.1 | оба модуля | — |
| `androidx.appcompat:appcompat` | 1.7.0 | `android/app` | — |
| `com.google.android.material:material` | 1.12.0 | `android/app` | — |
| `androidx.constraintlayout:constraintlayout` | 2.1.4 | `android/app` | — |
| `junit` / `androidx.test.*` | 4.13.2 / 1.2.1 / 1.6.2 / 1.6.1 | `android/app` (test) | — |
| `android/libs/tnp.aar`, `tnp-sources.jar` | сборка `gomobile bind` | на диске, в `.gitignore` | В git не попадают — **хорошо** |
| `gradle-wrapper.jar` | бинарник | **в git**: `android/gradle/wrapper/gradle-wrapper.jar`, при этом `.gitignore:36` его исключает | Целостность jar не проверяется (F15) |
| Release-подпись Flutter-хоста | `signingConfigs.getByName("debug")` | `app/android/app/build.gradle.kts:45` | Release-APK подписан debug-ключом (F13) |

### 3d. CI и политика обновлений

| Наблюдение | Факт | Оценка |
| --- | --- | --- |
| Пиннинг GitHub Actions | все `uses:` — по commit SHA с комментарием версии (`android.yml:16-18`, `flutter.yml:16-19`) | **соответствует** (лучшая практика) |
| Сканирование уязвимостей | `govulncheck`/`osv-scanner`/`trivy`/`snyk` в workflows — **0 совпадений** | пробел: контур 2 |
| Автообновление зависимостей | `.github/` содержит только `workflows/`; `dependabot.yml`/renovate — нет | пробел |
| Триггеры CI | `paths:` включают `go.mod`, `go.sum`, `app/**`, `android/**`; `reports/**` не триггерит workflows | **соответствует** (отчёт «CI pending») |

---

## 4. Черновики red-team сценариев (APK, динамический)

Полные пошаговые сценарии — `reports/audit-frontier-pass/red-team-scenarios.md`
(R1–R5). **Ни один не исполнялся.**

| # | Сценарий | Цель | Контрмера в текущем коде | Ожидаемый исход (по коду, требует прогона) |
| --- | --- | --- | --- | --- |
| R1 | Tapjacking/overlay против PIN-промпта | PIN в поддельное окно / перехват нажатий | **нет** app-level: `setFilterTouchesWhenObscured` и `setHideOverlayWindows` — 0 совпадений; **частично** OS-level (targetSdk 36 → блокирование untrusted touch с API 31) | вероятно блокируется ОС; требует подтверждения на устройстве (F21) |
| R2 | Task hijacking (`taskAffinity`) | подмена экрана «назад»/перехват task | **есть** для Flutter-хоста (`taskAffinity=""`, `launchMode="singleTop"`, `app/…/AndroidManifest.xml:34-36`); **нет** для `android/`-сборки (`android/…/AndroidManifest.xml:31-34`) | Flutter-хост защищён; `android/`-сборка уязвима (F23) |
| R3 | Манипуляция intent / exported-компоненты | вызов внутренних действий без UI | **есть**: `NodeService exported="false"`, extras не читаются, провайдеров/ресиверов нет; **наблюдение**: `genesis.bin`/`peers.txt` читаются из `<external files>/networks/` без проверки источника (`FlutterBridge.kt:130-140`) | подмена genesis = другая сеть (само-сертификация `network_id`), риск ограничен |
| R4 | Frida/Objection: FLAG_SECURE, биометрия, дамп памяти | снять защиту экрана, подписать без биометрии, вытащить секреты | FLAG_SECURE — **нет** (снимается вызовом моста, `FlutterBridge.kt:321-333`); биометрия — **есть** (Keystore, окно 0, `CryptoObject`); память — **частично** (ключ неэкспортируем, PIN-файл шифрован, легенда приманки — открытый текст); детект root/Frida/debugger — **нет** (0 совпадений) | ожидаемо успешно по FLAG_SECURE и подмене `DuressState.active`; подпись без биометрии — нет (F22) |
| R5 | Побочные каналы: кэш, бэкап, `run-as` | адрес/история без обхода UI | `allowBackup=false` + 4 домена исключены — **есть**; очистка кэша после шаринга — **нет**; FLAG_SECURE на карточке — **нет** | PNG карточки остаётся в кэше (F1); журнал читается `run-as` только на debug-сборке |

---

## 5. Eclipse / sybil — дизайн атаки

Полный дизайн — `reports/audit-frontier-pass/eclipse-sybil-design.md`.
**Не исполнялся.**

| # | Наблюдение | Файл/строка | Оценка |
| --- | --- | --- | --- |
| 5.1 | Порог `eclipse_n` применяется **только** к `OfflineStatus`, не к приёму данных | `net/node.go:988` (единственное `Confirmed`), `net/node.go:577` и `net/sync.go:387` (`Corroboration.Add` до финальности) | **высокая** (F16) → **Решено** в теме fix-f5-f16: 3 соучастника в 3 `/24` дают «финализировано» для tx, которая проиграет конфликт — двойная трата в офлайн-канале |
| 5.2 | Классификатор несвязанности — только `/24` (IPv4) / `/64` (IPv6), ASN отсутствует | `net/eclipse.go:25-40`; «ASN-классификация требует внешних данных — PENDING» — `net/eclipse.go:9-10` | **средняя** (F17): три `/24` одного ASN проходят N=3 |
| 5.3 | Кэш пиров: до 64 адресов, без TTL, шаг 1 каскада, заполняется из непроверенного peer-exchange | `net/bootstrap.go:229-268` (кэш), `:270-275` (шаг 1), `:337-370` (PX) | **средняя** (F18): отравление кэша переживает рестарт |
| 5.4 | Свежесть seed-списка не проверяется: `VerifySeedList(..., now=0)` | `net/bootstrap.go:285`; `SeedListMaxAge` (7 сут) — `net/seedlist.go:35,129` | **низкая** (F19): старый подписанный список принимается |
| 5.5 | Фильтр чужой сети — сравнение строки `UserAgent` | `net/host.go:26,63,101-115` | **низкая** (F20): атакующий объявляет тот же префикс и проходит; криптографической привязки нет |
| 5.6 | Детект эклипса (доля одного источника, смена набора пиров) | отсутствует | наблюдение в рамках 5.3 |
| 5.7 | Смешивание источников (минимум по одному из каждого, до `MinPeers*2 = 6`) | `net/bootstrap.go:168-228`, `net/config.go:141` | **соответствует** 12.3 |
| 5.8 | Sybil на уровне кошельков закрыт поручительствами (K=3, возраст ≥ 3 мес.), а не сетью | `core/ledger/validate.go`, `core/ledger/invariants.go` | **соответствует** 7.4 |

**Контрольный прогон** (по образцу A3-флуда, `cmd/tnp-testnet/flood.go`):
жертва — отдельный процесс с отдельной БД, драйвер запускает себя с
`-role victim`; предлагаемые параметры — `-link-groups=true|false`
(негативный контроль), 3 атакующих на `127.0.0.1` / `127.0.1.1` /
`127.0.2.1` (три разных `/24` без контейнеров), `-baseline 15`,
`-duration 60`, `-candidates 12`, `-victim-port 4171`; метрики —
`LinkSpread(txid)` → `(peers, groups)`, `journal_len`, `state_root`,
`OfflineStatus(txid)`.

---

## 6. Перечень FINDING

Шкала: **высокая** — обесценивает заявленную защиту или даёт
финансовый ущерб; **средняя** — требует условия (root/подготовленный
нападающий/несколько IP), но достижимо; **низкая** — гигиена,
инвентаризация, дрейф документации.

| ID | Пункт | Находка | Файл/строка | Значимость |
| --- | --- | --- | --- | --- |
| F1 | 1e | PNG карточки восстановления (адрес + гардианы + `network_id` + QR) остаётся в кэше приложения после шаринга: `XFile.fromData` → `getTemporaryDirectory()`, файл не удаляется | `app/lib/screens/recovery/recovery_card_screen.dart:64-69`; `share_plus_platform_interface-6.1.0/lib/method_channel/method_channel_share.dart:96,125` | средняя |
| F2 | 1c | Карточка восстановления не покрыта FLAG_SECURE и отсутствует в перечне экранов темы UI-защиты (ни «помечены», ни «не помечены») | `app/lib/screens/recovery/recovery_card_screen.dart` (нет `SecureMode.set`); `reports/REPORT-UI-OWNER-PROTECTION.md:49-58` | низкая |
| F3 | 1b | logcat-строки ядра содержат первые 12 hex адреса подписанта, txid, peer_id, `state_root`; сумм нет; `Log.d` активен в release (нет `-assumenosideeffects`) | `FlutterBridge.kt:116`; `net/cons.go:171`; `net/node.go:587`; `mobile/mobile.go:629`; `app/android/app/proguard-rules.pro` | низкая |
| F4 | 1a | Полные canonical-байты транзакций (суммы, адреса, подписи) лежат в SQLite открытым текстом; шифрования БД нет | `core/store/sqlite.go:48-56`; `net/node.go:731`; `NodeController.kt:84` | средняя |
| F5 | 2 | В режиме-приманке остаток подменяется, а адрес — нет: `walletAddress` возвращает реальный адрес без проверки `duress.active`; карточка восстановления показывает его. Заявление README 8.2 «адрес в режиме-приманке по сети не совпадёт с реальным» не выполняется локально: принуждающий видит реальный адрес и проверяет баланс любым клиентом | `FlutterBridge.kt:354`; `recovery_card_screen.dart:109`; `app_state.dart:84,136`; `README.md:404` | **высокая** → **Решено** (тема fix-f5-f16, коммит a8c16f2; отчёт `reports/REPORT-FIX-F5-F16.md`) |
| F6 | 2 | Заявленный «список отозванных аттестационных сертификатов через сеть» не реализован (0 совпадений `revocation`/`revoked`/`crl`) | `README.md:275`; `core/attest/*` | средняя |
| F7 | 2 | Сети с `signing_auth = device_credential` не обслуживаются: платформенный слой всегда `BIOMETRIC_STRONG`, ветки device_credential нет | `README.md:391`; `BiometricGate.kt:54,95`; `KeystoreWallet.kt:78-81`; `core/wire/genesis.go:80` | средняя |
| F8 | 2 | Спека 12.5 «Drift/SQLite» против факта: Drift не используется (нет в `pubspec.yaml`/`pubspec.lock`), хранилище — Go SQLite `ncruces/go-sqlite3`; README 12.5 уже исправлен, спека — нет | `docs/KNOWLEDGE-4.md:149`; `README.md:545`; `core/store/sqlite.go:1-25` | низкая |
| F9 | 2 | Спека 12.5 «лёгкий узел хранит свои входящие» против кода: входящие не персистентятся (только память) | `docs/KNOWLEDGE-4.md:151`; `net/node.go:229-238, 127-132` | низкая |
| F10 | 2b | ENG-спека отстаёт от RU: история версий без 9.4–9.7; §8.2 без «UI safety rules»; §8.3 с открытым «❓ P/P_max» против «✅ 3000/30000»; §11 без карточки восстановления | `docs/KNOWLEDGE-ENG.md:25→27`; `KNOWLEDGE-ENG-3.md:148-158,165`; `KNOWLEDGE-ENG-4.md:88-100` | низкая |
| F11 | 2 | Комментарий кода всё ещё «PENDING: вопрос №2 — точная деривация не определена Документом», хотя v9.11 её зафиксировала | `core/crypto/address.go:1-11` | низкая |
| F12 | 3 | `androidx.biometric:biometric:1.4.0-alpha07` — alpha-версия в критичном пути (биометрический промпт и `CryptoObject`-подпись), в двух модулях | `android/app/build.gradle.kts:46`; `app/android/app/build.gradle.kts:84` | средняя |
| F13 | 3 | Release-сборка Flutter-хоста подписана debug-ключом | `app/android/app/build.gradle.kts:45` | средняя |
| F14 | 3 | Два разных тулчейна Gradle в одном репозитории: AGP 8.9.1/Kotlin 2.0.20/Gradle 8.11.1 и AGP 9.1.0/Kotlin 2.4.0/Gradle 9.3.1 — двойная поверхность supply-chain | `android/build.gradle.kts:4-5`; `app/android/settings.gradle.kts:22-23` | низкая |
| F15 | 3 | `gradle-wrapper.jar` в git (при исключении в `.gitignore:36`); целостность jar не проверяется (пиннинг есть только для `distributionUrl`) | `android/gradle/wrapper/gradle-wrapper.jar`; `android/gradle/wrapper/gradle-wrapper.properties:4` | низкая |
| F16 | 5 | Порог `eclipse_n` гейтит только `OfflineStatus`; `Corroboration.Add` идёт до финальности → 3 соучастника в 3 `/24` дают «финализировано» для tx, которая проиграет конфликт (двойная трата офлайн-канала) | `net/node.go:988, 577`; `net/sync.go:387` | **высокая** → **Решено** (тема fix-f5-f16, коммит 53bb8bd; отчёт `reports/REPORT-FIX-F5-F16.md`) |
| F17 | 5 | Классификатор несвязанности — только `/24`/`/64`, ASN отсутствует: три `/24` одного ASN проходят N=3 | `net/eclipse.go:25-40, 9-10` | средняя |
| F18 | 5 | Кэш пиров (до 64 адресов, без TTL, шаг 1 каскада) заполняется из непроверенного peer-exchange → отравление переживает рестарт | `net/bootstrap.go:229-268, 270-275, 337-370` | средняя |
| F19 | 5 | Свежесть seed-списка не проверяется (`now=0`), `SeedListMaxAge` (7 сут) не применяется | `net/bootstrap.go:285`; `net/seedlist.go:35,129` | низкая |
| F20 | 5 | Фильтр чужой сети — сравнение строки `UserAgent`, не криптографическая привязка к `network_id` | `net/host.go:26,63,101-115` | низкая |
| F21 | 4 | Нет app-level защиты от overlay/tapjacking: `setFilterTouchesWhenObscured` и `setHideOverlayWindows` не вызываются; расчёт только на системное блокирование untrusted touch (targetSdk 36) — требует проверки на устройстве | `app/lib/screens/pin_gate.dart`; `android/.../ui/MainActivity.kt`; `BiometricGate.kt` | средняя |
| F22 | 4 | FLAG_SECURE снимается вызовом моста `setSecureMode(false)`; детекта снятия/повторной установки нет; детекта root/Frida/debugger нет | `FlutterBridge.kt:321-333`; `grep isDebuggerConnected/ptrace/TracerPid/frida` = 0 | низкая |
| F23 | 4 | `android/`-сборка без `taskAffinity` — паритет с Flutter-хостом (`taskAffinity=""`) не выдержан | `android/app/src/main/AndroidManifest.xml:31-34`; `app/android/app/src/main/AndroidManifest.xml:34-36` | низкая |
| F24 | 1/2 | `isStrongBoxCapable()` всегда `false` → `minSecurityLevel()` всегда 1 (TEE), StrongBox не запрашивается; спека 7.2 допускает и TEE, т.е. не нарушение, но в коде заглушка | `KeystoreWallet.kt:145-150` | низкая |

### FINDING вне мандата

| ID | Находка | Файл | Примечание |
| --- | --- | --- | --- |
| X1 | Два файла находок лежат в корне репозитория, а не в `reports/`, вопреки конвенции («логи, находки — в `reports/`») | `FINDING-awaiting-offline-tx-not-persistent.txt`, `FINDING-vouching-screen-stale-nonestatus.txt` | Не чинилось (правки — отдельные темы по решению владельца) |

---

## 7. Артефакты прогона

| Файл | Что внутри | Размер |
| --- | --- | --- |
| `reports/REPORT-AUDIT-FRONTIER-PASS.md` | этот отчёт | ~40 КБ |
| `reports/audit-frontier-pass/sbom-cyclonedx.json` | SBOM CycloneDX 1.5, 209 компонентов (go 115, pub 78, maven 16) | 92 КБ |
| `reports/audit-frontier-pass/logcat-inventory.txt` | все точки вывода в logcat (13 Kotlin + журнал ядра), 63 строки | 8 КБ |
| `reports/audit-frontier-pass/red-team-scenarios.md` | R1–R5: цель, шаги, признак успеха, контрмера | 17 КБ |
| `reports/audit-frontier-pass/eclipse-sybil-design.md` | параметры атаки, срабатывающие/несрабатывающие проверки, параметры контрольного прогона | 14 КБ |

Сумма артефактов — ~131 КБ (лимит этапа ~5 МБ, отдельный файл ≤ 1 МБ —
соблюдено).

## 8. Изменённые файлы

Только `reports/`:

```
reports/REPORT-AUDIT-FRONTIER-PASS.md          (новый)
reports/audit-frontier-pass/sbom-cyclonedx.json      (новый)
reports/audit-frontier-pass/logcat-inventory.txt     (новый)
reports/audit-frontier-pass/red-team-scenarios.md    (новый)
reports/audit-frontier-pass/eclipse-sybil-design.md  (новый)
```

Код, спека, README, промпты — без изменений (проверяется
`git diff --stat main -- ':!reports'` = пусто).

## 9. CI и локальная проверка

`reports/**` не триггерит workflows (пути в `.github/workflows/*.yml`
не включают `reports/`) — прогонов по этой ветке не будет.
**CI pending** (статусов не ждём, правило CI-verify).

Локальная проверка дерева (код не менялся — контроль, что ветка
собрана от зелёного `main`):

| Проверка | Результат |
| --- | --- |
| `go test ./...` | зелёный (`net` 58,9 с, остальные — из кэша) |
| `flutter analyze` (в `app/`) | `No issues found!` |
| `flutter test` (в `app/`) | `All tests passed!` (54 теста) |
| `git status` после прогона | чисто, кроме новых файлов `reports/` |

## 10. Метод и ограничения

- Только чтение файлов репозитория на tip `0ca176b`; сеть не
  использовалась, `govulncheck`/`osv-scanner` не запускались (контур 2).
- Оценки «известных проблем» зависимостей — по знаниям модели, без
  сверки с базами CVE; каждая строка помечена как требующая сверки.
- Динамические сценарии (п.4) и eclipse-прогон (п.5) не исполнялись —
  это черновики по решению владельца.
- Числа «подтверждено/расходится» относятся к заявлениям, попавшим в
  мандат (README 8.1, 8.2, 12.5, 9.3, 7.x; KNOWLEDGE 7–9, 11);
  полный перебор всех утверждений спеки не выполнялся.
