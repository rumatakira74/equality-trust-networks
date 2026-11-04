// Тесты режима периода-якоря слота челленджа выпуска (манифест-поле 63
// mint_slot_mode, тема fix-mint-window, вариант A аудита Sonnet-2 п. 3.2)
// и правки дефекта M5 для quota_reset (п. 3.3).
//
// Режим 0 — наследие: слот заявляемого периода (достижимость 1/15 за
// период, пропуск периода необратим). Режим 1 — слот периода выпуска:
// окно есть у каждого кошелька каждый период. Оба режима проверяются на
// одной и той же геометрии (T = 30 сут, W = ±48 ч) и одном наборе
// кошельков — различие только в поле манифеста.
package ledger

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// modeGenesis — genesis боевой геометрии (T = 30 сут, W = ±48 ч) с
// заданным режимом якоря слота.
func modeGenesis(mode uint64) *wire.GenesisPackage {
	gp := e2eGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	gp.Manifest.Profile.MintSlotMode = mode
	wire.SetManifestHash(gp)
	return gp
}

// newModeEnv — окружение с окном челленджа ВКЛЮЧЕНО и заданным режимом.
func newModeEnv(t *testing.T, mode uint64) *e2eEnv {
	t.Helper()
	l, err := New(modeGenesis(mode), Options{})
	if err != nil {
		t.Fatalf("genesis (mode %d): %v", mode, err)
	}
	e := &e2eEnv{t: t, ledger: l, now: e2eT0,
		keys:  make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte)}
	return e
}

// TestMintSlotModeAnchor — период-якорь по режиму манифеста: 0 —
// заявляемый период, 1 — период выпуска (periodAt(now)).
func TestMintSlotModeAnchor(t *testing.T) {
	const claim = uint64(5)
	for _, tc := range []struct {
		mode uint64
		now  uint64
	}{
		{0, e2eT0 + 6*winT + 3600},
		{1, e2eT0 + 6*winT + 3600},
		{1, e2eT0 + 9*winT + 3600},
	} {
		e := newModeEnv(t, tc.mode)
		want := claim // наследие: заявляемый период
		if tc.mode == 1 {
			want = e.ledger.periodAt(tc.now) // период выпуска
		}
		if got := e.ledger.MintWindowAnchor(claim, tc.now); got != want {
			t.Fatalf("mode %d, now=%d: якорь %d, хочу %d", tc.mode, tc.now, got, want)
		}
	}
}

// TestMintSlotModeReachability — один и тот же кошелёк и один и тот же
// момент подачи дают РАЗНЫЙ вердикт по режиму: это и есть смысл поля
// (иначе режим не был бы консенсус-параметром).
//
// Случай «слот периода выпуска в середине периода»: в режиме 1 mint
// проходит (окно вокруг слота периода выпуска), в режиме 0 — E_POW_HARD
// (окно вокруг слота заявляемого периода, который лежит в предыдущем
// периоде и отстоит от момента подачи дальше 48 ч).
func TestMintSlotModeIssuingPeriodReachable(t *testing.T) {
	const q, p = uint64(5), uint64(6)
	for _, tc := range []struct {
		mode uint64
		want Outcome
	}{
		{1, Accepted},
		{0, Rejected},
	} {
		e := newModeEnv(t, tc.mode)
		g, T, _ := e.winGeom()
		// слот периода выпуска p — в середине периода p
		id := e.findWallet(p, func(u uint64) bool { return u > 14*86400 && u < 16*86400 })
		e.injectActive(id, q, 0)
		at := mintChallengeSlot(g, e.addr(id), p, T)
		res := e.tryMint(id, q, at)
		if res.Outcome != tc.want {
			t.Fatalf("mode %d: mint за период %d в слоте периода %d — %v/%s, хочу %v",
				tc.mode, q, p, res.Outcome, res.Code, tc.want)
		}
		if tc.mode == 0 && res.Code != "E_POW_HARD" {
			t.Fatalf("mode 0: код %s, хочу E_POW_HARD (окно слота заявляемого периода)", res.Code)
		}
	}
}

