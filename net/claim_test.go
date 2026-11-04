// Тесты claim-барьера применения (H1 аудита Sonnet-2): сверка, импорт
// журнала и повтор буфера не применяют tx с claim впереди потолка допуска
// применения — сеттлемент settleTo(max(txNow)) необратимо увёл бы NetTime
// вперёд (демередж/сгорание за «будущее»). Tx не теряется: откладывается и
// применяется, когда потолок догонит claim. Потолок — минимум из часов узла
// (+tol) и консенсусного времени последнего чекпоинта (NetworkTime + эпоха +
// maxShift + tol).
package net

import (
	"context"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestClaimCeiling — потолок допуска применения: до первого чекпоинта —
// часы узла + tol; с чекпоинтом — детерминированная граница от его
// сетевого времени, когда она строже локальной.
func TestClaimCeiling(t *testing.T) {
	e := newTestEnv(t)
	n := e.newNode(nodeOpts{level: LevelFull})
	tol := e.gp.Manifest.Profile.TimestampToleranceSeconds

	// без чекпоинта: только часы узла
	now := n.Clock.NetTime()
	ceil, ok := n.claimCeiling()
	if !ok {
		t.Fatal("барьер выключен вне sandbox")
	}
	if ceil < now+tol || ceil > now+tol+5 {
		t.Fatalf("потолок без чекпоинта: %d, ожидается часы узла %d + tol %d", ceil, now, tol)
	}

	// чекпоинт с сетевым временем в прошлом: консенсусная граница строже
	// локальной и не зависит от часов узла
	n.consClock = cons.NewClock(e.genesisT, 86400, 6*3600)
	stale := &wire.Checkpoint{Epoch: 1, NetworkTime: now - 10*86400}
	n.consClock.OnCheckpoint(stale)
	want := stale.NetworkTime + 86400 + 6*3600 + tol
	if ceil, ok = n.claimCeiling(); !ok || ceil != want {
		t.Fatalf("потолок от отставшего чекпоинта: %d (действует %v), ожидается %d", ceil, ok, want)
	}
	// tx с claim между консенсусной границей и часами узла — не admissible
	if n.claimAdmissible(&wire.Transaction{TimestampClaim: want + 1}) {
		t.Fatal("claim за консенсусной границей признан допустимым")
	}
	if !n.claimAdmissible(&wire.Transaction{TimestampClaim: want}) {
		t.Fatal("claim на консенсусной границе отвергнут")
	}

	// свежий чекпоинт: строже локальная граница
	fresh := &wire.Checkpoint{Epoch: 2, NetworkTime: n.Clock.NetTime()}
	n.consClock.OnCheckpoint(fresh)
	now = n.Clock.NetTime()
	if ceil, _ = n.claimCeiling(); ceil < now+tol || ceil > now+tol+5 {
		t.Fatalf("потолок от свежего чекпоинта: %d, ожидается часы узла %d + tol", ceil, now)
	}
}

// TestClaimCeilingSandboxManaged — sandbox/тестовые сети: барьер управляем,
// как допуск C9 (короткие period_seconds и предсобранные tx прогонов).
func TestClaimCeilingSandboxManaged(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) { p.Sandbox = true })
	n := e.newNode(nodeOpts{level: LevelFull})
	if _, ok := n.claimCeiling(); ok {
		t.Fatal("sandbox: барьер включён")
	}
	if !n.claimAdmissible(&wire.Transaction{TimestampClaim: 1 << 40}) {
		t.Fatal("sandbox: claim отвергнут барьером")
	}
}

