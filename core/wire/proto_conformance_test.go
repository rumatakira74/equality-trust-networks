// Тест-консистентности proto/*.proto ↔ ручной кодек (тема «Fuzz-долги»):
// ловит дрейф номеров полей и wire-типов между .proto-файлами
// (attest, checkpoint, core, genesis, transactions) и ручным кодеком
// core/wire. Механика: разбор .proto в тесте; для каждого сообщения
// кодек кодирует полностью населённый экземпляр, тест разбирает
// фактические (номер, wire-тип) пары и сверяет с .proto в обе стороны
// (каждое закодированное поле есть в .proto с тем же wire-типом; каждое
// не-oneof поле .proto кодируется). Oneof: кодируется ровно одна ветвь.
// Enum-значения сверяются по номерам. Выбор механики: разбор .proto
// напрямую падает при изменении .proto, golden-векторы — нет.
package wire

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Разбор .proto
// ---------------------------------------------------------------------------

type protoField struct {
	name  string
	wt    byte
	oneof bool
}

type protoSchema struct {
	messages map[string]map[uint32]protoField // msg -> field num -> field
	enums    map[string]map[string]uint32     // enum -> value name -> number
}

var (
	reMessage = regexp.MustCompile(`^message\s+(\w+)\s*\{`)
	reEnum    = regexp.MustCompile(`^enum\s+(\w+)\s*\{`)
	reOneof   = regexp.MustCompile(`^oneof\s+\w+\s*\{`)
	reField   = regexp.MustCompile(`^(?:repeated\s+)?(\w+)\s+(\w+)\s*=\s*(\d+)\s*;`)
	reEnumVal = regexp.MustCompile(`^([A-Z][A-Z0-9_]*)\s*=\s*(\d+)\s*;`)
)

// wireTypeOf — wire-тип по имени типа поля .proto.
func (s *protoSchema) wireTypeOf(typ string) (byte, bool) {
	switch typ {
	case "uint32", "uint64", "int32", "int64", "bool", "sint32", "sint64":
		return 0, true
	case "bytes", "string":
		return 2, true
	}
	if _, ok := s.enums[typ]; ok {
		return 0, true
	}
	if _, ok := s.messages[typ]; ok {
		return 2, true
	}
	return 0, false
}

// parseProtoFiles — разбор всех .proto из ../../proto.
func parseProtoFiles(t *testing.T) *protoSchema {
	t.Helper()
	dir := filepath.Join("..", "..", "proto")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("proto dir: %v", err)
	}
	s := &protoSchema{
		messages: make(map[string]map[uint32]protoField),
		enums:    make(map[string]map[string]uint32),
	}
	type rawField struct {
		typ, name string
		num       uint32
		oneof     bool
	}
	raw := make(map[string][]rawField)
	msgNames := make(map[string]bool)
	// проход 1: собрать сырые поля и enum-значения
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".proto") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		var curMsg, curEnum string
		inOneof := false
		for _, line := range strings.Split(string(data), "\n") {
			if i := strings.Index(line, "//"); i >= 0 {
				line = line[:i]
			}
			line = strings.TrimSpace(line)
			switch {
			case reMessage.MatchString(line):
				curMsg = reMessage.FindStringSubmatch(line)[1]
				msgNames[curMsg] = true
				curEnum = ""
				inOneof = false
			case reEnum.MatchString(line):
				curEnum = reEnum.FindStringSubmatch(line)[1]
				curMsg = ""
				s.enums[curEnum] = make(map[string]uint32)
			case reOneof.MatchString(line):
				inOneof = true
			case line == "}":
				if inOneof {
					inOneof = false
				} else {
					curMsg, curEnum = "", ""
				}
			case curEnum != "" && reEnumVal.MatchString(line):
				m := reEnumVal.FindStringSubmatch(line)
				fmt.Sscanf(m[2], "%d", new(int))
				var num uint32
				fmt.Sscanf(m[2], "%d", &num)
				s.enums[curEnum][m[1]] = num
			case curMsg != "" && reField.MatchString(line):
				m := reField.FindStringSubmatch(line)
				var num uint32
				fmt.Sscanf(m[3], "%d", &num)
				raw[curMsg] = append(raw[curMsg], rawField{m[1], m[2], num, inOneof})
			}
		}
	}
	// проход 2: разрешить wire-типы (имена сообщений известны все)
	for name := range msgNames {
		s.messages[name] = nil // тип-пространство имён сообщений
	}
	for msg, fields := range raw {
		mf := make(map[uint32]protoField, len(fields))
		for _, f := range fields {
			wt, ok := s.wireTypeOf(f.typ)
			if !ok {
				t.Fatalf("proto: тип %q поля %s.%s не разрешён", f.typ, msg, f.name)
			}
			mf[f.num] = protoField{name: f.name, wt: wt, oneof: f.oneof}
		}
		s.messages[msg] = mf
	}
	return s
}

