// Регресс-тесты №40 (блокер 8.1, этап 9): пинирование Epoch/Period в
// cons.Replay при пересчёте из БД — та же семантика, что у сетевой сверки
// (SyncJournalFrom/ImportJournal, net.ReplayClock). Критерии: рестарт узла
// из БД с vouch-ами разных эпох не даёт E_VOUCH_EPOCH; store-replay
// сходится с холодной синхронизацией (state_root A=B=C, демередж по
// историческим границам периодов).
package net

import (
	"context"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// txAt — транзакция с заданным timestamp_claim (исторические границы
// периодов/эпох в тестах сходимости); остальное как e.tx.
func (e *testEnv) txAt(n *Node, id string, typ wire.TxType, payload wire.Payload, hard bool, ts uint64) *wire.Transaction {
	e.t.Helper()
	// под applyMu.RLock: фоновый gossip применяет параллельно (этап 9)
	n.applyMu.RLock()
	w := n.Ledger.State.Wallet(e.addr(id))
	if w == nil {
		n.applyMu.RUnlock()
		e.t.Fatalf("кошелёк %s не зарегистрирован", id)
	}
	seq := w.Seq + 1
	prev := append([]byte(nil), w.TipTxID...)
	n.applyMu.RUnlock()
	tx := &wire.Transaction{
		NetworkID:       e.gp.Genesis.NetworkID,
		Wallet:          e.addr(id),
		Seq:             seq,
		Prev:            prev,
		Type:            typ,
		Payload:         payload,
		ProtocolVersion: e.gp.Genesis.ProtocolVersion,
		TimestampClaim:  ts,
	}
	e.solveAndSign(tx, id, hard)
	return tx
}

// mustRoot — state_root узла (сравнение путей пересчёта).
func mustRoot(t *testing.T, n *Node) []byte {
	t.Helper()
	r, err := n.StateRoot()
	if err != nil {
		t.Fatalf("state_root: %v", err)
	}
	return r
}

// TestStoreReplayRestartMultiEpochVouch — рестарт узла из БД с vouch-ами
// разных эпох в журнале (№40, критерий 1; прогон 8.1-2: 96 отказов
// E_VOUCH_EPOCH, флот потерян). До фикса cons.Replay делегировал эпоху
// живым часам и NewNode падал; после — replay пинирует Epoch/Period на
// timestamp_claim каждого vouch-а и узел поднимается, состояние сходится
// с состоянием до рестарта.
//
// Сетап журнала — пинированным путём (ImportJournal, семантика сетевой
// сверки): живой apply на 2-с эпохах гоняется с границей эпохи между
// сборкой (timestamp_claim) и применением (эпоха apply) — RegisteredEpoch
// живого применения расходился бы с replay на ±1 у границы; оба пути
// теста пинируют часы на claim — сходимость детерминирована. Объект
// теста — рестарт из БД (cons.Replay), а не способ построения журнала.
func TestStoreReplayRestartMultiEpochVouch(t *testing.T) {
	// короткие эпохи (2 с): vouch-ы разных эпох в журнале за секунды;
	// возраст/периоды поручителей сняты (посевные 65 дн. < 90)
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.EpochSeconds = 2
		p.VoucherMinAgeDays = 0
		p.VoucherMinMintPeriods = 0
	})
	db := dbPath(t, "restart.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})

	// журнал: минт, два кандидата, vouch в эпохе E и (после сна) в E+1;
	// эпоха vouch-а выводится из его timestamp_claim — пинированный
	// replay видит ту же эпоху, что и применение
	apply := func(tx *wire.Transaction) {
		t.Helper()
		if n1.ImportJournal([]*wire.Transaction{tx}) != 1 {
			t.Fatalf("не применён tx типа %d (seq %d)", tx.Type, tx.Seq)
		}
	}
	apply(e.mintTx(n1, "V1"))
	apply(e.register(n1, "CAND1"))
	apply(e.register(n1, "CAND2"))
	ts1 := uint64(time.Now().Unix())
	apply(e.txAt(n1, "V1", wire.Vouch, &wire.VouchPayload{
		Candidate: e.addr("CAND1"), Epoch: (ts1 - e.genesisT) / 2,
	}, false, ts1))

	time.Sleep(3 * time.Second) // эпоха ушла (epoch_seconds = 2)
	ts2 := uint64(time.Now().Unix())
	v2 := e.txAt(n1, "V2", wire.Vouch, &wire.VouchPayload{
		Candidate: e.addr("CAND2"), Epoch: (ts2 - e.genesisT) / 2,
	}, false, ts2)
	v1Epoch := (ts1 - e.genesisT) / 2
	if v1Epoch == v2.Payload.(*wire.VouchPayload).Epoch {
		t.Fatal("vouch-ы в одной эпохе — тест не воспроизводит №40")
	}
	apply(v2)

	// рестарт из той же БД: NewNode → initLedger → cons.Replay (store-replay)
	n2 := e.newNode(nodeOpts{level: LevelFull, db: db})
	for _, id := range []string{"CAND1", "CAND2"} {
		cand := n2.Ledger.State.Wallet(e.addr(id))
		if cand == nil || len(cand.VouchesReceived) != 1 {
			t.Fatalf("рестарт из БД: vouch для %s не восстановлен (cand=%v)", id, cand)
		}
	}
	// состояние после рестарта сходится с состоянием до рестарта
	root1, root2 := mustRoot(t, n1), mustRoot(t, n2)
	if string(root1) != string(root2) {
		t.Fatalf("state_root после рестарта из БД разошёлся: %x != %x", root1, root2)
	}
}

