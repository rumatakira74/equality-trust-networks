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
