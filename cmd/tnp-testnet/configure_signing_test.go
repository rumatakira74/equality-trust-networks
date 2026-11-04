// Тесты боевой release-подписи (тема A8): выбор ключа по типу сети
// (std → tnp-std, rec → tnp-rec; глобальная — std), генерация
// signing.properties из credentials.txt (chmod 600, вне git), явная
// диагностика при отсутствии ключей. Пароли в тестах — фиктивные.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSigningDir — временный signing-каталог с фиктивными keystores
// (файлы-заглушки; содержимое не читается configure) и credentials.txt.
func fakeSigningDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "signing")
	if err := fakeSigningFiles(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestSigningKind — вид боевого ключа по типу сети: rec-сети — rec,
// std-сети и глобальная — std.
func TestSigningKind(t *testing.T) {
	if got := signingKind(true); got != "rec" {
		t.Fatalf("signingKind(true) = %q, want rec", got)
	}
	if got := signingKind(false); got != "std" {
		t.Fatalf("signingKind(false) = %q, want std", got)
	}
}

// TestConfigureSigningProperties — configure генерирует signing.properties
// (chmod 600) с путями обоих keystores и паролями из credentials.txt;
// android.json получает key_kind (rec-сеть) и путь signing-конфига.
func TestConfigureSigningProperties(t *testing.T) {
	sgn := fakeSigningDir(t)
	gp, aj, _ := configureRun(t, "-slug", "monte-cloud", "-signing-dir", sgn)
	if !gp.Manifest.Profile.Recurring {
		t.Fatalf("шаблон 8: recurring обязан быть on")
	}
	if aj["key_kind"] != "rec" {
		t.Fatalf("key_kind: %v (want rec)", aj["key_kind"])
	}
	propsPath, _ := aj["signing_properties"].(string)
	if propsPath != filepath.Join(sgn, "signing.properties") {
		t.Fatalf("signing_properties: %v", aj["signing_properties"])
	}
	fi, err := os.Stat(propsPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("signing.properties: права %v, want 0600", fi.Mode().Perm())
	}
	b, err := os.ReadFile(propsPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		"std.storeFile=" + filepath.Join(sgn, "tnp-std.jks"),
		"std.keyAlias=tnp-std", "std.storePassword=fake-std-store", "std.keyPassword=fake-std-key",
		"rec.storeFile=" + filepath.Join(sgn, "tnp-rec.jks"),
		"rec.keyAlias=tnp-rec", "rec.storePassword=fake-rec-store", "rec.keyPassword=fake-rec-key",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("signing.properties: нет %q", want)
		}
	}
}

// TestConfigureSigningStdKind — std-сеть (recurring=off): key_kind=std.
func TestConfigureSigningStdKind(t *testing.T) {
	sgn := fakeSigningDir(t)
	_, aj, _ := configureRun(t, "-slug", "monte-cloud", "-recurring", "off",
		"-signing-dir", sgn)
	if aj["key_kind"] != "std" {
		t.Fatalf("key_kind: %v (want std)", aj["key_kind"])
	}
}

// TestConfigureSigningGlobalStd — глобальная сеть: key_kind=std
// (комбинации global+rec не существует по genesis-валидации).
func TestConfigureSigningGlobalStd(t *testing.T) {
	sgn := fakeSigningDir(t)
	out := filepath.Join(t.TempDir(), "net")
	if err := configureCmd([]string{"-template", "7", "-global",
		"-out", out, "-signing-dir", sgn}); err != nil {
		t.Fatalf("global: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(out, "android.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"key_kind": "std"`) {
		t.Fatalf("android.json глобальной сети: нет key_kind std:\n%s", b)
	}
}

// TestConfigureSigningMissingKeys — ключей нет: configure собирает
// genesis-артефакты, key_kind выбран, но signing-конфиг не сгенерирован
// (release-сборка будет отвергнута градлом — явная диагностика).
func TestConfigureSigningMissingKeys(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "signing")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	_, aj, _ := configureRun(t, "-slug", "monte-cloud", "-signing-dir", empty)
	if aj["key_kind"] != "rec" {
		t.Fatalf("key_kind: %v (want rec)", aj["key_kind"])
	}
	if _, ok := aj["signing_properties"]; ok {
		t.Fatalf("signing_properties не должен генерироваться без ключей: %v",
			aj["signing_properties"])
	}
	if _, err := os.Stat(filepath.Join(empty, "signing.properties")); err == nil {
		t.Fatalf("signing.properties не должен существовать без ключей")
	}
}
