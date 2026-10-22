// Пакет tnp — расширение моста для UI приложения (этап 7).
// Тонкие обёртки read-запросов net/query.go под ограничения gomobile
// (int64/[]byte/string; списки и структуры — JSON-снапшотами) и
// конструкторы транзакций, которых не было в этапе 6: vouch, отмена
// отложенного перевода, смена порога P, guardian_offer, migrate.
//
// Параметры сети НЕ хардкодятся в приложении: всё читается из
// genesis-пакета (ManifestJSON) и состояния узла.
package tnp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// ---------------------------------------------------------------------------
// Конструкторы транзакций (этап 7)
// ---------------------------------------------------------------------------

// NewRegisterGuardians — register с личными гардианами (11: назначаются
// заранее, при создании кошелька; максимум 3; вопрос №6 — обновление
// списка вне register не кодируется в v1).
func NewRegisterGuardians(networkID, wallet, pub, chain, challenge []byte,
	androidID string, guardiansJSON string, protocolVersion, ts int64) (*Tx, error) {
	var guardians [][]byte
	if guardiansJSON != "" {
		var hexes []string
		if err := json.Unmarshal([]byte(guardiansJSON), &hexes); err != nil {
			return nil, fmt.Errorf("гардианы: %w", err)
		}
		for _, h := range hexes {
			g, err := hex.DecodeString(h)
			if err != nil {
				return nil, fmt.Errorf("гардиан %s: %w", h, err)
			}
			guardians = append(guardians, g)
		}
	}
	if len(guardians) > 3 {
		return nil, errors.New("гардианов не больше 3")
	}
	tx := newTx(networkID, wallet, 1, make([]byte, 32), wire.Register,
		&wire.RegisterPayload{
			AttestedPublicKey:    append([]byte(nil), pub...),
			AttestationChain:     append([]byte(nil), chain...),
			AttestationChallenge: append([]byte(nil), challenge...),
			AndroidID:            androidID,
			PersonalGuardians:    guardians,
		}, protocolVersion, ts)
	return tx, nil
}

// NewVouch — поручительство (7.4): подписывается поручителем, содержит
// адрес кандидата и эпоху подачи (обязана быть текущей, V8).
func NewVouch(networkID, wallet []byte, seq int64, prev, candidate []byte,
	epoch, protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, seq, prev, wire.Vouch,
		&wire.VouchPayload{
			Candidate: append([]byte(nil), candidate...),
			Epoch:     uint64(epoch),
		}, protocolVersion, ts)
}

// NewTransferCancel — отмена отложенного перевода в течение D (8.3;
// кодирование в TRANSFER — решение D2, вопрос №4).
func NewTransferCancel(networkID, wallet []byte, seq int64, prev []byte,
	pendingSeq, protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, seq, prev, wire.Transfer,
		&wire.TransferPayload{Cancel: &wire.TransferCancel{
			PendingSeq: uint64(pendingSeq),
		}}, protocolVersion, ts)
}

// NewSetThreshold — смена личного порога P (8.3; вопрос №3): повышение
// вступает через 48 ч, понижение — сразу; в пределах P_max.
func NewSetThreshold(networkID, wallet []byte, seq int64, prev []byte,
	newThreshold, protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, seq, prev, wire.Transfer,
		&wire.TransferPayload{SetThreshold: &wire.ThresholdUpdate{
			NewThreshold: uint64(newThreshold),
		}}, protocolVersion, ts)
}

// NewGuardianOffer — объявление себя публичным гардианом (11): само
// наличие транзакции = объявление; условия (active, ≥ 12 мес.) — в ядре.
func NewGuardianOffer(networkID, wallet []byte, seq int64, prev []byte,
	protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, seq, prev, wire.GuardianOffer,
		&wire.GuardianOfferPayload{}, protocolVersion, ts)
}

// NewMigratePropose — предложение миграции в цепочке НОВОГО кошелька
// (11). approvalsJSON: [{"guardian":"hex","sig":"hex"}, ...] — подписи
// гардианов над MigrateStatement (MigrateStatementHash), собранные
// заранее (гардиан подписывает в своём приложении).
func NewMigratePropose(networkID, newWallet []byte, seq int64, prev,
	oldAddress []byte, approvalsJSON string, nonce, protocolVersion,
	ts int64) (*Tx, error) {
	var approvals []wire.GuardianApproval
	if approvalsJSON != "" {
		var raw []struct {
			Guardian string `json:"guardian"`
			Sig      string `json:"sig"`
		}
		if err := json.Unmarshal([]byte(approvalsJSON), &raw); err != nil {
			return nil, fmt.Errorf("approvals: %w", err)
		}
		for _, a := range raw {
			g, err := hex.DecodeString(a.Guardian)
			if err != nil {
				return nil, fmt.Errorf("гардиан: %w", err)
			}
			s, err := hex.DecodeString(a.Sig)
			if err != nil {
				return nil, fmt.Errorf("подпись: %w", err)
			}
			approvals = append(approvals, wire.GuardianApproval{
				Guardian: g, StatementSig: s,
			})
		}
	}
	tx := newTx(networkID, newWallet, seq, prev, wire.Migrate,
		&wire.MigratePayload{Propose: &wire.MigratePropose{
			OldAddress: append([]byte(nil), oldAddress...),
			Approvals:  approvals,
			Nonce:      uint64(nonce),
		}}, protocolVersion, ts)
	return tx, nil
}

