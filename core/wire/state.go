// Состояние и чекпоинты: proto/checkpoint.proto (spec/07).
package wire

import "crypto/sha256"

// StateLeaf — лист sparse Merkle-дерева, 9 полей (решение D7, spec/07 7.1.1).
type StateLeaf struct {
	Seq                  uint64
	Balance              uint64
	Status               WalletStatus
	Flags                *WalletFlags
	RegisteredEpoch      uint64
	ActivatedEpoch       uint64
	LastClaimedPeriod    uint64
	DelayedThreshold     uint64
	ThresholdChangeEpoch uint64
}

func (l *StateLeaf) Encode() []byte {
	var b []byte
	b = appendU64Field(b, 1, l.Seq)
	b = appendU64Field(b, 2, l.Balance)
	b = appendU32Field(b, 3, uint32(l.Status))
	if l.Flags != nil {
		b = appendMessageField(b, 4, l.Flags.Encode())
	}
	b = appendU64Field(b, 5, l.RegisteredEpoch)
	b = appendU64Field(b, 6, l.ActivatedEpoch)
	b = appendU64Field(b, 7, l.LastClaimedPeriod)
	b = appendU64Field(b, 8, l.DelayedThreshold)
	b = appendU64Field(b, 9, l.ThresholdChangeEpoch)
	return b
}

func DecodeStateLeaf(b []byte) (*StateLeaf, error) {
	d := &decoder{b: b}
	p := &StateLeaf{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			if p.Seq, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 0:
			if p.Balance, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 3 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Status = WalletStatus(v)
		case f == 4 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			fl, err := DecodeWalletFlags(v)
			if err != nil {
				return nil, err
			}
			p.Flags = fl
		case f == 5 && wt == 0:
			if p.RegisteredEpoch, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 6 && wt == 0:
			if p.ActivatedEpoch, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 7 && wt == 0:
			if p.LastClaimedPeriod, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 8 && wt == 0:
			if p.DelayedThreshold, err = d.varint(); err != nil {
				return nil, err
			}
		case f == 9 && wt == 0:
			if p.ThresholdChangeEpoch, err = d.varint(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

func DecodeWalletFlags(b []byte) (*WalletFlags, error) {
	d := &decoder{b: b}
	p := &WalletFlags{}
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
			p.Flagged = v != 0
		case f == 2 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Synthetic = v != 0
		case f == 3 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Seed = v != 0
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

// SignatureShare — подпись одного подписанта из N-из-M набора (spec/07 7.2).
type SignatureShare struct {
	Signer    []byte
	Signature []byte
}

func (p *SignatureShare) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, p.Signer)
	b = appendBytesField(b, 2, p.Signature)
	return b
}

// ThresholdSignature — N-из-M набор подписей ECDSA P-256 (решение этапа 0).
type ThresholdSignature struct {
	Signatures []SignatureShare
}

func (p *ThresholdSignature) Encode() []byte {
	var b []byte
	for i := range p.Signatures {
		b = appendMessageField(b, 1, p.Signatures[i].Encode())
	}
	return b
}

// Checkpoint — чекпоинт эпохи (spec/07 7.1).
type Checkpoint struct {
	NetworkID          []byte
	Epoch              uint64
	StateRoot          []byte
	PrevCheckpointHash []byte
	NetworkTime        uint64
	ProtocolVersion    uint32
	ActiveWallets      uint64
	TotalSupply        uint64
	Participants       [][]byte
	Threshold          uint64
	ThresholdSig       *ThresholdSignature
	// FinalizedTxIDs — txid, финализированные в эпохе (поле 12; этап 5,
	// spec/04 INV-1c): выбранные ветки конфликтов — по одной на
	// (wallet, seq); альтернативные ветки остаются в журнале хранения.
	FinalizedTxIDs [][]byte
}

// CheckpointHash = SHA-256(canonical(Checkpoint без threshold_sig)) (spec/07 7.1).
func (c *Checkpoint) CheckpointHash() [32]byte {
	cp := *c
	cp.ThresholdSig = nil
	return sha256.Sum256(cp.Encode())
}

func (c *Checkpoint) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, c.NetworkID)
	b = appendU64Field(b, 2, c.Epoch)
	b = appendBytesField(b, 3, c.StateRoot)
	b = appendBytesField(b, 4, c.PrevCheckpointHash)
	b = appendU64Field(b, 5, c.NetworkTime)
	b = appendU32Field(b, 6, c.ProtocolVersion)
	b = appendU64Field(b, 7, c.ActiveWallets)
	b = appendU64Field(b, 8, c.TotalSupply)
	for _, p := range c.Participants {
		b = appendBytesField(b, 9, p)
	}
	b = appendU64Field(b, 10, c.Threshold)
	if c.ThresholdSig != nil {
		b = appendMessageField(b, 11, c.ThresholdSig.Encode())
	}
	for _, id := range c.FinalizedTxIDs {
		b = appendBytesField(b, 12, id)
	}
	return b
}

// MerkleProof — доказательство листа из чекпоинта (spec/07, для лёгких узлов).
type MerkleProof struct {
	Leaf     []byte
	Siblings [][]byte
	Root     []byte
}

// MigrateStatement — каноническое утверждение, подписываемое гардианом
// (proto/transactions.proto; не транзакция).
type MigrateStatement struct {
	NetworkID  []byte
	OldAddress []byte
	NewAddress []byte
	Nonce      uint64
	Epoch      uint64
}

func (m *MigrateStatement) Encode() []byte {
	var b []byte
	b = appendBytesField(b, 1, m.NetworkID)
	b = appendBytesField(b, 2, m.OldAddress)
	b = appendBytesField(b, 3, m.NewAddress)
	b = appendU64Field(b, 4, m.Nonce)
	b = appendU64Field(b, 5, m.Epoch)
	return b
}

// StatementHash — хеш, подписываемый гардианом (spec/01 1.5.5).
func (m *MigrateStatement) Hash() [32]byte {
	return sha256.Sum256(m.Encode())
}
