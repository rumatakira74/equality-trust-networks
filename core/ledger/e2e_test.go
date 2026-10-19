// Сквозной headless-сценарий (этап 3, п. 3.5.3): register -> vouch x3 ->
// mint -> transfer -> отложенный перевод > P -> исполнение по сроку ->
// эквивокация + evidence -> migrate с гардианами. INV-8 (сохранение
// предложения) проверяется на каждом шаге. Replay из SQLite — INV-3.
package ledger

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/binary"
	"path/filepath"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/store"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

type e2eEnv struct {
	t        *testing.T
	ledger   *Ledger
	clock    *FixedClock
	keys     map[string]*ecdsa.PrivateKey
	addrs    map[string][]byte
	rootKey  *ecdsa.PrivateKey
	rootCert *x509.Certificate
}

func newE2E(t *testing.T) *e2eEnv {
	gp := params.TestGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := New(gp, clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	env := &e2eEnv{
		t: t, ledger: l, clock: clock,
		keys:  make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte),
	}
	env.rootKey = crypto.DeterministicKey("e2e-root")
	cert, _, err := attest.SyntheticRoot("e2e", env.rootKey)
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	env.rootCert = cert
	l.TrustedRoots = append(l.TrustedRoots, cert)
	return env
}

func (e *e2eEnv) key(id string) *ecdsa.PrivateKey {
	if k, ok := e.keys[id]; ok {
		return k
	}
	k := crypto.DeterministicKey("e2e:" + id)
	e.keys[id] = k
	return k
}

func (e *e2eEnv) addr(id string) []byte {
	if a, ok := e.addrs[id]; ok {
		return a
	}
	pub := crypto.MarshalPubKey(&e.key(id).PublicKey)
	fp := crypto.Fingerprint(pub)
	a := crypto.DeriveAddress(fp, e.ledger.State.Net.NetworkID)
	e.addrs[id] = a
	return a
}