// TestMintSlotModeLegacyReachable — обратный случай: слот ЗАЯВЛЯЕМОГО
// периода в последних 48 ч периода. В режиме 0 mint проходит сразу после
// границы (окно слота q накрывает начало периода q+1), в режиме 1 —
// E_POW_HARD, если слот периода выпуска отстоит от границы больше 48 ч.
func TestMintSlotModeLegacyReachable(t *testing.T) {
	const q = uint64(5)
	for _, tc := range []struct {
		mode uint64
		want Outcome
	}{
		{0, Accepted},
		{1, Rejected},
	} {
		e := newModeEnv(t, tc.mode)
		g, T, W := e.winGeom()
		boundary := g + (q+1)*T
		// слот q — в последних 48 ч периода q; слот q+1 — дальше 48 ч
		// от границы (иначе в режиме 1 окно тоже было бы открыто)
		id := ""
		for i := 0; i < 20000; i++ {
			cand := fmt.Sprintf("mw%d", i)
			u := mintChallengeSlot(g, e.addr(cand), q, T) - (g + q*T)
			if u < T-W {
				continue
			}
			if mintChallengeSlot(g, e.addr(cand), q+1, T) <= boundary+W {
				continue
			}
			id = cand
			break
		}
		if id == "" {
			t.Fatal("не найден кошелёк под условие")
		}
		e.injectActive(id, q, 0)
		res := e.tryMint(id, q, boundary)
		if res.Outcome != tc.want {
			t.Fatalf("mode %d: mint за период %d на границе периода %d — %v/%s, хочу %v",
				tc.mode, q, q+1, res.Outcome, res.Code, tc.want)
		}
	}
}

// TestMintWindowBoundsTruncatedByPeriod — достижимое окно в режиме 1
// усечено периодом выпуска (аудит п. 3.1), в режиме 0 — нет.
func TestMintWindowBoundsTruncatedByPeriod(t *testing.T) {
	const q = uint64(5)
	e1 := newModeEnv(t, 1)
	g, T, W := e1.winGeom()
	// слот периода выпуска 6 — в начале периода: окно «влево» вышло бы
	// за границу периода 6
	id := e1.findWallet(6, func(u uint64) bool { return u < W/2 })
	e1.injectActive(id, q, 0)
	now := g + 6*T + 3600
	lo, hi := e1.ledger.MintWindowBounds(e1.addr(id), q, now)
	if lo < g+6*T {
		t.Fatalf("mode 1: окно начинается %d, до границы периода %d", lo, g+6*T)
	}
	if hi > g+7*T {
		t.Fatalf("mode 1: окно кончается %d, за границей периода %d", hi, g+7*T)
	}

	e0 := newModeEnv(t, 0)
	// в режиме 0 якорь — заявляемый период 5: окно вокруг его слота
	lo0, hi0 := e0.ledger.MintWindowBounds(e0.addr(id), q, now)
	if lo0 >= hi0 {
		t.Fatalf("mode 0: пустое окно [%d, %d]", lo0, hi0)
	}
	slot := mintChallengeSlot(g, e0.addr(id), q, T)
	if lo0 != slot-W || hi0 != slot+W {
		t.Fatalf("mode 0: окно [%d, %d], хочу [%d, %d] (без усечения периодом)",
			lo0, hi0, slot-W, slot+W)
	}
}

// TestMintSlotModeEncoding — поле 63: нулевое значение не кодируется
// (манифест прежней сети байт-в-байт как раньше, network_id не меняется),
// единица кодируется и декодируется; значение вне {0,1} — невалидный
// genesis.
func TestMintSlotModeEncoding(t *testing.T) {
	base := modeGenesis(0)
	withField := modeGenesis(1)
	if bytes.Equal(base.Manifest.Profile.Encode(), withField.Manifest.Profile.Encode()) {
		t.Fatal("mode 1 не попал в кодировку профиля")
	}
	// режим 0 — байт-в-байт как профиль без поля
	p := base.Manifest.Profile
	p.MintSlotMode = 0
	dec, err := wire.DecodeProfileParams(p.Encode())
	if err != nil {
		t.Fatalf("decode mode 0: %v", err)
	}
	if dec.MintSlotMode != 0 {
		t.Fatalf("mode 0 декодирован как %d", dec.MintSlotMode)
	}
	dec1, err := wire.DecodeProfileParams(withField.Manifest.Profile.Encode())
	if err != nil {
		t.Fatalf("decode mode 1: %v", err)
	}
	if dec1.MintSlotMode != 1 {
		t.Fatalf("mode 1 декодирован как %d", dec1.MintSlotMode)
	}
	// значение вне {0,1} — невалидный genesis (тихо применять нельзя)
	if _, err := New(modeGenesis(2), Options{}); err == nil {
		t.Fatal("mint_slot_mode = 2 принят genesis-валидацией")
	}
}

