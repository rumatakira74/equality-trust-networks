# Отчёт темы фиксов F5 и F16 — ветка `fix-f5-f16`

Мандат: `prompts/fix-f5-f16-prompt.md` + продление (эмуляторный прогон приманки F5).
Основание: находки высокой значимости аудита контура 1 (`reports/REPORT-AUDIT-FRONTIER-PASS.md`,
F5 — п.2, F16 — п.5). Дата: 2026-11-02.
Статус: **F5 и F16 реализованы, протестированы, доказаны контрольным прогоном. Эмуляторный
прогон приманки (F5) выполнен: он вскрыл находку высокой значимости (карточка восстановления
в приманке показывала реальный адрес) — находка исправлена, покрыта регресс-тестом и доказана
повторным прогоном. Ветка запушена, CI pending.**

## (а) Чек-лист мандата

| Пункт mandate | Статус |
| --- | --- |
| 1. F5 — подменной адрес во всех точках выдачи приманки | ✅ код (a8c16f2): `walletAddress`, decoy-кошелёк (`walletInfo`), `btSendAddress` → `displayAddress()`; персистентный адрес легенды в `tnp_duress`; генерация при создании легенды (`mobile.GenerateDecoyAddress`); тесты Go+Dart зелёные. ✅ эмуляторный прогон: приманка — карточка и QR с подменным адресом, обычный PIN — реальный, logcat без адресов; FINDING прогона (кэш адреса при старте) исправлен, регресс-тест, повторный прогон — `reports/fix-f5-f16/emu/*-postfix.*` |
| 2. F16 — Corroboration только из финализированных веток | ✅ код (53bb8bd): `Corroboration.Confirmed` гейтится предикатом финальности (`isFinalizedTx` = чекпоинты + Snowball-finals); тесты зелёные; контрольный прогон доказал; `protocolVersion` НЕ поднят, wire не тронут |
| 3. Тесты (двойная трата / честный путь / F5 / регресс, -race) | ✅ `net/f16_finality_test.go` (unit-гейт + интеграция двойной траты), `mobile` TestGenerateDecoyAddress, `app/test/decoy_address_test.dart`; регресс — полный `go test ./...` и `flutter test` зелёные; `-race ./net/ ./mobile/` зелёные |
| 4. Контрольный прогон eclipse headless + эмуляторный прогон приманки | ✅ headless (матрица v3, ниже); ✅ эмуляторный — сценарий (б).1 пройден после фикса, артефакты `reports/fix-f5-f16/emu/` (pre-fix — доказательство FINDING, `-postfix` — доказательство фикса) |
| 5. Документация и отчёт | ✅ этот отчёт; F5/F16 → «Решено» в `REPORT-AUDIT-FRONTIER-PASS.md`; README 8.2 не тронут; спека не расходится (12.3/12.4/8.2 уже описывают целевую семантику — правки RU+ENG не нужны) |
| Предкоммитная проверка | ✅ gofmt/vet/build чисты; go test ./... зелёный; -race net+mobile зелёный; flutter analyze чист; flutter test 57/57; AAR+APK собраны (APK пересобран после фикса) |
| Финализация: промежуточные коммиты, чистое дерево, пуш ветки, CI не ждать | ✅ коммиты a8c16f2, 53bb8bd, 4194589 + финальный коммит (этот отчёт + артефакты); пуш `fix-f5-f16`; CI pending |

## Механика F5 (кратко)

- Подменной адрес генерируется **один раз при создании легенды** приманки
  (`setupPin`/`register` → `DuressState.ensureDecoyAddress` → `tnp.Tnp.generateDecoyAddress`):
  случайная ключевая пара (ECDSA P-256), адрес обычной деривацией
  `AddressOfKey(MarshalPubKey(pub), networkID)` — валидного формата (32 байта hex),
  **≠ реальному** (перегенерация до 8 раз), **не деривируется** от реального ключа/фингерпринта.
