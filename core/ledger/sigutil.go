// Вспомогательные обёртки для инвариантных проверок (без циклы импортов).
package ledger

import (
	"crypto/ecdsa"

	"github.com/rumatakira74/trust-networks-platform/core/crypto"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

func parsePub(b []byte) (*ecdsa.PublicKey, error) { return crypto.ParsePubKey(b) }
func verifySig(pub *ecdsa.PublicKey, hash, sig []byte) bool {
	return crypto.VerifyDER(pub, hash, sig)
}

func txidBytes(tx *wire.Transaction) []byte {
	id := tx.TxID()
	return id[:]
}