// register — реальная REGISTER через ядро (аттестация + усиленный PoW).
func (e *e2eEnv) register(id string, guardians ...string) Result {
	priv := e.key(id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	challenge := attest.ChallengeFormula(pub, crypto.ProtocolID, e.ledger.State.Net.NetworkID)
	ext := attest.Extension{
		SecurityLevel: attest.LevelTEE, VerifiedBootState: attest.BootVerified,
		DeviceLocked: true, PatchYear: 2026, PatchMonth: 10,
	}
	leaf, err := attest.BuildLeaf(&priv.PublicKey, challenge, ext, e.rootKey, e.rootCert, 42)
	if err != nil {
		e.t.Fatalf("leaf: %v", err)
	}
	rp := &wire.RegisterPayload{
		AttestedPublicKey:    pub,
		AttestationChain:     attest.EncodeChain(leaf),
		AttestationChallenge: challenge,
	}
	for _, g := range guardians {
		rp.PersonalGuardians = append(rp.PersonalGuardians, e.addr(g))
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

// tx — сборка транзакции с корректным seq/prev, PoW и подписью.
func (e *e2eEnv) tx(id string, typ wire.TxType, payload wire.Payload, hard bool) *wire.Transaction {
	w := e.ledger.State.Wallet(e.addr(id))
	tx := &wire.Transaction{
		NetworkID:       e.ledger.State.Net.NetworkID,
		Wallet:          e.addr(id),
		Seq:             w.Seq + 1,
		Prev:            append([]byte(nil), w.TipTxID...),
		Type:            typ,
		Payload:         payload,
		ProtocolVersion: e.ledger.State.Net.ProtocolVersion,
		TimestampClaim:  e.clock.NetTime(),
	}
	e.solveAndSign(tx, id, hard)
	return tx
}

func (e *e2eEnv) solveAndSign(tx *wire.Transaction, id string, hard bool) {
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
	sig, err := crypto.SignDER(e.key(id), sh[:])
	if err != nil {
		e.t.Fatalf("sign: %v", err)
	}
	tx.Signature = sig
}

func (e *e2eEnv) requireAccepted(res Result, what string) {
	e.t.Helper()
	if res.Outcome != Accepted {
		e.t.Fatalf("%s: %v/%s", what, res.Outcome, res.Code)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		e.t.Fatalf("%s: инварианты: %v", what, bad)
	}
}

// TestRegisterChallengeBinding — R4b (этап 6): challenge внутри цепочки
// аттестации обязан совпадать с challenge регистрации — переиспользованная
// цепочка (выданная под другой challenge) отклоняется.
func TestRegisterChallengeBinding(t *testing.T) {
	e := newE2E(t)
	priv := e.key("CH")
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	right := attest.ChallengeFormula(pub, crypto.ProtocolID, e.ledger.State.Net.NetworkID)
	wrong := attest.ChallengeFormula(pub, crypto.ProtocolID, []byte("другая сеть"))
	ext := attest.Extension{
		SecurityLevel: attest.LevelTEE, VerifiedBootState: attest.BootVerified,
		DeviceLocked: true, PatchYear: 2026, PatchMonth: 10,
	}
	leaf, err := attest.BuildLeaf(&priv.PublicKey, wrong, ext, e.rootKey, e.rootCert, 43)
	if err != nil {
		t.Fatalf("leaf: %v", err)
	}
	rp := &wire.RegisterPayload{
		AttestedPublicKey:    pub,
		AttestationChain:     attest.EncodeChain(leaf),
		AttestationChallenge: right, // формула верна, но цепочка выдана под другой challenge
	}
	tx := &wire.Transaction{
		NetworkID:       e.ledger.State.Net.NetworkID,
		Wallet:          e.addr("CH"),
		Seq:             1,
		Type:            wire.Register,
		Payload:         rp,
		ProtocolVersion: e.ledger.State.Net.ProtocolVersion,
		TimestampClaim:  e.clock.NetTime(),
	}
	e.solveAndSign(tx, "CH", true)
	res := e.ledger.Apply(tx)
	if res.Outcome != Rejected || res.Code != "E_ATTEST" {
		t.Fatalf("переиспользованная цепочка принята: %v/%s", res.Outcome, res.Code)
	}
}

// TestE2EHeadlessScenario — полный путь участника.
func TestE2EHeadlessScenario(t *testing.T) {
	e := newE2E(t)
	// посевной круг: три давних активных поручителя (для скорости — pre-state)
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := e.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e.clock.NetTime() - 400*86400,
			ActivatedAt:     e.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		e.ledger.State.Wallets[string(addr)] = w
		e.ledger.State.FingerprintIndex[string(fp)] = addr
	}

	// 1. register новичка с гардианами
	e.requireAccepted(e.register("ALICE", "V1", "V2", "V3"), "register ALICE")
	alice := e.ledger.State.Wallet(e.addr("ALICE"))
	if alice.Status != wire.StatusRegistered {
		t.Fatalf("ALICE не registered: %s", alice.Status)
	}

	// 2. vouch x3 -> active
	for _, v := range []string{"V1", "V2", "V3"} {
		res := e.ledger.Apply(e.tx(v, wire.Vouch, &wire.VouchPayload{
			Candidate: e.addr("ALICE"), Epoch: e.clock.Epoch(),
		}, false))
		e.requireAccepted(res, "vouch "+v)
	}
	if alice.Status != wire.StatusActive {
		t.Fatalf("ALICE не active после 3 vouch: %s", alice.Status)
	}

	// 3. mint первого периода (активация в периоде 1; заявляем период 1)
	// current period = 1 -> завершённых периодов нет; сдвигаем время на период 2
	e.clock.P = 2
	e.clock.T += 31 * 86400
	e.requireAccepted(e.ledger.Apply(e.tx("ALICE", wire.Mint,
		&wire.MintPayload{Period: 1, Amount: 1000}, true)), "mint p1")
	if alice.TotalBalance() != 901 { // 1000 − R 100 + остаток деления 1
		t.Fatalf("баланс ALICE после mint: %d, ожидается 901", alice.TotalBalance())
	}
	// R-доля поручителям: 100 / 3 = 33 каждому, остаток 1 — майнящему
	for _, v := range []string{"V1", "V2", "V3"} {
		if got := e.ledger.State.Wallet(e.addr(v)).TotalBalance(); got != 33 {
			t.Fatalf("R-доля %s: %d, ожидается 33", v, got)
		}
	}

	// 4. перевод сразу после выпуска: свежая когорта ещё в окне
	// переводимости (3 мес., T8) -> E_BALANCE — так и должно быть
	e.clock.T += 14*86400 + 60
	res4 := e.ledger.Apply(e.tx("ALICE", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V1"), Amount: 100},
	}, false))
	if res4.Outcome != Rejected || res4.Code != "E_BALANCE" {
		t.Fatalf("перевод свежего выпуска: %v/%s, ожидается reject/E_BALANCE", res4.Outcome, res4.Code)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("после отказа: %v", bad)
	}

	// 5. созревание когорты: период 5 (окно 3: период 1 переводим с 4)
	e.clock.P = 5
	e.clock.T += 3 * 31 * 86400
	e.ledger.syncTimeForTest()
	res := e.ledger.Apply(e.tx("ALICE", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V1"), Amount: 100},
	}, false))
	e.requireAccepted(res, "transfer после созревания")

	// 6. отложенный перевод выше P: pending -> исполнение через 48 ч
	e.ledger.State.Wallet(e.addr("V1")).Liquid += 100000
	e.ledger.State.TotalMinted += 100000 // пополнение для сценария (INV-8)
	res = e.ledger.Apply(e.tx("V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("ALICE"), Amount: 5000},
	}, false))
	e.requireAccepted(res, "отложенный перевод")
	if e.ledger.State.Wallet(e.addr("V1")).Locked == 0 {
		t.Fatal("перевод выше P не зарезервирован")
	}
	e.clock.T += 49 * 3600
	e.ledger.syncTimeForTest()
	if e.ledger.State.Wallet(e.addr("ALICE")).Liquid < 5000 {
		t.Fatal("отложенный перевод не исполнен по сроку")
	}

	// 7. эквивокация: два перевода с одним seq; evidence -> flagged
	seq := e.ledger.State.Wallet(e.addr("ALICE")).Seq + 1
	t1 := e.tx("ALICE", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V1"), Amount: 10}}, false)
	if t1.Seq != seq {
		t.Fatalf("seq: %d != %d", t1.Seq, seq)
	}
	e.requireAccepted(e.ledger.Apply(t1), "первая ветка")
	t2 := e.tx("ALICE", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 20}}, false)
	t2.Seq = seq
	t2.Prev = t1.Prev
	e.solveAndSign(t2, "ALICE", false)
	if res := e.ledger.Apply(t2); res.Outcome != Conflict {
		t.Fatalf("конфликтная ветка: %v/%s", res.Outcome, res.Code)
	}
	ev := e.tx("V2", wire.Evidence, &wire.EvidencePayload{
		TxA: t1, TxB: t2, Kind: wire.EvidenceEquivocation,
	}, false)
	e.requireAccepted(e.ledger.Apply(ev), "evidence")
	if !e.ledger.State.Wallet(e.addr("ALICE")).Flagged {
		t.Fatal("ALICE не помечен flagged")
	}

	// 8. migrate: ALICE теряет телефон; новый кошелёк NEW с 2 из 3 гардианов
	e.requireAccepted(e.register("NEW"), "register NEW")
	pro := &wire.MigratePropose{
		OldAddress: e.addr("ALICE"),
		Nonce:      1,
	}
	for _, g := range []string{"V1", "V2"} {
		stmt := &wire.MigrateStatement{
			NetworkID:  e.ledger.State.Net.NetworkID,
			OldAddress: e.addr("ALICE"), NewAddress: e.addr("NEW"),
			Nonce: 1, Epoch: e.clock.Epoch(),
		}
		sh := stmt.Hash()
		sig, err := crypto.SignDER(e.key(g), sh[:])
		if err != nil {
			t.Fatalf("guardian sig: %v", err)
		}
		pro.Approvals = append(pro.Approvals, wire.GuardianApproval{
			Guardian: e.addr(g), StatementSig: sig,
		})
	}
	e.requireAccepted(e.ledger.Apply(e.tx("NEW", wire.Migrate,
		&wire.MigratePayload{Propose: pro}, false)), "migrate propose")
	// исполнение через 14 дней
	e.clock.T += 15 * 86400
	e.ledger.syncTimeForTest()
	newW := e.ledger.State.Wallet(e.addr("NEW"))
	oldW := e.ledger.State.Wallet(e.addr("ALICE"))
	if newW.Status != wire.StatusActive {
		t.Fatalf("NEW не active после миграции: %s", newW.Status)
	}
	if !oldW.Terminal {
		t.Fatal("старый адрес не терминален")
	}
	if !e.ledger.State.FingerprintBlacklist[string(oldW.Fingerprint)] {
		t.Fatal("фингерпринт старого кошелька не в чёрном списке")
	}
	if newW.TotalBalance() == 0 {
		t.Fatal("баланс не перенесён при миграции")
	}

	// INV-8 финально + FullCheck
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
	if bad := e.ledger.FullCheck(); len(bad) > 0 {
		t.Fatalf("FullCheck: %v", bad)
	}
}

