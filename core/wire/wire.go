// Пакет wire: canonical-кодирование protobuf и структуры сообщений протокола.
// Источник: spec/01-wire-format.md, proto/*.proto.
//
// Canonical-кодирование (spec/00 0.2): поля сериализуются в порядке номеров,
// unknown-поля запрещены, proto3-правило нулевых значений (нулевые скаляры,
// пустые bytes/строки и unset-сообщения не кодируются). Кодек написан вручную
// (без protoc) именно для гарантии детерминизма байтов: хеши txid/signing_hash
// обязаны совпадать между узлами (spec/01 1.2, тест стабильности).
package wire

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// Типы транзакций (proto/core.proto; закрытый перечень 9.3)
// ---------------------------------------------------------------------------

type TxType uint32

const (
	TxUnspecified       TxType = 0
	Register            TxType = 1
	Vouch               TxType = 2
	Mint                TxType = 3
	Transfer            TxType = 4
	Migrate             TxType = 5
	Evidence            TxType = 6
	GuardianOffer       TxType = 7
	PoolCreate          TxType = 8
	PoolDonate          TxType = 9
	PoolClose           TxType = 10
	PoolClaim           TxType = 11
	Redeem              TxType = 12
	Receipt             TxType = 13
	ContributionClaim   TxType = 14
	ContributionConfirm TxType = 15
)

func (t TxType) String() string {
	switch t {
	case Register:
		return "REGISTER"
	case Vouch:
		return "VOUCH"
	case Mint:
		return "MINT"
	case Transfer:
		return "TRANSFER"
	case Migrate:
		return "MIGRATE"
	case Evidence:
		return "EVIDENCE"
	case GuardianOffer:
		return "GUARDIAN_OFFER"
	case PoolCreate:
		return "POOL_CREATE"
	case PoolDonate:
		return "POOL_DONATE"
	case PoolClose:
		return "POOL_CLOSE"
	case PoolClaim:
		return "POOL_CLAIM"
	case Redeem:
		return "REDEEM"
	case Receipt:
		return "RECEIPT"
	case ContributionClaim:
		return "CONTRIBUTION_CLAIM"
	case ContributionConfirm:
		return "CONTRIBUTION_CONFIRM"
	default:
		return "TX_UNSPECIFIED"
	}
}

type WalletStatus uint32

const (
	StatusUnspecified WalletStatus = 0
	StatusRegistered  WalletStatus = 1
	StatusActive      WalletStatus = 2
	StatusExpired     WalletStatus = 3
)

func (s WalletStatus) String() string {
	switch s {
	case StatusRegistered:
		return "REGISTERED"
	case StatusActive:
		return "ACTIVE"
	case StatusExpired:
		return "EXPIRED"
	default:
		return "WALLET_UNSPECIFIED"
	}
}

// Метки кошелька (proto/core.proto WalletFlags).
type WalletFlags struct {
	Flagged   bool
	Synthetic bool
	Seed      bool
}

type PowAlgorithm uint32

const (
	PowUnspecified PowAlgorithm = 0
	PowLight       PowAlgorithm = 1
	PowArgon2id    PowAlgorithm = 2
)

type PowProof struct {
	Algorithm PowAlgorithm
	Nonce     []byte
}

type TimeSlot struct {
	Start uint64
	End   uint64
}

// ---------------------------------------------------------------------------
// Payload-сообщения (proto/transactions.proto)
// ---------------------------------------------------------------------------

type RegisterPayload struct {
	AttestedPublicKey    []byte
	AttestationChain     []byte // DER: лист + промежуточные (одним блобом)
	AttestationChallenge []byte
	AndroidID            string
	PersonalGuardians    [][]byte
	Synthetic            bool
}

type VouchPayload struct {
	Candidate []byte
	Epoch     uint64
}

type MintPayload struct {
	Period uint64
	Amount uint64
}

type TransferSend struct {
	Recipient []byte
	Amount    uint64
}

type TransferCancel struct {
	PendingSeq uint64
}

type ThresholdUpdate struct {
	NewThreshold uint64
}

type TransferPayload struct {
	// ровно один из трёх не nil (решение D2, spec/00 0.6)
	Send         *TransferSend
	Cancel       *TransferCancel
	SetThreshold *ThresholdUpdate
}

type GuardianApproval struct {
	Guardian     []byte
	StatementSig []byte
}

type MigratePropose struct {
	OldAddress []byte
	Approvals  []GuardianApproval
	Nonce      uint64
}

type MigrateCancel struct {
	NewAddress []byte
	ProposeSeq uint64
}

