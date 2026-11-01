// Конвенция слота челленджа выпуска (правка этапа 8, долг этапа 7):
// слот отсчитывается от genesis-времени (9.2 — календарные периоды от
// genesis), а не от 1970 г.; окно ±MintChallengeWindowHours (7.6).
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func TestMintChallengeSlotFromGenesis(t *testing.T) {
	genesisTime := uint64(1_760_000_000)
	addr := make([]byte, 32)
	for _, period := range []uint64{0, 1, 2, 7} {
		slot := mintChallengeSlot(genesisTime, addr, period, 30*86400)
		start := genesisTime + period*30*86400
		if slot < start || slot >= start+30*86400 {
			t.Fatalf("период %d: слот %d вне периода [%d, %d)",
				period, slot, start, start+30*86400)
		}
	}
}

// период заданной длины (genesis-поле 61, тестовые сети живого прогона):
// слот масштабируется длиной периода, окно ±48 ч всегда накрывает
// заявляемый завершённый период
func TestMintChallengeSlotShortPeriod(t *testing.T) {
	genesisTime := uint64(1_760_000_000)
	addr := make([]byte, 32)
	const periodSecs = 120
	for _, period := range []uint64{0, 1, 2, 1000} {
		slot := mintChallengeSlot(genesisTime, addr, period, periodSecs)
		start := genesisTime + period*periodSecs
		if slot < start || slot >= start+periodSecs {
			t.Fatalf("период %d: слот %d вне периода [%d, %d)",
				period, slot, start, start+periodSecs)
		}
	}
}

func TestCheckMintChallengeWindowFromGenesis(t *testing.T) {
	gp := params.TestGenesis()
	gp.Genesis.GenesisTime = 1_760_000_000 // genesis в 2020-х, не 1970
	l, err := New(gp, Options{})
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	addr := []byte{1, 2, 3}
	period := uint64(2)
	slot := mintChallengeSlot(gp.Genesis.GenesisTime, addr, period, 30*86400)
	if !l.CheckMintChallengeWindow(addr, period, slot) {
		t.Fatal("окно должно быть открыто в слоте")
	}
	if !l.CheckMintChallengeWindow(addr, period, slot+47*3600) {
		t.Fatal("окно должно быть открыто внутри ±48 ч")
	}
	if l.CheckMintChallengeWindow(addr, period, slot-49*3600) {
		t.Fatal("окно должно быть закрыто до слота")
	}
	if l.CheckMintChallengeWindow(addr, period, slot+49*3600) {
		t.Fatal("окно должно быть закрыто после слота")
	}
	// при genesis-времени в 2020-х слот не выпадает в 1970-е
	// (старая конвенция: period×30 сут от 1970 — окно всегда закрыто)
	if slot < gp.Genesis.GenesisTime {
		t.Fatal("слот до genesis-времени (конвенция 1970 не исправлена)")
	}
}

// TestMintChallengeWindowShortPeriodsSeedReachability — блокер 2 прогона
// 12-testnet (v9.13, этап 15): генезис в прошлом (95 сут — возраст
// посевного круга), period_seconds = 120 с. Слот челленджа
// масштабируется длиной периода (поле 61): слот периода P лежит строго
// внутри периода P. Первые заявляемые периоды посевного круга
// (квоты с периода 1, quota_reset) лежат у genesis: окно шаблона ±48 ч
// их не накрывает, достижимость задаёт поле 51
// mint_challenge_window_hours (прогон: 2400 ч). Текущий период
// достижим всегда: слот внутри него, окно ±48 ч накрывает.
func TestMintChallengeWindowShortPeriodsSeedReachability(t *testing.T) {
	now := uint64(1767000000) // фиксированное «сейчас» (детерминизм)
	genesis := now - 95*86400
	const periodSecs = 120
	addr := []byte("seed-wallet")

	mk := func(windowHours uint64) *Ledger {
		gp := params.TestGenesis()
		gp.Genesis.GenesisTime = genesis
		gp.Manifest.Profile.PeriodSeconds = periodSecs // поле 61
		gp.Manifest.Profile.MintChallengeWindowHours = windowHours
		wire.SetManifestHash(gp)
		l, err := New(gp, Options{})
		if err != nil {
			t.Fatalf("ledger: %v", err)
		}
		return l
	}

	// слот масштабируется period_seconds: слот периода P — внутри P
	cur := (now - genesis) / periodSecs
	for _, p := range []uint64{1, 2, cur - 1, cur} {
		slot := mintChallengeSlot(genesis, addr, p, periodSecs)
		start := genesis + p*periodSecs
		if slot < start || slot >= start+periodSecs {
			t.Fatalf("период %d: слот %d вне периода [%d, %d)",
				p, slot, start, start+periodSecs)
		}
	}

	// окно шаблона ±48 ч: текущий период достижим (слот внутри него)
	l48 := mk(48)
	if !l48.CheckMintChallengeWindow(addr, cur, now) {
		t.Fatal("слот текущего периода недостижим с окном ±48 ч")
	}
	// ...но первый заявляемый период (1) у genesis — недостижим
	if l48.CheckMintChallengeWindow(addr, 1, now) {
		t.Fatal("слот периода 1 (у genesis) достижим с окном ±48 ч — окно не работает")
	}

	// поле 51: окно 2400 ч (100 сут) накрывает слот периода 1 —
	// достижимость первых заявляемых периодов посевного круга
	l2400 := mk(2400)
	if !l2400.CheckMintChallengeWindow(addr, 1, now) {
		t.Fatal("слот периода 1 недостижим с окном 2400 ч (поле 51)")
	}
	// claim обязан лежать внутри claim-окна слота: окно проверяется
	// от txNow = claim (не от часов узла)
	slot1 := mintChallengeSlot(genesis, addr, 1, periodSecs)
	if !l2400.CheckMintChallengeWindow(addr, 1, slot1) {
		t.Fatal("claim в слоте обязан проходить окно")
	}
	win := uint64(2400 * 3600)
	if l2400.CheckMintChallengeWindow(addr, 1, slot1-win-1) {
		t.Fatal("claim до окна слота прошёл — окно не от claim")
	}
	if l2400.CheckMintChallengeWindow(addr, 1, slot1+win+1) {
		t.Fatal("claim после окна слота прошёл — окно не от claim")
	}
}
