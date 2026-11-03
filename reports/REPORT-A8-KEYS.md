# Отчёт темы A8 — боевые ключи подписи APK (единый std + отдельный rec)

Дата: 03.11.2026. Ветка `a8-keys` от `cd6a7d8` (main содержит merge fuzz-долгов
`d5d3e86`). Мандат: тема A8 (промпт в `prompts/`, план — `docs/development-plan.md`).
Основание: находка A8 аудита (`reports/AUDIT-equality-trust-networks.md`,
подтверждена повторным аудитом 01.11.2026) и «release debug-подпись» (средняя)
аудита контура 1 (`reports/REPORT-AUDIT-FRONTIER-PASS.md`).

## 1. Инвентаризация подписи (шаг 0)

Release-APK сети — сборка Flutter-хоста `app/android` (платформенный слой
этапа 6 компилируется в него через srcDir). Параметры сети генерирует
конфигуратор `cmd/tnp-testnet configure` в `android.json` (имя APK
`equality-{global|slug}-{std|rec}-v*.apk`, `applicationId`
`io.equality.<slug>.x<hash8>`, label `Eq-<hash8>`, display_name, adaptive-иконка
`<out>/icon/`) и передаёт в сборку свойствами
`-PTNP_APPLICATION_ID=… -PTNP_APP_LABEL=…` (механика отчёта 12.3,
`app/android/app/build.gradle.kts` `findProperty`). Подпись в этой цепочке
не участвовала: release-тип брал debug-ключ AGP.

| Точка сборки | Текущая подпись | Где задаётся |
| --- | --- | --- |
| `app/android` — Flutter-хост, release-APK сети (`equality-…-v….apk`) | debug-ключ AGP | `app/android/app/build.gradle.kts:44–45` — `signingConfig = signingConfigs.getByName("debug")` (находка A8) |
| `app/android` — debug-сборки | debug-ключ AGP | стандарт AGP, явно не задаётся |
| `android/` — приложение этапа 6, release | без подписи (unsigned APK) | `android/app/build.gradle.kts` — signingConfig в release отсутствует |
| `android/` — debug | debug-ключ AGP | стандарт AGP |
| CI `flutter.yml`, job `apk` | debug APK | `flutter build apk --debug` — release в CI не собирается |
| CI `android.yml`, job `apk` | debug + unsigned release | `./gradlew :app:assembleDebug :app:assembleRelease` — release без signingConfig → unsigned |

Вывод: боевая подпись нужна в release-типе Flutter-хоста; CI release-сборок
Flutter-хоста не выполняет (только debug), CI без боевых ключей остаётся
нетронутым.

## 2. Механика signing-конфига

Каталог боевых ключей — вне клонов репо, по умолчанию `~/.tnp-signing/`
(переопределение: флаг конфигуратора `-signing-dir` или env
`TNP_SIGNING_DIR`). Состав:

- `tnp-std.jks`, `tnp-rec.jks` — keystores (chmod 600), генерация keytool
  на машине владельца (задача 2);
- `credentials.txt` — пароли (chmod 600; пароли keystore и ключа
  отдельные, разные для std и rec; в git/CI/логи/отчёт не попадают);
- `signing.properties` — signing-конфиг (chmod 600, вне git):
  генерируется `configure` при каждом запуске из `credentials.txt`
  (абсолютные пути обоих keystores + пароли; секции `std.*` и `rec.*`).

Цепочка: `configure` по типу сети выбирает вид ключа (std → `tnp-std`,
rec → `tnp-rec`; глобальная сеть — std — комбинации global+rec не
существует по genesis-валидации), проверяет наличие/доступность
keystores и credentials, генерирует `signing.properties`, пишет `key_kind`
и путь signing-конфига в `android.json` и печатает готовую команду
release-сборки (без паролей). Сборка (`app/android/app/build.gradle.kts`)
читает signing-конфиг по пути env `TNP_SIGNING_PROPERTIES` (или `-P`;
дефолт `~/.tnp-signing/signing.properties`), вид ключа — env `TNP_KEY_KIND`
(или `-P`). Отсутствие/недоступность ключей — явный отказ release-сборки
(`GradleException` с диагностикой, артефакт
`reports/a8-keys/05-refusal-diagnostics.txt`); debug-сборки не затронуты
(signing-конфиг разрешается только когда в графe задач есть release-задачи).

