// Тест предохранителя сеттлемента (H1 аудита Sonnet-2, шаг 3): один Apply
// не уводит время реестра больше чем на MaxSettleShiftSeconds — claim в
// далёком будущем иначе стоил бы O(эпох × кошельков) под applyMu (анти-DoS).
// Отказ не двигает состояние: сеттлемент не начат, журнал и replay tx не видят.
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestSettleShiftFuse — граница предохранителя: ровно на пороге вердикт
// выносится по существу (предохранитель не срабатывает), за порогом —
// E_TIME_SHIFT и состояние не меняется.
func TestSettleShiftFuse(t *testing.T) {
	// на пороге: сеттлемент уходит на век, предохранитель молчит —
	// вердикт приходит от правил регистрации (аттестация), не от времени
	e := newE2E(t)
	e.now = e.ledger.State.NetTime + MaxSettleShiftSeconds
	if res := e.register("EDGE"); res.Code == "E_TIME_SHIFT" {
		t.Fatalf("предохранитель сработал на пороге: %s", res.Code)
	}

	// за порогом: отказ предохранителя, состояние не двигается
	e = newE2E(t)
	base := e.ledger.State.NetTime
	before := stateFingerprint(e.ledger)
	e.now = base + MaxSettleShiftSeconds + 1
	res := e.register("FAR")
	if res.Outcome != Rejected || res.Code != "E_TIME_SHIFT" {
		t.Fatalf("за порогом: %v/%s, ожидается reject/E_TIME_SHIFT", res.Outcome, res.Code)
	}
	if e.ledger.State.NetTime != base {
		t.Fatalf("отказ сдвинул NetTime: %d, ожидалось %d", e.ledger.State.NetTime, base)
	}
	if got := stateFingerprint(e.ledger); got != before {
		t.Fatalf("отказ изменил состояние:\n было %s\n стало %s", before, got)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("инварианты после отказа: %v", bad)
	}
	// журнал пуст: отклонённая tx не попала ни в принятые, ни в конфликты
	if n := len(e.ledger.AcceptedTxs()); n != 0 {
		t.Fatalf("журнал после отказа: %d транзакций", n)
	}
}

// TestSettleShiftFuseDeterministic — вердикт предохранителя одинаков на двух
// независимых ядрах (live = replay по построению: State.NetTime и txNow —
// свойства журнала, а не часы узла).
func TestSettleShiftFuseDeterministic(t *testing.T) {
	verdict := func() (Outcome, string) {
		e := newE2E(t)
		e.now = e.ledger.State.NetTime + MaxSettleShiftSeconds + 86400
		res := e.register("FAR")
		return res.Outcome, res.Code
	}
	a1, c1 := verdict()
	a2, c2 := verdict()
	if a1 != a2 || c1 != c2 {
		t.Fatalf("вердикт зависит от окружения: %v/%s vs %v/%s", a1, c1, a2, c2)
	}
	if c1 != "E_TIME_SHIFT" {
		t.Fatalf("ожидался E_TIME_SHIFT, получен %s", c1)
	}
}

// TestSettleShiftFuseAllowsLongShift — предохранитель не мешает долгому
// честному сеттлементу: сдвиг на два периода оседает целиком (границы
// периодов проходят), NetTime доходит до txNow.
func TestSettleShiftFuseAllowsLongShift(t *testing.T) {
	e := newE2E(t)
	e.seedVouchers()
	e.requireAccepted(e.register("ALICE"), "register")
	e.activate("ALICE")
	e.now += 60 * 86400
	e.requireAccepted(e.ledger.Apply(e.tx("ALICE", wire.Mint,
		&wire.MintPayload{Period: 1, Amount: 1000}, true)), "mint")
	if got := e.ledger.State.NetTime; got != e.now {
		t.Fatalf("NetTime %d, ожидался txNow %d", got, e.now)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("инварианты: %v", bad)
	}
}
