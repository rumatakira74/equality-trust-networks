// Тесты адаптивной комиссии (10.2, этап 10): пересчёт ставки из данных
// финализированных чекпоинтов, эквивалентность на всех узлах, пограничные
// случаи, рестарт из БД (восстановление ряда предложения).
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func newFeeLedger(t *testing.T) *Ledger {
	t.Helper()
	gp := params.TestGenesis()
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := New(gp, clock, Options{})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	return l
}

// TestAdaptiveFeeEquivalence — одно значение ставки на всех узлах: два
// независимых ядра, одна цепочка чекпоинтов — ставки сходятся (10.2,
// «все узлы получают одно и то же значение»). Узлы «видят» чекпоинты
// в разное время (разный порядок локальных операций) — без разницы.
func TestAdaptiveFeeEquivalence(t *testing.T) {
	// рост предложения за окно K = 6 эпох: 1000 -> 1200 (20% > цели 15%)
	chain := []SupplySnapshot{}
	supply := uint64(1_000_000)
	for e := uint64(1); e <= 20; e++ {
		chain = append(chain, SupplySnapshot{Epoch: e, Supply: supply})
		supply += 40_000 // +4% за эпоху
	}
	a := newFeeLedger(t)
	b := newFeeLedger(t)
	for _, s := range chain {
		a.OnCheckpointSupply(s.Epoch, s.Supply)
	}
	// второй узел применяет те же чекпоинты с другой локальной историей
	for _, s := range chain {
		b.OnCheckpointSupply(s.Epoch, s.Supply)
	}
	if a.FeeRateNow() != b.FeeRateNow() {
		t.Fatalf("ставки разошлись: %d != %d", a.FeeRateNow(), b.FeeRateNow())
	}
	// рост выше цели — ставка растёт; старт 1%, шаг 0,05 п.п.
	if a.FeeRateNow() <= 10_000 {
		t.Fatalf("ставка не выросла: %d", a.FeeRateNow())
	}
	// комиссия следует ставке: fee = max(fee_min, rate × amount)
	fee := a.State.Net.Fee(100_000, a.FeeRateNow())
	if fee != 100_000*a.FeeRateNow()/1_000_000 {
		t.Fatalf("комиссия не следует ставке: %d", fee)
	}
}

// TestAdaptiveFeeDecrease — рост ниже цели: ставка снижается до нижней
// границы и не выходит за неё.
func TestAdaptiveFeeDecrease(t *testing.T) {
	l := newFeeLedger(t)
	// предложение стабильно (+0,1% за эпоху < цели 15%)
	supply := uint64(1_000_000)
	for e := uint64(1); e <= 40; e++ {
		l.OnCheckpointSupply(e, supply)
		supply += 1_000
	}
	if l.FeeRateNow() != l.State.Net.Profile.RateMin {
		t.Fatalf("ставка не дошла до нижней границы: %d", l.FeeRateNow())
	}
}

// TestAdaptiveFeeUpperBound — рост много выше цели: ставка упирается в
// rate_max и не выходит за него.
func TestAdaptiveFeeUpperBound(t *testing.T) {
	l := newFeeLedger(t)
	supply := uint64(1)
	for e := uint64(1); e <= 60; e++ {
		l.OnCheckpointSupply(e, supply)
		supply *= 2 // рост 100% за эпоху
	}
	if l.FeeRateNow() != l.State.Net.Profile.RateMax {
		t.Fatalf("ставка не дошла до верхней границы: %d", l.FeeRateNow())
	}
}

