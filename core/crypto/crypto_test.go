// Тесты криптопакета: подписи DER, PoW, деривация адреса (PENDING №2).
package crypto

import (
	"bytes"
	"math/big"
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

// TestSignDERCanonicalLowS — каноничность подписи (аудит криптографии,
// этап 9): SignDER всегда выдаёт low-s (s <= N/2), VerifyDER отклоняет
// маllеабельные формы — иначе третья сторона чеканит вторую валидную
// кодировку чужой транзакции (s -> N−s, лишние нули в INTEGER) с другим
// txid (TxID включает подпись) — ложный конфликт той же seq/prev.
func TestSignDERCanonicalLowS(t *testing.T) {
	priv := DeterministicKey("low-s")
	for i := 0; i < 64; i++ {
		hash := bytes.Repeat([]byte{byte(i + 1)}, 32)
		sig, err := SignDER(priv, hash)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if !VerifyDER(&priv.PublicKey, hash, sig) {
			t.Fatalf("валидная подпись %d не проходит", i)
		}
		r, s, err := decodeDERSignature(sig)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if s.Cmp(halfN) > 0 {
			t.Fatalf("high-s в подписи %d: %v > %v", i, s, halfN)
		}
		if r.Cmp(ecdsaP256N) >= 0 {
			t.Fatalf("r вне порядка кривой: %v", r)
		}
	}
}

// TestVerifyDERRejectsMalleated — маllеабельные кодировки валидной
// подписи отклоняются: s -> N−s (вторая валидная подпись), неминимальный
// INTEGER (лишний ведущий нуль), «отрицательный» r.
func TestVerifyDERRejectsMalleated(t *testing.T) {
	priv := DeterministicKey("malleated")
	// подбираем хеш с r без старшего бита: тогда ведущий 0x00 у r —
	// НЕминимальная (маllеабельная) кодировка; при старшем бите у r
	// префикс 0x00 обязателен (канонично) и тест теряет смысл
	var hash [32]byte
	var sig []byte
	var r, s *big.Int
	for i := 0; ; i++ {
		for j := range hash {
			hash[j] = byte(i + 1)
		}
		var err error
		sig, err = SignDER(priv, hash[:])
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		r, s, err = decodeDERSignature(sig)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if rb := r.Bytes(); rb[0]&0x80 == 0 {
			break
		}
	}

	// s -> N−s: математически валидная «вторая» подпись
	sFlip := new(big.Int).Sub(ecdsaP256N, s)
	flipped, err := encodeDERSignature(r, sFlip)
	if err != nil {
		t.Fatalf("encode flip: %v", err)
	}
	if VerifyDER(&priv.PublicKey, hash[:], flipped) {
		t.Fatal("high-s (s -> N−s) принята — маllеабельность")
	}

	// неминимальный r: ведущий нуль без необходимости
	rb := r.Bytes()
	padded := append([]byte{0x00}, rb...)
	padSig, err := encodeDERSignatureRaw(padded, s.Bytes())
	if err != nil {
		t.Fatalf("encode padded: %v", err)
	}
	if VerifyDER(&priv.PublicKey, hash[:], padSig) {
		t.Fatal("неминимальный INTEGER принят — маllеабельность")
	}

	// «отрицательный» r (знаковый бит без префикса)
	neg := append([]byte{}, rb...)
	neg[0] |= 0x80
	negSig, err := encodeDERSignatureRaw(neg, s.Bytes())
	if err != nil {
		t.Fatalf("encode neg: %v", err)
	}
	if VerifyDER(&priv.PublicKey, hash[:], negSig) {
		t.Fatal("отрицательный INTEGER принят")
	}
}

// encodeDERSignatureRaw — DER-кодировка с заданными байтовыми r/s без
// нормализации (только для тестов маllеабельности).
func encodeDERSignatureRaw(rb, sb []byte) ([]byte, error) {
	out := []byte{0x30, byte(4 + len(rb) + len(sb)), 0x02, byte(len(rb))}
	out = append(out, rb...)
	out = append(out, 0x02, byte(len(sb)))
	out = append(out, sb...)
	return out, nil
}
