package threshold

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
)

func verifyDER(pub *ecdsa.PublicKey, hash, sig []byte) bool {
	return ecdsa.VerifyASN1(pub, hash, sig)
}

func ecdsaP256() elliptic.Curve { return elliptic.P256() }

func sha256Sum(b []byte) [32]byte { return sha256.Sum256(b) }
