// Пакет present — модуль attest (6.1, этап 10): предъявление
// (доказательство человечности) и проверяющая библиотека.
//
// Предъявление = подписанное кошельком утверждение (wire.AttestStatement:
// network_id, адрес, статус, возраст, глубина графа поручительств, метка
// flagged, хеш манифеста, эпоха чекпоинта, challenge проверяющего, срок
// годности, app_id) + Merkle-доказательство состояния кошелька из
// финализированного чекпоинта.
//
// Проверяющему не нужно быть узлом: он проверяет подпись кошелька,
// Merkle-доказательство против корня состояния чекпоинта и пороговую
// подпись самого чекпоинта (лёгкий клиент-синхронизатор — LightSync).
// Вердикт выносит проверяющий (порог возраста, глубина, уровень гарантий):
// протокол вердиктов не выносит (6.1).
package present

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Presentation — предъявление (6.1): утверждение + подпись кошелька +
// открытый ключ кошелька (проверяющий не имеет реестра; адрес проверяется
// деривацией из ключа) + лист состояния и соседи Merkle-доказательства.
type Presentation struct {
	Statement *wire.AttestStatement `json:"statement"`
	PublicKey []byte                `json:"public_key"` // DER/несжатый ключ кошелька
	Signature []byte                `json:"signature"`  // подпись над Statement.Hash()
	Leaf      *wire.StateLeaf       `json:"leaf"`       // лист состояния из чекпоинта
	Siblings  [][]byte              `json:"siblings"`   // соседи по пути (32 байта каждый)
}

// Build — сборка предъявления кошельком (или его приложением) из состояния
// ядра. challenge — вызов проверяющего (защита от повторного использования,
// 6.1), expiresAt — срок годности (net_time), appID — идентификатор
// стороннего приложения (H(address ‖ app_id), 6.1), manifestHash — хеш
// манифеста сети (3.3: предъявление несёт хеш манифеста), epoch — эпоха
// чекпоинта, к которому построено доказательство.
func Build(l *ledger.Ledger, addr []byte, priv *ecdsa.PrivateKey,
	challenge, appID, manifestHash []byte, epoch, expiresAt uint64) (*Presentation, error) {
	w := l.State.Wallet(addr)
	if w == nil {
		return nil, errors.New("present: кошелёк не найден")
	}
	now := l.Clock.NetTime()
	leafHash, leaf, siblings, ok := cons.StateProof(l, addr)
	if !ok {
		return nil, errors.New("present: не удалось построить доказательство")
	}
	_ = leafHash
	stmt := &wire.AttestStatement{
		NetworkID:   append([]byte(nil), l.State.Net.NetworkID...),
		Address:     append([]byte(nil), addr...),
		Status:      w.Status,
		AgeDays:     w.AgeDays(now),
		VouchDepth:  VouchDepth(l, addr),
		Flagged:     w.Flagged,
		ManifestHash: append([]byte(nil), manifestHash...),
		Epoch:       epoch,
		Challenge:   append([]byte(nil), challenge...),
		ExpiresAt:   expiresAt,
		AppID:       append([]byte(nil), appID...),
	}
	h := stmt.Hash()
	sig, err := crypto.SignDER(priv, h[:])
	if err != nil {
		return nil, fmt.Errorf("present: подпись: %w", err)
	}
	sibs := make([][]byte, len(siblings))
	for i, s := range siblings {
		sibs[i] = append([]byte(nil), s[:]...)
	}
	return &Presentation{
		Statement: stmt,
		PublicKey: crypto.MarshalPubKey(&priv.PublicKey),
		Signature: sig,
		Leaf:      leaf,
		Siblings:  sibs,
	}, nil
}

// Encode — транспортное кодирование предъявления (JSON, base64 в bytes).
func (p *Presentation) Encode() ([]byte, error) {
	return json.Marshal(p)
}

// Decode — разбор предъявления.
func Decode(b []byte) (*Presentation, error) {
	var p Presentation
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("present: decode: %w", err)
	}
	if p.Statement == nil || p.Leaf == nil {
		return nil, errors.New("present: неполное предъявление")
	}
	return &p, nil
}

// VouchDepth — расстояние по графу поручительств от посевного круга до
// addr (6.1: глубина графа поручительств в предъявлении). 0 — сам посевной;
// -1 — недостижим. Рёбра: vouch-транзакции (поручитель → кандидат), BFS.
func VouchDepth(l *ledger.Ledger, addr []byte) int64 {
	s := l.State
	children := make(map[string][][]byte)
	for _, tx := range l.AcceptedTxs() {
		if tx.Type != wire.Vouch {
			continue
		}
		vp, ok := tx.Payload.(*wire.VouchPayload)
		if !ok {
			continue
		}
		children[string(tx.Wallet)] = append(children[string(tx.Wallet)], vp.Candidate)
	}
	type qi struct {
		addr  []byte
		depth int64
	}
	var queue []qi
	seen := make(map[string]bool)
	for _, w := range s.Wallets {
		if w.Seed {
			queue = append(queue, qi{w.Address, 0})
			seen[string(w.Address)] = true
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if string(cur.addr) == string(addr) {
			return cur.depth
		}
		for _, next := range children[string(cur.addr)] {
			if !seen[string(next)] {
				seen[string(next)] = true
				queue = append(queue, qi{next, cur.depth + 1})
			}
		}
	}
	return -1
}

// AppIdentity — идентификатор «один человек — одно действие» в стороннем
// приложении (6.1): H(address ‖ app_id); публично связан с адресом
// (тайные голосования в v1 не поддерживаются).
func AppIdentity(addr, appID []byte) [32]byte {
	return sha256.Sum256(append(append([]byte{}, addr...), appID...))
}
