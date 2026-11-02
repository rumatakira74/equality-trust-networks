// Тесты темы net-DoS (A3): порядок валидации «подпись до дорогой
// работы», бюджет дорогих операций на пира, peer score. Обязательный
// набор промпта: (1) невалидная подпись не запускает Argon2id;
// (2) флуд от одного пира не блокирует честного; (3) счёт флудера
// падает, честного — держится/растёт; (4) регресс полного пути
// валидных tx; (5) валидная подпись + невалидный PoW — код E_POW*,
// дешёвый путь сохранён; (6) бюджет восстанавливается по окну.
package net

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// resign — переподпись tx ключом метки (после правки полей, входящих в
// signing_hash; PoW при этом не пересобирается — тест невалидного PoW
// с валидной подписью).
func (e *testEnv) resign(tx *wire.Transaction, id string) {
	e.t.Helper()
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(e.key(id), sh[:])
	if err != nil {
		e.t.Fatalf("resign: %v", err)
	}
	tx.Signature = sig
}

// TestDosInvalidSigSkipsArgon2 — (1) tx с невалидной подписью и
// синтаксически корректным PoW-полем не запускает Argon2id/PoW:
// счётчик дорогих проверок (expensiveChecks) не растёт. Варианты:
// известный кошелёк (подпись проверяется до PoW), регистрация (ключ
// из payload) и неизвестный кошелёк (дешёвый отказ полного узла —
// ядро дало бы E_UNKNOWN_WALLET).
func TestDosInvalidSigSkipsArgon2(t *testing.T) {
	e := newTestEnv(t)
	n := e.newNode(nodeOpts{level: LevelFull})
	e.start(n)
	from := peer.ID("dos-bad-sig-peer")

	// известный кошелёк: битая подпись, корректное PoW-поле
	badSig := e.mintTx(n, "V1")
	badSig.Signature = []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}
	if err := n.relayAdmit(badSig, from); err == nil {
		t.Fatal("битая подпись прошла путь приёма")
	}
	// регистрация: ключ из payload, подпись битая — Argon2id не запускается
	reg := e.register(n, "DOSREG1")
	reg.Signature = []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}
	if err := n.relayAdmit(reg, from); err == nil {
		t.Fatal("регистрация с битой подписью прошла путь приёма")
	}
	// неизвестный кошелёк: дешёвый отказ полного узла (A3 — без
	// подписи дорогая работа не выполняется вовсе)
	unknown := e.mintTx(n, "V1")
	unknown.Wallet = make([]byte, 32)
	unknown.Wallet[0] = 0xAB
	if err := n.relayAdmit(unknown, from); err == nil {
		t.Fatal("tx неизвестного кошелька прошла путь приёма")
	}
	if got := n.expensiveChecks.Load(); got != 0 {
		t.Fatalf("Argon2id запущен до проверки подписи: %d", got)
	}
}

// TestDosFloodDoesNotBlockHonest — (2) флуд мусорных tx от одного
// пира не блокирует и не замедляет приём валидных tx от честного
// пира (временной замер): под потоком мусора валидная tx доходит до
// узла-жертвы в обычное время, дорогие проверки не выполняются.
func TestDosFloodDoesNotBlockHonest(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull})
	e.start(a)
	b := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a)})
	e.start(b)
	if !waitPeers(t, b, 1, 15*time.Second) {
		t.Fatal("B не подключился к A")
	}
	time.Sleep(500 * time.Millisecond)

	// базовое состояние: V1 выпускает период 1 (баланс для переводов)
	ctx := context.Background()
	if res, err := b.SubmitTx(ctx, e.mintTx(b, "V1")); err != nil || res.Outcome != ledger.Accepted {
		t.Fatalf("mint: %v/%s", err, res.Code)
	}
	if !waitJournal(t, a, 1, 15*time.Second) {
		t.Fatal("базовая tx не дошла до A")
	}

	// флудер подключается к A и заливает мусор (битая подпись,
	// корректное PoW-поле) — как в TestSpamNotRelayed
	ep := newEvilPublisher(t, e, manualAddrs(t, a)[0])
	time.Sleep(500 * time.Millisecond)
	flood := 200
	for i := 0; i < flood; i++ {
		spam := e.transferTx(a, "V1", "V2", uint64(10+i))
		spam.Signature = []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}
		ep.publish(t, spam)
	}

	// валидная tx от честного пира B под флудом: замер времени доставки
	start := time.Now()
	ok := e.transferTx(b, "V1", "V2", 10)
	if res, err := b.SubmitTx(ctx, ok); err != nil || res.Outcome != ledger.Accepted {
		t.Fatalf("валидная tx под флудом отклонена: %v/%s", err, res.Code)
	}
	if !waitJournal(t, a, 2, 15*time.Second) {
		t.Fatalf("валидная tx под флудом не дошла: %+v", a.Stats())
	}
	latency := time.Since(start)
	if latency > 10*time.Second {
		t.Fatalf("флуд замедлил приём валидной tx: %v", latency)
	}
	// дорогие проверки не выполнялись: мусор отсечён подписью,
	// валивная tx — лёгкий PoW (не Argon2id)
	if got := a.expensiveChecks.Load(); got != 1 {
		t.Fatalf("expensiveChecks на A: %d (ожидался 1 — валидный mint)", got)
	}
	if got := int(a.Stats().JournalLen); got != 2 {
		t.Fatalf("журнал A: %d (мусор применён или валидная потеряна)", got)
	}
}

