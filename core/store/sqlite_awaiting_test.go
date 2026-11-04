// Юнит-тесты офлайн-буфера публикации в SQLite (тема fix-d11-buffer):
// запись/загрузка/дедуп по txid/удаление, восстановление после Close/Open.
package store

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestStoreAwaitingRoundTrip(t *testing.T) {
	s := openTestStore(t)
	tx1, tx2 := testTx(10), testTx(11)
	if err := s.SaveAwaitingTx(tx1, 1000); err != nil {
		t.Fatalf("save tx1: %v", err)
	}
	if err := s.SaveAwaitingTx(tx2, 1001); err != nil {
		t.Fatalf("save tx2: %v", err)
	}
	txs, err := s.LoadAwaitingTxs()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("в буфере %d транзакций, ожидается 2", len(txs))
	}
	if txs[0].Seq != 10 || txs[1].Seq != 11 {
		t.Fatalf("порядок нарушен: seq %d, %d (ожидается 10, 11)", txs[0].Seq, txs[1].Seq)
	}

	// дедуп по txid: повторная запись не растит буфер
	if err := s.SaveAwaitingTx(tx1, 1002); err != nil {
		t.Fatalf("save dup: %v", err)
	}
	txs, err = s.LoadAwaitingTxs()
	if err != nil {
		t.Fatalf("load after dup: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("дедуп сломан: %d транзакций, ожидается 2", len(txs))
	}

	// удаление по txid
	id1 := tx1.TxID()
	if err := s.DeleteAwaitingTx(id1[:]); err != nil {
		t.Fatalf("delete: %v", err)
	}
	txs, err = s.LoadAwaitingTxs()
	if err != nil {
		t.Fatalf("load after delete: %v", err)
	}
	if len(txs) != 1 || txs[0].Seq != 11 {
		t.Fatalf("после удаления: %d транзакций (ожидалась одна, seq 11)", len(txs))
	}

	// удаление несуществующего txid — не ошибка
	if err := s.DeleteAwaitingTx([]byte("нет-такого")); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}

func TestStoreAwaitingReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "awaiting.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	tx := testTx(7)
	if err := s.SaveAwaitingTx(tx, 1234); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// рестарт: та же БД — буфер переживает Close/Open (D11)
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	txs, err := s2.LoadAwaitingTxs()
	if err != nil {
		t.Fatalf("load after reopen: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("после reopen: %d транзакций, ожидается 1", len(txs))
	}
	want := tx.TxID()
	got := txs[0].TxID()
	if !bytes.Equal(want[:], got[:]) {
		t.Fatalf("txid не совпал после reopen")
	}
	if !bytes.Equal(txs[0].Encode(), tx.Encode()) {
		t.Fatalf("canonical-байты не совпали после reopen")
	}
}
