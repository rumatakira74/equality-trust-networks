// Интерфейсы, реализуемые на этапах 4–5 (сеть и консенсус).
// Этап 3 — headless: финализация — заглушка.
//
// v9.13 (этап фикса времени): ядро не читает часы узла. Время валидации —
// txNow = timestamp_claim самой транзакции; эпохи/периоды — арифметически
// от genesis. Отдельного «режима часов» replay нет: live и replay
// применяют журнал одним механизмом (ledger.Apply). Часы остаются у сети:
// cons.Clock (закрытие эпох) и net.RealClock (отображение, допуск C9).

package ledger

// Finalizer — разрешение конфликтных веток (9.1: выборочное голосование;
// этап 5). Headless-заглушка выбирает первую поступившую ветку.
type Finalizer interface {
	// Choose возвращает txid ветки-победителя из конфликтной пары.
	Choose(a, b TxRef) TxRef
}

// TxRef — ссылка на транзакцию (кошелёк + seq + txid).
type TxRef struct {
	Wallet []byte
	Seq    uint64
	TxID   []byte
}

// FirstWins — заглушка финализации: побеждает первая по порядку поступления.
type FirstWins struct{}

func (FirstWins) Choose(a, b TxRef) TxRef { return a }

// CheckpointStore — хранение заголовков чекпоинтов (этап 5).
type CheckpointStore interface {
	SaveCheckpointHeader(epoch uint64, hash []byte) error
	LatestCheckpointHeader() (epoch uint64, hash []byte, ok bool)
}