// TestDosPeerScore — (3) счёт пира: у флудера падает, у честного
// держится/растёт; pubsub peer score (WithPeerScore) включён и
// отражает то же.
func TestDosPeerScore(t *testing.T) {
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

	// честная активность: валидные tx от B — счёт B неотрицательный
	if res, err := b.SubmitTx(ctx, e.mintTx(b, "V1")); err != nil || res.Outcome != ledger.Accepted {
		t.Fatalf("mint: %v/%s", err, res.Code)
	}
	if !waitJournal(t, a, 1, 15*time.Second) {
		t.Fatal("валидная tx не дошла")
	}
	if s := a.dos.score(b.Host.ID()); s <= 0 {
		t.Fatalf("счёт честного пира не вырос: %f", s)
	}

	// флудер: мусор с битой подписью — счёт падает
	ep := newEvilPublisher(t, e, manualAddrs(t, a)[0])
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 20; i++ {
		spam := e.transferTx(a, "V1", "V2", uint64(10+i))
		spam.Signature = []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}
		ep.publish(t, spam)
	}
	deadline := time.Now().Add(10 * time.Second)
	for a.dos.score(ep.host.ID()) >= 0 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if s := a.dos.score(ep.host.ID()); s >= 0 {
		t.Fatalf("счёт флудера не упал: %f", s)
	}
	// pubsub peer score (WithPeerScore) включён: параметры приняты при
	// старте (NewGossipSub отклоняет невалидные), его счёт приложения —
	// тот же dosTracker (проверка параметров — TestDosPeerScoreParams)
	// честный не пострадал
	if s := a.dos.score(b.Host.ID()); s < 0 {
		t.Fatalf("счёт честного упал: %f", s)
	}
}

// TestDosValidTxFullPath — (4) регресс: валидные tx (подпись + PoW)
// проходят полный путь как раньше; валидный register/mint получает
// свою Argon2id-проверку (дорогая работа для честных выполняется).
func TestDosValidTxFullPath(t *testing.T) {
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

	// регистрация нового кошелька на B — A применяет через gossip
	regTx := e.register(b, "DOSFULL1")
	if res, err := b.SubmitTx(ctx, regTx); err != nil || res.Outcome != ledger.Accepted {
		t.Fatalf("register: %v/%s", err, res.Code)
	}
	if !waitJournal(t, a, 1, 15*time.Second) {
		t.Fatalf("register не дошла до A: %+v", a.Stats())
	}
	// валидный mint (усиленный PoW) — полный путь, Argon2id выполнен
	mint := e.mintTx(b, "V1")
	if res, err := b.SubmitTx(ctx, mint); err != nil || res.Outcome != ledger.Accepted {
		t.Fatalf("mint: %v/%s", err, res.Code)
	}
	if !waitJournal(t, a, 2, 15*time.Second) {
		t.Fatalf("mint не дошла до A: %+v", a.Stats())
	}
	if got := a.expensiveChecks.Load(); got < 2 {
		t.Fatalf("валидные register+mint не получили дорогую проверку: %d", got)
	}
	// валидный перевод — полный путь
	tr := e.transferTx(b, "V1", "V2", 10)
	if res, err := b.SubmitTx(ctx, tr); err != nil || res.Outcome != ledger.Accepted {
		t.Fatalf("transfer: %v/%s", err, res.Code)
	}
	if !waitJournal(t, a, 3, 15*time.Second) {
		t.Fatalf("transfer не дошла до A: %+v", a.Stats())
	}
	if bad := a.Ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("инварианты: %v", bad)
	}
}

