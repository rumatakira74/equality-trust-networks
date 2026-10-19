package attest

import "crypto/sha256"

func newHasher() *sha256Digest { return &sha256Digest{} }

type sha256Digest struct{ inner []byte }

func (d *sha256Digest) Write(p []byte) { d.inner = append(d.inner, p...) }
func (d *sha256Digest) Sum(b []byte) []byte {
	s := sha256.Sum256(d.inner)
	return append(b, s[:]...)
}