// TestReplayFromStore — INV-3: пересчёт состояния из genesis + журнала.
func TestReplayFromStore(t *testing.T) {
	e := newE2E(t)
	// маленький сценарий: register + vouch + mint
	e.requireAccepted(e.register("R1"), "register R1")
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := e.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e.clock.NetTime() - 400*86400,
			ActivatedAt:     e.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		e.ledger.State.Wallets[string(addr)] = w
		e.ledger.State.FingerprintIndex[string(fp)] = addr
	}
	for _, v := range []string{"V1", "V2", "V3"} {
		e.requireAccepted(e.ledger.Apply(e.tx(v, wire.Vouch, &wire.VouchPayload{
			Candidate: e.addr("R1"), Epoch: e.clock.Epoch(),
		}, false)), "vouch")
	}

	// сохранить журнал в SQLite
	dbPath := filepath.Join(t.TempDir(), "replay.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	// AcceptedTxs() каузален с этапа 5 (порядок принятия, не map) —
	// сортировка из фикса этапа 4 удалена.
	for _, tx := range e.ledger.AcceptedTxs() {
		if err := st.SaveTx(tx, 0); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	// replay: новое ядро, применяем журнал в порядке (wallet, seq)
	loaded, err := st.LoadTxs()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	clock2 := &FixedClock{T: e.clock.NetTime(), E: e.clock.Epoch(), P: e.clock.Period()}
	gp := params.TestGenesis()
	gp.Manifest.Profile.Pow = e.ledger.State.Net.Profile.Pow
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	l2, err := New(gp, clock2, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis2: %v", err)
	}
	// pre-state посевного круга воспроизводим так же, как в сценарии
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := e.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e.clock.NetTime() - 400*86400,
			ActivatedAt:     e.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		l2.State.Wallets[string(addr)] = w
		l2.State.FingerprintIndex[string(fp)] = addr
	}
	l2.TrustedRoots = e.ledger.TrustedRoots
	for _, tx := range loaded {
		if res := l2.Apply(tx); res.Outcome != Accepted {
			t.Fatalf("replay: tx %s: %v/%s", tx.TxID(), res.Outcome, res.Code)
		}
	}
	// сверка состояний (INV-3)
	if len(l2.State.Wallets) != len(e.ledger.State.Wallets) {
		t.Fatalf("replay: кошельков %d, ожидается %d", len(l2.State.Wallets), len(e.ledger.State.Wallets))
	}
	for k, w1 := range e.ledger.State.Wallets {
		w2, ok := l2.State.Wallets[k]
		if !ok {
			t.Fatalf("replay: кошелёк %x отсутствует", w1.Address)
		}
		if w1.Status != w2.Status || w1.TotalBalance() != w2.TotalBalance() ||
			w1.Seq != w2.Seq || w1.Flagged != w2.Flagged {
			t.Fatalf("replay: состояние %x расходится", w1.Address)
		}
	}
	if l2.State.TotalMinted != e.ledger.State.TotalMinted ||
		l2.State.TotalBurned != e.ledger.State.TotalBurned {
		t.Fatal("replay: счётчики INV-8 расходятся")
	}
}
