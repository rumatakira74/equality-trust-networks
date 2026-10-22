// PENDING №2 (отчёт 11.2): settle отложенных операций после рестарта
// узла. Replay из БД пинирует часы на timestamp_claim (cons.Replay), а
// flushPending срабатывал только при живом apply-триггере — зрелые
// отложенные переводы «зависали» до ближайшей транзакции. После фикса
// узел доводит сеттлемент до текущего времени на старте (Start) и
// периодически (settleLoop) — без внешнего триггера.
package net

import (
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// ptKeyNet — ключ отложенного перевода (та же схема, что ledger.ptKey).
func ptKeyNet(addr []byte, seq uint64) string {
	return string(addr) + "|" + string([]byte{
		byte(seq >> 56), byte(seq >> 48), byte(seq >> 40), byte(seq >> 32),
		byte(seq >> 24), byte(seq >> 16), byte(seq >> 8), byte(seq),
	})
}

// settleRestartEnv — окружение с отложенным переводом в журнале:
// посевной V1 минтит, R регистрируется, V1 -> R перевод выше порога P
// уходит в отложенные (окно D задаётся профилем теста).
func settleRestartEnv(t *testing.T, delayHours uint64) (*testEnv, *Node, string, uint64) {
	t.Helper()
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.TransferDelayHours = delayHours
		p.TransferMoratoriumDays = 0
		p.DelayedThresholdDefault = 1000 // перевод 1500 > P — отложенный
	})
	db := dbPath(t, "settle-restart.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})
	apply := func(tx *wire.Transaction) {
		t.Helper()
		if n1.ImportJournal([]*wire.Transaction{tx}) != 1 {
			t.Fatalf("не применён tx типа %d (seq %d)", tx.Type, tx.Seq)
		}
	}
	apply(e.mintTx(n1, "V1"))
	apply(e.register(n1, "R"))
	seq := uint64(0)
	n1.applyMu.RLock()
	if w := n1.Ledger.State.Wallet(e.addr("V1")); w != nil {
		seq = w.Seq + 1
	}
	n1.applyMu.RUnlock()
	apply(e.transferTx(n1, "V1", "R", 1500))
	return e, n1, db, seq
}

// waitExecuted — дождаться, пока узел после старта сам применит
// зрелый перевод (settle после первичной сверки, ~500 мс; без внешнего
// триггера).
func waitExecuted(t *testing.T, n *Node, key string, timeoutMs int) {
	t.Helper()
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		n.applyMu.RLock()
		pt := n.Ledger.State.PendingTransfers[key]
		done := pt != nil && pt.Executed
		n.applyMu.RUnlock()
		if done {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("settle не применил зрелый перевод за %d мс", timeoutMs)
}

// TestSettleAfterRestartMatured — рестарт узла после зрелости
// отложенного перевода (D = 0): перевод применяется после сверки без
// внешнего триггера; повторное применение исключено (повторный flush и
// повторный рестарт из той же БД баланс не меняют).
func TestSettleAfterRestartMatured(t *testing.T) {
	e, n1, db, seq := settleRestartEnv(t, 0)

	// до рестарта: перевод в ожидании, резерв заблокирован
	n1.applyMu.RLock()
	pt := n1.Ledger.State.PendingTransfers[ptKeyNet(e.addr("V1"), seq)]
	r := n1.Ledger.State.Wallet(e.addr("R"))
	n1.applyMu.RUnlock()
	if pt == nil || pt.Executed || pt.Canceled {
		t.Fatalf("перевод не в ожидании: %+v", pt)
	}
	if r == nil || r.Liquid != 0 {
		t.Fatalf("получатель до исполнения: %v", r)
	}

	// рестарт из той же БД: replay + settle после сверки — без триггера
	n2 := e.newNode(nodeOpts{level: LevelFull, db: db})
	e.start(n2)
	waitExecuted(t, n2, ptKeyNet(e.addr("V1"), seq), 5000)
	n2.applyMu.RLock()
	r2 := n2.Ledger.State.Wallet(e.addr("R"))
	v2 := n2.Ledger.State.Wallet(e.addr("V1"))
	pt2 := n2.Ledger.State.PendingTransfers[ptKeyNet(e.addr("V1"), seq)]
	n2.applyMu.RUnlock()
	if r2 == nil || r2.Liquid != 1500 {
		t.Fatalf("зрелый перевод не применён после рестарта: R=%v", r2)
	}
	if v2 == nil || v2.Locked != 0 {
		t.Fatalf("резерв отправителя не разобран: %v", v2)
	}
	if pt2 == nil || !pt2.Executed {
		t.Fatal("перевод не помечен исполненным")
	}

	// повторное применение исключено: повторный flush без эффекта
	n2.settleNow()
	n2.applyMu.RLock()
	got := n2.Ledger.State.Wallet(e.addr("R")).Liquid
	n2.applyMu.RUnlock()
	if got != 1500 {
		t.Fatalf("повторный flush применил перевод повторно: %d", got)
	}

	// повторный рестарт из той же БД: детерминированный пересчёт
	// применяет перевод ровно один раз
	n3 := e.newNode(nodeOpts{level: LevelFull, db: db})
	e.start(n3)
	waitExecuted(t, n3, ptKeyNet(e.addr("V1"), seq), 5000)
	n3.applyMu.RLock()
	got = n3.Ledger.State.Wallet(e.addr("R")).Liquid
	locked := n3.Ledger.State.Wallet(e.addr("V1")).Locked
	n3.applyMu.RUnlock()
	if got != 1500 || locked != 0 {
		t.Fatalf("повторный рестарт: R=%d (ожидается 1500), locked=%d", got, locked)
	}
}

// TestSettleAfterRestartImmature — незрелый перевод (D = 48 ч) после
// рестарта остаётся в ожидании: резерв заблокирован, исполнения нет.
func TestSettleAfterRestartImmature(t *testing.T) {
	e, _, db, seq := settleRestartEnv(t, 48)

	n2 := e.newNode(nodeOpts{level: LevelFull, db: db})
	e.start(n2)
	// первичная сверка узла прошла (~500 мс); дополнительный явный
	// settle — тот же путь, что периодический (синхронно, для
	// детерминизма; зрелый случай покрыт ожиданием выше)
	time.Sleep(700 * time.Millisecond)
	n2.settleNow()
	n2.applyMu.RLock()
	r := n2.Ledger.State.Wallet(e.addr("R"))
	v := n2.Ledger.State.Wallet(e.addr("V1"))
	pt := n2.Ledger.State.PendingTransfers[ptKeyNet(e.addr("V1"), seq)]
	n2.applyMu.RUnlock()
	if pt == nil || pt.Executed {
		t.Fatal("незрелый перевод исполнен после рестарта")
	}
	if r == nil || r.Liquid != 0 {
		t.Fatalf("незрелый перевод зачислен: R=%v", r)
	}
	if v == nil || v.Locked == 0 {
		t.Fatalf("резерв не заблокирован: %v", v)
	}
}
