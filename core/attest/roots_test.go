// Тесты корней Google (A2, этап 14): встроенный набор — оба корня
// (RSA + ECDSA P-384), источник — официальная страница Key Attestation;
// SHA256 DER закреплены, чтобы подмена файла обнаружилась тестом.
package attest

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
)

// SHA256(DER) корней с официальной страницы Google (этап 14).
const (
	rootRSASHA256   = "cedb1cb6dc896ae5ec797348bce9286753c2b38ee71ce0fbe34a9a1248800dfc"
	rootECDSASHA256 = "6d9db4ce6c5c0b293166d08986e05774a8776ceb525d9e4329520de12ba4bcc0"
)

func TestGoogleRoots(t *testing.T) {
	roots, err := GoogleRoots()
	if err != nil {
		t.Fatalf("GoogleRoots: %v", err)
	}
	if len(roots) != 2 {
		t.Fatalf("ожиданы 2 корня Google (RSA + ECDSA), получено %d", len(roots))
	}
	got := map[string]bool{}
	for _, r := range roots {
		if !r.IsCA {
			t.Errorf("корень %v: IsCA = false", r.Subject)
		}
		sum := sha256.Sum256(r.Raw)
		got[hex.EncodeToString(sum[:])] = true
		switch pub := r.PublicKey.(type) {
		case *rsa.PublicKey:
			if pub.N.BitLen() != 4096 {
				t.Errorf("RSA-корень: ожидалось 4096 бит, %d", pub.N.BitLen())
			}
		case *ecdsa.PublicKey:
			if pub.Curve.Params().BitSize != 384 {
				t.Errorf("ECDSA-корень: ожидался P-384, %d бит", pub.Curve.Params().BitSize)
			}
		default:
			t.Errorf("неожиданный тип ключа корня: %T", r.PublicKey)
		}
	}
	for _, want := range []string{rootRSASHA256, rootECDSASHA256} {
		if !got[want] {
			t.Errorf("корень с SHA256 %s отсутствует в наборе (есть %v)", want, got)
		}
	}
}

func TestIsSyntheticRoot(t *testing.T) {
	key := crypto.DeterministicKey("is-syn")
	cert, _, err := SyntheticRoot("seed", key)
	if err != nil {
		t.Fatalf("synthetic root: %v", err)
	}
	if !IsSyntheticRoot(cert) {
		t.Error("синтетический корень не распознан")
	}
	roots, err := GoogleRoots()
	if err != nil {
		t.Fatalf("GoogleRoots: %v", err)
	}
	for _, r := range roots {
		if IsSyntheticRoot(r) {
			t.Errorf("корень Google %v распознан как синтетический", r.Subject)
		}
	}
	if IsSyntheticRoot(nil) {
		t.Error("nil распознан как синтетический корень")
	}
}
