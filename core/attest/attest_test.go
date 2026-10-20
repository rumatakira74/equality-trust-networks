// Тесты аттестации: синтетические цепочки (этап 3), тампер — отклонение.
package attest

import (
	"bytes"
	"crypto/x509"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
)

func goodExt() Extension {
	return Extension{SecurityLevel: LevelTEE, VerifiedBootState: BootVerified,
		DeviceLocked: true, PatchYear: 2026, PatchMonth: 6}
}

func TestAttestValidChain(t *testing.T) {
	rootKey := crypto.DeterministicKey("root")
	rootCert, _, err := SyntheticRoot("test", rootKey)
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	walletKey := crypto.DeterministicKey("wallet")
	pub := crypto.MarshalPubKey(&walletKey.PublicKey)
	challenge := ChallengeFormula(pub, crypto.ProtocolID, []byte("net"))
	leaf, err := BuildLeaf(&walletKey.PublicKey, challenge, goodExt(), rootKey, rootCert, 7)
	if err != nil {
		t.Fatalf("leaf: %v", err)
	}
	chain := EncodeChain(leaf)
	now := time.Unix(1_790_000_000, 0).UTC()
	if _, err := Verify(chain, []*x509.Certificate{rootCert}, LevelTEE, 24, now); err != nil {
		t.Fatalf("валидная цепочка отклонена: %v", err)
	}
}

func TestAttestTampered(t *testing.T) {
	rootKey := crypto.DeterministicKey("root")
	rootCert, _, _ := SyntheticRoot("test", rootKey)
	walletKey := crypto.DeterministicKey("wallet")
	pub := crypto.MarshalPubKey(&walletKey.PublicKey)
	challenge := ChallengeFormula(pub, crypto.ProtocolID, []byte("net"))
	now := time.Unix(1_790_000_000, 0).UTC()

	cases := []struct {
		name string
		ext  Extension
		want string
	}{
		{"device unlocked", Extension{LevelTEE, BootVerified, false, 2026, 6}, "deviceLocked"},
		{"boot not verified", Extension{LevelTEE, BootSelfSigned, true, 2026, 6}, "verifiedBootState"},
		{"software level", Extension{LevelSoftware, BootVerified, true, 2026, 6}, "уровень"},
		{"stale patch", Extension{LevelTEE, BootVerified, true, 2023, 1}, "патча"},
		// №43 (минор-фикс, этап 9): дата патча в будущем — отклоняется
		{"future patch", Extension{LevelTEE, BootVerified, true, 2030, 1}, "будущем"},
	}
	for _, c := range cases {
		leaf, err := BuildLeaf(&walletKey.PublicKey, challenge, c.ext, rootKey, rootCert, 7)
		if err != nil {
			t.Fatalf("leaf: %v", err)
		}
		_, err = Verify(EncodeChain(leaf), []*x509.Certificate{rootCert}, LevelTEE, 24, now)
		if err == nil {
			t.Errorf("%s: тампер прошёл", c.name)
		}
	}
}

func TestAttestChallengeFormula(t *testing.T) {
	// PENDING: вопрос №23 — параметризуемая формула; дефолт связывает
	// ключ, протокол и сеть
	pub := []byte("pubkey")
	a := ChallengeFormula(pub, crypto.ProtocolID, []byte("net1"))
	b := ChallengeFormula(pub, crypto.ProtocolID, []byte("net2"))
	if bytes.Equal(a, b) {
		t.Fatal("challenge не зависит от сети")
	}
}
