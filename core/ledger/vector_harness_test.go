// Harness тестовых векторов (vectors/*.json): материализация плейсхолдеров
// (ключи, подписи, PoW, синтетические цепочки аттестации), конструирование
// pre_state и проверка ожиданий. Формат — vectors/README.md (tnp-vector-1).
package ledger

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// ---------------------------------------------------------------------------
// Структуры вектора
// ---------------------------------------------------------------------------

type vecParams map[string]json.RawMessage

type vecWallet struct {
	Status            string   `json:"status"`
	RegisteredAt      uint64   `json:"registered_at"`
	ActivatedAt       uint64   `json:"activated_at"`
	ActivatedPeriod   uint64   `json:"activated_period"`
	Balance           uint64   `json:"balance"`
	Seq               uint64   `json:"seq"`
	LastMintPeriod    uint64   `json:"last_mint_period"`
	SlotsUsed         int      `json:"slots_used"`
	MintPeriodsDone   uint64   `json:"mint_periods_done"`
	VouchCount        int      `json:"vouch_count"`
	Activators        []string `json:"activators"`
	Flags             []string `json:"flags"`
	DelayedThreshold  uint64   `json:"delayed_threshold"`
	PersonalGuardians []string `json:"personal_guardians"`
	Wards             uint64   `json:"wards"`
	PublicGuardian    bool     `json:"public_guardian"`
	BalanceLocked     uint64   `json:"balance_locked"`
	VouchLinks        []string `json:"vouch_links"`
}

type vecTx struct {
	ID              string `json:"id"`
	Wallet          string `json:"wallet"`
	Seq             uint64 `json:"seq"`
	Prev            string `json:"prev"`
	Type            string `json:"type"`
	TimestampClaim  uint64 `json:"timestamp_claim"`
	ProtocolVersion uint32 `json:"protocol_version"`
	Pow             struct {
		Algorithm string `json:"algorithm"`
		Nonce     string `json:"nonce"`
	} `json:"pow"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

type vecExpected struct {
	Tx      string          `json:"tx"`
	Outcome string          `json:"outcome"`
	Reason  string          `json:"reason"`
	Effects json.RawMessage `json:"effects"`
}

type vecKnownTx struct {
	Wallet         string          `json:"wallet"`
	Seq            uint64          `json:"seq"`
	Prev           string          `json:"prev"`
	Type           string          `json:"type"`
	TimestampClaim uint64          `json:"timestamp_claim"`
	Payload        json.RawMessage `json:"payload"`
}

type vector struct {
	Schema  string `json:"schema"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Source  string `json:"source"`
	Network struct {
		Params vecParams `json:"params"`
	} `json:"network"`
	PreState struct {
		Epoch            uint64                `json:"epoch"`
		NetworkTime      uint64                `json:"network_time"`
		CurrentPeriod    uint64                `json:"current_period"`
		Wallets          map[string]*vecWallet `json:"wallets"`
		Notes            []string              `json:"notes"`
		KnownConflicting *struct {
			Note string      `json:"note"`
			TxA  *vecKnownTx `json:"tx_a"`
			TxB  *vecKnownTx `json:"tx_b"`
		} `json:"known_conflicting_transactions"`
	} `json:"pre_state"`
	Transactions []vecTx       `json:"transactions"`
	Expected     []vecExpected `json:"expected"`
}

// ---------------------------------------------------------------------------
// Материализация
// ---------------------------------------------------------------------------

// harness — окружение одного вектора.
type harness struct {
	t        *testing.T
	vec      *vector
	ledger   *Ledger
	clock    *FixedClock
	keys     map[string]*ecdsa.PrivateKey
	addrs    map[string][]byte
	rootKey  *ecdsa.PrivateKey
	rootCert *wirePlaceholderCert
	builtTxs map[string]*wire.Transaction
}

type wirePlaceholderCert struct {
	cert interface{}
}

func num(p vecParams, name string, def uint64) uint64 {
	raw, ok := p[name]
	if !ok {
		return def
	}
	var v uint64
	if err := json.Unmarshal(raw, &v); err != nil {
		return def
	}
	return v
}

func str(p vecParams, name string) string {
	raw, ok := p[name]
	if !ok {
		return ""
	}
	var v string
	json.Unmarshal(raw, &v)
	return v
}