// NewMigrateCancel — отмена миграции СТАРЫМ ключом в цепочке старого
// кошелька (11; кодирование — решение D3, вопрос №34).
func NewMigrateCancel(networkID, oldWallet []byte, seq int64, prev,
	newAddress []byte, proposeSeq, protocolVersion, ts int64) *Tx {
	return newTx(networkID, oldWallet, seq, prev, wire.Migrate,
		&wire.MigratePayload{Cancel: &wire.MigrateCancel{
			NewAddress: append([]byte(nil), newAddress...),
			ProposeSeq: uint64(proposeSeq),
		}}, protocolVersion, ts)
}

// MigrateStatementHash — хеш утверждения миграции, который подписывает
// гардиан (своим приложением, биометрией; 11). Совпадает с ядром:
// wire.MigrateStatement.Hash() над {network_id, old, new, nonce, epoch}.
func MigrateStatementHash(networkID, oldAddress, newAddress []byte,
	nonce, epoch int64) []byte {
	stmt := &wire.MigrateStatement{
		NetworkID:  append([]byte(nil), networkID...),
		OldAddress: append([]byte(nil), oldAddress...),
		NewAddress: append([]byte(nil), newAddress...),
		Nonce:      uint64(nonce),
		Epoch:      uint64(epoch),
	}
	h := stmt.Hash()
	return h[:]
}

// NewUpdateGuardiansPropose — перевыбор личных гардианов (11, решение
// владельца 21.10.2026): публикуется в цепочке владельца, подпись —
// текущим ключом. guardiansJSON: ["hex", ...] (≤ 3); approvalsJSON:
// [{"guardian":"hex","sig":"hex"}, ...] — подписи новых гардианов над
// хешем смены (GuardianChangeStatementHash), собранные вручную.
func NewUpdateGuardiansPropose(networkID, wallet []byte, seq int64, prev []byte,
	guardiansJSON, approvalsJSON string, nonce, protocolVersion,
	ts int64) (*Tx, error) {
	var guardians [][]byte
	if guardiansJSON != "" {
		var hexes []string
		if err := json.Unmarshal([]byte(guardiansJSON), &hexes); err != nil {
			return nil, fmt.Errorf("гардианы: %w", err)
		}
		for _, h := range hexes {
			g, err := hex.DecodeString(h)
			if err != nil {
				return nil, fmt.Errorf("гардиан %s: %w", h, err)
			}
			guardians = append(guardians, g)
		}
	}
	var approvals []wire.GuardianApproval
	if approvalsJSON != "" {
		var raw []struct {
			Guardian string `json:"guardian"`
			Sig      string `json:"sig"`
		}
		if err := json.Unmarshal([]byte(approvalsJSON), &raw); err != nil {
			return nil, fmt.Errorf("approvals: %w", err)
		}
		for _, a := range raw {
			g, err := hex.DecodeString(a.Guardian)
			if err != nil {
				return nil, fmt.Errorf("гардиан: %w", err)
			}
			s, err := hex.DecodeString(a.Sig)
			if err != nil {
				return nil, fmt.Errorf("подпись: %w", err)
			}
			approvals = append(approvals, wire.GuardianApproval{
				Guardian: g, StatementSig: s,
			})
		}
	}
	tx := newTx(networkID, wallet, seq, prev, wire.UpdateGuardians,
		&wire.UpdateGuardiansPayload{Propose: &wire.UpdateGuardiansPropose{
			NewGuardians: guardians,
			Approvals:    approvals,
			Nonce:        uint64(nonce),
		}}, protocolVersion, ts)
	return tx, nil
}

// NewUpdateGuardiansCancel — отмена смены гардианов подписью владельца
// в окне 48 ч (11): публикуется в цепочке того же кошелька.
func NewUpdateGuardiansCancel(networkID, wallet []byte, seq int64, prev []byte,
	proposeSeq, protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, seq, prev, wire.UpdateGuardians,
		&wire.UpdateGuardiansPayload{Cancel: &wire.UpdateGuardiansCancel{
			ProposeSeq: uint64(proposeSeq),
		}}, protocolVersion, ts)
}

// GuardianChangeStatementHash — хеш смены гардианов, который подписывает
// каждый новый гардиан (своим приложением, биометрией). Совпадает с ядром:
// wire.GuardianChangeStatement.Hash() над {network_id, ward, список, nonce}.
func GuardianChangeStatementHash(networkID, ward []byte, guardiansJSON string,
	nonce int64) ([]byte, error) {
	var guardians [][]byte
	if guardiansJSON != "" {
		var hexes []string
		if err := json.Unmarshal([]byte(guardiansJSON), &hexes); err != nil {
			return nil, fmt.Errorf("гардианы: %w", err)
		}
		for _, h := range hexes {
			g, err := hex.DecodeString(h)
			if err != nil {
				return nil, fmt.Errorf("гардиан %s: %w", h, err)
			}
			guardians = append(guardians, g)
		}
	}
	stmt := &wire.GuardianChangeStatement{
		NetworkID:    append([]byte(nil), networkID...),
		Ward:         append([]byte(nil), ward...),
		NewGuardians: guardians,
		Nonce:        uint64(nonce),
	}
	h := stmt.Hash()
	return h[:], nil
}

