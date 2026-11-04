---
theme: fix-release-keys
task: "00000030"
branch: fix-release-keys
base: origin/main @ 0e001eb (Merge branch 'fix-mint-window' into main)
session-id: e0ddd989-639a-2ce4-8b17-d3d958842ace
mandate: prompts/fix-release-keys-prompt.md (blob e7b3156)
source: reports/DIGEST-AUDIT-SONNET-2.md; origin/audit-sonnet:reports/audit-sonnet/REPORT-AUDIT-SONNET-2.md §1.3, §1.4, §1.7
model:
  alias: deepseek-flash
  name: deepseek-v4-1-flash-1017
  thinking: max
window:
  start: 2026-11-04T11:30:07Z
  timeout_min: 90
  checkpoints:
    - 2026-11-04T11:40Z  # шаги 1–4 + тесты, коммит 0235657
verdict: шаги 1–4 выполнены; go build/test, flutter analyze/test зелёные; матрица verifySigning 9/9; НЕ смержено
---

# REPORT-FIX-RELEASE-KEYS

P1 аудита Sonnet-2 (`a8-keys`, §1.4 «отказ release-сборки не принудителен»,
§1.7 «права файлов не принудительны», §1.3 «тесты не герметичны») — до
подписания релиза v3.0.0. Ветка `fix-release-keys` от свежего `main`
(`0e001eb`), запушена, **не смержена**. P2/P3 (PKCS12 ≥32 симв., изоляция
подписи, `RELEASE-KEYS.md` + независимый канал отпечатков) — вне темы по
мандату.

## 1. План (как выполнен)

| Шаг мандата | Что сделано | Где |
| --- | --- | --- |
| 1. Сверка отпечатка | задача `verifySigning` считает SHA-256 файла ключа и сверяет с `TNP_SIGNING_KEY_SHA256`; `configure` печатает отпечаток и готовую команду сборки | `app/android/app/build.gradle.kts:43`, `:57`, `:93`; `cmd/tnp-testnet/signing.go:50`, `:83`; `cmd/tnp-testnet/configure.go:406` |
| 2. Права файлов ключей | проверка ≤0600 (только `rw` владельца) в сборке и в `configure`; `Chmod` после записи `signing.properties` | `app/android/app/build.gradle.kts:74`, `:148`; `cmd/tnp-testnet/signing.go:68`, `:154`, `:200` |
| 3. Герметичность тестов | все 11 вызовов `configureCmd` получают `-signing-dir` с временным каталогом; страж `TestMain` ловит любой выход за `t.TempDir()` | `cmd/tnp-testnet/configure_hermetic_test.go:98`; 9 правок в 5 файлах (см. §5) |
| 4. `wantsRelease` | release-артефакт в графе при выключенном signing-конфиге — отказ с явным сообщением; поведение задач-подстрок сохранено | `app/android/app/build.gradle.kts:50`, `:105`, `:182` |
| Тесты | юниты отпечатка/прав, герметичность, `go build ./...`, `go test ./...`, `flutter analyze/test` | `cmd/tnp-testnet/signing_verify_test.go`, `configure_hermetic_test.go` |

## 2. Шаг 1 — сверка отпечатка боевого ключа

**Сборка.** Задача `verifySigning` (`app/android/app/build.gradle.kts:93`)
навешивается на release-артефактные задачи (`:182`:
`packageRelease`, `packageReleaseUniversalApk`, `packageReleaseBundle`,
`makeApkFromBundleForRelease`, `assembleRelease`, `bundleRelease`). Внутри:

- ожидаемый отпечаток — `TNP_SIGNING_KEY_SHA256` (env или `-P`, `:43`),
  срезаются пробелы и приводится к нижнему регистру;
- фактический — SHA-256 **файла** keystore, выбранного `TNP_KEY_KIND`
  (`sha256Hex`, `:57`; путь берётся из `signing.properties`);
- расхождение — `GradleException` с обоими значениями (`:170`);
- отпечаток не задан: при `wantsRelease` — отказ (`:158`), при обычной
  сборке — предупреждение с фактическим SHA-256.

**Конфигуратор.** `configure` считает отпечаток выбранного ключа
(`setupSigning`, `cmd/tnp-testnet/signing.go:182`), сверяет его с
`TNP_SIGNING_KEY_SHA256`, если тот задан (`:212`), печатает отпечаток и
готовую команду сборки, включающую `TNP_SIGNING_KEY_SHA256`
(`cmd/tnp-testnet/configure.go:406-407`). Расхождение на этом шаге —
`ok=false` с диагностикой: genesis-артефакты собираются, release-сборка
будет отвергнута (прежний контракт `setupSigning` сохранён).

Функции `signingKeySHA256` (`signing.go:50`) и `verifySigningKey`
(`:83`) — единственная реализация сверки на стороне Go, покрыты юнитами.
Сторона Gradle дублирует алгоритм на JDK (`MessageDigest`), чтобы сборка
не зависела от тулчейна Go; её поведение проверено прогонами (§6), а не
юнитами.

