// Тесты фикса «слот от периода выпуска» (вариант A,
// REPORT-AUDIT-SONNET-2.md п. 3; тема fix-mint-window). Файл — из аудита
// Sonnet-2 (reports/audit-sonnet/go-tests/core_ledger_mint_window_fixed_test.go.txt),
// build-тег mintslotfix снят: поведение включено по умолчанию для новых
// сетей (манифест-поле 63 mint_slot_mode = 1), характеризационные тесты
// дофиксного поведения удалены из mint_window_prod_test.go.
//
// Тела тестов не менялись. Режим якоря задаёт хелпер newWindowEnv
// (mint_slot_mode = 1): в режиме 0 (наследие) эти тесты КРАСНЫЕ —
// см. mint_slot_mode_test.go, который фиксирует оба режима.
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Любой кошелёк, при любом положении слота внутри периода выпуска p, в
// собственном окне может выпустить завершённый период p−1.
func TestMintWindowFixed_IssuingPeriodReachableForAll(t *testing.T) {
	const q, p = uint64(5), uint64(6)
	classes := []struct {
		name string
		pred func(u uint64) bool
	}{
		{"слот в начале периода", func(u uint64) bool { return u < 2*86400 }},
		{"слот в середине периода", func(u uint64) bool { return u > 14*86400 && u < 16*86400 }},
		{"слот в конце периода", func(u uint64) bool { return u >= winT-2*86400 }},
	}
	for _, c := range classes {
		c := c
		t.Run(c.name, func(t *testing.T) {
			e := newWindowEnv(t)
			g, T, _ := e.winGeom()
			id := e.findWallet(p, c.pred) // слот периода ВЫПУСКА p
			e.injectActive(id, q, 0)
			at := mintChallengeSlot(g, e.addr(id), p, T)
			if res := e.tryMint(id, q, at); res.Outcome != Accepted {
				t.Fatalf("mint за период %d в слоте периода %d отклонён: %v/%s",
					q, p, res.Outcome, res.Code)
			}
		})
	}
}

// Пропущенные периоды наверстываются: wallet с base = 0 в окне периода 20
// выпускает q = K (diff = K) — поглощающее состояние сброшено.
func TestMintWindowFixed_AccrualCatchUpAfterMiss(t *testing.T) {
	e := newWindowEnv(t)
	g, T, _ := e.winGeom()
	prof := e.ledger.State.Net.Profile
	if prof.IssuanceMode == wire.IssuanceQuotaReset || prof.AccrualWindowPeriods < 2 {
		t.Skip("нужен периодный профиль с накоплением (K >= 2)")
	}
	K := prof.AccrualWindowPeriods
	const p = uint64(20)
	id := e.findWallet(p, func(u uint64) bool { return u > 10*86400 && u < 20*86400 })
	e.injectActive(id, 1, 0)
	at := mintChallengeSlot(g, e.addr(id), p, T)
	if res := e.tryMint(id, K, at); res.Outcome != Accepted {
		t.Fatalf("наверстывающий mint q=K отклонён: %v/%s", res.Outcome, res.Code)
	}
	// разность больше K по-прежнему недопустима (M5 не ослаблен)
	if res := e.tryMint(id, 2*K+1, at); res.Outcome != Rejected {
		t.Fatalf("q=2K+1 (diff=K+1) принят: %v/%s", res.Outcome, res.Code)
	}
}
