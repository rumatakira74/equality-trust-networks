# Дайджест аудита REPORT-AUDIT-SONNET-2 (аудитор Claude Sonnet, 04.11.2026)

Источник: ветка audit-sonnet (7e6e500, 54d544e), reports/audit-sonnet/REPORT-AUDIT-SONNET-2.md. Объекты: ветка a8-keys (2bb50f3), main@0e44d83 (cons/, net/, core/ledger, core/attest). Полный отчёт и артефакты (window_load_model.py, pow_target_check.py, mint_window_model.py, go-tests/) — на ветке audit-sonnet.

## Вердикты и веса

| ID | Находка | Вес | Куда |
| --- | --- | --- | --- |
| C1 | Подделка чекпоинта: Threshold=0 принимается Verify без подписей; принимающий не сверяет Participants/Threshold с cons.Participants по манифесту; подписант подписывает любое предложение эпохи без сверки StateRoot. Итог: произвольный StateRoot/TotalSupply/NetworkTime/FinalizedTxIDs → выбор ветки эквивокации (двойная трата) | КРИТИЧНЫЙ | фикс-тема до release-v3 |
| H1 | Claim-барьер C9 только на gossip и локальной публикации; SyncJournalFrom (и ImportJournal) применяет tx с произвольным TimestampClaim в Ledger.Apply → сеттлемент settleTo(max(txNow)) необратимо уводит NetTime вперёд (демередж/сгорание за «будущее»), + CPU-DoS циклом settle на эпохи | ВЫСОКИЙ | фикс-тема до release-v3 |
| — | Окно mint: проект фикса вариант A (слот от периода выпуска) через НОВОЕ поле манифеста mint_slot_mode (номер 62; 0=наследие/1=период выпуска; protocol_version НЕ трогать — старые сети не ломаются, энкодер опускает нули); дефект M5 quota_reset (пропуск периода = мёртвые выпуски навсегда) требует отдельной правки M5; пик 49× без джиттера → нужен клиентский джиттер | высокий (дефект), дизайн-решение владельца | решение → фикс-тема |
| — | a8-keys: секретов в git/CI НЕТ; но отказ release без ключей не принудителен (wantsRelease по подстроке имён задач; assemble/aR → неподписанный APK без ошибки; TNP_SIGNING_PROPERTIES на любой JKS → подписанный чужим ключом); сверка отпечатка при сборке отсутствует; права файлов не принудятся; JKS слаб (hashcat 15500) — рекомендован PKCS12 ≥32 симв., изоляция подписи; RELEASE-KEYS.md недостаочен (нужен независимый канал отпечатков: Zenodo/подписанный тег, контрольные суммы APK в подписанном манифесте, ротация/отзыв) | P1 до публичных релизов, P2/P3 — по плану | тема fix-release-keys (P1) до release-v3 |
| V1 | Snowball PollRound не дедуплицирует голоса по кошелькам (сибиль по пирам) | СРЕДНИЙ | 8-live |
| — | Аттестация: origin=IMPORTED в аттестации — платформа помечает, публичный API не выдаёт; фикс-харденинг origin==GENERATED + keymasterSecurityLevel дёшев — «делать» | СРЕДНИЙ | 8-live или отдельная тема |
| — | Предсказуемый слот: атаки малоценны, сейчас НЕ трогать; расхождение со спекой 7.4 (слот от checkpoint_hash) зафиксировать в QUESTIONS.md как осознанное решение | НИЗКИЙ | мелочь в ближайшей docs-теме |
| — | Тесты не герметичны: configureCmd без -signing-dir в 6 вызовах → на машине владельца go test читает реальный credentials.txt и перезаписывает signing.properties | НИЗКИЙ | включить в fix-release-keys |
| — | A1 (лист для ключа A, регистрация ключа B): уже покрыт TestRegisterLeafKeyMismatch | — | закрыто |

## Готовые Go-тесты (на ветке audit-sonnet, go-tests/)

- cpforgefix: TestForgedCheckpoint_ZeroThresholdRejectedByVerify, ..._CollectorNotReady — красные до фикса C1.
- mintslotfix: TestMintWindowFixed_IssuingPeriodReachableForAll, ..._AccrualCatchUpAfterMiss — красные до фикса окна (вариант A).
- Характеризационные (зелёные сейчас) — удалить после фикса.

## Минимальные планы фиксов (из отчёта)

- C1: (1) вызывающий сверяет cp.Participants/Threshold с cons.Participants(...) по режиму манифеста, Threshold==0 отвергать; (2) подписант подписывает только при совпадении StateRoot со своим реестром; (3) проверка «эпоха наступила» и принадлежность строителю; (4) валидатор темы чекпоинтов. Консенсус-правило — поле манифеста/версия по образцу 3.2.
- H1: (1) немедленно — односторонняя граница claim ≤ now_узла + tol в SyncJournalFrom, ImportJournal, retryPending + очередь отложенных (не терять tx с claim чуть впереди); (2) структурно — детерминированная граница от NetworkTime последнего чекпоинта + эпоха + maxShift; (3) ограничение settleTo за вызов. Граница «по порядку журнала» НЕ годится (цепочка дешёвых tx сдвигает время).
- Окно: поле манифеста mint_slot_mode (62) + Encode/Decode + валидация mode∈{0,1} + configure default 1 + spec/02 + QUESTIONS.md №10; M5 quota_reset: заявлять любой завершённый mp.Period > LastClaimedPeriod суммой IssuanceAmount; джиттер клиента max(S−W, boundary+jitter).

## Решения владельца (ждут)

1. Добавить ли до release-v3 темы fix-cp-threshold (C1) и fix-sync-claim (H1) — рекомендация: ДА, обе до релиза (C1 = двойная трата, H1 = необратимое отравление времени).
2. mint_slot_mode вариант A + правка M5 quota_reset + джиттер — консенсус-дизайн, решение владельца.
3. fix-release-keys (P1: verifySigning SHA-256 свера + права файлов + герметичность тестов; P2/P3 в 8-live).
