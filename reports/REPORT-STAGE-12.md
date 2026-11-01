# Отчёт: этап 12 — профиль №8 «Абонентский ресурс»: recurring-единицы и идентификация сети

Ветка `stage12-recurring` (от main `9991835`), session-id:
`116a5b0-…` (продолжение `5e38d1f` после рестарта ОС). Мандат:
`prompts/stage-12-prompt.md`. Спека v9.10 (docs/KNOWLEDGE.md + ENG) — эталон,
сверена, не правилась. Документация (user-docs, programming-docs) не тронута.

## 1. Перечень изменений

### Шаг 2 — ядро

| пункт | файл | что сделано |
| --- | --- | --- |
| genesis-поля | core/wire/genesis.go, genesis_decode.go | `ProfileParams.Recurring` (поле 60), `Manifest.Slug` (поле 7), `TransferModePeerRedeem` (= 4) |
| деривация network_id | core/wire/genesis.go | `DeriveNetworkID` = SHA-256(canonical(Manifest) ‖ canonical(GenesisRecord)), оба с пустым `network_id` (иначе круг: network_id входит в оба кодирования); `FinalizeGenesis` (сборка), `VerifySelfCertified` (проверка), `SetManifestHash` (тестовые genesis); константа `GlobalNetworkID = "global"`, `IsGlobalNetwork`, `EqLabel` |
| genesis-валидация комбинаций | core/profiles/profiles.go | `Validate`: recurring ⇒ quota(resource_id) + transfer_mode ∈ {restricted, peer-redeem}; `ValidateGenesis`: network_id непуст, global+rec запрещён (комбинации global+rec не существует) |
| сгорание по периодной метке | core/ledger/state.go, validate.go, modules.go, settle.go | `RecurringSpendable`/`RecurringSpend`/`CreditCohorts` (получателю доли приходят с метками периода отправителя); `recurringSend` в transfer и redeem: перевод единицы периода P отвергается после границы P+1 — `E_BURNED` (остаток есть, метка просрочена; отличается от `E_BALANCE`); сброса баланса нет — сгоревшие когорты остаются в кошельке, история публична; при recurring settlePeriod НЕ сжигает когорты (сгорание — отвержением перевода, не сбросом) |
| peer-redeem | core/ledger/validate.go | перевод участник → участнику (получатель — зарегистрированный кошелёк, иначе `E_TRANSFER_RECIPIENT`); погашение участник → поставщик — через `redeem` (6.2) |
| шаблон профиля №8 | core/profiles/profiles.go | `subscriber-resource`: quota(resource), quota_reset, recurring=on, peer-redeem, invite_only, redeem-модуль; генерирует валидный genesis-пакет (проверено прогоном) |
| коды/спека | spec/00, spec/02, spec/03 | `E_BURNED`; T9/T10 (transfer), D3 (redeem); деривация network_id + slug в 2.1/2.6 |

Тесты (core/ledger/recurring_test.go, core/wire, core/profiles): граница P+1
(внутри окна accept, после — E_BURNED; полученные единицы несут метку и тоже
сгорают), новая квота нового периода переводима, E_BALANCE ≠ E_BURNED,
peer-redeem (участнику accept, неизвестному E_TRANSFER_RECIPIENT, redeem
поставщику accept, после границы E_BURNED), replay между сетями (E_NET),
`network_id = "global"` участвует в валидации так же, комбинации genesis
(global+rec, recurring+free/soulbound/не-quota/без resource_id),
детерминизм/подделка деривации, round-trip новых полей.

### Шаг 3 — конфигуратор (cmd/tnp-testnet configure)

- Флаги: `-slug` (обязателен для не-глобальной), `-global`, `-recurring
  auto|on|off`, `-display-name` (по умолчанию slug, предупреждение при >24
  симв.), `-logo` (PNG), `-app-version`, `-providers` (hex-адреса),
  `-provider-seed` (поставщик = кошелёк посевного круга, адрес из манифеста).
