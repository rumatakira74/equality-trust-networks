// Чекпоинты (spec/07): сборка, проверка цепочки и пороговой подписи
// N-из-M (core/threshold). finalized_txid_list — финализированные в эпохе
// txid (правка этапа 5, spec/04 INV-1c).
package cons

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/rumatakira74/trust-networks-platform/core/ledger"
	"github.com/rumatakira74/trust-networks-platform/core/smt"
	"github.com/rumatakira74/trust-networks-platform/core/threshold"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// StateRoot — корень sparse Merkle-дерева состояния (spec/07 7.1.1;
// лист = StateLeaf, D7; строится каждым полным узлом детерминированно).
func StateRoot(l *ledger.Ledger) []byte {
	tree := smt.New()
	for _, w := range l.State.Wallets {
		leaf := stateLeafOf(w)
		var idx [32]byte
		copy(idx[:], w.Address)
		lh := sha256.Sum256(leaf.Encode())
		tree.SetLeaf(idx, lh)
	}
	r := tree.Root()
	return r[:]
}

func stateLeafOf(w *ledger.Wallet) *wire.StateLeaf {
	return &wire.StateLeaf{
		Seq:               w.Seq,
		Balance:           w.TotalBalance(),
		Status:            w.Status,
		Flags:             &wire.WalletFlags{Flagged: w.Flagged, Synthetic: w.Synthetic, Seed: w.Seed},
		RegisteredEpoch:   w.RegisteredEpoch,
		ActivatedEpoch:    w.ActivatedEpoch,
		LastClaimedPeriod: w.LastClaimedPeriod,
		DelayedThreshold:  w.DelayedThreshold,
	}
}

// Build — сборка чекпоинта эпохи (без подписей; подписи собирает Collector).
// prev — предыдущий финализированный чекпоинт (nil — эпоха 1 после genesis).
func Build(networkID []byte, epoch uint64, l *ledger.Ledger, prev *wire.Checkpoint,
	netTime uint64, protocolVersion uint32, participants [][]byte, thresholdN uint64,
	finalizedTxIDs [][]byte) (*wire.Checkpoint, error) {
	cp := &wire.Checkpoint{
		NetworkID:       networkID,
		Epoch:           epoch,
		StateRoot:       StateRoot(l),
		NetworkTime:     netTime,
		ProtocolVersion: protocolVersion,
		ActiveWallets:   countActive(l),
		TotalSupply:     l.State.TotalMinted,
		Participants:    participants,
		Threshold:       thresholdN,
		FinalizedTxIDs:  finalizedTxIDs,
	}
	if prev != nil {
		h := prev.CheckpointHash()
		cp.PrevCheckpointHash = h[:]
		cp.Epoch = prev.Epoch + 1
	} else {
		cp.PrevCheckpointHash = make([]byte, 32)
	}
	return cp, nil
}

func countActive(l *ledger.Ledger) uint64 {
	n := uint64(0)
	for _, w := range l.State.Wallets {
		if w.Status == wire.StatusActive {
			n++
		}
	}
	return n
}