type MigratePayload struct {
	Propose *MigratePropose
	Cancel  *MigrateCancel
}

type EvidenceKind uint32

const (
	EvidenceUnspecified  EvidenceKind = 0
	EvidenceEquivocation EvidenceKind = 1
	EvidenceDoubleMint   EvidenceKind = 2
)

type EvidencePayload struct {
	TxA  *Transaction
	TxB  *Transaction
	Kind EvidenceKind
}

type GuardianOfferPayload struct{}

// Payload — интерфейс маркировки payload-ветвей oneof.
type Payload interface{ payloadTag() uint32 }

func (*RegisterPayload) payloadTag() uint32      { return 10 }
func (*VouchPayload) payloadTag() uint32         { return 11 }
func (*MintPayload) payloadTag() uint32          { return 12 }
func (*TransferPayload) payloadTag() uint32      { return 13 }
func (*MigratePayload) payloadTag() uint32       { return 14 }
func (*EvidencePayload) payloadTag() uint32      { return 15 }
func (*GuardianOfferPayload) payloadTag() uint32 { return 16 }

// ---------------------------------------------------------------------------
// Transaction (конверт, proto/transactions.proto)
// ---------------------------------------------------------------------------

type Transaction struct {
	NetworkID       []byte
	Wallet          []byte
	Seq             uint64
	Prev            []byte
	Type            TxType
	TimestampClaim  uint64
	ProtocolVersion uint32
	Pow             *PowProof
	Signature       []byte
	Payload         Payload
}

// SigningPreimage возвращает canonical-кодирование транзакции с пустой
// подписью (spec/01 1.2).
func (t *Transaction) SigningPreimage() []byte {
	cp := *t
	cp.Signature = nil
	return cp.Encode()
}

// SigningHash = SHA-256(signing_preimage || pow.nonce) (spec/01 1.2).
// Транзакция без PoW невалидна (C10/C11), но хеш обязан вычисляться без паники.
func (t *Transaction) SigningHash() [32]byte {
	pre := t.SigningPreimage()
	if t.Pow != nil {
		pre = append(pre, t.Pow.Nonce...)
	}
	return sha256.Sum256(pre)
}

// TxID = SHA-256(canonical(полная транзакция)) (spec/01 1.2).
func (t *Transaction) TxID() [32]byte {
	return sha256.Sum256(t.Encode())
}

// ---------------------------------------------------------------------------
// Примитивы protobuf (wire format)
// ---------------------------------------------------------------------------

func appendTag(b []byte, field uint32, wiretype byte) []byte {
	return appendVarint(b, uint64(field)<<3|uint64(wiretype))
}

func appendVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func appendBytesField(b []byte, field uint32, v []byte) []byte {
	if len(v) == 0 {
		return b // proto3: пустые bytes не кодируются
	}
	b = appendTag(b, field, 2)
	b = appendVarint(b, uint64(len(v)))
	return append(b, v...)
}

func appendStringField(b []byte, field uint32, v string) []byte {
	if v == "" {
		return b
	}
	return appendBytesField(b, field, []byte(v))
}

func appendU64Field(b []byte, field uint32, v uint64) []byte {
	if v == 0 {
		return b
	}
	b = appendTag(b, field, 0)
	return appendVarint(b, v)
}

func appendU32Field(b []byte, field uint32, v uint32) []byte {
	if v == 0 {
		return b
	}
	b = appendTag(b, field, 0)
	return appendVarint(b, uint64(v))
}

func appendBoolField(b []byte, field uint32, v bool) []byte {
	if !v {
		return b
	}
	b = appendTag(b, field, 0)
	return append(b, 1)
}

func appendMessageField(b []byte, field uint32, msg []byte) []byte {
	if len(msg) == 0 {
		return b
	}
	b = appendTag(b, field, 2)
	b = appendVarint(b, uint64(len(msg)))
	return append(b, msg...)
}

// ---------------------------------------------------------------------------
// Кодирование сообщений (поля строго в порядке номеров)
// ---------------------------------------------------------------------------

func (p *PowProof) Encode() []byte {
	var b []byte
	b = appendU32Field(b, 1, uint32(p.Algorithm))
	b = appendBytesField(b, 2, p.Nonce)
	return b
}

func (p *WalletFlags) Encode() []byte {
	var b []byte
	b = appendBoolField(b, 1, p.Flagged)
	b = appendBoolField(b, 2, p.Synthetic)
	b = appendBoolField(b, 3, p.Seed)
	return b
}