CI: workflows не правились; боевых ключей в CI нет и не появится
(signing-конфиг живёт вне git, CI release-сборок Flutter-хоста не
выполняет). Затронутых правками CI-джоб нет: `flutter.yml` собирает
только `--debug`, `android.yml` собирает проект `android/` (release там
остался unsigned — без изменения поведения).

## 3. Параметры ключей (без паролей)

| | tnp-std | tnp-rec |
| --- | --- | --- |
| Назначение | все std-сети (включая глобальную) | все rec-сети (recurring, коммерческая сборка) |
| Алгоритм | RSA 4096 (SHA256withRSA при подписи APK) | RSA 4096 |
| Срок валидности | 10000 дней (~27 лет, до 21.03.2054) | 10000 дней (до 21.03.2054) |
| Alias | `tnp-std` | `tnp-rec` |
| Keystore | JKS (`tnp-std.jks`), отдельные пароли keystore и ключа | JKS (`tnp-rec.jks`), отдельные пароли keystore и ключа |
| Владелец сертификата | CN=Equality std release key, O=Equality Project | CN=Equality rec release key, O=Equality Project |

Обоснование: RSA 4096 — стандартная практика Android-подписи (RSA,
SHA-256-совместимые схемы); 4096-бит выбран из-за срока ≥25 лет
(NIST SP 800-57: 2048-бит ~112-битная стойкость — горизонт до ~2030);
срок 10000 дней — классическая величина Android-документации
(≥25 лет, требования Google: сертификат действителен минимум до 2033).
JKS — единственный storetype keytool с раздельными паролями keystore и
ключа (PKCS12 использует один пароль).

## 4. Сборка и верификация

Тестовые genesis из `cmd/tnp-testnet configure` (артефакты вывода —
`reports/a8-keys/01-*`, `02-*`):

- std-сеть: шаблон 7, slug `stdnet`, recurring off →
  `equality-stdnet-std-v1.0.0.apk`, applicationId
  `io.equality.stdnet.x24b5f628`, label `Eq-24b5f628`;
- rec-сеть: шаблон 8, slug `recnet` → `equality-recnet-rec-v1.0.0.apk`,
  applicationId `io.equality.recnet.xcbea5bee`, label `Eq-cbea5bee`.

Сборка `flutter build apk --release` с env `TNP_SIGNING_PROPERTIES` +
`TNP_KEY_KIND` (+ `ORG_GRADLE_PROJECT_TNP_APPLICATION_ID`/`_TNP_APP_LABEL`)
— обе успешны (выжимка логов — `reports/a8-keys/06-build-excerpt.txt`);
AAR ядра — `gomobile bind` как в CI. APK в git не коммитились
(копии — `/tmp/a8-networks/`, вне репо).

`apksigner verify --print-certs` (артефакты `03-*`, `04-*`):

- `equality-stdnet-std-v1.0.0.apk` — Signer SHA-256
  `deb05e1f…e9efc9f4` — сертификат `tnp-std` (совпадение с keytool -list);
- `equality-recnet-rec-v1.0.0.apk` — Signer SHA-256
  `fa35d49d…822aea39` — сертификат `tnp-rec`.

Отрицательные проверки (`05-refusal-diagnostics.txt`): release без
signing-конфига и без `TNP_KEY_KIND` — явный отказ с диагностикой.

