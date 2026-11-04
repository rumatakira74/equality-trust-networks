// Тесты консенсус-правила C1 (тема fix-cp-threshold): собственная
// непротиворечивость порога/состава в Verify и сверка состава с режимом
// манифеста (CheckParticipants — вызывающий, поле 62 cp_set_check).
package cons

import (
	"bytes"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestVerifyThresholdSanity — порог больше числа участников и пустой
// состав отвергаются Verify: легального чекпоинта с такими значениями не
// бывает ни в одном режиме (N ≤ M, modes.go).
func TestVerifyThresholdSanity(t *testing.T) {
	const genesisTime, epochSeconds, maxShift = uint64(1_700_000_000), uint64(86400), uint64(3600)
	base := func() *wire.Checkpoint {
		return &wire.Checkpoint{
			NetworkID:          bytes.Repeat([]byte{1}, 32),
			Epoch:              1,
			StateRoot:          bytes.Repeat([]byte{0xAA}, 32),
			PrevCheckpointHash: make([]byte, 32),
			NetworkTime:        genesisTime + epochSeconds,
			Participants:       [][]byte{bytes.Repeat([]byte{2}, 32), bytes.Repeat([]byte{3}, 32)},
			Threshold:          2,
			ThresholdSig:       &wire.ThresholdSignature{},
		}
	}
	// контроль: порог 2 из 2 участников, но подписей нет — отказ по подписям
	cp := base()
	if err := Verify(cp, nil, genesisTime, epochSeconds, maxShift, noKeys); err == nil {
		t.Fatal("чекпоинт без подписей принят")
	}
	// порог больше состава
	cp = base()
	cp.Threshold = 3
	if err := Verify(cp, nil, genesisTime, epochSeconds, maxShift, noKeys); err == nil {
		t.Fatal("порог 3 при 2 участниках принят")
	}
	// пустой состав
	cp = base()
	cp.Participants = nil
	if err := Verify(cp, nil, genesisTime, epochSeconds, maxShift, noKeys); err == nil {
		t.Fatal("чекпоинт с пустым составом принят")
	}
}

// TestCheckParticipants — сверка состава/порога с вычисленными по режиму:
// совпадение проходит, расхождение порога, числа участников, адреса или
// порядка отвергается (порядок значим — очередь строителей индексирует
// состав, №42 (A)).
func TestCheckParticipants(t *testing.T) {
	a := bytes.Repeat([]byte{0xA1}, 32)
	b := bytes.Repeat([]byte{0xB2}, 32)
	want := [][]byte{a, b}
	cp := &wire.Checkpoint{Participants: [][]byte{a, b}, Threshold: 2}
	if err := CheckParticipants(cp, want, 2); err != nil {
		t.Fatalf("совпадающий состав отвергнут: %v", err)
	}
	if err := CheckParticipants(&wire.Checkpoint{Participants: [][]byte{a, b}, Threshold: 1}, want, 2); err == nil {
		t.Fatal("расхождение порога не отвергнуто")
	}
	if err := CheckParticipants(&wire.Checkpoint{Participants: [][]byte{a}, Threshold: 2}, want, 2); err == nil {
		t.Fatal("расхождение числа участников не отвергнуто")
	}
	if err := CheckParticipants(&wire.Checkpoint{Participants: [][]byte{a, a}, Threshold: 2}, want, 2); err == nil {
		t.Fatal("подмена участника не отвергнута")
	}
	if err := CheckParticipants(&wire.Checkpoint{Participants: [][]byte{b, a}, Threshold: 2}, want, 2); err == nil {
		t.Fatal("перестановка состава не отвергнута (порядок значим — очередь строителей)")
	}
	if err := CheckParticipants(nil, want, 2); err == nil {
		t.Fatal("пустой чекпоинт не отвергнут")
	}
}