- Персистентен в легенде: `tnp_duress` prefs, ключ `decoy_address` — стабилен между сессиями
  и рестартами (смена адреса выдала бы приманку). Выход обычным PIN → реальный адрес.
- Точки выдачи (все через `displayAddress()` в `FlutterBridge.kt`): `walletAddress`
  (→ `app_state.dart:84,136` → карточка восстановления `recovery_card_screen.dart:109`,
  QR ваучинга `vouching_screen.dart:220`), decoy-поле `address` в `decoyWalletJson`,
  `btSendAddress` (уходит по Bluetooth — тоже подменной). `btReceiveAddress` — чужой адрес, не тронут.
- Гардианы в приманке: легенда уже подменяет их (пустой список в decoy-кошельке) — согласовано,
  скоуп не расширялся (стоп-вопрос не потребовался).
- Реальный адрес не попадает в UI приманки; в логах/артефактах прогона — проверено на
  эмуляторе (0 вхождений в logcat, см. раздел прогона).

## Механика F16 (кратко)

- `Corroboration` получил предикат финальности `finalizedOf`; `Confirmed(txid)` =
  (доставлено ≥ N несвязанными группами/пирами) **И** `finalizedOf(txid)`.
  Узел задаёт предикат в `NewNode`: `n.isFinalizedTx` = `finalityView().IsFinalized` —
  то же представление финальности, что и для перестройки состояния (чекпоинты цепочки
  `FinalizedTxIDs` + финализированные выборы Snowball). Порядок блокировок: `c.mu → n.mu.RLock`,
  обратного порядка в коде нет.
- `OfflineStatus`: `finalized` = `Confirmed` (с гейтом). Приём tx не меняется —
  `Corroboration.Add` по-прежнему пишется при доставке (счётчик для метрик и для
  post-finality подтверждения), гейт только в `Confirmed`. `protocolVersion` не поднят.
- Механика выбрана единственная без wire-изменений (peer-context потребовал бы протокольных
  правок — mandate запрещает поднимать protocolVersion самовольно); стоп-вопрос не потребовался.

## Замеры и доказательства

### Тесты

| Тест | Где | Результат |
| --- | --- | --- |
| Unit-гейт Corroboration (3 пира до финальности → не Confirmed; после → Confirmed; nil-предикат — старое поведение) | `net/f16_finality_test.go` TestF16CorroborationFinalityGate | PASS |
| Двойная трата офлайн-канала (интеграция): A доставлена 3 группами до финальности → `published`; конфликтная B побеждает, чекпоинт финализирует B → B `finalized`, A остаётся `published`; латентность честного пути ~1.8 с (эпоха 2 с) | TestF16OfflineDoubleSpend | PASS |
| F5: адрес валиден (32 байта), ≠ реальному, две генерации различны, не вырожден | `mobile` TestGenerateDecoyAddress | PASS |
| F5 (Dart): адрес стабилен между «рестартами» AppState, 64-hex, ≠ реальному; карточка восстановления в приманке — подменной адрес в тексте и QR; выход обычным PIN → реальный; реестр не тронут | `app/test/decoy_address_test.dart` | PASS |
| Регресс: полный честный путь (перевод/офлайн-платёж без приманки и конфликтов) | весь `go test ./...`, `flutter test` 56/56, sidechannel/eclipse тесты | PASS |
| -race по задетым пакетам | `go test -race ./net/ ./mobile/` (147 с) | PASS |

### Эмуляторный прогон приманки (F5) — итог

AVD `stage12ga` (API 36, x86_64), `-gpu host`; amd64-пиры песочницы (4001/4002) detached
в эмуляторе; APK debug из ветки (AAR с F5-кодом). Онбординг: PIN 1234, тревожный 9876,
легенда 100; отпечаток enrolled (`adb emu finger touch 1`).

