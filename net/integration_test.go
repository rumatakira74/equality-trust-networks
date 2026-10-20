// Интеграционные тесты этапа 4 (4.11): N headless-узлов in-process;
// синхронизация реестра с нуля; узел офлайн -> подключается и догоняет;
// конфликтные ветки доставлены обе; invite_only; изоляция сетей;
// relay-сценарий без публичных адресов (эмуляция CGNAT); лёгкий узел.
package net

import (
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestNodesSyncFromScratch — 3 узла: A публикует, B и C подключаются и
// сходятся к одному реестру (критерий готовности этапа 4).
func TestNodesSyncFromScratch(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull, db: dbPath(t, "a.db")})
	e.start(a)

	// A: mint посевного кошелька + перевод (2 транзакции)
	ctx := context.Background()
	mint := e.mintTx(a, "V1")
	if res, err := a.SubmitTx(ctx, mint); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s (%v)", res.Outcome, res.Code, err)
	}
	tr := e.transferTx(a, "V1", "V2", 100)
	if res, err := a.SubmitTx(ctx, tr); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("transfer: %v/%s (%v)", res.Outcome, res.Code, err)
	}

	b := e.newNode(nodeOpts{level: LevelFull, db: dbPath(t, "b.db"), manual: manualAddrs(t, a)})
	c := e.newNode(nodeOpts{level: LevelFull, db: dbPath(t, "c.db"), manual: manualAddrs(t, a)})
	e.start(b)
	e.start(c)

	if !waitJournal(t, b, 2, 30*time.Second) {
		t.Fatalf("B не догнал журнал: %d", b.Stats().JournalLen)
	}
	if !waitJournal(t, c, 2, 30*time.Second) {
		t.Fatalf("C не догнал журнал: %d", c.Stats().JournalLen)
	}
	// сверка корней состояния (INV-3: пересчёт каждым узлом)
	ra, _ := a.StateRoot()
	rb, _ := b.StateRoot()
	rc, _ := c.StateRoot()
	if string(ra) != string(rb) || string(rb) != string(rc) {
		t.Fatal("корни состояния расходятся")
	}
	// баланс получателя сошёлся
	if w := b.Ledger.State.Wallet(e.addr("V2")); w == nil || w.TotalBalance() != 100 {
		t.Fatalf("баланс V2 на B: %+v", w)
	}
}

// TestOfflineNodeCatchesUp — узел офлайн, пока идут транзакции; подключается
// и догоняет (синхронизация из genesis + журнала, INV-3).
func TestOfflineNodeCatchesUp(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull, db: dbPath(t, "a.db")})
	e.start(a)
	ctx := context.Background()

	// пока B «офлайн», A публикует цепочку транзакций
	if res, err := a.SubmitTx(ctx, e.mintTx(a, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	if res, err := a.SubmitTx(ctx, e.transferTx(a, "V1", "V2", 100)); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("transfer1: %v/%s", res.Outcome, res.Code)
	}
	if res, err := a.SubmitTx(ctx, e.transferTx(a, "V1", "V3", 50)); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("transfer2: %v/%s", res.Outcome, res.Code)
	}

	// B подключается позже и догоняет
	b := e.newNode(nodeOpts{level: LevelFull, db: dbPath(t, "b.db"), manual: manualAddrs(t, a)})
	e.start(b)
	if !waitJournal(t, b, 3, 30*time.Second) {
		t.Fatalf("B не догнал: %d", b.Stats().JournalLen)
	}
	ra, _ := a.StateRoot()
	rb, _ := b.StateRoot()
	if string(ra) != string(rb) {
		t.Fatal("корни расходятся после догона")
	}
}

// TestConflictBranchesDelivered — эквивокация: обе ветки доставлены,
// никаких падений; выбор за финализацией (этап 5).
func TestConflictBranchesDelivered(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull, log: true})
	e.start(a)
	b := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a), log: true})
	e.start(b)
	ctx := context.Background()
	if !waitPeers(t, b, 1, 15*time.Second) {
		t.Fatal("B не подключился к A")
	}
	// дать gossip-подпискам обменяться (pubsub обменивается темами при
	// установлении соединения)
	time.Sleep(500 * time.Millisecond)

	if res, err := a.SubmitTx(ctx, e.mintTx(a, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	// эквивокация: одна seq, две разные ветки
	t1 := e.transferTx(a, "V1", "V2", 100)
	t2 := e.transferTx(a, "V1", "V3", 100)
	if res, err := a.SubmitTx(ctx, t1); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("t1: %v/%s", res.Outcome, res.Code)
	}
	res2, err := a.SubmitTx(ctx, t2)
	if err != nil {
		t.Fatalf("t2: %v", err)
	}
	if res2.Outcome.String() != "conflict" {
		t.Fatalf("t2: ожидался конфликт, получено %v/%s", res2.Outcome, res2.Code)
	}
	// обе ветки доставлены B: первая применена, вторая записана как конфликт
	if !waitJournal(t, b, 2, 30*time.Second) {
		t.Fatalf("B не получил ветки: %d", b.Stats().JournalLen)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b.ConflictsCount() >= 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if b.ConflictsCount() < 1 {
		t.Fatalf("конфликтная ветка не доставлена B: stats=%+v конфликтов=%d", b.Stats(), b.ConflictsCount())
	}
	// никаких падений: инварианты ядра целы
	if bad := b.Ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("инварианты B: %v", bad)
	}
}

