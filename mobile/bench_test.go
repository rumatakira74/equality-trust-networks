// Бенчмарки моста (12.1, вопрос №9): время лёгкого PoW (register/
// transfer) и усиленного Argon2id (выпуск) — ориентир для финализации
// таргетов «под средний телефон». Замеры: go test ./mobile/ -bench .
package tnp

import (
	"testing"
)

// BenchmarkSolveLight24 — лёгкий PoW, таргет ~2^24 попыток (0x0000010000000000).
func BenchmarkSolveLight24(b *testing.B) {
	tx := NewTransfer([]byte("bench"), []byte("wallet"), 1, make([]byte, 32),
		[]byte("to"), 100, 1, 1234567)
	for i := 0; i < b.N; i++ {
		tx.SolveLight(0x0000_0100_0000_0000)
	}
}

// BenchmarkSolveLight20 — лёгкий PoW, таргет ~2^20 попыток.
func BenchmarkSolveLight20(b *testing.B) {
	tx := NewTransfer([]byte("bench"), []byte("wallet"), 1, make([]byte, 32),
		[]byte("to"), 100, 1, 1234567)
	for i := 0; i < b.N; i++ {
		tx.SolveLight(0x0000_1000_0000_0000)
	}
}

// BenchmarkMintPow32MiB — полный поиск усиленного PoW выпуска по
// ПРЕДЛАГАЕМОМУ таргету №9 (2^57 ≈ 128 итераций Argon2id 32 МиБ × 1);
// свежий preimage на итерацию (иначе поиск повторяет тот же nonce).
func BenchmarkMintPow32MiB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tx := NewMint([]byte("bench"), []byte("wallet"), 1, make([]byte, 32),
			1, 1000, 1, int64(1234567+i))
		tx.SolveHard(0x0080_0000_0000_0000, 32768, 1, 1)
	}
}

// BenchmarkMintPow64MiB — то же с 64 МиБ (вариант №9).
func BenchmarkMintPow64MiB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tx := NewMint([]byte("bench"), []byte("wallet"), 1, make([]byte, 32),
			1, 1000, 1, int64(1234567+i))
		tx.SolveHard(0x0080_0000_0000_0000, 65536, 1, 1)
	}
}

// BenchmarkBridgeCall — накладные расходы вызова моста: сборка +
// кодирование + разбор (без PoW).
func BenchmarkBridgeCall(b *testing.B) {
	netID := []byte("bench")
	addr := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		tx := NewTransfer(netID, addr, 1, make([]byte, 32), addr, 100, 1, 1234567)
		enc := tx.Encode()
		if _, err := DecodeTx(enc); err != nil {
			b.Fatal(err)
		}
	}
}
