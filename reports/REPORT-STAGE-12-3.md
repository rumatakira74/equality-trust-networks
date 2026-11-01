# Отчёт мини-этапа 12.3 — фикс applicationId: префикс x перед hash8

Дата: 01.11.2026. Ветка `stage12-3-appid` от `4a0687d` (main содержит
merge 12.2 `1e47bf5`). Мандат: `prompts/stage-12-3-prompt.md`
(решение владельца: префикс `x` перед hash8 — без маппинга, инъективно).

## Итог

applicationId = `io.equality.<sanitized-slug>.x<hash8>` (глобальная —
`io.equality.global.x<hash8>`): префикс «x» ставится всегда — сегмент
Android-пакета обязан начинаться с буквы, hash8 начинается с hex-цифры
в 10/16 случаев (62,5% — находка прогона 12.2: AAPT отверг
`io.equality.cirrus.783da646`). С префиксом валидность гарантирована
для любой сети. Изменение только в конфигураторе; wire, палитра,
label, имена APK, display_name не тронуты. Сборка APK end-to-end для
сети с цифрой в начале hash8 — AAPT принимает пакет.

## 1. Старое/новое

| | Старое (12.1) | Новое (12.3) |
| --- | --- | --- |
| Формула | `io.equality.<sanitized-slug>.<hash8>` | `io.equality.<sanitized-slug>.x<hash8>` |
| Глобальная | `io.equality.global.<hash8>` | `io.equality.global.x<hash8>` |
| Валидность Android | сегмент `<hash8>` с hex-цифры в начале (10/16) — AAPT error | сегмент всегда начинается с буквы `x` — валиден для любой сети |
| Инъективность | хеш в суффиксе | та же: префикс фиксированный, без маппинга |
| label / APK / display_name | `Eq-<hash8>`, `equality-…-v…apk`, slug | не изменились |

Реализация: `cmd/tnp-testnet/android.go` `ApplicationID` — единственная
точка генерации (hash8 := "x" + hex(SHA-256(network_id)[:4]));
вызов в `configure.go` не менялся. Комментарий обновлён (правило
Android, находка 12.2, инъективность).

## 2. Кейсы (буква/цифра в начале hash8)

- **Цифра**: сеть из прогона 12.2, где применялся обход «подбор с
  буквы» — network_id `7cd5b877…` (slug `cirrus`, genesis-time
  1790807400), hash8 `783da646`: applicationId
  `io.equality.cirrus.x783da646` — валиден (тест
  `TestApplicationIDLetterPrefix`, фикстура `net783da646`).
- **Буква**: сеть прогона 12.2 после обхода — network_id `7aab00e6…`,
  hash8 `e5796f9c`: `io.equality.cirrus.xe5796f9c` — валиден, префикс
  ставится и здесь (единообразие).
- **Любая сеть**: `TestApplicationIDAnyNetworkValid` — 64
  детерминированных network_id × slug {monte-cloud, abc, global}: все
  пакеты проходят правило «сегмент с буквы».
- Детерминизм (тот же network_id → тот же пакет), глобальная
  (`io.equality.global.x…`), регрессия санитизации slug (дефис → `_`),
  неизменность label `Eq-<hash8>` и имён APK —
  `TestApplicationIDUnchangedArtifacts` + обновлённые тесты 12.1/12.2
  (`TestApplicationIDSlugHash`, `TestConfigureProfile8Rec`,
  `TestConfigureGlobalStd`).

## 3. Сборка APK end-to-end (reports/stage12-3/)

Сеть с цифрой в начале hash8 (та же, что падала в 12.2):

- `01-configure-digit-hash8.txt` — configure: applicationId
  `io.equality.cirrus.x783da646`, label `Eq-783da646`, APK
  `equality-cirrus-rec-v1.0.0.apk` (не изменились).
- `02-apk-build-digit-hash8.txt` — `gradlew assembleRelease
  -PTNP_APPLICATION_ID=io.equality.cirrus.x783da646
  -PTNP_APP_LABEL=Eq-783da646`: **BUILD SUCCESSFUL**, `aapt dump
  badging`: `package: name='io.equality.cirrus.x783da646'`,
  `application-label:'Eq-783da646'`. Контроль: та же сборка без
  префикса x падала в 12.2 (AAPT error, PENDING 1 отчёта 12.2).

Обход «подбор сети с буквы» удалён: он жил только в скриптах подбора
genesis прогона 12.2 (вне репозитория, /tmp); в коде и тестах репо
его не было — вместо него в тесте зафиксирована сеть с цифрой
(`net783da646`). Gradle/Kotlin-демоны после сборки остановлены.

## 4. Тесты

- `go test -count=1 ./...` — зелёный (13 пакетов; в первом прогоне
  единичное падение `net` — известный timing-фактор под полной
  нагрузкой, в одиночном прогоне пакета и в повторном полном — зелёный;
  код net в 12.3 не менялся).
- `go test -race ./cmd/tnp-testnet/` — зелёный (задетый пакет).
- flutter analyze/test не выполнялись — Dart/Kotlin не менялись.

## 5. Для спеки (не правилась — отдельный заход)

Формула applicationId в спеке v9.10/v9.11 не зафиксирована (applicationId
— артефакт конфигуратора, не протокола); при обновлении документации
(раздел 16/конфигуратор) указать: `io.equality.<sanitized-slug>.x<hash8>`,
глобальная — `io.equality.global.x<hash8>`, где hash8 = первые 8 hex
SHA-256(network_id) — тот же хеш, что в label Eq-<hash8>; префикс x —
требование Android к первому символу сегмента пакета.

## 6. PENDING

Новых нет. PENDING 1 отчёта 12.2 (applicationId с hex-цифры) закрыт
настоящим этапом; остальные PENDING 12.2 (E_BURNED/мораторий —
календарные границы, среда gomobile/эмулятора) — без изменений.

## 7. Дерево

Чистое. Артефакты — reports/stage12-3/ (2 файла); APK в git не входит
(app/build, вне репо).