// TestNetworkIsolation — пир чужой сети отсеивается сразу (4.6): identify
// показывает чужой network_id -> соединение закрывается.
func TestNetworkIsolation(t *testing.T) {
	e1 := newTestEnv(t)
	a := e1.newNode(nodeOpts{level: LevelFull})
	e1.start(a)

	// сеть 2: другой network_id
	e2 := newTestEnv(t, func(p *wire.ProfileParams) {})
	e2.gp.Genesis.NetworkID = []byte("tnp-testnet-0002")
	e2.gp.Manifest.NetworkID = e2.gp.Genesis.NetworkID
	// адреса пересчитываются от network_id
	e2.addrs = make(map[string][]byte)
	mh2 := e2.gp.Manifest.ManifestHash()
	e2.gp.Genesis.ManifestHash = mh2[:]

	b := e2.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a)})
	e2.start(b)

	// A обязан закрыть соединение с B (чужой UserAgent после identify)
	deadline := time.Now().Add(15 * time.Second)
	disconnected := false
	for time.Now().Before(deadline) {
		conns := a.Host.Network().ConnsToPeer(b.Host.ID())
		open := 0
		for _, c := range conns {
			if !c.IsClosed() {
				open++
			}
		}
		if open == 0 {
			disconnected = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !disconnected {
		t.Fatal("пир чужой сети не отключён")
	}
	// синхронизация невозможна: протоколы разных сетей не совпадают
	_, err := b.SyncFromPeer(context.Background(), a.Host.ID())
	if err == nil {
		t.Fatal("sync между сетями прошёл (изоляция нарушена)")
	}
}

// TestInviteOnlyNetwork — invite_only (4.6): первый узел один, второй
// подключается по приглашению (ManualAddrs = QR/ссылка), реестр сходится.
func TestInviteOnlyNetwork(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.NetworkAccess = wire.NetworkAccessInviteOnly
	})
	n1 := e.newNode(nodeOpts{level: LevelFull, inviteOnly: true})
	e.start(n1) // холодный старт: ждёт входящих

	ctx := context.Background()
	if res, err := n1.SubmitTx(ctx, e.mintTx(n1, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}

	n2 := e.newNode(nodeOpts{level: LevelFull, inviteOnly: true, manual: manualAddrs(t, n1)})
	e.start(n2)
	if !waitJournal(t, n2, 1, 30*time.Second) {
		t.Fatalf("n2 не синхронизировался по приглашению: %d", n2.Stats().JournalLen)
	}
	// каналы и зашитый список не используются: каскад даёт только manual/px
	n3 := e.newNode(nodeOpts{level: LevelFull, inviteOnly: true, manual: manualAddrs(t, n2)})
	e.start(n3)
	if !waitJournal(t, n3, 1, 30*time.Second) {
		t.Fatalf("n3 не синхронизировался: %d", n3.Stats().JournalLen)
	}
}

// TestInviteOnlySkipsChannels — в invite_only seed-списки и зашитый список
// не используются (12.3): каскад их пропускает.
func TestInviteOnlySkipsChannels(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.NetworkAccess = wire.NetworkAccessInviteOnly
	})
	// подписанный seed-список с валидной мультиподписью
	sl := &SeedList{
		NetworkID: e.gp.Genesis.NetworkID,
		Timestamp: uint64(time.Now().Unix()),
		Peers:     []peerAddrMsg{{ID: "fake", Addrs: []string{"/ip4/127.0.0.1/tcp/1"}}},
	}
	blob := SignSeedList(sl, []*ecdsa.PrivateKey{e.key("V1"), e.key("V2")})
	n := e.newNode(nodeOpts{
		level:      LevelFull,
		inviteOnly: true,
		manual:     []string{"/ip4/127.0.0.1/tcp/65531"},
	})
	n.Cfg.SeedList = blob
	n.Cfg.SeedPubKeys = map[string]*ecdsa.PublicKey{
		string(seedSignerID(e.key("V1"))): &e.key("V1").PublicKey,
		string(seedSignerID(e.key("V2"))): &e.key("V2").PublicKey,
	}
	n.Cfg.SeedThreshold = 2
	n.Cfg.EmbeddedAddrs = []string{"/ip4/127.0.0.1/tcp/65530"}
	candidates := n.discoverCascade(context.Background())
	for _, c := range candidates {
		if c.source == srcSeedList || c.source == srcEmbedded {
			t.Fatalf("invite_only использует запрещённый источник %s", c.source.name())
		}
	}
}