// TestSyncFutureClaimDoesNotMoveNetTime — H1: сверка с «отравленным» пиром
// (в журнале tx с claim на 30 суток вперёд) не двигает NetTime получателя;
// tx отложена, а якорь сверки ушёл за ней — повторная сверка её не теряет
// и не дублирует.
func TestSyncFutureClaimDoesNotMoveNetTime(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	n1.applyLive(e.mintTx(n1, "V1"))
	now := uint64(time.Now().Unix())
	future := now + 30*86400
	// путь живого применения (без C9): так «отравленный» узел и получает
	// журнал с будущим claim — сверкой от такого же пира
	n1.applyLive(e.txAt(n1, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 100},
	}, false, future))
	n1.applyMu.RLock()
	poisoned := n1.Ledger.State.NetTime
	n1.applyMu.RUnlock()
	if poisoned != future {
		t.Fatalf("сетап: NetTime пира %d, ожидался claim %d", poisoned, future)
	}
	e.start(n1)

	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	if _, _, _, err := n2.SyncJournalFrom(context.Background(), n1.Host.ID(), 0, nil); err != nil {
		t.Fatalf("сверка: %v", err)
	}

	n2.applyMu.RLock()
	netTime := n2.Ledger.State.NetTime
	journal := len(n2.Ledger.AcceptedTxs())
	n2.applyMu.RUnlock()
	if journal != 1 {
		t.Fatalf("журнал получателя %d, ожидалось 1 (tx с будущим claim не применяется)", journal)
	}
	if netTime >= future {
		t.Fatalf("H1: NetTime получателя уведён будущим claim: %d (claim %d)", netTime, future)
	}
	if netTime > now+1800+5 {
		t.Fatalf("NetTime получателя %d вне допуска часов узла %d", netTime, now)
	}
	if got := n2.deferredLen(); got != 1 {
		t.Fatalf("отложено %d транзакций, ожидалась 1 (tx не должна теряться)", got)
	}

	// повторная сверка: якорь ушёл за отложенной tx — она не дублируется
	// и не теряется (пир её уже не пришлёт)
	if _, _, _, err := n2.SyncJournalFrom(context.Background(), n1.Host.ID(), 0, nil); err != nil {
		t.Fatalf("повторная сверка: %v", err)
	}
	if got := n2.deferredLen(); got != 1 {
		t.Fatalf("после повторной сверки отложено %d, ожидалась 1", got)
	}
	n2.applyMu.RLock()
	journal = len(n2.Ledger.AcceptedTxs())
	n2.applyMu.RUnlock()
	if journal != 1 {
		t.Fatalf("журнал после повторной сверки %d, ожидалось 1", journal)
	}
}

// TestDeferredClaimAppliedWhenTimeComes — отложенная tx не теряется: когда
// потолок допуска применения догоняет claim, она применяется и сеттлемент
// оседает на её txNow. tol = 1 с — потолок двигают часы узла.
func TestDeferredClaimAppliedWhenTimeComes(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) { p.TimestampToleranceSeconds = 1 })
	n1 := e.newNode(nodeOpts{level: LevelFull})
	n1.applyLive(e.mintTx(n1, "V1"))
	claim := uint64(time.Now().Unix()) + 5
	n1.applyLive(e.txAt(n1, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 100},
	}, false, claim))
	e.start(n1)

	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	if _, _, _, err := n2.SyncJournalFrom(context.Background(), n1.Host.ID(), 0, nil); err != nil {
		t.Fatalf("сверка: %v", err)
	}
	if got := n2.deferredLen(); got != 1 {
		t.Fatalf("отложено %d, ожидалась 1 (claim %d впереди потолка)", got, claim)
	}
	n2.applyMu.RLock()
	before := n2.Ledger.State.NetTime
	n2.applyMu.RUnlock()
	if before >= claim {
		t.Fatalf("сеттлемент сдвинут до claim: NetTime %d >= claim %d", before, claim)
	}

	// наступление claim: потолок догоняет — очередь доводится
	deadline := time.Now().Add(30 * time.Second)
	for n2.deferredLen() > 0 && time.Now().Before(deadline) {
		n2.retryDeferred()
		time.Sleep(200 * time.Millisecond)
	}
	if got := n2.deferredLen(); got != 0 {
		t.Fatalf("отложенная tx не применена: в очереди %d", got)
	}
	n2.applyMu.RLock()
	netTime := n2.Ledger.State.NetTime
	journal := len(n2.Ledger.AcceptedTxs())
	n2.applyMu.RUnlock()
	if journal != 2 {
		t.Fatalf("журнал %d, ожидалось 2 (отложенная tx применена)", journal)
	}
	if netTime != claim {
		t.Fatalf("NetTime %d, ожидался claim отложенной tx %d", netTime, claim)
	}
}

// TestImportJournalClaimBarrier — импорт журнала (восстановление из БД,
// источник наполнен путями gossip/сверки) применяет claim-барьер: tx с
// будущим claim откладывается, сеттлемент не сдвигается.
func TestImportJournalClaimBarrier(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	mint := e.mintTx(n1, "V1")
	n1.applyLive(mint)
	now := uint64(time.Now().Unix())
	future := now + 30*86400
	late := e.txAt(n1, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 100},
	}, false, future)

	n2 := e.newNode(nodeOpts{level: LevelFull})
	if applied := n2.ImportJournal([]*wire.Transaction{mint, late}); applied != 1 {
		t.Fatalf("импорт применил %d, ожидалось 1 (tx с будущим claim откладывается)", applied)
	}
	if got := n2.deferredLen(); got != 1 {
		t.Fatalf("отложено %d, ожидалась 1", got)
	}
	n2.applyMu.RLock()
	netTime := n2.Ledger.State.NetTime
	n2.applyMu.RUnlock()
	if netTime >= future {
		t.Fatalf("импорт увёл NetTime будущим claim: %d (claim %d)", netTime, future)
	}
}