// ---------------------------------------------------------------------------
// Разбор фактических тегов кодека
// ---------------------------------------------------------------------------

// emittedTags — (номер поля -> wire-тип) из закодированных байтов.
func emittedTags(t *testing.T, b []byte) map[uint32]byte {
	t.Helper()
	out := make(map[uint32]byte)
	pos := 0
	readVarint := func() (uint64, bool) {
		var v uint64
		var shift uint
		for {
			if pos >= len(b) {
				return 0, false
			}
			c := b[pos]
			pos++
			v |= uint64(c&0x7f) << shift
			if c < 0x80 {
				return v, true
			}
			shift += 7
			if shift > 63 {
				return 0, false
			}
		}
	}
	for pos < len(b) {
		tag, ok := readVarint()
		if !ok {
			t.Fatalf("кодек: обрезанный тег в %x", b)
		}
		f, wt := uint32(tag>>3), byte(tag&7)
		switch wt {
		case 0:
			if _, ok := readVarint(); !ok {
				t.Fatalf("кодек: обрезанный varint у поля %d", f)
			}
		case 2:
			n, ok := readVarint()
			if !ok || uint64(len(b)-pos) < n {
				t.Fatalf("кодек: обрезанное поле %d", f)
			}
			pos += int(n)
		default:
			t.Fatalf("кодек: неожидаемый wire-тип %d у поля %d", wt, f)
		}
		out[f] = wt
	}
	return out
}

// checkConformance — сверка закодированного сообщения с .proto.
func checkConformance(t *testing.T, s *protoSchema, msg string, enc []byte) {
	t.Helper()
	exp, ok := s.messages[msg]
	if !ok {
		t.Fatalf("сообщение %s отсутствует в proto/*.proto", msg)
	}
	em := emittedTags(t, enc)
	for f, wt := range em {
		pf, ok := exp[f]
		if !ok {
			t.Errorf("%s: кодек кодирует поле %d, которого нет в .proto", msg, f)
			continue
		}
		if pf.wt != wt {
			t.Errorf("%s: поле %d (%s): wire-тип кодека %d, .proto %d",
				msg, f, pf.name, wt, pf.wt)
		}
	}
	oneofEmitted := 0
	hasOneof := false
	for f, pf := range exp {
		_, emitted := em[f]
		if pf.oneof {
			hasOneof = true
			if emitted {
				oneofEmitted++
			}
			continue
		}
		if !emitted {
			t.Errorf("%s: поле %s (%d) из .proto не кодируется", msg, pf.name, f)
		}
	}
	if hasOneof && oneofEmitted > 1 {
		// 0 допустим: ветвь-сообщение без полей (GuardianOfferPayload)
		// кодируется в 0 байт — тега oneof нет
		t.Errorf("%s: oneof: закодировано ветвей %d, ожидается не более 1", msg, oneofEmitted)
	}
}

// ---------------------------------------------------------------------------
// Полностью населённые экземпляры всех сообщений кодека
// ---------------------------------------------------------------------------

func confTx(t TxType, pl Payload) *Transaction {
	return &Transaction{
		NetworkID:       []byte("net"),
		Wallet:          []byte("wallet"),
		Seq:             7,
		Prev:            []byte("prev"),
		Type:            t,
		TimestampClaim:  1790000000,
		ProtocolVersion: 1,
		Pow:             &PowProof{Algorithm: PowArgon2id, Nonce: []byte{9, 9}},
		Signature:       []byte("sig"),
		Payload:         pl,
	}
}