1. **Pre-fix прогон (FINDING):** вход тревожным PIN 9876 → кошелёк 100 (легенда), карточка
   восстановления показала **реальный** адрес `85e4e22a…8e4c` вместо подменного
   `0930e9a0…b30c`; после выхода обычным PIN 1234 карточка показала тот же `85e4e22a…8e4c`.
   Причина — `AppState._address` кэшируется при старте под замком и не перечитывается при
   разблокировке. Артефакты: `reports/fix-f5-f16/emu/01..04-*.png`, `address-check.txt`,
   `logcat-duress-phase.txt`.
2. **Фикс:** `AppState.refreshAfterUnlock()` (перечитывает `walletAddress()` и `refreshWallet()`)
   вызывается из `lock_screen` после успешной разблокировки — и по PIN (`res == 'normal'`),
   и по отпечатку. Регресс-тест `app/test/decoy_address_test.dart` — реальная последовательность:
   `load()` до входа → вход тревожным PIN → `refreshAfterUnlock()` → карточка/QR с подменным
   адресом → выход обычным PIN → реальный.
3. **Post-fix прогон:** приманка — кошелёк 100, карточка `cce61b35…775e` (совпадает с
   `decoy_address` в `tnp_duress`), QR ваучинга из того же `app.address`; обычный PIN —
   кошелёк 0, карточка `172c97e6…d14a` (реальный), адреса различны. logcat (приманка и полный):
   0 вхождений decoy и реального адреса. Артефакты: `reports/fix-f5-f16/emu/*-postfix.png`,
   `address-check-postfix.txt`, `logcat-duress-phase-postfix.txt`.
   Кошельковые PNG чёрные — FLAG_SECURE (штатно, экраны с балансом).

### Контрольный прогон eclipse (headless) — итог

Драйвер: новый режим `tnp-testnet eclipse` (`cmd/tnp-testnet/eclipse.go`, коммит 4194589 +
итерации): victim (отдельный процесс + отдельная БД, статус-файл 500 мс), 3 relay-узла на
`127.0.0.1`/`127.0.1.1`/`127.0.2.1` (три /24), driver (2 честных узла-подписанта, регистрация
V/V2, выпуски seeds, A=transfer W→V, B=transfer W→V2 конфликт, C=transfer S15→V). Pre-fix бинарь —
worktree origin/main + tooling. Артефакты: `reports/fix-f5-f16/eclipse/*` (summary/samples/driver/
victim-excerpt/relay), воспроизведение: `reports/fix-f5-f16/REPRODUCE.txt`.

Матрица v3 (genesis epoch 300 s — детерминированное pre-finality окно; фаза атаки 60 с):

| Прогон | link-groups | атакующие | A при 5 пирах/3 группах ДО финальности | после финализации |
| --- | --- | --- | --- | --- |
| a1-pre (main+tooling) | true | 3 × /24 | **`finalized` при f=false — БАГ** | finalized (f=true, голосование) |
| a1-post (ветка) | true | 3 × /24 | **`published` при f=false — ФИКС** | finalized (f=true) |
| a3-pre | false | 3 × /24 | `finalized` при f=false — баг | — |
| a3-post | false | 3 × /24 | `published` при f=false — фикс | — |
| a2-pre/a2-post | true | все в одной /24 | группы=1 (классификатор схлопнул) — атака не проходит даже до фикса | — |

Точная иллюстрация из samples (колонка A = peers/groups/status/finalized):
`a1-pre`: attack `5/3/finalized/f=false`; `a1-post`: attack `5/3/published/f=false`,
conflict `5/3/finalized/f=true`.

**Доказательство фикса:** «финализировано» через соучастников до финальности не достигается
ни в одном post-fix прогоне (a1-post, a3-post: `published` при 3 группах и f=false);
на pre-fix — достигается (баг воспроизведён). На негативном контроле (одна /24,
link-groups=true) группы=1 — поведение объяснено: /24-классификатор схлопывает одну подсеть.