// TestRelayCGNAT — эмуляция CGNAT (4.7): узлы без публичных адресов
// (NoAnnounce) соединяются только через relay (circuit relay v2).
func TestRelayCGNAT(t *testing.T) {
	e := newTestEnv(t)
	r := e.newNode(nodeOpts{level: LevelFull, forcePublic: true, log: true}) // добровольный relay
	e.start(r)
	rAddrs := manualAddrs(t, r)

	a := e.newNode(nodeOpts{level: LevelFull, noAnnounce: true, relay: rAddrs, log: true})
	b := e.newNode(nodeOpts{level: LevelFull, noAnnounce: true, relay: rAddrs, log: true})
	e.start(a)
	e.start(b)

	// резервы слотов получены
	if a.relayReserved == 0 || b.relayReserved == 0 {
		t.Fatal("relay-резерв не получен")
	}
	// адрес B через relay (circuit): A <- relay <- B
	rAddrInfo, err := addrInfoOf(r)
	if err != nil {
		t.Fatal(err)
	}
	circuit, err := CircuitAddr(rAddrInfo, b.Host.ID())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Host.Connect(context.Background(), addrInfoOfMA(circuit)); err != nil {
		t.Fatalf("подключение через relay: %v", err)
	}
	ctx := context.Background()
	if res, err := a.SubmitTx(ctx, e.mintTx(a, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	if !waitJournal(t, b, 1, 30*time.Second) {
		t.Fatalf("B не получил транзакцию через relay: %d", b.Stats().JournalLen)
	}
}

// TestLightNode — лёгкий узел (12.5): своя цепочка, входящие, заголовки,
// Merkle-доказательство баланса (core/smt).
func TestLightNode(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull})
	e.start(a)
	ctx := context.Background()

	// V1 mint + перевод V2; V2 mint (своя цепочка V2)
	if res, err := a.SubmitTx(ctx, e.mintTx(a, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint V1: %v/%s", res.Outcome, res.Code)
	}
	if res, err := a.SubmitTx(ctx, e.transferTx(a, "V1", "V2", 100)); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("transfer: %v/%s", res.Outcome, res.Code)
	}
	if res, err := a.SubmitTx(ctx, e.mintTx(a, "V2")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint V2: %v/%s", res.Outcome, res.Code)
	}

	l := e.newNode(nodeOpts{level: LevelLight, ownWallet: e.addr("V2"), manual: manualAddrs(t, a)})
	e.start(l)
	if !waitPeers(t, l, 1, 15*time.Second) {
		t.Fatal("лёгкий узел не подключился")
	}
	if err := l.LightSync(ctx); err != nil {
		t.Fatalf("LightSync: %v", err)
	}
	// своя цепочка: mint V2
	if len(l.Light.OwnChain) != 1 {
		t.Fatalf("своя цепочка: %d транзакций, ожидается 1", len(l.Light.OwnChain))
	}
	// входящие: перевод от V1
	if len(l.Light.Incoming) != 1 {
		t.Fatalf("входящие: %d транзакций, ожидается 1", len(l.Light.Incoming))
	}
	// Merkle-доказательство баланса (12.5)
	proof, err := l.GetBalanceProof(ctx, a.Host.ID(), e.addr("V2"))
	if err != nil {
		t.Fatalf("proof: %v", err)
	}
	if !VerifyBalanceProof(e.addr("V2"), proof) {
		t.Fatal("Merkle-доказательство не сходится")
	}
	// баланс из листа доказательства (D7: StateLeaf, 9 полей)
	leaf, err := wire.DecodeStateLeaf(proof.LeafData)
	if err != nil || leaf.Balance != 2100 {
		t.Fatalf("баланс в листе: %+v (%v)", leaf, err)
	}
	if leaf.Seq != 1 {
		t.Fatalf("seq в листе: %d, ожидается 1", leaf.Seq)
	}
}

// TestJournalTxGossipDedup — txid журнала отмечается в дедупе gossip
// (блокер 8.1, авторизовано владельцем): после рестарта seen пуст, а
// журнал наполнен replay/синхронизацией — повторная доставка известной
// транзакции должна отсеиваться ДО cheapCheck (иначе каждая копия
// register/mint проходит Argon2id-верификацию ради отказа E_DUP).
func TestJournalTxGossipDedup(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	e.start(n1)

	// применение без публикации (как replay при старте из БД)
	tx := e.mintTx(n1, "V1")
	n1.applyLive(tx)

	// контракт фикса: txid журнала отмечен в дедупе gossip
	id := tx.TxID()
	n1.mu.RLock()
	marked := n1.seen[string(id[:])]
	n1.mu.RUnlock()
	if !marked {
		t.Fatal("txid журнала не отмечен в дедупе gossip")
	}

	// повторная доставка (эхо/рестарт): тихий пропуск до cheapCheck,
	// журнал не растёт и повторного apply не происходит
	n1.handleTxFromPeer(tx, n1.Host.ID())
	if got := len(n1.journalSnapshot()); got != 1 {
		t.Fatalf("журнал вырос при повторной доставке: %d", got)
	}
}
