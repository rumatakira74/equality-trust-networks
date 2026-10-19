// PoW: лёгкий (SHA-256) и усиленный memory-hard (Argon2id) — 7.6, 9.3, spec/01 1.3.
//
// Таргеты — предположение (вопрос №9, значения «под средний телефон»
// выбираются на этапах 1.2/2); в тестах используются малые таргеты.
package crypto

import (
	"crypto/sha256"
	"encoding/binary"

	"golang.org/x/crypto/argon2"
)

// HashBelowTarget: хеш (как big-endian uint64 из первых 8 байт) меньше таргета.
// target = 0 означает «выключено» (принять всё); MaxUint64 — максимальная строгость.
func HashBelowTarget(h []byte, target uint64) bool {
	if target == 0 {
		return true
	}
	return binary.BigEndian.Uint64(h[:8]) < target
}

// SolveLight ищет nonce: SHA-256(preimage || nonce) < target.
func SolveLight(preimage []byte, target uint64) []byte {
	if target == 0 {
		return []byte{0}
	}
	for n := uint64(0); ; n++ {
		var nb [8]byte
		binary.BigEndian.PutUint64(nb[:], n)
		h := sha256.New()
		h.Write(preimage)
		h.Write(nb[:])
		if HashBelowTarget(h.Sum(nil), target) {
			return nb[:]
		}
	}
}

// VerifyLight проверяет лёгкий PoW (spec/01 1.3).
func VerifyLight(preimage []byte, nonce []byte, target uint64) bool {
	if target == 0 {
		return true
	}
	h := sha256.New()
	h.Write(preimage)
	h.Write(nonce)
	return HashBelowTarget(h.Sum(nil), target)
}

// Argon2Params — параметры Argon2id (вопрос №9, предположение).
type Argon2Params struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint32
}

// SolveHard ищет nonce: Argon2id(password=nonce, salt=preimage) < target.
func SolveHard(preimage []byte, target uint64, p Argon2Params) []byte {
	if target == 0 {
		return []byte{0}
	}
	for n := uint64(0); ; n++ {
		var nb [8]byte
		binary.BigEndian.PutUint64(nb[:], n)
		out := argon2.IDKey(nb[:], preimage, p.Iterations, p.MemoryKiB, uint8(p.Parallelism), 32)
		if HashBelowTarget(out, target) {
			return nb[:]
		}
	}
}

// VerifyHard проверяет усиленный PoW (spec/01 1.3).
func VerifyHard(preimage, nonce []byte, target uint64, p Argon2Params) bool {
	if target == 0 {
		return true
	}
	out := argon2.IDKey(nonce, preimage, p.Iterations, p.MemoryKiB, uint8(p.Parallelism), 32)
	return HashBelowTarget(out, target)
}