**Латентность честного пути до/после** (прогоны v1, epoch 20 s): C (честная tx, 3 группы)
финализирована на жертве за **26 564 мс (pre)** / **30 334 мс (post)** — рост в пределах одной
эпохи, узкое место — 30-с тик syncLoop доставки, а не чекпоинт; неприемлемого роста нет.

## (б) Остаток работы

1. CI-verify (владелец, `prompts/ci-verify-prompt.md` с веткой fix-f5-f16), затем merge
   по подтверждению владельца. Номера прогонов CI на момент пуша — см. (д), «CI pending».
2. Эмуляторный прогон приманки (F5) завершён (раздел «Эмуляторный прогон приманки (F5) — итог»);
   инструкции прогона сохранены в (в) как воспроизводимые приёмы.

## (в) «Как делать»-приёмы, найденные за сессию (обязательно)

1. **Enroll отпечатка на эмуляторе (API 36, AVD stage12ga):** app → «Set up fingerprint»
   (или `adb shell am start -a android.settings.BIOMETRIC_ENROLL`) → «Pixel Imprint + PIN» →
   PIN `input text 1234` дважды → «Lock screen» → Done → «Set up Pixel Imprint» → MORE →
   sensor screen; завершение — `adb -s emulator-5554 emu finger touch 1`; проверка
   `adb shell "dumpsys fingerprint" | grep prints` (ожидается `{"id":0,...}`).
   Эмулятор запускать с **`-gpu host`** (`-gpu swiftshader_indirect`/`off`, `-accel off` → SIGSEGV).
2. **Accessibility/semantics для uiautomator (Flutter UI иначе не виден):**
   `adb shell "settings put secure accessibility_enabled 1"` — после этого uiautomator dump
   видит Flutter-semantics (content-desc с текстами и bounds). Диалог TalkBack отклонять
   (`input tap` по «Don't allow» или `input keyevent 4`).
3. **Detached-пиры внутри эмулятора с нужными peer ID:** бинарь amd64 (arm64 падает под
   трансляцией), `adb push` в /data/local/tmp, запуск
   `adb shell "setsid <bin> -port 4001 -nodeseed tnp-sandbox-peer-node-key > log 2>&1 < /dev/null &"`
   (без setsid процесс умирает при закрытии adb shell). Peer ID детерминированы от nodeseed:
   `tnp-sandbox-peer-node-key` → QmNPJienGU6g9WBVbwaX26kBUr7ePWPHDNsaLMv9VVXdtP (порт 4001),
   `tnp-sandbox-peer-node-key-2` → QmNXG1JRSkhWPzc8QzkiDm2mBFniQni9vSwW9BCM8JBtJC (порт 4002) —
   эти ID зашиты в `FlutterBridge.kt` песочницы.
4. **Сборка Go-бинарника под Android-emulator (x86_64 AVD):** `GOOS=android GOARCH=amd64`
   требует cgo: `CC=$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin/x86_64-linux-android21-clang
   CGO_ENABLED=1 go build`. При go1.26+ link падает на `wlynxg/anet: invalid reference to
   net.zoneCache` — лечится replace anet на локальную копию без linkname (no-op zone cache);
   с `GOFLAGS=-ldflags=-checklinkname=0` линкуется, но падает в runtime на Android (nil linkname).
5. **AAR (gomobile bind) с go1.27:** `GOFLAGS="-ldflags=-checklinkname=0" gomobile bind -target
   android -androidapi 26 -o android/libs/tnp.aar ./mobile` (проверка: `javap` на tnp.Tnp.class
   из classes.jar — наличие нового метода).
6. **Три разных /24 на loopback без root:** `unshare -Urn sh -c 'ip link set lo up; ip addr add
   127.0.1.1/32 dev lo; ip addr add 127.0.2.1/32 dev lo; ...'` — тогда `ip route get 127.0.1.1`
   даёт `src 127.0.1.1` и source-адрес dial'а становится 127.0.x.1 (без этого ядро выбирает
   src 127.0.0.1 для всей 127/8 и /24-группы схлопываются). Весь прогон (victim+relays) —
   внутри одного такого namespace; жертва слушает на каждом IP атакующих, реле диалит её
   адрес на своём IP. Реле должны диалить ТОЛЬКО жертву (первый dial к 127.0.0.1 создаёт
   0.0.0.0-dialer, который потом переиспользуется и даёт source 127.0.0.1).
