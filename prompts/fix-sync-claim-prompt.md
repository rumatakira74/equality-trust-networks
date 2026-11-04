# Промпт фикс-темы fix-sync-claim — H1 аудита Sonnet-2: обход claim-барьера через sync

Ветка: `fix-sync-claim` от свежего `main` (рабочий клон; личный клон не трогать; `audit*` запрещён). Кодовая тема: закрыть H1 — высокий долг аудита Sonnet-2.

Источник: reports/DIGEST-AUDIT-SONNET-2.md в main; полный отчёт: git show origin/audit-sonnet:reports/audit-sonnet/REPORT-AUDIT-SONNET-2.md (секция про TimestampClaim/sync).

Долг (H1): claim-барьер C9 действует только на gossip и локальной публикации; SyncJournalFrom (и ImportJournal) применяет tx с произвольным TimestampClaim в Ledger.Apply → сеттлемент settleTo(max(txNow)) необратимо уводит NetTime вперёд (демередж/сгорание за «будущее»), + CPU-DoS циклом settle на эпохи; заражение расползается между пирами через sync.

Шаги (минимальный план аудита):
1. Немедленная граница: односторонний clamp claim ≤ now_узла + tol в SyncJournalFrom, ImportJournal, retryPending; tx с claim чуть впереди — НЕ терять: очередь отложенных с применением при наступлении времени.
2. Структурно: детерминированная граница от NetworkTime последнего чекпоинта + эпоха + maxShift (вместо/поверх локального now).
3. Ограничение сдвига settleTo за один вызов (анти-DoS).
Граница «по порядку журнала» НЕ годится (цепочка дешёвых tx сдвигает время). Протокол и консенсус НЕ менять (кламп на применении, не на валидности сети).

Тесты: юнит клампа (границы, tol); интеграционный: sync с «будущим» tx не двигает NetTime; отложенный tx применяется при наступлении; go test ./net/ ./cons/ зелёные; go test -race ./net/ зелёные.

Отчёт: reports/REPORT-FIX-SYNC-CLAIM.md на ветке: план, file:line, сценарии, результаты, session-ID.

Правила: чекпоинты событийные + ~60 мин (session-ID), чистое дерево, пуш ветки, НЕ мержить. Модель и окно — YAML-шапкой.
