# Черновики red-team сценариев (контур 1, п.4) — APK, динамический

**Статус: НЕ ИСПОЛНЯЛИСЬ.** Это исполнимые пошаговые черновики для
отдельного прогона по решению владельца. Снято на ветке
`audit-frontier-pass` от `main` `0ca176b`, режим «только чтение».

Ссылки на код — по состоянию tip `0ca176b`. Для каждого сценария:
цель → шаги → ожидаемый признак успеха → контрмера в текущем коде
(есть / частично / нет).

---

## R1. Tapjacking / overlay против PIN-промпта

**Цель.** Заставить владельца ввести PIN в окно, поверх которого
рисуется поддельный слой, либо перехватить нажатия «ОК»/«Отмена».

**Цели в коде.** PIN-диалог деструктивных операций —
`app/lib/screens/pin_gate.dart` (`_PinDialog`, `barrierDismissible: false`);
экран блокировки — `app/lib/screens/lock_screen.dart` (`TextField`,
`obscureText: true`); PIN-диалог Kotlin-экрана —
`android/.../ui/MainActivity.kt` (`askPin`, `AlertDialog`).

**Шаги.**
1. Собрать debug-APK (`flutter build apk --debug`) и установить на
   эмулятор API 36.
2. Установить рядом тестовое приложение с
   `SYSTEM_ALERT_WINDOW` + `TYPE_APPLICATION_OVERLAY`, рисующее
   полупрозрачный слой с фальшивым «Подтвердите перевод».
3. Выдать overlay-разрешение через `adb shell appops set <pkg>
   SYSTEM_ALERT_WINDOW allow` (или экран настроек).
4. Открыть приложение, инициировать перевод (PIN-промпт), накрыть
   окно overlay-слоем, ввести PIN.
5. Повторить с `FLAG_NOT_TOUCHABLE` и с прозрачностью 0/50/100%.

**Признак успеха.** Нажатия в PIN-диалоге регистрируются
overlay-приложением; либо PIN уходит в поддельное поле; либо
overlay остаётся поверх диалога после ввода.

**Контрмера в текущем коде.**
- App-level: **нет** — `setFilterTouchesWhenObscured` не вызывается
  нигде (`grep` по `android/`, `app/` — 0 совпадений);
  `HIDE_OVERLAY_WINDOWS` (`Window.setHideOverlayWindows`) не
  используется.
- OS-level: **частично** — `targetSdk = 36`
  (`app/android/app/build.gradle.kts`, `android/app/build.gradle.kts`),
  поэтому системное блокирование «untrusted touch» Android 12+
  (API 31+) действует по умолчанию. Требует подтверждения на устройстве:
  поведение зависит от типа окна атакующего и прозрачности.
- `barrierDismissible: false` в `pin_gate.dart` закрывает только
  закрытие тапом вне диалога, не подмену слоя.

**Рекомендация (не реализовывать в этом прогоне).** Явный
`setFilterTouchesWhenObscured(true)` на PIN-экранах и
`setHideOverlayWindows(true)` на окне активности; тест-кейс в
`androidTest`.

---

## R2. Task hijacking (taskAffinity / launchMode)

**Цель.** Встроить активность атакующего в task приложения, чтобы
«назад» из PIN-экрана попадал в поддельный экран, либо перехватить
запуск.

**Факты манифестов.**
- `app/android/app/src/main/AndroidManifest.xml`: `MainActivity`
  `exported="true"`, `launchMode="singleTop"`, `taskAffinity=""`,
  intent-filter только `MAIN`/`LAUNCHER`.
- `android/app/src/main/AndroidManifest.xml`: `MainActivity`
  `exported="true"`, **без** `launchMode` и **без** `taskAffinity`
  (значения по умолчанию: `standard`, affinity = имя пакета).
- `NodeService` в обоих манифестах: `exported="false"`.
- Других активностей/провайдеров/ресиверов нет.

**Шаги.**
1. Установить приложение-атакующий с `taskAffinity`, равным
   `applicationId` цели (`platform.tnp.tnp_app` или
   `io.equality.<slug>.x<hash8>` из конфигуратора).
2. Запустить активность атакующего с
   `FLAG_ACTIVITY_NEW_TASK` и тем же affinity, затем — цель.
3. Проверить `adb shell dumpsys activity activities` — общий ли task.
4. Нажать «Назад» с PIN-экрана; проверить, куда возвращает.
5. Повторить для `android/`-сборки (там affinity по умолчанию).

**Признак успеха.** Активность атакующего оказывается в task цели
(или цель — в task атакующего), «назад» ведёт на поддельный экран.

**Контрмера в текущем коде.**
- `taskAffinity=""` в Flutter-хосте — **есть** (снимает
  affinity-вектор для `app/`).
