// Параметры сети: значения по умолчанию для тестового genesis.
//
// Источники (правило: в коде — из манифеста, не зашиты; здесь только дефолты):
//   - зафиксировано этапом 1.1 (не трогать): K = 3, слоты 3/5/10, R = 10%,
//     посевной круг >= 300 (раздел 17);
//   - выбрано этапом 1.2 (equality-anti-farm-simulation/Mistral/REPORT.md, раздел 5):
//     демередж 1%/мес + floor 3000; комиссия fee_min = 1, [0,5%; 2%], старт
//     1%, fee_mode = BURN; окно переводимости 3 мес.; мораторий 14 дней;
//     P = 3000, P_max = 30000;
//   - решено №9 (утверждено 19.10.2026): таргеты PoW; MaxPatchAgeMonths —
//     предположение этапа 3.
package params

import (
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Network — параметры сети, извлечённые из манифеста (spec/02) и дополненные
// производными величинами. Ядро читает только эту структуру.
type Network struct {
	NetworkID       []byte
	Profile         wire.ProfileParams
	ProtocolVersion uint32

	// Производные (вычисляются из Profile при загрузке genesis):
	EpochSeconds uint64
	SlotTiers    []wire.SlotTier
	Pow          wire.PowParams
	// FlaggedMintSuspensionPeriods — приостановка выпуска при flagged
	// (7.5): 3 мес., при повторе — навсегда. Решение №12 утверждено
	// пользователем (этап 5); повтор — в applyEvidence (core/ledger).
	FlaggedMintSuspensionPeriods uint64
	// EclipseN — подтверждение данных от N несвязанных пиров (12.3).
	// Решение №35 утверждено (этап 5): N = 3, параметр genesis-профиля.
	EclipseN uint64
	// MaxPatchAgeMonths — «не просроченный уровень патча» (7.2);
	// предположение этапа 3 (в Документе числа нет).
	MaxPatchAgeMonths int
	// MinSecurityLevel — минимальный аппаратный уровень (strict: TEE/StrongBox).
	MinSecurityLevel int
}

// FromManifest строит Network из манифеста (значения — из манифеста, не зашиты).
func FromManifest(m *wire.Manifest, protocolVersion uint32) *Network {
	n := &Network{
		NetworkID:       m.NetworkID,
		Profile:         *m.Profile,
		ProtocolVersion: protocolVersion,
		EpochSeconds:    m.Profile.EpochSeconds,
		SlotTiers:       m.Profile.SlotTiers,
	}
	if m.Profile.Pow != nil {
		n.Pow = *m.Profile.Pow
	}
	// Приостановка выпуска при flagged: 3 мес., при повторе — навсегда
	// (решение №12, этап 5).
	n.FlaggedMintSuspensionPeriods = 3
	// Eclipse N (решение №35, этап 5): дефолт 3.
	n.EclipseN = m.Profile.EclipseN
	if n.EclipseN == 0 {
		n.EclipseN = 3
	}
	// Уровень патча не старше 24 мес. — предположение (вопросов нет, но числа
	// в Документе тоже нет; помечено в отчёте этапа 3).
	n.MaxPatchAgeMonths = 24
	// strict -> TEE и выше (7.2: StrongBox/TEE).
	n.MinSecurityLevel = 1
	if m.Profile.DevicePolicy == wire.DevicePolicyRelaxed {
		n.MinSecurityLevel = 0
	}
	return n
}

// SlotCapacity — размер пула слотов поручителя по возрасту (7.4, 17).
func (n *Network) SlotCapacity(ageDays uint64, seed bool, seedAgeDays uint64) uint32 {
	if seed && seedAgeDays < 365 {
		return n.Profile.SeedSlotCount // 10 в первый год
	}
	// tiers упорядочены по возрастанию min_age; берём последний подходящий
	cap := uint32(0)
	for _, t := range n.SlotTiers {
		if ageDays >= t.MinAgeDays {
			cap = t.Slots
		}
	}
	return cap
}

// Fee — комиссия на движение (10.2): fee = max(fee_min, rate × amount).
// rate — в micros (1% = 10000); fee сжигается при fee_mode = BURN.
func (n *Network) Fee(amount uint64) uint64 {
	if n.Profile.FeeMode == wire.FeeModeNone {
		return 0
	}
	// Текущая ставка: адаптивный пересчёт на границе эпохи — этап 5;
	// детерминированная заглушка этапа 3 — константа RateMin (старт 1%, 1.2).
	fee := amount * n.Profile.RateMin / 1_000_000
	if fee < n.Profile.FeeMin {
		fee = n.Profile.FeeMin
	}
	return fee
}

// TestGenesis — тестовый genesis со значениями по умолчанию:
// 1.1 (зафиксировано) + 1.2 (REPORT.md, раздел 5) + предположения №9.
func TestGenesis() *wire.GenesisPackage {
	networkID := []byte("tnp-testnet-0001") // 16 байт для тестов; в проде 32
	profile := wire.ProfileParams{
		Unit:                       wire.UnitCurrency,
		TransferMode:               wire.TransferModeFree,
		IssuanceMode:               wire.IssuancePeriodicEqual,
		IssuanceAmount:             1000,
		IssuancePeriodMonths:       1,
		AccrualWindowPeriods:       6,
		FreshTransferWindowPeriods: 3, // 1.2, вопрос №11
		FeeMode:                    wire.FeeModeBurn,
		FeeMin:                     1,     // 1.2
		RateMin:                    5000,  // 0,5% (1.2)
		RateMax:                    20000, // 2% (1.2)
		DemurrageRateMicros:        10000, // 1%/мес (1.2)
		DemurrageFloor:             3000,  // 3 месячных выпуска (1.2)
		EntryMode:                  wire.EntryOpenVouch,
		VouchK:                     3, // 1.1
		VoucherMinAgeDays:          90,
		VoucherMinMintPeriods:      2,
		SlotTiers: []wire.SlotTier{ // 1.1: 3 при 3–12 мес., 5 старше
			{MinAgeDays: 90, Slots: 3},
			{MinAgeDays: 365, Slots: 5},
		},
		SeedSlotCount:              10, // 1.1: посевному кругу в первый год
		SlotFreeMintPeriods:        3,
		SlotFreeDays:               90,
		VoucherRewardMicros:        100000, // R = 10% (1.1)
		VoucherRewardPeriods:       12,     // 1.1: первые 12 выпусков
		DevicePolicy:               wire.DevicePolicyStrict,
		SigningAuth:                wire.SigningAuthBiometricStrong,
		NetworkAccess:              wire.NetworkAccessPublic,
		CheckpointMode:             wire.CheckpointModeDecentralized,
		EpochSeconds:               86400, // решение №16 (18.10.2026): сутки; песочница — time_scale
		Sandbox:                    false,
		TimeScale:                  wire.TimeScaleReal,
		ParamsMutable:              false,
		TransferMoratoriumDays:     14,    // 1.2
		TransferDelayHours:         48,    // D = 48 ч, фикс. (8.3)
		DelayedThresholdDefault:    3000,  // P (1.2, из удобства)
		DelayedThresholdMax:        30000, // P_max (1.2)
		RegisteredExpiryDays:       3 * 365,
		GuardianPersonalThreshold:  2,
		GuardianPersonalCount:      3,
		GuardianPublicThreshold:    3,
		GuardianPublicCount:        5,
		GuardianMinAgeDays:         365,
		GuardianMaxWards:           20,
		GuardianPersonalMaxWards:   3,     // вопрос №7: предположение K = 3
		GuardianReward:             50000, // решение №29 (этап 5): 5% месячного выпуска подопечного
		EclipseN:                   3,     // решение №35 (этап 5)
		ValidatorsM:                21,    // решение №17 (18.10.2026): M = 21, N = ceil(2/3·21) = 14
		GuardianRewardCooldownDays: 365,
		MigrateDelayDays:           14,
		TimestampToleranceSeconds:  1800, // ±30 мин (9.2)
		MintChallengeWindowHours:   48,   // ±48 ч (7.6)
		Pow: &wire.PowParams{
			// Решение №9 (утверждено 19.10.2026, замеры —
			// reports/stage6/pow-host.md): лёгкий PoW SHA-256 с таргетом
			// 2^24 (~0,6 с хост / ~1–4 с телефон); усиленный Argon2id
			// 32 МиБ × 1 итерация с таргетом 2^56 (~64 итерации:
			// ~5 с хост / ~10–20 с телефон) — «под средний телефон» (7.6).
			// В тестах genesis переопределяется на малые таргеты.
			TargetLight:           0x0000_0100_0000_0000, // 2^24
			Argon2MemoryKiB:       32768,
			Argon2Iterations:      1,
			Argon2Parallelism:     1,
			TargetHard:            0x0040_0000_0000_0000, // 2^56
			RepeatAndroidIDTarget: 0,
		},
	}
	// Стартовая ставка 1% (1.2); поле RateMin несёт текущее значение ставки
	// (адаптивный пересчёт на границе эпохи — этап 5, детерминированная
	// заглушка этапа 3: константа).
	profile.RateMin = 10000
	manifest := &wire.Manifest{
		NetworkID:   networkID,
		DisplayName: "TNP Test Genesis",
		Profile:     &profile,
		SeedSunset:  &wire.SeedSunset{UseActiveWallets: true, ActiveWallets: 100, DeadlineEpoch: 24}, // решение №15 (18.10.2026)
	}
	mh := manifest.ManifestHash()
	genesis := &wire.GenesisRecord{
		NetworkID:       networkID,
		ManifestHash:    mh[:],
		GenesisTime:     0,
		EpochZero:       0,
		ProtocolVersion: 1,
		SeedSunset:      manifest.SeedSunset,
	}
	return &wire.GenesisPackage{Manifest: manifest, Genesis: genesis}
}
