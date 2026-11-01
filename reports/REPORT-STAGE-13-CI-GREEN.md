# Отчёт этапа 13 — CI-зелёный (ветка stage13-ci-green)

Дата: 2026-11-01/02. Мандат: prompts/stage-13-ci-green-prompt.md (cc6747e +
уточнение e7f42d1). Основание: аудит от 01.11.2026 (reports/AUDIT-equality-trust-networks.md,
HEAD f6deb05) — на main красный весь CI, кроме cons.

## 1. Что было красным и почему

| Workflow / job | Причина (по аудиту и логам прогонов) |
| --- | --- |
| core/test | gofmt: 7 файлов core/ не отформатированы; шаг gofmt падал за секунды, go vet и go test не запускались вовсе |
| net/test | gofmt: 4 файла cmd/; далее (после снятия gofmt) — тайминговые флюки тестов под -race (см. п. 4) |
| android/aar | gomobile bind без `-ldflags "-checklinkname=0"` (F6/A11) + отсутствие каталога android/libs/ в свежем checkout (каталог в .gitignore, gomobile не создаёт промежуточные каталоги; локально существовал от прошлых сборок) |
| android/apk | `android/gradlew` при `working-directory: android/` — двойной путь `android/android/gradlew`, exit 127 (шаги wrapper / unit-тесты / сборка APK) |
| android/mobile-go | gofmt: mobile/query.go |
| flutter/analyze-test | по аудиту причина была неизвестна; в прогонах этапа 13 оказался зелёным (падение flutter-воркфлоу было в apk-джобе, см. ниже) |
| flutter/apk | то же отсутствие android/libs/ при gomobile bind |
| cons/test | зелёный и до этапа (единственный) |

## 2. Что сделано

### Задача 1 — gofmt (коммит 4ddbf84)
`gofmt -l .` до: 12 файлов — совпадает с аудитом (7 core/, 4 cmd/, mobile/query.go).
Выправлено `gofmt -w`; проверка «только форматирование»: `git diff -w` пуст,
143+/143-. После: `gofmt -l .` пуст.

Список файлов: cmd/tnp-testnet/{configure.go, configure_stage12_1_test.go,
export.go, main.go}, core/ledger/{modules_test.go, state.go,
update_guardians_test.go}, core/present/present.go, core/profiles/profiles.go,
core/wire/{state.go, wire.go}, mobile/query.go.

### Задача 2 — -race (коммиты 77ba85c, a44d660, d84775b)
`-race` добавлен в go test всех Go jobs (core, net, cons, android/mobile-go).
Найденные и починенные гонки — п. 4.

### Задача 3 — закрепление версий (коммит 77ba85c)
- gomobile/gobind: `v0.0.0-20261011034528-197770444867` (псевдоверсия =
  текущий @latest на дату этапа) вместо `@latest` — android.yml, flutter.yml.
- Flutter: `flutter-version: "3.47.9"` (stable, ревизия c197ca5199 — локальная
  среда разработки этапа 12) вместо плавающего `channel: stable` — flutter.yml.
- GitHub Actions — по SHA коммитов с комментарием тега: checkout v4.4.0
  (11d5960a…), setup-go v5.6.0 (40f1582b…), setup-java v4.9.1 (cf277c60…),
  upload-artifact v4.6.2 (ea165f8d…), download-artifact v4.3.0 (d3f86a10…),
  subosito/flutter-action v2.9.1 (9a48871c…) — все пять workflow-файлов.
- distributionSha256Sum: android (gradle-8.11.1-bin,
  f397b287…151c6) и app/android (gradle-9.3.1-all, 17f27786…c6e43).

### Задача 4 — checklinkname (коммит 77ba85c)
`-ldflags "-checklinkname=0"` добавлен в gomobile bind android.yml (закрывает
F6/A11; в flutter.yml флаг уже был).

### Задача 5 — падения flutter/android (коммиты d84775b, ca819e0)
- `mkdir -p android/libs` перед gomobile bind в android.yml и flutter.yml —
  каталог в .gitignore и не существует в свежем checkout CI.
- `./gradlew` вместо `android/gradlew` в трёх шагах apk-джобы android.yml
  (working-directory уже android/).

## 3. Статусы jobs CI на ветке (финал)

| Workflow | Job | Статус | Прогон (SHA, время) |
| --- | --- | --- | --- |
| core | test | success | a44d660, 3m20s (run 41315963889) |
| net | test | success | d84775b, 2m39s (run 41317583580) |
| cons | test | success | d84775b, 3m46s (run 41317583675) |
| flutter | analyze-test | success | d84775b, всего 11m30s (run 41317583525) |
| flutter | apk | success | там же |
| android | aar | success | ca819e0, всего 8m5s (run 41318133868) |
| android | apk | success | там же |
| android | mobile-go | success | там же |

Прогоны path-фильтрованы: каждый workflow зелёный на последнем коммите,
затрагивающем его пути (ca819e0 менял только android.yml — три остальных
workflow на нём не запускаются и их зелёные прогоны на d84775b актуальны).

## 4. Гонки, найденные -race (задача 2)

Подробности: reports/FINDING-lightnode-proof-race.txt.

1. TestLightNodeCheckpointProof (net): тайминговая гонка — предложение чекпоинта
   строится один раз на эпоху из состояния на тике cpLoop; под -race первый тик
   строителя опережал применение mint, чекпоинт №1 финализировался с корнем
   до mint, а доказательство берётся от текущего состояния. Фикс: тест ждёт
   чекпоинт с совпадающим корнем (цикл LightSync, 30 с; эпохи consEpoch=2).
   До фикса 3/8 красных, после 8/8 зелёных.
2. TestSyncAnchorContinuation (net): (а) тест ждал прокси-условие (длина
   журнала), а не само утверждение (якорь pos=3) — журнал наполняется gossip'ом,
   якорь двигает только сверка; (б) дата-рейс — anchorOf возвращал указатель,
   тест читал поля вне n.mu. Фиксы: waitFor по условию якоря; anchorOf
   возвращает копию под RLock. 30/30 зелёных.
3. TestSideChannelPublishPropagates (net): публикация офлайн-транзакции
   одноразовая, при E_SEQ (выпуск ещё не догнал издателя) транзакция остаётся
   в ожидании без буфера; тест не ждал догонания mint издателем. Фикс:
   waitJournal(b, 1) до публикации (как в соседнем тесте). 20/20 зелёных.

Все три — гонки тестового тайминга/хелперов; протокольная семантика не менялась.

## 5. Локальная проверка (задача 6)

- `gofmt -l .` — пусто.
- `go vet ./...` — чисто.
- `go test -race ./...` — все пакеты ok (net 74–120 с, sim 275 с, ledger 204 с).
- `flutter analyze` — No issues found; `flutter test` — 43/43 (Flutter 3.47.9).

## 6. Границы этапа

Находки аудита A1–A12 не чинились (предмет следующих этапов); правки — только
форматирование, CI-конфиги, закрепление версий, гонки -race и причины падений
jobs (включая код тестов). Спека не правилась. BLOCKED-external нет: все
падения оказались воспроизводимы и устранимы в репо.

## 7. Финализация

- Ветка stage13-ci-green запушена; merge в main — по подтверждению владельца.
- После прогона: Gradle-демоны остановлены, tnp-node/qemu проверены (pgrep),
  процессов не осталось.
