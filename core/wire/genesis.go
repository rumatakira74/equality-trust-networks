// Genesis и манифест: proto/genesis.proto (spec/02). Каноническое кодирование
// нужно для manifest_hash = SHA-256(canonical(Manifest)) и проверки genesis.
package wire

import "crypto/sha256"

type UnitType uint32

const (
	UnitUnspecified UnitType = 0
	UnitCurrency    UnitType = 1
	UnitHours       UnitType = 2
	UnitQuota       UnitType = 3
	UnitPoints      UnitType = 4
	UnitVoucher     UnitType = 5
	UnitHeartbeat   UnitType = 6
)

type TransferMode uint32

const (
	TransferModeUnspecified TransferMode = 0
	TransferModeFree        TransferMode = 1
	TransferModeRestricted  TransferMode = 2
	TransferModeSoulbound   TransferMode = 3
)

type IssuanceMode uint32

const (
	IssuanceUnspecified    IssuanceMode = 0
	IssuancePeriodicEqual  IssuanceMode = 1
	IssuanceQuotaReset     IssuanceMode = 2
	IssuanceOnContribution IssuanceMode = 3
	IssuancePoolClaim      IssuanceMode = 4
	IssuanceHeartbeat      IssuanceMode = 5
	IssuanceNone           IssuanceMode = 6
)

type FeeMode uint32

const (
	FeeModeUnspecified FeeMode = 0
	FeeModeBurn        FeeMode = 1
	FeeModePool        FeeMode = 2
	FeeModeNone        FeeMode = 3
)

type EntryMode uint32

const (
	EntryUnspecified     EntryMode = 0
	EntryOpenVouch       EntryMode = 1
	EntryFoundersSeed    EntryMode = 2
	EntryInviteOnlyVouch EntryMode = 3
)

type DevicePolicy uint32

const (
	DevicePolicyUnspecified DevicePolicy = 0
	DevicePolicyStrict      DevicePolicy = 1
	DevicePolicyRelaxed     DevicePolicy = 2
)

type SigningAuth uint32

const (
	SigningAuthUnspecified      SigningAuth = 0
	SigningAuthBiometricStrong  SigningAuth = 1
	SigningAuthDeviceCredential SigningAuth = 2
)

type NetworkAccess uint32

const (
	NetworkAccessUnspecified NetworkAccess = 0
	NetworkAccessPublic      NetworkAccess = 1
	NetworkAccessInviteOnly  NetworkAccess = 2
)

type CheckpointMode uint32

const (
	CheckpointModeUnspecified   CheckpointMode = 0
	CheckpointModeDecentralized CheckpointMode = 1
	CheckpointModeMembersQuorum CheckpointMode = 2
)

type TimeScale uint32

const (
	TimeScaleUnspecified TimeScale = 0
	TimeScaleReal        TimeScale = 1
	TimeScaleAccelerated TimeScale = 2
)

// SlotTier — пул слотов поручителя по возрасту (7.4).
type SlotTier struct {
	MinAgeDays uint64
	Slots      uint32
}

func (p *SlotTier) Encode() []byte {
	var b []byte
	b = appendU64Field(b, 1, p.MinAgeDays)
	b = appendU32Field(b, 2, p.Slots)
	return b
}

// PowParams — параметры PoW (7.6, 17; значения — предположение, вопрос №9).
type PowParams struct {
	TargetLight           uint64 // таргет лёгкого PoW: хеш (BE u64) < target
	Argon2MemoryKiB       uint32
	Argon2Iterations      uint32
	Argon2Parallelism     uint32
	TargetHard            uint64
	RepeatAndroidIDTarget uint64
}

func (p *PowParams) Encode() []byte {
	var b []byte
	b = appendU64Field(b, 1, p.TargetLight)
	b = appendU32Field(b, 2, p.Argon2MemoryKiB)
	b = appendU32Field(b, 3, p.Argon2Iterations)
	b = appendU32Field(b, 4, p.Argon2Parallelism)
	b = appendU64Field(b, 5, p.TargetHard)
	b = appendU64Field(b, 6, p.RepeatAndroidIDTarget)
	return b
}

