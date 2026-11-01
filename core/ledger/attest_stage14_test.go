// Тесты этапа 14 (аттестация): A1 — привязка SPKI листа к регистрируемому
// ключу (E_ATTEST_KEY_MISMATCH); срок цепочки по timestamp_claim (листы
// короткоживущие, replay даёт тот же вердикт, что live); A12 — нулевой
// PoW-таргет вне песочницы отклоняется genesis-валидацией.
package ledger

import (
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// registerWithChain — register с явной цепочкой и регистрируемым ключом
// (расцеплены намеренно: тесты A1 подменяют один из них).
func (e *e2eEnv) registerWithChain(id string, pub []byte, chain []byte) Result {
	challenge := attest.ChallengeFormula(pub, crypto.ProtocolID, e.ledger.State.Net.NetworkID)
	rp := &wire.RegisterPayload{
		AttestedPublicKey:    pub,
		AttestationChain:     chain,
		AttestationChallenge: challenge,
	}
	tx := &wire.Transaction{
		NetworkID:       e.ledger.State.Net.NetworkID,
		Wallet:          e.addr(id),
		Seq:             1,
		Type:            wire.Register,
		Payload:         rp,
		ProtocolVersion: e.ledger.State.Net.ProtocolVersion,
		TimestampClaim:  e.clock.NetTime(),
	}
	e.solveAndSign(tx, id, true)
	return e.ledger.Apply(tx)
}

// A1, негативный: валидная цепочка (до корня, challenge сходится) выдана
// для ключа X, а регистрируется ключ Y — до фикса проходило R1–R7,
// фингерпринт и адрес считались бы от неподтверждённого ключа.
func TestRegisterLeafKeyMismatch(t *testing.T) {
	e := newE2E(t)
	attested := e.key("attested") // ключ, для которого выдана цепочка
	other := e.key("other")       // регистрируемый ключ
	otherPub := crypto.MarshalPubKey(&other.PublicKey)
	// challenge — по регистрируемому ключу (R4/R4b сходятся),
	// лист — от attested-ключа: атака из аудита A1
	challenge := attest.ChallengeFormula(otherPub, crypto.ProtocolID,
		e.ledger.State.Net.NetworkID)
	leaf, err := attest.BuildLeaf(&attested.PublicKey, challenge,
		attest.Extension{SecurityLevel: attest.LevelTEE,
			VerifiedBootState: attest.BootVerified, DeviceLocked: true,
			PatchYear: 2023, PatchMonth: 6},
		e.rootKey, e.rootCert, 43)
	if err != nil {
		t.Fatalf("leaf: %v", err)
	}
	res := e.registerWithChain("other", otherPub, attest.EncodeChain(leaf))
	if res.Code != "E_ATTEST_KEY_MISMATCH" {
		t.Fatalf("ожидался отказ E_ATTEST_KEY_MISMATCH, получено %q (%v)",
			res.Code, res.Outcome)
	}
}

// A1, положительный: цепочка выдана для регистрируемого ключа — принимается.
func TestRegisterLeafKeyMatchAccepted(t *testing.T) {
	e := newE2E(t)
	priv := e.key("honest")
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	challenge := attest.ChallengeFormula(pub, crypto.ProtocolID,
		e.ledger.State.Net.NetworkID)
	leaf, err := attest.BuildLeaf(&priv.PublicKey, challenge,
		attest.Extension{SecurityLevel: attest.LevelTEE,
			VerifiedBootState: attest.BootVerified, DeviceLocked: true,
			PatchYear: 2023, PatchMonth: 6},
		e.rootKey, e.rootCert, 44)
	if err != nil {
		t.Fatalf("leaf: %v", err)
	}
	res := e.registerWithChain("honest", pub, attest.EncodeChain(leaf))
	if res.Outcome != Accepted {
		t.Fatalf("регистрация с совпадающим листом отклонена: %q", res.Code)
	}
}

// Срок цепочки (этап 14): валидность листа — на момент timestamp_claim,
// не на часы узла. Лист просрочен/ещё не действует на момент claim —
// reject; replay и live дают одинаковый вердикт.
func TestRegisterLeafValidityAtTimestampClaim(t *testing.T) {
	claim := time.Unix(int64(1_700_000_000), 0).UTC()
	cases := []struct {
		name       string
		notBefore  time.Time
		notAfter   time.Time
		wantReject bool
	}{
		{"лист просрочен на момент claim", claim.Add(-24 * time.Hour), claim.Add(-time.Hour), true},
		{"лист ещё не действует на момент claim", claim.Add(time.Hour), claim.Add(24 * time.Hour), true},
		{"лист действует на момент claim", claim.Add(-time.Hour), claim.Add(time.Hour), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newE2E(t)
			priv := e.key("leaf-validity")
			pub := crypto.MarshalPubKey(&priv.PublicKey)
			challenge := attest.ChallengeFormula(pub, crypto.ProtocolID,
				e.ledger.State.Net.NetworkID)
			leaf, err := attest.BuildLeafValidity(&priv.PublicKey, challenge,
				attest.Extension{SecurityLevel: attest.LevelTEE,
					VerifiedBootState: attest.BootVerified, DeviceLocked: true,
					PatchYear: 2023, PatchMonth: 6},
				e.rootKey, e.rootCert, 45, c.notBefore, c.notAfter)
			if err != nil {
				t.Fatalf("leaf: %v", err)
			}
			res := e.registerWithChain("leaf-validity", pub, attest.EncodeChain(leaf))
			if c.wantReject && res.Outcome != Rejected {
				t.Fatalf("ожидался отказ (строгая политика, E_ATTEST), получено %q", res.Code)
			}
			if !c.wantReject && res.Outcome != Accepted {
				t.Fatalf("регистрация отклонена: %q", res.Code)
			}
		})
	}
}

// A12: не-sandbox genesis с нулевым PoW-таргетом (или без параметров PoW)
// — невалиден; sandbox — допустимо (симуляции).
func TestGenesisZeroPowTarget(t *testing.T) {
	mk := func(mutate func(p *wire.ProfileParams)) error {
		gp := params.TestGenesis()
		mutate(gp.Manifest.Profile)
		wire.SetManifestHash(gp)
		_, err := New(gp, &FixedClock{T: 1_700_000_000, E: 1, P: 1}, Options{})
		return err
	}
	if err := mk(func(p *wire.ProfileParams) { p.Pow.TargetLight = 0 }); err == nil {
		t.Error("не-sandbox genesis с target_light = 0 принят")
	}
	if err := mk(func(p *wire.ProfileParams) { p.Pow.TargetHard = 0 }); err == nil {
		t.Error("не-sandbox genesis с target_hard = 0 принят")
	}
	if err := mk(func(p *wire.ProfileParams) { p.Pow = nil }); err == nil {
		t.Error("не-sandbox genesis без параметров PoW принят")
	}
	if err := mk(func(p *wire.ProfileParams) {
		p.Sandbox = true
		p.Pow.TargetLight = 0
		p.Pow.TargetHard = 0
	}); err != nil {
		t.Errorf("sandbox genesis с нулевыми таргетами отклонён: %v", err)
	}
}
