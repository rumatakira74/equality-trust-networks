// Юнит-тесты SQLite-хранилища (тема «Fuzz-долги»: до сих пор был только
// bench). Round-trip всех таблиц, повторная запись, восстановление после
// Close/Open.
package store

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func testTx(seq uint64) *wire.Transaction {
	return &wire.Transaction{
		NetworkID:       []byte("net-1"),
		Wallet:          bytes.Repeat([]byte{0xAB}, 32),
		Seq:             seq,
		Prev:            bytes.Repeat([]byte{0xCD}, 32),
		Type:            wire.Transfer,
		TimestampClaim:  1790000000,
		ProtocolVersion: 3,
		Pow:             &wire.PowProof{Algorithm: wire.PowLight, Nonce: []byte{1, 2, 3}},
		Signature:       bytes.Repeat([]byte{0xEE}, 70),
		Payload:         &wire.TransferPayload{Send: &wire.TransferSend{Recipient: bytes.Repeat([]byte{0x01}, 32), Amount: 500}},
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStoreTxRoundTrip(t *testing.T) {
	s := openTestStore(t)
	txs := []*wire.Transaction{testTx(1), testTx(2), testTx(3)}
	for i, tx := range txs {
		if err := s.SaveTx(tx, uint64(1000+i)); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	// повторная запись того же txid — INSERT OR IGNORE, без дубликата
	if err := s.SaveTx(txs[0], 9999); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	loaded, err := s.LoadTxs()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != len(txs) {
		t.Fatalf("загружено %d tx, ожидается %d", len(loaded), len(txs))
	}
	for i := range txs {
		if !bytes.Equal(txs[i].Encode(), loaded[i].Encode()) {
			t.Fatalf("tx %d: round-trip разошёлся", i)
		}
		if txs[i].TxID() != loaded[i].TxID() {
			t.Fatalf("tx %d: txid нестабилен", i)
		}
	}
}

func TestStoreFingerprints(t *testing.T) {
	s := openTestStore(t)
	fp1, addr1 := []byte("fp-1"), []byte("addr-1")
	fp2, addr2 := []byte("fp-2"), []byte("addr-2")
	if err := s.SaveFingerprint(fp1, addr1); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SaveFingerprint(fp2, addr2); err != nil {
		t.Fatalf("save: %v", err)
	}
	// REPLACE: тот же фингерпринт перезаписывается
	if err := s.SaveFingerprint(fp1, []byte("addr-1b")); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, err := s.LoadFingerprints()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("фингерпринтов %d, ожидается 2", len(got))
	}
	if string(got["fp-1"]) != "addr-1b" {
		t.Fatalf("fp-1 -> %q, ожидается перезаписанный addr-1b", got["fp-1"])
	}
	if string(got["fp-2"]) != "addr-2" {
		t.Fatalf("fp-2 -> %q", got["fp-2"])
	}
}

func TestStoreBlacklist(t *testing.T) {
	s := openTestStore(t)
	if err := s.SaveBlacklisted([]byte("bad-1")); err != nil {
		t.Fatalf("save: %v", err)
	}
	// IGNORE: повторная запись того же фингерпринта не падает
	if err := s.SaveBlacklisted([]byte("bad-1")); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	if err := s.SaveBlacklisted([]byte("bad-2")); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.LoadBlacklist()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 2 || !got["bad-1"] || !got["bad-2"] {
		t.Fatalf("чёрный список: %v", got)
	}
}

func TestStoreCheckpoints(t *testing.T) {
	s := openTestStore(t)
	// заголовки: Latest — по максимальной эпохе, REPLACE перезаписывает
	if err := s.SaveCheckpointHeader(3, []byte("h3")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SaveCheckpointHeader(7, []byte("h7")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SaveCheckpointHeader(7, []byte("h7b")); err != nil {
		t.Fatalf("replace: %v", err)
	}
	epoch, hash, ok, err := s.LatestCheckpointHeader()
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if !ok || epoch != 7 || string(hash) != "h7b" {
		t.Fatalf("latest: epoch=%d hash=%q ok=%v", epoch, hash, ok)
	}
	// предложение: ряд по возрастанию эпох
	for _, e := range []struct {
		epoch  uint64
		supply uint64
	}{{5, 500}, {9, 900}, {7, 700}} {
		if err := s.SaveCheckpointSupply(e.epoch, e.supply); err != nil {
			t.Fatalf("save supply: %v", err)
		}
	}
	epochs, supplies, err := s.LoadCheckpointSupply()
	if err != nil {
		t.Fatalf("load supply: %v", err)
	}
	if len(epochs) != 3 {
		t.Fatalf("строк предложения %d, ожидается 3", len(epochs))
	}
	for i, want := range []uint64{5, 7, 9} {
		if epochs[i] != want {
			t.Fatalf("эпохи не по возрастанию: %v", epochs)
		}
	}
	if supplies[1] != 700 {
		t.Fatalf("supply(7)=%d, ожидается 700", supplies[1])
	}
}

func TestStoreMeta(t *testing.T) {
	s := openTestStore(t)
	if _, ok, err := s.LoadMeta("missing"); err != nil || ok {
		t.Fatalf("missing: ok=%v err=%v", ok, err)
	}
	if err := s.SaveMeta("genesis", []byte("blob-1")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SaveMeta("genesis", []byte("blob-2")); err != nil {
		t.Fatalf("replace: %v", err)
	}
	v, ok, err := s.LoadMeta("genesis")
	if err != nil || !ok || string(v) != "blob-2" {
		t.Fatalf("load: v=%q ok=%v err=%v", v, ok, err)
	}
}

func TestStoreReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	tx := testTx(1)
	if err := s.SaveTx(tx, 1); err != nil {
		t.Fatalf("save tx: %v", err)
	}
	if err := s.SaveFingerprint([]byte("fp"), []byte("addr")); err != nil {
		t.Fatalf("save fp: %v", err)
	}
	if err := s.SaveBlacklisted([]byte("bad")); err != nil {
		t.Fatalf("save bl: %v", err)
	}
	if err := s.SaveCheckpointHeader(2, []byte("h2")); err != nil {
		t.Fatalf("save hdr: %v", err)
	}
	if err := s.SaveCheckpointSupply(2, 200); err != nil {
		t.Fatalf("save supply: %v", err)
	}
	if err := s.SaveMeta("k", []byte("v")); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	loaded, err := s2.LoadTxs()
	if err != nil || len(loaded) != 1 || loaded[0].TxID() != tx.TxID() {
		t.Fatalf("tx после reopen: len=%d err=%v", len(loaded), err)
	}
	fp, err := s2.LoadFingerprints()
	if err != nil || len(fp) != 1 || string(fp["fp"]) != "addr" {
		t.Fatalf("fp после reopen: %v err=%v", fp, err)
	}
	bl, err := s2.LoadBlacklist()
	if err != nil || len(bl) != 1 {
		t.Fatalf("blacklist после reopen: %v err=%v", bl, err)
	}
	if _, hash, ok, err := s2.LatestCheckpointHeader(); err != nil || !ok || string(hash) != "h2" {
		t.Fatalf("header после reopen: %q ok=%v err=%v", hash, ok, err)
	}
	epochs, supplies, err := s2.LoadCheckpointSupply()
	if err != nil || len(epochs) != 1 || supplies[0] != 200 {
		t.Fatalf("supply после reopen: %v err=%v", epochs, err)
	}
	if v, ok, err := s2.LoadMeta("k"); err != nil || !ok || string(v) != "v" {
		t.Fatalf("meta после reopen: %q ok=%v err=%v", v, ok, err)
	}
}
