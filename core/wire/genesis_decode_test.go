// Round-trip тесты декодера genesis-пакета (этап 8): EncodeGenesisPackage ->
// DecodeGenesisPackage — манифест и запись восстанавливаются байт в байт,
// manifest_hash сходится.
package wire

import (
	"bytes"
	"testing"
)

func testGenesisPackage() *GenesisPackage {
	profile := &ProfileParams{
		Unit:                       UnitCurrency,
		TransferMode:               TransferModeFree,
		IssuanceMode:               IssuancePeriodicEqual,
		IssuanceAmount:             1000,
		IssuancePeriodMonths:       1,
		AccrualWindowPeriods:       6,
		FreshTransferWindowPeriods: 3,
		FeeMode:                    FeeModeBurn,
		FeeMin:                     1,
		RateMin:                    10000,
		RateMax:                    20000,
		DemurrageRateMicros:        10000,
		DemurrageFloor:             3000,
		EntryMode:                  EntryInviteOnlyVouch,
		VouchK:                     3,
		VoucherMinAgeDays:          90,
		VoucherMinMintPeriods:      2,
		SlotTiers:                  []SlotTier{{MinAgeDays: 90, Slots: 3}, {MinAgeDays: 365, Slots: 5}},
		SeedSlotCount:              10,
		SlotFreeMintPeriods:        3,
		SlotFreeDays:               90,
		VoucherRewardMicros:        100000,
		VoucherRewardPeriods:       12,
		DevicePolicy:               DevicePolicyStrict,
		SigningAuth:                SigningAuthBiometricStrong,
		NetworkAccess:              NetworkAccessInviteOnly,
		CheckpointMode:             CheckpointModeDecentralized,
		EpochSeconds:               86400,
		Sandbox:                    false,
		TimeScale:                  TimeScaleReal,
		ParamsMutable:              false,
		Modules:                    []uint32{1, 2},
		TransferMoratoriumDays:     14,
		TransferDelayHours:         48,
		DelayedThresholdDefault:    3000,
		DelayedThresholdMax:        30000,
		RegisteredExpiryDays:       1095,
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
		MigrateDelayDays:           14,
		TimestampToleranceSeconds:  1800,
		MintChallengeWindowHours:   48,
		Pow: &PowParams{
			TargetLight: 0x1000000000000, Argon2MemoryKiB: 32768,
			Argon2Iterations: 1, Argon2Parallelism: 1,
			TargetHard: 0x40000000000000, RepeatAndroidIDTarget: 0,
		},
	}
	manifest := &Manifest{
		NetworkID:   []byte("tnp-testnet-roundtrip"),
		DisplayName: "TNP RoundTrip",
		Profile:     profile,
		SeedCircle: []SeedMember{
			{Address: []byte("addr-1"), PublicKey: []byte("pub-1")},
			{Address: []byte("addr-2"), PublicKey: []byte("pub-2")},
		},
		SeedSunset: &SeedSunset{UseActiveWallets: true, ActiveWallets: 100, DeadlineEpoch: 24},
	}
	mh := manifest.ManifestHash()
	genesis := &GenesisRecord{
		NetworkID:       manifest.NetworkID,
		ManifestHash:    mh[:],
		GenesisTime:     1_760_000_000,
		EpochZero:       0,
		ProtocolVersion: 1,
		SeedSunset:      manifest.SeedSunset,
		InviteKey:       []byte("invite-key-test"),
	}
	return &GenesisPackage{Manifest: manifest, Genesis: genesis}
}

func TestGenesisPackageRoundTrip(t *testing.T) {
	gp := testGenesisPackage()
	blob := gp.EncodeGenesisPackage()
	got, err := DecodeGenesisPackage(blob)
	if err != nil {
		t.Fatalf("декодирование: %v", err)
	}
	if !bytes.Equal(got.Genesis.ManifestHash, gp.Genesis.ManifestHash) {
		t.Fatal("manifest_hash не совпал")
	}
	if !bytes.Equal(got.Manifest.NetworkID, gp.Manifest.NetworkID) {
		t.Fatal("network_id манифеста не совпал")
	}
	if got.Genesis.GenesisTime != gp.Genesis.GenesisTime {
		t.Fatalf("genesis_time: %d != %d", got.Genesis.GenesisTime, gp.Genesis.GenesisTime)
	}
	if !bytes.Equal(got.Genesis.InviteKey, gp.Genesis.InviteKey) {
		t.Fatal("invite_key не совпал")
	}
	if len(got.Manifest.SeedCircle) != len(gp.Manifest.SeedCircle) {
		t.Fatalf("посевной круг: %d != %d", len(got.Manifest.SeedCircle), len(gp.Manifest.SeedCircle))
	}
	if got.Manifest.Profile.VouchK != 3 || got.Manifest.Profile.MintChallengeWindowHours != 48 {
		t.Fatal("параметры профиля не восстановились")
	}
	if got.Manifest.Profile.Pow == nil || got.Manifest.Profile.Pow.TargetHard != gp.Manifest.Profile.Pow.TargetHard {
		t.Fatal("PoW-параметры не восстановились")
	}
	if len(got.Manifest.Profile.SlotTiers) != 2 || got.Manifest.Profile.SlotTiers[1].Slots != 5 {
		t.Fatal("ярусы слотов не восстановились")
	}
	if got.Manifest.SeedSunset == nil || !got.Manifest.SeedSunset.UseActiveWallets ||
		got.Manifest.SeedSunset.ActiveWallets != 100 || got.Manifest.SeedSunset.DeadlineEpoch != 24 {
		t.Fatal("seed_sunset не восстановился")
	}
	// повторное кодирование декодированного пакета — тот же блоб (каноничность)
	if !bytes.Equal(got.EncodeGenesisPackage(), blob) {
		t.Fatal("повторное кодирование изменило блоб")
	}
	// manifest_hash декодированного манифеста сходится с записью
	mh := got.Manifest.ManifestHash()
	if !bytes.Equal(mh[:], got.Genesis.ManifestHash) {
		t.Fatal("manifest_hash декодированного манифеста не сходится")
	}
}

func TestGenesisPackageRejectsUnknownField(t *testing.T) {
	gp := testGenesisPackage()
	blob := gp.EncodeGenesisPackage()
	// поле 3 в пакете не определено — каноничность требует отказа
	bad := append([]byte{}, blob...)
	bad = append(bad, 0x1a, 0x00) // тег поля 3, wiretype 2, длина 0
	if _, err := DecodeGenesisPackage(bad); err == nil {
		t.Fatal("неизвестное поле должно отвергаться")
	}
}