func (p *RegisterPayload) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.AttestedPublicKey)
	b = appendBytesField(b, 2, p.AttestationChain)
	b = appendBytesField(b, 3, p.AttestationChallenge)
	b = appendStringField(b, 4, p.AndroidID)
	for _, g := range p.PersonalGuardians {
		b = appendBytesField(b, 5, g)
	}
	b = appendBoolField(b, 6, p.Synthetic)
	return b
}

func (p *VouchPayload) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.Candidate)
	b = appendU64Field(b, 2, p.Epoch)
	return b
}

func (p *MintPayload) Encode() []byte {
	var b []byte
	b = appendU64Field(b, 1, p.Period)
	b = appendU64Field(b, 2, p.Amount)
	return b
}

func (p *TransferSend) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.Recipient)
	b = appendU64Field(b, 2, p.Amount)
	return b
}

func (p *TransferCancel) Encode() []byte {
	var b []byte
	b = appendU64Field(b, 1, p.PendingSeq)
	return b
}

func (p *ThresholdUpdate) Encode() []byte {
	var b []byte
	b = appendU64Field(b, 1, p.NewThreshold)
	return b
}

func (p *TransferPayload) Encode() []byte {
	var b []byte
	switch {
	case p.Send != nil:
		b = appendMessageField(b, 1, p.Send.Encode())
	case p.Cancel != nil:
		b = appendMessageField(b, 2, p.Cancel.Encode())
	case p.SetThreshold != nil:
		b = appendMessageField(b, 3, p.SetThreshold.Encode())
	}
	return b
}

func (p *GuardianApproval) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.Guardian)
	b = appendBytesField(b, 2, p.StatementSig)
	return b
}

func (p *MigratePropose) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.OldAddress)
	for i := range p.Approvals {
		b = appendMessageField(b, 2, p.Approvals[i].Encode())
	}
	b = appendU64Field(b, 3, p.Nonce)
	return b
}

func (p *MigrateCancel) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.NewAddress)
	b = appendU64Field(b, 2, p.ProposeSeq)
	return b
}

func (p *MigratePayload) Encode() []byte {
	var b []byte
	switch {
	case p.Propose != nil:
		b = appendMessageField(b, 1, p.Propose.Encode())
	case p.Cancel != nil:
		b = appendMessageField(b, 2, p.Cancel.Encode())
	}
	return b
}

func (p *GuardianOfferPayload) Encode() []byte { return nil }

func (p *EvidencePayload) Encode() []byte {
	var b []byte
	if p.TxA != nil {
		b = appendMessageField(b, 1, p.TxA.Encode())
	}
	if p.TxB != nil {
		b = appendMessageField(b, 2, p.TxB.Encode())
	}
	b = appendU32Field(b, 3, uint32(p.Kind))
	return b
}

func (t *Transaction) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, t.NetworkID)
	b = appendBytesField(b, 2, t.Wallet)
	b = appendU64Field(b, 3, t.Seq)
	b = appendBytesField(b, 4, t.Prev)
	b = appendU32Field(b, 5, uint32(t.Type))
	b = appendU64Field(b, 6, t.TimestampClaim)
	b = appendU32Field(b, 7, t.ProtocolVersion)
	if t.Pow != nil {
		b = appendMessageField(b, 8, t.Pow.Encode())
	}
	b = appendBytesField(b, 9, t.Signature)
	switch pl := t.Payload.(type) {
	case *RegisterPayload:
		b = appendMessageField(b, 10, pl.Encode())
	case *VouchPayload:
		b = appendMessageField(b, 11, pl.Encode())
	case *MintPayload:
		b = appendMessageField(b, 12, pl.Encode())
	case *TransferPayload:
		b = appendMessageField(b, 13, pl.Encode())
	case *MigratePayload:
		b = appendMessageField(b, 14, pl.Encode())
	case *EvidencePayload:
		b = appendMessageField(b, 15, pl.Encode())
	case *GuardianOfferPayload:
		b = appendMessageField(b, 16, pl.Encode())
	case nil:
	default:
		panic(fmt.Sprintf("wire: неизвестный payload %T", t.Payload))
	}
	return b
}

// ---------------------------------------------------------------------------
// Декодирование
// ---------------------------------------------------------------------------

var errUnknownField = errors.New("wire: unknown field (canonical: unknown-поля запрещены)")

type decoder struct {
	b   []byte
	pos int
}

func (d *decoder) tag() (field uint32, wiretype byte, err error) {
	v, err := d.varint()
	if err != nil {
		return 0, 0, err
	}
	return uint32(v >> 3), byte(v & 7), nil
}

