// Сеттлемент на границах эпох/периодов: демередж (10.3), истечение
// registered (7.3), освобождение слотов (7.4), seed_sunset (13),
// отложенное повышение P (8.3).
package ledger

import (
	"fmt"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func newSettleLedger(t *testing.T) (*Ledger, *FixedClock) {
	gp := params.TestGenesis()
	// демередж 1%/мес, floor 3000 (значения 1.2)
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &FixedClock{T: 1_790_000_000, E: 100, P: 16}
	l, err := New(gp, clock, Options{})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	// состояние на момент теста: периоды/эпохи уже материализованы
	l.State.CurrentPeriod = 16
	l.State.Epoch = 100
	return l, clock
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
	l, clock := newSettleLedger(t)
	// кошелёк с 10000: демередж 1% на (10000 − 3000) = 70 за период
	w := mkWallet(l, "RICH", wire.StatusActive, 10000)
	// кошелёк с 2000 (ниже floor): не демереджится
	poor := mkWallet(l, "POOR", wire.StatusActive, 2000)
	clock.P = 17
	l.syncTimeForTest()
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
	clock.P = 18
	l.syncTimeForTest()
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
	l, clock := newSettleLedger(t)
	w := mkWallet(l, "OLD", wire.StatusRegistered, 0)
	// 3 года без активации (7.3)
	w.RegisteredAt = clock.NetTime() - (3*365*86400 + 10)
	clock.E++
	l.syncTimeForTest()
	if w.Status != wire.StatusExpired {
		t.Fatalf("registered не истёк: %s", w.Status)
	}
}

func TestSettleSlotRelease(t *testing.T) {
	l, clock := newSettleLedger(t)
	v := mkWallet(l, "V", wire.StatusActive, 0)
	// слот с неактивированным кандидатом: 90 дней -> освобождение
	v.OccupiedSlots = append(v.OccupiedSlots, Slot{
		Candidate:    []byte("cand-1"),
		SinceNetTime: clock.NetTime() - 91*86400,
	})
	// слот с активированным кандидатом, завершившим 3 периода
	cand := mkWallet(l, "CAND", wire.StatusActive, 0)
	cand.LastClaimedPeriod = 15
	v.OccupiedSlots = append(v.OccupiedSlots, Slot{
		Candidate: cand.Address, SinceNetTime: clock.NetTime(),
		CandidateActive: true, MintsBase: 12,
	})
	if len(v.OccupiedSlots) != 2 {
		t.Fatal("слоты не заняты")
	}
	clock.E++
	l.syncTimeForTest()
	if len(v.OccupiedSlots) != 0 {
		t.Fatalf("слоты не освободились: %d", len(v.OccupiedSlots))
	}
}

func TestSettleSeedSunset(t *testing.T) {
	l, clock := newSettleLedger(t)
	// порог 100 active (решение №15, тестовый genesis) — заселим 100
	for i := 0; i < 100; i++ {
		mkWallet(l, fmt.Sprintf("S%d", i), wire.StatusActive, 0)
	}
	if !l.State.SeedRightsActive {
		t.Fatal("права посевного круга уже отключены")
	}
	clock.E++
	l.syncTimeForTest()
	if l.State.SeedRightsActive {
		t.Fatal("seed_sunset не сработал")
	}
}

func TestSettleThresholdRaise(t *testing.T) {
	l, clock := newSettleLedger(t)
	w := mkWallet(l, "T", wire.StatusActive, 0)
	w.PendingThreshold = 5000
	w.PendingThresholdNotBefore = clock.NetTime() + 48*3600
	clock.T += 49 * 3600
	l.syncTimeForTest()
	if w.DelayedThreshold != 5000 {
		t.Fatalf("повышение P не вступило: %d", w.DelayedThreshold)
	}
}
