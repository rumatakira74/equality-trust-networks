// Тесты персистентности узла приложения (этап 16, №38): жёсткий рестарт
// из файловой БД — журнал/баланс/история восстанавливаются локально
// (cons.Replay strict), сверка с пирами после рестарта инкрементальна
// (якорь №41: первая сверка полная, далее только новые), повторное
// применение журнала не двигает демередж (монотонность от max(txNow),
// этап 15), E_DUP-шторм 8.1 не возвращается (дедуп txid до apply —
// seen заполняется из журнала при replay). Эталон сходимости путей
// пересчёта — тесты №40 (TestStoreReplayConvergesWithColdSync, этап 15).
package net

import (
	"context"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// spamOf — счётчик спама узла (под mu; поле пишет countSpam).
func spamOf(n *Node) int64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.spamCount
}

// seenHas — txid в дедуп-множестве узла (механизм 8.1: seen из журнала).
func seenHas(n *Node, tx *wire.Transaction) bool {
	id := tx.TxID()
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.seen[string(id[:])]
}

// TestRestartWithoutPeersThenConverge — жёсткий рестарт узла из БД без
// пиров (мандат этапа 16, тест 2): баланс, история и журнал — из
// локальной БД, state_root идентичен состоянию до рестарта; после
// появления пира сверка доходит до сети — корни совпадают.
func TestRestartWithoutPeersThenConverge(t *testing.T) {
	e := newTestEnv(t)
	db := dbPath(t, "restart-nopeers.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})
	n1.applyLive(e.mintTx(n1, "V1"))
	n1.applyLive(e.register(n1, "CAND"))
	n1.applyLive(e.transferTx(n1, "V1", "CAND", 500))
	rootBefore := mustRoot(t, n1)

	// «сеть»: пир с тем же журналом (полная сверка)
	e.start(n1)
	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	waitFor(t, 30*time.Second, "журнал пира после сверки", func() bool {
		return len(n2.journalSnapshot()) == 3
	})

	// жёсткий рестарт: узел из той же БД, пиров нет (manual пуст) —
	// весь I/O off-main в приложении, здесь — синхронный replay
	n1.Close()
	n3 := e.newNode(nodeOpts{level: LevelFull, db: db})
	n3.applyMu.RLock()
	journal := len(n3.Ledger.AcceptedTxs())
	var bal uint64
	if w := n3.Ledger.State.Wallet(e.addr("CAND")); w != nil {
		bal = w.TotalBalance() // как walletInfoJSON («balance» приложения)
	}
	n3.applyMu.RUnlock()
	if journal != 3 {
		t.Fatalf("журнал после рестарта без пиров: %d, хочу 3", journal)
	}
	if bal != 500 {
		t.Fatalf("баланс CAND после рестарта без пиров: %d, хочу 500", bal)
	}
	if r := mustRoot(t, n3); string(r) != string(rootBefore) {
		t.Fatalf("state_root после рестарта без пиров: %x, хочу %x",
			r, rootBefore)
	}

	// появление пира: рестарт с сетью — сверка доходит до состояния сети
	n4 := e.newNode(nodeOpts{level: LevelFull, db: db, manual: manualAddrs(t, n2)})
	e.start(n4)
	waitFor(t, 30*time.Second, "сверка после появления пира", func() bool {
		a := n4.anchorOf(n2.Host.ID().String())
		return a != nil && a.pos == 3
	})
	r2, r4 := mustRoot(t, n2), mustRoot(t, n4)
	if string(r2) != string(rootBefore) || string(r4) != string(rootBefore) {
		t.Fatalf("state_root разошёлся после появления пира: n2=%x n4=%x до=%x",
			r2, r4, rootBefore)
	}
}

// TestSettlementReplayIdempotent — идемпотентность сеттлемента после
// рестарта (мандат этапа 16, тест 4): повторное применение журнала из БД
// не двигает демередж — сеттлемент оседает от max(txNow) журнала
// (v9.13), повторные replay дают тот же state_root и тот же burn.
func TestSettlementReplayIdempotent(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.DemurrageFloor = 0 // демередж виден на малых балансах теста
	})
	// genesis отодвинут (как в TestStoreReplayConvergesWithColdSync):
	// границы периодов достижимы для оседания демереджа
	e.genesisT -= 60 * 86400
	e.gp.Genesis.GenesisTime = e.genesisT
	day := uint64(86400)
	db := dbPath(t, "settle-idem.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})
	apply := func(tx *wire.Transaction) {
		t.Helper()
		if n1.ImportJournal([]*wire.Transaction{tx}) != 1 {
			t.Fatalf("не применён tx типа %d (seq %d)", tx.Type, tx.Seq)
		}
	}
	apply(e.txAt(n1, "V1", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true, e.genesisT+61*day))
	apply(e.txAt(n1, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 500},
	}, false, e.genesisT+62*day))
	apply(e.txAt(n1, "V2", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true, e.genesisT+62*day+3600))
	apply(e.txAt(n1, "V1", wire.Mint, &wire.MintPayload{Period: 2, Amount: 1000}, true, e.genesisT+121*day))
	n1.applyMu.RLock()
	burn1, netTime1 := n1.Ledger.State.TotalBurned, n1.Ledger.State.NetTime
	n1.applyMu.RUnlock()
	root1 := mustRoot(t, n1)
	if burn1 == 0 {
		t.Fatal("демередж не осел — тест не воспроизводит Δ")
	}

	// рестарт №1 и рестарт №2 из той же БД: повторное применение журнала
	// воспроизводит состояние ровно один раз (монотонность от max(txNow))
	for i := 0; i < 2; i++ {
		n := e.newNode(nodeOpts{level: LevelFull, db: db})
		n.applyMu.RLock()
		burn, netTime := n.Ledger.State.TotalBurned, n.Ledger.State.NetTime
		n.applyMu.RUnlock()
		root := mustRoot(t, n)
		if burn != burn1 || netTime != netTime1 || string(root) != string(root1) {
			t.Fatalf("рестарт #%d: демередж/время/корень разошёлся: burn=%d (хочу %d), "+
				"netTime=%d (хочу %d), root=%x (хочу %x)",
				i+1, burn, burn1, netTime, netTime1, root, root1)
		}
	}
}

