# Отчёт blind-аудита темы A8 — боевые ключи подписи APK (ветка `a8-keys`)

Дата: 03.11.2026. Аудиторская ветка `audit-a8-keys` — от tip ветки `a8-keys`
`2bb50f3` (база ветки `cd6a7d8`; предусловие промпта выполнено, tip не ушёл).
Режим: только чтение + `reports/`; код, тесты, спека, README, промпты не
правились, находки не исправлялись. Мандат: `prompts/a8-keys-prompt.md`.

Слепость соблюдена: пункты 1–5 выполнены до открытия
`reports/REPORT-A8-KEYS.md`; его заявления проверялись в пункте 6 как
гипотезы. Правило CI-verify: статусы CI не ожидались.

Артефакты аудита — `reports/audit-a8-keys/` (10–17, только отпечатки и
диагностика; секретов нет).

---

## 1. Задачи мандата: статус по факту

| № | Задача мандата | Статус | Доказательство | Замечание |
| --- | --- | --- | --- | --- |
| 1 | Инвентаризация подписи (шаг 0): таблица «точка сборки → подпись → где задаётся»; путь параметров `configure` → сборка | **выполнено** | Таблица из 6 строк — `reports/REPORT-A8-KEYS.md` §1, коммит `a6389c0`. Сверено с кодом независимо: `app/android/app/build.gradle.kts:61-99` (release — боевой ключ), `android/app/build.gradle.kts:21-30` (release без `signingConfig` → unsigned), `.github/workflows/flutter.yml:55-56` (`--debug`), `.github/workflows/android.yml:61-62` (`:app:assembleDebug :app:assembleRelease`, проект `android/`). Путь параметров проверен прогоном: `android.json` (`apk_name`, `application_id`, `label`, `key_kind`, `signing_properties`) → `-P`/env → манифест APK (`aapt2 dump badging`: `io.equality.monte_cloud.x44abc8ce`, label `Eq-44abc8ce`) — артефакты 12, 13 | Таблица инвентаризации верна. Описание механизма в §2 отчёта неточно — см. A8-AUD-06 |
| 2 | Боевые ключи std + rec: генерация локально, `~/.tnp-signing/`, RSA, срок ≥25 лет, отдельные пароли | **выполнено** | `~/.tnp-signing/` — каталог `700`, файлы `600`; `keytool -list -v`: `tnp-std` и `tnp-rec`, `PrivateKeyEntry`, 4096-bit RSA, валиден 03.11.2026 → 21.03.2054 (10000 дней, ~27 лет); четыре разных пароля (keystore≠ключ, std≠rec) — проверено сравнением, значения не выводились (артефакт 15). `.gitignore:56-60` — `*.jks`, `*.keystore`, `credentials.txt`, `signing.properties`, `.tnp-signing/` | Нет |
| 3 | Точка конфигурации подписи в `configure`: std → `tnp-std`, rec → `tnp-rec`, глобальная — std; signing-конфиг из локального файла вне git; явная диагностика при отсутствии ключей; debug не затронут | **выполнено** (с замечаниями) | `cmd/tnp-testnet/signing.go:110` `setupSigning`, `:23` `signingKind`, `:32` `signingDirDefault` (env `TNP_SIGNING_DIR` → `~/.tnp-signing`); `configure.go:33` флаг `-signing-dir`, `:307-311` вызов точки, `:324` `key_kind`, `:334` `signing_properties`; `app/android/app/build.gradle.kts:21-23` путь/вид ключа, `:65-99` отказ + `signingConfigs.create`. Прогоны: std-сеть (глобальная, шаблон 7) → `key_kind: std`; rec-сеть (шаблон 8) → `key_kind: rec`; отказ без конфига и без `TNP_KEY_KIND` — `GradleException` с явной диагностикой; `:app:assembleDebug` проходит (артефакты 12–14) | A8-AUD-02 (проверяются оба keystore), A8-AUD-03 (release определяется по имени задачи) |
| 4 | Сборка и верификация release-APK std + rec боевыми ключами; `apksigner verify --print-certs`; установка в эмуляторе | **выполнено** | Независимая пересборка аудитом: std `flutter build apk --release` (86.6 s), rec (30.2 s), обе успешны; `apksigner verify --print-certs -v` — std `deb05e1f…e9efc9f4`, rec `fa35d49d…822aea39`, RSA 4096, схема v2 (артефакты 10, 11). Установка/онбординг — `reports/a8-keys/07-install-run.txt`, `08-emulator-onboarding.png` (1080×2424) | Эмуляторный прогон аудитом не воспроизводился (артефакты на месте, содержимое скриншота не перепроверялось) |
| 5 | `docs/RELEASE-KEYS.md`: отпечатки SHA-256 обоих сертификатов, назначение, дата генерации, строка-пример проверки | **выполнено** | `docs/RELEASE-KEYS.md:9-12` — таблица std/rec с отпечатками и датой 03.11.2026, `:14-18` параметры, `:20-26` пример `apksigner verify --print-certs`. Отпечатки сверены с собственной сборкой и с `keytool` — совпадают | A8-AUD-04 (утверждение про CI, строка 38) |
| 6 | Закрытие находок и отчёт | **выполнено** | A8 → «Решено»: `reports/AUDIT-equality-trust-networks.md:93` (та же конвенция, что A3/A4/A5/A6); F13/«release debug-подпись» → «решено»: `reports/REPORT-AUDIT-FRONTIER-PASS.md:232` и `:308`. `reports/REPORT-A8-KEYS.md` — 188 строк (инвентаризация, механика, параметры без паролей, верификация, отпечатки, изменённые файлы, номера прогонов CI). README/DISCLAIMER, спека, wire fingerprint-манифест 12.2 — не тронуты (`git diff --name-only main...a8-keys` — ни одного файла `README*`, `DISCLAIMER*`, `docs/KNOWLEDGE*`, `wire*`) | A8-AUD-05, A8-AUD-07 (неточности в отчёте) |