func confProfile() *ProfileParams {
	return &ProfileParams{
		Unit:                       UnitVoucher,
		ResourceID:                 "res",
		TransferMode:               TransferModePeerRedeem,
		IssuanceMode:               IssuanceHeartbeat,
		IssuanceAmount:             1000,
		IssuancePeriodMonths:       1,
		AccrualWindowPeriods:       6,
		FreshTransferWindowPeriods: 3,
		FeeMode:                    FeeModePool,
		FeeMin:                     1,
		RateMin:                    100,
		RateMax:                    200,
		DemurrageRateMicros:        10000,
		DemurrageFloor:             3000,
		EntryMode:                  EntryInviteOnlyVouch,
		VouchK:                     3,
		VoucherMinAgeDays:          90,
		VoucherMinMintPeriods:      2,
		SlotTiers:                  []SlotTier{{90, 3}, {365, 5}},
		SeedSlotCount:              10,
		SlotFreeMintPeriods:        3,
		SlotFreeDays:               90,
		VoucherRewardMicros:        100000,
		VoucherRewardPeriods:       12,
		DevicePolicy:               DevicePolicyRelaxed,
		SigningAuth:                SigningAuthDeviceCredential,
		NetworkAccess:              NetworkAccessInviteOnly,
		CheckpointMode:             CheckpointModeMembersQuorum,
		EpochSeconds:               86400,
		Sandbox:                    true,
		TimeScale:                  TimeScaleAccelerated,
		ParamsMutable:              true,
		MutableBounds:              []MutableBound{{"demurrage_rate_micros", 0, 50000}, {"fee_min", 1, 9}},
		Modules:                    []uint32{ModuleAttest, ModulePool},
		Providers:                  [][]byte{[]byte("p1"), []byte("p2")},
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
		MigrateDelayDays:           14,
		TimestampToleranceSeconds:  1800,
		MintChallengeWindowHours:   48,
		Pow:                        &PowParams{1, 2, 3, 4, 5, 6},
		EclipseN:                   3,
		ValidatorsM:                21,
		CpBuilderQueue:             8,
		FeeAdjustEpochs:            7,
		FeeTargetGrowthMicros:      50000,
		FeeStepMicros:              1000,
		Recurring:                  true,
		PeriodSeconds:              2592000,
	}
}