7. **Corroboration по нескольким пирам идёт через sync, не gossip:** msgID темы tx =
   `sha256(data)` (`txMessageID`), поэтому один и тот же tx от 3 реле через gossip дедупится
   в одну доставку; `Corroboration.Add` от каждого пира даёт `SyncJournalFrom` (журналы реле).
8. **SubmitTx блокируется на PublishTx** — в инструментах вызывать в goroutine (локальный
   commit в журнал идёт до публикации).
9. **vouch от посевного круга требует выпусков** (`E_VOUCH_PERIODS`) — в прогонах используем
   transfer после mints (окно челленджа периода подбирать через `findlabels -period N` с
   genesis-time = now − 90 сут, чтобы окно было открыто).
10. **pkill/pgrep -f с паттерном в собственной команде убивает свой shell** — паттерны класть
    в файл/готовить через base64, или kill по PID из `ps aux | grep "[p]attern"`.
11. **Полный узел на эмуляторе требует Wi-Fi + зарядки** (`NodeController.shouldBeFull`):
    без Wi-Fi при онбординге узел стартует лёгким и падает с «лёгкому узлу нужен адрес
    кошелька» → регистрация отвечает `NONODE`. Перед прогоном:
    `adb shell "cmd wifi connect-network AndroidWifi open"` (эмуляторный AP; `dumpsys battery`
    — `AC powered: true`), затем повторить «Register wallet».

## (г) FINDING / блокеры / наблюдения

- **FINDING (высокая значимость, F5) → «Решено»:** в приманке карточка восстановления
  показывала **реальный** адрес вместо подменного. Механика: `AppState._address` кэшируется
  один раз в `load()` при старте приложения — ещё под замком, когда `duress.active == false`
  (`main.dart:29`); `lock_screen` после успешного входа (и PIN, и отпечаток) только ставил
  `_unlocked`, адрес не перечитывал. Точки выдачи, читающие `app.address`: карточка
  восстановления (`recovery_card_screen.dart:82`), QR ваучинга (`vouching_screen.dart:220`);
  кошелёк показывает decoy-снапшот через `walletInfo`, но адрес в состоянии оставался реальным.
  Pre-fix прогон: приманка — карточка `85e4e22a…8e4c` (реальный), decoy в `tnp_duress` —
  `0930e9a0…b30c`; обычный PIN — тот же `85e4e22a…8e4c`. **Фикс:** `AppState.refreshAfterUnlock()`
  перечитывает `walletAddress()` и `refreshWallet()`; вызывается из `lock_screen` после
  `_unlocked = true` в обоих путях (PIN `res == 'normal'`, биометрия). **Тесты:** новый
  регресс-тест в `app/test/decoy_address_test.dart` — реальная последовательность (`load()` до
  входа → тревожный PIN → `refreshAfterUnlock()` → карточка/QR с подменным → обычный PIN →
  реальный); прежний тест этот порядок не покрывал, т.к. вызывал `load()` уже после входа.
  **Повторный прогон:** приманка — карточка `cce61b35…775e` == `decoy_address`, QR из того же
  `app.address`; обычный PIN — `172c97e6…d14a` (реальный); адреса различны; logcat без адресов.
  Артефакты: `reports/fix-f5-f16/emu/` (pre-fix — доказательство находки, `-postfix` — фикса).
