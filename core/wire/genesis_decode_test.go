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
			{PublicKey: []byte("pub-1")},
			{PublicKey: []byte("pub-2")},
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

// TestRecurringSlugRoundTrip — новые поля v9.10 (recurring — поле 60
// профиля, slug — поле 7 манифеста, peer-redeem — режим 4) переживают
// каноническое кодирование.
func TestRecurringSlugRoundTrip(t *testing.T) {
	m := &Manifest{
		NetworkID:   []byte("net"),
		DisplayName: "Имя",
		Slug:        "monte-cloud",
		Profile: &ProfileParams{
			Unit: UnitQuota, ResourceID: "compute", Recurring: true,
			TransferMode: TransferModePeerRedeem, IssuanceMode: IssuanceQuotaReset,
		},
	}
	back, err := DecodeManifest(m.Encode())
	if err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if back.Slug != "monte-cloud" || !back.Profile.Recurring ||
		back.Profile.TransferMode != TransferModePeerRedeem {
		t.Fatalf("round-trip: slug=%q recurring=%v mode=%d",
			back.Slug, back.Profile.Recurring, back.Profile.TransferMode)
	}
}

// TestPeriodSecondsRoundTrip — поле 61 (period_seconds, блокер живого
// прогона мини-тестнета этапа 12): длина периода выпуска кодируется и
// восстанавливается; 0 — конвенция 30 сут (PeriodSecsOrDefault).
func TestPeriodSecondsRoundTrip(t *testing.T) {
	for _, secs := range []uint64{0, 60, 120, 30 * 86400} {
		m := &Manifest{
			NetworkID:   []byte("net"),
			DisplayName: "Имя",
			Profile:     &ProfileParams{Unit: UnitQuota, ResourceID: "r", PeriodSeconds: secs},
		}
		back, err := DecodeManifest(m.Encode())
		if err != nil {
			t.Fatalf("decode manifest: %v", err)
		}
		if back.Profile.PeriodSeconds != secs {
			t.Fatalf("period_seconds: %d != %d", back.Profile.PeriodSeconds, secs)
		}
	}
	def := &ProfileParams{}
	if def.PeriodSecsOrDefault() != 30*86400 {
		t.Fatalf("default period: %d", def.PeriodSecsOrDefault())
	}
	if p := (&ProfileParams{PeriodSeconds: 120}); p.PeriodSecsOrDefault() != 120 {
		t.Fatalf("period 120: %d", p.PeriodSecsOrDefault())
	}
}

// TestDeriveNetworkID — само-сертификация (v9.10): network_id =
// H(manifest ‖ genesis) детерминирован, меняется от любого поля
// манифеста/genesis, совпадает в манифесте и записи после FinalizeGenesis.
func TestDeriveNetworkID(t *testing.T) {
	gp := testGenesisPackage()
	nid1 := DeriveNetworkID(gp.Manifest, gp.Genesis)
	if !bytes.Equal(nid1, DeriveNetworkID(gp.Manifest, gp.Genesis)) {
		t.Fatalf("деривация недетерминирована")
	}
	// любое изменение манифеста меняет network_id
	gp2 := testGenesisPackage()
	gp2.Manifest.DisplayName = "другое имя"
	if bytes.Equal(nid1, DeriveNetworkID(gp2.Manifest, gp2.Genesis)) {
		t.Fatalf("изменение манифеста не меняет network_id")
	}
	// изменение genesis-поля меняет network_id
	gp3 := testGenesisPackage()
	gp3.Genesis.GenesisTime++
	if bytes.Equal(nid1, DeriveNetworkID(gp3.Manifest, gp3.Genesis)) {
		t.Fatalf("изменение genesis не меняет network_id")
	}
}

// TestFinalizeGenesisVerify — сборка сети конфигуратором: FinalizeGenesis
// заполняет network_id в манифест и запись одним значением; пакет
// проходит VerifySelfCertified; подделка любого поля ломает проверку;
// глобальная сеть (константа "global") проверку проходит без деривации.
func TestFinalizeGenesisVerify(t *testing.T) {
	gp := testGenesisPackage()
	FinalizeGenesis(gp)
	if err := VerifySelfCertified(gp); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if !bytes.Equal(gp.Manifest.NetworkID, gp.Genesis.NetworkID) {
		t.Fatalf("network_id манифеста и записи обязан совпадать")
	}
	// подделка параметра манифеста — не само-сертифицирован
	tampered := testGenesisPackage()
	FinalizeGenesis(tampered)
	tampered.Manifest.Profile.IssuanceAmount = 999999
	if err := VerifySelfCertified(tampered); err == nil {
		t.Fatalf("подделка манифеста обязана ломать само-сертификацию")
	}
	// подделка network_id (согласованная в манифесте и записи)
	stolen := testGenesisPackage()
	FinalizeGenesis(stolen)
	stolen.Manifest.NetworkID = append([]byte(nil), "fake-network-id..."...)
	stolen.Genesis.NetworkID = append([]byte(nil), stolen.Manifest.NetworkID...)
	if err := VerifySelfCertified(stolen); err == nil {
		t.Fatalf("подмена network_id обязана ломать само-сертификацию")
	}
	// глобальная сеть — константа, не деривируется
	gg := testGenesisPackage()
	gg.Manifest.NetworkID = GlobalNetworkID
	gg.Genesis.NetworkID = GlobalNetworkID
	if err := VerifySelfCertified(gg); err != nil {
		t.Fatalf("global: %v", err)
	}
	// пустой network_id не бывает
	gg.Manifest.NetworkID = nil
	if err := VerifySelfCertified(gg); err == nil {
		t.Fatalf("пустой network_id обязан отвергаться")
	}
}
