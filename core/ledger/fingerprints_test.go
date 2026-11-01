// Тесты fingerprint-манифеста (v9.11, мини-этап 12.2): манифест хранит
// публичные ключи посевного круга и fingerprint'ы поставщиков, адреса
// нигде не хранятся — выводятся из финального network_id; членство
// посевного круга и поставщика — по fingerprint, принадлежность
// fingerprint'а заявителю — подпись (C8).
package ledger

import (
	"bytes"
	"crypto/ecdsa"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// newFingerprintEnv — genesis с посевным кругом из публичных ключей
// (адресов в манифесте нет — поле удалено из wire) и поставщиком-
// fingerprint; та же деривация ключей, что в e2e (e.key/e.addr).
func newFingerprintEnv(t *testing.T, networkID []byte) *e2eEnv {
	t.Helper()
	gp := params.TestGenesis()
	gp.Manifest.NetworkID = networkID
	gp.Genesis.NetworkID = networkID
	prof := gp.Manifest.Profile
	prof.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	prof.FreshTransferWindowPeriods = 0
	prof.Modules = []uint32{wire.ModuleRedeem}
	prof.ResourceID = "resource"
	prof.FeeMode = wire.FeeModeNone
	// посевной круг — публичные ключи (fingerprint-манифест v9.11)
	var circle []wire.SeedMember
	for _, id := range []string{"S1", "S2", "S3"} {
		priv := crypto.DeterministicKey("e2e:" + id)
		circle = append(circle, wire.SeedMember{
			PublicKey: crypto.MarshalPubKey(&priv.PublicKey),
		})
	}
	gp.Manifest.SeedCircle = circle
	// поставщик — fingerprint (не обязан быть посевным участником)
	provPub := crypto.MarshalPubKey(&crypto.DeterministicKey("e2e:PROV").PublicKey)
	prof.Providers = [][]byte{crypto.Fingerprint(provPub)}
	wire.SetManifestHash(gp)
	clock := &FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := New(gp, clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	return &e2eEnv{t: t, ledger: l, clock: clock,
		keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
}

// TestSeedAddressesDerivedFromNetworkID — адреса посевных выводятся из
// финального network_id (в манифесте только публичные ключи): кошельки
// создаются по деривации SHA-256(fingerprint ‖ network_id), fingerprint
// совпадает с выведенным из ключа манифеста, индекс фингерпринтов
// указывает на выведенный адрес; в другой сети тот же ключ даёт другой
// адрес (привязка к сети).
func TestSeedAddressesDerivedFromNetworkID(t *testing.T) {
	e := newFingerprintEnv(t, []byte("fp-manifest-a"))
	for _, id := range []string{"S1", "S2", "S3"} {
		w := e.ledger.State.Wallet(e.addr(id))
		if w == nil {
			t.Fatalf("кошелёк %s не создан по выведенному адресу", id)
		}
		if !w.Seed || w.Status != wire.StatusActive {
			t.Fatalf("кошелёк %s: seed=%v status=%v", id, w.Seed, w.Status)
		}
		pub := crypto.MarshalPubKey(&e.key(id).PublicKey)
		if !bytes.Equal(w.Fingerprint, crypto.Fingerprint(pub)) {
			t.Fatalf("кошелёк %s: fingerprint не выведен из ключа манифеста", id)
		}
		if !bytes.Equal(e.ledger.State.FingerprintIndex[string(w.Fingerprint)], w.Address) {
			t.Fatalf("кошелёк %s: индекс фингерпринтов не указывает на выведенный адрес", id)
		}
	}
	// та же деривация, другой network_id — другой адрес (привязка к сети)
	e2 := newFingerprintEnv(t, []byte("fp-manifest-b"))
	if bytes.Equal(e.addr("S1"), e2.addr("S1")) {
		t.Fatalf("адрес S1 одинаков в разных сетях — деривация не зависит от network_id")
	}
}

// TestSeedMembershipByFingerprintSignature — принадлежность fingerprint'а
// заявителю — подпись: транзакция посевного кошелька с подписью ключа из
// манифеста принимается (C8), подпись чужим ключом — E_SIG.
func TestSeedMembershipByFingerprintSignature(t *testing.T) {
	e := newFingerprintEnv(t, []byte("fp-manifest-sig"))
	e.clock.P = 2 // период 1 завершён (заявляются завершённые, №24)
	// выпуск посевным кошельком: подпись ключом из манифеста — accept
	mint := e.tx("S1", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true)
	if res := e.ledger.Apply(mint); res.Outcome != Accepted {
		t.Fatalf("mint посевного с подписью по fingerprint: %v/%s", res.Outcome, res.Code)
	}
	// та же транзакция от имени S3, подписана ключом S1 — E_SIG
	w := e.ledger.State.Wallet(e.addr("S3"))
	tx := &wire.Transaction{
		NetworkID:       e.ledger.State.Net.NetworkID,
		Wallet:          e.addr("S3"),
		Seq:             w.Seq + 1,
		Prev:            append([]byte(nil), w.TipTxID...),
		Type:            wire.Mint,
		Payload:         &wire.MintPayload{Period: 1, Amount: 1000},
		ProtocolVersion: e.ledger.State.Net.ProtocolVersion,
		TimestampClaim:  e.clock.NetTime(),
	}
	e.solveAndSign(tx, "S3", true)
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(e.key("S1"), sh[:])
	if err != nil {
		t.Fatal(err)
	}
	tx.Signature = sig
	if res := e.ledger.Apply(tx); res.Outcome != Rejected || res.Code != "E_SIG" {
		t.Fatalf("подпись чужим ключом: %v/%s, ожидается reject/E_SIG", res.Outcome, res.Code)
	}
}

// TestProviderMembershipByFingerprint — членство поставщика по
// fingerprint: redeem на выведенный адрес поставщика принимается,
// на адрес не-поставщика — E_REDEEM_PROVIDER; restricted-перевод —
// только поставщику (адрес выводится, кошелёк не нужен для проверки).
func TestProviderMembershipByFingerprint(t *testing.T) {
	e := newFingerprintEnv(t, []byte("fp-manifest-prov"))
	// участник с квотой и кошелёк поставщика (погашение кредитует
	// существующий кошелёк — иначе INV-8: единицы исчезают)
	e.preStateWallet("ALICE")
	e.preStateWallet("PROV")
	e.clock.P = 12
	mint := e.tx("ALICE", wire.Mint, &wire.MintPayload{Period: 11, Amount: 1000}, true)
	e.requireAccepted(e.ledger.Apply(mint), "mint ALICE")
	// redeem на выведенный адрес поставщика — accept (членство по
	// fingerprint, кошелёк поставщика не обязан существовать)
	provAddr := crypto.AddressOfKey(
		crypto.MarshalPubKey(&crypto.DeterministicKey("e2e:PROV").PublicKey),
		e.ledger.State.Net.NetworkID)
	redeem := e.tx("ALICE", wire.Redeem,
		&wire.RedeemPayload{Provider: provAddr, ResourceID: "resource", Amount: 100}, false)
	e.requireAccepted(e.ledger.Apply(redeem), "redeem поставщику")
	// redeem на адрес не-поставщика — E_REDEEM_PROVIDER
	other := e.addr("BOB")
	redeem = e.tx("ALICE", wire.Redeem,
		&wire.RedeemPayload{Provider: other, ResourceID: "resource", Amount: 100}, false)
	if res := e.ledger.Apply(redeem); res.Outcome != Rejected || res.Code != "E_REDEEM_PROVIDER" {
		t.Fatalf("redeem не-поставщику: %v/%s, ожидается reject/E_REDEEM_PROVIDER",
			res.Outcome, res.Code)
	}
}
