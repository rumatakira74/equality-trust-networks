// Бенчмарки хранилища (этап 16, №38): вставка и чтение журнала —
// критерий «скорость на ARM» при выборе SQLite-драйвера. Замеры:
// go test ./core/store/ -bench Store -benchtime 5x
package store

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// benchTx — транзакция журнала фиксированного размера (transfer).
func benchTx(seq uint64) *wire.Transaction {
	return &wire.Transaction{
		NetworkID:       []byte("bench-network-id"),
		Wallet:          []byte("bench-wallet-address-32-bytes-0000"),
		Seq:             seq,
		Prev:            make([]byte, 32),
		Type:            wire.Transfer,
		ProtocolVersion: 3,
		TimestampClaim:  1760000000 + seq,
		Payload: &wire.TransferPayload{Send: &wire.TransferSend{
			Recipient: []byte("bench-recipient-address-32-bytes00"), Amount: 100,
		}},
	}
}

// benchJournal — журнал из n транзакций (уникальные seq).
func benchJournal(n int) []*wire.Transaction {
	out := make([]*wire.Transaction, n)
	for i := 0; i < n; i++ {
		out[i] = benchTx(uint64(i + 1))
	}
	return out
}

// BenchmarkStoreInsert1000 — вставка 1000 транзакций журнала (приём
// сверки/публикации; одна транзакция — один Exec).
func BenchmarkStoreInsert1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s, err := Open(b.TempDir() + "/bench.db")
		if err != nil {
			b.Fatal(err)
		}
		for _, tx := range benchJournal(1000) {
			if err := s.SaveTx(tx, uint64(i)); err != nil {
				b.Fatal(err)
			}
		}
		if err := s.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStoreLoad1000 — чтение журнала 1000 транзакций (рестарт
// приложения: LoadTxs -> cons.Replay strict).
func BenchmarkStoreLoad1000(b *testing.B) {
	s, err := Open(b.TempDir() + "/bench.db")
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	for _, tx := range benchJournal(1000) {
		if err := s.SaveTx(tx, 0); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txs, err := s.LoadTxs()
		if err != nil {
			b.Fatal(err)
		}
		if len(txs) != 1000 {
			b.Fatalf("прочитано %d, хочу 1000", len(txs))
		}
	}
}
