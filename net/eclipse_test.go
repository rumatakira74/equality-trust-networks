// Eclipse-тест (4.9, критерий готовности этапа): узел, все источники
// которого контролируются атакующим, обязан либо получить противоречие
// от постороннего источника, либо не принять единственную версию реестра
// как окончательную (до финализации — этап 5).
package net

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// TestEclipseSubnetLinks — несвязанность по подсетям (№35, этап 6):
// подтверждение — N различных групп связи, а не N пиров. Пиры одной
// подсети (/24) считаются связанными и не увеличивают счётчик.
func TestEclipseSubnetLinks(t *testing.T) {
	subnet := func(id, ip string) (peer.ID, string) {
		return peer.ID(id), ip
	}
	p1, g1 := subnet("p1", "10.0.1.0/24")
	p2, g2 := subnet("p2", "10.0.1.0/24") // та же подсеть, что у p1
	p3, g3 := subnet("p3", "10.0.2.0/24")
	p4, g4 := subnet("p4", "10.0.3.0/24")
	linkOf := func(p peer.ID) string {
		switch p {
		case p1:
			return g1
		case p2:
			return g2
		case p3:
			return g3
		case p4:
			return g4
		}
		return "peer:" + string(p)
	}
	c := NewCorroborationLinked(3, linkOf)
	txid := []byte("txid-subnet-test")
	c.Add(txid, p1)
	c.Add(txid, p2)
	c.Add(txid, p3)
	// три пира, но только две подсети — подтверждения нет
	if c.Confirmed(txid) {
		t.Fatal("три пира в двух подсетях посчитаны как N=3 несвязанных")
	}
	if peers, groups := c.LinkSpread(txid); peers != 3 || groups != 2 {
		t.Fatalf("метрика несвязанности: пиры=%d группы=%d, ожидается 3/2", peers, groups)
	}
	// четвёртая подсеть доводит группы до N
	c.Add(txid, p4)
	if !c.Confirmed(txid) {
		t.Fatal("три различные подсети не дали подтверждения N=3")
	}
	if _, groups := c.LinkSpread(txid); groups != 3 {
		t.Fatalf("групп после четвёртого пира: %d, ожидается 3", groups)
	}
}

// TestEclipseSubnetLinkMultiaddr — группа связи из multiaddr соединения:
// IPv4 → /24, IPv6 → /64, без IP — пустая (уникальная группа на пира).
func TestEclipseSubnetLinkMultiaddr(t *testing.T) {
	a4, err := ma.NewMultiaddr("/ip4/203.0.113.7/tcp/9001")
	if err != nil {
		t.Fatal(err)
	}
	if g := PeerSubnetLink([]ma.Multiaddr{a4}); g != "203.0.113.0/24" {
		t.Fatalf("IPv4: группа %q, ожидается 203.0.113.0/24", g)
	}
	a6, err := ma.NewMultiaddr("/ip6/2001:db8:1:2:3:4:5:6/tcp/9001")
	if err != nil {
		t.Fatal(err)
	}
	if g := PeerSubnetLink([]ma.Multiaddr{a6}); g != "2001:db8:1:2::/64" {
		t.Fatalf("IPv6: группа %q, ожидается 2001:db8:1:2::/64", g)
	}
	if g := PeerSubnetLink(nil); g != "" {
		t.Fatalf("без адреса: группа %q, ожидается пустая", g)
	}
}

// TestEclipseSameSubnetNotUnrelated — по реальным соединениям localhost:
// все пиры в одной подсети 127.0.0.0/24 — связанные; подтверждение N=2
// не достигается, пока deliveries из одной подсети (LinkGroups: true).
func TestEclipseSameSubnetNotUnrelated(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	e.start(n1)
	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1), corroN: 2, linkGroups: true})
	e.start(n2)
	if !waitPeers(t, n2, 1, 15*time.Second) {
		t.Fatal("узлы не соединились")
	}
	ctx := context.Background()
	mint := e.mintTx(n1, "V1")
	if res, err := n1.SubmitTx(ctx, mint); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	if !waitJournal(t, n2, 1, 30*time.Second) {
		t.Fatal("транзакция не доставлена")
	}
	txid := mint.TxID()
	if _, _, _, err := n2.SyncJournalFrom(ctx, n1.Host.ID(), 0, nil); err != nil {
		t.Fatalf("сверка: %v", err)
	}
	// доставка от одного пира: одна группа (127.0.0.0/24), N=2 не достигнуто
	if n2.Corroboration.Confirmed(txid[:]) {
		peers, groups := n2.Corroboration.LinkSpread(txid[:])
		t.Fatalf("одна подсеть принята как несвязанные: пиры=%d группы=%d", peers, groups)
	}
}