// TestDosValidSigBadPow — (5) валидная подпись + невалидный PoW:
// корректный код E_POW* (ядро), дешёвый путь сохранён — лёгкий PoW
// не расходует бюджет, усиленный расходует ровно одну проверку и
// дальше не выполняется.
func TestDosValidSigBadPow(t *testing.T) {
	e := newTestEnv(t)
	n := e.newNode(nodeOpts{level: LevelFull})
	e.start(n)
	ctx := context.Background()
	from := peer.ID("dos-bad-pow-peer")

	// лёгкий PoW: валидная подпись (переподписана над новым nonce —
	// nonce входит в signing_hash, spec/01 1.2), неверный PoW —
	// E_POW_LIGHT
	badLight := e.transferTx(n, "V1", "V2", 10)
	badLight.Pow.Nonce = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	e.resign(badLight, "V1")
	if err := n.relayAdmit(badLight, from); err == nil {
		t.Fatal("невалидный лёгкий PoW прошёл путь приёма")
	}
	res, err := n.SubmitTx(ctx, badLight)
	if err != nil || res.Outcome != ledger.Rejected || res.Code != "E_POW_LIGHT" {
		t.Fatalf("ожидался E_POW_LIGHT: %v/%s/%v", err, res.Code, res.Outcome)
	}

	// усиленный PoW: валидная подпись, неверный nonce — E_POW_HARD,
	// ровно одна Argon2id-проверка (дальше путь не идёт)
	before := n.expensiveChecks.Load()
	badHard := e.mintTx(n, "V1")
	badHard.Pow.Nonce = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	e.resign(badHard, "V1")
	if err := n.relayAdmit(badHard, from); err == nil {
		t.Fatal("невалидный усиленный PoW прошёл путь приёма")
	}
	if got := n.expensiveChecks.Load(); got != before+1 {
		t.Fatalf("усиленный PoW: ожидалась одна дорогая проверка, получено %d", got-before)
	}
	res, err = n.SubmitTx(ctx, badHard)
	if err != nil || res.Outcome != ledger.Rejected || res.Code != "E_POW_HARD" {
		t.Fatalf("ожидался E_POW_HARD: %v/%s/%v", err, res.Code, res.Outcome)
	}
	// мусор не попал в журнал
	if got := n.Stats().JournalLen; got != 0 {
		t.Fatalf("tx с невалидным PoW применена: журнал %d", got)
	}
}

// TestDosBudgetRecovers — (6) бюджет дорогих операций на пира
// восстанавливается по скользящему окну: исчерпание — дешёвый отказ
// (errDosBudget), после окна — приём возобновляется.
func TestDosBudgetRecovers(t *testing.T) {
	e := newTestEnv(t)
	n := e.newNode(nodeOpts{level: LevelFull, dosBudget: 2, dosWindow: 400 * time.Millisecond})
	e.start(n)
	// валидная mint собирается один раз (сборка решает PoW дольше
	// окна — между проверками узла ничего дорогого не происходит)
	valid := e.mintTx(n, "V1")
	from := peer.ID("dos-budget-peer")

	// бюджет 2 — первые две проверки проходят
	for i := 0; i < 2; i++ {
		if err := n.relayAdmit(valid, from); err != nil {
			t.Fatalf("валидная mint %d отклонена в бюджете: %v", i, err)
		}
	}
	// третья за окно — дешёвый отказ без Argon2id
	before := n.expensiveChecks.Load()
	if err := n.relayAdmit(valid, from); err == nil {
		t.Fatal("исчерпание бюджета не сработало")
	}
	if got := n.expensiveChecks.Load(); got != before {
		t.Fatalf("Argon2id выполнен сверх бюджета: %d", got-before)
	}
	if got := n.budgetRejected.Load(); got != 1 {
		t.Fatalf("счётчик отказов по бюджету: %d", got)
	}
	// окно истекло — бюджет восстановился
	time.Sleep(450 * time.Millisecond)
	if err := n.relayAdmit(valid, from); err != nil {
		t.Fatalf("бюджет не восстановился по окну: %v", err)
	}
	// чужой пир не зависит от чужого бюджета
	if err := n.relayAdmit(valid, peer.ID("dos-budget-peer-2")); err != nil {
		t.Fatalf("бюджет одного пира ограничил другого: %v", err)
	}
}

// TestDosPeerScoreParams — параметры peer score валидны для libp2p
// (WithPeerScore принял их при старте — узел запущен) и порог
// разрыва соединения достижим: счёт флудера уходит ниже порога и
// соединение разрывается (крайняя мера).
func TestDosPeerScoreParams(t *testing.T) {
	e := newTestEnv(t)
	n := e.newNode(nodeOpts{level: LevelFull})
	e.start(n)
	params, thresholds := n.peerScoreParams()
	if params.Topics[n.txTopic()] == nil {
		t.Fatal("нет параметров счёта для темы tx")
	}
	if thresholds.GraylistThreshold >= 0 || thresholds.PublishThreshold >= thresholds.GossipThreshold {
		t.Fatalf("некорректные пороги: %+v", thresholds)
	}
	if params.AppSpecificScore == nil {
		t.Fatal("нет счёта приложения (dosTracker)")
	}
	// счёт пира с шумом уходит ниже порога разрыва соединения
	p := peer.ID("dos-threshold-peer")
	now := time.Now()
	for i := 0; i < 16; i++ {
		n.dos.penalty(p, dosPenaltyInvalidSig, now)
	}
	if s := n.dos.score(p); s >= dosDisconnectScore {
		t.Fatalf("счёт не достиг порога разрыва: %f", s)
	}
}
