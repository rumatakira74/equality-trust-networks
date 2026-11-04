# Промпт фикс-темы fix-d2-clamp — AUDIT-MINT-FIX-02 (D2): кламп claim по accrual_window в mobile-мосту

Ветка: `fix-d2-clamp` от свежего `main` (рабочий клон /home/projects/audit-equality-trust-networks; личный клон не трогать). Кодовая фикс-тема: малый кламп в mobile/query.go; ядро (core/) и протокол НЕ менять.

1. Долг (REPORT-AUDIT-MINT-FIX.md §8, AUDIT-MINT-FIX-02; подтверждён аудитом pre-live как D2): mobile/query.go:929 — claim = period - 1 без клампа; accrual_window отдаётся в JSON (:953), но для клампа не используется. Кошелёк, пропустивший ≥ accrual_window периодов, получает «отравленный» claim: UI предлагает claim, ядро на подачу отвечает E_ACCRUAL_WINDOW.
2. Фикс: клампить claim по эталонной семантике ядра — сверить точную формулу с core/ledger/validate.go:150-196 (CheckMintChallengeWindow) и повторить её в MintWindowJSON (например claim = min(period-1, last + accrual_window - 1) — окончательную формулу взять из кода ядра, чтобы мост и ядро совпадали побуквенно). Комментарии по-русски.
3. Тесты (Go, mobile-мост): (а) свежий кошелёк — claim не изменился; (б) пропуск < accrual_window — кламп по формуле ядра; (в) пропуск ≥ accrual_window (например 7+) — UI-claim равен клампу ядра, подача не встречает E_ACCRUAL_WINDOW. go test ./mobile/... зелёные; go build ./... rc=0.
4. (Опционально, только если остаётся время окна) смоук на эмуляторе по скиллу emu-run: экран mint до/после — скриншоты в reports/fix-d2-clamp/; обязательная уборка по скиллу. Для merge не требуется.
5. Отчёт: reports/REPORT-FIX-D2-CLAMP.md: формулы до/после, file:line, тесты и результаты, session-ID.

Правила: чекпоинты событийные + ~60 мин (session-ID), чистое дерево, пуш ветки, НЕ мержить. Модель и окно — YAML-шапкой задачи очереди.
