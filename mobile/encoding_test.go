// Тест цепочки строк (тема UI-защиты владельца, п.7 — кодировки
// надписей): строка с кириллицей и спецсимволами проходит путь
// генезиса Go -> wire-байты (UTF-8) -> граница gomobile (форма
// LoadGenesis/ManifestJSON — то, что принимает Kotlin и отдаёт во
// Flutter) байт-в-байт. Явный UTF-8: Go-строки — UTF-8 по контракту,
// декодирование в Kotlin/Dart — строковое (UTF-16), байтовых
// перекодировок в цепочке нет (см. отчёт темы).
package tnp

import (
	"encoding/json"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func TestCyrillicStringChainRoundTrip(t *testing.T) {
	const name = "Сеть доверия №1 — «тест», ünïcode ✓ 日本語"
	gp := params.TestGenesis()
	gp.Manifest.DisplayName = name
	wire.SetManifestHash(gp)

	// Go -> байты -> LoadGenesis (путь genesis.bin приложения)
	gen, err := LoadGenesis(gp.EncodeGenesisPackage())
	if err != nil {
		t.Fatalf("LoadGenesis: %v", err)
	}
	// ManifestJSON — строка, которую мост отдаёт в Kotlin/Flutter
	var m map[string]any
	if err := json.Unmarshal([]byte(gen.ManifestJSON()), &m); err != nil {
		t.Fatalf("ManifestJSON не парсится: %v", err)
	}
	if got, _ := m["display_name"].(string); got != name {
		t.Fatalf("display_name после цепочки: %q, ожидалось %q", got, name)
	}

	// повторный Encode -> Load (передача файла сети) — строка стабильна
	gen2, err := LoadGenesis(gen.Encode())
	if err != nil {
		t.Fatalf("повторный LoadGenesis: %v", err)
	}
	var m2 map[string]any
	if err := json.Unmarshal([]byte(gen2.ManifestJSON()), &m2); err != nil {
		t.Fatalf("ManifestJSON (2) не парсится: %v", err)
	}
	if got, _ := m2["display_name"].(string); got != name {
		t.Fatalf("display_name после повторной цепочки: %q, ожидалось %q", got, name)
	}
}
