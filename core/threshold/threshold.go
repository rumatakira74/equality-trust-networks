// Пороговая подпись чекпоинтов: N-из-M набор подписей ECDSA P-256
// (закрытое решение этапа 0, spec/07 7.2). Сетевой сбор подписей — этап 5.
package threshold

import (
	"crypto/ecdsa"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Verify проверяет набор: >= threshold валидных подписей от подписантов из
// participants (дедупликация подписантов; каждый подписант учитывается один раз).
func Verify(checkpointHash [32]byte, participants [][]byte, threshold uint64,
	sigs []wire.SignatureShare, pubOf func(addr []byte) (*ecdsa.PublicKey, bool)) int {

	inSet := make(map[string]bool, len(participants))
	for _, p := range participants {
		inSet[string(p)] = true
	}
	counted := make(map[string]bool)
	valid := 0
	for i := range sigs {
		key := string(sigs[i].Signer)
		if !inSet[key] || counted[key] {
			continue
		}
		pub, ok := pubOf(sigs[i].Signer)
		if !ok || pub == nil {
			continue
		}
		if verifyDER(pub, checkpointHash[:], sigs[i].Signature) {
			counted[key] = true
			valid++
		}
	}
	return valid
}

// Meets: valid >= threshold.
func Meets(valid, threshold uint64) bool { return uint64(valid) >= threshold }
