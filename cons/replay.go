// Финальность и replay (INV-3 с правкой этапа 5, spec/04): журнал хранения
// содержит обе конфликтные ветки; выбор финализации восстанавливается из
// finalized_txid_list чекпоинтов — победившая ветка применяется первой,
// альтернатива фиксируется как конфликт (и материал evidence, 7.5).
package cons

import (
	"fmt"

	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
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

// pinClock — часы replay: NetTime, Epoch и Period пинируются на
// timestamp_claim каждой транзакции (№40 — блокер 8.1, этап 9): эпоха и
// период вычисляются от пинированного времени арифметически от genesis —
// та же семантика, что у сетевой сверки (net.ReplayClock,
// SyncJournalFrom/ImportJournal). C9 ±30 мин проверяется на момент
// публикации, а не пересчёта; vouch-ы прошлых эпох применяются (V8
// сверяется с эпохой публикации — рестарт из БД больше не даёт
// E_VOUCH_EPOCH), демередж оседает по историческим границам периодов
// (расхождение store-replay/синхронизация Δ125 закрыто).
type pinClock struct {
	genesis    uint64
	epochSecs  uint64
	periodSecs uint64
	t          uint64
}

// newPinClock — параметры genesis берутся из состояния реестра; конвенция
// периода — 30 суток от genesis (headless-ядро: net.RealClock,
// mintChallengeSlot).
func newPinClock(l *ledger.Ledger) *pinClock {
	epochSecs := l.State.Net.EpochSeconds
	if epochSecs == 0 {
		epochSecs = 24 * 3600
	}
	return &pinClock{
		genesis:    l.State.GenesisTime,
		epochSecs:  epochSecs,
		periodSecs: 30 * 86400,
	}
}

// Pin фиксирует время транзакции (её timestamp_claim).
func (c *pinClock) Pin(t uint64) { c.t = t }

func (c *pinClock) NetTime() uint64 { return c.t }

func (c *pinClock) Epoch() uint64 {
	if c.t <= c.genesis {
		return 0
	}
	return (c.t - c.genesis) / c.epochSecs
}

func (c *pinClock) Period() uint64 {
	if c.t <= c.genesis {
		return 0
	}
	return (c.t - c.genesis) / c.periodSecs
}

var _ ledger.EpochSource = (*pinClock)(nil)

// Replay — пересчёт состояния из genesis + журнала (INV-3): применяет
// транзакции в каноническом порядке с пинированием часов (NetTime, Epoch,
// Period) на timestamp_claim; конфликтная ветка (не победитель)
// фиксируется через Apply как Conflict и остаётся в журнале хранения.
func Replay(l *ledger.Ledger, txs []*wire.Transaction, view *FinalityView) error {
	orig := l.Clock
	pc := newPinClock(l)
	l.Clock = pc
	defer func() { l.Clock = orig }()
	for _, tx := range Canonicalize(txs, view) {
		pc.Pin(tx.TimestampClaim)
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
