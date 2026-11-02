// F16 (аудит контура 1, высокая значимость): «финализировано» офлайн-
// требования достигается только подтверждениями из финализированных веток.
// Unit-тест гейта Corroboration и интеграционный тест двойной траты
// офлайн-канала: 3 соучастника до финальности не дают «финализировано» для
// проигрывающей конфликт tx; честный путь (финализированная ветка + N пиров)
// «финализировано» достигает.
package net

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TestF16CorroborationFinalityGate — unit: Confirmed гейтится предикатом
// финальности; без предиката — прежнее поведение (обратная совместимость).
func TestF16CorroborationFinalityGate(t *testing.T) {
	txid := []byte("f16-unit-txid")
	c := NewCorroboration(3)
	finalized := false
	c.SetFinalityCheck(func(id []byte) bool { return finalized })
	for _, p := range []peer.ID{"a", "b", "c"} {
		c.Add(txid, p)
	}
	if peers, groups := c.LinkSpread(txid); peers != 3 || groups != 3 {
		t.Fatalf("доставка: пиры=%d группы=%d", peers, groups)
	}
	if c.Confirmed(txid) {
		t.Fatal("F16: 3 пира до финальности дали Confirmed")
	}
	finalized = true
	if !c.Confirmed(txid) {
		t.Fatal("F16: после финальности 3 пира должны давать Confirmed")
	}
	// nil-предикат — старое поведение
	c2 := NewCorroboration(2)
	c2.Add(txid, "x")
	c2.Add(txid, "y")
	if !c2.Confirmed(txid) {
		t.Fatal("без предиката финальности Confirmed должен работать как раньше")
	}
}

// TestF16OfflineDoubleSpend — двойная трата офлайн-канала (F16): tx A
// доставлена жертве 3 «соучастниками» (3 группы) ДО финальности, конфликтная
// ветка B побеждает — «финализировано» для A не наступает ни при каком
// раскладе; для B (финализированная ветка + 3 пира) — наступает.
func TestF16OfflineDoubleSpend(t *testing.T) {
	e := newTestEnv(t)
	// честная сеть, короткие эпохи (2 с): V1,V2,V3 — узлы-подписанты
	n1 := e.consNode("V1", nodeOpts{level: LevelFull})
	e.start(n1)
	n2 := e.consNode("V2", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	n3 := e.consNode("V3", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n3)
	// жертва: N=3, linkGroups=false (несвязанность = уникальные Peer ID —
	// негативный контроль классификатора подсетей), подключена ко всем узлам
	v := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1, n2, n3), corroN: 3})
	e.start(v)
	if !waitPeers(t, v, 3, 15*time.Second) {
		t.Fatal("жертва не подключилась к честным узлам")
	}
	ctx := context.Background()
	// V1 выпускает (seq 1) — баланс для переводов
	mint := e.mintTx(n1, "V1")
	if res, err := n1.SubmitTx(ctx, mint); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	if !waitJournal(t, v, 1, 30*time.Second) {
		t.Fatal("жертва не получила выпуск")
	}
	// двойная трата: A (V1->V2) и B (V1->V3) с одним seq/prev
	txA := e.transferTx(n1, "V1", "V2", 100)
	txB := e.transferTx(n1, "V1", "V3", 100)
	if txA.Seq != txB.Seq || string(txA.Prev) != string(txB.Prev) {
		t.Fatal("A и B должны конфликтовать по seq/prev")
	}
	// B публикуется в честной сети первой — выигрывает финализацию
	bStart := time.Now()
	if res, err := n1.SubmitTx(ctx, txB); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("B: %v/%s", res.Outcome, res.Code)
	}
	// A доставляется жертве «соучастниками» (3 фейковых пира = 3 группы)
	// ДО финальности — как 3 sybil-реле в 3 подсетях
	for _, p := range []peer.ID{"atk1", "atk2", "atk3"} {
		v.handleTxFromPeer(txA, p, true)
	}
	idA := txA.TxID()
	if peers, groups := v.Corroboration.LinkSpread(idA[:]); peers != 3 || groups != 3 {
		t.Fatalf("доставка A: пиры=%d группы=%d, ожидается 3/3", peers, groups)
	}
	// F16: до финальности «финализировано» НЕ достигается
	if v.Corroboration.Confirmed(idA[:]) {
		t.Fatal("F16: 3 соучастника до финальности дали Confirmed(A)")
	}
	if got := v.OfflineStatus(idA); got == OfflineFinalized {
		t.Fatalf("F16: OfflineStatus(A) = %v до финальности", got)
	}
	t.Logf("F16: A доставлена 3 группами, до финальности статус=%v", v.OfflineStatus(idA))

	// ждём финализацию: чекпоинт на жертве (B в финализированной ветке)
	idB := txB.TxID()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if v.CheckpointCount() >= 1 && v.isFinalizedTx(idB[:]) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if v.CheckpointCount() < 1 {
		t.Fatal("чекпоинт не финализирован на жертве за 90 с")
	}
	if !v.isFinalizedTx(idB[:]) {
		t.Fatal("B должна быть в финализированной ветке после чекпоинта")
	}
	if v.isFinalizedTx(idA[:]) {
		t.Fatal("A проиграла конфликт, но попала в финализированную ветку")
	}
	// B доставлена жертве честными узлами (gossip): ждём 3 группы
	deadline = time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, groups := v.Corroboration.LinkSpread(idB[:]); groups >= 3 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if _, groups := v.Corroboration.LinkSpread(idB[:]); groups < 3 {
		t.Fatalf("B доставлена %d группами, ожидается >=3", groups)
	}
	// честный путь: B финализирована + подтверждена N пирами -> «финализировано»
	if !v.Corroboration.Confirmed(idB[:]) {
		t.Fatal("F16: честный путь — Confirmed(B) после финальности не достигнут")
	}
	if got := v.OfflineStatus(idB); got != OfflineFinalized {
		t.Fatalf("F16: OfflineStatus(B) = %v, ожидается finalized", got)
	}
	t.Logf("F16: честный путь — B финализирована за %s, статус=%v",
		time.Since(bStart).Round(time.Millisecond), v.OfflineStatus(idB))
	// двойная трата: A так и не получила «финализировано»
	if v.Corroboration.Confirmed(idA[:]) {
		t.Fatal("F16: проигравшая конфликт A получила Confirmed")
	}
	if got := v.OfflineStatus(idA); got == OfflineFinalized {
		t.Fatalf("F16: OfflineStatus(A) = %v после финализации конфликта", got)
	}
	t.Logf("F16: A (двойная трата) после финализации конфликта: статус=%v", v.OfflineStatus(idA))
}
