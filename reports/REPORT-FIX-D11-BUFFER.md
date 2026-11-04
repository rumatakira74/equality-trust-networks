# REPORT-FIX-D11-BUFFER — офлайн-буфер публикации переживает рестарт

Тема: D11 (долг: `FINDING-awaiting-offline-tx-not-persistent.txt`;
`reports/REPORT-STAGE-16-NODE-PERSISTENCE.md` §7). Дата: 04.11.2026.
Ветка: `fix-d11-buffer` от main `0c89331`.

Проблема: офлайн-буфер публикации (`awaiting`, Bluetooth 12.4) жил только
в памяти (`net/node.go`: `OnOfflineTx` / `awaiting` / `PublishAwaiting`) —
при force stop получатель видел «принято», но платёж не был опубликован
и терялся навсегда.

## Дизайн (минимум; протокол и консенсус не тронуты)

- Таблица **`awaiting_txs`** в существующем SQLite-store (слой этапа 16,
  стиль существующих миграций — `core/store/sqlite.go:73-79`):
  `txid BLOB PRIMARY KEY, raw BLOB NOT NULL, added_at INTEGER NOT NULL`.
  Дедуп по txid (PK); порядок восстановления — `rowid` (поступление).
- **Запись при приёме** — `OnOfflineTx` (`net/node.go:962-981`): под тем же
  `n.mu`, что и вставка в память (память и store согласованы), только для
  новых txid; `SaveAwaitingTx` — одиночный `INSERT OR IGNORE` (атомарен —
  транзакционность обеспечена на уровне SQLite-оператора).
- **Восстановление при старте** — `restoreAwaiting`
  (`net/node.go:983-997`), вызывается из `NewNode` для **обоих уровней**
  (офлайн-приём есть и у лёгкого узла, 12.2/12.4): до старта хоста, при
  ошибке — отказ `NewNode` (как у restoreLight).
- **Удаление после обработки** — `PublishAwaiting`
  (`net/node.go:999-1050`): все три ветки снятия с ожидания (принята/
  конфликт, E_DUP, невалидная) синхронно удаляют запись из store
  (`deleteAwaitingFromStore`, `net/node.go:1052-1062`); ошибка удаления
  не фатальна (после рестарта буфер восстановит tx, следующая публикация
  снимет её как E_DUP/невалидную) и журналируется. Сетевая ошибка
  публикации — `continue`: транзакция остаётся в памяти и store.
- Границы: `wire`, `ledger`, `cons` не менялись; сетевой протокол не
  менялся.

## Изменения (file:line по ветке fix-d11-buffer)

- `core/store/sqlite.go:73-79` — таблица `awaiting_txs` в `migrate()`;
  `:232-269` — `SaveAwaitingTx` / `LoadAwaitingTxs` / `DeleteAwaitingTx`.
- `net/node.go:962-981` — `OnOfflineTx` с записью в store;
  `:983-997` — `restoreAwaiting`; `:1006-1010` — вызов из `NewNode`;
  `:1011-1050` — `PublishAwaiting` с удалением из store;
  `:1052-1062` — `deleteAwaitingFromStore`.

## Тесты

Юнит (`core/store/sqlite_awaiting_test.go`):
- `TestStoreAwaitingRoundTrip` — запись двух tx, порядок (rowid), дедуп
  повторной записи, удаление по txid, удаление несуществующего — не ошибка.
- `TestStoreAwaitingReopen` — Close/Open той же БД: буфер переживает
  рестарт, txid и canonical-байты совпадают.

Интеграционный (`net/awaiting_persistence_test.go`):
- `TestAwaitingBufferSurvivesRestart` — журнал (register SENDER, register
  RECVR, минт посевного V1, перевод V1→SENDER) через `ImportJournal`;
  офлайн-приём перевода SENDER→RECVR (`OnOfflineTx`) → статус
  `OfflineAwaiting`, 1 запись в store; повторный приём — дедуп; **force
  stop** (`Close`); **рестарт** — новый экземпляр `NewNode` на той же БД:
  статус `OfflineAwaiting`, 1 запись в store; `Start` + `PublishAwaiting`
  → `OfflinePublished`, буфер пуст и в памяти, и в store, перевод в
  реестре получателя (Liquid 100).

## Результаты прогонов (04.11.2026, main-клон, Go 1.27.3)

- `go build ./...` — ok; `go vet ./...` — чисто; `gofmt -l .` — пусто.
- `go test ./... -count=1` — все пакеты ok (core/ledger 68,9 с, net 71,6 с,
  sim 140,5 с — с новыми тестами).
- `go test -race ./net/ ./cons/ ./mobile/ -count=1` — ok (net 152,5 с).
- Новые тесты: `TestStoreAwaitingRoundTrip`, `TestStoreAwaitingReopen`,
  `TestAwaitingBufferSurvivesRestart` — PASS.

## Session-ID

Сессия zcode: sess_3f637418-634e-4c4a-af66-37782647233a (исполнитель —
сессия завершения предрелизного этапа, промпт владельца «fix-d11-buffer +
fix-a7-claims + релиз v3.0.0»).