// buildGenesis — тестовый genesis с переопределениями из network.params вектора.
func (h *harness) buildGenesis() *wire.GenesisPackage {
	gp := params.TestGenesis()
	prof := gp.Manifest.Profile
	p := h.vec.Network.Params

	prof.IssuanceAmount = num(p, "issuance_amount", prof.IssuanceAmount)
	prof.AccrualWindowPeriods = num(p, "accrual_window_periods", prof.AccrualWindowPeriods)
	if v, ok := p["vouch_k"]; ok {
		json.Unmarshal(v, &prof.VouchK)
	}
	prof.VoucherMinAgeDays = num(p, "voucher_min_age_days", prof.VoucherMinAgeDays)
	if v, ok := p["voucher_min_mint_periods"]; ok {
		json.Unmarshal(v, &prof.VoucherMinMintPeriods)
	}
	prof.VoucherRewardMicros = num(p, "voucher_reward_micros", prof.VoucherRewardMicros)
	if v, ok := p["voucher_reward_periods"]; ok {
		json.Unmarshal(v, &prof.VoucherRewardPeriods)
	}
	prof.TransferMoratoriumDays = num(p, "transfer_moratorium_days", prof.TransferMoratoriumDays)
	prof.TransferDelayHours = num(p, "transfer_delay_hours", prof.TransferDelayHours)
	prof.DelayedThresholdDefault = num(p, "delayed_threshold_default", prof.DelayedThresholdDefault)
	prof.DelayedThresholdMax = num(p, "delayed_threshold_max", prof.DelayedThresholdMax)
	switch str(p, "fee_mode") {
	case "NONE":
		prof.FeeMode = wire.FeeModeNone
	case "POOL":
		prof.FeeMode = wire.FeeModePool
	case "":
		// остаётся BURN
	default:
		prof.FeeMode = wire.FeeModeBurn
	}
	prof.FeeMin = num(p, "fee_min", prof.FeeMin)
	prof.RateMin = num(p, "rate_micros", prof.RateMin) // текущая ставка (заглушка этапа 3)
	prof.RateMax = num(p, "rate_micros", prof.RateMax)
	prof.DemurrageRateMicros = num(p, "demurrage_rate_micros", prof.DemurrageRateMicros)
	prof.DemurrageFloor = num(p, "demurrage_floor", prof.DemurrageFloor)
	prof.RegisteredExpiryDays = num(p, "registered_expiry_days", prof.RegisteredExpiryDays)
	prof.SlotFreeDays = num(p, "slot_free_days", prof.SlotFreeDays)
	if v, ok := p["slot_free_mint_periods"]; ok {
		json.Unmarshal(v, &prof.SlotFreeMintPeriods)
	}
	prof.GuardianPersonalThreshold = num(p, "guardian_personal_threshold", prof.GuardianPersonalThreshold)
	prof.GuardianPersonalCount = num(p, "guardian_personal_count", prof.GuardianPersonalCount)
	prof.GuardianPublicThreshold = num(p, "guardian_public_threshold", prof.GuardianPublicThreshold)
	prof.GuardianPublicCount = num(p, "guardian_public_count", prof.GuardianPublicCount)
	prof.GuardianMinAgeDays = num(p, "guardian_min_age_days", prof.GuardianMinAgeDays)
	prof.GuardianMaxWards = num(p, "guardian_max_wards", prof.GuardianMaxWards)
	prof.MigrateDelayDays = num(p, "migrate_delay_days", prof.MigrateDelayDays)
	prof.TimestampToleranceSeconds = num(p, "timestamp_tolerance_seconds", prof.TimestampToleranceSeconds)
	prof.MintChallengeWindowHours = num(p, "mint_challenge_window_hours", prof.MintChallengeWindowHours)

	// Таргеты PoW для тестов: малые (вопрос №9 — значения продакшена не выбраны)
	prof.Pow = &wire.PowParams{
		TargetLight:       0x0010_0000_0000_0000, // ~2^8 попыток
		Argon2MemoryKiB:   8,
		Argon2Iterations:  1,
		Argon2Parallelism: 1,
		TargetHard:        0x0010_0000_0000_0000,
	}
	// пересчитать manifest_hash после переопределений
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	return gp
}