// TestProtoConformance — все сообщения кодека против proto/*.proto.
func TestProtoConformance(t *testing.T) {
	s := parseProtoFiles(t)
	guardians := [][]byte{[]byte("g1"), []byte("g2")}
	approvals := []GuardianApproval{{[]byte("g"), []byte("sig")}}

	cases := []struct {
		msg string
		enc []byte
	}{
		// core.proto
		{"WalletFlags", (&WalletFlags{Flagged: true, Synthetic: true, Seed: true}).Encode()},
		{"PowProof", (&PowProof{Algorithm: PowArgon2id, Nonce: []byte{1}}).Encode()},
		{"TimeSlot", (&TimeSlot{Start: 1, End: 2}).Encode()},
		// transactions.proto: конверт — по ветке payload
		{"Transaction", confTx(Register, &RegisterPayload{
			AttestedPublicKey: []byte("pk"), AttestationChain: []byte("chain"),
			AttestationChallenge: []byte("ch"), AndroidID: "id",
			PersonalGuardians: guardians, Synthetic: true,
		}).Encode()},
		{"Transaction", confTx(Vouch, &VouchPayload{Candidate: []byte("c"), Epoch: 7}).Encode()},
		{"Transaction", confTx(Mint, &MintPayload{Period: 13, Amount: 100}).Encode()},
		{"Transaction", confTx(Transfer, &TransferPayload{Send: &TransferSend{[]byte("r"), 5}}).Encode()},
		{"Transaction", confTx(Transfer, &TransferPayload{Cancel: &TransferCancel{2}}).Encode()},
		{"Transaction", confTx(Transfer, &TransferPayload{SetThreshold: &ThresholdUpdate{3000}}).Encode()},
		{"Transaction", confTx(Migrate, &MigratePayload{Propose: &MigratePropose{
			[]byte("old"), approvals, 1,
		}}).Encode()},
		{"Transaction", confTx(Migrate, &MigratePayload{Cancel: &MigrateCancel{[]byte("new"), 3}}).Encode()},
		{"Transaction", confTx(Evidence, &EvidencePayload{
			TxA:  confTx(Transfer, &TransferPayload{Send: &TransferSend{[]byte("r"), 5}}),
			TxB:  confTx(Transfer, &TransferPayload{Send: &TransferSend{[]byte("r"), 6}}),
			Kind: EvidenceDoubleMint,
		}).Encode()},
		{"Transaction", confTx(GuardianOffer, &GuardianOfferPayload{}).Encode()},
		{"Transaction", confTx(PoolCreate, &PoolCreatePayload{
			[]byte("r"), 100, 9, PoolReleaseContinuous, PoolRefundNone,
		}).Encode()},
		{"Transaction", confTx(PoolDonate, &PoolDonatePayload{[]byte("pool"), 7}).Encode()},
		{"Transaction", confTx(PoolClose, &PoolClosePayload{[]byte("pool")}).Encode()},
		{"Transaction", confTx(PoolClaim, &PoolClaimPayload{[]byte("pool")}).Encode()},
		{"Transaction", confTx(Redeem, &RedeemPayload{
			[]byte("p"), "res", 3, &TimeSlot{Start: 1, End: 2},
		}).Encode()},
		{"Transaction", confTx(Receipt, &ReceiptPayload{
			[]byte("tx"), 3, "res", &TimeSlot{Start: 1, End: 2},
		}).Encode()},
		{"Transaction", confTx(ContributionClaim, &ContributionClaimPayload{
			[]byte("pool"), []byte("h"), 5, 2,
		}).Encode()},
		{"Transaction", confTx(ContributionConfirm, &ContributionConfirmPayload{[]byte("claim")}).Encode()},
		{"Transaction", confTx(UpdateGuardians, &UpdateGuardiansPayload{Propose: &UpdateGuardiansPropose{
			guardians, approvals, 4,
		}}).Encode()},
		{"Transaction", confTx(UpdateGuardians, &UpdateGuardiansPayload{Cancel: &UpdateGuardiansCancel{6}}).Encode()},
		// transactions.proto: payload-сообщения
		{"RegisterPayload", (&RegisterPayload{
			[]byte("pk"), []byte("chain"), []byte("ch"), "id", guardians, true,
		}).Encode()},
		{"VouchPayload", (&VouchPayload{[]byte("c"), 7}).Encode()},
		{"MintPayload", (&MintPayload{13, 100}).Encode()},
		{"TransferPayload", (&TransferPayload{Send: &TransferSend{[]byte("r"), 5}}).Encode()},
		{"TransferPayload", (&TransferPayload{Cancel: &TransferCancel{2}}).Encode()},
		{"TransferPayload", (&TransferPayload{SetThreshold: &ThresholdUpdate{3000}}).Encode()},
		{"TransferSend", (&TransferSend{[]byte("r"), 5}).Encode()},
		{"TransferCancel", (&TransferCancel{2}).Encode()},
		{"ThresholdUpdate", (&ThresholdUpdate{3000}).Encode()},
		{"MigratePayload", (&MigratePayload{Propose: &MigratePropose{[]byte("old"), approvals, 1}}).Encode()},
		{"MigratePayload", (&MigratePayload{Cancel: &MigrateCancel{[]byte("new"), 3}}).Encode()},
		{"MigratePropose", (&MigratePropose{[]byte("old"), approvals, 1}).Encode()},
		{"MigrateCancel", (&MigrateCancel{[]byte("new"), 3}).Encode()},
		{"MigrateStatement", (&MigrateStatement{[]byte("net"), []byte("old"), []byte("new"), 4, 5}).Encode()},
		{"GuardianApproval", (&GuardianApproval{[]byte("g"), []byte("sig")}).Encode()},
		{"EvidencePayload", (&EvidencePayload{
			TxA:  confTx(Transfer, &TransferPayload{Send: &TransferSend{[]byte("r"), 5}}),
			TxB:  confTx(Transfer, &TransferPayload{Send: &TransferSend{[]byte("r"), 6}}),
			Kind: EvidenceEquivocation,
		}).Encode()},
		{"GuardianOfferPayload", (&GuardianOfferPayload{}).Encode()},
		{"PoolCreatePayload", (&PoolCreatePayload{[]byte("r"), 100, 9, PoolReleaseOnGoal, PoolRefundIfUnmet}).Encode()},
		{"PoolDonatePayload", (&PoolDonatePayload{[]byte("pool"), 7}).Encode()},
		{"PoolClosePayload", (&PoolClosePayload{[]byte("pool")}).Encode()},
		{"PoolClaimPayload", (&PoolClaimPayload{[]byte("pool")}).Encode()},
		{"RedeemPayload", (&RedeemPayload{[]byte("p"), "res", 3, &TimeSlot{1, 2}}).Encode()},
		{"ReceiptPayload", (&ReceiptPayload{[]byte("tx"), 3, "res", &TimeSlot{1, 2}}).Encode()},
		{"ContributionClaimPayload", (&ContributionClaimPayload{[]byte("pool"), []byte("h"), 5, 2}).Encode()},
		{"ContributionConfirmPayload", (&ContributionConfirmPayload{[]byte("claim")}).Encode()},
		{"UpdateGuardiansPayload", (&UpdateGuardiansPayload{Propose: &UpdateGuardiansPropose{guardians, approvals, 4}}).Encode()},
		{"UpdateGuardiansPayload", (&UpdateGuardiansPayload{Cancel: &UpdateGuardiansCancel{6}}).Encode()},
		{"UpdateGuardiansPropose", (&UpdateGuardiansPropose{guardians, approvals, 4}).Encode()},
		{"UpdateGuardiansCancel", (&UpdateGuardiansCancel{6}).Encode()},
		{"GuardianChangeStatement", (&GuardianChangeStatement{[]byte("net"), []byte("ward"), guardians, 4}).Encode()},
		// checkpoint.proto
		{"StateLeaf", (&StateLeaf{
			Seq: 3, Balance: 42, Status: StatusExpired,
			Flags:           &WalletFlags{Flagged: true, Synthetic: true, Seed: true},
			RegisteredEpoch: 5, ActivatedEpoch: 6, LastClaimedPeriod: 7,
			DelayedThreshold: 3000, ThresholdChangeEpoch: 9,
		}).Encode()},
		{"Checkpoint", (&Checkpoint{
			NetworkID: []byte("net"), Epoch: 2, StateRoot: []byte("root"),
			PrevCheckpointHash: []byte("prev"), NetworkTime: 1790000000,
			ProtocolVersion: 1, ActiveWallets: 5, TotalSupply: 100,
			Participants: [][]byte{[]byte("p1"), []byte("p2")}, Threshold: 3,
			ThresholdSig: &ThresholdSignature{Signatures: []SignatureShare{
				{[]byte("s1"), []byte("sig1")}, {[]byte("s2"), []byte("sig2")},
			}},
			FinalizedTxIDs: [][]byte{[]byte("f1"), []byte("f2")},
		}).Encode()},
		{"ThresholdSignature", (&ThresholdSignature{Signatures: []SignatureShare{
			{[]byte("s1"), []byte("sig1")}, {[]byte("s2"), []byte("sig2")},
		}}).Encode()},
		{"SignatureShare", (&SignatureShare{[]byte("s"), []byte("sig")}).Encode()},
		// attest.proto
		{"AttestStatement", (&AttestStatement{
			NetworkID: []byte("net"), Address: []byte("addr"), Status: StatusActive,
			AgeDays: 100, VouchDepth: 3, Flagged: true, ManifestHash: []byte("mh"),
			Epoch: 7, Challenge: []byte("ch"), ExpiresAt: 1790000000, AppID: []byte("app"),
		}).Encode()},
		// genesis.proto
		{"Manifest", (&Manifest{
			NetworkID: []byte("net"), DisplayName: "name", Slug: "slug",
			Profile:     confProfile(),
			SeedCircle:  []SeedMember{{PublicKey: []byte("k1")}, {PublicKey: []byte("k2")}},
			SeedSunset:  &SeedSunset{UseActiveWallets: true, ActiveWallets: 100, DeadlineEpoch: 24},
			GenesisHash: []byte("gh"),
		}).Encode()},
		{"ProfileParams", confProfile().Encode()},
		{"SlotTier", (&SlotTier{90, 3}).Encode()},
		{"PowParams", (&PowParams{1, 2, 3, 4, 5, 6}).Encode()},
		{"SeedMember", (&SeedMember{PublicKey: []byte("k")}).Encode()},
		{"SeedSunset", (&SeedSunset{UseActiveWallets: true, ActiveWallets: 100, DeadlineEpoch: 24}).Encode()},
		{"MutableBound", (&MutableBound{"param", 1, 9}).Encode()},
		{"GenesisRecord", (&GenesisRecord{
			NetworkID: []byte("net"), ManifestHash: []byte("mh"), GenesisTime: 1,
			EpochZero: 2, ProtocolVersion: 1,
			SeedSunset: &SeedSunset{UseActiveWallets: true, ActiveWallets: 100, DeadlineEpoch: 24},
			InviteKey:  []byte("key"),
		}).Encode()},
		{"GenesisPackage", (&GenesisPackage{
			Manifest: &Manifest{
				NetworkID: []byte("net"), DisplayName: "n", Slug: "s", Profile: confProfile(),
				SeedCircle: []SeedMember{{PublicKey: []byte("k")}},
				SeedSunset: &SeedSunset{DeadlineEpoch: 24}, GenesisHash: []byte("gh"),
			},
			Genesis: &GenesisRecord{
				NetworkID: []byte("net"), ManifestHash: []byte("mh"), GenesisTime: 1,
				EpochZero: 2, ProtocolVersion: 1, InviteKey: []byte("key"),
			},
		}).EncodeGenesisPackage()},
	}
	for _, c := range cases {
		checkConformance(t, s, c.msg, c.enc)
	}
}

