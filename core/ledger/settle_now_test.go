// PENDING №2 (отчёт 11.2): SettleNow — публичная точка доведения
// сеттлемента до текущего времени без транзакции-триггера. После
// рестарта replay пинирует часы на timestamp_claim, и зрелые отложенные
// операции не применяются до ближайшей живой транзакции; здесь —
// применение смены гардианов (окно 48 ч) только по часам.
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestSettleNowGuardianUpdateWithoutTrigger — зрелая смена гардианов
// вступает по SettleNow без новой транзакции; незрелая — остаётся в
// ожидании; повторный SettleNow не применяет смену повторно.
func TestSettleNowGuardianUpdateWithoutTrigger(t *testing.T) {
	u := newUG(t)
	t0 := u.clock.NetTime()

	// публикация смены (окно 48 ч); больше транзакций не будет
	res := u.ledger.Apply(u.propose(1, "V2", "G4"))
	u.requireAccepted(res, "смена опубликована")
	pu := u.ledger.State.PendingGuardianUpdates[0]

	// незрелая: за час до окна — остаётся в ожидании
	u.clock.T = t0 + 47*3600
	u.ledger.SettleNow()
	w := u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 3 || pu.Effective {
		t.Fatal("незрелая смена вступила до окна")
	}

	// зрелая: без транзакции-триггера — только SettleNow
	u.clock.T = t0 + 49*3600
	u.ledger.SettleNow()
	w = u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 2 ||
		!equalAddr(w.PersonalGuardians[0], u.addr("V2")) ||
		!equalAddr(w.PersonalGuardians[1], u.addr("G4")) {
		t.Fatal("зрелая смена не вступила по SettleNow")
	}
	if !pu.Effective || w.GuardianChanges != 1 {
		t.Fatalf("вступление: effective=%v, счётчик=%d", pu.Effective, w.GuardianChanges)
	}

	// повторное применение исключено: повторный SettleNow ничего не меняет
	u.ledger.SettleNow()
	w = u.ledger.State.Wallet(u.addr("W"))
	if w.GuardianChanges != 1 || !pu.Effective {
		t.Fatalf("повторный SettleNow применил смену повторно: %d", w.GuardianChanges)
	}
	if got := u.ledger.State.Wallet(u.addr("G4")).Wards; got != 1 {
		t.Fatalf("wards G4: %d, ожидается 1", got)
	}
	if bad := u.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}

// TestSettleNowPendingTransferWithoutTrigger — зрелый отложенный
// перевод исполняется по SettleNow без новой транзакции; незрелый — нет.
func TestSettleNowPendingTransferWithoutTrigger(t *testing.T) {
	u := newUG(t)
	e := u.e2eEnv
	// короткое окно D, без моратория; порог P кошелька ниже суммы
	e.ledger.State.Net.Profile.TransferDelayHours = 2
	e.ledger.State.Net.Profile.TransferMoratoriumDays = 0
	w := e.ledger.State.Wallet(e.addr("W"))
	w.DelayedThreshold = 100
	w.Liquid += 1000
	e.ledger.State.TotalMinted += 1000 // INV-8: баланс входит в эмиссию
	t0 := e.clock.NetTime()

	// отложенный перевод (сумма > P)
	seq := w.Seq + 1
	res := e.ledger.Apply(e.tx("W", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("G4"), Amount: 500},
	}, false))
	e.requireAccepted(res, "отложенный перевод")

	// незрелый: до дедлайна остаётся в ожидании
	e.clock.T = t0 + 1*3600
	e.ledger.SettleNow()
	pt := e.ledger.State.PendingTransfers[ptKey(e.addr("W"), seq)]
	if pt == nil || pt.Executed {
		t.Fatal("незрелый перевод исполнился до дедлайна")
	}

	// зрелый: исполнение без транзакции-триггера
	e.clock.T = t0 + 3*3600
	e.ledger.SettleNow()
	if !pt.Executed {
		t.Fatal("зрелый перевод не исполнен по SettleNow")
	}
	if got := e.ledger.State.Wallet(e.addr("G4")).Liquid; got != 500 {
		t.Fatalf("получатель: %d, ожидается 500", got)
	}
	if got := e.ledger.State.Wallet(e.addr("W")).Locked; got != 0 {
		t.Fatalf("резерв отправителя не разобран: %d", got)
	}

	// повторное применение исключено
	e.ledger.SettleNow()
	if got := e.ledger.State.Wallet(e.addr("G4")).Liquid; got != 500 {
		t.Fatalf("повторный SettleNow исполнил перевод повторно: %d", got)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}