## 3. Шаг 2 — права файлов ключей

Правило: **≤0600** — только `rw` владельца; любой бит вне `0600`
(доступ группы, остальных, исполнение) — нарушение. Это буквальная
битовая маска мандата: `perm &^ 0o600 != 0`.

- сборка: `permProblem` (`build.gradle.kts:74`) для `signing.properties`,
  выбранного keystore и `credentials.txt` рядом с ним; нарушение — отказ
  с перечислением файлов и `rw-r--r--`-видом прав (`:148`);
- `configure`: `checkSigningPerms` (`signing.go:68`) для
  `credentials.txt`, `tnp-std.jks`, `tnp-rec.jks` (`:200`), нарушение —
  `ok=false` и диагностика с подсказкой `chmod 600`;
- `writeSigningProperties` (`signing.go:124`) теперь делает `Chmod 0600`
  после записи (`:154`): `os.WriteFile(0o600)` задаёт права только при
  создании, а восстановленный из бэкапа или скопированный
  `signing.properties` сохранял чужие права (находка §1.7).

Каталог ключей (`~/.tnp-signing`, 0700) не проверяется: мандат говорит о
файлах; у владельца каталог уже `0700`. Зафиксировано в §8.

## 4. Шаг 4 — release без ключа/отпечатка

`wantsRelease` (`build.gradle.kts:28`) оставлен как был — подстрока
«elease» в запрошенных именах задач. Добавлен второй контур в
`verifySigning` (`:105`): если в графе задач есть release-артефактная
задача (`:50`), а `wantsRelease=false` (то есть `signingConfig = null`),
сборка отвергается с сообщением, называющим и задачу графа, и
запрошенные задачи, и оба выхода (debug-сборка / настройка ключей +
`assembleRelease`). Это закрывает сценарий `./gradlew assemble`, `aR`,
`build` из §1.4 аудита: раньше молча получался
`app-release-unsigned.apk`.

Сообщения конфигурационного блока (`:227`, `:235`) дополнены ссылкой на
отпечаток, поведение и тексты задач-подстрок не менялись.

## 5. Шаг 3 — герметичность тестов

Аудит насчитал 6 вызовов `configureCmd` без `-signing-dir`; фактически их
**9 в 5 файлах** (в аудите не учтены `configure_stage12_test.go` и второй
вызов в `configure_testnet_test.go`). Исправлены все:

| Файл:строка | Что |
| --- | --- |
| `configure_test.go:25` | `-signing-dir` в срез `args` (два подтеста) |
| `configure_testnet_test.go:39`, `:44` | два вызова |
| `configure_stage12_1_test.go:77` | шаблон 3 |
| `configure_stage12_2_test.go:81`, `:89` | дубликаты fingerprint'ов |
| `configure_stage12_test.go:111`, `:123`, `:178` | global+rec, global, slug-валидация |

Уже были герметичны: `configure_stage12_test.go:23` (хелпер
`configureRun`), `configure_signing_test.go:41`, `:80`, `:92`, `:112`.

**Страж.** `TestMain` (`configure_hermetic_test.go:98`) на время прогона
пакета ставит `TNP_SIGNING_DIR` на временный каталог с фиктивными ключами
и «меченым» `signing.properties`, снимает SHA-256 и права всех файлов до
`m.Run()` и сверяет после. Любой тест, забывший `-signing-dir`, читает и
перезаписывает **этот** каталог, и страж падает с расхождением
(«создан / удалён / было → стало»). Проверено обратным экспериментом:
после снятия `-signing-dir` в одном вызове страж дал
`signing.properties: 287b0f92… → db8c7dc7…` и `FAIL`.

Дополнительно `TestConfigureSigningDirWinsOverEnv`
(`configure_hermetic_test.go:149`) фиксирует приоритет `-signing-dir` над
`TNP_SIGNING_DIR`: каталог из env не меняется.

## 6. Матрица fail/warn (проверена прогонами)

Gradle, `app/android`, `--offline`, `JAVA_HOME=java-21-openjdk`,
`TNP_SIGNING_PROPERTIES` всегда во временном каталоге — реальный
`~/.tnp-signing` не читался и не менялся.

