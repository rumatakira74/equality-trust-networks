// Стабильность canonical-кодирования: хеши txid/signing_hash обязаны
// совпадать между узлами (spec/01 1.2); unknown-поля запрещены (spec/00 0.2).
package wire

import (
	"bytes"
	"testing"
)

func sampleTx() *Transaction {
	return &Transaction{
		NetworkID:       []byte("net-1"),
		Wallet:          bytes.Repeat([]byte{0xAB}, 32),
		Seq:             7,
		Prev:            bytes.Repeat([]byte{0xCD}, 32),
		Type:            Transfer,
		TimestampClaim:  1790000000,
		ProtocolVersion: 1,
		Pow:             &PowProof{Algorithm: PowLight, Nonce: []byte{1, 2, 3}},
		Signature:       bytes.Repeat([]byte{0xEE}, 70),
		Payload:         &TransferPayload{Send: &TransferSend{Recipient: bytes.Repeat([]byte{0x01}, 32), Amount: 500}},
	}
}

func TestCanonicalRoundTrip(t *testing.T) {
	tx := sampleTx()
	raw := tx.Encode()
	back, err := DecodeTransaction(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	raw2 := back.Encode()
	if !bytes.Equal(raw, raw2) {
		t.Fatal("canonical-кодирование не стабильно: repeated encode differs")
	}
	if back.TxID() != tx.TxID() {
		t.Fatal("txid не стабилен")
	}
	if back.SigningHash() != tx.SigningHash() {
		t.Fatal("signing_hash не стабилен")
	}
}

func TestCanonicalUnknownFieldRejected(t *testing.T) {
	tx := sampleTx()
	raw := tx.Encode()
	// дописать unknown-поле 99 (varint)
	tampered := append(append([]byte{}, raw...), 0x98, 0x06, 0x01)
	if _, err := DecodeTransaction(tampered); err == nil {
		t.Fatal("unknown-поле должно отклоняться")
	}
}

func TestCanonicalZeroOmitted(t *testing.T) {
	// нулевой amount не кодируется (proto3): байты стабильны
	a := &Transaction{Wallet: []byte("w"), Type: Mint, Payload: &MintPayload{Period: 5}}
	b := &Transaction{Wallet: []byte("w"), Type: Mint, Payload: &MintPayload{Period: 5, Amount: 0}}
	if !bytes.Equal(a.Encode(), b.Encode()) {
		t.Fatal("нулевые значения должны кодироваться одинаково")
	}
}

func TestGenesisHashStable(t *testing.T) {
	// manifest_hash детерминирован
	m := &Manifest{NetworkID: []byte("n"), DisplayName: "x"}
	if m.ManifestHash() != m.ManifestHash() {
		t.Fatal("manifest_hash не детерминирован")
	}
}

func TestStateLeafRoundTrip(t *testing.T) {
	leaf := &StateLeaf{
		Seq: 3, Balance: 42, Status: StatusActive,
		Flags:           &WalletFlags{Flagged: true},
		RegisteredEpoch: 5, ActivatedEpoch: 6, LastClaimedPeriod: 7,
		DelayedThreshold: 3000, ThresholdChangeEpoch: 9,
	}
	raw := leaf.Encode()
	back, err := DecodeStateLeaf(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(raw, back.Encode()) {
		t.Fatal("StateLeaf не стабилен")
	}
}
