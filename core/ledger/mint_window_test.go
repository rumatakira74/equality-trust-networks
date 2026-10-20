// Конвенция слота челленджа выпуска (правка этапа 8, долг этапа 7):
// слот отсчитывается от genesis-времени (9.2 — календарные периоды от
// genesis), а не от 1970 г.; окно ±MintChallengeWindowHours (7.6).
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/params"
)

func TestMintChallengeSlotFromGenesis(t *testing.T) {
	genesisTime := uint64(1_760_000_000)
	addr := make([]byte, 32)
	for _, period := range []uint64{0, 1, 2, 7} {
		slot := mintChallengeSlot(genesisTime, addr, period)
		start := genesisTime + period*30*86400
		if slot < start || slot >= start+30*86400 {
			t.Fatalf("период %d: слот %d вне периода [%d, %d)",
				period, slot, start, start+30*86400)
		}
	}
}

func TestCheckMintChallengeWindowFromGenesis(t *testing.T) {
	gp := params.TestGenesis()
	gp.Genesis.GenesisTime = 1_760_000_000 // genesis в 2020-х, не 1970
	l, err := New(gp, &FixedClock{}, Options{})
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	addr := []byte{1, 2, 3}
	period := uint64(2)
	slot := mintChallengeSlot(gp.Genesis.GenesisTime, addr, period)
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