- `launchMode="singleTop"` — **есть** (не даёт второй экземпляр
  сверху, но не блокирует перехват task).
- `android/`-сборка: `taskAffinity` не задан — **нет** (affinity =
  `platform.tnp.android`; вектор открыт для этой сборки).
- `singleInstancePerTask`/`documentLaunchMode` не используются.

**Рекомендация.** Для `android/`-сборки задать `taskAffinity=""`
(паритет с Flutter-хостом); проверить `dumpsys` в прогоне.

---

## R3. Манипуляция intent / exported-компонентами

**Цель.** Дотянуться до внутренних экранов/сервиса, вызвать
привилегированные действия без UI, или подсунуть данные через intent.

**Факты.**
- `MainActivity` (`exported="true"`) принимает только
  `MAIN`/`LAUNCHER`; **дополнительных action/extra не обрабатывает** —
  `onCreate` не читает `getIntent().getExtras()`
  (`android/.../ui/MainActivity.kt`), Flutter-хост —
  `MainActivity.kt` в `app/android/.../tnp_app/`.
- `NodeService`: `exported="false"`, но `ACTION_STOP` —
  константа `platform.tnp.android.node.STOP`; вызов извне
  заблокирован `exported=false`.
- `POST_NOTIFICATIONS`, `FOREGROUND_SERVICE_DATA_SYNC`,
  `BLUETOOTH_CONNECT`, `ACCESS_FINE_LOCATION` (`maxSdkVersion=30`) —
  объявлены в манифестах.
- Content-provider'ов, `FileProvider`, `BroadcastReceiver` — нет.

**Шаги.**
1. `adb shell am start -n <pkg>/.MainActivity -e <ключ> <значение>` —
   проверить, что extras игнорируются.
2. `adb shell am startservice -n <pkg>/platform.tnp.android.node.NodeService`
   — ожидается `SecurityException` (не exported).
3. `adb shell am broadcast -a platform.tnp.android.node.STOP` —
   ожидается отсутствие получателей.
4. Проверить `dumpsys package <pkg>` на `exported` у всех
   компонентов и на `grantUriPermissions`.
5. Подсунуть `genesis.bin`/`peers.txt` в `<external files>/networks/`
   (внешний каталог приложения) — проверить, принимает ли узел
   genesis из внешнего каталога без проверки источника.

**Признак успеха.** Любой вызов, меняющий состояние (остановка узла,
подмена genesis/пиров), без согласия пользователя.

**Контрмера в текущем коде.**
- `exported="false"` у сервиса — **есть**.
- Отсутствие обработки extras — **есть** (нет вектора).
- `genesis.bin` из `<external files>/networks/` читается без проверки
  подписи/источника — `FlutterBridge.kt:130-140` (`ensureGenesis`:
  сначала `filesDir/networks`, затем `getExternalFilesDir("networks")`).
  **Нет** проверки, что genesis пришёл из приглашения; genesis
  само-сертифицируется (`network_id = H(manifest ‖ genesis)`), но
  подменённый genesis = другая сеть, а не подмена в текущей — риск
  ограничен (нужен ещё и `peers.txt`). Отметить как наблюдение,
  не как дыру.

**Рекомендация.** Прогон: проверить, что подмена `genesis.bin` в
внешнем каталоге даёт «другую сеть» и не смешивается с текущей
(`network_id`-фильтр по UserAgent, см. п.5).

---

## R4. Frida / Objection — обход FLAG_SECURE, hook биометрии, дамп памяти

**Цель.** Снять защиту экрана, обойти биометрическую авторизацию,
вытащить секреты из памяти.

**Факты.**
- FLAG_SECURE ставится/снимается мостом:
  `FlutterBridge.kt:321-333` (`setSecureMode`), `SecureMode`
  (`app/lib/services/secure_mode.dart`), `MainActivity.kt:44-47`.
- Ключ кошелька: `setUserAuthenticationRequired(true)` +
  `setUserAuthenticationParameters(0, AUTH_BIOMETRIC_STRONG)` +
  `setInvalidatedByBiometricEnrollment(true)`
  (`KeystoreWallet.kt:88-103`); подпись — `NONEwithECDSA` над готовым
  дайджестом, low-s нормализация.
- PIN: `pin_state.bin` в `filesDir`, AES-GCM ключом Keystore
  (`PinManager.kt`); PBKDF2-HMAC-SHA256, 120 000 итераций.
- Легенда приманки: SharedPreferences `tnp_duress` (`DuressState.kt`).
- Счётчик попыток: SharedPreferences `tnp_pin_attempts`.
- Детекта рута/отладчика/хуков в коде **нет** (grep по
  `isDebuggerConnected`, `ptrace`, `TracerPid`, `frida` — 0).

**Шаги (на rooted-эмуляторе/устройстве, отдельный прогон).**
1. `adb root` (эмулятор `google_apis`/`aosp`), залить
   `frida-server` под ABI эмулятора, запустить.