// MutableBound — границы изменяемого параметра (4, 13).
type MutableBound struct {
	Param string
	Min   uint64
	Max   uint64
}

func (p *MutableBound) Encode() []byte {
	var b []byte
	b = appendStringField(b, 1, p.Param)
	b = appendU64Field(b, 2, p.Min)
	b = appendU64Field(b, 3, p.Max)
	return b
}

// SeedMember — участник посевного круга.
type SeedMember struct {
	Address   []byte
	PublicKey []byte
}

func (p *SeedMember) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.Address)
	b = appendBytesField(b, 2, p.PublicKey)
	return b
}

// SeedSunset — правило самоликвидации посевного круга (13; решение №15,
// утверждено 18.10.2026): срабатывает ЛЮБОЕ из заданных условий
// (100 active-кошельков или 24 эпохи); 0 — условие не задано.
type SeedSunset struct {
	UseActiveWallets bool
	ActiveWallets    uint64
	DeadlineEpoch    uint64
}

func (p *SeedSunset) Encode() []byte {
	var b []byte
	// оба условия кодируются независимо (решение №15: ИЛИ, не ИЛИ-ИЛИ)
	if p.UseActiveWallets && p.ActiveWallets > 0 {
		b = appendU64Field(b, 1, p.ActiveWallets)
	}
	if p.DeadlineEpoch > 0 {
		b = appendU64Field(b, 2, p.DeadlineEpoch)
	}
	return b
}

// ProfileParams — параметры профиля (раздел 4; proto/genesis.proto).
type ProfileParams struct {
	Unit                       UnitType
	ResourceID                 string
	TransferMode               TransferMode
	IssuanceMode               IssuanceMode
	IssuanceAmount             uint64
	IssuancePeriodMonths       uint32
	AccrualWindowPeriods       uint64
	FreshTransferWindowPeriods uint64 // вопрос №11, закрыт этапом 1.2
	FeeMode                    FeeMode
	FeeMin                     uint64
	RateMin                    uint64 // micros: 1% = 10000
	RateMax                    uint64
	DemurrageRateMicros        uint64
	DemurrageFloor             uint64
	EntryMode                  EntryMode
	VouchK                     uint32
	VoucherMinAgeDays          uint64
	VoucherMinMintPeriods      uint32
	SlotTiers                  []SlotTier
	SeedSlotCount              uint32
	SlotFreeMintPeriods        uint32
	SlotFreeDays               uint64
	VoucherRewardMicros        uint64
	VoucherRewardPeriods       uint32
	DevicePolicy               DevicePolicy
	SigningAuth                SigningAuth
	NetworkAccess              NetworkAccess
	CheckpointMode             CheckpointMode
	EpochSeconds               uint64
	Sandbox                    bool
	TimeScale                  TimeScale
	ParamsMutable              bool
	MutableBounds              []MutableBound
	Modules                    []uint32
	Providers                  [][]byte
	TransferMoratoriumDays     uint64
	TransferDelayHours         uint64
	DelayedThresholdDefault    uint64
	DelayedThresholdMax        uint64
	RegisteredExpiryDays       uint64
	GuardianPersonalThreshold  uint64
	GuardianPersonalCount      uint64
	GuardianPublicThreshold    uint64
	GuardianPublicCount        uint64
	GuardianMinAgeDays         uint64
	GuardianMaxWards           uint64
	GuardianPersonalMaxWards   uint64
	GuardianReward             uint64
	GuardianRewardCooldownDays uint64
	// EclipseN — подтверждение данных от N несвязанных пиров (поле 54;
	// вопрос №35, этап 5); дефолт 3.
	EclipseN uint64
	// ValidatorsM — размер выборки валидаторов эпохи в decentralized
	// (поле 55; решение №17, утверждено 18.10.2026: M = 21,
	// порог N = ceil(2/3 M)); 0 — дефолт 21.
	ValidatorsM uint64
	// CpBuilderQueue — очередь строителей чекпоинта (поле 56; №42,
	// решение владельца (A), этап 9): первые K участников состава,
	// ротация через epoch/2; 0 — дефолт 3. Значение черновое — как у
	// соседних констант (eclipse_n = 3), симуляцией не подбирали.
	CpBuilderQueue            uint64
	MigrateDelayDays          uint64
	TimestampToleranceSeconds uint64
	MintChallengeWindowHours  uint64
	Pow                       *PowParams
	// FeeAdjustEpochs — окно K эпох адаптивной комиссии (поле 57; 10.2,
	// этап 10): на границе эпохи rate пересчитывается по росту предложения
	// за последние K эпох. 0 — адаптивный пересчёт выключен (rate
	// постоянен, равен rate_min; векторы этапа 0).
	FeeAdjustEpochs uint64
	// FeeTargetGrowthMicros — целевой рост предложения за K эпох в micros
	// (поле 58; 10.2: «выше целевого — rate растёт на шаг, иначе снижается»).
	FeeTargetGrowthMicros uint64
	// FeeStepMicros — шаг пересчёта rate в micros (поле 59).
	FeeStepMicros uint64
}