func (h *harness) key(id string) *ecdsa.PrivateKey {
	if k, ok := h.keys[id]; ok {
		return k
	}
	k := crypto.DeterministicKey(h.vec.ID + ":" + id)
	h.keys[id] = k
	return k
}

func (h *harness) addr(id string) []byte {
	if a, ok := h.addrs[id]; ok {
		return a
	}
	pub := crypto.MarshalPubKey(&h.key(id).PublicKey)
	fp := crypto.Fingerprint(pub)
	a := crypto.DeriveAddress(fp, h.ledger.State.Net.NetworkID)
	h.addrs[id] = a
	return a
}

// virtualTip — txid виртуальной цепочки до вектора (pre_state задаёт seq без истории).
func (h *harness) virtualTip(id string, seq uint64) []byte {
	sum := sha256.Sum256([]byte(fmt.Sprintf("vector-tip:%s:%s:%d", h.vec.ID, id, seq)))
	return sum[:]
}

// newHarness — построить окружение: genesis, pre_state, ключи, корень аттестации.
func newHarness(t *testing.T, vec *vector) *harness {
	h := &harness{
		t:        t,
		vec:      vec,
		keys:     make(map[string]*ecdsa.PrivateKey),
		addrs:    make(map[string][]byte),
		builtTxs: make(map[string]*wire.Transaction),
	}
	gp := h.buildGenesis()
	h.clock = &FixedClock{T: vec.PreState.NetworkTime, E: vec.PreState.Epoch, P: vec.PreState.CurrentPeriod}
	l, err := New(gp, h.clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	h.ledger = l
	// pre_state — состояние НА момент вектора: периоды/эпохи уже материализованы,
	// сеттлемент «догоном» не выполняется
	l.State.CurrentPeriod = vec.PreState.CurrentPeriod
	l.State.Epoch = vec.PreState.Epoch

	// синтетический корень аттестации (уникальный на вектор)
	h.rootKey = crypto.DeterministicKey("attest-root:" + vec.ID)
	cert, _, err := attest.SyntheticRoot(vec.ID, h.rootKey)
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	l.TrustedRoots = append(l.TrustedRoots, cert)
	h.rootCert = &wirePlaceholderCert{cert}

	// pre_state: кошельки
	for id, vw := range vec.PreState.Wallets {
		w := &Wallet{
			Address:           h.addr(id),
			Status:            wire.StatusUnspecified,
			RegisteredAt:      vw.RegisteredAt,
			ActivatedAt:       vw.ActivatedAt,
			ActivatedPeriod:   vw.ActivatedPeriod,
			Liquid:            vw.Balance,
			Locked:            vw.BalanceLocked,
			Seq:               vw.Seq,
			LastClaimedPeriod: vw.LastMintPeriod,
			DelayedThreshold:  vw.DelayedThreshold,
			PublicGuardian:    vw.PublicGuardian,
			Wards:             vw.Wards,
			LastTouchEpoch:    vec.PreState.Epoch,
		}
		w.PublicKey = &h.key(id).PublicKey
		pub := crypto.MarshalPubKey(w.PublicKey)
		w.Fingerprint = crypto.Fingerprint(pub)
		switch vw.Status {
		case "ACTIVE":
			w.Status = wire.StatusActive
		case "REGISTERED":
			w.Status = wire.StatusRegistered
		case "EXPIRED":
			w.Status = wire.StatusExpired
		}
		for _, f := range vw.Flags {
			if f == "FLAGGED" {
				w.Flagged = true
			}
		}
		// выполненные периоды выпуска (V3): если задано mint_periods_done
		if vw.MintPeriodsDone > 0 {
			w.ActivatedPeriod = 1
			w.LastClaimedPeriod = vw.MintPeriodsDone
		}
		// занятые слоты (фиктивные кандидаты)
		for i := 0; i < vw.SlotsUsed; i++ {
			w.OccupiedSlots = append(w.OccupiedSlots, Slot{
				Candidate:    h.addr(fmt.Sprintf("%s-slot-%d", id, i)),
				SinceNetTime: vec.PreState.NetworkTime,
			})
		}
		// поручительства полученные
		nVouch := vw.VouchCount
		if nVouch == 0 && len(vw.Activators) > 0 {
			nVouch = len(vw.Activators)
		}
		for i := 0; i < nVouch; i++ {
			voucher := fmt.Sprintf("%s-v%d", id, i)
			if i < len(vw.Activators) {
				voucher = vw.Activators[i]
			}
			w.VouchesReceived = append(w.VouchesReceived, VouchReceipt{
				Voucher: h.addr(voucher),
				Epoch:   vec.PreState.Epoch,
			})
		}
		if w.Status == wire.StatusActive && len(w.Activators) > 0 {
			for _, a := range vw.Activators {
				w.Activators = append(w.Activators, h.addr(a))
			}
		} else if w.Status == wire.StatusActive && len(w.VouchesReceived) > 0 {
			for _, r := range w.VouchesReceived {
				w.Activators = append(w.Activators, r.Voucher)
			}
		}
		for _, g := range vw.PersonalGuardians {
			w.PersonalGuardians = append(w.PersonalGuardians, h.addr(g))
		}
		if vw.Seq > 0 {
			w.TipTxID = h.virtualTip(id, vw.Seq)
		}
		l.State.Wallets[string(w.Address)] = w
		l.State.FingerprintIndex[string(w.Fingerprint)] = w.Address
	}
	return h
}

// buildAttestChain — синтетическая цепочка аттестации для кошелька.
func (h *harness) buildAttestChain(id string) (pubKey, chain, challenge []byte) {
	priv := h.key(id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	challenge = attest.ChallengeFormula(pub, crypto.ProtocolID, h.ledger.State.Net.NetworkID)
	// №43 (этап 9): дата патча — в прошлом относительно сетевого
	// времени вектора (future-patch отклоняется Verify)
	patch := time.Unix(int64(h.clock.NetTime()), 0).UTC().AddDate(0, -2, 0)
	ext := attest.Extension{
		SecurityLevel:     attest.LevelTEE,
		VerifiedBootState: attest.BootVerified,
		DeviceLocked:      true,
		PatchYear:         patch.Year(),
		PatchMonth:        int(patch.Month()),
	}
	serial := sha256.Sum256([]byte("serial:" + h.vec.ID + ":" + id))
	leaf, err := attest.BuildLeaf(&priv.PublicKey, challenge, ext, h.rootKey,
		h.rootCert.cert.(*x509.Certificate), int64(serial[0])%100000+1)
	if err != nil {
		h.t.Fatalf("attest leaf: %v", err)
	}
	return pub, attest.EncodeChain(leaf), challenge
}

// resolvePrev — prev по ссылке "txid:W:seq" (tip кошелька) или "0^32".
func (h *harness) resolvePrev(ref string) []byte {
	if ref == "" || ref == "0^32" {
		return nil
	}
	parts := strings.SplitN(ref, ":", 3)
	if len(parts) == 3 && parts[0] == "txid" {
		w := h.ledger.State.Wallet(h.addr(parts[1]))
		if w != nil && w.TipTxID != nil {
			return w.TipTxID
		}
		return nil
	}
	h.t.Fatalf("неизвестная ссылка prev: %q", ref)
	return nil
}

// buildTx — материализация транзакции вектора.
func (h *harness) buildTx(vt *vecTx, known map[string]*wire.Transaction) *wire.Transaction {
	now := h.clock.NetTime()
	// правило часов: tx в допуске не двигает сетевое время; иначе — сдвиг
	tol := h.ledger.State.Net.Profile.TimestampToleranceSeconds
	if vt.TimestampClaim+tol < now || now+tol < vt.TimestampClaim {
		h.clock.T = vt.TimestampClaim
	}
	h.ledger.Clock = h.clock

	tx := &wire.Transaction{
		NetworkID:       h.ledger.State.Net.NetworkID,
		Wallet:          h.addr(vt.Wallet),
		Seq:             vt.Seq,
		Prev:            h.resolvePrev(vt.Prev),
		TimestampClaim:  vt.TimestampClaim,
		ProtocolVersion: vt.ProtocolVersion,
	}
	if tx.ProtocolVersion == 0 {
		tx.ProtocolVersion = h.ledger.State.Net.ProtocolVersion
	}
	hard := false
	switch vt.Type {
	case "REGISTER":
		tx.Type = wire.Register
		hard = true
		var p struct {
			Register struct {
				AttestedPublicKey    string   `json:"attested_public_key"`
				AttestationChain     string   `json:"attestation_chain"`
				AttestationChallenge string   `json:"attestation_challenge"`
				AndroidID            string   `json:"android_id"`
				PersonalGuardians    []string `json:"personal_guardians"`
				Synthetic            bool     `json:"synthetic"`
			} `json:"register"`
		}
		if err := json.Unmarshal(vt.Payload, &p); err != nil {
			h.t.Fatalf("payload register: %v", err)
		}
		pub, chain, challenge := h.buildAttestChain(vt.Wallet)
		rp := &wire.RegisterPayload{
			AttestedPublicKey:    pub,
			AttestationChain:     chain,
			AttestationChallenge: challenge,
			AndroidID:            p.Register.AndroidID,
			Synthetic:            p.Register.Synthetic,
		}
		for _, g := range p.Register.PersonalGuardians {
			rp.PersonalGuardians = append(rp.PersonalGuardians, h.addr(g))
		}
		tx.Payload = rp
	case "VOUCH":
		tx.Type = wire.Vouch
		var p struct {
			Vouch struct {
				Candidate string `json:"candidate"`
				Epoch     uint64 `json:"epoch"`
			} `json:"vouch"`
		}
		json.Unmarshal(vt.Payload, &p)
		tx.Payload = &wire.VouchPayload{Candidate: h.addr(p.Vouch.Candidate), Epoch: p.Vouch.Epoch}
	case "MINT":
		tx.Type = wire.Mint
		hard = true
		var p struct {
			Mint struct {
				Period uint64 `json:"period"`
				Amount uint64 `json:"amount"`
			} `json:"mint"`
		}
		json.Unmarshal(vt.Payload, &p)
		tx.Payload = &wire.MintPayload{Period: p.Mint.Period, Amount: p.Mint.Amount}
	case "TRANSFER":
		tx.Type = wire.Transfer
		var p struct {
			Transfer struct {
				Send *struct {
					Recipient string `json:"recipient"`
					Amount    uint64 `json:"amount"`
				} `json:"send"`
				Cancel *struct {
					PendingSeq uint64 `json:"pending_seq"`
				} `json:"cancel"`
				SetThreshold *struct {
					NewThreshold uint64 `json:"new_threshold"`
				} `json:"set_threshold"`
			} `json:"transfer"`
		}
		if err := json.Unmarshal(vt.Payload, &p); err != nil {
			h.t.Fatalf("payload transfer: %v", err)
		}
		tp := &wire.TransferPayload{}
		if p.Transfer.Send != nil {
			tp.Send = &wire.TransferSend{Recipient: h.addr(p.Transfer.Send.Recipient), Amount: p.Transfer.Send.Amount}
		}
		if p.Transfer.Cancel != nil {
			tp.Cancel = &wire.TransferCancel{PendingSeq: p.Transfer.Cancel.PendingSeq}
		}
		if p.Transfer.SetThreshold != nil {
			tp.SetThreshold = &wire.ThresholdUpdate{NewThreshold: p.Transfer.SetThreshold.NewThreshold}
		}
		tx.Payload = tp
	case "MIGRATE":
		tx.Type = wire.Migrate
		var p struct {
			Migrate struct {
				Propose *struct {
					OldAddress string `json:"old_address"`
					Approvals  []struct {
						Guardian     string `json:"guardian"`
						StatementSig string `json:"statement_sig"`
					} `json:"approvals"`
					Nonce uint64 `json:"nonce"`
				} `json:"propose"`
				Cancel *struct {
					NewAddress string `json:"new_address"`
					ProposeSeq uint64 `json:"propose_seq"`
				} `json:"cancel"`
			} `json:"migrate"`
		}
		if err := json.Unmarshal(vt.Payload, &p); err != nil {
			h.t.Fatalf("payload migrate: %v", err)
		}
		mp := &wire.MigratePayload{}
		if p.Migrate.Propose != nil {
			pro := &wire.MigratePropose{
				OldAddress: h.addr(p.Migrate.Propose.OldAddress),
				Nonce:      p.Migrate.Propose.Nonce,
			}
			for _, ap := range p.Migrate.Propose.Approvals {
				stmt := &wire.MigrateStatement{
					NetworkID:  h.ledger.State.Net.NetworkID,
					OldAddress: h.addr(p.Migrate.Propose.OldAddress),
					NewAddress: h.addr(vt.Wallet),
					Nonce:      p.Migrate.Propose.Nonce,
					Epoch:      h.vec.PreState.Epoch,
				}
				sh := stmt.Hash()
				sig, err := crypto.SignDER(h.key(ap.Guardian), sh[:])
				if err != nil {
					h.t.Fatalf("guardian sig: %v", err)
				}
				pro.Approvals = append(pro.Approvals, wire.GuardianApproval{
					Guardian:     h.addr(ap.Guardian),
					StatementSig: sig,
				})
			}
			mp.Propose = pro
		}
		if p.Migrate.Cancel != nil {
			mp.Cancel = &wire.MigrateCancel{
				NewAddress: h.addr(p.Migrate.Cancel.NewAddress),
				ProposeSeq: p.Migrate.Cancel.ProposeSeq,
			}
		}
		tx.Payload = mp
	case "EVIDENCE":
		tx.Type = wire.Evidence
		var p struct {
			Evidence struct {
				TxA  string `json:"tx_a"`
				TxB  string `json:"tx_b"`
				Kind string `json:"kind"`
			} `json:"evidence"`
		}
		json.Unmarshal(vt.Payload, &p)
		ep := &wire.EvidencePayload{}
		var ok1, ok2 bool
		ep.TxA, ok1 = known[p.Evidence.TxA]
		ep.TxB, ok2 = known[p.Evidence.TxB]
		if !ok1 || !ok2 {
			h.t.Fatalf("evidence: неизвестные ссылки %q/%q", p.Evidence.TxA, p.Evidence.TxB)
		}
		switch p.Evidence.Kind {
		case "EVIDENCE_DOUBLE_MINT":
			ep.Kind = wire.EvidenceDoubleMint
		default:
			ep.Kind = wire.EvidenceEquivocation
		}
		tx.Payload = ep
	case "GUARDIAN_OFFER":
		tx.Type = wire.GuardianOffer
		tx.Payload = &wire.GuardianOfferPayload{}
	default:
		h.t.Fatalf("неизвестный тип %q", vt.Type)
	}

	// PoW: решается на том же preimage, что и верификация (включая поле pow)
	h.solvePow(tx, hard)
	// подпись
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(h.key(vt.Wallet), sh[:])
	if err != nil {
		h.t.Fatalf("sign: %v", err)
	}
	tx.Signature = sig
	return tx
}

// buildKnownTx — материализация конфликтной пары для EVIDENCE.
func (h *harness) buildKnownTx(id string, kt *vecKnownTx) *wire.Transaction {
	vt := &vecTx{
		ID:              id,
		Wallet:          kt.Wallet,
		Seq:             kt.Seq,
		Prev:            kt.Prev,
		Type:            kt.Type,
		TimestampClaim:  kt.TimestampClaim,
		ProtocolVersion: h.ledger.State.Net.ProtocolVersion,
		Payload:         kt.Payload,
	}
	return h.buildTx(vt, nil)
}

// solvePow подбирает nonce так, чтобы проверка ядра сошлась: preimage
// включает поле pow (spec/01 1.2: signing_hash = SHA-256(preimage || nonce)).
func (h *harness) solvePow(tx *wire.Transaction, hard bool) {
	net := h.ledger.State.Net
	if hard {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowArgon2id}
		ap := crypto.Argon2Params{
			MemoryKiB: net.Pow.Argon2MemoryKiB, Iterations: net.Pow.Argon2Iterations, Parallelism: net.Pow.Argon2Parallelism}
		for n := uint64(0); ; n++ {
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], n)
			tx.Pow.Nonce = nb[:]
			if crypto.VerifyHard(tx.SigningPreimage(), nb[:], net.Pow.TargetHard, ap) {
				return
			}
		}
	}
	tx.Pow = &wire.PowProof{Algorithm: wire.PowLight}
	for n := uint64(0); ; n++ {
		var nb [8]byte
		binary.BigEndian.PutUint64(nb[:], n)
		tx.Pow.Nonce = nb[:]
		if crypto.VerifyLight(tx.SigningPreimage(), nb[:], net.Pow.TargetLight) {
			return
		}
	}
}

var _ = bytes.Equal
var _ = filepath.Join
var _ = os.ReadFile
var _ = strings.Split
