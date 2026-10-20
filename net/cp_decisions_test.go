// Тесты решений владельца по №42 (этап 9, ветка stage9-decisions):
// (A) очередь строителей — первые K участников состава, ротация через
// epoch/2: строитель без узла не держит эпоху; (D) пропуск застрявшей
// эпохи — чекпоинт эпохи N+1 строит следующий строитель (окно очереди),
// эпоха N закрывается пустой (гэп нумерации), стагнация ≤ 1 эпохи.
package net

import (
	"context"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestCpBuilderQueueRotation — регресс очереди строителей (№42 (A)):
// кошелёк-строитель без узла (V1 — первый участник посевного круга —
// офлайн) не держит эпоху: окно очереди уходит живому строителю, эпоха
// закрывается. У тестового env genesis 65 суток назад — окно очереди
// первого чекпоинта детерминированно стоит у part[K-1] (V3, онлайн);
// ротация окон от свежего due покрыта TestCpSkipProposal. До решения
// (A) фолббек «любой узел» срабатывал через 2×epoch_seconds окном
// мульти-строителей (фрагментация блокера 8.1); при откате очереди без
// фолббека и с офлайн-строителем окна тест падает таймаутом.
func TestCpBuilderQueueRotation(t *testing.T) {
	e := newTestEnv(t)
	// V1 (строитель окна 0) НЕ запускается — кошелёк-строитель без узла;
	// онлайн — V2 (окно 1) и V3 (окно 2, подписант)
	n2 := e.consNode("V2", nodeOpts{level: LevelFull})
	e.start(n2)
	n3 := e.consNode("V3", nodeOpts{level: LevelFull, manual: manualAddrs(t, n2)})
	e.start(n3)
	if !waitPeers(t, n3, 1, 15*time.Second) {
		t.Fatal("узлы не соединились")
	}
	// epoch = 2 с (окно ротации 1 с, очередь K=3): окно V1 простаивает,
	// окно V2 строит, подписи 2 из 3 (V2+V3) — эпоха 1 закрывается
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if n2.CheckpointCount() >= 1 && n3.CheckpointCount() >= 1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if n2.CheckpointCount() < 1 || n3.CheckpointCount() < 1 {
		t.Fatalf("стройка без строителя первого окна: чекпоинты не закрылись " +
			"(строитель без узла держит эпоху — регресс №42 (A))")
	}
	if cp := n2.Chain[len(n2.Chain)-1]; cp.Epoch != 1 {
		t.Fatalf("эпоха первого чекпоинта: %d", cp.Epoch)
	}
}

// TestCpSkipProposal — механизм пропуска (№42 (D)) без сети: узел с
// застрявшей эпохой в цепочке строит предложение следующей эпохи
// (prev+2, застрявшая — пустой гэп), пока очередь не исчерпана —
// строится обычная (prev+1).
func TestCpSkipProposal(t *testing.T) {
	e := newTestEnv(t)
	now := uint64(time.Now().Unix())

	// фаза A: очередь не исчерпана (elapsed = 2 окна: due = now-2) —
	// строится обычная эпоха 2; окно 2 очереди — V3 (наш кошелёк)
	n := e.consNode("V3", nodeOpts{level: LevelFull})
	n.consClock = consNewClock(e)
	n.mu.Lock()
	n.Chain = append(n.Chain, &wire.Checkpoint{Epoch: 1, NetworkTime: now - 4})
	n.mu.Unlock()
	n.cpTick(context.Background())
	if n.lastProposal == nil || n.lastProposal.Epoch != 2 {
		t.Fatalf("до исчерпания очереди строится эпоха %v, хочу 2",
			epochOf(n.lastProposal))
	}

	// фаза B: очередь исчерпана (elapsed >= K×epoch/2 = 3 с) — застрявшая
	// эпоха 2 пропускается, строится эпоха 3
	n2 := e.consNode("V3", nodeOpts{level: LevelFull})
	n2.consClock = consNewClock(e)
	n2.mu.Lock()
	n2.Chain = append(n2.Chain, &wire.Checkpoint{Epoch: 1, NetworkTime: now - 5})
	n2.mu.Unlock()
	n2.cpTick(context.Background())
	if n2.lastProposal == nil || n2.lastProposal.Epoch != 3 {
		t.Fatalf("после исчерпания очереди строится эпоха %v, хочу 3 (пропуск застрявшей 2)",
			epochOf(n2.lastProposal))
	}
}

// TestCpSkipSignerGate — подписанты принимают предложение пропускающей
// эпохи (last+2) только когда застрявшая эпоха исчерпала очередь по их
// часам: преждевременный пропуск подписант не подпишет — порог не
// соберётся (защита от злонамеренного преждевременного пропуска).
func TestCpSkipSignerGate(t *testing.T) {
	e := newTestEnv(t)
	now := uint64(time.Now().Unix())

	// предложение эпохи 3 поверх цепочки с эпохой 1
	cp := &wire.Checkpoint{
		Epoch: 3, NetworkTime: now,
		Participants: [][]byte{e.addr("V1"), e.addr("V2"), e.addr("V3")},
		Threshold:    2,
	}

	// очередь НЕ исчерпана — предложение пропускающей эпохи не принимается
	n := e.consNode("V3", nodeOpts{level: LevelFull})
	n.consClock = consNewClock(e)
	n.mu.Lock()
	n.Chain = append(n.Chain, &wire.Checkpoint{Epoch: 1, NetworkTime: now})
	n.mu.Unlock()
	n.onCpMessage(cp, wire.SignatureShare{}, false)
	if n.hasProposalForEpochLocked(3) {
		t.Fatal("преждевременный пропуск принят подписантом")
	}

	// очередь исчерпана — принимается (и подписывается участником)
	n2 := e.consNode("V3", nodeOpts{level: LevelFull})
	n2.consClock = consNewClock(e)
	n2.mu.Lock()
	n2.Chain = append(n2.Chain, &wire.Checkpoint{Epoch: 1, NetworkTime: now - 5})
	n2.mu.Unlock()
	n2.onCpMessage(cp, wire.SignatureShare{}, false)
	if !n2.hasProposalForEpochLocked(3) {
		t.Fatal("пропуск после исчерпания очереди не принят подписантом")
	}
}

// consNewClock — консенсусные часы для тестов без Start (создаётся в
// Start; механизм пропуска тестируется на не запущенном узле).
// epoch = 2 с (как consNode), max_shift = 0.
func consNewClock(e *testEnv) *cons.Clock {
	return cons.NewClock(e.gp.Genesis.GenesisTime, 2, 0)
}

// epochOf — номер эпохи предложения (0 — нет предложения).
func epochOf(cp *wire.Checkpoint) uint64 {
	if cp == nil {
		return 0
	}
	return cp.Epoch
}
