// Тесты клампа claim окном накопления (AUDIT-MINT-FIX-02, D2; тема
// fix-d2-clamp). UI-claim из MintWindowJSON обязан совпадать с формулой
// ядра (core/ledger/validate.go, applyMint: claim ≤ base + accrual_window,
// base = last_claimed, для первого выпуска base = activated_period − 1), а
// подача по нему — проходить M5 без E_ACCRUAL_WINDOW.
package tnp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// mintClampPeriodSecs — конвенция headless-ядра (30 сут; genesis-поле 61
// в песочнице не задано).
const mintClampPeriodSecs = 30 * 24 * 3600

// mintClampEnv — узел песочницы, живые часы которого стоят в периоде
// period, и зарегистрированный кошелёк с заданным состоянием M4/M5
// (activated/last). Регистрация всегда даёт activated = текущий период,
// поэтому состояние «пропустившего периоды» кошелька инъектируется в
// реестр напрямую (тот же приём, что injectActive в core-тестах).
type mintClampEnv struct {
	node *Node
	gen  *Genesis
	addr []byte
	priv []byte
}

// mintWindow — разобранный ответ моста (MintWindowJSON).
type mintWindow struct {
	claim    uint64
	amount   uint64
	period   uint64
	accrual  uint64
	issuance uint64
	last     uint64
	open     bool
}

func newMintClampEnv(t *testing.T, period, activated, last uint64) *mintClampEnv {
	t.Helper()
	// genesis в прошлом на (period периодов + 5 сут): часы узла дают
	// ровно period, запас внутри периода — 5 сут
	ago := time.Duration(int64(period*mintClampPeriodSecs+5*24*3600)) * time.Second
	gen := SandboxGenesis("tnp-mint-clamp", time.Now().Add(-ago).Unix(), 3600,
		0x0010_0000_0000_0000, 0x0010_0000_0000_0000, 8, 1, 1)
	addr, pub, priv := softRegister(t, gen)
	node, err := StartNode(gen, "full", "", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { node.Close() })
	if res := submitRegister(t, node, gen, addr, pub, priv, "android-mint-clamp", "[]"); res != "" {
		t.Fatalf("register: %s", res)
	}
	w := node.n.Ledger.State.Wallet(addr)
	if w == nil {
		t.Fatal("кошелёк не найден в реестре")
	}
	// инъекция активного кошелька с заданными M4/M5-полями (приём
	// injectActive из core-тестов: регистрация оставляет статус
	// registered до набора поручительств, полный vouch-путь здесь не нужен)
	w.Status = wire.StatusActive
	w.ActivatedPeriod = activated
	w.LastClaimedPeriod = last
	return &mintClampEnv{node: node, gen: gen, addr: addr, priv: priv}
}

// window — текущий ответ моста по окну выпуска.
func (e *mintClampEnv) window(t *testing.T) mintWindow {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(e.node.MintWindowJSON(e.addr)), &raw); err != nil {
		t.Fatal(err)
	}
	num := func(k string) uint64 {
		v, ok := raw[k].(float64)
		if !ok {
			t.Fatalf("%s нет в ответе моста: %v", k, raw[k])
		}
		return uint64(v)
	}
	open, _ := raw["open"].(bool)
	return mintWindow{
		claim:    num("claim_period"),
		amount:   num("claim_amount"),
		period:   num("current_period"),
		accrual:  num("accrual_window_periods"),
		issuance: num("issuance_amount"),
		last:     num("last_claimed_period"),
		open:     open,
	}
}

// submitMint — mint через мост (усиленный PoW и подпись ключом кошелька);
// возвращает "outcome/code" вердикта ядра.
func (e *mintClampEnv) submitMint(t *testing.T, period, amount uint64) string {
	t.Helper()
	w := e.node.n.Ledger.State.Wallet(e.addr)
	if w == nil {
		t.Fatal("кошелёк не найден в реестре")
	}
	tx := NewMint(e.gen.NetworkID(), e.addr, int64(w.Seq)+1, w.TipTxID,
		int64(period), int64(amount), e.gen.ProtocolVersion(), time.Now().Unix())
	if err := tx.SolveHard(e.gen.PowTargetHard(), e.gen.Argon2MemKiB(), e.gen.Argon2Iterations(), e.gen.Argon2Par()); err != nil {
		t.Fatal(err)
	}
	sig, err := SignWithKey(e.priv, tx.SigningHash())
	if err != nil {
		t.Fatal(err)
	}
	tx.SetSignature(sig)
	res, err := e.node.Submit(tx)
	if err != nil {
		t.Fatal(err)
	}
	return res.Outcome() + "/" + res.Code()
}

