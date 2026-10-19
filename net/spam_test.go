// Спам-тест (4.11.3): невалидные транзакции (подпись/PoW/размер) не
// ретранслируются и не переполняют очередь. Издатель-нарушитель —
// raw gossipsub без валидатора (реалистичная модель атакующего).
package net

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"

	"github.com/rumatakira74/trust-networks-platform/core/crypto"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// evilPublisher — узел-нарушитель: та же тема, но без валидатора
// (публикует всё, что хочет атакующий).
type evilPublisher struct {
	host  host.Host
	ps    *pubsub.PubSub
	topic *pubsub.Topic
}

func newEvilPublisher(t *testing.T, e *testEnv, dial string) *evilPublisher {
	t.Helper()
	// нарушитель подделывает UserAgent платформы (фильтр network_id его
	// пропускает — защита от спама обязана работать в валидаторе)
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/udp/0/quic-v1"),
		libp2p.UserAgent("tnp/"+hexEncode(e.gp.Genesis.NetworkID)),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Close() })
	ps, err := pubsub.NewGossipSub(context.Background(), h,
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithMessageIdFn(txMessageID),
	)
	if err != nil {
		t.Fatal(err)
	}
	topic, err := ps.Join("tx/" + hexEncode(e.gp.Genesis.NetworkID))
	if err != nil {
		t.Fatal(err)
	}
	ep := &evilPublisher{host: h, ps: ps, topic: topic}
	if dial != "" {
		if err := h.Connect(context.Background(), addrInfoOfMA(maNew(dial))); err != nil {
			t.Fatalf("evil: connect: %v", err)
		}
	}
	return ep
}

func (ep *evilPublisher) publish(t *testing.T, tx *wire.Transaction) {
	t.Helper()
	if err := ep.topic.Publish(context.Background(), tx.Encode()); err != nil {
		t.Fatalf("evil publish: %v", err)
	}
}

// TestSpamNotRelayed — невалидные транзакции не ретранслируются и не
// применяются; счётчик спама растёт; очередь не переполняется.
func TestSpamNotRelayed(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull})
	e.start(a)
	b := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a)})
	e.start(b)
	if !waitPeers(t, b, 1, 15*time.Second) {
		t.Fatal("B не подключился к A")
	}
	time.Sleep(500 * time.Millisecond)
	ctx := context.Background()

	// валидная транзакция для базового состояния
	good := e.mintTx(a, "V1")
	if res, err := a.SubmitTx(ctx, good); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	if !waitJournal(t, b, 1, 15*time.Second) {
		t.Fatal("валидная транзакция не дошла")
	}

	// нарушитель подключается к B и публикует невалидные транзакции
	ep := newEvilPublisher(t, e, manualAddrs(t, b)[0])
	time.Sleep(500 * time.Millisecond)

	// 1. плохая подпись
	badSig := e.transferTx(a, "V1", "V2", 10)
	badSig.Signature = []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}
	ep.publish(t, badSig)
	// 2. плохой PoW (валидная подпись, неверный nonce)
	badPow := e.transferTx(a, "V1", "V2", 10)
	badPow.Pow.Nonce = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	ep.publish(t, badPow)
	// 3. чужая сеть (network_id)
	foreign := e.transferTx(a, "V1", "V2", 10)
	foreign.NetworkID = []byte("tnp-foreign-net")
	ep.publish(t, foreign)

	// ждём возможной (запрещённой) доставки
	time.Sleep(3 * time.Second)
	if b.Stats().JournalLen != 1 {
		t.Fatalf("невалидные транзакции применены: журнал %d", b.Stats().JournalLen)
	}
	if b.ConflictsCount() != 0 {
		t.Fatalf("невалидные транзакции создали конфликты: %d", b.ConflictsCount())
	}
	// все три отсечены валидатором (подпись, PoW, чужая сеть)
	if b.Stats().SpamRejected < 3 {
		t.Fatalf("счётчик спама не вырос: %+v", b.Stats())
	}
	if bad := b.Ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("инварианты: %v", bad)
	}

	// 4. превышение размера: pubsub обрывает соединение с нарушителем
	// (лимит кадра MaxTxSize) — отдельное соединение
	epBig := newEvilPublisher(t, e, manualAddrs(t, b)[0])
	time.Sleep(300 * time.Millisecond)
	big := e.transferTx(a, "V1", "V2", 10)
	big.NetworkID = make([]byte, 70*1024)
	epBig.publish(t, big)
	time.Sleep(time.Second)
	if b.Stats().JournalLen != 1 {
		t.Fatalf(" oversized-транзакция применена: %d", b.Stats().JournalLen)
	}

	// 5. очередь не переполняется: поток невалидных транзакций
	before := b.Stats().SpamRejected
	for i := 0; i < 200; i++ {
		// каждая транзакция уникальна (иначе pubsub дедуплицирует по txid)
		spam := e.transferTx(a, "V1", "V2", uint64(10+i))
		spam.Signature = []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}
		ep.publish(t, spam)
	}
	time.Sleep(3 * time.Second)
	st := b.Stats()
	if st.JournalLen != 1 || st.PendingN != 0 {
		t.Fatalf("спам переполнил очередь: %+v", st)
	}
	if st.SpamRejected <= before {
		t.Fatalf("поток спама не учтён: %+v (было %d)", st, before)
	}
	// валидная транзакция по-прежнему проходит после спама
	ok := e.transferTx(a, "V1", "V2", 10)
	if res, err := a.SubmitTx(ctx, ok); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("после спама валидная отклонена: %v/%s", res.Outcome, res.Code)
	}
	if !waitJournal(t, b, 2, 15*time.Second) {
		t.Fatalf("после спама валидная не дошла: %+v", b.Stats())
	}
}

// TestCheapCheckPoW — дешёвая проверка отсекает неверный алгоритм PoW
// и чужую подпись.
func TestCheapCheckPoW(t *testing.T) {
	e := newTestEnv(t)
	n := e.newNode(nodeOpts{level: LevelFull})
	tx := e.mintTx(n, "V1")
	// усиленный тип с лёгким PoW
	tx2 := e.mintTx(n, "V1")
	tx2.Pow.Algorithm = wire.PowLight
	if err := n.cheapCheck(tx2); err == nil {
		t.Fatal("лёгкий PoW на усиленном типе прошёл дешёвую проверку")
	}
	// валидная транзакция проходит
	if err := n.cheapCheck(tx); err != nil {
		t.Fatalf("валидная транзакция не прошла дешёвую проверку: %v", err)
	}
	// подпись не той парой
	tx3 := e.transferTx(n, "V1", "V2", 10)
	sh := tx3.SigningHash()
	sig, err := crypto.SignDER(e.key("V3"), sh[:])
	if err != nil {
		t.Fatal(err)
	}
	tx3.Signature = sig
	if err := n.cheapCheck(tx3); err == nil {
		t.Fatal("чужая подпись прошла дешёвую проверку")
	}
}