func (d *decoder) varint() (uint64, error) {
	var v uint64
	var shift uint
	for {
		if d.pos >= len(d.b) {
			return 0, errors.New("wire: обрезанное varint")
		}
		c := d.b[d.pos]
		d.pos++
		v |= uint64(c&0x7f) << shift
		if c < 0x80 {
			return v, nil
		}
		shift += 7
		if shift > 63 {
			return 0, errors.New("wire: слишком длинное varint")
		}
	}
}

func (d *decoder) bytes() ([]byte, error) {
	n, err := d.varint()
	if err != nil {
		return nil, err
	}
	if uint64(len(d.b)-d.pos) < n {
		return nil, errors.New("wire: обрезанное bytes-поле")
	}
	v := d.b[d.pos : d.pos+int(n)]
	d.pos += int(n)
	return append([]byte(nil), v...), nil
}

func (d *decoder) skip(wiretype byte) error {
	switch wiretype {
	case 0:
		_, err := d.varint()
		return err
	case 2:
		_, err := d.bytes()
		return err
	default:
		return fmt.Errorf("wire: неподдерживаемый wiretype %d", wiretype)
	}
}

func (d *decoder) finish() error {
	if d.pos != len(d.b) {
		return errors.New("wire: лишние байты в конце сообщения")
	}
	return nil
}

