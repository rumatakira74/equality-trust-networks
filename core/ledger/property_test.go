// Property/fuzz-тесты по spec/04 4.3: генератор случайных потоков транзакций
// (валидные, невалидные, конфликтные), оракул Apply -> OK/E_*, инварианты
// INV-1..INV-14 после каждого шага (дешёвые) и на границах эпох (FullCheck),
// метаморфный тест: мутация валидной транзакции меняет ровно один код.
package ledger

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/rumatakira74/trust-networks-platform/core/attest"
	"github.com/rumatakira74/trust-networks-platform/core/crypto"
	"github.com/rumatakira74/trust-networks-platform/core/params"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// propEnv — окружение генератора: ключи, адреса, корень аттестации.
type propEnv struct {
	t        *testing.T
	ledger   *Ledger
	clock    *FixedClock
	keys     map[string]*ecdsa.PrivateKey
	addrs    map[string][]byte
	rootKey  *ecdsa.PrivateKey
	rootCert *x509.Certificate
}

func newPropEnv(t *testing.T, seedWallets int) *propEnv {
	gp := params.TestGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	gp.Manifest.Profile.DemurrageRateMicros = 0 // демередж — отдельный тест сеттлемента
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &FixedClock{T: 1_790_000_000, E: 100, P: 16}
	l, err := New(gp, clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	env := &propEnv{
		t: t, ledger: l, clock: clock,
		keys:  make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte),
	}
	env.rootKey = crypto.DeterministicKey("prop-root")
	cert, _, err := attest.SyntheticRoot("prop", env.rootKey)
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	env.rootCert = cert
	l.TrustedRoots = append(l.TrustedRoots, cert)

	// пред-сеяные активные кошельки (эквивалент pre_state векторов)
	for i := 0; i < seedWallets; i++ {
		id := string(rune('A' + i))
		env.registerWalletState(id, wire.StatusActive, 10_000)
	}
	return env
}

// registerWalletState — прямой ввод кошелька в состояние (как pre_state векторов).
func (e *propEnv) registerWalletState(id string, status wire.WalletStatus, balance uint64) {
	priv := crypto.DeterministicKey("prop:" + id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	addr := crypto.DeriveAddress(fp, e.ledger.State.Net.NetworkID)
	w := &Wallet{
		Address:           addr,
		Fingerprint:       fp,
		PublicKey:         &priv.PublicKey,
		Status:            status,
		RegisteredAt:      e.clock.NetTime() - 400*86400,
		ActivatedAt:       e.clock.NetTime() - 300*86400,
		ActivatedPeriod:   5,
		LastClaimedPeriod: 12,
		LastTouchEpoch:    e.clock.Epoch(),
		Liquid:            balance,
		DelayedThreshold:  3000,
		Seq:               1,
	}
	w.TipTxID = make([]byte, 32)
	copy(w.TipTxID, []byte("tip:"+id))
	for i := 0; i < 3; i++ {
		w.VouchesReceived = append(w.VouchesReceived, VouchReceipt{Voucher: e.addr(string(rune('S' + i)))})
	}
	e.keys[id] = priv
	e.addrs[id] = addr
	e.ledger.State.Wallets[string(addr)] = w
	e.ledger.State.FingerprintIndex[string(fp)] = addr
	// пред-сеяный баланс — исторически уже выпущен (INV-8)
	e.ledger.State.TotalMinted += balance
}

func (e *propEnv) addr(id string) []byte {
	if a, ok := e.addrs[id]; ok {
		return a
	}
	priv := crypto.DeterministicKey("prop:" + id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	a := crypto.DeriveAddress(fp, e.ledger.State.Net.NetworkID)
	e.keys[id] = priv
	e.addrs[id] = a
	return a
}

// buildSignedTx — собрать, решить PoW и подписать транзакцию.
func (e *propEnv) buildSignedTx(tx *wire.Transaction, keyID string, hard bool) *wire.Transaction {
	// заполняются только незаданные поля: мутации тестов не затираются
	if len(tx.NetworkID) == 0 {
		tx.NetworkID = e.ledger.State.Net.NetworkID
	}
	if tx.ProtocolVersion == 0 {
		tx.ProtocolVersion = e.ledger.State.Net.ProtocolVersion
	}
	if tx.TimestampClaim == 0 {
		tx.TimestampClaim = e.clock.NetTime()
	}
	net := e.ledger.State.Net
	if hard {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowArgon2id}
		ap := crypto.Argon2Params{MemoryKiB: net.Pow.Argon2MemoryKiB,
			Iterations: net.Pow.Argon2Iterations, Parallelism: net.Pow.Argon2Parallelism}
		for n := uint64(0); ; n++ {
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], n)
			tx.Pow.Nonce = nb[:]
			if crypto.VerifyHard(tx.SigningPreimage(), nb[:], net.Pow.TargetHard, ap) {
				break
			}
		}
	} else {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowLight}
		for n := uint64(0); ; n++ {
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], n)
			tx.Pow.Nonce = nb[:]
			if crypto.VerifyLight(tx.SigningPreimage(), nb[:], net.Pow.TargetLight) {
				break
			}
		}
	}
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(e.keys[keyID], sh[:])
	if err != nil {
		e.t.Fatalf("sign: %v", err)
	}
	tx.Signature = sig
	return tx
}

