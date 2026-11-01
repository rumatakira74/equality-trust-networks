// Settle отложенных операций после рестарта узла (v9.13 — этап фикса
// времени): сеттлемент — только от max(txNow) по журналу. Зрелый
// отложенный перевод применяется ближайшей ПРИНЯТОЙ tx с достаточным
// claim (на любом узле и в replay одинаково); доведения до часов узла
// нет — оно расходило бы live с replay. Незрелый перевод остаётся
// в ожидании независимо от локального времени.
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
		p.DelayedThresholdDefault = 500 // перевод 800 > P — отложенный
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
	apply(e.transferTx(n1, "V1", "R", 800))
	return e, n1, db, seq
}

// triggerTx — принятая tx-триггер сеттлемента: регистрация свежего
// кошелька (claim = текущее время, без побочных эффектов на проверяемые
// данные; сеттлемент оседает от её txNow).
func (e *testEnv) triggerTx(n *Node) *wire.Transaction {
	e.t.Helper()
	return e.register(n, "TRIG")
}

// TestSettleAfterRestartMaturedOnTrigger — рестарт узла после зрелости
// отложенного перевода (D = 0): без транзакций перевод остаётся в
// ожидании (сеттлемент — от журнала, не от часов); ближайшая принятая
// tx с достаточным claim применяет его; повторный рестарт из той же БД
// воспроизводит то же состояние ровно один раз (детерминизм).
func TestSettleAfterRestartMaturedOnTrigger(t *testing.T) {
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

	// рестарт из той же БД: replay применяет журнал, но триггера нет —
	// зрелый перевод НЕ исполняется от часов узла (v9.13)
	time.Sleep(200 * time.Millisecond) // D = 0: дедлайн давно прошёл
	n2 := e.newNode(nodeOpts{level: LevelFull, db: db})
	n2.applyMu.RLock()
	pt2 := n2.Ledger.State.PendingTransfers[ptKeyNet(e.addr("V1"), seq)]
	r2 := n2.Ledger.State.Wallet(e.addr("R"))
	n2.applyMu.RUnlock()
	if pt2 == nil || pt2.Executed {
		t.Fatal("перевод исполнен без tx-триггера — сеттлемент ушёл от журнала")
	}
	if r2 == nil || r2.Liquid != 0 {
		t.Fatalf("зрелый перевод зачислен без триггера: R=%v", r2)
	}

	// ближайшая принятая tx (claim строго позже max(txNow) журнала —
	// сеттлемент монотонно движится вперёд) применяет перевод
	time.Sleep(1100 * time.Millisecond)
	if n2.ImportJournal([]*wire.Transaction{e.triggerTx(n2)}) != 1 {
		t.Fatal("триггер не применён")
	}
	n2.applyMu.RLock()
	r2 = n2.Ledger.State.Wallet(e.addr("R"))
	v2 := n2.Ledger.State.Wallet(e.addr("V1"))
	pt2 = n2.Ledger.State.PendingTransfers[ptKeyNet(e.addr("V1"), seq)]
	n2.applyMu.RUnlock()
	if r2 == nil || r2.Liquid != 800 {
		t.Fatalf("зрелый перевод не применён от claim принятой tx: R=%v", r2)
	}
	if v2 == nil || v2.Locked != 0 {
		t.Fatalf("резерв отправителя не разобран: %v", v2)
	}
	if pt2 == nil || !pt2.Executed {
		t.Fatal("перевод не помечен исполненным")
	}

	// повторный рестарт из той же БД: детерминированный пересчёт
	// применяет перевод ровно один раз (триггер — в журнале)
	n3 := e.newNode(nodeOpts{level: LevelFull, db: db})
	n3.applyMu.RLock()
	got := n3.Ledger.State.Wallet(e.addr("R")).Liquid
	locked := n3.Ledger.State.Wallet(e.addr("V1")).Locked
	n3.applyMu.RUnlock()
	if got != 800 || locked != 0 {
		t.Fatalf("повторный рестарт: R=%d (ожидается 800), locked=%d", got, locked)
	}
}

// TestSettleAfterRestartImmature — незрелый перевод (D = 48 ч) после
// рестарта остаётся в ожидании даже при принятых tx: claim триггера
// раньше дедлайна; резерв заблокирован, исполнения нет.
func TestSettleAfterRestartImmature(t *testing.T) {
	e, _, db, seq := settleRestartEnv(t, 48)

	n2 := e.newNode(nodeOpts{level: LevelFull, db: db})
	// принятая tx-триггер с claim = сейчас (< дедлайна): исполнения нет
	if n2.ImportJournal([]*wire.Transaction{e.triggerTx(n2)}) != 1 {
		t.Fatal("триггер не применён")
	}
	n2.applyMu.RLock()
	r := n2.Ledger.State.Wallet(e.addr("R"))
	v := n2.Ledger.State.Wallet(e.addr("V1"))
	pt := n2.Ledger.State.PendingTransfers[ptKeyNet(e.addr("V1"), seq)]
	n2.applyMu.RUnlock()
	if pt == nil || pt.Executed {
		t.Fatal("незрелый перевод исполнен до дедлайна")
	}
	if r == nil || r.Liquid != 0 {
		t.Fatalf("незрелый перевод зачислен: R=%v", r)
	}
	if v == nil || v.Locked == 0 {
		t.Fatalf("резерв не заблокирован: %v", v)
	}
}
