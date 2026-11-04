// Тесты конфигуратора этапа 12: шаблон профиля 8, деривация network_id,
// имена APK (std/rec), applicationId, label Eq-<hash8>, глобальная сеть
// (recurring=on отвергается при сборке, константа "global").
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// configureRun — сборка сети configure с шаблоном и оверрайдами;
// возвращает genesis-пакет и android.json. Signing-каталог — временный
// (тесты герметичны: боевых ключей и паролей в тестах нет).
func configureRun(t *testing.T, extra ...string) (*wire.GenesisPackage, map[string]any, string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "net")
	args := append([]string{"-template", "8", "-out", out,
		"-signing-dir", filepath.Join(t.TempDir(), "signing")}, extra...)
	err := configureCmd(args)
	if err != nil {
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
	ajb, err := os.ReadFile(filepath.Join(out, "android.json"))
	if err != nil {
		t.Fatal(err)
	}
	var aj map[string]any
	if err := json.Unmarshal(ajb, &aj); err != nil {
		t.Fatal(err)
	}
	return gp, aj, out
}

// TestConfigureProfile8Rec — шаблон 8: recurring=on, peer-redeem,
// network_id деривирован и само-сертифицирован, APK-имя с суффиксом rec,
// applicationId io.equality.<slug>, label Eq-<hash8>, иконка с бейджем.
func TestConfigureProfile8Rec(t *testing.T) {
	gp, aj, out := configureRun(t, "-slug", "monte-cloud", "-display-name", "Monte Cloud")
	prof := gp.Manifest.Profile
	if !prof.Recurring || prof.TransferMode != wire.TransferModePeerRedeem ||
		prof.Unit != wire.UnitQuota || prof.IssuanceMode != wire.IssuanceQuotaReset {
		t.Fatalf("профиль 8: recurring=%v mode=%d unit=%d issuance=%d",
			prof.Recurring, prof.TransferMode, prof.Unit, prof.IssuanceMode)
	}
	if gp.Manifest.Slug != "monte-cloud" || gp.Manifest.DisplayName != "Monte Cloud" {
		t.Fatalf("slug/display_name: %q / %q", gp.Manifest.Slug, gp.Manifest.DisplayName)
	}
	if err := wire.VerifySelfCertified(gp); err != nil {
		t.Fatalf("network_id не само-сертифицирован: %v", err)
	}
	if aj["apk_name"] != "equality-monte-cloud-rec-v1.0.0.apk" {
		t.Fatalf("apk_name: %v", aj["apk_name"])
	}
	wantLabel := wire.EqLabel(gp.Genesis.NetworkID)
	if aj["label"] != wantLabel || !strings.HasPrefix(wantLabel, "Eq-") || len(wantLabel) != 11 {
		t.Fatalf("label: %v (want %s)", aj["label"], wantLabel)
	}
	// applicationId (решение владельца 01.11.2026): дефисы slug → «_»,
	// суффикс <hash8> = первые 8 hex SHA-256(network_id) — тот же хеш,
	// что в label Eq-<hash8>
	wantAppID := "io.equality.monte_cloud.x" + strings.TrimPrefix(wantLabel, "Eq-")
	if aj["application_id"] != wantAppID {
		t.Fatalf("application_id: %v (want %s)", aj["application_id"], wantAppID)
	}
	icon := aj["icon"].(map[string]any)
	if icon["badge"] != true || icon["foreground"] != "identicon" {
		t.Fatalf("icon: %v", icon)
	}
	if _, err := os.Stat(filepath.Join(out, "icon", "foreground.png")); err != nil {
		t.Fatalf("foreground.png: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "icon", "background.png")); err != nil {
		t.Fatalf("background.png: %v", err)
	}
}

// TestConfigureProfile8Std — тот же шаблон со recurring=off: std-сборка,
// имя APK с суффиксом std, бейджа нет.
func TestConfigureProfile8Std(t *testing.T) {
	gp, aj, _ := configureRun(t, "-slug", "monte-cloud", "-recurring", "off")
	if gp.Manifest.Profile.Recurring {
		t.Fatalf("recurring обязан быть off")
	}
	if aj["apk_name"] != "equality-monte-cloud-std-v1.0.0.apk" {
		t.Fatalf("apk_name: %v", aj["apk_name"])
	}
	icon := aj["icon"].(map[string]any)
	if icon["badge"] != false {
		t.Fatalf("бейдж у std-сборки обязан отсутствовать: %v", icon)
	}
}

// TestConfigureGlobalRecurringRejected — глобальная сеть: recurring=on
// отвергается при сборке (комбинации global+rec не существует).
func TestConfigureGlobalRecurringRejected(t *testing.T) {
	err := configureCmd([]string{"-template", "8", "-global",
		"-out", filepath.Join(t.TempDir(), "net"),
		"-signing-dir", filepath.Join(t.TempDir(), "signing")})
	if err == nil || !strings.Contains(err.Error(), "глобальной сети") {
		t.Fatalf("global+recurring обязан отвергаться, got: %v", err)
	}
}

// TestConfigureGlobalStd — глобальная сеть без recurring собирается:
// network_id = "global" (константа), applicationId io.equality.global,
// имя APK equality-global-std-...apk.
func TestConfigureGlobalStd(t *testing.T) {
	out := filepath.Join(t.TempDir(), "net")
	if err := configureCmd([]string{"-template", "7", "-global", "-out", out,
		"-signing-dir", filepath.Join(t.TempDir(), "signing")}); err != nil {
		t.Fatalf("global: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "genesis.bin"))
	if err != nil {
		t.Fatal(err)
	}
	gp, err := wire.DecodeGenesisPackage(b)
	if err != nil {
		t.Fatal(err)
	}
	if !wire.IsGlobalNetwork(gp.Genesis.NetworkID) ||
		!wire.IsGlobalNetwork(gp.Manifest.NetworkID) {
		t.Fatalf("network_id глобальной сети: %x", gp.Genesis.NetworkID)
	}
	ajb, err := os.ReadFile(filepath.Join(out, "android.json"))
	if err != nil {
		t.Fatal(err)
	}
	var aj map[string]any
	if err := json.Unmarshal(ajb, &aj); err != nil {
		t.Fatal(err)
	}
	// applicationId глобальной сети: io.equality.global.<hash8>,
	// <hash8> — от константы network_id = "global" (тот же хеш,
	// что в label Eq-<hash8>)
	wantAppID := "io.equality.global.x" + strings.TrimPrefix(wire.EqLabel(wire.GlobalNetworkID), "Eq-")
	if aj["application_id"] != wantAppID {
		t.Fatalf("application_id: %v (want %s)", aj["application_id"], wantAppID)
	}
	if aj["apk_name"] != "equality-global-std-v1.0.0.apk" {
		t.Fatalf("apk_name: %v", aj["apk_name"])
	}
	if aj["label"] != wire.EqLabel(wire.GlobalNetworkID) {
		t.Fatalf("label глобальной сети: %v", aj["label"])
	}
}

// TestConfigureBadSlug — slug валидируется по решению владельца 01.11.2026:
// ^[a-z][a-z0-9-]{2,23}$ — кириллица, верхний регистр, старт с цифры
// и длина вне 3–24 невалидны; дефисы валидны (санитизируются в «_»
// только для applicationId, имя APK сохраняет дефисы).
func TestConfigureBadSlug(t *testing.T) {
	for _, slug := range []string{
		"",                          // пустой — обязан быть -slug
		"Monte",                     // верхний регистр
		"mon te",                    // пробел
		"1monte",                    // старт с цифры
		"-monte",                    // старт с дефиса
		"москва",                    // кириллица
		"ab",                        // короче 3
		"abcdefghijklmnopqrstuvwxy", // 25 символов — длиннее 24
	} {
		err := configureCmd([]string{"-template", "8", "-slug", slug,
			"-out", filepath.Join(t.TempDir(), "net"),
			"-signing-dir", filepath.Join(t.TempDir(), "signing")})
		if err == nil {
			t.Fatalf("slug %q обязан отвергаться", slug)
		}
	}
}