// TestProtoEnums — номера значений enum'ов кодека против .proto.
func TestProtoEnums(t *testing.T) {
	s := parseProtoFiles(t)
	codec := map[string]map[string]uint32{
		"TxType": {
			"TX_UNSPECIFIED": uint32(TxUnspecified), "REGISTER": uint32(Register),
			"VOUCH": uint32(Vouch), "MINT": uint32(Mint), "TRANSFER": uint32(Transfer),
			"MIGRATE": uint32(Migrate), "EVIDENCE": uint32(Evidence),
			"GUARDIAN_OFFER": uint32(GuardianOffer), "POOL_CREATE": uint32(PoolCreate),
			"POOL_DONATE": uint32(PoolDonate), "POOL_CLOSE": uint32(PoolClose),
			"POOL_CLAIM": uint32(PoolClaim), "REDEEM": uint32(Redeem), "RECEIPT": uint32(Receipt),
			"CONTRIBUTION_CLAIM":   uint32(ContributionClaim),
			"CONTRIBUTION_CONFIRM": uint32(ContributionConfirm),
			"UPDATE_GUARDIANS":     uint32(UpdateGuardians),
		},
		"WalletStatus": {
			"WALLET_UNSPECIFIED": uint32(StatusUnspecified), "REGISTERED": uint32(StatusRegistered),
			"ACTIVE": uint32(StatusActive), "EXPIRED": uint32(StatusExpired),
		},
		"PowAlgorithm": {
			"POW_UNSPECIFIED": uint32(PowUnspecified), "POW_LIGHT": uint32(PowLight),
			"POW_ARGON2ID": uint32(PowArgon2id),
		},
		"EvidenceKind": {
			"EVIDENCE_UNSPECIFIED":  uint32(EvidenceUnspecified),
			"EVIDENCE_EQUIVOCATION": uint32(EvidenceEquivocation),
			"EVIDENCE_DOUBLE_MINT":  uint32(EvidenceDoubleMint),
		},
		"ReleaseMode": {
			"RELEASE_UNSPECIFIED": uint32(PoolReleaseUnspecified),
			"RELEASE_ON_DEADLINE": uint32(PoolReleaseOnDeadline),
			"RELEASE_ON_GOAL":     uint32(PoolReleaseOnGoal),
			"RELEASE_CONTINUOUS":  uint32(PoolReleaseContinuous),
		},
		"RefundMode": {
			"REFUND_UNSPECIFIED": uint32(PoolRefundUnspecified),
			"REFUND_IF_UNMET":    uint32(PoolRefundIfUnmet),
			"REFUND_NONE":        uint32(PoolRefundNone),
		},
		"ModuleId": {
			"MODULE_UNSPECIFIED": 0, "MODULE_ATTEST": ModuleAttest, "MODULE_REDEEM": ModuleRedeem,
			"MODULE_POOL": ModulePool, "MODULE_CONTRIBUTION": ModuleContribution,
			"MODULE_SANDBOX": ModuleSandbox,
		},
		"UnitType": {
			"UNIT_UNSPECIFIED": uint32(UnitUnspecified), "UNIT_CURRENCY": uint32(UnitCurrency),
			"UNIT_HOURS": uint32(UnitHours), "UNIT_QUOTA": uint32(UnitQuota),
			"UNIT_POINTS": uint32(UnitPoints), "" + "UNIT_VOUCHER": uint32(UnitVoucher),
			"UNIT_HEARTBEAT": uint32(UnitHeartbeat),
		},
		"TransferMode": {
			"TRANSFER_MODE_UNSPECIFIED": uint32(TransferModeUnspecified),
			"TRANSFER_MODE_FREE":        uint32(TransferModeFree),
			"TRANSFER_MODE_RESTRICTED":  uint32(TransferModeRestricted),
			"TRANSFER_MODE_SOULBOUND":   uint32(TransferModeSoulbound),
			"TRANSFER_MODE_PEER_REDEEM": uint32(TransferModePeerRedeem),
		},
		"IssuanceMode": {
			"ISSUANCE_UNSPECIFIED":     uint32(IssuanceUnspecified),
			"ISSUANCE_PERIODIC_EQUAL":  uint32(IssuancePeriodicEqual),
			"ISSUANCE_QUOTA_RESET":     uint32(IssuanceQuotaReset),
			"ISSUANCE_ON_CONTRIBUTION": uint32(IssuanceOnContribution),
			"ISSUANCE_POOL_CLAIM":      uint32(IssuancePoolClaim),
			"ISSUANCE_HEARTBEAT":       uint32(IssuanceHeartbeat),
			"ISSUANCE_NONE":            uint32(IssuanceNone),
		},
		"FeeMode": {
			"FEE_MODE_UNSPECIFIED": uint32(FeeModeUnspecified), "FEE_MODE_BURN": uint32(FeeModeBurn),
			"FEE_MODE_POOL": uint32(FeeModePool), "FEE_MODE_NONE": uint32(FeeModeNone),
		},
		"EntryMode": {
			"ENTRY_UNSPECIFIED": uint32(EntryUnspecified), "ENTRY_OPEN_VOUCH": uint32(EntryOpenVouch),
			"ENTRY_FOUNDERS_SEED":     uint32(EntryFoundersSeed),
			"ENTRY_INVITE_ONLY_VOUCH": uint32(EntryInviteOnlyVouch),
		},
		"DevicePolicy": {
			"DEVICE_POLICY_UNSPECIFIED": uint32(DevicePolicyUnspecified),
			"DEVICE_POLICY_STRICT":      uint32(DevicePolicyStrict),
			"DEVICE_POLICY_RELAXED":     uint32(DevicePolicyRelaxed),
		},
		"SigningAuth": {
			"SIGNING_AUTH_UNSPECIFIED":       uint32(SigningAuthUnspecified),
			"SIGNING_AUTH_BIOMETRIC_STRONG":  uint32(SigningAuthBiometricStrong),
			"SIGNING_AUTH_DEVICE_CREDENTIAL": uint32(SigningAuthDeviceCredential),
		},
		"NetworkAccess": {
			"NETWORK_ACCESS_UNSPECIFIED": uint32(NetworkAccessUnspecified),
			"NETWORK_ACCESS_PUBLIC":      uint32(NetworkAccessPublic),
			"NETWORK_ACCESS_INVITE_ONLY": uint32(NetworkAccessInviteOnly),
		},
		"CheckpointMode": {
			"CHECKPOINT_MODE_UNSPECIFIED":    uint32(CheckpointModeUnspecified),
			"CHECKPOINT_MODE_DECENTRALIZED":  uint32(CheckpointModeDecentralized),
			"CHECKPOINT_MODE_MEMBERS_QUORUM": uint32(CheckpointModeMembersQuorum),
		},
		"TimeScale": {
			"TIME_SCALE_UNSPECIFIED": uint32(TimeScaleUnspecified),
			"TIME_SCALE_REAL":        uint32(TimeScaleReal),
			"TIME_SCALE_ACCELERATED": uint32(TimeScaleAccelerated),
		},
	}
	names := make([]string, 0, len(codec))
	for name := range codec {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pe, ok := s.enums[name]
		if !ok {
			t.Errorf("enum %s отсутствует в proto/*.proto", name)
			continue
		}
		for v, n := range codec[name] {
			if pn, ok := pe[v]; !ok {
				t.Errorf("enum %s: значение %s кодека нет в .proto", name, v)
			} else if pn != n {
				t.Errorf("enum %s: %s = %d у кодека, %d в .proto", name, v, n, pn)
			}
		}
		for v, n := range pe {
			if _, ok := codec[name][v]; !ok {
				t.Errorf("enum %s: значение %s (= %d) есть в .proto, но не в кодеке", name, v, n)
			}
		}
	}
}
