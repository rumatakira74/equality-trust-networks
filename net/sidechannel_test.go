// Тесты офлайн-канала (4.10, раздел 12.4): полезная нагрузка —
// wire-Transaction; полученная транзакция кладётся со статусом «ожидает
// публикации»; «получено по Bluetooth» ≠ оплата; при публикации —
// проверка seq/prev (в core/ledger); статус финализации — заглушка
// до этапа 5 (подтверждение N пирами).
package net

import (
	"context"
	"testing"
	"time"
)

// TestSideChannelAwaitingPublication — полученное по офлайн-каналу
// ожидает публикации и не является оплатой до подтверждения.
func TestSideChannelAwaitingPublication(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull})
	e.start(a)

	// V1 выпускает средства, затем собирает офлайн-перевод (V1 -> V2)
	ctxBg := context.Background()
	if res, err := a.SubmitTx(ctxBg, e.mintTx(a, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	tx := e.transferTx(a, "V1", "V2", 100)
	ch := NewMemSideChannel(4)
	if err := ch.Send(tx); err != nil {
		t.Fatal(err)
	}
	// получатель был офлайн; подключается и публикует
	receiver := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a)})
	e.start(receiver)
	if !waitJournal(t, receiver, 1, 30*time.Second) {
		t.Fatalf("получатель не догнал выпуск: %d", receiver.Stats().JournalLen)
	}
	recv := <-ch.Receive()
	receiver.OnOfflineTx(recv)

	id := recv.TxID()
	if got := receiver.OfflineStatus(id); got != OfflineAwaiting {
		t.Fatalf("статус до публикации: %v, ожидается awaiting_publication", got)
	}
	// публикация: проверка seq/prev в core/ledger (12.4) и рассылка
	receiver.PublishAwaiting(context.Background())
	if got := receiver.OfflineStatus(id); got == OfflineAwaiting {
		t.Fatal("после публикации статус не изменился")
	}
	if w := receiver.Ledger.State.Wallet(e.addr("V2")); w == nil || w.TotalBalance() != 100 {
		t.Fatalf("баланс V2 после публикации: %+v", w)
	}
	// финализация — заглушка до этапа 5: без подтверждения N пирами
	// статус «опубликовано», но не «финализировано» (не оплата)
	if got := receiver.OfflineStatus(id); got != OfflinePublished {
		t.Fatalf("статус без подтверждения: %v, ожидается published", got)
	}
}

// TestSideChannelPublishPropagates — опубликованная офлайн-транзакция
// доходит до сети; эквивокация (двойная трата по каналу) обнаруживается
// проверкой seq/prev при публикации.
func TestSideChannelPublishPropagates(t *testing.T) {
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull})
	e.start(a)
	b := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a)})
	e.start(b)
	if !waitPeers(t, b, 1, 15*time.Second) {
		t.Fatal("B не подключился")
	}
	time.Sleep(500 * time.Millisecond)

	// V1 выпускает (баланс), затем офлайн-перевод B2
	ctx := context.Background()
	if res, err := a.SubmitTx(ctx, e.mintTx(a, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	offline := e.transferTx(a, "V1", "V2", 100)
	ch := NewMemSideChannel(4)
	if err := ch.Send(offline); err != nil {
		t.Fatal(err)
	}
	recv := <-ch.Receive()
	b.OnOfflineTx(recv)
	b.PublishAwaiting(ctx)
	if !waitJournal(t, b, 2, 30*time.Second) {
		t.Fatalf("опубликованная офлайн-транзакция не применена: %+v", b.Stats())
	}
	// эквивокация: та же сумма второму получателю по каналу — при
	// публикации ядро фиксирует конфликт (seq/prev заняты; 12.4:
	// быстрая находка эквивокации), применённой остаётся первая ветка
	double := e.transferTx(a, "V1", "V3", 100)
	ch2 := NewMemSideChannel(4)
	if err := ch2.Send(double); err != nil {
		t.Fatal(err)
	}
	recv2 := <-ch2.Receive()
	b.OnOfflineTx(recv2)
	b.PublishAwaiting(ctx)
	if b.ConflictsCount() < 1 {
		t.Fatal("эквивокация не обнаружена при публикации")
	}
	if w := b.Ledger.State.Wallet(e.addr("V3")); w != nil && w.TotalBalance() != 0 {
		t.Fatalf("двойная трата применилась: %+v", w)
	}
	// «получено по Bluetooth» ≠ оплата: без подтверждения N пирами
	// статус финализации не достигнут (заглушка до этапа 5)
	if got := b.OfflineStatus(double.TxID()); got == OfflineFinalized {
		t.Fatal("неподтверждённая транзакция финализирована")
	}
}
