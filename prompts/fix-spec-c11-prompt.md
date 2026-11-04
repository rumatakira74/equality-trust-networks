# Промпт фикс-темы fix-spec-c11 — PLA-F-01/D1 (+PLA-F-02, PLA-F-03): spec/ и комментарии к факту кода v9.14/3

Ветка: `fix-spec-c11` от свежего `main` (рабочий клон /home/projects/audit-equality-trust-networks: git fetch origin && git checkout main && git reset --hard origin/main; git checkout -B fix-spec-c11 origin/main). Личный клон /home/projects/equality-trust-networks НЕ трогать. Префикс `audit*` запрещён. Тема документационная: привести spec/ и устаревшие комментарии к факту кода main; поведение кода НЕ менять (правки только тексты spec/ и комментарии в .go).

1. PLA-F-01 (средняя; REPORT-PRE-LIVE-AUDIT.md раздел 2, долг D1):
   - spec/03-validation.md:31 (C11): правило описывает якорь слота ЗАЯВЛЯЕМОГО периода; код v9.14 — якорь ТЕКУЩЕГО периода (core/ledger/validate.go:150-196, CheckMintChallengeWindow). Переписать C11 по факту кода: якорь — текущий период, формула слота H(address‖period) от якоря текущего периода. Сверить всю нормативную таблицу C1–C11 с validate.go построчно; реализация по старому тексту C11 давала бы дофиксное правило M7.
   - protocol_version: spec/00-overview.md:46 и spec/02-genesis-manifest.md:86 — «= 2»; код — ProtocolVersion: 3 (core/params/params.go:210, cmd/tnp-testnet/main.go:393). Исправить на 3 в обоих файлах.
2. PLA-F-02 (низкая): «2^24» против факта 2^40 (TargetLight): docs/KNOWLEDGE-5.md:130, комментарий core/params/params.go:180, QUESTIONS.md №9 («таргет 2^24 (0x0000010000000000…)» — сам hex равен 2^40). Erratum: «2^24» → «2^40»; hex-значения верны, их не трогать.
3. PLA-F-03 (низкая): core/crypto/address.go:1-11 — устаревший комментарий «PENDING: вопрос №2»; деривация зафиксирована v9.11 (адрес = SHA-256(fingerprint ‖ network_id), DeriveAddress). Обновить комментарий, PENDING убрать.
4. Проверки: go build ./... rc=0; go test ./core/... зелёные; git diff --stat — только .md и комментарии .go (кода-логики нет).
5. Отчёт: reports/REPORT-FIX-SPEC-C11.md на ветке: список правок file:line, цитаты до/после, результаты проверок, session-ID.

Правила: чекпоинты событийные + ~60 мин (session-ID), чистое дерево, пуш ветки после набора, НЕ мержить. Модель и окно — YAML-шапкой задачи очереди.
