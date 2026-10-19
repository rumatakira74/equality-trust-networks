// Тесты криптопакета: подписи DER, PoW, деривация адреса (PENDING №2).
package crypto

import (
	"bytes"
	"testing"
)

func TestSignVerifyDER(t *testing.T) {
	priv := DeterministicKey("test")
	hash := bytes.Repeat([]byte{0x77}, 32)
	sig, err := SignDER(priv, hash)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !VerifyDER(&priv.PublicKey, hash, sig) {
		t.Fatal("валидная подпись не проходит")
	}
	tampered := append([]byte{}, sig...)
	tampered[len(tampered)-1] ^= 0x01
	if VerifyDER(&priv.PublicKey, hash, tampered) {
		t.Fatal("битая подпись проходит")
	}
	other := DeterministicKey("other")
	if VerifyDER(&other.PublicKey, hash, sig) {
		t.Fatal("чужой ключ проходит")
	}
}

func TestPowLight(t *testing.T) {
	pre := []byte("preimage")
	target := uint64(0x0010_0000_0000_0000)
	nonce := SolveLight(pre, target)
	if !VerifyLight(pre, nonce, target) {
		t.Fatal("лёгкий PoW не сходится")
	}
	if VerifyLight([]byte("other"), nonce, target) {
		t.Fatal("PoW сходится на чужом preimage")
	}
}

func TestPowHard(t *testing.T) {
	pre := []byte("preimage-hard")
	target := uint64(0x0100_0000_0000_0000)
	ap := Argon2Params{MemoryKiB: 8, Iterations: 1, Parallelism: 1}
	nonce := SolveHard(pre, target, ap)
	if !VerifyHard(pre, nonce, target, ap) {
		t.Fatal("усиленный PoW не сходится")
	}
}

func TestAddressDerivation(t *testing.T) {
	// PENDING: вопрос №2 — заменяемая функция; дефолт SHA-256(fp || network)
	fp := []byte("fingerprint")
	a1 := DeriveAddress(fp, []byte("net1"))
	a2 := DeriveAddress(fp, []byte("net2"))
	if bytes.Equal(a1, a2) {
		t.Fatal("адрес не привязан к сети")
	}
	if len(a1) != 32 {
		t.Fatalf("адрес %d байт, ожидается 32", len(a1))
	}
}
