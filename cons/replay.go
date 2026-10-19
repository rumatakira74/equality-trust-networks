// Финальность и replay (INV-3 с правкой этапа 5, spec/04): журнал хранения
// содержит обе конфликтные ветки; выбор финализации восстанавливается из
// finalized_txid_list чекпоинтов — победившая ветка применяется первой,
// альтернатива фиксируется как конфликт (и материал evidence, 7.5).
package cons

import (
	"fmt"

	"github.com/rumatakira74/trust-networks-platform/core/ledger"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// FinalityView — финализированные txid из finalized_txid_list цепочки
// чекпоинтов (публичное обязательство выбора веток).
type FinalityView struct {
	finalized map[string]bool
}

// NewFinalityView — собрать из цепочки финализированных чекпоинтов.
func NewFinalityView(chain []*wire.Checkpoint) *FinalityView {
	f := &FinalityView{finalized: make(map[string]bool)}
	for _, cp := range chain {
		for _, id := range cp.FinalizedTxIDs {
			f.finalized[string(id)] = true
		}
	}
	return f
}

// IsFinalized — txid входит в finalized_txid_list (глобальная финальность).
func (f *FinalityView) IsFinalized(txid []byte) bool {
	return f.finalized[string(txid)]
}

// MarkFinalized — дополнительная финальность (локальные победители
// голосования, ещё не зафиксированные чекпоинтом).
func (f *FinalityView) MarkFinalized(txid []byte) { f.finalized[string(txid)] = true }

// Len — число финализированных txid.
func (f *FinalityView) Len() int { return len(f.finalized) }

// Canonicalize — канонический порядок журнала для replay: для каждой
// группы конфликтных веток (один wallet+seq) победившая ветка (из
// finalized_txid_list) идёт раньше альтернатив; прочий порядок
// сохраняется (стабильно). Если победитель ещё не финализирован
// (эпоха не закрыта) — порядок прихода (FirstWins ядра).
func Canonicalize(txs []*wire.Transaction, view *FinalityView) []*wire.Transaction {
	if view == nil {
		return txs
	}
	pos := make(map[string][]int)
	for i, tx := range txs {
		k := ConflictKey(tx)
		pos[k] = append(pos[k], i)
	}
	used := make([]bool, len(txs))
	out := make([]*wire.Transaction, 0, len(txs))
	for i, tx := range txs {
		if used[i] {
			continue
		}
		idxs := pos[ConflictKey(tx)]
		if len(idxs) <= 1 {
			out = append(out, tx)
			used[i] = true
			continue
		}
		// группа конфликтных веток: победитель первым
		winner := i
		for _, j := range idxs {
			if !used[j] {
				id := txs[j].TxID()
				if view.IsFinalized(id[:]) {
					winner = j
					break
				}
			}
		}
		out = append(out, txs[winner])
		used[winner] = true
		for _, j := range idxs {
			if !used[j] {
				out = append(out, txs[j])
				used[j] = true
			}
		}
	}
	return out
}

// pinClock — часы replay: NetTime пинируется на timestamp_claim каждой
// транзакции (C9 ±30 мин проверяется на момент публикации, а не
// пересчёта); эпоха/период делегируются исходным часам узла.
type pinClock struct {
	inner ledger.EpochSource
	t     uint64
}

func (c *pinClock) NetTime() uint64 { return c.t }
func (c *pinClock) Epoch() uint64   { return c.inner.Epoch() }
func (c *pinClock) Period() uint64  { return c.inner.Period() }

// Replay — пересчёт состояния из genesis + журнала (INV-3): применяет
// транзакции в каноническом порядке с пинированием часов на
// timestamp_claim; конфликтная ветка (не победитель) фиксируется через
// Apply как Conflict и остаётся в журнале хранения.
func Replay(l *ledger.Ledger, txs []*wire.Transaction, view *FinalityView) error {
	orig := l.Clock
	pc := &pinClock{inner: orig}
	l.Clock = pc
	defer func() { l.Clock = orig }()
	for _, tx := range Canonicalize(txs, view) {
		pc.t = tx.TimestampClaim
		res := l.Apply(tx)
		switch res.Outcome {
		case ledger.Accepted:
		case ledger.Conflict:
			// альтернативная ветка — честно воспроизводит выбор (INV-1c)
		default:
			return fmt.Errorf("cons: replay: tx %x: %s", res.TxID, res.Code)
		}
	}
	return nil
}
