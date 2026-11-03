// Тесты фикса M7 (тема mint-fix, вариант A, решение владельца 03.11.2026):
// якорь слота челленджа — ТЕКУЩИЙ период выпуска (l.tt.period от txNow),
// не заявленный mp.Period. Свойство окна, вердикты в/вне окна, накопление
// M6, accrual_window, live==replay, песочница, protocolVersion,
// дофиксный вектор (смена семантики). Данные репродуцируемы: ключи
// crypto.DeterministicKey, времена фиксированы, адреса подбираются
// детерминированным поиском.
package ledger

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

const (
	mfixGenesisTime = uint64(1_700_000_000)
	mfixPeriodSecs  = uint64(30 * 86400) // конвенция headless-ядра (поле 61: 0)
	mfixWindowSecs  = uint64(48 * 3600)  // ±48 ч (7.6)
)

// mfixGenesis — тестовый genesis под фикс M7: малые PoW-таргеты, без
// демереджа, окно ±48 ч, genesis фиксирован (детерминизм периодов).
func mfixGenesis() *wire.GenesisPackage {
	gp := params.TestGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	gp.Manifest.Profile.DemurrageRateMicros = 0
	gp.Manifest.Profile.MintChallengeWindowHours = 48
	gp.Genesis.GenesisTime = mfixGenesisTime
	wire.SetManifestHash(gp)
	return gp
}

// mfixEnv — окружение: genesis + пред-сеяные активные кошельки
// (как pre_state векторов). skip — песочница (SkipMintChallengeWindow).
type mfixEnv struct {
	t      *testing.T
	ledger *Ledger
	keys   map[string]*ecdsa.PrivateKey
	addrs  map[string][]byte
}