// TestEclipseIsolation — жертва подключена только к узлам атакующего.
// Транзакция применяется (доставка — не финальность), но подтверждение
// данными N несвязанных пиров (12.3) не достигнуто: версия реестра не
// окончательная. Посторонний источник (честный узел) доводит
// подтверждение до N.
func TestEclipseIsolation(t *testing.T) {
	e := newTestEnv(t)
	// атакующий: два узла под его контролем
	a1 := e.newNode(nodeOpts{level: LevelFull})
	e.start(a1)
	a2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a1)})
	e.start(a2)

	// жертва: единственные источники — узлы атакующего (N = 3)
	v := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a1, a2), corroN: 3})
	e.start(v)
	if !waitPeers(t, v, 2, 15*time.Second) {
		t.Fatal("жертва не подключилась к узлам атакующего")
	}
	time.Sleep(500 * time.Millisecond)

	// атакующий публикует транзакцию
	ctx := context.Background()
	mint := e.mintTx(a1, "V1")
	if res, err := a1.SubmitTx(ctx, mint); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	// ждём доставки жертве (gossip)
	if !waitJournal(t, v, 1, 30*time.Second) {
		t.Fatalf("жертва не получила транзакцию: %d", v.Stats().JournalLen)
	}
	// полная сверка с каждым подключённым (только атакующий): каждый
	// ответ подтверждает каждую транзакцию журнала (corroboration)
	if _, _, _, err := v.SyncJournalFrom(ctx, a1.Host.ID(), 0, nil); err != nil {
		t.Fatalf("сверка с A1: %v", err)
	}
	if _, _, _, err := v.SyncJournalFrom(ctx, a2.Host.ID(), 0, nil); err != nil {
		t.Fatalf("сверка с A2: %v", err)
	}
	txid := mint.TxID()

	// (1) версия не окончательная: подтверждение < N, хотя применена
	if v.Corroboration.Confirmed(txid[:]) {
		t.Fatal("eclipse: версия от единственного источника принята как окончательная")
	}
	if v.Corroboration.Count(txid[:]) > v.Corroboration.N() {
		t.Fatalf("подтверждений больше N: %d", v.Corroboration.Count(txid[:]))
	}
	if v.Stats().JournalLen != 1 {
		t.Fatalf("журнал жертвы: %d", v.Stats().JournalLen)
	}

	// (2) посторонний источник: честный узел подключается к атакующему
	// (получает ту же транзакцию) и к жертве; сверка с ним доводит
	// подтверждение до N — противоречия нет, данные подтверждены.
	h := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a1, v)})
	e.start(h)
	if !waitPeers(t, h, 2, 15*time.Second) {
		t.Fatal("честный узел не подключился")
	}
	if !waitJournal(t, h, 1, 30*time.Second) {
		t.Fatalf("честный узел не получил транзакцию: %d", h.Stats().JournalLen)
	}
	// жертва сверяется с посторонним источником (полная, от genesis)
	if _, _, _, err := v.SyncJournalFrom(ctx, h.Host.ID(), 0, nil); err != nil {
		t.Fatalf("сверка с честным узлом: %v", err)
	}
	if !v.Corroboration.Confirmed(txid[:]) {
		count := v.Corroboration.Count(txid[:])
		t.Fatalf("после постороннего источника подтверждение не достигло N: %d", count)
	}
	// реестр жертвы не искажён: корень совпадает с честным узлом
	rv, _ := v.StateRoot()
	rh, _ := h.StateRoot()
	if string(rv) != string(rh) {
		t.Fatal("корни жертвы и честного узла расходятся")
	}
}