| # | Сценарий | `wantsRelease` | Итог | Сообщение |
| --- | --- | --- | --- | --- |
| A | `verifySigning`, signing-конфига нет | false | **warn**, rc=0 | «signing-конфиг … не найден — ключей нет, отпечаток не сверен» |
| B | отпечаток совпал, права 0600 | false | **ok**, rc=0 | «отпечаток ключа tnp-std.jks совпал (…), права файлов ключей ≤0600» |
| C | отпечаток не совпал | false | **fail**, rc=1 | «отпечаток боевого ключа не совпал: … → …, ожидался … (TNP_SIGNING_KEY_SHA256)» |
| D | права keystore 0644 | false | **fail**, rc=1 | «права файлов ключей шире 0600 … rw-r--r-- … нужен chmod 600» |
| E | отпечаток не задан | false | **warn**, rc=0 | «TNP_SIGNING_KEY_SHA256 не задан — отпечаток ключа не сверен … Фактический SHA-256 …» |
| F | `assembleRelease`, отпечатка нет | true | **fail**, rc=1 | «release-сборка без ожидаемого отпечатка ключа …» |
| G | `assembleRelease`, отпечаток совпал | true | **ok**, rc=0 | «отпечаток ключа tnp-std.jks совпал …» |
| H | `assemble` (release-артефакт в графе, подписи нет) | false | **fail**, rc=1 | «release-сборка без боевой подписи: в графе задач assembleRelease … Неподписанный release-APK не собирается молча …» |
| I | `assembleDebug` | false | **ok**, rc=0 | `verifySigning` в графе отсутствует (0 упоминаний в `--dry-run`) |

Сторона `configure` (юниты `signing_verify_test.go`):

| Ситуация | `setupSigning` | `signing_properties` в `android.json` |
| --- | --- | --- |
| ключей нет | `ok=false`, «боевой ключ … недоступен» | нет |
| права > 0600 (любой из трёх файлов) | `ok=false`, «права файлов ключей … chmod 600» | нет |
| `TNP_SIGNING_KEY_SHA256` ≠ отпечаток | `ok=false`, «отпечаток ключа не совпал» | нет |
| `TNP_SIGNING_KEY_SHA256` = отпечаток или пуст | `ok=true` | есть |

## 7. Тесты и прогоны

| Прогон | Результат |
| --- | --- |
| `go build ./...` | rc=0 |
| `go test ./...` | зелёный по всем пакетам (`net` 94,4 с, остальные cached/быстро) |
| `go vet ./cmd/tnp-testnet/`, `gofmt -l` | чисто |
| матрица `verifySigning` (9 сценариев, §6) | 9/9 как ожидалось |
| `flutter analyze` (app) | `No issues found!` |
| `flutter test` (app) | 57 passed |
| Gradle/Kotlin демоны | остановлены (`--stop` + добивание), `ps` пуст |

Новые юниты: `signing_verify_test.go` — `TestSigningKeySHA256` (`:19`),
`TestVerifySigningKey` (`:46`, совпадение/регистр/расхождение/пустой
отпечаток/нет файла), `TestCheckSigningPerms` (`:93`, 0600 и 0400
проходят, 0644/0640/0604/0660/0700/0777 отвергаются),
`TestSetupSigningFingerprint` (`:127`), `TestSetupSigningPermsRejected`
(`:179`), `TestWriteSigningPropertiesChmod` (`:202`),
`TestConfigureSigningFingerprintEnv` (`:226`).

## 8. PENDING и за пределами темы

1. **`docs/RELEASE-KEYS.md` не описывает `TNP_SIGNING_KEY_SHA256`** —
   новый обязательный вход release-сборки известен только из вывода
   `configure` и этого отчёта. Правка документации — P3 мандата (вне
   темы).
2. **Отпечаток — файла, не сертификата.** При миграции keystore на PKCS12
   (P2) файл меняется, отпечаток придётся пересчитать и раздать заново.
   Плюс: файл-отпечаток ловит и подмену пароля/пересборку контейнера.
3. **`testReleaseUnitTest` / `lintRelease`** по-прежнему требуют ключи:
   это следствие подстрочного `wantsRelease`, которое мандат велел
   сохранить (рекомендация аудита «(в) условие по `taskGraph`» — не в
   этой теме).
4. **`./gradlew assemble` и `build` теперь падают** без боевых ключей
   (шаг 4). Это осознанное изменение поведения: раньше молча собирался
   неподписанный release-APK. CI не затронут (`flutter.yml` собирает
   только debug, `android.yml` — проект этапа 6 без подписи).
5. **`android.injected.signing.*`** (свойства AGP из мастера Android
   Studio) перекрывают DSL: `verifySigning` проверяет ключ из DSL, а
   подписан может оказаться другой. Для CLI-сборок не воспроизводится;
   аудит это тоже не проверял.
6. **Каталог ключей** (0700) и **изоляция подписи на отдельном хосте**
   (главный практический риск §1.7 — цепочка поставок сборки) — P2,
   решение владельца.
7. **Независимый канал отпечатков** (Zenodo / подписанный тег, контрольные
   суммы APK) — P3, вне темы.

## 9. Что дальше

Ветка `fix-release-keys` запушена и готова к авто-merge при зелёном CI.
Merge не делал (мандат). `docs/KNOWLEDGE-*`, `spec/`, `README.md` не
менялись; секретов, ключей и паролей в коммитах нет (в тестах — только
фиктивные файлы-заглушки).