---

## 2. Секреты и гигиена ключей (критично)

| Проверка | Результат |
| --- | --- |
| `git ls-files '*.jks' '*.keystore'` | пусто (0 файлов) |
| `git status` | чисто (только `reports/audit-a8-keys/` — артефакты аудита) |
| `.gitignore`-паттерны | `*.jks`, `*.keystore`, `credentials.txt`, `signing.properties`, `.tnp-signing/` — `.gitignore:56-60` |
| signing-конфиг вне git | `~/.tnp-signing/signing.properties` (0600), путь вне клона; `.tnp-signing/` в `.gitignore` |
| История ветки: добавления `.jks`/`.keystore`/`credentials.txt`/`signing.properties` | `git log --all --diff-filter=A` по этим путям — пусто |
| Пароли в диффе `main...a8-keys` | только фиктивные тестовые (`fake-std-*`, `fake-rec-*`) в `configure_signing_test.go` и строки формата `signing.go:89`; боевых значений нет |
| Пароли в артефактах и `docs/RELEASE-KEYS.md` | нет (grep по `storePassword|keyPassword|PRIVATE KEY` — пусто) |
| CI/workflows | не правились (`git diff main...a8-keys -- .github/` — 0 файлов); `secrets.*`, `keystore`, `jks`, `STORE_PASSWORD` в workflows — нет |
| Копии keystores в рабочем дереве репо | нет (`find` по `*.jks`, `*.keystore`, `credentials.txt`, `signing.properties` — пусто) |
| APK в git | нет; `*.apk` в `.gitignore:31`; сборки вне репо (`/tmp/a8-networks/`, `app/build/`) |
| Права | каталог `700`, `credentials.txt`/`signing.properties`/`*.jks` — `600` |

**Утечек секретов не обнаружено.** Критических находок по пункту 2 нет.

---

## 3. Воспроизводимость сборки и верификация

| Сборка | Команда (env) | Результат | Отпечаток SHA-256 сертификата | Сверка с `docs/RELEASE-KEYS.md` |
| --- | --- | --- | --- | --- |
| std-сеть (`global`, шаблон 7) | `TNP_SIGNING_PROPERTIES=~/.tnp-signing/signing.properties TNP_KEY_KIND=std ORG_GRADLE_PROJECT_TNP_APPLICATION_ID=io.equality.global.x8001c274 ORG_GRADLE_PROJECT_TNP_APP_LABEL=Eq-8001c274 flutter build apk --release` | успешно, 86.6 s, 175.2 МБ | `deb05e1f59843a676bb15c6782919af6d418240a2cf2ffa5b6b989b9e9efc9f4` | совпадает |
| rec-сеть (`monte-cloud`, шаблон 8) | `TNP_KEY_KIND=rec … TNP_APPLICATION_ID=io.equality.monte_cloud.x44abc8ce TNP_APP_LABEL=Eq-44abc8ce` | успешно, 30.2 s, 175.2 МБ | `fa35d49d234d67a7b07fe873da7d9de5db3ea177335fe9b73c371868822aea39` | совпадает |

