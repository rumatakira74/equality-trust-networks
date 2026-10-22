// Пакет profiles — шаблоны семи профилей (KNOWLEDGE 5, этап 10) и их
// валидация (раздел 4), вычисление уровня гарантий из манифеста (3.3).
//
// Шаблон — стартовая точка конфигуратора сети: организатор без
// программирования выбирает сценарий, задаёт имя и параметры посевного
// круга; конфигуратор собирает genesis-пакет и печатает уровень гарантий.
//
// Числа, которых Документ не задаёт (ставки демереджа профилей 5/7,
// размеры квот/выпусков конкретных программ), — черновые значения
// раздела 17, помечены в описании; живые сети подбирают их симуляцией.
package profiles

import (
	"fmt"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Template — шаблон профиля сети (раздел 5).
type Template struct {
	ID          int
	Name        string // латиницей — для конфигуратора
	DisplayName string
	Description string
	Profile     wire.ProfileParams
}

// base — параметры по умолчанию («Валютия», раздел 4/17): значения
// зафиксированы этапами 1.1/1.2 и решениями №9/№15–№17.
func base() wire.ProfileParams {
	return wire.ProfileParams{
		Unit:                       wire.UnitCurrency,
		TransferMode:               wire.TransferModeFree,
		IssuanceMode:               wire.IssuancePeriodicEqual,
		IssuanceAmount:             1000,
		IssuancePeriodMonths:       1,
		AccrualWindowPeriods:       6,
		FreshTransferWindowPeriods: 3,
		FeeMode:                    wire.FeeModeBurn,
		FeeMin:                     1,
		RateMin:                    5000,
		RateMax:                    20000,
		DemurrageRateMicros:        0,
		DemurrageFloor:             3000,
		EntryMode:                  wire.EntryOpenVouch,
		VouchK:                     3,
		VoucherMinAgeDays:          90,
		VoucherMinMintPeriods:      2,
		SlotTiers: []wire.SlotTier{
			{MinAgeDays: 90, Slots: 3},
			{MinAgeDays: 365, Slots: 5},
		},
		SeedSlotCount:              10,
		SlotFreeMintPeriods:        3,
		SlotFreeDays:               90,
		VoucherRewardMicros:        100000,
		VoucherRewardPeriods:       12,
		DevicePolicy:               wire.DevicePolicyStrict,
		SigningAuth:                wire.SigningAuthBiometricStrong,
		NetworkAccess:              wire.NetworkAccessPublic,
		CheckpointMode:             wire.CheckpointModeDecentralized,
		EpochSeconds:               86400,
		Sandbox:                    false,
		TimeScale:                  wire.TimeScaleReal,
		ParamsMutable:              false,
		TransferMoratoriumDays:     14,
		TransferDelayHours:         48,
		DelayedThresholdDefault:    3000,
		DelayedThresholdMax:        30000,
		RegisteredExpiryDays:       3 * 365,
		GuardianPersonalThreshold:  2,
		GuardianPersonalCount:      3,
		GuardianPublicThreshold:    3,
		GuardianPublicCount:        5,
		GuardianMinAgeDays:         365,
		GuardianMaxWards:           20,
		GuardianPersonalMaxWards:   3,
		GuardianReward:             50000,
		GuardianRewardCooldownDays: 365,
		EclipseN:                   3,
		ValidatorsM:                21,
		CpBuilderQueue:             3,
		MigrateDelayDays:           14,
		TimestampToleranceSeconds:  1800,
		MintChallengeWindowHours:   48,
		FeeAdjustEpochs:            6,
		FeeTargetGrowthMicros:      150000,
		FeeStepMicros:              500,
		Pow: &wire.PowParams{
			TargetLight:      0x0000_0100_0000_0000,
			Argon2MemoryKiB:  32768,
			Argon2Iterations: 1,
			Argon2Parallelism: 1,
			TargetHard:       0x0040_0000_0000_0000,
		},
	}
}

// All — семь шаблонов (раздел 5, таблица профилей).
func All() []*Template {
	t := func(id int, name, display, desc string, mod func(*wire.ProfileParams)) *Template {
		p := base()
		mod(&p)
		return &Template{ID: id, Name: name, DisplayName: display, Description: desc, Profile: p}
	}
	return []*Template{
		t(1, "laboratory", "Лаборатория (образование)",
			"Учебная сеть: points, ускоренные эпохи, relaxed-устройства, синтетические кошельки; параметры — часть учебного сценария",
			func(p *wire.ProfileParams) {
				p.Unit = wire.UnitPoints
				p.Sandbox = true
				p.TimeScale = wire.TimeScaleAccelerated
				p.EpochSeconds = 300 // ускоренные эпохи (учебный прогон)
				p.EntryMode = wire.EntryFoundersSeed
				p.VouchK = 2
				p.DevicePolicy = wire.DevicePolicyRelaxed
				p.SigningAuth = wire.SigningAuthDeviceCredential
				p.FeeMode = wire.FeeModeNone
				p.DemurrageRateMicros = 0
				p.FreshTransferWindowPeriods = 0 // полный путь за один прогон
				p.Modules = []uint32{wire.ModuleSandbox, wire.ModulePool, wire.ModuleContribution}
			}),
		t(2, "humanness", "Доказательство человечности",
			"Ежемесячный «пульс» с усиленным PoW копит стаж; единицы не передаются и не имеют стоимости; предъявление через attest",
			func(p *wire.ProfileParams) {
				p.Unit = wire.UnitHeartbeat
				p.IssuanceMode = wire.IssuanceHeartbeat
				p.IssuanceAmount = 1 // счётчик пульса
				p.TransferMode = wire.TransferModeSoulbound
				p.FeeMode = wire.FeeModeNone
				p.DemurrageRateMicros = 0
				p.Modules = []uint32{wire.ModuleAttest}
			}),
		t(3, "quota", "Равные квоты на ресурс",
			"Квота на ресурс сгорает в конце периода; передача только поставщику; двойная квота = выгода сибиллы — strict",
			func(p *wire.ProfileParams) {
				p.Unit = wire.UnitQuota
				p.ResourceID = "resource" // организатор задаёт свой resource_id
				p.IssuanceMode = wire.IssuanceQuotaReset
				p.AccrualWindowPeriods = 0
				p.TransferMode = wire.TransferModeRestricted
				p.EntryMode = wire.EntryInviteOnlyVouch
				p.NetworkAccess = wire.NetworkAccessInviteOnly
				p.FeeMode = wire.FeeModeNone
				p.DemurrageRateMicros = 0
				p.Modules = []uint32{wire.ModuleRedeem}
			}),
		t(4, "voucher", "Волонтёрская / уличная программа",
			"Ваучеры программы: равные доли из бюджета (pool_claim), передача только партнёрской точке; участники — со смартфоном",
			func(p *wire.ProfileParams) {
				p.Unit = wire.UnitVoucher
				p.ResourceID = "program" // организатор задаёт program_id
				p.IssuanceMode = wire.IssuancePoolClaim
				p.TransferMode = wire.TransferModeRestricted
				p.EntryMode = wire.EntryInviteOnlyVouch
				p.NetworkAccess = wire.NetworkAccessInviteOnly
				p.FeeMode = wire.FeeModeNone
				p.DemurrageRateMicros = 0
				p.Modules = []uint32{wire.ModulePool, wire.ModuleRedeem}
			}),
		t(5, "community", "Внутренняя валюта / банк времени",
			"hours: малые равные выпуски, свободная передача, демередж включён, минимальная комиссия",
			func(p *wire.ProfileParams) {
				p.Unit = wire.UnitHours
				p.IssuanceAmount = 10 // малые суммы (банк времени)
				p.DemurrageRateMicros = 10000 // 1%/мес — обязателен (10.3, 10.4)
				p.DemurrageFloor = 30         // 3 месячных выпуска (v9.9: решение владельца 22.10.2026 по итогам блока 3)
				p.RateMax = 10000             // комиссия минимальная
				p.Modules = []uint32{wire.ModulePool}
			}),
		t(6, "contribution", "Учёт волонтёрского вклада",
			"points за подтверждённый вклад: peer-подтверждения, выплата из пула программы, soulbound",
			func(p *wire.ProfileParams) {
				p.Unit = wire.UnitPoints
				p.IssuanceMode = wire.IssuanceOnContribution
				p.IssuanceAmount = 100 // потолок на человека за период (предположение)
				p.TransferMode = wire.TransferModeSoulbound
				p.EntryMode = wire.EntryInviteOnlyVouch
				p.NetworkAccess = wire.NetworkAccessInviteOnly
				p.FeeMode = wire.FeeModeNone
				p.DemurrageRateMicros = 0
				p.Modules = []uint32{wire.ModuleContribution, wire.ModuleRedeem, wire.ModulePool}
			}),
		t(7, "mutual-aid", "Касса взаимопомощи",
			"currency: равные выпуски, демередж + сжигаемая комиссия, сборы через pool, предъявление стажа через attest",
			func(p *wire.ProfileParams) {
				p.DemurrageRateMicros = 10000 // 1%/мес — обязателен (10.3, 10.4)
				p.Modules = []uint32{wire.ModulePool, wire.ModuleAttest}
			}),
	}
}

// ByID — шаблон по номеру (1–7); nil — нет такого.
func ByID(id int) *Template {
	for _, t := range All() {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// ByName — шаблон по имени.
func ByName(name string) *Template {
	for _, t := range All() {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// Validate — валидация параметров профиля (раздел 4 + защиты 3.3/10.3).
// Возвращает список нарушений (пустой — профиль корректен).
func Validate(p *wire.ProfileParams) []string {
	var bad []string
	add := func(format string, a ...any) {
		bad = append(bad, fmt.Sprintf(format, a...))
	}
	if p.RateMin > p.RateMax {
		add("rate_min %d > rate_max %d", p.RateMin, p.RateMax)
	}
	if p.FeeMode != wire.FeeModeNone && p.FeeMin == 0 {
		add("fee_min = 0 при включённой комиссии (анти-спам, 10.2)")
	}
	if len(p.SlotTiers) == 0 {
		add("slot_tiers пусты (7.4)")
	}
	if p.VouchK < 1 {
		add("vouch_k < 1")
	}
	// relaxed — только для сетей с низкими ставками (раздел 4: device_policy);
	// при выключенной комиссии ставки не действуют
	if p.DevicePolicy == wire.DevicePolicyRelaxed && p.FeeMode != wire.FeeModeNone &&
		p.RateMax > 10000 {
		add("relaxed-устройства при ставке до %d micros (допустимо только для сетей с низкими ставками, раздел 4)", p.RateMax)
	}
	// quota_reset: окно накопления 0 (раздел 4), ресурс задан
	if p.IssuanceMode == wire.IssuanceQuotaReset {
		if p.AccrualWindowPeriods != 0 {
			add("quota_reset с accrual_window %d (раздел 4: для quota_reset — 0)", p.AccrualWindowPeriods)
		}
		if p.Unit == wire.UnitQuota && p.ResourceID == "" {
			add("unit quota без resource_id")
		}
	}
	// демередж обязателен для передаваемых единиц обмена (10.3/10.4:
	// профили 5 и 7), кроме песочницы (учебный сценарий)
	if !p.Sandbox && (p.Unit == wire.UnitHours || p.Unit == wire.UnitCurrency) &&
		p.DemurrageRateMicros == 0 {
		add("unit hours/currency без демереджа (10.3: обязателен для профилей 5 и 7)")
	}
	// heartbeat: единицы без стоимости — комиссии и демереджа нет
	if p.Unit == wire.UnitHeartbeat {
		if p.FeeMode != wire.FeeModeNone || p.DemurrageRateMicros != 0 {
			add("heartbeat со стоимостью (fee/demurrage должны быть выключены)")
		}
		if p.TransferMode != wire.TransferModeSoulbound {
			add("heartbeat не soulbound (2: единицы не передаются)")
		}
	}
	// soulbound: несовместим с pool_claim (доля пула передаётся на кошелёк —
	// это выпуск, не перевод; допустимо), но redeem требует restricted-путь
	if p.TransferMode == wire.TransferModeSoulbound && p.IssuanceMode == wire.IssuancePoolClaim {
		// soulbound + pool_claim: доля приходит выпуском — допустимо
	}
	// модули соответствуют единице и режиму выпуска (раздел 5)
	has := func(m uint32) bool {
		for _, x := range p.Modules {
			if x == m {
				return true
			}
		}
		return false
	}
	if p.IssuanceMode == wire.IssuancePoolClaim && !has(wire.ModulePool) {
		add("issuance pool_claim без модуля pool")
	}
	if p.IssuanceMode == wire.IssuanceOnContribution && !has(wire.ModuleContribution) {
		add("issuance on_contribution без модуля contribution")
	}
	if p.Unit == wire.UnitQuota && !has(wire.ModuleRedeem) {
		add("unit quota без модуля redeem")
	}
	if p.Unit == wire.UnitVoucher && !has(wire.ModuleRedeem) {
		add("unit voucher без модуля redeem")
	}
	// песочница: параметры неизменны (13) — проверяется и в ledger.New
	if p.Sandbox && p.ParamsMutable {
		add("sandbox с params_mutable = true (13)")
	}
	// invite_only_vouch — закрытый доступ
	if p.EntryMode == wire.EntryInviteOnlyVouch && p.NetworkAccess != wire.NetworkAccessInviteOnly {
		add("invite_only_vouch с открытым доступом (4: network_access)")
	}
	return bad
}

// GuaranteeLevel — уровень гарантий вычисляется из манифеста, а не
// заявляется (3.3): высокий — strict + decentralized + open_vouch;
// средний — relaxed или members_quorum (или вход по приглашению —
// интерпретация этапа 10: закрытый круг ниже высокого); учебный —
// sandbox. Возвращает уровень и строку-обоснование.
func GuaranteeLevel(p *wire.ProfileParams) (string, string) {
	if p.Sandbox {
		return "учебный", "sandbox = true (3.3): сеть не связана с внешней ценностью"
	}
	high := p.DevicePolicy == wire.DevicePolicyStrict &&
		p.CheckpointMode == wire.CheckpointModeDecentralized &&
		p.EntryMode == wire.EntryOpenVouch
	if high {
		return "высокий", "strict + decentralized + open_vouch (3.3)"
	}
	switch {
	case p.DevicePolicy == wire.DevicePolicyRelaxed:
		return "средний", "relaxed-устройства (3.3)"
	case p.CheckpointMode == wire.CheckpointModeMembersQuorum:
		return "средний", "members_quorum (3.3)"
	default:
		return "средний", "вход по приглашению — закрытый круг (интерпретация этапа 10: ниже высокого)"
	}
}
