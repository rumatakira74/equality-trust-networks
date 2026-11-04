// Тесты находки C1 аудита Sonnet-2 (reports/audit-sonnet/go-tests/
// cons_forged_checkpoint_test.go.txt, ветка audit-sonnet): чекпоинт без
// единой подписи (порог 0) не должен считаться финализированным.
//
// До фикса (тема fix-cp-threshold) тесты были КРАСНЫЕ под тегом
// `cpforgefix`; после фикса тег снят — тесты входят в обычный прогон
// `go test ./cons/`. Ожидания аудита не менялись, адаптировались только
// комментарии (сигнатура Verify сохранена).
package cons

import (
	"bytes"
	"crypto/ecdsa"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func noKeys(addr []byte) (*ecdsa.PublicKey, bool) { return nil, false }

// Verify принимает чекпоинт с Threshold = 0 и пустой ThresholdSignature:
// valid(0) < 0 ложно. state_root произвольный.
func TestForgedCheckpoint_ZeroThresholdRejectedByVerify(t *testing.T) {
	const genesisTime, epochSeconds, maxShift = uint64(1_700_000_000), uint64(86400), uint64(3600)
	cp := &wire.Checkpoint{
		NetworkID:          bytes.Repeat([]byte{1}, 32),
		Epoch:              1,
		StateRoot:          bytes.Repeat([]byte{0xAA}, 32), // не связан ни с каким состоянием
		PrevCheckpointHash: make([]byte, 32),
		NetworkTime:        genesisTime + epochSeconds,
		Participants:       nil,
		Threshold:          0,
		ThresholdSig:       &wire.ThresholdSignature{},
	}
	if err := Verify(cp, nil, genesisTime, epochSeconds, maxShift, noKeys); err == nil {
		t.Fatal("чекпоинт без подписей и с порогом 0 принят Verify")
	}
}

// Collector.Ready отдаёт непустую ThresholdSignature при Threshold = 0 без
// единой доли — net.collectCpPending сочтёт такое предложение готовым.
func TestForgedCheckpoint_ZeroThresholdCollectorNotReady(t *testing.T) {
	cp := &wire.Checkpoint{Participants: [][]byte{{1}}, Threshold: 0}
	c := NewCollector(cp, noKeys)
	if c.Ready() != nil {
		t.Fatal("порог 0 без единой подписи считается достигнутым")
	}
}
