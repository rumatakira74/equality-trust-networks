// Регресс-тест фикса времени (v9.13, этап 15), C9 — слой допуска сети:
// tx с claim вне ±timestamp_tolerance_seconds часов узла не принимается
// и не ретранслируется (cheapCheck/handleTxFromPeer/SubmitTx); ядро и
// replay C9 не применяют — та же tx валидируется ядром без допуска
// (вердикт детерминирован от txNow). Sandbox — допуск управляем:
// проверка не применяется (тестовые сети с короткими period_seconds).
package net

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestC9AdmissionLayer — допуск C9 в net при приёме/распространении.
func TestC9AdmissionLayer(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	now := uint64(time.Now().Unix())

	// claim двумя часами раньше часов узла — вне допуска ±30 мин
	stale := e.txAt(n1, "V1", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true, now-2*3600)
	// claim сейчас — в допуске
	fresh := e.txAt(n1, "V2", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true, now)

	// приём/ретрансляция: cheapCheck (валидатор gossip) отклоняет stale
	if err := n1.cheapCheck(stale); err == nil {
		t.Fatal("cheapCheck принял tx с claim вне допуска C9")
	}
	if err := n1.cheapCheck(fresh); err != nil {
		t.Fatalf("cheapCheck отклонил tx в допуске: %v", err)
	}

	// приём от пира: stale не применяется (журнал не растёт), fresh — да
	from := peer.ID("c9-test-peer")
	n1.handleTxFromPeer(stale, from)
	if got := len(n1.journalSnapshot()); got != 0 {
		t.Fatalf("tx вне допуска C9 применена: журнал %d", got)
	}
	n1.handleTxFromPeer(fresh, from)
	if got := len(n1.journalSnapshot()); got != 1 {
		t.Fatalf("tx в допуске не применена: журнал %d", got)
	}

	// локальная публикация (SubmitTx) stale claim — отказ с понятной
	// диагностикой (E_TIME от слоя допуска сети)
	n2 := e.newNode(nodeOpts{level: LevelFull})
	stale2 := e.txAt(n2, "V1", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true, now-2*3600)
	res, err := n2.SubmitTx(t.Context(), stale2)
	if err == nil || res.Code != "E_TIME" {
		t.Fatalf("SubmitTx вне допуска: err=%v code=%s, ожидается E_TIME", err, res.Code)
	}

	// ядро и replay БЕЗ C9: та же stale tx валидна по существу —
	// применяется ядром (ImportJournal, путь восстановления/сверки)
	n3 := e.newNode(nodeOpts{level: LevelFull})
	if applied := n3.ImportJournal([]*wire.Transaction{stale2}); applied != 1 {
		t.Fatalf("ядро отклонило tx по C9 (допуск — не дело ядра): применено %d", applied)
	}
}

// TestC9SandboxManaged — sandbox/тестовые сети: допуск управляем,
// проверка C9 не применяется (короткие period_seconds и предсборка tx
// прогонов не ломаются).
func TestC9SandboxManaged(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.Sandbox = true
		p.PeriodSeconds = 120 // короткие периоды (поле 61)
	})
	n1 := e.newNode(nodeOpts{level: LevelFull})
	now := uint64(time.Now().Unix())
	// предсобранная tx (claim на час раньше) — принимается: sandbox
	stale := e.txAt(n1, "V1", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true, now-3600)
	if err := n1.cheapCheck(stale); err != nil {
		t.Fatalf("sandbox: C9 сломал прогон с короткими period_seconds: %v", err)
	}
	n1.handleTxFromPeer(stale, peer.ID("c9-sandbox-peer"))
	if got := len(n1.journalSnapshot()); got != 1 {
		t.Fatalf("sandbox: tx вне допуска не применена: журнал %d", got)
	}
}
