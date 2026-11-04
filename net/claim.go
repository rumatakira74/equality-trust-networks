// Claim-барьер применения (H1 аудита Sonnet-2): допуск C9 действует только
// на приёме по gossip и локальной публикации. Пути сверки (SyncJournalFrom),
// импорта журнала (ImportJournal) и повтора буфера внеочередных
// (retryPending) применяют tx прямо в Ledger.Apply, а сеттлемент
// settleTo(max(txNow)) необратимо уводит NetTime вперёд: демередж и
// сгорание исполняются за всё «будущее», плюс цикл эпох обходит все
// кошельки на каждую границу (CPU-DoS под applyMu). Заражение расходится
// между пирами через сверку.
//
// Барьер односторонний: claim не позже потолка допуска применения. Потолок —
// минимум из двух границ:
//
//  1. часы узла + timestamp_tolerance_seconds — немедленная граница
//     (честные tx из прошлого проходят всегда, будущее отсекается);
//  2. консенсусное время + tol: NetworkTime последнего финализированного
//     чекпоинта + эпоха + maxShift (cons.Clock.NetTime) — детерминированная
//     граница от данных консенсуса, а не от часов узла (структурный шаг).
//
// Tx с claim впереди потолка НЕ теряется и НЕ отклоняется: она откладывается
// в очередь deferred и применяется, когда потолок догонит claim. Иначе она
// пропала бы навсегда — позиционный якорь сверки (№41) уходит за
// обработанным, и пир больше её не пришлёт.
//
// Граница «по порядку журнала» (claim ≤ State.NetTime + Δ) намеренно не
// используется: цепочка дешёвых tx сдвигала бы время на Δ за tx. Протокол и
// консенсус не меняются — кламп стоит на применении, а не на валидности
// сети; ядро остаётся детерминированным от txNow (live = replay).
package net

import (
	"sort"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// maxDeferred — предел очереди отложенных по claim (спам не переполняет
// память). Вытесняется самая далёкая по claim запись: она придёт повторной
// сверкой, а ближние к потолку — самые ценные (применятся первыми).
const maxDeferred = 4096

// claimCeiling — потолок допуска применения: claim ≤ потолок. Второе
// значение — действует ли барьер (sandbox/тестовые сети и узел без часов —
// управляемый режим, как у допуска C9: короткие period_seconds и
// предсобранные tx прогонов не ломаются).
func (n *Node) claimCeiling() (uint64, bool) {
	if n.Cfg == nil || n.Cfg.Genesis == nil || n.Cfg.Genesis.Manifest == nil ||
		n.Cfg.Genesis.Manifest.Profile == nil {
		return 0, false
	}
	prof := n.Cfg.Genesis.Manifest.Profile
	if prof.Sandbox || n.Clock == nil {
		return 0, false
	}
	tol := prof.TimestampToleranceSeconds
	ceil := n.Clock.NetTime() + tol
	// Структурная граница (шаг 2): сетевое время последнего чекпоинта +
	// эпоха + предел сдвига. Детерминирована (данные консенсуса), не
	// зависит от часов узла; применяется, когда строже локальной.
	if n.consClock != nil {
		if cp := n.consClock.LastCheckpoint(); cp != nil {
			hi := cp.NetworkTime + n.ConsParams.EpochSeconds +
				n.ConsParams.MaxTimeShift + tol
			if hi < ceil {
				ceil = hi
			}
		}
	}
	return ceil, true
}

// claimAdmissible — claim транзакции в пределах потолка допуска применения.
func (n *Node) claimAdmissible(tx *wire.Transaction) bool {
	ceil, ok := n.claimCeiling()
	if !ok {
		return true
	}
	return tx.TimestampClaim <= ceil
}

// deferTx — отложить транзакцию до наступления её claim (потолок догонит
// claim позже: часы узла идут, чекпоинты двигают сетевое время).
func (n *Node) deferTx(tx *wire.Transaction) {
	id := tx.TxID()
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.deferred == nil {
		n.deferred = make(map[string]*wire.Transaction)
	}
	if _, ok := n.deferred[string(id[:])]; ok {
		return
	}
	if len(n.deferred) >= maxDeferred {
		// вытесняем самую далёкую по claim (наименее вероятную к применению)
		var farKey string
		var farClaim uint64
		for k, t := range n.deferred {
			if farKey == "" || t.TimestampClaim > farClaim {
				farKey, farClaim = k, t.TimestampClaim
			}
		}
		delete(n.deferred, farKey)
	}
	n.deferred[string(id[:])] = tx
}

// retryDeferred — применение отложенных по claim транзакций, чей claim уже
// в пределах потолка. Потолок двигают часы узла (идут всегда) и чекпоинты
// (сетевое время), поэтому вызов привязан к тикам сверки и принятию
// чекпоинта. Порядок применения — по claim (ранние первыми): сеттлемент
// монотонен от max(txNow), порядок на итоговое состояние не влияет, но
// ближние к границе применяются раньше.
//
// Вызывается только ВНЕ applyMu (applyLive берёт Lock).
func (n *Node) retryDeferred() int {
	ceil, ok := n.claimCeiling()
	if !ok {
		return 0
	}
	n.mu.Lock()
	ready := make([]*wire.Transaction, 0, len(n.deferred))
	for k, tx := range n.deferred {
		if tx.TimestampClaim <= ceil {
			ready = append(ready, tx)
			delete(n.deferred, k)
		}
	}
	n.mu.Unlock()
	if len(ready) == 0 {
		return 0
	}
	sort.Slice(ready, func(i, j int) bool {
		if ready[i].TimestampClaim != ready[j].TimestampClaim {
			return ready[i].TimestampClaim < ready[j].TimestampClaim
		}
		a, b := ready[i].TxID(), ready[j].TxID()
		return string(a[:]) < string(b[:])
	})
	for _, tx := range ready {
		n.applyLive(tx)
	}
	return len(ready)
}

// deferredLen — длина очереди отложенных (диагностика, тесты).
func (n *Node) deferredLen() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.deferred)
}