2. `frida-ps -U` → найти процесс приложения.
3. **FLAG_SECURE:** hook `android.view.Window.setFlags` /
   `clearFlags`; либо вызвать мост `setSecureMode(false)` через
   `frida -U -f <pkg> -l clear_secure.js`; снять скриншот
   (`adb exec-out screencap -p`) на вкладке кошелька.
4. **Биометрия:** hook
   `BiometricPrompt$AuthenticationCallback.onAuthenticationSucceeded`
   → вызвать `KeystoreWallet.sign(...)` без реальной биометрии.
5. **Дамп памяти:** `frida-trace`/`objection memory dump` по
   `libgojni.so` и heap; искать `pin_state.bin`, легенду приманки,
   адреса/суммы в JSON-строках моста.
6. **Экран блокировки:** hook `PinManager.verify` → вернуть
   `PinCheck.NORMAL`; проверить, открывается ли UI.
7. **Приманка:** hook `DuressState.active` → `false`; проверить,
   подменяется ли остаток.

**Признак успеха.** (3) скриншот вкладки кошелька непустой;
(4) подпись получена без биометрии; (5) в дампе видны PIN-хеш/легенда/
суммы; (6) UI открыт без PIN; (7) приманка снята.

**Контрмера в текущем коде.**
- (3) **нет** — FLAG_SECURE снимается вызовом моста; повторной
  установки флага и детекта снятия нет.
- (4) **есть** — ключ привязан к Keystore, окно авторизации 0,
  `CryptoObject` на операцию; hook callback'а не даёт подписи без
  реальной авторизации (проверить в прогоне: возможен ли hook
  `Signature.update/sign` на уже авторизованном объекте).
- (5) **частично** — приватный ключ неэкспортируем (Keystore);
  PIN-хеш и легенда приманки лежат в памяти приложения и в
  `filesDir`/SharedPreferences (защита — только песочница + Keystore
  для PIN-файла); JSON моста с суммами проходит через Dart-heap.
- (6) **частично** — hook `verify` даёт UI, но подпись всё равно
  требует Keystore-авторизации; приманка при этом не активируется.
- (7) **нет** — `DuressState.active` — обычное поле в памяти.

**Рекомендация.** Прогон зафиксировать как «ожидаемо успешный» по
(3), (6), (7) при rooted-устройстве; в отчёт — честная формулировка:
защита от подготовленного нападающего с root/Frida не заявлена
(README 8.2, «Что не защищено»), но детект снятия FLAG_SECURE —
дешёвое усиление.

---

## R5. Побочные каналы вне UI (кэш, бэкап, `run-as`)

**Цель.** Получить адрес/историю без обхода UI.

**Факты.**
- Карточка восстановления экспортируется как PNG:
  `recovery_card_screen.dart:56-78` (`XFile.fromData` → `SharePlus`);
  `share_plus_platform_interface-6.1.0/lib/method_channel/method_channel_share.dart:96,125`
  пишет файл в `getTemporaryDirectory()` (кэш приложения) и **не
  удаляет** его после шаринга.
- `allowBackup="false"` + `data_extraction_rules.xml` /
  `backup_rules.xml` исключают `root`, `database`, `sharedpref`,
  `file` (в обоих приложениях).
- БД узла: `context.getDatabasePath("tnp-node.db")`
  (`NodeController.kt:84`) — `/data/data/<pkg>/databases/`.
- `android:debuggable` в манифестах не задан (debug-сборка — true по
  умолчанию, release — false).

**Шаги.**
1. Открыть карточку восстановления, нажать «Поделиться», отменить
   шаринг; `adb shell run-as <pkg> ls -la cache/` (debug-сборка) —
   проверить наличие `recovery-card.png`.
2. `adb backup` / `bmgr` — ожидается отказ (allowBackup=false).
3. `adb shell run-as <pkg> cat databases/tnp-node.db` (debug) —
   проверить читаемость журнала.
4. `adb shell dumpsys activity recents` — проверить, есть ли превью
   экранов без FLAG_SECURE (карточка восстановления, реестр).

**Признак успеха.** PNG карточки остаётся в кэше; журнал читается
через `run-as` на debug-сборке; превью карточки восстановления
непустое.

**Контрмера в текущем коде.**
- `allowBackup=false` + правила исключения — **есть**.
- Очистка кэша после шаринга — **нет**.
- FLAG_SECURE на карточке восстановления — **нет** (см. отчёт, п.1c).
- `run-as` на release-сборке недоступен — **есть** (свойство ОС).

**Рекомендация.** Удалять временный файл после шаринга; решить,
нужен ли FLAG_SECURE на карточке (она по замыслу экспортируемая —
вопрос политики, не дыра).