Дополнительно: `apksigner verify` — «Verifies», схема v2, один подписант,
RSA 4096; манифест APK несёт ожидаемые `applicationId` и label (путь
`configure` → сборка работает). Отрицательные проверки — явный отказ
release-сборки без signing-конфига и без `TNP_KEY_KIND`, debug не затронут.
APK в git не коммитились.

---

## 4. Находки

Правки не выполнялись (решение владельца). Значимость: критическая /
высокая / средняя / низкая.

| ID | Файл:строка | Значимость | Суть | Рекомендация |
| --- | --- | --- | --- | --- |
| A8-AUD-01 | `.gitignore:62-64` | низкая | **Вне мандата**: вместе с signing-паттернами в `.gitignore` добавлены `.vibe/`, `.wrangler/` (локальный state инструментов CLI/wrangler). Раскрыто в отчёте исполнителя §6, но предметом темы не является | Вынести в отдельное решение/коммит либо убрать из темы |
| A8-AUD-02 | `cmd/tnp-testnet/signing.go:112-118` | низкая | `setupSigning` требует наличия **обоих** keystores (`std` и `rec`) независимо от вида ключа сети. Проверено: при наличии только `tnp-std.jks` `configure` std-сети не генерирует `signing.properties` и сообщает «боевой ключ rec недоступен» → release-сборка отвергается. Держатель только одного ключа собрать свою сеть не может. Побочно: `signing.properties` всегда содержит пароли обоих ключей | Проверять и писать только секцию нужного вида (`kind`), второй ключ — опционально |
| A8-AUD-03 | `app/android/app/build.gradle.kts:21-23` | средняя | Признак release-сборки — подстрока `elease` в **именах** задач (`gradle.startParameter.taskNames`), а не наличие release-задач в графе. Проверено: `./gradlew :app:build` содержит `:app:assembleRelease`/`:app:packageRelease` в графе, но ошибки подписи не возникает — release-APK соберётся **без подписи, молча** (артефакт 17). Гарантия «явный отказ без ключей» действует только для `assembleRelease`/`bundleRelease` (в т. ч. `flutter build apk --release`) | Определять release по графу задач (`gradle.taskGraph`/`tasks.withType<...>`) либо явно запрещать unsigned release |
| A8-AUD-04 | `docs/RELEASE-KEYS.md:38` | низкая | «CI боевых ключей не имеет и release-APK не собирает» — неточно: `android.yml:61-62` выполняет `:app:assembleRelease` для проекта `android/` (release там unsigned). Отчёт исполнителя §1 фиксирует это верно — расхождение документ↔факт | Уточнить формулировку: «CI не собирает release-APK Flutter-хоста; release проекта `android/` — unsigned» |
| A8-AUD-05 | `reports/REPORT-A8-KEYS.md` §3 (таблица параметров) | низкая | «RSA 4096 (SHA256withRSA при подписи APK)» не соответствует факту: схема v2 подписана **RSASSA-PSS with SHA-512** (algorithm ID `0x0104`, разбор APK Signing Block — артефакт 16); самоподпись сертификата — `SHA384withRSA`. Фактическая подпись сильнее заявленной, но заявление неверно | Исправить формулировку в отчёте (документ-справка) |
| A8-AUD-06 | `reports/REPORT-A8-KEYS.md` §2 | низкая | «signing-конфиг разрешается только когда в графе задач есть release-задачи» — неточно: проверяются имена задач (см. A8-AUD-03) | Синхронизировать с A8-AUD-03 |
| A8-AUD-07 | `reports/REPORT-A8-KEYS.md` §7 | низкая | «`go test ./...` — зелёный (16 пакетов)»: фактически 17 пакетов (15 с тестами + 2 без тестовых файлов). Счёт незначим, прогон зелёный | Уточнить число |

