// Unit-тесты моста Android (без устройства): сборка транзакции, PoW,
// подпись (мягкий ключ вместо Keystore), кодирование, аттестация.
package tnp

import (
	"bytes"
	"testing"
	"time"
)

// TestBridgeTxFlow — полный цикл транзакции через мост: сборка → лёгкий
// PoW → хеш → подпись → проверка → кодирование → разбор.
func TestBridgeTxFlow(t *testing.T) {
	pair, err := SoftKey()
	if err != nil {
		t.Fatal(err)
	}
	pub, priv := pair.Pub(), pair.PrivDER()
	netID := []byte("tnp-bridge-test")
	addr, err := AddressFromPublicKey(pub, netID)
	if err != nil {
		t.Fatal(err)
	}
	if len(addr) != 32 {
		t.Fatalf("адрес: %d байт", len(addr))
	}
	tx := NewTransfer(netID, addr, 1, make([]byte, 32), addr, 100, 1, time.Now().Unix())
	tx.SolveLight(0x0010_0000_0000_0000)
	hash := tx.SigningHash()
	sig, err := SignWithKey(priv, hash)
	if err != nil {
		t.Fatal(err)
	}
	tx.SetSignature(sig)
	ok, err := tx.VerifySignature(pub)
	if err != nil || !ok {
		t.Fatalf("VerifySignature: %v %v", ok, err)
	}
	// кодирование/разбор (Bluetooth, публикация)
	enc := tx.Encode()
	tx2, err := DecodeTx(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tx.ID(), tx2.ID()) {
		t.Fatal("txid после кодирования изменился")
	}
	if ok, _ := tx2.VerifySignature(pub); !ok {
		t.Fatal("подпись не проверяется после кодирования")
	}
	// tampered-подпись не проходит
	tx2.SetSignature([]byte{1, 2, 3})
	if ok, _ := tx2.VerifySignature(pub); ok {
		t.Fatal("битая подпись прошла")
	}
}

// TestBridgeAttestation — challenge и разбор синтетической цепочки
// (эмулятор); формат реальной цепочки — core/attest/keymint_test.
func TestBridgeAttestation(t *testing.T) {
	pair2, err := SoftKey()
	if err != nil {
		t.Fatal(err)
	}
	pub := pair2.Pub()
	if err != nil {
		t.Fatal(err)
	}
	netID := []byte("tnp-bridge-test")
	chal := AttestationChallengeFor(pub, netID)
	if len(chal) != 32 {
		t.Fatalf("challenge: %d байт", len(chal))
	}
	// синтетическая цепочка эмулятора: корень из SandboxGenesis
	gen := SandboxGenesis("tnp-bridge-test", 0, 3600,
		0x0010_0000_0000_0000, 0x0010_0000_0000_0000, 8, 1, 1)
	rootDER := gen.TrustedRootsDER()
	if len(rootDER) == 0 {
		t.Fatal("корень genesis не построен")
	}
	roots := AppendCert(nil, rootDER)
	// цепочка строится в тестах core/attest; здесь — негатив: разбор
	// мусора должен давать ошибцию, не панику
	if _, err := ParseAttestation([]byte("мусор"), roots, 0, 24, time.Now().Unix()); err == nil {
		t.Fatal("мусорная цепочка разобралась")
	}
}

// TestBridgeGenesis — песочница: параметры PoW и эпохи доходят до узла.
func TestBridgeGenesis(t *testing.T) {
	gen := SandboxGenesis("tnp-bridge-gen", time.Now().Add(-65*24*time.Hour).Unix(),
		7200, 0x0010_0000_0000_0000, 0x0010_0000_0000_0000, 8, 1, 1)
	if gen.gp.Manifest.Profile.EpochSeconds != 7200 {
		t.Fatalf("epoch_seconds: %d", gen.gp.Manifest.Profile.EpochSeconds)
	}
	if gen.gp.Manifest.Profile.Pow.TargetLight != 0x0010_0000_0000_0000 {
		t.Fatalf("pow target: %x", gen.gp.Manifest.Profile.Pow.TargetLight)
	}
	if string(gen.gp.Genesis.NetworkID) != "tnp-bridge-gen" {
		t.Fatalf("network_id: %q", gen.gp.Genesis.NetworkID)
	}
}
