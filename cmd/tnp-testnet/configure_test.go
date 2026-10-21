package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestConfigureMigrateDelayDays — флаг -migrate-delay-days выставляет
// migrate_delay_days (genesis-поле 49) в genesis-пакете (находка №6
// прогона stage11-prep: поле не выставлялось конфигуратором и правилось
// пост-обработкой хелпером). Без флага — значение шаблона.
func TestConfigureMigrateDelayDays(t *testing.T) {
	run := func(t *testing.T, extra ...string) *wire.GenesisPackage {
		t.Helper()
		out := filepath.Join(t.TempDir(), "net")
		args := append([]string{
			"-template", "laboratory",
			"-network-id", "migrate-delay-test",
			"-out", out,
		}, extra...)
		if err := configureCmd(args); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(out, "genesis.bin"))
		if err != nil {
			t.Fatal(err)
		}
		gp, err := wire.DecodeGenesisPackage(b)
		if err != nil {
			t.Fatal(err)
		}
		return gp
	}

	t.Run("флаг выставляет поле 49", func(t *testing.T) {
		gp := run(t, "-migrate-delay-days", "1")
		if got := gp.Manifest.Profile.MigrateDelayDays; got != 1 {
			t.Fatalf("migrate_delay_days = %d, хочу 1", got)
		}
	})

	t.Run("без флага — значение шаблона", func(t *testing.T) {
		gp := run(t)
		if got := gp.Manifest.Profile.MigrateDelayDays; got != 14 {
			t.Fatalf("migrate_delay_days = %d, хочу 14 (шаблон)", got)
		}
	})
}