Криптографических находок **вне мандата** нет: ветка не трогает протокол,
wire, `protocolVersion`, genesis-семантику и код ядра/сети/консенсуса
(`git diff --name-status main...a8-keys` — только signing-механика,
`.gitignore`, `docs/RELEASE-KEYS.md`, отчёт и артефакты). Ветка `main`
использовалась только как база сверки и не аудировалась.

---

## 5. Процесс и гигиена ветки

| Проверка | Результат |
| --- | --- |
| Предмет диффа `main...a8-keys` | 18 файлов: `.gitignore`, `app/android/app/build.gradle.kts`, `cmd/tnp-testnet/{configure.go,signing.go,configure_signing_test.go,configure_stage12_test.go}`, `docs/RELEASE-KEYS.md`, `reports/{AUDIT-equality-trust-networks.md,REPORT-A8-KEYS.md,REPORT-AUDIT-FRONTIER-PASS.md}`, `reports/a8-keys/*` (8). Протокол/wire/спека/genesis/раздел 13/A9/A10/прочие FINDING — не тронуты |
| Событийные коммиты | 7 коммитов, по одному на шаг мандата: `a6389c0` инвентаризация → `7b508f3` `.gitignore` → `9436428` configure+градл → `bd9649e` сборка/верификация → `e7b98be` `docs/RELEASE-KEYS.md` → `e6935f2` закрытие находок+отчёт → `2bb50f3` номера CI |
| Форматирование отдельным коммитом | отдельного коммита нет и не требуется: `gofmt -l .` — пусто |
| Файлы >1 МБ | нет (максимум — `08-emulator-onboarding.png`, 113 КБ) |
| Объём артефактов темы | `reports/a8-keys/` — 144 КБ (лимит ~5 МБ) |
| APK в git | нет |
| Правки вне мандата | одна: `.gitignore:62-64` — A8-AUD-01 |

---

## 6. Сверка с отчётом исполнителя

| Заявление `reports/REPORT-A8-KEYS.md` | Факт | Вердикт |
| --- | --- | --- |
| Ветка от `cd6a7d8`, main содержит merge fuzz-долгов `d5d3e86` | `git merge-base main a8-keys` = `cd6a7d8`; `d5d3e86` — предок `main` | подтверждено |
| Инвентаризация: release Flutter-хоста подписан debug-ключом (`app/android/app/build.gradle.kts:44–45`) | до правки — `signingConfig = signingConfigs.getByName("debug")`; после — боевой ключ | подтверждено |
| `android/` release — unsigned; CI `flutter.yml` — только debug; CI `android.yml` — debug + unsigned release | сверено по коду и workflows | подтверждено |
| Механика: `configure` выбирает вид ключа, генерирует `signing.properties` (0600) вне git, печатает команду сборки без паролей; градл читает `TNP_SIGNING_PROPERTIES` + `TNP_KEY_KIND` | сверено по `signing.go`, `configure.go`, `build.gradle.kts`; прогоны подтверждают | подтверждено |
| «signing-конфиг разрешается только когда в графе задач есть release-задачи» | проверяются **имена** задач; `gradlew build` собирает release молча без подписи | **расходится** → A8-AUD-03, A8-AUD-06 |
| Параметры ключей: RSA 4096, 10000 дней (до 21.03.2054), alias `tnp-std`/`tnp-rec`, отдельные пароли | `keytool -list -v` — совпадает; 4 разных пароля | подтверждено |
| «SHA256withRSA при подписи APK» | RSASSA-PSS with SHA-512 (v2, ID `0x0104`); сертификат — `SHA384withRSA` | **расходится** → A8-AUD-05 |
| Сети: std `stdnet`/шаблон 7 → `equality-stdnet-std-v1.0.0.apk`; rec `recnet`/шаблон 8 → `equality-recnet-rec-v1.0.0.apk` | артефакты 01/02 и мои прогоны согласованы (имена и `applicationId` по конвенции) | подтверждено |
| Обе release-сборки успешны; APK в git не коммитились | моя независимая пересборка успешна; APK в `/tmp/a8-networks/` и `app/build/` | подтверждено |
| `apksigner`: std `deb05e1f…e9efc9f4`, rec `fa35d49d…822aea39` | мои сборки дают те же отпечатки | подтверждено |
| Отрицательные проверки: отказ без конфига и без `TNP_KEY_KIND` | воспроизведено, тексты диагностики совпадают | подтверждено |
| Установка std release-APK в эмуляторе, онбординг виден | артефакты `07-install-run.txt`, `08-emulator-onboarding.png` на месте; аудит эмулятор не запускал | не перепроверено (артефакты есть) |
| Изменённые файлы (§6) | совпадает с `git diff --name-status main...a8-keys` (отчёт не перечисляет сам себя — тривиально); `.vibe/`/`.wrangler/` раскрыты как «остатки диагностических сессий» | подтверждено (с A8-AUD-01) |
| `gofmt -l .` пусто, `go vet ./...` чисто, `go test ./...` зелёный, `-race` по задетым пакетам, `flutter analyze`/`flutter test` (57) | воспроизведено: `gofmt` пусто, `go vet` чисто, `go test ./...` зелёный, `-race ./cmd/tnp-testnet/` и `./mobile/` зелёные, `flutter analyze` — No issues, `flutter test` — 57 passed | подтверждено (счёт пакетов — A8-AUD-07) |
| `git ls-files '*.jks' '*.keystore'` пусто, `git status` чисто | подтверждено | подтверждено |
| Номера прогонов CI: `cons` 41608478220, `net` 41608478245, `flutter` 41608478221, «CI pending»; `core`/`android` не запускаются | path-фильтры проверены: `cons.yml`/`net.yml` включают `cmd/**` (изменён) → запуск ожидаем; `flutter.yml` — `app/**` → запуск ожидаем; `core.yml` (`core/**`, `go.mod`, `vectors/**`) и `android.yml` (`android/**` — `app/android/**` не совпадает) → не запускаются. Сами номера прогонов в этой сессии не проверяемы: у доступного GitHub-коннектора нет чтения workflow-runs, статусы коммита не отдаются; правило CI-verify — статусы не ожидаются | объяснение подтверждено; номера — **не проверены** (вне возможности сессии) |
| PENDING нет; примечание про SIGSEGV headless-эмулятора | вне предмета темы, не проверялось | не проверено |

