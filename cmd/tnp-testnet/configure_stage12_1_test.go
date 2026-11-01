// Тесты мини-этапа 12.1 (решения владельца 01.11.2026): applicationId
// io.equality.<sanitized-slug>.<hash8>, палитра Material 16 тонов,
// окно свежести шаблона 3 = 0 (расхождение №5 отчёта этапа 12).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestApplicationIDSlugHash — applicationId = io.equality.<sanitized-slug>.
// <hash8>: дефисы slug → «_» (Android запрещает их в пакете), <hash8> —
// первые 8 hex SHA-256(network_id), тот же хеш, что в label Eq-<hash8>;
// глобальная сеть — io.equality.global.<hash8>. Уникальность — на хеше:
// равные slug разных организаторов не коллизируют (network_id уникален).
func TestApplicationIDSlugHash(t *testing.T) {
	nid := sha256.Sum256([]byte("manifest-genesis")) // детерминированный network_id
	h := sha256.Sum256(nid[:])                        // hash8 = SHA-256(network_id)
	hash8 := hex.EncodeToString(h[:4])
	if got, want := ApplicationID(nid[:], "monte-cloud"), "io.equality.monte_cloud."+hash8; got != want {
		t.Fatalf("application_id: %q (want %q)", got, want)
	}
	if got, want := ApplicationID(nid[:], "abc"), "io.equality.abc."+hash8; got != want {
		t.Fatalf("application_id без дефисов: %q (want %q)", got, want)
	}
	// глобальная сеть: hash8 от константы network_id = "global"
	gh := sha256.Sum256(wire.GlobalNetworkID)
	if got, want := ApplicationID(wire.GlobalNetworkID, "global"),
		"io.equality.global."+hex.EncodeToString(gh[:4]); got != want {
		t.Fatalf("application_id глобальной сети: %q (want %q)", got, want)
	}
}

// TestPaletteMaterial — палитра Equality: 16 тонов Material 500–600,
// hex валидны и уникальны; индекс детерминирован — одна сеть один
// цвет всегда (SHA-256(network_id) mod 16, механика не менялась).
func TestPaletteMaterial(t *testing.T) {
	if len(equalityPalette) != 16 {
		t.Fatalf("палитра: %d тонов (нужно 16)", len(equalityPalette))
	}
	seen := map[string]bool{}
	for i, tone := range equalityPalette {
		if seen[tone.Hex] {
			t.Fatalf("тон %d: hex %s повторяется", i, tone.Hex)
		}
		seen[tone.Hex] = true
		if _, err := parseHexColor(tone.Hex); err != nil {
			t.Fatalf("тон %d (%s): %v", i, tone.Hex, err)
		}
	}
	for _, nid := range [][]byte{wire.GlobalNetworkID, []byte("net-a"), []byte("net-b")} {
		h := sha256.Sum256(nid)
		want := int(h[0]) % len(equalityPalette)
		s1 := iconSpec(nid, false, false)
		s2 := iconSpec(nid, true, true)
		if s1.PaletteIndex != want || s2.PaletteIndex != want {
			t.Fatalf("индекс палитры %x: %d/%d (want %d)", nid, s1.PaletteIndex, s2.PaletteIndex, want)
		}
		if s1.Background != equalityPalette[want].Hex || s2.Background != equalityPalette[want].Hex {
			t.Fatalf("цвет фона %x: %s/%s (want %s)", nid, s1.Background, s2.Background, equalityPalette[want].Hex)
		}
	}
}

// TestConfigureTemplate3FreshWindow — шаблон 3 (quota): окно свежести 0
// (решение владельца 01.11.2026): окно 3 из base длиннее жизни квоты при
// quota_reset — единица сгорала бы на границе периода, не созрев.
func TestConfigureTemplate3FreshWindow(t *testing.T) {
	out := filepath.Join(t.TempDir(), "net")
	if err := configureCmd([]string{"-template", "3", "-slug", "monte-cloud", "-out", out}); err != nil {
		t.Fatalf("configure: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "genesis.bin"))
	if err != nil {
		t.Fatal(err)
	}
	gp, err := wire.DecodeGenesisPackage(b)
	if err != nil {
		t.Fatal(err)
	}
	if gp.Manifest.Profile.FreshTransferWindowPeriods != 0 {
		t.Fatalf("окно свежести шаблона 3: %d (нужно 0)",
			gp.Manifest.Profile.FreshTransferWindowPeriods)
	}
}