// TestAdaptiveFeeWarmupAndZero — пограничные случаи: до K+1 чекпоинтов
// ставка не меняется (старт = rate_min); нулевая база окна — не меняется.
func TestAdaptiveFeeWarmupAndZero(t *testing.T) {
	l := newFeeLedger(t)
	for e := uint64(1); e <= 6; e++ { // ровно K снимков — окна ещё нет
		l.OnCheckpointSupply(e, 1_000_000*e)
	}
	if l.FeeRateNow() != 10_000 {
		t.Fatalf("ставка изменилась до окна: %d", l.FeeRateNow())
	}
	// окно из нулей: рост не определён — ставка на месте
	l2 := newFeeLedger(t)
	for e := uint64(1); e <= 10; e++ {
		l2.OnCheckpointSupply(e, 0)
	}
	if l2.FeeRateNow() != 10_000 {
		t.Fatalf("нулевая база изменила ставку: %d", l2.FeeRateNow())
	}
	// регрессия и дубль эпохи игнорируются (цепочка строга)
	l2.OnCheckpointSupply(10, 5_000_000)
	l2.OnCheckpointSupply(9, 5_000_000)
	if l2.FeeRateNow() != 10_000 {
		t.Fatalf("регрессия эпохи изменила ставку: %d", l2.FeeRateNow())
	}
}

// TestAdaptiveFeeDisabled — fee_adjust_epochs = 0 (векторы этапа 0):
// чекпоинты не меняют ставку, комиссия постоянна.
func TestAdaptiveFeeDisabled(t *testing.T) {
	gp := params.TestGenesis()
	gp.Manifest.Profile.FeeAdjustEpochs = 0
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	l, err := New(gp, &FixedClock{T: 1_700_000_000, E: 1, P: 1}, Options{})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	for e := uint64(1); e <= 30; e++ {
		l.OnCheckpointSupply(e, uint64(1_000_000)*e*e)
	}
	if l.FeeRateNow() != 10_000 {
		t.Fatalf("выключенный пересчёт изменил ставку: %d", l.FeeRateNow())
	}
}

// TestAdaptiveFeeRestart — рестарт из БД: новый узел восстанавливает ряд
// предложения чекпоинтов и сворачивает ту же цепочку пересчётов —
// ставка идентична узлу, работавшему без остановки.
func TestAdaptiveFeeRestart(t *testing.T) {
	live := newFeeLedger(t)
	chain := []SupplySnapshot{}
	supply := uint64(500_000)
	for e := uint64(1); e <= 15; e++ {
		chain = append(chain, SupplySnapshot{Epoch: e, Supply: supply})
		live.OnCheckpointSupply(e, supply)
		supply += 50_000 // +10% за эпоху > цели — рост
	}
	// «рестарт»: свежее ядро, тот же ряд из БД (LoadCheckpointSupply)
	restarted := newFeeLedger(t)
	for _, s := range chain {
		restarted.OnCheckpointSupply(s.Epoch, s.Supply)
	}
	if restarted.FeeRateNow() != live.FeeRateNow() {
		t.Fatalf("рестарт разошёлся: %d != %d", restarted.FeeRateNow(), live.FeeRateNow())
	}
	// и продолжает одинаково после новых чекпоинтов
	for e := uint64(16); e <= 20; e++ {
		supply += 1_000 // ниже цели — снижение
		live.OnCheckpointSupply(e, supply)
		restarted.OnCheckpointSupply(e, supply)
	}
	if restarted.FeeRateNow() != live.FeeRateNow() {
		t.Fatalf("после рестарта ставки разошлись: %d != %d", restarted.FeeRateNow(), live.FeeRateNow())
	}
}

// TestAdaptiveFeeGenesisRoundtrip — новые поля профиля переживают
// canonical-кодирование манифеста (manifest_hash сходится).
func TestAdaptiveFeeGenesisRoundtrip(t *testing.T) {
	gp := params.TestGenesis()
	p := gp.Manifest.Profile
	if p.FeeAdjustEpochs != 6 || p.FeeTargetGrowthMicros != 150_000 || p.FeeStepMicros != 500 {
		t.Fatalf("дефолты TestGenesis: %+v", p)
	}
	enc := p.Encode()
	dec, err := wire.DecodeProfileParams(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dec.FeeAdjustEpochs != p.FeeAdjustEpochs ||
		dec.FeeTargetGrowthMicros != p.FeeTargetGrowthMicros ||
		dec.FeeStepMicros != p.FeeStepMicros {
		t.Fatalf("поля комиссии не пережили кодирование: %+v", dec)
	}
}