- Сборка: `FinalizeGenesis` (network_id = H(manifest ‖ genesis); глобальная —
  константа "global"), `ValidateGenesis` + `VerifySelfCertified` — ошибочные
  комбинации отвергаются с понятными ошибками (артефакт 03: global+rec).
- Артефакты Android: `android.json` + `icon/` — имя APK
  `equality-{global|slug}-{std|rec}-v{версия}.apk`, applicationId
  `io.equality.<slug>` / `io.equality.global`, label `Eq-` + 8 hex
  SHA-256(network_id), display_name, adaptive-иконка вариант 3: background —
  детерминированный цвет из палитры (~16 цветов) по SHA-256(network_id),
  foreground — логотип организатора или идентикон из network_id, rec-бейдж
  (круг с белым кольцом) в углу foreground, у std — нет (PNG 432×432).
- Подкоманды CLI: `redeem` (погашение поставщиком, 6.2), `-key-domain` у
  transfer/redeem, `-period 0` отличим от незаданного, `senderAddr` (адрес
  посевного кошелька из манифеста, см. §3).

### Шаг 4 — приложение (Flutter)

- Мост (mobile/query.go): `network_id` — hex, + `slug`, `eq_label`
  (Eq-+8 hex SHA-256(network_id)), `recurring`; transferModeName знает
  peer-redeem.
- models.dart: `slug`, `eqLabel`, `recurring`, `displayPair`
  («Monte Cloud · Eq-34757f49») — display_name всегда в паре с Eq-хешем,
  никогда отдельно.
- Экран «О сети» (настройки): пара display_name · Eq-хеш, network_id (hex),
  slug, строка recurring (вкл/выкл) + пояснение сгорания; онбординг — строка
  «Сеть: пара» на первом шаге; карточка восстановления — пара вместо
  голого display_name.
- Перевод: `E_BURNED` → локализованное сообщение (tx_errors.dart), подсказка
  о границе периода в recurring-сети; PIN-only на деструктивных экранах не
  тронут.
- i18n ×5 (ru/en/es/pt/zh): aboutNetwork, networkSlug, paramRecurring,
  recurringOn/Off, recurringNote, burnedUnitError, obNetworkLine;
  networkLine удалён (не используется).
- Android-сборка: `applicationId` (`-PTNP_APPLICATION_ID`) и label
  (`-PTNP_APP_LABEL`, placeholder `${appLabel}`) — параметры сети при сборке
  APK; в launcher — только label (Eq-хеш), display_name живёт внутри
  приложения.

### Шаг 5 — эмуляторный прогон (reports/stage12/, 31 артефакт)

Среда: opensuse tumbleweed (после переустановки ОС с linuxmint 22.3),
эмулятор API 36 google_apis x86_64. Находки среды: SwiftShader бандла
эмулятора сегфолтит на glibc 2.44 (обход: Xvfb :99 + `-gpu host`, системный
Mesa); playstore-образ требует диалог adb-авторизации (недоступен headless) —
взят google_apis; firewalld после рестарта блокировал порты libp2p
(остановлен на сессию). Уборка после прогона: эмулятор, Gradle-демоны, Xvfb,
узлы — остановлены, pgrep пуст.

Сеть rec (профиль 8, recurring=on, peer-redeem; slug `omega`,
network_id `b3dc3f07…`, label `Eq-34757f49`, APK
`equality-omega-rec-v1.0.0.apk`, applicationId `io.equality.omega`,
display_name «Monte Cloud»; genesis 2026-08-02, периоды 30 сут):

