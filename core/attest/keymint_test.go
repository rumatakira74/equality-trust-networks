// Тесты разбора реального расширения аттестации KeyMint/Keymaster (этап 6,
// вопрос №7): длинные теги [704]/[706], challenge, fallback в Verify.
package attest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"testing"
	"time"

	"github.com/rumatakira74/trust-networks-platform/core/crypto"
)

// derWrap — DER-элемент с идентификатором (включая длинную форму тега).
func derWrap(id []byte, content []byte) []byte {
	var l []byte
	n := len(content)
	if n < 128 {
		l = []byte{byte(n)}
	} else {
		var rev []byte
		for n > 0 {
			rev = append(rev, byte(n&0xff))
			n >>= 8
		}
		l = append([]byte{byte(0x80 | len(rev))}, rev...)
	}
	return append(append(append([]byte{}, id...), l...), content...)
}

// authList — AuthorizationList с [704] RootOfTrust и [706] osPatchLevel
// (длинная форма тегов: 704 = 0x85 0x40, 706 = 0x85 0x42).
func authList(tee bool) []byte {
	rot, err := asn1.Marshal(kmRootOfTrust{
		VerifiedBootKey: []byte("bootkey"), DeviceLocked: true,
		VerifiedBootState: BootVerified, VerifiedBootHash: []byte("hash"),
	})
	if err != nil {
		panic(err)
	}
	patch, err := asn1.Marshal(202609) // сентябрь 2026
	if err != nil {
		panic(err)
	}
	var fields []byte
	if tee {
		fields = append(fields, derWrap([]byte{0xBF, 0x85, 0x40}, rot)...)
		fields = append(fields, derWrap([]byte{0xBF, 0x85, 0x42}, patch)...)
	} else {
		// softwareEnforced: только osPatchLevel (RootOfTrust нет)
		fields = append(fields, derWrap([]byte{0xBF, 0x85, 0x42}, patch)...)
	}
	return derWrap([]byte{0x30}, fields)
}

// keymintExt — DER KeyDescription (реальный формат).
func keymintExt(challenge []byte, tee bool) []byte {
	kd := kmKeyDescription{
		AttestationVersion:       300,
		AttestationSecurityLevel: LevelTEE,
		KeymasterVersion:         200,
		KeymasterSecurityLevel:   LevelTEE,
		AttestationChallenge:     challenge,
		UniqueID:                 []byte("unique"),
	}
	if tee {
		kd.TeeEnforced = asn1.RawValue{FullBytes: authList(true)}
		kd.SoftwareEnforced = asn1.RawValue{FullBytes: authList(false)}
	} else {
		// всё в softwareEnforced — fallback-ветка сканирования
		kd.SoftwareEnforced = asn1.RawValue{FullBytes: authList(true)}
		kd.TeeEnforced = asn1.RawValue{FullBytes: authList(false)}
	}
	b, err := asn1.Marshal(kd)
	if err != nil {
		panic(err)
	}
	return b
}

// TestKeymintParse — разбор реального расширения: уровень, boot, блокировка,
// патч, challenge.
func TestKeymintParse(t *testing.T) {
	challenge := []byte("tnp-challenge-тест")
	ext, ch, err := parseKeymint(keymintExt(challenge, true))
	if err != nil {
		t.Fatalf("parseKeymint: %v", err)
	}
	if string(ch) != string(challenge) {
		t.Fatalf("challenge: %q, ожидается %q", ch, challenge)
	}
	if ext.SecurityLevel != LevelTEE {
		t.Fatalf("уровень: %d, ожидается TEE", ext.SecurityLevel)
	}
	if ext.VerifiedBootState != BootVerified || !ext.DeviceLocked {
		t.Fatalf("boot=%d locked=%v", ext.VerifiedBootState, ext.DeviceLocked)
	}
	if ext.PatchYear != 2026 || ext.PatchMonth != 9 {
		t.Fatalf("патч: %d-%02d", ext.PatchYear, ext.PatchMonth)
	}
}

// TestKeymintSoftwareFallback — RootOfTrust в softwareEnforced (часть
// устройств): сканируется вторая ветка.
func TestKeymintSoftwareFallback(t *testing.T) {
	_, _, err := parseKeymint(keymintExt([]byte("c"), false))
	if err != nil {
		t.Fatalf("softwareEnforced fallback: %v", err)
	}
}

// TestKeymintVerifyChain — полная Verify с реальным форматом: лист с
// KeyMint-расширением, подписанный синтетическим корнем (эмулятор).
func TestKeymintVerifyChain(t *testing.T) {
	rootKey := crypto.DeterministicKey("keymint-root")
	rootCert, _, err := SyntheticRoot("keymint", rootKey)
	if err != nil {
		t.Fatal(err)
	}
	walletKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	challenge := []byte("challenge-узла")
	// лист с реальным KeyMint-расширением, подписанный синтетическим корнем
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(11),
		Subject:      pkix.Name{CommonName: "TNP Keymint Attestation"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(4102444800, 0),
		ExtraExtensions: []pkix.Extension{
			{Id: oidAttestation, Value: keymintExt(challenge, true)},
		},
		BasicConstraintsValid: true,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, tpl, rootCert, &walletKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	chain := EncodeChain(leafDER)
	c, err := Verify(chain, []*x509.Certificate{rootCert}, LevelTEE, 24, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if string(c.Challenge) != string(challenge) {
		t.Fatalf("challenge цепочки: %q", c.Challenge)
	}
	if c.Extension.SecurityLevel != LevelTEE {
		t.Fatalf("уровень: %d", c.Extension.SecurityLevel)
	}
	// патч сентябрь 2026, окно 24 мес. — не просрочен; просроченное окно
	if _, err := Verify(chain, []*x509.Certificate{rootCert}, LevelTEE, 0,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("просроченный патч прошёл")
	}
	// StrongBox требуется — TEE-цепочка не проходит
	if _, err := Verify(chain, []*x509.Certificate{rootCert}, LevelStrongBox, 24,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("TEE прошёл как StrongBox")
	}
}
