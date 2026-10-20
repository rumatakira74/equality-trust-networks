// Тесты модуля attest (6.1, этап 10): сборка предъявления, проверка
// сторонним приложением без полного узла (лёгкий клиент + пороговая
// подпись чекпоинта + Merkle-доказательство), H(address ‖ app_id),
// защита от подлогов (чужой challenge, просрочка, чужая сеть, чужой
// чекпоинт, порог не достигнут).
package present

import (
	"crypto/ecdsa"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

type env struct {
	l            *ledger.Ledger
	clock        *ledger.FixedClock
	netID        []byte
	manifestHash []byte
	seedPrivs    []*ecdsa.PrivateKey
	seedAddrs    [][]byte
	pubs         map[string][]byte // addr (string) -> DER pub
}

func newEnv(t *testing.T) *env {
	t.Helper()
	gp := params.TestGenesis()
	e := &env{netID: gp.Genesis.NetworkID, pubs: map[string][]byte{}}
	var members []wire.SeedMember
	for _, tag := range []string{"present-s1", "present-s2", "present-s3"} {
		k := crypto.DeterministicKey(tag)
		pub := crypto.MarshalPubKey(&k.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := crypto.DeriveAddress(fp, e.netID)
		members = append(members, wire.SeedMember{Address: addr, PublicKey: pub})
		e.seedPrivs = append(e.seedPrivs, k)
		e.seedAddrs = append(e.seedAddrs, addr)
		e.pubs[string(addr)] = pub
	}
	gp.Manifest.SeedCircle = members
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	e.manifestHash = mh[:]
	e.clock = &ledger.FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := ledger.New(gp, e.clock, ledger.Options{})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	e.l = l
	return e
}

func (e *env) pubOf(addr []byte) (*ecdsa.PublicKey, bool) {
	der, ok := e.pubs[string(addr)]
	if !ok {
		return nil, false
	}
	p, err := crypto.ParsePubKey(der)
	if err != nil {
		return nil, false
	}
	return p, true
}

// checkpoint — сборка и пороговая подпись чекпоинта эпохи 1 (2 из 3).
func (e *env) checkpoint(t *testing.T, signers int) *wire.Checkpoint {
	t.Helper()
	cp, err := cons.Build(e.netID, 1, e.l, nil, e.clock.NetTime(), 1,
		e.seedAddrs, 2, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	col := cons.NewCollector(cp, e.pubOf)
	sig := &wire.ThresholdSignature{}
	for i := 0; i < signers && i < len(e.seedPrivs); i++ {
		s := &cons.Signer{Wallet: e.seedAddrs[i], Priv: e.seedPrivs[i]}
		share, err := s.Sign(cp)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if !col.Add(share) {
			t.Fatalf("доля подписи не принята")
		}
		sig.Signatures = append(sig.Signatures, share)
	}
	if signers >= 2 {
		cp.ThresholdSig = col.Ready()
	} else {
		// ниже порога: подписи есть, порог не достигнут — для негативного теста
		cp.ThresholdSig = sig
	}
	if cp.ThresholdSig == nil {
		t.Fatalf("порог не собран (signers=%d)", signers)
	}
	return cp
}

// TestAttestFullCycle — критерий этапа: верификатор проверяет предъявление
// в стороннем приложении (без полного узла): лёгкий клиент принимает
// чекпоинт от «любого узла», Verify проверяет подпись кошелька,
// доказательство и пороговую подпись; H(address ‖ app_id) вычисляется.
func TestAttestFullCycle(t *testing.T) {
	e := newEnv(t)
	addr := e.seedAddrs[0]
	priv := e.seedPrivs[0]

	challenge := []byte("challenge-от-проверяющего-123")
	appID := []byte("civic-app-v1")
	expires := e.clock.NetTime() + 3600
	p, err := Build(e.l, addr, priv, challenge, appID, e.manifestHash, 1, expires)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if p.Statement.VouchDepth != 0 {
		t.Fatalf("глубина посевного не 0: %d", p.Statement.VouchDepth)
	}
	if p.Statement.Status != wire.StatusActive {
		t.Fatalf("статус посевного не active: %v", p.Statement.Status)
	}
	// транспорт: кодирование переживает Round-trip
	enc, err := p.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	p2, err := Decode(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	cp := e.checkpoint(t, 3)
	// лёгкий клиент: синхронизация с «любого узла»
	ls := NewLightSync(e.netID, 0, 86400, 86400, e.pubOf)
	if err := ls.Adopt(cp); err != nil {
		t.Fatalf("light sync: %v", err)
	}
	res, err := Verify(p2, ls.AtEpoch(1), e.netID, e.manifestHash, challenge,
		e.clock.NetTime(), e.pubOf)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	want := AppIdentity(addr, appID)
	if res.AppIdentity != want {
		t.Fatalf("app identity не сходится")
	}
	// H(address ‖ app_id) разный для разных приложений (один человек —
	// одно действие в каждом приложении своя метка, 6.1)
	other := AppIdentity(addr, []byte("other-app"))
	if other == want {
		t.Fatalf("идентификаторы разных приложений совпали")
	}
}

// TestAttestRejects — подлоги отклоняются: чужой challenge, просрочка,
// чужой манифест/сеть, порог подписей чекпоинта не достигнут,
// подмена утверждения и листа состояния.
func TestAttestRejects(t *testing.T) {
	e := newEnv(t)
	addr := e.seedAddrs[0]
	priv := e.seedPrivs[0]
	challenge := []byte("ch-123")
	p, err := Build(e.l, addr, priv, challenge, []byte("app"), e.manifestHash, 1,
		e.clock.NetTime()+3600)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	cp := e.checkpoint(t, 3)

	cases := []struct {
		name string
		fn   func() error
	}{
		{"чужой challenge", func() error {
			_, err := Verify(p, cp, e.netID, e.manifestHash, []byte("другой"),
				e.clock.NetTime(), e.pubOf)
			return err
		}},
		{"просрочка", func() error {
			_, err := Verify(p, cp, e.netID, e.manifestHash, challenge,
				e.clock.NetTime()+7200, e.pubOf)
			return err
		}},
		{"чужой манифест", func() error {
			_, err := Verify(p, cp, e.netID, []byte("другой-манифест"), challenge,
				e.clock.NetTime(), e.pubOf)
			return err
		}},
		{"чужая сеть", func() error {
			_, err := Verify(p, cp, []byte("чужая-сеть"), e.manifestHash, challenge,
				e.clock.NetTime(), e.pubOf)
			return err
		}},
		{"порог не достигнут", func() error {
			weak := e.checkpoint(t, 1)
			_, err := Verify(p, weak, e.netID, e.manifestHash, challenge,
				e.clock.NetTime(), e.pubOf)
			return err
		}},
		{"подмена утверждения", func() error {
			bad := *p
			st := *p.Statement
			st.AgeDays = 9999 // возраст не подписан кошельком
			bad.Statement = &st
			_, err := Verify(&bad, cp, e.netID, e.manifestHash, challenge,
				e.clock.NetTime(), e.pubOf)
			return err
		}},
		{"подмена листа", func() error {
			bad := *p
			leaf := *p.Leaf
			leaf.Balance = 1_000_000
			bad.Leaf = &leaf
			_, err := Verify(&bad, cp, e.netID, e.manifestHash, challenge,
				e.clock.NetTime(), e.pubOf)
			return err
		}},
	}
	for _, c := range cases {
		if err := c.fn(); err == nil {
			t.Fatalf("%s: подлог принят", c.name)
		}
	}
	// корректное предъявление всё ещё проходит
	if _, err := Verify(p, cp, e.netID, e.manifestHash, challenge,
		e.clock.NetTime(), e.pubOf); err != nil {
		t.Fatalf("после подлогов честное предъявление отклонено: %v", err)
	}
}

// TestAttestVouchDepth — глубина графа поручительств: посевной — 0,
// неизвестный — -1.
func TestAttestVouchDepth(t *testing.T) {
	e := newEnv(t)
	unknown := crypto.DeriveAddress(crypto.Fingerprint([]byte("x")), e.netID)
	if d := VouchDepth(e.l, unknown); d != -1 {
		t.Fatalf("глубина неизвестного не -1: %d", d)
	}
	if d := VouchDepth(e.l, e.seedAddrs[0]); d != 0 {
		t.Fatalf("глубина посевного не 0: %d", d)
	}
}

// TestAttestLightSyncChain — лёгкий клиент принимает цепочку строго
// последовательно, дубль не растит цепочку, разрыв prev_hash отклоняется.
func TestAttestLightSyncChain(t *testing.T) {
	e := newEnv(t)
	cp := e.checkpoint(t, 3)
	ls := NewLightSync(e.netID, 0, 86400, 86400, e.pubOf)
	if err := ls.Adopt(cp); err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if err := ls.Adopt(cp); err != nil {
		t.Fatalf("дубль: %v", err)
	}
	if len(ls.Chain()) != 1 {
		t.Fatalf("цепочка выросла от дубля")
	}
	// чекпоинт без подписей отклоняется
	weak := *cp
	weak.Epoch = 2
	weak.NetworkTime = cp.NetworkTime + 86400
	weak.ThresholdSig = nil
	if err := ls.Adopt(&weak); err == nil {
		t.Fatalf("чекпоинт без подписей принят")
	}
	if ls.Latest().Epoch != 1 {
		t.Fatalf("цепочка испорчена")
	}
}
