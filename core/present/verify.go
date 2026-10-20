// Проверяющая библиотека (6.1): проверка предъявления без полного узла.
// Проверяются: подпись кошелька над утверждением, привязка ключа к адресу,
// срок годности и challenge, соответствие утверждения листу состояния,
// Merkle-доказательство против корня состояния финализированного
// чекпоинта и пороговая подпись чекпоинта. Вердикт («что считать
// достаточным») выносит проверяющий — протокол вердиктов не выносит.
package present

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/smt"
	"github.com/rumatakira74/equality-trust-networks/core/threshold"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// PubSource — источник открытых ключей подписантов чекпоинтов для
// пороговой проверки (без полного узла): генерируется из манифеста
// (посевной круг) и/или доверенного снимка состава валидаторов.
type PubSource func(addr []byte) (*ecdsa.PublicKey, bool)

// VerifyResult — итог проверки предъявления: сам факт проверки и
// производные данные для решения проверяющего (H(address ‖ app_id),
// статус, возраст, глубина, уровень гарантий сети — из манифеста).
type VerifyResult struct {
	AppIdentity [32]byte // H(address ‖ app_id), 6.1
	Statement   *wire.AttestStatement
	Leaf        *wire.StateLeaf
}

// Verify — проверка предъявления (6.1). cp — финализированный чекпоинт
// эпохи statement.Epoch (лёгкий клиент гарантирует цепочку и подписи;
// здесь подпись чекпоинта перепроверяется). manifestHash — ожидаемый хеш
// манифеста сети, networkID — ожидаемая сеть, challenge — вызов, выданный
// проверяющим этому предъявлению, now — текущее время (срок годности).
func Verify(p *Presentation, cp *wire.Checkpoint, networkID, manifestHash, challenge []byte,
	now uint64, pubOf PubSource) (*VerifyResult, error) {
	if p == nil || p.Statement == nil || p.Leaf == nil {
		return nil, errors.New("present: пустое предъявление")
	}
	st := p.Statement
	if !bytes.Equal(st.NetworkID, networkID) {
		return nil, errors.New("present: чужая сеть")
	}
	if !bytes.Equal(st.ManifestHash, manifestHash) {
		return nil, errors.New("present: хеш манифеста не сходится (3.3)")
	}
	if len(st.Challenge) == 0 || !bytes.Equal(st.Challenge, challenge) {
		return nil, errors.New("present: challenge не совпадает (повторное использование, 6.1)")
	}
	if now >= st.ExpiresAt {
		return nil, errors.New("present: срок годности истёк")
	}
	// ключ кошелька и привязка к адресу (проверяющий не имеет реестра)
	pub, err := crypto.ParsePubKey(p.PublicKey)
	if err != nil {
		return nil, errors.New("present: ключ кошелька не разбирается")
	}
	fp := crypto.Fingerprint(p.PublicKey)
	addr := crypto.DeriveAddress(fp, networkID)
	if !bytes.Equal(addr, st.Address) {
		return nil, errors.New("present: адрес не соответствует ключу")
	}
	h := st.Hash()
	if !crypto.VerifyDER(pub, h[:], p.Signature) {
		return nil, errors.New("present: подпись кошелька неверна")
	}
	// чекпоинт: сеть, эпоха, пороговая подпись
	if cp == nil {
		return nil, errors.New("present: нет чекпоинта")
	}
	if !bytes.Equal(cp.NetworkID, networkID) {
		return nil, errors.New("present: чекпоинт чужой сети")
	}
	if cp.Epoch != st.Epoch {
		return nil, errors.New("present: эпоха чекпоинта не совпадает с утверждением")
	}
	if cp.ThresholdSig == nil {
		return nil, errors.New("present: чекпоинт без пороговой подписи")
	}
	valid := threshold.Verify(cp.CheckpointHash(), cp.Participants, cp.Threshold,
		cp.ThresholdSig.Signatures, pubOf)
	if uint64(valid) < cp.Threshold {
		return nil, errors.New("present: пороговая подпись чекпоинта не достигнута")
	}
	// Merkle-доказательство состояния из чекпоинта (6.1)
	var idx [32]byte
	copy(idx[:], st.Address)
	lh := stLeafHash(p.Leaf)
	sibs := make([][32]byte, len(p.Siblings))
	for i, s := range p.Siblings {
		if len(s) != 32 {
			return nil, errors.New("present: сосед доказательства не 32 байта")
		}
		copy(sibs[i][:], s)
	}
	if !smt.VerifyProof(idx, lh, sibs, asRoot(cp.StateRoot)) {
		return nil, errors.New("present: Merkle-доказательство не сходится с корнем чекпоинта")
	}
	// соответствие утверждения листу состояния
	if p.Leaf.Status != st.Status {
		return nil, errors.New("present: статус в утверждении не совпадает с листом")
	}
	if p.Leaf.Flags != nil && p.Leaf.Flags.Flagged != st.Flagged {
		return nil, errors.New("present: метка flagged не совпадает с листом")
	}
	return &VerifyResult{
		AppIdentity: AppIdentity(st.Address, st.AppID),
		Statement:   st,
		Leaf:        p.Leaf,
	}, nil
}

// VerifyAgeBound — производный возраст из листа (заявленный возраст не
// может превышать время от зарегистрированной эпохи до эпохи чекпоинта):
// проверяющий может сверить заявление AgeDays с этой оценкой.
func VerifyAgeBound(st *wire.AttestStatement, leaf *wire.StateLeaf, epochSeconds uint64) uint64 {
	if st.Epoch <= leaf.RegisteredEpoch {
		return 0
	}
	return (st.Epoch - leaf.RegisteredEpoch) * epochSeconds / 86400
}

func stLeafHash(l *wire.StateLeaf) [32]byte {
	return sha256.Sum256(l.Encode())
}

func asRoot(b []byte) [32]byte {
	var r [32]byte
	copy(r[:], b)
	return r
}