// transferTx — перевод от from к to.
func (e *propEnv) transferTx(from, to string, seq, amount uint64) *wire.Transaction {
	w := e.ledger.State.Wallet(e.addr(from))
	tx := &wire.Transaction{
		Wallet: e.addr(from),
		Seq:    seq,
		Prev:   append([]byte(nil), w.TipTxID...),
		Type:   wire.Transfer,
		Payload: &wire.TransferPayload{Send: &wire.TransferSend{
			Recipient: e.addr(to), Amount: amount,
		}},
	}
	return e.buildSignedTx(tx, from, false)
}

// mintTx — выпуск периода p.
func (e *propEnv) mintTx(id string, seq, p, amount uint64) *wire.Transaction {
	w := e.ledger.State.Wallet(e.addr(id))
	tx := &wire.Transaction{
		Wallet:  e.addr(id),
		Seq:     seq,
		Prev:    append([]byte(nil), w.TipTxID...),
		Type:    wire.Mint,
		Payload: &wire.MintPayload{Period: p, Amount: amount},
	}
	return e.buildSignedTx(tx, id, true)
}

// TestPropertyRandomStream — случайный поток операций; инварианты после каждого
// шага; на границах эпох (каждые 50 шагов) — FullCheck.
func TestPropertyRandomStream(t *testing.T) {
	env := newPropEnv(t, 6)
	ids := []string{"A", "B", "C", "D", "E", "F"}
	rng := rand.New(rand.NewSource(42))

	seqOf := func(id string) uint64 { return env.ledger.State.Wallet(env.addr(id)).Seq + 1 }

	for step := 0; step < 400; step++ {
		from := ids[rng.Intn(len(ids))]
		to := ids[rng.Intn(len(ids))]
		var res Result
		switch rng.Intn(10) {
		case 0, 1, 2: // перевод
			amount := uint64(rng.Intn(5000))
			res = env.ledger.Apply(env.transferTx(from, to, seqOf(from), amount))
		case 3, 4: // выпуск
			p := uint64(13 + rng.Intn(3))
			res = env.ledger.Apply(env.mintTx(from, seqOf(from), p, 1000))
		case 5: // конфликтная ветка (эквивокация): тот же seq дважды
			seq := seqOf(from)
			t1 := env.transferTx(from, to, seq, 100)
			res = env.ledger.Apply(t1)
			if res.Outcome == Accepted {
				t2 := env.transferTx(from, ids[(rng.Intn(len(ids)))], seq, 200)
				res2 := env.ledger.Apply(t2)
				if res2.Outcome != Conflict {
					t.Fatalf("шаг %d: конфликтная ветка: %v %s", step, res2.Outcome, res2.Code)
				}
			}
		case 6: // отложенный перевод выше P -> pending
			res = env.ledger.Apply(env.transferTx(from, to, seqOf(from), 9000))
		case 7: // guardian_offer
			w := env.ledger.State.Wallet(env.addr(from))
			tx := &wire.Transaction{
				Wallet:  env.addr(from),
				Seq:     seqOf(from),
				Prev:    append([]byte(nil), w.TipTxID...),
				Type:    wire.GuardianOffer,
				Payload: &wire.GuardianOfferPayload{},
			}
			res = env.ledger.Apply(env.buildSignedTx(tx, from, false))
		case 8: // невалидная мутация: чужая подпись
			tx := env.transferTx(from, to, seqOf(from), 10)
			tx.Signature = nil
			res = env.ledger.Apply(tx)
			if res.Code != "E_SIG" {
				t.Fatalf("шаг %d: ожидается E_SIG, получен %s", step, res.Code)
			}
		default: // невалидный seq
			tx := env.transferTx(from, to, seqOf(from)+5, 10)
			res = env.ledger.Apply(tx)
			if res.Code != "E_SEQ" {
				t.Fatalf("шаг %d: ожидается E_SEQ, получен %s", step, res.Code)
			}
		}
		_ = res
		if bad := env.ledger.CheckInvariants(); len(bad) > 0 {
			t.Fatalf("шаг %d: инварианты нарушены: %v", step, bad)
		}
		// раз в 50 шагов — граница эпохи: FullCheck
		if step%50 == 49 {
			env.clock.E++
			env.clock.P++
			env.ledger.syncTimeForTest()
			if bad := env.ledger.FullCheck(); len(bad) > 0 {
				t.Fatalf("шаг %d (граница эпохи): FullCheck: %v", step, bad)
			}
		}
	}
}

