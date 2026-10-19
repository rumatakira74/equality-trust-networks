// Интеграционные тесты консенсуса этапа 5 в сети: чекпоинты в переходном
// режиме (посевной круг N-из-M), голосование по конфликтам поверх реальных
// соединений, лёгкий узел проверяет Merkle-доказательство по чекпоинту.
package net

import (
	"context"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// consNodeOpts — узел-подписант (кошелёк из посевного круга) с короткими
// эпохами для тестов (№16: предложение 24 ч).
func (e *testEnv) consNode(id string, o nodeOpts) *Node {
	e.t.Helper()
	o.walletID = id
	o.consEpoch = 2
	return e.newNode(o)
}

// TestCheckpointTransitionalNetwork — переходный режим (9.1): первые
// чекпоинты подписывает посевной круг N-из-M (2 из 3); все узлы
// принимают финализированный чекпоинт.
func TestCheckpointTransitionalNetwork(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.consNode("V1", nodeOpts{level: LevelFull})
	e.start(n1)
	n2 := e.consNode("V2", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	n3 := e.consNode("V3", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n3)
	if !waitPeers(t, n2, 1, 15*time.Second) || !waitPeers(t, n3, 1, 15*time.Second) {
		t.Fatal("узлы не соединились")
	}
	// чекпоинт эпохи 1: сборка + подписи 2 из 3 + сбор по теме
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if n1.CheckpointCount() >= 1 && n2.CheckpointCount() >= 1 && n3.CheckpointCount() >= 1 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if n1.CheckpointCount() < 1 || n2.CheckpointCount() < 1 || n3.CheckpointCount() < 1 {
		t.Fatalf("чекпоинты не финализированы: %d/%d/%d",
			n1.CheckpointCount(), n2.CheckpointCount(), n3.CheckpointCount())
	}
	cp1 := n1.Chain[len(n1.Chain)-1]
	if cp1.Epoch != 1 {
		t.Fatalf("эпоха первого чекпоинта: %d", cp1.Epoch)
	}
	// переходный режим: подписанты — посевной круг (3), порог 2
	if len(cp1.Participants) != 3 || cp1.Threshold != 2 {
		t.Fatalf("переходный режим: M=%d N=%d, ожидается 3/2",
			len(cp1.Participants), cp1.Threshold)
	}
	if len(cp1.ThresholdSig.Signatures) < 2 {
		t.Fatalf("подписей: %d < 2", len(cp1.ThresholdSig.Signatures))
	}
	// заголовки (с корнем состояния) доступны лёгким узлам
	h, ok := n1.LatestHeader()
	if !ok || len(h.StateRoot) != 32 {
		t.Fatal("заголовок чекпоинта не выдан")
	}
}

// TestVoteFinalityNetwork — конфликт разрешается голосованием по сети:
// обе ветки доставлены, Snowball финализирует победителя (β раундов),
// узлы сходятся к одному состоянию.
func TestVoteFinalityNetwork(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.consNode("V1", nodeOpts{level: LevelFull})
	e.start(n1)
	n2 := e.consNode("V2", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	n3 := e.consNode("V3", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n3)
	if !waitPeers(t, n2, 1, 15*time.Second) || !waitPeers(t, n3, 1, 15*time.Second) {
		t.Fatal("узлы не соединились")
	}
	time.Sleep(500 * time.Millisecond)
	ctx := context.Background()

	// V1 выпускает, затем эквивокация: обе ветки доставлены всем
	if res, err := n1.SubmitTx(ctx, e.mintTx(n1, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	if !waitJournal(t, n2, 1, 15*time.Second) || !waitJournal(t, n3, 1, 15*time.Second) {
		t.Fatal("выпуск не доставлен")
	}
	t1 := e.transferTx(n1, "V1", "V2", 100)
	t2 := e.tx(n1, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V3"), Amount: 100},
	}, false)
	if res, err := n1.SubmitTx(ctx, t1); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("t1: %v/%s", res.Outcome, res.Code)
	}
	res2, err := n1.SubmitTx(ctx, t2)
	if err != nil || res2.Outcome.String() != "conflict" {
		t.Fatalf("t2: %v/%s (%v)", res2.Outcome, res2.Code, err)
	}
	// голосование: все узлы финализируют одну ветку
	deadline := time.Now().Add(30 * time.Second)
	var winner []byte
	for time.Now().Before(deadline) {
		var w []byte
		ok := true
		for _, n := range []*Node{n1, n2, n3} {
			st := n.Snowball.State(cons.ConflictKey(t1))
			if st == nil || !st.Final {
				ok = false
				break
			}
			if w == nil {
				w = st.Pref
			} else if string(w) != string(st.Pref) {
				ok = false
				break
			}
		}
		if ok {
			winner = w
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if winner == nil {
		t.Fatal("голосование не финализировало конфликт")
	}
	// состояние узлов сходится (все применяют победителя)
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		r1, _ := n1.StateRoot()
		r2, _ := n2.StateRoot()
		r3, _ := n3.StateRoot()
		if string(r1) == string(r2) && string(r2) == string(r3) {
			return // сошлись
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("состояния узлов не сошлись после финализации")
}

// TestVoteResponseSignature — долг этапа 5: ответ голосования подписан
// ключом кошелька отвечающего (wallet + подпись над key|pref|weight);
// опрашивающий проверяет подпись. Подписанный голос принимается,
// неподписанный (узел без кошелька) — нет.
func TestVoteResponseSignature(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.consNode("V1", nodeOpts{level: LevelFull})
	e.start(n1)
	n2 := e.consNode("V2", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	// узел без кошелька: голос без подписи не должен учитываться
	n3 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n3)
	if !waitPeers(t, n2, 1, 15*time.Second) || !waitPeers(t, n3, 1, 15*time.Second) {
		t.Fatal("узлы не соединились")
	}
	// конфликт в состоянии Snowball всех узлов
	key := string(e.addr("V1")) + "|\x00\x00\x00\x00\x00\x00\x00\x01"
	branchA, branchB := []byte("branch-AAAA"), []byte("branch-BBBB")
	for _, n := range []*Node{n1, n2, n3} {
		n.Snowball.Observe(key, branchA, branchB)
	}
	// подписанный голос V2 принимается
	pref, weight, ok := (&remotePeer{n: n1, id: n2.Host.ID()}).VotePreference(key)
	if !ok {
		t.Fatal("подписанный голос отклонён")
	}
	if string(pref) != string(branchA) || weight == 0 {
		t.Fatalf("голос: pref=%q weight=%d", pref, weight)
	}
	// голос узла без кошелька (без подписи) не учитывается
	if _, _, ok := (&remotePeer{n: n1, id: n3.Host.ID()}).VotePreference(key); ok {
		t.Fatal("неподписанный голос принят")
	}
	// подделка: подпись над другим конфликтом не проходит (домен хеша
	// включает key — перенос подписи невозможен)
	otherKey := string(e.addr("V1")) + "|\x00\x00\x00\x00\x00\x00\x00\x02"
	if pref, _, ok := (&remotePeer{n: n1, id: n2.Host.ID()}).VotePreference(otherKey); ok {
		t.Fatalf("ответ по чужому конфликту принят: pref=%q", pref)
	}
}

// TestLightNodeCheckpointProof — лёгкий узел получает заголовки
// (с корнем состояния) и проверяет Merkle-доказательство баланса по
// ФИНАЛИЗИРОВАННОМУ чекпоинту (12.5, критерий готовности этапа 5).
func TestLightNodeCheckpointProof(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.consNode("V1", nodeOpts{level: LevelFull})
	e.start(n1)
	n2 := e.consNode("V2", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	ctx := context.Background()
	if res, err := n1.SubmitTx(ctx, e.mintTx(n1, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	// дождаться чекпоинта (баланс V2 в корне состояния)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if n1.CheckpointCount() >= 1 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if n1.CheckpointCount() < 1 {
		t.Fatal("чекпоинт не финализирован")
	}
	// лёгкий узел: заголовки + доказательство по чекпоинту
	l := e.newNode(nodeOpts{level: LevelLight, ownWallet: e.addr("V2"), manual: manualAddrs(t, n1)})
	e.start(l)
	if !waitPeers(t, l, 1, 15*time.Second) {
		t.Fatal("лёгкий узел не подключился")
	}
	if err := l.LightSync(ctx); err != nil {
		t.Fatalf("LightSync: %v", err)
	}
	header, ok := l.LatestHeader()
	if !ok {
		t.Fatal("лёгкий узел не получил заголовок чекпоинта")
	}
	proof, err := l.GetBalanceProof(ctx, n1.Host.ID(), e.addr("V2"))
	if err != nil {
		t.Fatalf("proof: %v", err)
	}
	// доказательство против корня ФИНАЛИЗИРОВАННОГО чекпоинта
	if !l.VerifyProofAgainstCheckpoint(e.addr("V2"), proof, header) {
		t.Fatal("доказательство не сходится с корнем чекпоинта")
	}
	// доказательство против произвольного корня не проходит
	tampered := header
	tampered.StateRoot = append([]byte(nil), header.StateRoot...)
	tampered.StateRoot[0] ^= 0xff
	if l.VerifyProofAgainstCheckpoint(e.addr("V2"), proof, tampered) {
		t.Fatal("доказждение прошло против подменённого корня")
	}
}
