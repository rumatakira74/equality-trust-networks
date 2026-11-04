// Интеграционный тест офлайн-буфера (тема fix-d11-buffer, долг D11
// FINDING-awaiting-offline-tx-not-persistent): awaiting, принятый по
// офлайн-каналу до рестарта, восстанавливается из store в новом
// экземпляре узла и публикуется при появлении сети; после публикации
// буфер пуст и в памяти, и в store.
package net

import (
	"context"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func TestAwaitingBufferSurvivesRestart(t *testing.T) {
	// мораторий снят: перевод публикуется сразу после рестарта
	e := newTestEnv(t, func(p *wire.ProfileParams) {
		p.TransferMoratoriumDays = 0
	})
	db := dbPath(t, "awaiting.db")
	n1 := e.newNode(nodeOpts{level: LevelFull, db: db})

	// журнал: register отправителя и получателя, mint посевного V1
	// (активирован в периоде 1 — заявляемый период; у SENDER активация в
	// текущем периоде 2, его первый минт возможен только в периоде 3),
	// затем перевод V1 → SENDER (средства отправителя). Transfer строится
	// ПОСЛЕ импорта минта: минт двигает цепочку V1 (seq/prev).
	for _, tx := range []*wire.Transaction{
		e.register(n1, "SENDER"),
		e.register(n1, "RECVR"),
		e.mintTx(n1, "V1"),
	} {
		if n1.ImportJournal([]*wire.Transaction{tx}) != 1 {
			t.Fatalf("не применён tx типа %d (seq %d)", tx.Type, tx.Seq)
		}
	}
	fund := e.tx(n1, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("SENDER"), Amount: 500},
	}, false)
	if n1.ImportJournal([]*wire.Transaction{fund}) != 1 {
		t.Fatalf("не применён перевод V1→SENDER")
	}

	// офлайн-приём: перевод передан «по Bluetooth» узлу-получателю
	transfer := e.tx(n1, "SENDER", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("RECVR"), Amount: 100},
	}, false)
	txid := transfer.TxID()
	n1.OnOfflineTx(transfer)
	if got := n1.OfflineStatus(txid); got != OfflineAwaiting {
		t.Fatalf("статус после приёма: %v, ожидается %v", got, OfflineAwaiting)
	}
	rows, err := n1.Store.LoadAwaitingTxs()
	if err != nil || len(rows) != 1 {
		t.Fatalf("буфер в store: %d записей, err=%v (ожидается 1)", len(rows), err)
	}
	// дедуп: повторный приём той же tx не растит буфер
	n1.OnOfflineTx(transfer)
	if rows, err = n1.Store.LoadAwaitingTxs(); err != nil || len(rows) != 1 {
		t.Fatalf("дедуп сломан: %d записей, err=%v", len(rows), err)
	}

	// force stop
	if err := n1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// рестарт: новый экземпляр на той же БД — буфер восстановлен
	n2 := e.newNode(nodeOpts{level: LevelFull, db: db})
	if got := n2.OfflineStatus(txid); got != OfflineAwaiting {
		t.Fatalf("статус после рестарта: %v, ожидается %v", got, OfflineAwaiting)
	}
	if rows, err = n2.Store.LoadAwaitingTxs(); err != nil || len(rows) != 1 {
		t.Fatalf("буфер после рестарта: %d записей, err=%v", len(rows), err)
	}

	// публикация при появлении сети
	e.start(n2)
	deadline := time.Now().Add(15 * time.Second)
	for {
		n2.PublishAwaiting(context.Background())
		if n2.OfflineStatus(txid) == OfflinePublished {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("публикация не завершилась за 15 с: статус %v", n2.OfflineStatus(txid))
		}
		time.Sleep(300 * time.Millisecond)
	}
	// буфер пуст и в памяти, и в store
	if rows, err = n2.Store.LoadAwaitingTxs(); err != nil || len(rows) != 0 {
		t.Fatalf("буфер после публикации: %d записей, err=%v (ожидается 0)", len(rows), err)
	}
	// сеть приняла перевод: он в журнале узла
	w := n2.Ledger.State.Wallet(e.addr("RECVR"))
	if w == nil || w.Liquid != 100 {
		t.Fatalf("перевод не дошёл до получателя: liquid=%v", w)
	}
}