func DecodePowProof(b []byte) (*PowProof, error) {
	d := &decoder{b: b}
	p := &PowProof{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Algorithm = PowAlgorithm(v)
		case f == 2 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			p.Nonce = v
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeRegisterPayload(b []byte) (*RegisterPayload, error) {
	d := &decoder{b: b}
	p := &RegisterPayload{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if p.AttestedPublicKey, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			if p.AttestationChain, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 3 && wt == 2:
			if p.AttestationChallenge, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 4 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			p.AndroidID = string(v)
		case f == 5 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			p.PersonalGuardians = append(p.PersonalGuardians, v)
		case f == 6 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Synthetic = v != 0
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeVouchPayload(b []byte) (*VouchPayload, error) {
	d := &decoder{b: b}
	p := &VouchPayload{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if p.Candidate, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 0:
			if p.Epoch, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeMintPayload(b []byte) (*MintPayload, error) {
	d := &decoder{b: b}
	p := &MintPayload{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			if p.Period, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 0:
			if p.Amount, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeTransferPayload(b []byte) (*TransferPayload, error) {
	d := &decoder{b: b}
	p := &TransferPayload{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			sub, err := DecodeTransferSend(v)
			if err != nil {
				return nil, err
			}
			if p.Send != nil || p.Cancel != nil || p.SetThreshold != nil {
				return nil, errors.New("wire: несколько action в TransferPayload")
			}
			p.Send = sub
		case f == 2 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			sub, err := DecodeTransferCancel(v)
			if err != nil {
				return nil, err
			}
			if p.Send != nil || p.Cancel != nil || p.SetThreshold != nil {
				return nil, errors.New("wire: несколько action в TransferPayload")
			}
			p.Cancel = sub
		case f == 3 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			sub, err := DecodeThresholdUpdate(v)
			if err != nil {
				return nil, err
			}
			if p.Send != nil || p.Cancel != nil || p.SetThreshold != nil {
				return nil, errors.New("wire: несколько action в TransferPayload")
			}
			p.SetThreshold = sub
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeTransferSend(b []byte) (*TransferSend, error) {
	d := &decoder{b: b}
	p := &TransferSend{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if p.Recipient, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 0:
			if p.Amount, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeTransferCancel(b []byte) (*TransferCancel, error) {
	d := &decoder{b: b}
	p := &TransferCancel{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			if p.PendingSeq, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeThresholdUpdate(b []byte) (*ThresholdUpdate, error) {
	d := &decoder{b: b}
	p := &ThresholdUpdate{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			if p.NewThreshold, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeGuardianApproval(b []byte) (*GuardianApproval, error) {
	d := &decoder{b: b}
	p := &GuardianApproval{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if p.Guardian, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			if p.StatementSig, err = d.bytes(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeMigratePropose(b []byte) (*MigratePropose, error) {
	d := &decoder{b: b}
	p := &MigratePropose{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if p.OldAddress, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			ap, err := DecodeGuardianApproval(v)
			if err != nil {
				return nil, err
			}
			p.Approvals = append(p.Approvals, *ap)
		case f == 3 && wt == 0:
			if p.Nonce, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeMigrateCancel(b []byte) (*MigrateCancel, error) {
	d := &decoder{b: b}
	p := &MigrateCancel{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if p.NewAddress, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 0:
			if p.ProposeSeq, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeMigratePayload(b []byte) (*MigratePayload, error) {
	d := &decoder{b: b}
	p := &MigratePayload{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			sub, err := DecodeMigratePropose(v)
			if err != nil {
				return nil, err
			}
			if p.Propose != nil || p.Cancel != nil {
				return nil, errors.New("wire: несколько action в MigratePayload")
			}
			p.Propose = sub
		case f == 2 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			sub, err := DecodeMigrateCancel(v)
			if err != nil {
				return nil, err
			}
			if p.Propose != nil || p.Cancel != nil {
				return nil, errors.New("wire: несколько action в MigratePayload")
			}
			p.Cancel = sub
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeEvidencePayload(b []byte) (*EvidencePayload, error) {
	d := &decoder{b: b}
	p := &EvidencePayload{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			if p.TxA, err = DecodeTransaction(v); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			if p.TxB, err = DecodeTransaction(v); err != nil {
				return nil, err
			}
		case f == 3 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Kind = EvidenceKind(v)
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeGuardianOfferPayload(b []byte) (*GuardianOfferPayload, error) {
	d := &decoder{b: b}
	if err := d.finish(); err != nil {
		return nil, err
	}
	return &GuardianOfferPayload{}, nil
}

// DecodeTransaction разбирает canonical-кодирование конверта.
func DecodeTransaction(b []byte) (*Transaction, error) {
	d := &decoder{b: b}
	t := &Transaction{}
	var powRaw, payloadRaw []byte
	var payloadField uint32
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if t.NetworkID, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			if t.Wallet, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 3 && wt == 0:
			if t.Seq, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 4 && wt == 2:
			if t.Prev, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 5 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			t.Type = TxType(v)
		case f == 6 && wt == 0:
			if t.TimestampClaim, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 7 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			t.ProtocolVersion = uint32(v)
		case f == 8 && wt == 2:
			if powRaw, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 9 && wt == 2:
			if t.Signature, err = d.bytes(); err != nil {
				return nil, err
			}
		case f >= 10 && f <= 24 && wt == 2:
			if payloadRaw != nil {
				return nil, errors.New("wire: несколько payload-полей")
			}
			if payloadRaw, err = d.bytes(); err != nil {
				return nil, err
			}
			payloadField = f
		default:
			return nil, errUnknownField
		}
	}
	if powRaw != nil {
		p, err := DecodePowProof(powRaw)
		if err != nil {
			return nil, err
		}
		t.Pow = p
	}
	if payloadRaw != nil {
		pl, err := decodePayload(payloadField, payloadRaw)
		if err != nil {
			return nil, err
		}
		t.Payload = pl
	}
	return t, nil
}

func decodePayload(field uint32, b []byte) (Payload, error) {
	switch field {
	case 10:
		return DecodeRegisterPayload(b)
	case 11:
		return DecodeVouchPayload(b)
	case 12:
		return DecodeMintPayload(b)
	case 13:
		return DecodeTransferPayload(b)
	case 14:
		return DecodeMigratePayload(b)
	case 15:
		return DecodeEvidencePayload(b)
	case 16:
		return DecodeGuardianOfferPayload(b)
	default:
		// 17–24: модульные типы (этап 10) — структура ядра не мешает им,
		// но ядро их не декодирует.
		return nil, fmt.Errorf("wire: payload %d вне ядра (модули — этап 10)", field)
	}
}

// PayloadMatchesType: поле type обязано совпадать с веткой payload (C4/E_TYPE).
func PayloadMatchesType(t TxType, p Payload) bool {
	switch p.(type) {
	case *RegisterPayload:
		return t == Register
	case *VouchPayload:
		return t == Vouch
	case *MintPayload:
		return t == Mint
	case *TransferPayload:
		return t == Transfer
	case *MigratePayload:
		return t == Migrate
	case *EvidencePayload:
		return t == Evidence
	case *GuardianOfferPayload:
		return t == GuardianOffer
	default:
		return false
	}
}

// PayloadTxType возвращает тип по ветке payload.
func PayloadTxType(p Payload) TxType {
	switch p.(type) {
	case *RegisterPayload:
		return Register
	case *VouchPayload:
		return Vouch
	case *MintPayload:
		return Mint
	case *TransferPayload:
		return Transfer
	case *MigratePayload:
		return Migrate
	case *EvidencePayload:
		return Evidence
	case *GuardianOfferPayload:
		return GuardianOffer
	default:
		return TxUnspecified
	}
}

// bin.LE helpers для больших чисел (используются PoW-таргетами).
func PutU64BE(b []byte, v uint64) { binary.BigEndian.PutUint64(b, v) }