// ---------------------------------------------------------------------------
// Песочница: синтетическая аттестация для эмулятора (6.5)
// ---------------------------------------------------------------------------

// SandboxAttestationChain — синтетическая цепочка аттестации для песочницы
// (эмулятор, 6.5): лист над ключом кошелька, подписанный синтетическим
// корнем, детерминированным от network_id (одинаков у всех узлов сети).
// Реальные сети: цепочка KeyMint из Keystore против корней Google в APK
// (7.2); на эмуляторе такая цепочка не сойдётся с синтетическими корнями —
// поэтому полный путь участника в эмуляторе идёт через песочницу.
func SandboxAttestationChain(pub, networkID, challenge []byte) ([]byte, error) {
	pk, err := crypto.ParsePubKey(pub)
	if err != nil {
		return nil, fmt.Errorf("ключ кошелька: %w", err)
	}
	rootKey := crypto.DeterministicKey("tnp-android-root:" + string(networkID))
	rootCert, _, err := attest.SyntheticRoot("android", rootKey)
	if err != nil {
		return nil, err
	}
	// серийный номер листа детерминирован от ключа и challenge
	serial := int64(0)
	if len(challenge) >= 8 {
		for _, b := range challenge[:8] {
			serial = serial<<8 | int64(b)
		}
	}
	serial &= (1 << 62) - 1
	leaf, err := attest.BuildLeaf(pk, challenge,
		attest.Extension{
			SecurityLevel: attest.LevelTEE, VerifiedBootState: attest.BootVerified,
			DeviceLocked: true, PatchYear: 2026, PatchMonth: 10,
		}, rootKey, rootCert, serial)
	if err != nil {
		return nil, err
	}
	return attest.EncodeChain(leaf), nil
}

// ---------------------------------------------------------------------------
// Genesis: манифест для приложения (все параметры сети; 4, 17)
// ---------------------------------------------------------------------------

// Имена enum-ов профиля для JSON (в core/wire нет String(); значения
// соответствуют proto/genesis.proto).

func unitName(v wire.UnitType) string {
	switch v {
	case wire.UnitCurrency:
		return "currency"
	case wire.UnitHours:
		return "hours"
	case wire.UnitQuota:
		return "quota"
	case wire.UnitPoints:
		return "points"
	case wire.UnitVoucher:
		return "voucher"
	case wire.UnitHeartbeat:
		return "heartbeat"
	}
	return "unspecified"
}

func transferModeName(v wire.TransferMode) string {
	switch v {
	case wire.TransferModeFree:
		return "free"
	case wire.TransferModeRestricted:
		return "restricted"
	case wire.TransferModeSoulbound:
		return "soulbound"
	}
	return "unspecified"
}

func issuanceModeName(v wire.IssuanceMode) string {
	switch v {
	case wire.IssuancePeriodicEqual:
		return "periodic_equal"
	case wire.IssuanceQuotaReset:
		return "quota_reset"
	case wire.IssuanceOnContribution:
		return "on_contribution"
	case wire.IssuancePoolClaim:
		return "pool_claim"
	case wire.IssuanceHeartbeat:
		return "heartbeat"
	case wire.IssuanceNone:
		return "none"
	}
	return "unspecified"
}

func feeModeName(v wire.FeeMode) string {
	switch v {
	case wire.FeeModeBurn:
		return "burn"
	case wire.FeeModePool:
		return "pool"
	case wire.FeeModeNone:
		return "none"
	}
	return "unspecified"
}

func entryModeName(v wire.EntryMode) string {
	switch v {
	case wire.EntryOpenVouch:
		return "open_vouch"
	case wire.EntryFoundersSeed:
		return "founders_seed"
	case wire.EntryInviteOnlyVouch:
		return "invite_only_vouch"
	}
	return "unspecified"
}

func devicePolicyName(v wire.DevicePolicy) string {
	switch v {
	case wire.DevicePolicyStrict:
		return "strict"
	case wire.DevicePolicyRelaxed:
		return "relaxed"
	}
	return "unspecified"
}

func signingAuthName(v wire.SigningAuth) string {
	switch v {
	case wire.SigningAuthBiometricStrong:
		return "biometric_strong"
	case wire.SigningAuthDeviceCredential:
		return "device_credential"
	}
	return "unspecified"
}

func networkAccessName(v wire.NetworkAccess) string {
	switch v {
	case wire.NetworkAccessPublic:
		return "public"
	case wire.NetworkAccessInviteOnly:
		return "invite_only"
	}
	return "unspecified"
}

func checkpointModeName(v wire.CheckpointMode) string {
	switch v {
	case wire.CheckpointModeDecentralized:
		return "decentralized"
	case wire.CheckpointModeMembersQuorum:
		return "members_quorum"
	}
	return "unspecified"
}