// TestSyncAnchorIncrementalAfterRestart — инкрементальный якорь после
// рестарта (мандат этапа 16, тест 5): якорь хранится в памяти — первая
// сверка после рестарта полная (corroboration всем журналом), далее —
// только новые транзакции: повторная сверка не пересылает весь журнал.
func TestSyncAnchorIncrementalAfterRestart(t *testing.T) {
	e := newTestEnv(t)
	db := dbPath(t, "restart-anchor.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})
	n1.applyLive(e.mintTx(n1, "V1"))
	n1.applyLive(e.register(n1, "CAND"))
	n1.applyLive(e.transferTx(n1, "V1", "CAND", 500))
	rootBefore := mustRoot(t, n1)

	// «сеть»: пир с тем же журналом
	e.start(n1)
	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	waitFor(t, 30*time.Second, "журнал пира после сверки", func() bool {
		return len(n2.journalSnapshot()) == 3
	})

	// жёсткий рестарт из БД с подключением к сети: первая сверка —
	// полный проход (якорь in-memory пуст), всё известно
	n1.Close()
	n3 := e.newNode(nodeOpts{level: LevelFull, db: db, manual: manualAddrs(t, n2)})
	e.start(n3)
	waitFor(t, 30*time.Second, "якорь первой сверки после рестарта", func() bool {
		a := n3.anchorOf(n2.Host.ID().String())
		return a != nil && a.pos == 3 && a.txid != nil
	})
	if r := mustRoot(t, n3); string(r) != string(rootBefore) {
		t.Fatalf("state_root после рестарта: %x, хочу %x", r, rootBefore)
	}

	// у пира одна новая транзакция: сверка-продолжение переносит ТОЛЬКО
	// её — якорь был на границе (pos=3), весь журнал не пересылается
	n2.applyLive(e.mintTx(n2, "V2"))
	k, err := n3.SyncFromPeer(context.Background(), n2.Host.ID())
	if err != nil {
		t.Fatalf("сверка-продолжение: %v", err)
	}
	if k != 1 {
		t.Fatalf("сверка-продолжение обработала %d, хочу 1 (только новую)", k)
	}
	a := n3.anchorOf(n2.Host.ID().String())
	if a == nil || a.pos != 4 {
		t.Fatalf("якорь после продолжения: %+v, хочу pos=4", a)
	}
	// повторная сверка без изменений — пустая
	if k, _ := n3.SyncFromPeer(context.Background(), n2.Host.ID()); k != 0 {
		t.Fatalf("сверка без изменений обработала %d, хочу 0", k)
	}
	if r3, r2 := mustRoot(t, n3), mustRoot(t, n2); string(r3) != string(r2) {
		t.Fatalf("state_root разошёлся: n3=%x n2=%x", r3, r2)
	}
}

// TestLightRestoreFromDB — лёгкий узел после рестарта (этап 16, №38):
// своя цепочка восстанавливается из локальной БД (restoreLight) —
// история своих операций доступна без пиров, LightSync доносит только
// новое. Входящие НЕ сохраняются (неполный журнал сломал бы
// cons.Replay strict при смене уровня на full, 12.2).
func TestLightRestoreFromDB(t *testing.T) {
	e := newTestEnv(t)
	// полный узел строит цепочку кошелька CAND (ledger для сборки tx)
	full := e.newNode(nodeOpts{level: LevelFull})
	reg := e.register(full, "CAND")
	full.applyLive(reg)
	mint := e.mintTx(full, "CAND")
	full.applyLive(mint)

	liteDB := dbPath(t, "light-restore.db")
	lite := e.newNode(nodeOpts{level: LevelLight, db: liteDB, ownWallet: e.addr("CAND")})
	lite.applyLive(reg)  // своя — сохраняется в БД (lightStore)
	lite.applyLive(mint) // своя — сохраняется в БД

	// жёсткий рестарт лёгкого узла: своя цепочка — из локальной БД
	lite2 := e.newNode(nodeOpts{level: LevelLight, db: liteDB, ownWallet: e.addr("CAND")})
	lite2.mu.RLock()
	chain := lite2.Light.OwnChain
	lite2.mu.RUnlock()
	if len(chain) != 2 {
		t.Fatalf("своя цепочка после рестарта: %d, хочу 2", len(chain))
	}
	if chain[0].Seq != 1 || chain[1].Seq != 2 {
		t.Fatalf("порядок цепочки после рестарта: seq %d, %d — хочу 1, 2",
			chain[0].Seq, chain[1].Seq)
	}
	// lightCheckOwn видит tip восстановленной цепочки (публикация seq 3
	// с prev = tip- txid восстановленной цепочки)
	tip := chain[1].TxID()
	if err := lite2.lightCheckOwn(&wire.Transaction{
		NetworkID: e.gp.Genesis.NetworkID, Wallet: e.addr("CAND"), Seq: 3,
		Prev: tip[:],
	}); err != nil {
		t.Fatalf("tip восстановленной цепочки не сходится: %v", err)
	}
}

// TestNoDupStormAfterRestart — E_DUP-шторм 8.1 не возвращается (мандат
// этапа 16, тест 6): после рестарта из БД дедуп-множество seen наполнено
// из журнала (indexTx), повторная доставка известных транзакций —
// gossip'ом и сверкой — не проходит повторную валидацию (Argon2id), не
// растит спам-счётчик и не меняет журнал/корень.
func TestNoDupStormAfterRestart(t *testing.T) {
	e := newTestEnv(t)
	db := dbPath(t, "restart-dupstorm.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})
	known := e.mintTx(n1, "V1") // усиленный PoW (Argon2id) — шторм был бы дорогим
	n1.applyLive(known)
	n1.applyLive(e.register(n1, "CAND"))
	rootBefore := mustRoot(t, n1)

	// «сеть»: пир с тем же журналом
	e.start(n1)
	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	waitFor(t, 30*time.Second, "журнал пира после сверки", func() bool {
		return len(n2.journalSnapshot()) == 2
	})

	// жёсткий рестарт из БД: seen наполнен из replay (механизм 8.1)
	n1.Close()
	n3 := e.newNode(nodeOpts{level: LevelFull, db: db, manual: manualAddrs(t, n2)})
	e.start(n3)
	waitFor(t, 30*time.Second, "первая сверка после рестарта", func() bool {
		a := n3.anchorOf(n2.Host.ID().String())
		return a != nil && a.pos == 2
	})
	if !seenHas(n3, known) {
		t.Fatal("txid журнала не в дедуп-множестве после рестарта (механизм 8.1 сломан)")
	}

	// повторная доставка известной транзакции gossip'ом (публикатор
	// без валидатора — реалистичная модель): seen отсеивает ДО
	// cheapCheck/apply — ни спама, ни изменения журнала/корня
	ep := newEvilPublisher(t, e, "")
	ai, err := addrInfoOf(n3)
	if err != nil {
		t.Fatal(err)
	}
	if err := ep.host.Connect(context.Background(), ai); err != nil {
		t.Fatalf("evil: connect: %v", err)
	}
	ep.publish(t, known)
	ep.publish(t, known)
	time.Sleep(2 * time.Second) // доставка gossip — асинхронная
	if spam := spamOf(n3); spam != 0 {
		t.Fatalf("спам-счётчик вырос от известной tx: %d", spam)
	}
	if len(n3.journalSnapshot()) != 2 {
		t.Fatalf("журнал изменился от повторной доставки: %d, хочу 2",
			len(n3.journalSnapshot()))
	}
	if r := mustRoot(t, n3); string(r) != string(rootBefore) {
		t.Fatalf("state_root изменился от повторной доставки: %x, хочу %x",
			r, rootBefore)
	}

	// повторная сверка (весь журнал пира = известные tx): E_DUP
	// отсеивается первой проверкой apply (C3), состояние не меняется
	if k, _ := n3.SyncFromPeer(context.Background(), n2.Host.ID()); k != 0 {
		t.Fatalf("сверка без новых tx обработала %d, хочу 0", k)
	}
	if r := mustRoot(t, n3); string(r) != string(rootBefore) {
		t.Fatalf("state_root изменился от сверки-повтора: %x, хочу %x",
			r, rootBefore)
	}
}