// TestStoreReplayConvergesWithColdSync — store-replay сходится с холодной
// синхронизацией (№40, критерий 2): A (рестарт из БД, cons.Replay) = B
// (пинированное применение — семантика сетевой сверки) = C (холодная
// сверка SyncJournalFrom). Журнал пересекает границу периодов 1→2 с
// ненулевыми балансами: демередж обязан осесть по историческим границам
// (до фикса store-replay жал живым периодом — расхождение Δ из прогона 8).
func TestStoreReplayConvergesWithColdSync(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.DemurrageFloor = 0 // демередж виден на малых балансах теста
	})
	// genesis отодвинут до 125 суток: выпуск заявляется только с периода 1
	// (завершается на 60-е сут), а демередж требует границу периода ПОСЛЕ
	// появления ненулевых балансах — при 65 сут границы недостижимы
	e.genesisT -= 60 * 86400
	e.gp.Genesis.GenesisTime = e.genesisT
	day := uint64(86400)
	db := dbPath(t, "converge.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})

	// журнал (пинированное применение — как сверка): выпуск периода 1 и
	// перевод в периоде 2, затем выпуск периода 2 в периоде 4 — на его
	// apply оседает демередж периодов 3–4 с историческими балансами
	apply := func(tx *wire.Transaction) {
		t.Helper()
		if n1.ImportJournal([]*wire.Transaction{tx}) != 1 {
			t.Fatalf("не применён tx типа %d (seq %d)", tx.Type, tx.Seq)
		}
	}
	apply(e.txAt(n1, "V1", wire.Mint, &wire.MintPayload{Period: 1, Amount: 2000}, true, e.genesisT+61*day))
	apply(e.txAt(n1, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 500},
	}, false, e.genesisT+62*day))
	apply(e.txAt(n1, "V2", wire.Mint, &wire.MintPayload{Period: 1, Amount: 2000}, true, e.genesisT+62*day+3600))
	apply(e.txAt(n1, "V1", wire.Mint, &wire.MintPayload{Period: 2, Amount: 1000}, true, e.genesisT+121*day))

	// B — состояние n1 (пинированное применение, семантика сверки)
	rootB, burnB := mustRoot(t, n1), n1.Ledger.State.TotalBurned
	if burnB == 0 {
		t.Fatal("демередж не осел по историческим границам — тест не воспроизводит Δ")
	}

	// A — рестарт из той же БД (store-replay, cons.Replay)
	n2 := e.newNode(nodeOpts{level: LevelFull, db: db})
	rootA, burnA := mustRoot(t, n2), n2.Ledger.State.TotalBurned

	// C — холодная синхронизация (SyncJournalFrom)
	e.start(n1)
	n3 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n3)
	if _, err := n3.SyncFromPeer(context.Background(), n1.Host.ID()); err != nil {
		t.Fatalf("холодная сверка: %v", err)
	}
	rootC, burnC := mustRoot(t, n3), n3.Ledger.State.TotalBurned

	if string(rootA) != string(rootB) || string(rootB) != string(rootC) {
		t.Fatalf("state_root разошёлся: A=%x B=%x C=%x", rootA, rootB, rootC)
	}
	if burnA != burnB || burnB != burnC {
		t.Fatalf("демередж разошёлся: A=%d B=%d C=%d", burnA, burnB, burnC)
	}
}