func newMfixEnv(t *testing.T, skip bool) *mfixEnv {
	l, err := New(mfixGenesis(), Options{SkipMintChallengeWindow: skip})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	return &mfixEnv{t: t, ledger: l,
		keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
}

// addrOf — адрес детерминированного ключа (без ввода в состояние).
func (e *mfixEnv) addrOf(id string) []byte {
	priv := crypto.DeterministicKey("mint-fix:" + id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	return crypto.DeriveAddress(fp, e.ledger.State.Net.NetworkID)
}

// seedWallet — ввод активного кошелька (activated_period = 1,
// last_claimed = last, seq = 1, детерминированный tip).
func (e *mfixEnv) seedWallet(id string, last uint64) []byte {
	priv := crypto.DeterministicKey("mint-fix:" + id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	addr := crypto.DeriveAddress(fp, e.ledger.State.Net.NetworkID)
	w := &Wallet{
		Address:           addr,
		Fingerprint:       fp,
		PublicKey:         &priv.PublicKey,
		Status:            wire.StatusActive,
		RegisteredAt:      mfixGenesisTime,
		ActivatedAt:       mfixGenesisTime,
		ActivatedPeriod:   1,
		LastClaimedPeriod: last,
		LastTouchEpoch:    0,
		DelayedThreshold:  3000,
		Seq:               1,
	}
	tip := sha256.Sum256([]byte("mint-fix-tip:" + id))
	w.TipTxID = append([]byte(nil), tip[:]...)
	e.ledger.State.Wallets[key(addr)] = w
	e.ledger.State.FingerprintIndex[string(fp)] = addr
	e.keys[id] = priv
	e.addrs[id] = addr
	return addr
}

// mintTx — полностью валидная Mint-транзакция (PoW решён, подпись);
// version 0 — текущая версия протокола сети.
func (e *mfixEnv) mintTx(id string, claimPeriod, amount, txNow uint64, version uint32) *wire.Transaction {
	addr := e.addrs[id]
	w := e.ledger.State.Wallet(addr)
	tx := &wire.Transaction{
		NetworkID:      e.ledger.State.Net.NetworkID,
		Wallet:         addr,
		Seq:            w.Seq + 1,
		Prev:           w.TipTxID,
		Type:           wire.Mint,
		TimestampClaim: txNow,
		Payload:        &wire.MintPayload{Period: claimPeriod, Amount: amount},
		Pow:            &wire.PowProof{Algorithm: wire.PowArgon2id},
	}
	if version == 0 {
		version = e.ledger.State.Net.ProtocolVersion
	}
	tx.ProtocolVersion = version
	net := e.ledger.State.Net
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
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(e.keys[id], sh[:])
	if err != nil {
		e.t.Fatalf("sign: %v", err)
	}
	tx.Signature = sig
	return tx
}

// slotOf — слот челленджа адреса в периоде (формула ядра, 30 сут).
func (e *mfixEnv) slotOf(addr []byte, period uint64) uint64 {
	return mintChallengeSlot(mfixGenesisTime, addr, period, mfixPeriodSecs)
}

// findWallet — детерминированный поиск кошелька под условие на слоты:
// перебирает ключи mint-fix:<prefix><i>, вводит первый подходящий.
func (e *mfixEnv) findWallet(prefix string, last uint64, cond func(addr []byte) bool) string {
	for i := 0; i < 100000; i++ {
		id := fmt.Sprintf("%s%d", prefix, i)
		if cond(e.addrOf(id)) {
			e.seedWallet(id, last)
			return id
		}
	}
	e.t.Fatalf("не подобран адрес под условие (префикс %s)", prefix)
	return ""
}

// ---------------------------------------------------------------------------
// (a) свойство окна: пересечение [slot−48 ч, slot+48 ч] с текущим периодом
// всегда >= 48 ч — каждый период у каждого кошелька имеет достижимое окно
// ---------------------------------------------------------------------------

func TestMintFixWindowIntersectionProperty(t *testing.T) {
	for i := 0; i < 64; i++ {
		sum := sha256.Sum256([]byte(fmt.Sprintf("mint-fix-prop-addr:%d", i)))
		addr := sum[:]
		for period := uint64(1); period <= 8; period++ {
			slot := mintChallengeSlot(mfixGenesisTime, addr, period, mfixPeriodSecs)
			start := mfixGenesisTime + period*mfixPeriodSecs
			end := start + mfixPeriodSecs
			if slot < start || slot >= end {
				t.Fatalf("адрес %d, период %d: слот %d вне периода [%d, %d)",
					i, period, slot, start, end)
			}
			lo, hi := slot-mfixWindowSecs, slot+mfixWindowSecs
			isectLo, isectHi := lo, hi
			if start > isectLo {
				isectLo = start
			}
			if end < isectHi {
				isectHi = end
			}
			if isectHi-isectLo < mfixWindowSecs {
				t.Fatalf("адрес %d, период %d: пересечение окна с периодом %d < 48 ч",
					i, period, isectHi-isectLo)
			}
		}
	}
	// короткий период (genesis-поле 61, тестовые сети): окно ±48 ч шире
	// периода — накрывает текущий период целиком, достижимость тривиальна
	const shortPeriod = uint64(120)
	for i := 0; i < 8; i++ {
		sum := sha256.Sum256([]byte(fmt.Sprintf("mint-fix-short-addr:%d", i)))
		addr := sum[:]
		for period := uint64(1); period <= 5; period++ {
			slot := mintChallengeSlot(mfixGenesisTime, addr, period, shortPeriod)
			start := mfixGenesisTime + period*shortPeriod
			end := start + shortPeriod
			if slot < start || slot >= end {
				t.Fatalf("short: адрес %d, период %d: слот вне периода", i, period)
			}
			// любая точка текущего периода — внутри окна ±48 ч слота
			for _, probe := range []uint64{start, slot, end - 1} {
				lo, hi := slot-mfixWindowSecs, slot+mfixWindowSecs
				if probe < lo || probe > hi {
					t.Fatalf("short: адрес %d, период %d: точка %d периода вне окна",
						i, period, probe)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// (b) mint в окне текущего периода проходит; вне окна — E_POW_HARD
// ---------------------------------------------------------------------------

func TestMintFixWindowInOut(t *testing.T) {
	const cur = uint64(5)
	// слот достаточно ранний: slot+49 ч остаётся внутри текущего периода
	e := newMfixEnv(t, false)
	end := mfixGenesisTime + (cur+1)*mfixPeriodSecs
	idIn := e.findWallet("in-win-", 3, func(addr []byte) bool {
		return e.slotOf(addr, cur)+49*3600 < end
	})
	idOut := e.findWallet("out-win-", 3, func(addr []byte) bool {
		return e.slotOf(addr, cur)+49*3600 < end
	})
	claim := cur - 1 // 4: last=3, diff=1 (M6: amount = 1000)

	// в окне: txNow = слоту текущего периода — accept
	slotIn := e.slotOf(e.addrs[idIn], cur)
	res := e.ledger.Apply(e.mintTx(idIn, claim, 1000, slotIn, 0))
	if res.Outcome != Accepted || res.Code != "OK" {
		t.Fatalf("mint в окне: %v/%s, хочу accept/OK", res.Outcome, res.Code)
	}
	// вне окна: txNow = слот+49 ч (тот же текущий период) — E_POW_HARD
	slotOut := e.slotOf(e.addrs[idOut], cur)
	res = e.ledger.Apply(e.mintTx(idOut, claim, 1000, slotOut+49*3600, 0))
	if res.Outcome != Rejected || res.Code != "E_POW_HARD" {
		t.Fatalf("mint вне окна: %v/%s, хочу reject/E_POW_HARD", res.Outcome, res.Code)
	}
}

// ---------------------------------------------------------------------------
// (c) заявка старых периодов (накопление, diff <= 6) в окне текущего
// периода проходит; amount по M6 = issuance × diff
// ---------------------------------------------------------------------------

func TestMintFixAccumulationInWindow(t *testing.T) {
	const cur = uint64(5)
	e := newMfixEnv(t, false)
	id := e.findWallet("acc-", 1, func(addr []byte) bool { return true })
	// last=1, claim=4: diff=3 <= accrual_window=6 — накопление трёх
	// пропущенных периодов компенсируется в окне текущего периода 5
	slot := e.slotOf(e.addrs[id], cur)
	res := e.ledger.Apply(e.mintTx(id, 4, 3000, slot, 0))
	if res.Outcome != Accepted || res.Code != "OK" {
		t.Fatalf("mint с накоплением в окне: %v/%s, хочу accept/OK", res.Outcome, res.Code)
	}
	w := e.ledger.State.Wallet(e.addrs[id])
	if w.LastClaimedPeriod != 4 {
		t.Fatalf("last_claimed = %d, хочу 4", w.LastClaimedPeriod)
	}
	var total uint64
	for _, c := range w.Cohorts {
		total += c.Amount
	}
	if total != 3000 {
		t.Fatalf("когорты = %d, хочу 3000 (M6: 1000 × 3)", total)
	}
}

// ---------------------------------------------------------------------------
// (d) 6+ пропусков — E_ACCRUAL_WINDOW (правило M5 не изменилось)
// ---------------------------------------------------------------------------

func TestMintFixAccrualWindowUnchanged(t *testing.T) {
	const cur = uint64(10)
	e := newMfixEnv(t, false)
	// last=2, claim=9: diff=7 > accrual_window=6 — сгорание сохранено
	id := e.findWallet("accwin-", 2, func(addr []byte) bool { return true })
	slot := e.slotOf(e.addrs[id], cur)
	res := e.ledger.Apply(e.mintTx(id, 9, 7000, slot, 0))
	if res.Outcome != Rejected || res.Code != "E_ACCRUAL_WINDOW" {
		t.Fatalf("mint с diff=7: %v/%s, хочу reject/E_ACCRUAL_WINDOW", res.Outcome, res.Code)
	}
}

// ---------------------------------------------------------------------------
// (e) live == replay: один журнал — один state_root (окно от txNow,
// вердикт детерминирован; replay применяет тот же журнал через Apply)
// ---------------------------------------------------------------------------

// mfixDigest — детерминированный дайджест состояния (state_root для теста).
func mfixDigest(l *Ledger) [32]byte {
	h := sha256.New()
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], l.State.NetTime)
	h.Write(b[:])
	binary.BigEndian.PutUint64(b[:], l.State.TotalMinted)
	h.Write(b[:])
	binary.BigEndian.PutUint64(b[:], l.State.TotalBurned)
	h.Write(b[:])
	binary.BigEndian.PutUint64(b[:], uint64(len(l.State.Wallets)))
	h.Write(b[:])
	addrs := make([]string, 0, len(l.State.Wallets))
	for a := range l.State.Wallets {
		addrs = append(addrs, a)
	}
	sort.Strings(addrs)
	for _, a := range addrs {
		w := l.State.Wallets[a]
		h.Write([]byte(a))
		binary.BigEndian.PutUint64(b[:], w.Seq)
		h.Write(b[:])
		h.Write(w.TipTxID)
		binary.BigEndian.PutUint64(b[:], w.Liquid)
		h.Write(b[:])
		binary.BigEndian.PutUint64(b[:], w.Locked)
		h.Write(b[:])
		binary.BigEndian.PutUint64(b[:], w.LastClaimedPeriod)
		h.Write(b[:])
		binary.BigEndian.PutUint64(b[:], uint64(len(w.Cohorts)))
		h.Write(b[:])
		for _, c := range w.Cohorts {
			binary.BigEndian.PutUint64(b[:], c.Period)
			h.Write(b[:])
			binary.BigEndian.PutUint64(b[:], c.Amount)
			h.Write(b[:])
		}
	}
	return [32]byte(h.Sum(nil))
}

func TestMintFixLiveReplayDeterminism(t *testing.T) {
	const cur = uint64(5)
	// live: применение журнала к живому ядру
	live := newMfixEnv(t, false)
	id := live.findWallet("replay-", 1, func(addr []byte) bool { return true })
	slot := live.slotOf(live.addrs[id], cur)
	tx1 := live.mintTx(id, 4, 3000, slot, 0)
	if res := live.ledger.Apply(tx1); res.Outcome != Accepted {
		t.Fatalf("live mint 1: %v/%s", res.Outcome, res.Code)
	}
	// второй выпуск в следующем периоде (окно снова от текущего периода)
	cur2 := cur + 1
	slot2 := live.slotOf(live.addrs[id], cur2)
	tx2 := live.mintTx(id, 5, 1000, slot2, 0)
	if res := live.ledger.Apply(tx2); res.Outcome != Accepted {
		t.Fatalf("live mint 2: %v/%s", res.Outcome, res.Code)
	}

	// replay: тот же журнал в свежее ядро (cons.Replay применяет те же
	// tx тем же Apply — вердикты и состояние обязаны совпасть)
	replay := newMfixEnv(t, false)
	replay.seedWallet(id, 1)
	for _, tx := range []*wire.Transaction{tx1, tx2} {
		if res := replay.ledger.Apply(tx); res.Outcome != Accepted {
			t.Fatalf("replay: %v/%s", res.Outcome, res.Code)
		}
	}
	dLive, dReplay := mfixDigest(live.ledger), mfixDigest(replay.ledger)
	if dLive != dReplay {
		t.Fatalf("live и replay разошлись: %x != %x", dLive, dReplay)
	}

	// отклонённая tx не меняет state_root: один журнал — один корень
	before := mfixDigest(replay.ledger)
	out := replay.mintTx(id, 6, 1000, replay.slotOf(replay.addrs[id], cur2)+49*3600, 0)
	if res := replay.ledger.Apply(out); res.Code != "E_POW_HARD" {
		t.Fatalf("вне окна: %s, хочу E_POW_HARD", res.Code)
	}
	if mfixDigest(replay.ledger) != before {
		t.Fatal("отклонённая tx изменила состояние")
	}
}

// ---------------------------------------------------------------------------
// (f) песочница: SkipMintChallengeWindow — окно не проверяется
// ---------------------------------------------------------------------------

func TestMintFixSandboxSkip(t *testing.T) {
	const cur = uint64(5)
	e := newMfixEnv(t, true) // песочница
	end := mfixGenesisTime + (cur+1)*mfixPeriodSecs
	id := e.findWallet("sbox-", 3, func(addr []byte) bool {
		return e.slotOf(addr, cur)+49*3600 < end
	})
	slot := e.slotOf(e.addrs[id], cur)
	// вне окна (слот+49 ч) — песочница принимает
	res := e.ledger.Apply(e.mintTx(id, 4, 1000, slot+49*3600, 0))
	if res.Outcome != Accepted || res.Code != "OK" {
		t.Fatalf("sandbox mint вне окна: %v/%s, хочу accept/OK", res.Outcome, res.Code)
	}
}

// ---------------------------------------------------------------------------
// (g) прежняя protocolVersion (2) — E_VER
// ---------------------------------------------------------------------------

func TestMintFixOldProtocolVersion(t *testing.T) {
	const cur = uint64(5)
	e := newMfixEnv(t, false)
	id := e.findWallet("ver-", 3, func(addr []byte) bool { return true })
	slot := e.slotOf(e.addrs[id], cur)
	// валидная во всём остальном tx с версией 2 (до фикса M7)
	res := e.ledger.Apply(e.mintTx(id, 4, 1000, slot, 2))
	if res.Outcome != Rejected || res.Code != "E_VER" {
		t.Fatalf("tx с protocolVersion 2: %v/%s, хочу reject/E_VER", res.Outcome, res.Code)
	}
}

// ---------------------------------------------------------------------------
// (h) дофиксный вектор (фиксирует смену семантики): txNow после конца
// заявленного периода — вердикт от слота ТЕКУЩЕГО периода, не заявленного
// ---------------------------------------------------------------------------

func TestMintFixPreFixVector(t *testing.T) {
	const (
		claim = uint64(4) // заявленный период: [g+4P, g+5P)
		cur   = uint64(5) // текущий (txNow — сразу после конца заявленного)
	)
	// период p занимает [g+p×P, g+(p+1)×P) (periodAt, 9.2)
	startClaim := mfixGenesisTime + claim*mfixPeriodSecs
	endClaim := mfixGenesisTime + (claim+1)*mfixPeriodSecs
	txNow := endClaim + 3600 // час после конца заявленного периода

	// h2 — сменяющий вердикт вектор: слот заявленного периода в последних
	// 24 ч (старая семантика: окно заявленного периода накрывало txNow —
	// accept), слот текущего периода >= 49 ч от txNow (новая: E_POW_HARD)
	e2 := newMfixEnv(t, false)
	id2 := e2.findWallet("prefix-late-", 3, func(addr []byte) bool {
		slotClaim := e2.slotOf(addr, claim)
		if slotClaim < endClaim-24*3600 {
			return false
		}
		slotCur := e2.slotOf(addr, cur)
		return slotCur > txNow+49*3600 || slotCur < txNow-49*3600
	})
	res := e2.ledger.Apply(e2.mintTx(id2, claim, 1000, txNow, 0))
	if res.Outcome != Rejected || res.Code != "E_POW_HARD" {
		t.Fatalf("дофиксный вектор (поздний слот): %v/%s, хочу reject/E_POW_HARD "+
			"(ранее окно заявленного периода было открыто)", res.Outcome, res.Code)
	}

	// h1 — буквальный вектор мандата: слот заявленного периода в середине
	// периода, txNow после конца заявленного — E_POW_HARD; заявленный
	// период больше не участвует в вердикте (ранее окно считалось по нему)
	e1 := newMfixEnv(t, false)
	id1 := e1.findWallet("prefix-mid-", 3, func(addr []byte) bool {
		slotClaim := e1.slotOf(addr, claim)
		if slotClaim < startClaim+mfixPeriodSecs/3 ||
			slotClaim > startClaim+2*mfixPeriodSecs/3 {
			return false
		}
		slotCur := e1.slotOf(addr, cur)
		return slotCur > txNow+49*3600 || slotCur < txNow-49*3600
	})
	res = e1.ledger.Apply(e1.mintTx(id1, claim, 1000, txNow, 0))
	if res.Outcome != Rejected || res.Code != "E_POW_HARD" {
		t.Fatalf("дофиксный вектор (срединный слот): %v/%s, хочу reject/E_POW_HARD",
			res.Outcome, res.Code)
	}
}