// TestMintClampClaimByAccrualWindow — (а) свежий кошелёк, (б) пропуск
// меньше accrual_window, (б2) пропуск ровно в окно, (в) пропуск больше
// окна: claim из моста равен максимуму, допустимому ядром, сумма — по M6
// от клампа, подача по claim проходит M5 (accept), а claim+1 ядро
// отвергает E_ACCRUAL_WINDOW.
func TestMintClampClaimByAccrualWindow(t *testing.T) {
	cases := []struct {
		name                    string
		period, activated, last uint64
		wantClaim, wantAmount   uint64
	}{
		// (а) свежий кошелёк: активирован в прошлом периоде, ничего не
		// заявлено — claim = period−1, кламп не срабатывает (поведение
		// до фикса сохранено)
		{"свежий кошелёк", 2, 1, 0, 1, 1000},
		// (б) пропуск 3 периода (< accrual_window = 6): claim = period−1,
		// кламп не связывает
		{"пропуск меньше окна", 5, 1, 1, 4, 3000},
		// (б2) граница: diff = accrual_window — ядро принимает (M5
		// отвергает только diff > accrual_window), кламп не связывает
		{"пропуск равен окну", 8, 1, 1, 7, 6000},
		// (в) пропуск 8 периодов (> окна): claim клампится до last + 6
		{"пропуск больше окна", 10, 1, 1, 7, 6000},
		// (в2) первый выпуск после долгого отсутствия: base =
		// activated_period − 1 = 0, claim = base + accrual_window
		{"первый выпуск после пропуска", 10, 1, 0, 6, 6000},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			e := newMintClampEnv(t, c.period, c.activated, c.last)
			mw := e.window(t)
			if mw.period != c.period {
				t.Fatalf("current_period = %d, хочу %d", mw.period, c.period)
			}
			if mw.accrual != 6 {
				t.Fatalf("accrual_window_periods = %d, хочу 6 (TestGenesis)", mw.accrual)
			}
			if mw.claim != c.wantClaim {
				t.Fatalf("claim_period = %d, хочу %d (кламп ядра)", mw.claim, c.wantClaim)
			}
			if mw.amount != c.wantAmount {
				t.Fatalf("claim_amount = %d, хочу %d (M6 от клампа)", mw.amount, c.wantAmount)
			}
			if !mw.open {
				t.Errorf("open = false: в песочнице окно челленджа открыто весь период")
			}
			// claim+1 — уже за окном накопления: вердикт живого ядра
			// (не повтор формулы в тесте) обязан быть E_ACCRUAL_WINDOW
			if c.wantClaim+1 < c.period {
				if res := e.submitMint(t, c.wantClaim+1, mw.amount+mw.issuance); res != "reject/E_ACCRUAL_WINDOW" {
					t.Fatalf("claim+1=%d: %s, хочу reject/E_ACCRUAL_WINDOW", c.wantClaim+1, res)
				}
			}
			// подача по UI-claim проходит M5
			if res := e.submitMint(t, mw.claim, mw.amount); res != "accept/OK" {
				t.Fatalf("подача по claim=%d: %s, хочу accept/OK", mw.claim, res)
			}
			if after := e.window(t); after.last != c.wantClaim {
				t.Fatalf("last_claimed_period = %d после подачи, хочу %d", after.last, c.wantClaim)
			}
		})
	}
}

// TestMintClampOldClaimRejected — воспроизведение долга D2: claim =
// period−1 (поведение моста до фикса) при пропуске 8 периодов ядро
// отвергает E_ACCRUAL_WINDOW — кнопка выпуска «отравлена» навсегда.
// После фикса мост предлагает base + accrual_window, и подача проходит.
func TestMintClampOldClaimRejected(t *testing.T) {
	const period = uint64(10)
	e := newMintClampEnv(t, period, 1, 1)
	// до фикса: claim = period−1 = 9, diff = 8 > 6
	if res := e.submitMint(t, period-1, 8000); res != "reject/E_ACCRUAL_WINDOW" {
		t.Fatalf("claim=%d (до фикса): %s, хочу reject/E_ACCRUAL_WINDOW", period-1, res)
	}
	mw := e.window(t)
	if mw.claim != 7 {
		t.Fatalf("claim_period = %d, хочу 7 (last + accrual_window)", mw.claim)
	}
	if res := e.submitMint(t, mw.claim, mw.amount); res != "accept/OK" {
		t.Fatalf("подача по claim=%d: %s, хочу accept/OK", mw.claim, res)
	}
}

// TestMintClampQuotaResetNoClamp — quota_reset: накопления нет (M5 не
// применяется, заявляется один период), кламп не должен обнулять claim —
// иначе квота не выпускается вовсе.
func TestMintClampQuotaResetNoClamp(t *testing.T) {
	const period = uint64(10)
	e := newMintClampEnv(t, period, 1, 1)
	prof := &e.node.n.Ledger.State.Net.Profile
	prof.IssuanceMode = wire.IssuanceQuotaReset
	prof.AccrualWindowPeriods = 0
	mw := e.window(t)
	if mw.claim != period-1 {
		t.Fatalf("claim_period = %d, хочу %d (quota_reset: клампа нет)", mw.claim, period-1)
	}
	if mw.amount != mw.issuance {
		t.Fatalf("claim_amount = %d, хочу %d (quota_reset: один период)", mw.amount, mw.issuance)
	}
}
