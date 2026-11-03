// Fuzz-цели на разбор цепочек аттестации чужих устройств (тема
// «Fuzz-долги»): Verify (DecodeChain + синтетическое/KeyMint-расширение +
// x509-верификация против синтетического корня) и прямой разбор DER
// KeyMint (derScanner/scanAuthorizationList). Ошибки — ошибкой, не паникой.
package attest

import (
	"bytes"
	"crypto/x509"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
)

func FuzzAttestVerify(f *testing.F) {
	rootKey := crypto.DeterministicKey("fuzz-root")
	rootCert, _, err := SyntheticRoot("fuzz", rootKey)
	if err != nil {
		f.Fatalf("root: %v", err)
	}
	walletKey := crypto.DeterministicKey("fuzz-wallet")
	challenge := []byte("fuzz-challenge")
	leaf, err := BuildLeaf(&walletKey.PublicKey, challenge, goodExt(), rootKey, rootCert, 7)
	if err != nil {
		f.Fatalf("leaf: %v", err)
	}
	// валидная синтетическая цепочка (этап 3 / песочница)
	f.Add(EncodeChain(leaf))
	// граничные: пустой блоб, обрезанный префикс, обрезанный сертификат,
	// много «сертификатов» по одному байту (предел длины цепочки)
	f.Add([]byte(nil))
	f.Add([]byte{0})
	f.Add([]byte{0, 0, 0, 5, 0x30})
	f.Add(bytes.Repeat([]byte{0, 0, 0, 1, 0x30}, 64))

	now := time.Unix(1_790_000_000, 0).UTC()
	roots := []*x509.Certificate{rootCert}
	f.Fuzz(func(t *testing.T, blob []byte) {
		ch, err := Verify(blob, roots, LevelTEE, 24, now)
		if err != nil && ch != nil {
			t.Fatalf("ошибка с ненулевым результатом: %v", err)
		}
	})
}

func FuzzParseKeymint(f *testing.F) {
	// валидные векторы реального формата: TEE и software-fallback
	f.Add(keymintExt([]byte("challenge"), true))
	f.Add(keymintExt([]byte("challenge"), false))
	// граничные: пустой ввод, обрезанный DER, длинная форма тега
	// без конца, гигантская заявленная длина
	f.Add([]byte(nil))
	f.Add([]byte{0x30})
	f.Add(bytes.Repeat([]byte{0xbf, 0x85, 0x40}, 64))
	f.Add([]byte{0x30, 0x82, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, b []byte) {
		if _, _, err := parseKeymint(b); err != nil {
			return
		}
	})
}
