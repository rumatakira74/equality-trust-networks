// v9.13 (этап фикса времени): сеттлемент — только от max(txNow) по
// журналу. Зрелые отложенные операции (окно 48 ч смены гардианов,
// отложенный перевод D) применяет ближайшая ПРИНЯТАЯ транзакция с
// достаточным claim — детерминированно на любом узле и в replay.
// Доведения сеттлемента до часов узла (SettleNow этапа 11) больше нет:
// оно расходило бы live-узлы с replay (находка этапа 12-testnet).
// Триггер в тестах — register свежего кошелька: принятая tx без
// побочных эффектов на проверяемые данные.
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestSettleMaturesOnNextAcceptedTx — зрелая смена гардианов вступает
// от claim ближайшей принятой tx (не от часов узла); незрелая — остаётся
// в ожидании; повторные tx не применяют смену повторно (идемпотентность).
func TestSettleMaturesOnNextAcceptedTx(t *testing.T) {
	u := newUG(t)
	e := u.e2eEnv
	t0 := e.now

	// публикация смены (окно 48 ч); больше «полезных» транзакций не будет
	res := u.ledger.Apply(u.propose(1, "V2", "G4"))
	u.requireAccepted(res, "смена опубликована")
	pu := u.ledger.State.PendingGuardianUpdates[0]

	// незрелая: claim за час до окна — смена не вступает
	e.now = t0 + 47*3600
	u.requireAccepted(e.register("TRIG1"), "триггер t0+47ч")
	w := u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 3 || pu.Effective {
		t.Fatal("незрелая смена вступила до окна")
	}

	// зрелая: claim после окна — смена вступает
	e.now = t0 + 49*3600
	u.requireAccepted(e.register("TRIG2"), "триггер t0+49ч")
	w = u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 2 ||
		!equalAddr(w.PersonalGuardians[0], u.addr("V2")) ||
		!equalAddr(w.PersonalGuardians[1], u.addr("G4")) {
		t.Fatal("зрелая смена не вступила от claim принятой tx")
	}
	if !pu.Effective || w.GuardianChanges != 1 {
		t.Fatalf("вступление: effective=%v, счётчик=%d", pu.Effective, w.GuardianChanges)
	}

	// повторное применение исключено: следующая tx ничего не меняет
	e.now += 3600
	u.requireAccepted(e.register("TRIG3"), "триггер повторный")
	w = u.ledger.State.Wallet(u.addr("W"))
	if w.GuardianChanges != 1 || !pu.Effective {
		t.Fatalf("повторная tx применила смену повторно: %d", w.GuardianChanges)
	}
	if got := u.ledger.State.Wallet(u.addr("G4")).Wards; got != 1 {
		t.Fatalf("wards G4: %d, ожидается 1", got)
	}
	if bad := u.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}

// TestSettlePendingTransferOnNextAcceptedTx — зрелый отложенный перевод
// исполняется от claim ближайшей принятой tx; незрелый — остаётся в
// ожидании; повторные tx не исполняют перевод повторно.
func TestSettlePendingTransferOnNextAcceptedTx(t *testing.T) {
	u := newUG(t)
	e := u.e2eEnv
	// короткое окно D, без моратория; порог P кошелька ниже суммы
	e.ledger.State.Net.Profile.TransferDelayHours = 2
	e.ledger.State.Net.Profile.TransferMoratoriumDays = 0
	w := e.ledger.State.Wallet(e.addr("W"))
	w.DelayedThreshold = 100
	w.Liquid += 1000
	e.ledger.State.TotalMinted += 1000 // INV-8: баланс входит в эмиссию
	t0 := e.now

	// отложенный перевод (сумма > P)
	seq := w.Seq + 1
	res := e.ledger.Apply(e.tx("W", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("G4"), Amount: 500},
	}, false))
	e.requireAccepted(res, "отложенный перевод")

	// незрелый: claim до дедлайна — в ожидании
	e.now = t0 + 1*3600
	u.requireAccepted(e.register("TRIG1"), "триггер до дедлайна")
	pt := e.ledger.State.PendingTransfers[ptKey(e.addr("W"), seq)]
	if pt == nil || pt.Executed {
		t.Fatal("незрелый перевод исполнился до дедлайна")
	}

	// зрелый: claim после дедлайна — исполнение
	e.now = t0 + 3*3600
	u.requireAccepted(e.register("TRIG2"), "триггер после дедлайна")
	if !pt.Executed {
		t.Fatal("зрелый перевод не исполнен от claim принятой tx")
	}
	if got := e.ledger.State.Wallet(e.addr("G4")).Liquid; got != 500 {
		t.Fatalf("получатель: %d, ожидается 500", got)
	}
	if got := e.ledger.State.Wallet(e.addr("W")).Locked; got != 0 {
		t.Fatalf("резерв отправителя не разобран: %d", got)
	}

	// повторное применение исключено (идемпотентность сеттлемента)
	e.now += 3600
	u.requireAccepted(e.register("TRIG3"), "триггер повторный")
	if got := e.ledger.State.Wallet(e.addr("G4")).Liquid; got != 500 {
		t.Fatalf("повторная tx исполнила перевод повторно: %d", got)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}
