// Сборка PoW для транзакций (7.6, 9.3): используется cmd/tnp-node и
// прикладным кодом; ядро только проверяет (core/crypto).
package net

import (
	"encoding/binary"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// SolveTxPoW — подбирает nonce и заполняет tx.Pow (усиленный Argon2id для
// register/mint, лёгкий SHA-256 для остальных; spec/01 1.3).
func SolveTxPoW(tx *wire.Transaction, pow wire.PowParams, hard bool) {
	if hard {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowArgon2id}
		ap := crypto.Argon2Params{MemoryKiB: pow.Argon2MemoryKiB,
			Iterations: pow.Argon2Iterations, Parallelism: pow.Argon2Parallelism}
		for n := uint64(0); ; n++ {
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], n)
			tx.Pow.Nonce = nb[:]
			if crypto.VerifyHard(tx.SigningPreimage(), nb[:], pow.TargetHard, ap) {
				return
			}
		}
	}
	tx.Pow = &wire.PowProof{Algorithm: wire.PowLight}
	for n := uint64(0); ; n++ {
		var nb [8]byte
		binary.BigEndian.PutUint64(nb[:], n)
		tx.Pow.Nonce = nb[:]
		if crypto.VerifyLight(tx.SigningPreimage(), nb[:], pow.TargetLight) {
			return
		}
	}
}