| шаг | артефакт | результат |
| --- | --- | --- |
| configure | 01 | имя APK rec, applicationId, label Eq-34757f49, display_name · Eq-хеш |
| выпуск квоты V1 периода 1 в окне челленджа (±48 ч) | 04 | accept/OK |
| негатив: V2 вне окна | 05 | reject/E_POW_HARD |
| регистрация W1 | 06 | accept |
| V1 → W1 400 (peer-redeem, внутри периода) | 07 | accept |
| V1 → кошелёк приложения 200 | 15 | accept |
| экспорт журнала (кошельки, когорты с метками) | 17 | W1: cohorts 400 (метка периода 1) |
| W1 → V2 100 (участнику, внутри окна) | 18 | accept |
| W1 redeem 100 → поставщик V1 | 19 | accept |
| приложение: онбординг (строка сети), регистрация (PoW+биометрия), карточка восстановления | 08–10, 16 | пара «Monte Cloud · Eq-34757f49» на всех трёх экранах |
| приложение: экран «О сети» | 13 | display_name · Eq-хеш, network_id, slug, recurring on, пояснение |
| приложение: перевод 100 внутри окна (PIN+биометрия) | 20, 21 | accept, «Transfer sent» |
| после границы P+1 (период 3): V1 transfer | 22 | reject/**E_BURNED** |
| W1 transfer | 23 | reject/**E_BURNED** |
| W1 redeem | 24 | reject/**E_BURNED** |
| приложение: перевод сгоревшей единицы | 25 | локализованная ошибка «The unit has burned: … (E_BURNED)» |

Сеть std (тот же шаблон, recurring=off; slug `rho`, label `Eq-1a2b9c21`, APK
`equality-rho-std-v1.0.0.apk`): выпуск V3 — accept (26), регистрация W1 —
accept (27), V3 → W1 400 — accept (28); после границы периода: перевод V3 —
accept (29), перевод W1 — accept (30): полученные и «потронутые» единицы
(Liquid) в std-режиме не сгорают — перетекание остатка в новый период
возможно; recurring закрывает эту дыру (E_BURNED в rec-сети). Имена APK
различаются: `equality-omega-rec-…` / `equality-rho-std-…` (артефакты 01/02).

Глобальная сеть: `configure -template 8 -global` — отвергается при сборке
(артефакт 03): «recurring в глобальной сети (4: нет организатора,
определяющего период; комбинация global+rec не существует)».

## 2. Расхождения со спекой и интерпретации (спека не правилась)

1. **manifest_hash и цикличность деривации.** `network_id = H(manifest ‖
   genesis)`, но genesis содержит manifest_hash, а manifest содержит
   network_id — прямое вычисление круговое. Реализация: manifest_hash =
   SHA-256(манифест **с пустым network_id**), network_id =
   SHA-256(оба кодирования с пустым network_id). Проверка узла
   (ledger.New, spec/02 2.6) переведена на ту же конвенцию. Зафиксировано в
   spec/02; спека v9.10 байтовую формулу не задаёт.
2. **Адреса посевного круга при само-сертификации.** Адрес кошелька =
   H(fingerprint ‖ network_id), но network_id зависит от манифеста с
   посевным кругом — адреса посева не могут быть выведены из финального
   network_id. В тестовом конфигураторе адреса посева фиксируются по
   предварительному network_id (до добавления круга); CLI решает через
   `senderAddr` (адрес из манифеста по публичному ключу). В живых сетях
   адреса посевного круга приходят с устройств организаторов до сборки
   генезиса — вопрос semantics для владельца/документации.
3. **Поставщики в genesis — тот же круг.** Адрес поставщика в параметрах
   сети не может быть адресом кошелька этой же сети, выведенным по правилу
   H(fingerprint ‖ network_id) (см. п.2). В прогоне поставщик — кошелёк
   посевного круга (`-provider-seed`, адрес уже в манифесте). Для живых
   сетей: указывать поставщика до сборки генезиса (адрес из другой сети
   невозможен по правилу адресации) — вопрос владельцу.
4. **Первая квота посевного круга при quota_reset.** `last_claimed_period
   = 0` — сентинел «ничего не заявлено»: заявка периода 0 неотличима от
   повторной (M3), а период 1 даёт diff = 2 (uint64-перенос) → E_ACCRUAL_WINDOW
   — посевной круг не мог выпустить ни одной квоты. Фикс: посевному кругу
   `ActivatedPeriod = 1` (первая заявка — период 1, diff = 1, amount =
   эмиссия × 1). Побочно убран артефакт переноса (демо tnp-node и тесты
   мintили 2000 = эмиссия × 2).
5. **Окно свежести vs жизнь квоты.** `fresh_transfer_window_periods = 3`
   (base) длиннее жизни квоты при quota_reset: в std-режиме единица сгорала
   бы на границе периода, не созрев. Шаблон 8: окно 0 (квота переводима
   сразу). Тот же конфликт есть у шаблона 3 (этап 10) — не трогал (вне
   мандата), к решению владельца.
6. **std-режим: перетекание остатка.** Подтверждено прогоном: в std-сети
   «потронутые» единицы (Liquid) не сгорают на границе периода — остаток
   перетекает в новый период (артефакты 29/30). Сгорание сбросом
   (settlePeriod) действует только на нетронутые когорты. Ровно эта дыра
   закрывается recurring (E_BURNED, метки у получателя) — соответствует
   мотивации v9.10 («бизнесу невыгодно перетекание остатка»).
7. **slug с дефисом → невалидный applicationId.** Android запрещает дефисы
   в имени пакета: `io.equality.monte-cloud` — AAPT error (найдено при
   сборке APK). Нужен запрет дефисов в slug или правило трансформации —
   вопрос владельцу (примыкает к PENDING о коллизиях applicationId).
8. **Палитра Equality.** Спека задаёт «палитра Equality, ~16 цветов» без
   состава. Выбран черновой набор из 16 цветов (cmd/tnp-testnet/android.go);
   индекс = первый байт SHA-256(network_id) mod 16 — одна сеть всегда один
   цвет. Утверждение тонов — владельцу.
9. **«В некоммерческих сборках режим не валидируется» (16.2).**
   Интерпретировано как отсутствие технической проверки лицензии (16.3:
   DRM и проверка лицензии в приложении противоречат принципам):
   конфигуратор валидирует комбинации, rec-маркер в имени APK делает
   коммерческие сборки видимыми. Технической блокировки recurring в
   «некоммерческих сборках» нет.
10. **Окно челленджа №39 и quota_reset.** Посевному кругу нужно ≥2
    завершённых периодов для vouch (V3); при quota_reset (diff = 1) окна
    периодов 1 и 2 должны быть достижимы подряд — на практике почти
    недостижимо (слоты хеш-зависимы). В прогоне активация новых участников
    через vouch не выполнялась (сценарий не требует); участники получали
    квоту переводом от посевного кошелька. Усугубление №39 — к владельцу.

## 3. PENDING (за владельцем, не додумано)

1. **Ключи подписи APK** (по мандату): единый ключ проекта для std, отдельный
   для rec, или ключ на сеть от организатора. В конфигураторе явная точка
   конфигурации подписи отсутствует — сборка этапа 12 подписана debug-ключом;
   решение фиксирует владелец до релизных сборок.
2. **Коллизия applicationId при равных slug** (по мандату): `io.equality.<slug>`
   у двух сетей разных организаторов — Android перезапишет чужое приложение
   вместе с данными кошелька. Вариант с хешем (`io.equality.<hash8>` /
   `io.equality.<slug>.<hash8>`) — на решение владельца.
3. **Дефисы в slug** (п.2.7): запрет или трансформация для applicationId.
4. **Состав палитры Equality** (п.2.8).
5. **Поставщики в genesis при само-сертификации network_id** (п.2.3).

## 4. Тесты

- `go test ./...` — зелёный (все пакеты).
- `go test -race ./core/wire/ ./core/profiles/ ./core/ledger/
  ./cmd/tnp-testnet/` — зелёный.
- `flutter analyze` — чисто; `flutter test` — 43 зелёных.
- Известный флаки: `net/TestSyncAnchorContinuation` — timing-чувствительный
  под полной загрузкой (`-count=3` в одиночку зелёный), не связан с этапом 12.

## 5. Дерево

Чистое; бинарники сборок не коммитились (APK — в app/build, вне git);
артефакты прогона — reports/stage12/ (31 файл: логи configure/CLI, скриншоты
приложения, экспорт журнала). AAR пересобран gomobile bind из ветки
(android/libs/tnp.aar, не в git).