func (p *ProfileParams) Encode() []byte {
	var b []byte
	b = appendU32Field(b, 1, uint32(p.Unit))
	b = appendStringField(b, 2, p.ResourceID)
	b = appendU32Field(b, 3, uint32(p.TransferMode))
	b = appendU32Field(b, 4, uint32(p.IssuanceMode))
	b = appendU64Field(b, 5, p.IssuanceAmount)
	b = appendU32Field(b, 6, p.IssuancePeriodMonths)
	b = appendU64Field(b, 7, p.AccrualWindowPeriods)
	b = appendU32Field(b, 24, uint32(p.DevicePolicy))
	b = appendU32Field(b, 25, uint32(p.SigningAuth))
	b = appendU32Field(b, 26, uint32(p.NetworkAccess))
	b = appendU32Field(b, 27, uint32(p.CheckpointMode))
	b = appendU64Field(b, 28, p.EpochSeconds)
	b = appendBoolField(b, 29, p.Sandbox)
	b = appendU32Field(b, 30, uint32(p.TimeScale))
	b = appendBoolField(b, 31, p.ParamsMutable)
	for i := range p.MutableBounds {
		b = appendMessageField(b, 32, p.MutableBounds[i].Encode())
	}
	for _, m := range p.Modules {
		b = appendU32Field(b, 33, m)
	}
	for _, pr := range p.Providers {
		b = appendBytesField(b, 34, pr)
	}
	b = appendU64Field(b, 35, p.TransferMoratoriumDays)
	b = appendU64Field(b, 36, p.TransferDelayHours)
	b = appendU64Field(b, 37, p.DelayedThresholdDefault)
	b = appendU64Field(b, 38, p.DelayedThresholdMax)
	b = appendU64Field(b, 39, p.RegisteredExpiryDays)
	b = appendU64Field(b, 40, p.GuardianPersonalThreshold)
	b = appendU64Field(b, 41, p.GuardianPersonalCount)
	b = appendU64Field(b, 42, p.GuardianPublicThreshold)
	b = appendU64Field(b, 43, p.GuardianPublicCount)
	b = appendU64Field(b, 44, p.GuardianMinAgeDays)
	b = appendU64Field(b, 45, p.GuardianMaxWards)
	b = appendU64Field(b, 46, p.GuardianPersonalMaxWards)
	b = appendU64Field(b, 47, p.GuardianReward)
	b = appendU64Field(b, 48, p.GuardianRewardCooldownDays)
	b = appendU64Field(b, 54, p.EclipseN)
	b = appendU64Field(b, 55, p.ValidatorsM)
	b = appendU64Field(b, 56, p.CpBuilderQueue)
	b = appendU64Field(b, 57, p.FeeAdjustEpochs)
	b = appendU64Field(b, 58, p.FeeTargetGrowthMicros)
	b = appendU64Field(b, 59, p.FeeStepMicros)
	b = appendU64Field(b, 49, p.MigrateDelayDays)
	b = appendU64Field(b, 50, p.TimestampToleranceSeconds)
	b = appendU64Field(b, 51, p.MintChallengeWindowHours)
	if p.Pow != nil {
		b = appendMessageField(b, 52, p.Pow.Encode())
	}
	b = appendU64Field(b, 53, p.FreshTransferWindowPeriods)
	// Поля 8–23 (комиссия, демередж, вход, поручительства) — в порядке номеров:
	b = appendU32Field(b, 8, uint32(p.FeeMode))
	b = appendU64Field(b, 9, p.FeeMin)
	b = appendU64Field(b, 10, p.RateMin)
	b = appendU64Field(b, 11, p.RateMax)
	b = appendU64Field(b, 12, p.DemurrageRateMicros)
	b = appendU64Field(b, 13, p.DemurrageFloor)
	b = appendU32Field(b, 14, uint32(p.EntryMode))
	b = appendU32Field(b, 15, p.VouchK)
	b = appendU64Field(b, 16, p.VoucherMinAgeDays)
	b = appendU32Field(b, 17, p.VoucherMinMintPeriods)
	for i := range p.SlotTiers {
		b = appendMessageField(b, 18, p.SlotTiers[i].Encode())
	}
	b = appendU32Field(b, 19, p.SeedSlotCount)
	b = appendU32Field(b, 20, p.SlotFreeMintPeriods)
	b = appendU64Field(b, 21, p.SlotFreeDays)
	b = appendU64Field(b, 22, p.VoucherRewardMicros)
	b = appendU32Field(b, 23, p.VoucherRewardPeriods)
	return b
}

