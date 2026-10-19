// Интерфейсы, реализуемые на этапах 4–5 (сеть и консенсус).
// Этап 3 — headless: время внедряется тестами, финализация — заглушка.
package ledger

// EpochSource — источник сетевого времени (9.2: источник правды — эпохи
// чекпоинтов, не медиана timestamps). Реализация — этап 5.
type EpochSource interface {
	NetTime() uint64 // сетевое время, Unix-секунды UTC
	Epoch() uint64   // номер текущей эпохи
	Period() uint64  // номер текущего календарного периода выпуска (месяц)
}

// FixedClock — фиксированное время для тестов и векторов.
type FixedClock struct {
	T, E, P uint64
}

func (c *FixedClock) NetTime() uint64 { return c.T }
func (c *FixedClock) Epoch() uint64   { return c.E }
func (c *FixedClock) Period() uint64  { return c.P }

// Advance — сдвиг времени (детерминированный, для сценариев).
func (c *FixedClock) Advance(t uint64) { c.T = t }

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