// Verify — проверка чекпоинта по правилам spec/07 7.2.1: цепочка
// (epoch = prev+1, prev_hash), предел сдвига времени, порог подписей
// N-из-M, participants по режиму (проверка режима — вызывающий,
// см. modes.go).
func Verify(cp, prev *wire.Checkpoint, genesisTime, epochSeconds, maxShift uint64,
	pubOf func(addr []byte) (*ecdsa.PublicKey, bool)) error {
	if cp == nil {
		return errors.New("cons: пустой чекпоинт")
	}
	if prev == nil {
		if cp.Epoch != 1 {
			return fmt.Errorf("cons: первый чекпоинт обязан быть эпохой 1, а не %d", cp.Epoch)
		}
		if !bytes.Equal(cp.PrevCheckpointHash, make([]byte, 32)) {
			return errors.New("cons: prev_hash первого чекпоинта не нулевой")
		}
	} else {
		if cp.Epoch != prev.Epoch+1 {
			return fmt.Errorf("cons: epoch %d не следует за %d", cp.Epoch, prev.Epoch)
		}
		h := prev.CheckpointHash()
		if !bytes.Equal(cp.PrevCheckpointHash, h[:]) {
			return errors.New("cons: prev_hash не сходится")
		}
		// 9.2: сетевое время не может сдвигаться между чекпоинтами
		// более чем на заданный предел
		if cp.NetworkTime < prev.NetworkTime+epochSeconds {
			return errors.New("cons: эпоха короче epoch_seconds")
		}
		if cp.NetworkTime > prev.NetworkTime+epochSeconds+maxShift {
			return errors.New("cons: сдвиг сетевого времени сверх предела")
		}
	}
	if cp.NetworkTime < genesisTime+epochSeconds && prev != nil {
		return errors.New("cons: время до genesis+epoch")
	}
	if cp.ThresholdSig == nil {
		return errors.New("cons: нет подписей")
	}
	valid := threshold.Verify(cp.CheckpointHash(), cp.Participants, cp.Threshold,
		cp.ThresholdSig.Signatures, pubOf)
	if uint64(valid) < cp.Threshold {
		return fmt.Errorf("cons: подписей %d < порога %d", valid, cp.Threshold)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Подпись и сбор (тема checkpoint/<network_id> — net/gossip; здесь — логика)
// ---------------------------------------------------------------------------

// Signer — подписант чекпоинтов (кошелёк узла его аппаратным ключом).
type Signer struct {
	Wallet []byte
	Priv   *ecdsa.PrivateKey
}

// Sign — подпись над checkpoint_hash (spec/07 7.2).
func (s *Signer) Sign(cp *wire.Checkpoint) (wire.SignatureShare, error) {
	h := cp.CheckpointHash()
	sig, err := ecdsa.SignASN1(nil, s.Priv, h[:]) //nolint:staticcheck // rand nil = crypto/rand
	if err != nil {
		return wire.SignatureShare{}, err
	}
	return wire.SignatureShare{Signer: s.Wallet, Signature: sig}, nil
}

// Collector — сбор подписей чекпоинта (дедуп по подписанту; подписи
// композициональны, агрегирующий узел — любой участник, spec/07 7.3).
type Collector struct {
	cp     *wire.Checkpoint
	pubOf  func(addr []byte) (*ecdsa.PublicKey, bool)
	shares map[string]wire.SignatureShare
	order  [][]byte
}

// NewCollector — сборщик для конкретного чекпоинта.
func NewCollector(cp *wire.Checkpoint, pubOf func(addr []byte) (*ecdsa.PublicKey, bool)) *Collector {
	return &Collector{cp: cp, pubOf: pubOf, shares: make(map[string]wire.SignatureShare)}
}

// Add — принять долю подписи: подписант из participants, подпись верна.
func (c *Collector) Add(share wire.SignatureShare) bool {
	inSet := false
	for _, p := range c.cp.Participants {
		if string(p) == string(share.Signer) {
			inSet = true
			break
		}
	}
	if !inSet {
		return false
	}
	if _, dup := c.shares[string(share.Signer)]; dup {
		return false
	}
	pub, ok := c.pubOf(share.Signer)
	if !ok || pub == nil {
		return false
	}
	h := c.cp.CheckpointHash()
	if !ecdsa.VerifyASN1(pub, h[:], share.Signature) {
		return false
	}
	c.shares[string(share.Signer)] = share
	c.order = append(c.order, share.Signer)
	return true
}

// Count — число валидных подписей.
func (c *Collector) Count() int { return len(c.shares) }

// Ready — порог достигнут: возвращает полную ThresholdSignature.
func (c *Collector) Ready() *wire.ThresholdSignature {
	if uint64(len(c.shares)) < c.cp.Threshold {
		return nil
	}
	sig := &wire.ThresholdSignature{}
	for _, signer := range c.order {
		sig.Signatures = append(sig.Signatures, c.shares[string(signer)])
	}
	return sig
}