- **Блокер (снят):** qemu падал SIGSEGV со swiftshader — `-gpu host`; peer-бинарь arm64 падал
  под трансляцией — amd64-сборка; «No fingerprint configured» — force-stop/restart app после
  enroll. Эмуляторный прогон завершён.
- **Наблюдение (мандат):** файл `prompts/fix-f5-f16-continue-prompt.md` в репозитории
  отсутствует; продление мандата (эмуляторный прогон, FINDING → фикс в этой теме, требования
  к `refreshAfterUnlock`, регресс-тесту и повторному прогону) дано владельцем в сессии.
  Промпт не создавался (правило «промпт не редактировать»).

## (д) Состояние на момент пуша

- Ветка `fix-f5-f16`, tip: см. `git log` (коммиты: a8c16f2 F5, 53bb8bd F16, 4194589 tooling
  прогона, adc1b67 отчёт+артефакты eclipse, финальный — фикс F5 по FINDING + регресс-тест +
  артефакты эмуляторного прогона + этот отчёт).
- Тесты зелёные: `go test ./...` (все пакеты), `go test -race ./net/ ./mobile/`,
  `flutter analyze` чист, `flutter test` 57/57, `go vet ./...` чист, `gofmt -l .` пуст,
  `go build ./...` OK; AAR (gomobile) и APK debug собраны (APK пересобран после фикса).
- Контрольный прогон: матрица v3 — см. таблицу выше; артефакты `reports/fix-f5-f16/eclipse/`.
- Эмуляторный прогон приманки: pre-fix FINDING и post-fix доказательство — см. раздел выше;
  артефакты `reports/fix-f5-f16/emu/` (pre-fix + `-postfix`, ≤1 МБ).
- CI: **pending** — номера прогонов workflows будут известны после пуша; сессия CI не ждёт
  (правило mandate). Merge в main — только после CI-verify владельцем и подтверждения.
- Уборка после прогонов: эмулятор остановлен (`adb emu kill`), пиры внутри эмулятора завершены,
  gradle-демоны остановлены, `pgrep` по qemu/java/tnp-node/tnp-testnet — чисто (проверено в конце сессии).

## Изменённые файлы

- `mobile/mobile.go` (+`mobile_test.go`) — `GenerateDecoyAddress` + тест (F5).
- `android/app/src/main/kotlin/platform/tnp/android/pin/DuressState.kt` — персистентный
  подменной адрес легенды (F5).
- `app/android/app/src/main/kotlin/platform/tnp/android/flutterbridge/FlutterBridge.kt` —
  `displayAddress()` и точки выдачи (F5).
- `app/test/fake_platform.dart`, `app/test/decoy_address_test.dart` — контракт и тесты (F5),
  включая регресс-тест реальной последовательности «load() до входа → приманка → обычный PIN».
- `app/lib/state/app_state.dart` — `refreshAfterUnlock()`: перечитывает адрес и снапшот
  кошелька при разблокировке (F5-фикс по FINDING).
- `app/lib/screens/lock_screen.dart` — вызов `refreshAfterUnlock()` после успешного входа
  по PIN (`res == 'normal'`) и по отпечатку (F5-фикс).
- `net/eclipse.go`, `net/node.go`, `net/cons.go`, `net/sidechannel.go` — гейт финальности (F16).
- `net/f16_finality_test.go` (новый), `net/eclipse_test.go` — тесты (F16).
- `cmd/tnp-testnet/eclipse.go` (новый), `cmd/tnp-testnet/main.go` — драйвер контрольного прогона.
- `reports/REPORT-FIX-F5-F16.md` (этот), `reports/fix-f5-f16/` (артефакты+REPRODUCE),
  `reports/fix-f5-f16/emu/` (pre-fix FINDING и `-postfix` доказательство F5),
  `reports/REPORT-AUDIT-FRONTIER-PASS.md` (F5/F16 → «Решено"),
  `SESSION-FIX-F5-F16-CONSPEKT.txt` (конспект сессии по запросу владельца).
