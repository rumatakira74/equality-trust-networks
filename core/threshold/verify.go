package threshold

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
)

// verifyDER — через общий строгий верификатор (каноничный DER + low-s,
// аудит криптографии этапа 9): одна семантика подписи во всём ядре.
func verifyDER(pub *ecdsa.PublicKey, hash, sig []byte) bool {
	return crypto.VerifyDER(pub, hash, sig)
}

func ecdsaP256() elliptic.Curve { return elliptic.P256() }

func sha256Sum(b []byte) [32]byte { return sha256.Sum256(b) }
