# Промпт фикс-темы fix-mint-window — окно mint: вариант A + дефект M5 quota_reset + джиттер

Ветка: `fix-mint-window` от свежего `main` (рабочий клон; личный клон не трогать; `audit*` запрещён). Кодовая тема, консенсусная (новое поле манифеста).

Источник: reports/DIGEST-AUDIT-SONNET-2.md в main; полный отчёт: git show origin/audit-sonnet:reports/audit-sonnet/REPORT-AUDIT-SONNET-2.md (секция про окно mint и модель нагрузки); готовые тесты: origin/audit-sonnet:reports/audit-sonnet/go-tests/ (mintslotfix).

Шаги (проект фикса «вариант A» из аудита):
1. Манифест-поле mint_slot_mode (номер поля 62; 0 = наследие, 1 = слот от периода выпуска). protocol_version НЕ трогать; энкодер опускает нулевые поля — старые сети не ломаются.
2. Дефект M5 quota_reset: заявлять любой завершённый mp.Period > LastClaimedPeriod суммой IssuanceAmount (пропуск периода не должен навсегда убивать выпуски).
3. Клиентский джиттер: равномерный в max(S−W, boundary+jitter) — без него пик 49×.
4. spec/02: описание поля манифеста; QUESTIONS.md №10: заменить старую формулировку на mint_slot_mode (осознанное расхождение со спекой 7.4 фиксируется в QUESTIONS.md).
5. Тесты: mintslotfix из go-tests — красные на входе, зелёные после; характеризационные тесты (которые зелёные и кодируют старое поведение) — удалить; go build ./... rc=0; go test ./... зелёные; go test -race ./core/ ./net/ зелёные.

Отчёт: reports/REPORT-FIX-MINT-WINDOW.md на ветке: план, file:line, результаты модели (окно/пик), session-ID.

Правила: чекпоинты событийные + ~60 мин (session-ID), чистое дерево, пуш ветки, НЕ мержить. Модель и окно — YAML-шапкой.