---

## 7. Проверки аудита (воспроизводимость)

- `gofmt -l .` — пусто; `go vet ./...` — чисто.
- `go test ./...` (GOMAXPROCS=6) — зелёный, 17 пакетов; `-race ./cmd/tnp-testnet/` и `-race ./mobile/` — зелёные.
- `flutter analyze` — No issues found; `flutter test` — All tests passed (57).
- Две release-сборки боевыми ключами — по одной за раз (Gradle), обе успешны; отпечатки сверены.
- Ресурсы: доступная память не опускалась ниже ~5,5 ГБ (порог 2 ГБ не задет); Gradle-демоны остановлены (`gradlew --stop`), эмулятор не запускался, `pgrep` по `tnp-node|tnp-testnet|qemu|GradleDaemon` — пусто.

---

## 8. Вердикт

**готова с замечаниями.**

Блокеров нет. Предмет темы выполнен полностью и воспроизводимо: боевые ключи
std/rec с заявленными параметрами, точка конфигурации в `configure`,
release-подпись в градле с явным отказом без ключей, две независимо
пересобранные и верифицированные release-APK, опубликованные отпечатки,
закрытие A8/F13. Утечек секретов нет: keystores и пароли вне git, CI и
артефактов, история ветки чиста, workflows не правились.

Замечания (не блокирующие, правки — отдельным решением владельца):

1. **A8-AUD-03** (средняя) — гарантия «release без ключей отвергается» держится только на подстроке `elease` в именах задач: `./gradlew :app:build` соберёт release-APK без подписи молча.
2. **A8-AUD-02** (низкая) — `configure` требует оба keystore даже для одного вида ключа.
3. **A8-AUD-04** (низкая) — `docs/RELEASE-KEYS.md:38` неточно описывает release-сборку в CI.
4. **A8-AUD-05/06/07** (низкие) — неточности в `reports/REPORT-A8-KEYS.md`: алгоритм подписи APK, описание определения release-сборки, число пакетов в `go test`.
5. **A8-AUD-01** (низкая) — правка вне мандата: `.gitignore:62-64` (`.vibe/`, `.wrangler/`).

Merge в `main` — только после дайджеста и подтверждения владельца.
