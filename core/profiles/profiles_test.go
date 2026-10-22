// Тесты шаблонов и валидации (раздел 4/5, 3.3; этап 10).
package profiles

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestTemplatesValid — все семь шаблонов проходят валидацию.
func TestTemplatesValid(t *testing.T) {
	for _, tpl := range All() {
		if bad := Validate(&tpl.Profile); len(bad) > 0 {
			t.Errorf("шаблон %d (%s): %v", tpl.ID, tpl.Name, bad)
		}
	}
}

// TestValidateRules — правила раздела 4: relaxed при высоких ставках,
// quota_reset с накоплением, heartbeat со стоимостью, демередж профилей
// 5/7, модули по единице.
func TestValidateRules(t *testing.T) {
	// relaxed при высокой ставке — запрещён
	p := ByID(7).Profile
	p.DevicePolicy = wire.DevicePolicyRelaxed
	if bad := Validate(&p); len(bad) == 0 {
		t.Fatalf("relaxed при 2%% принят")
	}
	// quota_reset с окном накопления — запрещён
	p = ByID(3).Profile
	p.AccrualWindowPeriods = 6
	if bad := Validate(&p); len(bad) == 0 {
		t.Fatalf("quota_reset с накоплением принят")
	}
	// heartbeat с комиссией — запрещён
	p = ByID(2).Profile
	p.FeeMode = wire.FeeModeBurn
	if bad := Validate(&p); len(bad) == 0 {
		t.Fatalf("heartbeat с комиссией принят")
	}
	// валюта без демереджа (не песочница) — запрещён (профили 5/7)
	p = ByID(7).Profile
	p.DemurrageRateMicros = 0
	if bad := Validate(&p); len(bad) == 0 {
		t.Fatalf("валюта без демереджа принята")
	}
	// pool_claim без модуля pool — запрещён
	p = ByID(4).Profile
	p.Modules = nil
	if bad := Validate(&p); len(bad) == 0 {
		t.Fatalf("pool_claim без модуля pool принят")
	}
	// invite_only_vouch с открытым доступом — запрещён
	p = ByID(3).Profile
	p.NetworkAccess = wire.NetworkAccessPublic
	if bad := Validate(&p); len(bad) == 0 {
		t.Fatalf("invite_only с public принят")
	}
}

// TestGuaranteeLevel — уровень гарантий из манифеста (3.3): учебный —
// sandbox; высокий — strict+decentralized+open_vouch; средний — relaxed
// или members_quorum.
func TestGuaranteeLevel(t *testing.T) {
	level, _ := GuaranteeLevel(&ByID(1).Profile)
	if level != "учебный" {
		t.Fatalf("лаборатория: %s, ожидается учебный", level)
	}
	level, _ = GuaranteeLevel(&ByID(7).Profile)
	if level != "высокий" {
		t.Fatalf("касса взаимопомощи: %s, ожидается высокий", level)
	}
	p := ByID(7).Profile
	p.DevicePolicy = wire.DevicePolicyRelaxed
	p.RateMax = 10000
	level, _ = GuaranteeLevel(&p)
	if level != "средний" {
		t.Fatalf("relaxed: %s, ожидается средний", level)
	}
	p = ByID(7).Profile
	p.CheckpointMode = wire.CheckpointModeMembersQuorum
	level, _ = GuaranteeLevel(&p)
	if level != "средний" {
		t.Fatalf("members_quorum: %s, ожидается средний", level)
	}
}

// TestTemplateRoundtrip — профиль шаблона переживает canonical-кодирование.
func TestTemplateRoundtrip(t *testing.T) {
	for _, tpl := range All() {
		dec, err := wire.DecodeProfileParams(tpl.Profile.Encode())
		if err != nil {
			t.Errorf("шаблон %d: decode: %v", tpl.ID, err)
			continue
		}
		if dec.Unit != tpl.Profile.Unit || dec.IssuanceMode != tpl.Profile.IssuanceMode ||
			dec.TransferMode != tpl.Profile.TransferMode {
			t.Errorf("шаблон %d: поля не пережили кодирование", tpl.ID)
		}
	}
}

// TestDemurrageFloorProfiles — floor демереджа шаблонов (v9.9, решение
// владельца 22.10.2026 по итогам симуляции блока 3): профиль 5 (hours,
// выпуск 10/мес) — 30 = 3 месячных выпуска (наследованные 3000 были
// мёртвым механизмом — 300 выпусков), профиль 7 (currency, 1000/мес) —
// 3000 без изменений.
func TestDemurrageFloorProfiles(t *testing.T) {
	if got := ByID(5).Profile.DemurrageFloor; got != 30 {
		t.Fatalf("floor профиля 5 = %d, хочу 30", got)
	}
	if got := ByID(7).Profile.DemurrageFloor; got != 3000 {
		t.Fatalf("floor профиля 7 = %d, хочу 3000", got)
	}
}

// TestFeeTargetGrowthTemplates — цель адаптивной комиссии во всех шаблонах
// конфигуратора — 15% за окно (v9.9, №44; ранее 6%).
func TestFeeTargetGrowthTemplates(t *testing.T) {
	for _, tpl := range All() {
		if got := tpl.Profile.FeeTargetGrowthMicros; got != 150000 {
			t.Errorf("шаблон %d: fee_target_growth_micros = %d, хочу 150000", tpl.ID, got)
		}
	}
}
