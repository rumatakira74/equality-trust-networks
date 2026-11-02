package params

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/profiles"
)

// Тест «итерации из таргета» (тема UI-защиты владельца, A4):
// ожидаемая работа усиленного PoW вычисляется ИЗ константы и сверяется
// с решением №9 (этап 6: TargetHard = 2^56 → ~256 итераций Argon2id).
// Аудит 01.11.2026 (A4): фактическое значение было 0x0040_0000_0000_0000
// = 2^54 (1024 итерации) при комментарии «2^56» — константа приведена
// к решению, тест не даёт вернуться.
func TestTargetHardIterationsFromTarget(t *testing.T) {
	const decision9Target = uint64(1) << 56 // решение №9: 2^56
	const expectedIters = uint64(1) << 8    // 2^64 / 2^56 = 256 итераций

	gp := TestGenesis()
	pow := gp.Manifest.Profile.Pow
	if pow == nil {
		t.Fatal("TestGenesis: Pow не задан")
	}
	if pow.TargetHard != decision9Target {
		t.Fatalf("TestGenesis TargetHard = %#x, решение №9 — %#x (2^56)",
			pow.TargetHard, decision9Target)
	}
	if iters := iterationsFromTarget(pow.TargetHard); iters != expectedIters {
		t.Fatalf("итерации из таргета %#x = %d, ожидалось %d (2^64/2^56)",
			pow.TargetHard, iters, expectedIters)
	}

	// базовый профиль шаблонов (profiles.base) — та же константа
	for _, tpl := range profiles.All() {
		p := tpl.Profile.Pow
		if p == nil {
			t.Fatalf("профиль %q: Pow не задан", tpl.Name)
		}
		if p.TargetHard != decision9Target {
			t.Fatalf("профиль %q: TargetHard = %#x, решение №9 — %#x",
				tpl.Name, p.TargetHard, decision9Target)
		}
	}
}

// iterationsFromTarget — ожидаемое число итераций перебора: равномерный
// хеш 2^64, доля проходящих = target/2^64, работа = 2^64/target
// (без переполнения: (2^63/target)·2; таргеты №9 — степени двойки).
func iterationsFromTarget(target uint64) uint64 {
	if target == 0 {
		return 0
	}
	return (uint64(1) << 63) / target * 2
}
