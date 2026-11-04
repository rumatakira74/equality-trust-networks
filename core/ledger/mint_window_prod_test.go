// Хелперы для тестов окна челленджа выпуска на БОЕВЫХ параметрах
// (T = 30 сут, W = ±48 ч, окно ВКЛЮЧЕНО — все остальные Apply-тесты
// ставят SkipMintChallengeWindow). Файл — из аудита Sonnet-2
// (reports/audit-sonnet/go-tests/core_ledger_mint_window_prod_test.go.txt).
//
// Тема fix-mint-window: характеризационные тесты аудита
// (TestMintWindowProdParams_Characterization, TestMintAccrualLockout_Characterization),
// которые были зелёными и кодировали ДОФИКСНОЕ поведение (достижимость
// 1/15 и поглощающее состояние), удалены — поведение изменено фиксом
// (вариант A, манифест-поле 63 mint_slot_mode). Хелперы оставлены:
// ими пользуется mint_window_fixed_test.go.
package ledger

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// winT — period_seconds по умолчанию (PeriodSeconds = 0 → 30 суток).
const winT = uint64(30 * 86400)

// newWindowEnv — как newE2E, но окно челленджа ВКЛЮЧЕНО (Options{}).
// Режим якоря — mint_slot_mode = 1 (слот периода выпуска): тесты
// окна проверяют вариант A аудита Sonnet-2; наследие (0) — предмет
// mint_slot_mode_test.go.
func newWindowEnv(t *testing.T) *e2eEnv {
	t.Helper()
	gp := e2eGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	gp.Manifest.Profile.MintSlotMode = 1
	wire.SetManifestHash(gp)
	l, err := New(gp, Options{}) // окно челленджа включено
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	prof := l.State.Net.Profile
	if prof.PeriodSeconds != 0 || prof.MintChallengeWindowHours != 48 {
		t.Fatalf("ожидались боевые параметры (period 30 сут, окно 48 ч), получено %d с / %d ч",
			prof.PeriodSeconds, prof.MintChallengeWindowHours)
	}
	return &e2eEnv{
		t: t, ledger: l, now: e2eT0,
		keys:  make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte),
	}
}

// winGeom — genesis, длина периода и полуширина окна, секунды.
func (e *e2eEnv) winGeom() (g, T, w uint64) {
	return e.ledger.State.GenesisTime, winT,
		e.ledger.State.Net.Profile.MintChallengeWindowHours * 3600
}

// injectActive — активный кошелёк прямо в реестр (минуя register/vouch).
func (e *e2eEnv) injectActive(id string, activatedPeriod, lastClaimed uint64) {
	priv := e.key(id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	addr := e.addr(id)
	e.ledger.State.Wallets[string(addr)] = &Wallet{
		Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
		Status: wire.StatusActive, Seed: true,
		RegisteredAt: e.ledger.State.NetTime, LastTouchEpoch: e.ledger.State.Epoch,
		DelayedThreshold: 3000,
		ActivatedPeriod:  activatedPeriod, LastClaimedPeriod: lastClaimed,
	}
	e.ledger.State.FingerprintIndex[string(fp)] = addr
}

// findWallet — первый id, у которого смещение слота периода period внутри
// периода (u = slot − начало периода) удовлетворяет pred.
func (e *e2eEnv) findWallet(period uint64, pred func(u uint64) bool) string {
	e.t.Helper()
	g, T, _ := e.winGeom()
	for i := 0; i < 20000; i++ {
		id := fmt.Sprintf("mw%d", i)
		u := mintChallengeSlot(g, e.addr(id), period, T) - (g + period*T)
		if pred(u) {
			return id
		}
	}
	e.t.Fatalf("не найден кошелёк под условие")
	return ""
}

// tryMint — настоящий Apply(Mint за период q) с claim = at.
func (e *e2eEnv) tryMint(id string, q, at uint64) Result {
	w := e.ledger.State.Wallet(e.addr(id))
	base := w.LastClaimedPeriod
	if base == 0 {
		base = w.ActivatedPeriod - 1
	}
	prof := e.ledger.State.Net.Profile
	amount := prof.IssuanceAmount * (q - base)
	if prof.IssuanceMode == wire.IssuanceQuotaReset {
		amount = prof.IssuanceAmount // M5: quota_reset — один период
	}
	e.now = at
	tx := e.tx(id, wire.Mint, &wire.MintPayload{Period: q, Amount: amount}, true)
	return e.ledger.Apply(tx)
}