// Manifest — манифест сети (3.3, 4).
type Manifest struct {
	NetworkID   []byte
	DisplayName string
	Profile     *ProfileParams
	SeedCircle  []SeedMember
	SeedSunset  *SeedSunset
	GenesisHash []byte
}

func (m *Manifest) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, m.NetworkID)
	b = appendStringField(b, 2, m.DisplayName)
	if m.Profile != nil {
		b = appendMessageField(b, 3, m.Profile.Encode())
	}
	for i := range m.SeedCircle {
		b = appendMessageField(b, 4, m.SeedCircle[i].Encode())
	}
	if m.SeedSunset != nil {
		b = appendMessageField(b, 5, m.SeedSunset.Encode())
	}
	b = appendBytesField(b, 6, m.GenesisHash)
	return b
}

// ManifestHash = SHA-256(canonical(Manifest)) (3.3).
func (m *Manifest) ManifestHash() [32]byte {
	return sha256.Sum256(m.Encode())
}

// GenesisRecord — genesis-запись (3.1, 3.3).
type GenesisRecord struct {
	NetworkID       []byte
	ManifestHash    []byte
	GenesisTime     uint64
	EpochZero       uint64
	ProtocolVersion uint32
	SeedSunset      *SeedSunset
	InviteKey       []byte
}

func (g *GenesisRecord) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, g.NetworkID)
	b = appendBytesField(b, 2, g.ManifestHash)
	b = appendU64Field(b, 3, g.GenesisTime)
	b = appendU64Field(b, 4, g.EpochZero)
	b = appendU32Field(b, 5, g.ProtocolVersion)
	if g.SeedSunset != nil {
		b = appendMessageField(b, 6, g.SeedSunset.Encode())
	}
	b = appendBytesField(b, 7, g.InviteKey)
	return b
}

// GenesisPackage — манифест + genesis-запись.
type GenesisPackage struct {
	Manifest *Manifest
	Genesis  *GenesisRecord
}