// TestPropertyMetamorphic — мутация валидной транзакции меняет ровно один код.
func TestPropertyMetamorphic(t *testing.T) {
	env := newPropEnv(t, 3)
	base := env.transferTx("A", "B", 2, 100)
	if res := env.ledger.Apply(base); res.Outcome != Accepted {
		t.Fatalf("база не принята: %s", res.Code)
	}

	// Мутации применяются к НЕПОДПИСАННОЙ транзакции, затем PoW и подпись
	// пересобираются: отказ обязан приходить ровно от мутированной проверки.
	mutations := []struct {
		name string
		mut  func(tx *wire.Transaction)
		want string
	}{
		{"чужая сеть", func(tx *wire.Transaction) { tx.NetworkID = []byte("other") }, "E_NET"},
		{"чужая версия", func(tx *wire.Transaction) { tx.ProtocolVersion = 99 }, "E_VER"},
		{"нет pow", func(tx *wire.Transaction) { tx.Pow = nil }, "E_POW_LIGHT"},
		{"неверный тип", func(tx *wire.Transaction) { tx.Type = wire.Vouch }, "E_TYPE"},
		{"вне допуска времени", func(tx *wire.Transaction) { tx.TimestampClaim += 7200 }, "E_TIME"},
		{"битый prev", func(tx *wire.Transaction) { tx.Prev[0] ^= 0xFF }, "E_PREV"},
	}
	for _, m := range mutations {
		w := env.ledger.State.Wallet(env.addr("A"))
		tx := &wire.Transaction{
			Wallet:  env.addr("A"),
			Seq:     w.Seq + 1,
			Prev:    append([]byte(nil), w.TipTxID...),
			Type:    wire.Transfer,
			Payload: &wire.TransferPayload{Send: &wire.TransferSend{Recipient: env.addr("B"), Amount: 100}},
		}
		m.mut(tx)
		if m.name != "нет pow" {
			env.buildSignedTx(tx, "A", false)
		} else {
			tx.NetworkID = env.ledger.State.Net.NetworkID
			tx.TimestampClaim = env.clock.NetTime()
			tx.ProtocolVersion = env.ledger.State.Net.ProtocolVersion
		}
		res := env.ledger.Apply(tx)
		if res.Outcome != Rejected || res.Code != m.want {
			t.Errorf("мутация %q: получен %v/%s, ожидается reject/%s",
				m.name, res.Outcome, res.Code, m.want)
		}
	}
	// битая подпись: валидная сборка, затем порча подписи
	{
		w := env.ledger.State.Wallet(env.addr("A"))
		tx := env.transferTx("A", "B", w.Seq+1, 100)
		tx.Signature[3] ^= 0xFF
		res := env.ledger.Apply(tx)
		if res.Code != "E_SIG" {
			t.Errorf("мутация подписи: получен %s, ожидается E_SIG", res.Code)
		}
	}
}

// syncTimeForTest — доступ к syncTime из тестов того же пакета.
func (l *Ledger) syncTimeForTest() { l.syncTime() }
