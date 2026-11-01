// Сеттлемент на границах эпох/периодов: демередж (10.3), истечение
// registered (7.3), освобождение слотов (7.4), seed_sunset (13),
// отложенное повышение P (8.3).
//
// v9.13: сеттлемент детерминирован от времени сети (max(txNow) по
// журналу); genesis теста подобран так, что в момент T период = 16,
// эпоха = 100 (как pre-state теста), — тесты зовут settleTo напрямую
// (тот же примитив, что Apply принятых tx).
package ledger

import (
	"fmt"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// settle-тест: база времени (genesis → period 16 / epoch 100 при T)
const (
	settleGenesis   = 1_748_528_000    // T − 16 периодов (30 сут)
	settleEpochSecs = 41_472_000 / 100 // 16 периодов = 100 эпох ровно
	settleT         = 1_790_000_000
)

func newSettleLedger(t *testing.T) (*Ledger, uint64) {
	gp := params.TestGenesis()
	// демередж 1%/мес, floor 3000 (значения 1.2); длины эпохи/периода и
	// genesis_time — согласованы с pre-state (период 16, эпоха 100 при T)
	gp.Genesis.GenesisTime = settleGenesis
	gp.Manifest.Profile.EpochSeconds = settleEpochSecs
	gp.Manifest.Profile.PeriodSeconds = 30 * 86400
	wire.SetManifestHash(gp)
	l, err := New(gp, Options{})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	// состояние на момент теста: периоды/эпохи уже материализованы
	l.State.CurrentPeriod = 16
	l.State.Epoch = 100
	return l, settleT
}

// settleAtPeriod — сеттлемент к моменту внутри периода p.
func (l *Ledger) settleAtPeriod(p uint64) {
	l.settleTo(l.State.GenesisTime + p*30*86400 + 86400)
}

func mkWallet(l *Ledger, id string, status wire.WalletStatus, balance uint64) *Wallet {
	priv := crypto.DeterministicKey("settle:" + id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	addr := crypto.DeriveAddress(fp, l.State.Net.NetworkID)
	w := &Wallet{
		Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
		Status: status, Liquid: balance, Seed: true, // посевной круг: active из genesis (D8)
		RegisteredAt: l.State.NetTime, LastTouchEpoch: l.State.Epoch,
		DelayedThreshold: 3000,
		ActivatedPeriod:  10, LastClaimedPeriod: 12,
	}
	l.State.Wallets[string(addr)] = w
	l.State.FingerprintIndex[string(fp)] = addr
	l.State.TotalMinted += balance
	return w
}

func TestSettleDemurrage(t *testing.T) {
	l, now := newSettleLedger(t)
	// кошелёк с 10000: демередж 1% на (10000 − 3000) = 70 за период
	w := mkWallet(l, "RICH", wire.StatusActive, 10000)
	// кошелёк с 2000 (ниже floor): не демереджится
	poor := mkWallet(l, "POOR", wire.StatusActive, 2000)
	w.RegisteredAt, poor.RegisteredAt = now-400*86400, now-400*86400
	l.settleAtPeriod(17)
	if w.Liquid != 10000-70 {
		t.Fatalf("демередж: %d, ожидается %d", w.Liquid, 10000-70)
	}
	if poor.Liquid != 2000 {
		t.Fatalf("ниже floor демереджится: %d", poor.Liquid)
	}
	if l.State.TotalBurned != 70 {
		t.Fatalf("burned: %d, ожидается 70", l.State.TotalBurned)
	}
	// два периода: целочисленный burn = floor((base − floor) × rate / 10^6)
	// (детерминизм; эквивалент (1−r)^n с накоплением остатка)
	l.settleAtPeriod(18)
	base2 := uint64(9930)
	burn2 := (base2 - 3000) * 10000 / 1_000_000 // 69
	if w.Liquid != base2-burn2 {
		t.Fatalf("демередж 2 периода: %d, ожидается %d", w.Liquid, base2-burn2)
	}
	// INV-8 после сеттлемента
	if bad := l.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}

func TestSettleRegisteredExpiry(t *testing.T) {
	l, now := newSettleLedger(t)
	w := mkWallet(l, "OLD", wire.StatusRegistered, 0)
	// 3 года без активации (7.3)
	w.RegisteredAt = now - (3*365*86400 + 10)
	// одна эпоха вперёд (граница эпохи — момент сеттлемента)
	l.settleTo(l.State.GenesisTime + 101*settleEpochSecs)
	if w.Status != wire.StatusExpired {
		t.Fatalf("registered не истёк: %s", w.Status)
	}
}

func TestSettleSlotRelease(t *testing.T) {
	l, now := newSettleLedger(t)
	v := mkWallet(l, "V", wire.StatusActive, 0)
	v.RegisteredAt = now - 400*86400
	// слот с неактивированным кандидатом: 90 дней -> освобождение
	v.OccupiedSlots = append(v.OccupiedSlots, Slot{
		Candidate:    []byte("cand-1"),
		SinceNetTime: now - 91*86400,
	})
	// слот с активированным кандидатом, завершившим 3 периода
	cand := mkWallet(l, "CAND", wire.StatusActive, 0)
	cand.RegisteredAt = now - 400*86400
	cand.LastClaimedPeriod = 15
	v.OccupiedSlots = append(v.OccupiedSlots, Slot{
		Candidate: cand.Address, SinceNetTime: now,
		CandidateActive: true, MintsBase: 12,
	})
	if len(v.OccupiedSlots) != 2 {
		t.Fatal("слоты не заняты")
	}
	l.settleTo(l.State.GenesisTime + 101*settleEpochSecs)
	if len(v.OccupiedSlots) != 0 {
		t.Fatalf("слоты не освободились: %d", len(v.OccupiedSlots))
	}
}

func TestSettleSeedSunset(t *testing.T) {
	l, now := newSettleLedger(t)
	// порог 100 active (решение №15, тестовый genesis) — заселим 100
	for i := 0; i < 100; i++ {
		w := mkWallet(l, fmt.Sprintf("S%d", i), wire.StatusActive, 0)
		w.RegisteredAt = now - 400*86400
	}
	if !l.State.SeedRightsActive {
		t.Fatal("права посевного круга уже отключены")
	}
	l.settleTo(l.State.GenesisTime + 101*settleEpochSecs)
	if l.State.SeedRightsActive {
		t.Fatal("seed_sunset не сработал")
	}
}

func TestSettleThresholdRaise(t *testing.T) {
	l, now := newSettleLedger(t)
	w := mkWallet(l, "T", wire.StatusActive, 0)
	w.RegisteredAt = now - 400*86400
	w.PendingThreshold = 5000
	w.PendingThresholdNotBefore = now + 48*3600
	l.settleTo(now + 49*3600)
	if w.DelayedThreshold != 5000 {
		t.Fatalf("повышение P не вступило: %d", w.DelayedThreshold)
	}
}