func timeScaleName(v wire.TimeScale) string {
	switch v {
	case wire.TimeScaleReal:
		return "real"
	case wire.TimeScaleAccelerated:
		return "accelerated"
	}
	return "unspecified"
}

// ManifestJSON — все параметры профиля и манифеста одним JSON: единица,
// выпуск, комиссия, демередж, K, слоты, мораторий, P/P_max, D, гардианы,
// миграция, эпоха, PoW и т.д. Приложение не хардкодит ни один параметр.
func (g *Genesis) ManifestJSON() string {
	m := g.gp.Manifest
	p := m.Profile
	type tier struct {
		MinAgeDays uint64 `json:"min_age_days"`
		Slots      uint32 `json:"slots"`
	}
	tiers := make([]tier, 0, len(p.SlotTiers))
	for _, t := range p.SlotTiers {
		tiers = append(tiers, tier{t.MinAgeDays, t.Slots})
	}
	out := struct {
		NetworkID   string `json:"network_id"`
		DisplayName string `json:"display_name"`
		Unit        string `json:"unit"`
		ResourceID  string `json:"resource_id,omitempty"`

		TransferMode string `json:"transfer_mode"`
		IssuanceMode string `json:"issuance_mode"`

		IssuanceAmount         uint64 `json:"issuance_amount"`
		IssuancePeriodMonths   uint32 `json:"issuance_period_months"`
		AccrualWindowPeriods   uint64 `json:"accrual_window_periods"`
		FreshTransferWindowPer uint64 `json:"fresh_transfer_window_periods"`

		FeeMode string `json:"fee_mode"`
		FeeMin  uint64 `json:"fee_min"`
		RateMin uint64 `json:"rate_min"`
		RateMax uint64 `json:"rate_max"`

		DemurrageRateMicros uint64 `json:"demurrage_rate_micros"`
		DemurrageFloor      uint64 `json:"demurrage_floor"`

		EntryMode string `json:"entry_mode"`

		VouchK                uint32 `json:"vouch_k"`
		VoucherMinAgeDays     uint64 `json:"voucher_min_age_days"`
		VoucherMinMintPeriods uint32 `json:"voucher_min_mint_periods"`
		SlotTiers             []tier `json:"slot_tiers"`
		SeedSlotCount         uint32 `json:"seed_slot_count"`
		SlotFreeMintPeriods   uint32 `json:"slot_free_mint_periods"`
		SlotFreeDays          uint64 `json:"slot_free_days"`
		VoucherRewardMicros   uint64 `json:"voucher_reward_micros"`
		VoucherRewardPeriods  uint32 `json:"voucher_reward_periods"`

		DevicePolicy string `json:"device_policy"`
		SigningAuth  string `json:"signing_auth"`

		NetworkAccess  string `json:"network_access"`
		CheckpointMode string `json:"checkpoint_mode"`
		EpochSeconds   uint64 `json:"epoch_seconds"`

		Sandbox       bool   `json:"sandbox"`
		TimeScale     string `json:"time_scale"`
		ParamsMutable bool   `json:"params_mutable"`
		ValidatorsM   uint64 `json:"validators_m"`
		EclipseN      uint64 `json:"eclipse_n"`

		TransferMoratoriumDays  uint64 `json:"transfer_moratorium_days"`
		TransferDelayHours      uint64 `json:"transfer_delay_hours"`
		DelayedThresholdDefault uint64 `json:"delayed_threshold_default"`
		DelayedThresholdMax     uint64 `json:"delayed_threshold_max"`

		RegisteredExpiryDays uint64 `json:"registered_expiry_days"`

		GuardianPersonalThreshold  uint64 `json:"guardian_personal_threshold"`
		GuardianPersonalCount      uint64 `json:"guardian_personal_count"`
		GuardianPublicThreshold    uint64 `json:"guardian_public_threshold"`
		GuardianPublicCount        uint64 `json:"guardian_public_count"`
		GuardianMinAgeDays         uint64 `json:"guardian_min_age_days"`
		GuardianMaxWards           uint64 `json:"guardian_max_wards"`
		GuardianPersonalMaxWards   uint64 `json:"guardian_personal_max_wards"`
		GuardianReward             uint64 `json:"guardian_reward"`
		GuardianRewardCooldownDays uint64 `json:"guardian_reward_cooldown_days"`
		MigrateDelayDays           uint64 `json:"migrate_delay_days"`

		TimestampToleranceSeconds uint64 `json:"timestamp_tolerance_seconds"`
		MintChallengeWindowHours  uint64 `json:"mint_challenge_window_hours"`

		PowTargetLight   uint64 `json:"pow_target_light"`
		PowTargetHard    uint64 `json:"pow_target_hard"`
		Argon2MemKiB     uint32 `json:"argon2_mem_kib"`
		Argon2Iterations uint32 `json:"argon2_iterations"`
		Argon2Par        uint32 `json:"argon2_par"`
	}{
		NetworkID:   string(m.NetworkID),
		DisplayName: m.DisplayName,
		Unit:        unitName(p.Unit),
		ResourceID:  p.ResourceID,

		TransferMode: transferModeName(p.TransferMode),
		IssuanceMode: issuanceModeName(p.IssuanceMode),

		IssuanceAmount:         p.IssuanceAmount,
		IssuancePeriodMonths:   p.IssuancePeriodMonths,
		AccrualWindowPeriods:   p.AccrualWindowPeriods,
		FreshTransferWindowPer: p.FreshTransferWindowPeriods,

		FeeMode: feeModeName(p.FeeMode),
		FeeMin:  p.FeeMin,
		RateMin: p.RateMin,
		RateMax: p.RateMax,

		DemurrageRateMicros: p.DemurrageRateMicros,
		DemurrageFloor:      p.DemurrageFloor,

		EntryMode: entryModeName(p.EntryMode),

		VouchK:                p.VouchK,
		VoucherMinAgeDays:     p.VoucherMinAgeDays,
		VoucherMinMintPeriods: p.VoucherMinMintPeriods,
		SlotTiers:             tiers,
		SeedSlotCount:         p.SeedSlotCount,
		SlotFreeMintPeriods:   p.SlotFreeMintPeriods,
		SlotFreeDays:          p.SlotFreeDays,
		VoucherRewardMicros:   p.VoucherRewardMicros,
		VoucherRewardPeriods:  p.VoucherRewardPeriods,

		DevicePolicy: devicePolicyName(p.DevicePolicy),
		SigningAuth:  signingAuthName(p.SigningAuth),

		NetworkAccess:  networkAccessName(p.NetworkAccess),
		CheckpointMode: checkpointModeName(p.CheckpointMode),
		EpochSeconds:   p.EpochSeconds,

		Sandbox:       p.Sandbox,
		TimeScale:     timeScaleName(p.TimeScale),
		ParamsMutable: p.ParamsMutable,
		ValidatorsM:   p.ValidatorsM,
		EclipseN:      p.EclipseN,

		TransferMoratoriumDays:  p.TransferMoratoriumDays,
		TransferDelayHours:      p.TransferDelayHours,
		DelayedThresholdDefault: p.DelayedThresholdDefault,
		DelayedThresholdMax:     p.DelayedThresholdMax,

		RegisteredExpiryDays: p.RegisteredExpiryDays,

		GuardianPersonalThreshold:  p.GuardianPersonalThreshold,
		GuardianPersonalCount:      p.GuardianPersonalCount,
		GuardianPublicThreshold:    p.GuardianPublicThreshold,
		GuardianPublicCount:        p.GuardianPublicCount,
		GuardianMinAgeDays:         p.GuardianMinAgeDays,
		GuardianMaxWards:           p.GuardianMaxWards,
		GuardianPersonalMaxWards:   p.GuardianPersonalMaxWards,
		GuardianReward:             p.GuardianReward,
		GuardianRewardCooldownDays: p.GuardianRewardCooldownDays,
		MigrateDelayDays:           p.MigrateDelayDays,

		TimestampToleranceSeconds: p.TimestampToleranceSeconds,
		MintChallengeWindowHours:  p.MintChallengeWindowHours,

		PowTargetLight:   p.Pow.TargetLight,
		PowTargetHard:    p.Pow.TargetHard,
		Argon2MemKiB:     p.Pow.Argon2MemoryKiB,
		Argon2Iterations: p.Pow.Argon2Iterations,
		Argon2Par:        p.Pow.Argon2Parallelism,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Узел: JSON-снапшоты состояния для UI (net/query.go, этап 7)
// ---------------------------------------------------------------------------

func hexOr(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hex.EncodeToString(b)
}

// ClockJSON — сетевое время, эпоха, период (9.2).
func (nd *Node) ClockJSON() string {
	epoch, now, period := nd.n.QueryClock()
	b, _ := json.Marshal(map[string]int64{
		"epoch": int64(epoch), "net_time": int64(now), "period": int64(period),
	})
	return string(b)
}

// WalletInfoJSON — снапшот кошелька для UI: статус, возраст, балансы,
// порог P, поручительства, слоты, гардианы, метка flagged (7.3–7.5, 8.3).
// Лёгкий узел без Ledger — {"exists":false} (баланс — BalanceWithProof).
func (nd *Node) WalletInfoJSON(addr []byte) string {
	w := nd.n.QueryWallet(addr)
	if w == nil {
		return `{"exists":false}`
	}
	epoch, now, _ := nd.n.QueryClock()
	depth := nd.n.QueryVouchDepth(addr)
	out := map[string]any{
		"exists":                       true,
		"address":                      hexOr(w.Address),
		"status":                       w.Status.String(),
		"flagged":                      w.Flagged,
		"synthetic":                    w.Synthetic,
		"seed":                         w.Seed,
		"terminal":                     w.Terminal,
		"age_days":                     w.AgeDays(now),
		"registered_at":                int64(w.RegisteredAt),
		"activated_at":                 int64(w.ActivatedAt),
		"balance":                      int64(w.TotalBalance()),
		"liquid":                       int64(w.Liquid),
		"locked":                       int64(w.Locked),
		"seq":                          int64(w.Seq),
		"tip":                          hexOr(w.TipTxID),
		"last_period":                  int64(w.LastClaimedPeriod),
		"threshold_p":                  int64(w.DelayedThreshold),
		"pending_threshold":            int64(w.PendingThreshold),
		"pending_threshold_not_before": int64(w.PendingThresholdNotBefore),
		"epoch":                        int64(epoch),
		"vouches_received":             len(w.VouchesReceived),
		"completed_mint_periods":       int64(w.CompletedMintPeriods()),
		"vouch_depth":                  depth,
		"public_guardian":              w.PublicGuardian,
		"wards":                        int64(w.Wards),
		"guardian_changes":             int64(w.GuardianChanges),
		"mint_suspended_until_period":  int64(w.MintSuspendedUntilPeriod),
	}
	guardians := make([]string, 0, len(w.PersonalGuardians))
	for _, g := range w.PersonalGuardians {
		guardians = append(guardians, hexOr(g))
	}
	out["personal_guardians"] = guardians
	activators := make([]string, 0, len(w.Activators))
	for _, a := range w.Activators {
		activators = append(activators, hexOr(a))
	}
	out["activators"] = activators
	b, err := json.Marshal(out)
	if err != nil {
		return `{"exists":false}`
	}
	return string(b)
}

// VouchProgressJSON — сколько vouch набрано кандидатом до K (7.4).
func (nd *Node) VouchProgressJSON(addr []byte) string {
	received, k := nd.n.QueryVouchProgress(addr)
	b, _ := json.Marshal(map[string]int64{
		"received": int64(received), "k": int64(k),
	})
	return string(b)
}

// SlotsJSON — слоты поручителя (7.4): занятые/свободные, освобождение
// (3 периода выпуска / 90 дней), статистика (за скольких поручился).
func (nd *Node) SlotsJSON(addr []byte) string {
	w := nd.n.QueryWallet(addr)
	if w == nil {
		return `{"exists":false}`
	}
	_, now, _ := nd.n.QueryClock()
	capacity := nd.n.QuerySlotCapacity(addr)
	type slot struct {
		Candidate string `json:"candidate"`
		Since     uint64 `json:"since"`
		Active    bool   `json:"candidate_active"`
		MintsBase uint64 `json:"mints_base"`
	}
	occupied := make([]slot, 0, len(w.OccupiedSlots))
	for _, s := range w.OccupiedSlots {
		occupied = append(occupied, slot{
			Candidate: hexOr(s.Candidate), Since: s.SinceNetTime,
			Active: s.CandidateActive, MintsBase: s.MintsBase,
		})
	}
	b, _ := json.Marshal(map[string]any{
		"exists":   true,
		"capacity": int64(capacity),
		"occupied": occupied,
		"free":     int64(capacity) - int64(len(occupied)),
		"now":      int64(now),
	})
	return string(b)
}

// HistoryJSON — публичная история цепочки кошелька (9.1), новые сверху.
// limit <= 0 — вся цепочка. Лёгкий узел — своя цепочка и входящие (12.5).
func (nd *Node) HistoryJSON(addr []byte, limit int64) string {
	txs := nd.n.QueryHistory(addr)
	out := make([]map[string]any, 0, len(txs))
	for _, tx := range txs {
		e := map[string]any{
			"seq":       int64(tx.Seq),
			"type":      tx.Type.String(),
			"txid":      hexOr(func() []byte { id := tx.TxID(); return id[:] }()),
			"timestamp": int64(tx.TimestampClaim),
			"recipient": "",
			"amount":    int64(0),
			"period":    int64(0),
			"candidate": "",
			"note":      "",
		}
		switch p := tx.Payload.(type) {
		case *wire.MintPayload:
			e["period"] = int64(p.Period)
			e["amount"] = int64(p.Amount)
		case *wire.TransferPayload:
			switch {
			case p.Send != nil:
				e["recipient"] = hexOr(p.Send.Recipient)
				e["amount"] = int64(p.Send.Amount)
			case p.Cancel != nil:
				e["note"] = "отмена отложенного перевода"
				e["period"] = int64(p.Cancel.PendingSeq)
			case p.SetThreshold != nil:
				e["note"] = "смена порога P"
				e["amount"] = int64(p.SetThreshold.NewThreshold)
			}
		case *wire.VouchPayload:
			e["candidate"] = hexOr(p.Candidate)
		case *wire.MigratePayload:
			if p.Propose != nil {
				e["note"] = "миграция: предложение"
			} else if p.Cancel != nil {
				e["note"] = "миграция: отмена"
			}
		case *wire.GuardianOfferPayload:
			e["note"] = "объявление публичным гардианом"
		case *wire.UpdateGuardiansPayload:
			if p.Propose != nil {
				e["note"] = "смена гардианов: предложение"
				e["period"] = int64(len(p.Propose.NewGuardians))
			} else if p.Cancel != nil {
				e["note"] = "смена гардианов: отмена"
			}
		}
		out = append(out, e)
	}
	// новые сверху
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > int(limit) {
		out = out[:limit]
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// PendingTransfersJSON — отложенные переводы кошелька (8.3) с обратным
// отсчётом до исполнения (Deadline − NetTime).
func (nd *Node) PendingTransfersJSON(addr []byte) string {
	pts := nd.n.QueryPendingTransfers(addr)
	_, now, _ := nd.n.QueryClock()
	type pt struct {
		Seq       int64  `json:"seq"`
		Recipient string `json:"recipient"`
		Amount    int64  `json:"amount"`
		Fee       int64  `json:"fee"`
		CreatedAt uint64 `json:"created_at"`
		Deadline  uint64 `json:"deadline"`
		Remaining int64  `json:"remaining_sec"`
		Canceled  bool   `json:"canceled"`
		Executed  bool   `json:"executed"`
	}
	out := make([]pt, 0, len(pts))
	for _, p := range pts {
		out = append(out, pt{
			Seq: int64(p.Seq), Recipient: hexOr(p.Recipient),
			Amount: int64(p.Amount), Fee: int64(p.Fee),
			CreatedAt: p.CreatedAt, Deadline: p.Deadline,
			Remaining: int64(p.Deadline) - int64(now),
			Canceled:  p.Canceled, Executed: p.Executed,
		})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// PendingMigrationsJSON — незавершённые миграции (11) с обратным
// отсчётом 14 дней (ExecuteNotBefore − NetTime) и подписями гардианов.
func (nd *Node) PendingMigrationsJSON(addr []byte) string {
	pms := nd.n.QueryPendingMigrations(addr)
	_, now, _ := nd.n.QueryClock()
	type ap struct {
		Guardian string `json:"guardian"`
	}
	type pm struct {
		NewAddress string `json:"new_address"`
		OldAddress string `json:"old_address"`
		ProposeSeq uint64 `json:"propose_seq"`
		CreatedAt  uint64 `json:"created_at"`
		NotBefore  uint64 `json:"execute_not_before"`
		Remaining  int64  `json:"remaining_sec"`
		Nonce      uint64 `json:"nonce"`
		Canceled   bool   `json:"canceled"`
		Executed   bool   `json:"executed"`
		Approvals  []ap   `json:"approvals"`
	}
	out := make([]pm, 0, len(pms))
	for _, m := range pms {
		aps := make([]ap, 0, len(m.Approvals))
		for _, a := range m.Approvals {
			aps = append(aps, ap{Guardian: hexOr(a.Guardian)})
		}
		out = append(out, pm{
			NewAddress: hexOr(m.NewAddress), OldAddress: hexOr(m.OldAddress),
			ProposeSeq: m.ProposeSeq, CreatedAt: m.CreatedAt,
			NotBefore: m.ExecuteNotBefore,
			Remaining: int64(m.ExecuteNotBefore) - int64(now),
			Nonce:     m.Nonce, Canceled: m.Canceled, Executed: m.Executed,
			Approvals: aps,
		})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// PendingGuardianUpdatesJSON — смены гардианов кошелька (11, решение
// владельца 21.10.2026): активная смена с отсчётом окна 48 ч, а также
// отменённые/вступившие/истёкшие для истории экрана.
func (nd *Node) PendingGuardianUpdatesJSON(addr []byte) string {
	pus := nd.n.QueryPendingGuardianUpdates(addr)
	_, now, _ := nd.n.QueryClock()
	type pu struct {
		ProposeSeq   int64    `json:"propose_seq"`
		NewGuardians []string `json:"new_guardians"`
		CreatedAt    uint64   `json:"created_at"`
		EffectiveAt  uint64   `json:"effective_at"`
		Remaining    int64    `json:"remaining_sec"`
		Nonce        uint64   `json:"nonce"`
		Canceled     bool     `json:"canceled"`
		Effective    bool     `json:"effective"`
		Expired      bool     `json:"expired"`
	}
	out := make([]pu, 0, len(pus))
	for _, u := range pus {
		gs := make([]string, 0, len(u.NewGuardians))
		for _, g := range u.NewGuardians {
			gs = append(gs, hexOr(g))
		}
		out = append(out, pu{
			ProposeSeq:   int64(u.ProposeSeq),
			NewGuardians: gs,
			CreatedAt:    u.CreatedAt,
			EffectiveAt:  u.EffectiveAt,
			Remaining:    int64(u.EffectiveAt) - int64(now),
			Nonce:        u.Nonce,
			Canceled:     u.Canceled,
			Effective:    u.Effective,
			Expired:      u.Expired,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// MintWindowJSON — окно челленджа выпуска (7.6): открыто/закрыто, часы
// окна, окно накопления accrual_window, текущий период и последний
// заявленный. Точный момент слота не раскрывается (PENDING, отчёт 7).
// claim_period/claim_amount — период к заявлению (последний завершённый)
// и сумма по M6 (amount = эмиссия × diff; diff от base: для первого
// выпуска base = activated_period − 1, вопрос №24) — UI не дублирует
// формулу ядра. claim_period = 0 — заявлять нечего.
// open — окно заявляемого периода (M7, №39: ядро проверяет слот
// периода из Mint-транзакции, т.е. claim_period; находка №2 прогона
// stage11-prep — ранее передавался текущий период, и кнопка Mint была
// отключена при фактически открытом окне).
func (nd *Node) MintWindowJSON(addr []byte) string {
	w := nd.n.QueryWallet(addr)
	_, now, _ := nd.n.QueryClock()
	// период живых часов узла (RealClock, 30-сут. конвенция ядра):
	// applyMint проверяет M4 по живым часам, не по State.CurrentPeriod
	period := nd.n.Clock.Period()
	prof := nd.n.QueryProfile()
	if prof == nil {
		prof = &wire.ProfileParams{}
	}
	var last, activated uint64
	if w != nil {
		last = w.LastClaimedPeriod
		activated = w.ActivatedPeriod
	}
	claim := uint64(0)
	if period > 0 {
		claim = period - 1
	}
	if claim <= last {
		claim = 0
	}
	// окно заявляемого периода: слот челленджа привязан к периоду из
	// Mint-транзакции (claim), не к текущему (находка №2)
	open := claim > 0 && nd.n.QueryMintWindowOpen(addr, claim, now)
	var amount uint64
	if claim > 0 {
		base := last
		if base == 0 {
			base = activated - 1 // конвенция ядра (M4, uint64-перенос)
		}
		amount = prof.IssuanceAmount * (claim - base)
	}
	var lastOut int64
	if w != nil {
		lastOut = int64(w.LastClaimedPeriod)
	}
	b, _ := json.Marshal(map[string]any{
		"open":                   open,
		"window_hours":           int64(prof.MintChallengeWindowHours),
		"accrual_window_periods": int64(prof.AccrualWindowPeriods),
		"issuance_amount":        int64(prof.IssuanceAmount),
		"current_period":         int64(period),
		"claim_period":           int64(claim),
		"claim_amount":           int64(amount),
		"last_claimed_period":    lastOut,
		"now":                    int64(now),
	})
	return string(b)
}

// FeeQuote — комиссия на перевод суммы (10.2): платит отправитель
// сверх суммы; видна до отправки.
func (nd *Node) FeeQuote(amount int64) int64 {
	return int64(nd.n.QueryFee(uint64(amount)))
}

// CurrentFeeRate — текущая ставка комиссии в micros (адаптивный
// пересчёт из чекпоинтов, 10.2): 10000 = 1%.
func (nd *Node) CurrentFeeRate() int64 {
	return int64(nd.n.QueryFeeRate())
}

// SyncDiag — диагностика синхронизации (этап 7): принудительная сверка
// с каждым подключённым пиром, результат строкой (applied/ошибка) для
// журнала моста. Только диагностика песочницы.
func (nd *Node) SyncDiag() string {
	peers := nd.n.ConnectedPeers()
	out := make([]string, 0, len(peers))
	for _, p := range peers {
		k, err := nd.n.SyncFromPeer(context.Background(), p)
		out = append(out, fmt.Sprintf("%s: applied=%d err=%v",
			p.ShortString(), k, err))
	}
	return strings.Join(out, "; ")
}

// AwaitingJSON — офлайн-транзакции, ожидающие публикации (12.4): экран
// получателя со статусом «получено по Bluetooth — ожидает публикации».
func (nd *Node) AwaitingJSON() string {
	txs := nd.n.QueryAwaiting()
	type row struct {
		Txid      string `json:"txid"`
		Type      string `json:"type"`
		Recipient string `json:"recipient"`
		Amount    int64  `json:"amount"`
		Status    string `json:"status"`
	}
	out := make([]row, 0, len(txs))
	for _, tx := range txs {
		id := tx.TxID()
		e := row{Txid: hexOr(id[:]), Type: tx.Type.String()}
		if pl, ok := tx.Payload.(*wire.TransferPayload); ok && pl.Send != nil {
			e.Recipient = hexOr(pl.Send.Recipient)
			e.Amount = int64(pl.Send.Amount)
		}
		e.Status = nd.n.OfflineStatus(id).String()
		out = append(out, e)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// GuardianOffersJSON — публичные гардианы сети (11): для экрана выбора
// при миграции (порог 3 из 5, комбинируется с личными).
func (nd *Node) GuardianOffersJSON() string {
	addrs := nd.n.QueryPublicGuardians()
	out := make([]map[string]any, 0, len(addrs))
	for _, a := range addrs {
		w := nd.n.QueryWallet(a)
		if w == nil {
			continue
		}
		_, now, _ := nd.n.QueryClock()
		out = append(out, map[string]any{
			"address":  hexOr(a),
			"age_days": w.AgeDays(now),
			"wards":    int64(w.Wards),
			"status":   w.Status.String(),
			"flagged":  w.Flagged,
		})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// RegistryJSON — публичный реестр для списка участников (9.1): адрес,
// статус, возраст, метка flagged. Глубина графа — по запросу экрана
// участника (WalletInfoJSON).
func (nd *Node) RegistryJSON() string {
	addrs := nd.n.QueryRegistry()
	_, now, _ := nd.n.QueryClock()
	type row struct {
		Address string `json:"address"`
		Status  string `json:"status"`
		AgeDays uint64 `json:"age_days"`
		Flagged bool   `json:"flagged"`
	}
	out := make([]row, 0, len(addrs))
	for _, a := range addrs {
		w := nd.n.QueryWallet(a)
		if w == nil {
			continue
		}
		out = append(out, row{
			Address: hexOr(a), Status: w.Status.String(),
			AgeDays: w.AgeDays(now), Flagged: w.Flagged,
		})
	}
	b, _ := json.Marshal(out)
	return string(b)
}