// TestQuotaResetMissedPeriodClaimable — дефект M5 (аудит п. 3.3): при
// quota_reset заявляется ЛЮБОЙ завершённый период > LastClaimedPeriod
// суммой IssuanceAmount. До фикса требование diff == 1 делало пропуск
// периода необратимым: кошелёк выпускал base+1 < p−1, когорта сгорала на
// границе, и каждый следующий выпуск был мёртв навсегда.
func TestQuotaResetMissedPeriodClaimable(t *testing.T) {
	gp := e2eGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	p := gp.Manifest.Profile
	p.IssuanceMode = wire.IssuanceQuotaReset
	p.AccrualWindowPeriods = 0
	wire.SetManifestHash(gp)
	l, err := New(gp, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	e := &e2eEnv{t: t, ledger: l, now: e2eT0,
		keys:  make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte)}
	w := e.preStateWallet("ALICE")
	w.ActivatedPeriod = 1
	w.LastClaimedPeriod = 0
	e.now = l.State.GenesisTime + 5*30*86400 + 86400 // период 5, заявляем 4
	e.settle()

	// два периода суммой — по-прежнему нельзя (накопления нет)
	multi := e.tx("ALICE", wire.Mint, &wire.MintPayload{Period: 4, Amount: 2 * p.IssuanceAmount}, true)
	if res := l.Apply(multi); res.Outcome != Rejected || res.Code != "E_MINT_AMOUNT" {
		t.Fatalf("мульти-периодная квота: %v/%s, хочу reject/E_MINT_AMOUNT", res.Outcome, res.Code)
	}
	// пропущенные периоды 1–3 не убивают выпуск: период 4 заявляется
	// суммой одного периода
	one := e.tx("ALICE", wire.Mint, &wire.MintPayload{Period: 4, Amount: p.IssuanceAmount}, true)
	if res := l.Apply(one); res.Outcome != Accepted {
		t.Fatalf("квота пропущенного периода 4: %v/%s, хочу accept", res.Outcome, res.Code)
	}
	if w.LastClaimedPeriod != 4 {
		t.Fatalf("last_claimed = %d, хочу 4", w.LastClaimedPeriod)
	}
	if w.TotalBalance() != p.IssuanceAmount {
		t.Fatalf("начислено %d, хочу %d (один период)", w.TotalBalance(), p.IssuanceAmount)
	}
}

// TestQuotaResetVoucherShareOnePeriod — для quota_reset цикл доли
// поручителя R считается за ОДИН период (аудит п. 3.3): при пропуске
// периодов доля не умножается на их число.
func TestQuotaResetVoucherShareOnePeriod(t *testing.T) {
	gp := e2eGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	p := gp.Manifest.Profile
	p.IssuanceMode = wire.IssuanceQuotaReset
	p.AccrualWindowPeriods = 0
	wire.SetManifestHash(gp)
	l, err := New(gp, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	e := &e2eEnv{t: t, ledger: l, now: e2eT0,
		keys:  make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte)}
	w := e.preStateWallet("ALICE")
	w.ActivatedPeriod = 1
	w.LastClaimedPeriod = 0
	// K активаторов (доля R начисляется при len(activators) == vouch_k);
	// кошельки активаторов должны быть в реестре — доля пишется в их когорты
	for _, id := range []string{"V1", "V2", "V3"} {
		e.preStateWallet(id)
		w.Activators = append(w.Activators, append([]byte(nil), e.addr(id)...))
	}
	e.now = l.State.GenesisTime + 5*30*86400 + 86400
	e.settle()

	tx := e.tx("ALICE", wire.Mint, &wire.MintPayload{Period: 4, Amount: p.IssuanceAmount}, true)
	if res := l.Apply(tx); res.Outcome != Accepted {
		t.Fatalf("квота периода 4: %v/%s", res.Outcome, res.Code)
	}
	// доля R за ОДИН период: 10 % от IssuanceAmount (периоды 1–3
	// пропущены; старый цикл base+1..mp.Period дал бы 4 × доля)
	share := p.IssuanceAmount * p.VoucherRewardMicros / 1_000_000
	var gotShare uint64
	for _, id := range []string{"V1", "V2", "V3"} {
		for _, c := range e.ledger.State.Wallet(e.addr(id)).Cohorts {
			gotShare += c.Amount
		}
	}
	// остаток деления доли между поручителями (share mod K) уходит минтеру
	wantShare := share - share%uint64(p.VouchK)
	if gotShare != wantShare {
		t.Fatalf("доля поручителей %d, хочу %d (один период, а не 4)", gotShare, wantShare)
	}
	// эмиссия одного периода: минтер + доли поручителей
	if total := w.TotalBalance(); total != p.IssuanceAmount-gotShare {
		t.Fatalf("начислено минтеру %d, хочу %d (эмиссия одного периода минус доли)",
			total, p.IssuanceAmount-gotShare)
	}
}
