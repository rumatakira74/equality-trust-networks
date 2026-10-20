// Сеттлемент на границах эпох и периодов (spec/03 3.10, 3.6.2; spec/06).
// Детерминирован: результат зависит только от состояния и времени.
package ledger

import (
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// settleEpoch — проверки на границе эпохи: истечение registered (7.3),
// освобождение слотов (7.4), seed_sunset (13).
func (l *Ledger) settleEpoch(epoch, now uint64) {
	prof := l.State.Net.Profile
	// истечение registered: 3 года без активации (7.3)
	for _, w := range l.State.Wallets {
		if w.Status == wire.StatusRegistered && !w.Terminal {
			if now > w.RegisteredAt && now-w.RegisteredAt > prof.RegisteredExpiryDays*86400 {
				w.Status = wire.StatusExpired
			}
		}
	}
	// освобождение слотов (7.4): 3 периода выпуска поручаемого без evidence
	// или 90 дней без активации
	for _, v := range l.State.Wallets {
		keep := v.OccupiedSlots[:0]
		for _, s := range v.OccupiedSlots {
			cand := l.State.Wallet(s.Candidate)
			release := false
			if s.CandidateActive && cand != nil {
				if cand.LastClaimedPeriod >= s.MintsBase &&
					cand.LastClaimedPeriod-s.MintsBase >= uint64(prof.SlotFreeMintPeriods) {
					release = true
				}
			} else if !s.CandidateActive {
				if now >= s.SinceNetTime && now-s.SinceNetTime >= prof.SlotFreeDays*86400 {
					release = true
				}
			}
			if !release {
				keep = append(keep, s)
			}
		}
		v.OccupiedSlots = keep
	}
	// seed_sunset (13): порог числа active или срок
	if l.State.SeedRightsActive && l.seedSunsetMet(epoch) {
		l.State.SeedRightsActive = false
	}
}

func (l *Ledger) seedSunsetMet(epoch uint64) bool {
	// SeedSunset из манифеста (решение №15, утверждено 18.10.2026:
	// 100 active-кошельков ИЛИ 24 эпохи — срабатывает любое из условий)
	if l.SeedSunset == nil {
		return false
	}
	if l.SeedSunset.UseActiveWallets && l.SeedSunset.ActiveWallets > 0 {
		active := uint64(0)
		for _, w := range l.State.Wallets {
			if w.Status == wire.StatusActive {
				active++
			}
		}
		if active >= l.SeedSunset.ActiveWallets {
			return true
		}
	}
	return l.SeedSunset.DeadlineEpoch > 0 && epoch >= l.SeedSunset.DeadlineEpoch
}

// settlePeriod — демередж на границе периода (10.3; №31: n от последней
// операции; материализация каждый период эквивалентна (1−rate)^n).
// Ставка задана «в месяц» (10.3), поэтому якорь — период, не эпоха
// (интерпретация этапа 3; в spec «эпохи» — см. отчёт, отклонение D-2).
func (l *Ledger) settlePeriod(period, now uint64) {
	prof := l.State.Net.Profile
	if prof.DemurrageRateMicros == 0 {
		return
	}
	for _, w := range l.State.Wallets {
		base := w.TotalBalance() - w.Locked // pending-резерв не демереджится (№18)
		if base <= prof.DemurrageFloor {
			continue
		}
		burn := (base - prof.DemurrageFloor) * prof.DemurrageRateMicros / 1_000_000
		if burn == 0 {
			continue
		}
		// списание: сначала liquid, затем когорты
		if w.Liquid >= burn {
			w.Liquid -= burn
		} else {
			burn -= w.Liquid
			w.Liquid = 0
			for i := range w.Cohorts {
				if burn == 0 {
					break
				}
				if w.Cohorts[i].Amount <= burn {
					burn -= w.Cohorts[i].Amount
					w.Cohorts[i].Amount = 0
				} else {
					w.Cohorts[i].Amount -= burn
					burn = 0
				}
			}
			keep := w.Cohorts[:0]
			for _, c := range w.Cohorts {
				if c.Amount > 0 {
					keep = append(keep, c)
				}
			}
			w.Cohorts = keep
		}
		l.State.TotalBurned += burn
		w.LastTouchEpoch = l.State.Epoch
	}
}

// flushPending — исполнение сроков: отложенные переводы (8.3), миграции (11),
// повышения порога P (8.3).
func (l *Ledger) flushPending(now uint64) {
	prof := l.State.Net.Profile
	// отложенные переводы: исполнение через D без отмены
	for k, pt := range l.State.PendingTransfers {
		if pt.Canceled || pt.Executed || now < pt.Deadline {
			continue
		}
		w := l.State.Wallet(pt.Recipient)
		_ = w
		sender := l.State.WalletByKey(k) // ключ содержит адрес отправителя
		if sender == nil {
			continue
		}
		if sender.Locked < pt.Amount+pt.Fee {
			continue // инвариант нарушен бы — не исполняем
		}
		sender.Locked -= pt.Amount + pt.Fee
		if sender.Liquid >= pt.Amount+pt.Fee {
			sender.Liquid -= pt.Amount + pt.Fee
		} else {
			sender.Spend(pt.Amount+pt.Fee, l.State.CurrentPeriod, prof.FreshTransferWindowPeriods)
		}
		if rcpt := l.State.Wallet(pt.Recipient); rcpt != nil {
			rcpt.Liquid += pt.Amount
		}
		if prof.FeeMode == wire.FeeModeBurn {
			l.State.TotalBurned += pt.Fee
		} else if prof.FeeMode == wire.FeeModePool {
			l.State.CommonFund += pt.Fee // общий фонд сети (раздел 4)
		}
		pt.Executed = true
	}
	// миграции: исполнение через 14 дней без отмены (11)
	for _, pm := range l.State.PendingMigrations {
		if pm.Canceled || pm.Executed || now < pm.ExecuteNotBefore {
			continue
		}
		l.executeMigration(pm)
		pm.Executed = true
	}
	// повышения порога P: через 48 ч (8.3)
	for _, w := range l.State.Wallets {
		if w.PendingThreshold != 0 && now >= w.PendingThresholdNotBefore {
			w.DelayedThreshold = w.PendingThreshold
			w.PendingThreshold = 0
		}
	}
}

// executeMigration — перенос состояния при миграции (11; spec/03 3.6.2).
func (l *Ledger) executeMigration(pm *PendingMigration) {
	oldW := l.State.Wallet(pm.OldAddress)
	newW := l.State.Wallet(pm.NewAddress)
	if oldW == nil || newW == nil || oldW.Terminal {
		return
	}
	// переносятся: статус, возраст, история выпусков, last_claimed_period,
	// поручительства (полученные и занятые слоты — иначе миграция обходит
	// лимиты), метка flagged и приостановка. Баланс: в списке 3.6.2 не назван,
	// но перенос следует из назначения миграции (иначе она равна перезапуску,
	// где «средства теряются», 11) — интерпретация этапа 3, см. отчёт.
	newW.Status = oldW.Status
	newW.ActivatedAt = oldW.ActivatedAt
	newW.ActivatedEpoch = oldW.ActivatedEpoch
	newW.ActivatedPeriod = oldW.ActivatedPeriod
	newW.LastClaimedPeriod = oldW.LastClaimedPeriod
	newW.VouchesReceived = oldW.VouchesReceived
	newW.Activators = oldW.Activators
	newW.OccupiedSlots = oldW.OccupiedSlots
	newW.Flagged = oldW.Flagged
	newW.MintSuspendedUntilPeriod = oldW.MintSuspendedUntilPeriod
	newW.Liquid += oldW.Liquid
	newW.Cohorts = append(newW.Cohorts, oldW.Cohorts...)
	// старый адрес терминален; фингерпринт — навсегда в чёрный список (11)
	oldW.Terminal = true
	oldW.Status = wire.StatusExpired
	oldW.Liquid = 0
	oldW.Cohorts = nil
	oldW.OccupiedSlots = nil
	oldW.VouchesReceived = nil
	oldW.Activators = nil
	l.State.FingerprintBlacklist[string(oldW.Fingerprint)] = true
	delete(l.State.FingerprintIndex, string(oldW.Fingerprint))
	// гардианы: подопечные++; награда публичным — решение №29 (этап 5):
	// guardian_reward — доля в micros от месячного выпуска подопечного
	// (5% = 50 000), не чаще раза в guardian_reward_cooldown_days (12 мес.)
	// на подопечного (кулдаун по паре гардиан|подопечный).
	prof := l.State.Net.Profile
	for i := range pm.Approvals {
		gw := l.State.Wallet(pm.Approvals[i].Guardian)
		if gw == nil {
			continue
		}
		gw.Wards++
		if !gw.PublicGuardian || prof.GuardianReward == 0 {
			continue
		}
		coolKey := string(pm.Approvals[i].Guardian) + "|" + string(pm.OldAddress)
		if last, ok := l.State.GuardianLastReward[coolKey]; ok &&
			l.State.NetTime < last+prof.GuardianRewardCooldownDays*86400 {
			continue
		}
		reward := prof.IssuanceAmount * prof.GuardianReward / 1_000_000
		if reward == 0 {
			continue
		}
		gw.Liquid += reward
		l.State.TotalMinted += reward
		l.State.GuardianLastReward[coolKey] = l.State.NetTime
	}
}