Установка и запуск std release-APK в эмуляторе (AVD stage12ga, API 36,
`-wipe-data`; headless-режим `-no-window` падает SIGSEGV в SwiftShader —
известная особенность среды, прогон выполнен под Xvfb, как в
ui-owner-protection 02.11): `adb install -r` — Success, запуск
`io.equality.stdnet.x24b5f628/platform.tnp.tnp_app.MainActivity`,
онбординг виден на скриншоте `08-emulator-onboarding.png` (палитра
совпадает с эталоном `reports/ui-owner-protection/01-onboarding-ru.png`).
Полный путь участников не выполнялся (вне мандата).

## 5. Отпечатки сертификатов (SHA-256)

| Ключ | Отпечаток SHA-256 |
| --- | --- |
| `tnp-std` (std-сети, глобальная) | `deb05e1f59843a676bb15c6782919af6d418240a2cf2ffa5b6b989b9e9efc9f4` |
| `tnp-rec` (rec-сети) | `fa35d49d234d67a7b07fe873da7d9de5db3ea177335fe9b73c371868822aea39` |

Опубликовано: `docs/RELEASE-KEYS.md` (назначение, дата генерации,
строка-пример проверки установленной APK). Wire fingerprint-манифест
(мини-этап 12.2) не тронут.

## 6. Изменённые файлы

- `cmd/tnp-testnet/signing.go` — новый: выбор ключа по типу сети,
  генерация `signing.properties` (chmod 600), диагностика отсутствия;
- `cmd/tnp-testnet/configure.go` — флаг `-signing-dir`, вызов signing-точки,
  `key_kind`/`signing_properties` в `android.json`, печать команды сборки;
- `cmd/tnp-testnet/configure_signing_test.go` — новый: тесты выбора вида,
  генерации, глобальной сети, отсутствия ключей;
- `cmd/tnp-testnet/configure_stage12_test.go` — герметичность тестов
  (временный signing-каталог);
- `app/android/app/build.gradle.kts` — боевая release-подпись из локального
  signing-конфига, явный отказ без ключей, debug не затронут;
- `.gitignore` — паттерны `*.jks`, `*.keystore`, `credentials.txt`,
  `signing.properties`, `.tnp-signing/` (+ локальный state инструментов
  `.vibe/`, `.wrangler/` — остатки диагностических сессий, для чистого
  `git status`);
- `docs/RELEASE-KEYS.md` — новый: публичные отпечатки;
- `reports/AUDIT-equality-trust-networks.md` — A8 → «Решено»;
- `reports/REPORT-AUDIT-FRONTIER-PASS.md` — F13/«release debug-подпись» →
  «Решено»;
- `reports/a8-keys/` — артефакты верификации (8 файлов, 144 КБ).

## 7. Проверки

- `gofmt -l .` — пусто; `go vet ./...` — чисто;
- `go test ./...` — зелёный (16 пакетов, GOMAXPROCS=6);
  `-race ./cmd/tnp-testnet/ ./mobile/` — зелёный (задетые пакеты);
- `flutter analyze` — No issues found; `flutter test` — All tests passed
  (57);
- сборка release-APK std+rec боевыми ключами — предмет темы, прошла;
- `git ls-files '*.jks' '*.keystore'` — пусто; `git status` — чисто;
- после прогона: эмулятор погашен (`adb emu kill`), Xvfb остановлен,
  Gradle-демоны остановлены (`gradlew --stop`), `pgrep` по
  qemu/java/Xvfb/tnp-* — пусто.

## 8. CI

Правки CI-джоб не требуются и не делались (см. раздел 2). Номера прогонов
всех workflows для ветки `a8-keys` на момент пуша — «CI pending»
(правило CI-verify: сессия статусы не ждёт):

- `cons`, `core`, `net`, `android`, `flutter` — запуски на push ветки
  `a8-keys`: см. сводку сессии (номера зафиксированы после пуша;
  статусы на момент пуша — pending).

## 9. PENDING

Нет. Примечание вне мандата (не блокер): headless-эмулятор
(`-no-window`) на этой машине падает SIGSEGV в SwiftShader при холодном
бутe — прогон выполнен под Xvfb (обход уже использовался сессией
ui-owner-protection 02.11); воспроизводится и 01.11, и 02.11 (coredumpctl).
