// Тесты сверки отпечатка и прав файлов ключей (тема fix-release-keys,
// шаги 1–2; P1 аудита Sonnet-2 §1.4, §1.7): SHA-256 файла ключа против
// TNP_SIGNING_KEY_SHA256, права ≤0600, диагностика configure при
// расхождении. Ключи фиктивные, пароли фиктивные, каталоги — временные.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSigningKeySHA256 — отпечаток файла ключа: hex в нижнем регистре,
// 64 символа, совпадает с SHA-256 содержимого; отсутствующий файл —
// ошибка, а не пустой отпечаток.
func TestSigningKeySHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tnp-std.jks")
	content := []byte("tnp-test-key-material")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])
	got, err := signingKeySHA256(path)
	if err != nil {
		t.Fatalf("отпечаток: %v", err)
	}
	if got != want {
		t.Fatalf("отпечаток %s != %s", got, want)
	}
	if len(got) != 64 || strings.ToLower(got) != got {
		t.Fatalf("формат отпечатка: %q (нужен hex нижнего регистра, 64 символа)", got)
	}
	if _, err := signingKeySHA256(filepath.Join(dir, "нет-такого.jks")); err == nil {
		t.Fatal("отпечаток отсутствующего файла обязан быть ошибкой")
	}
}

// TestVerifySigningKey — сверка отпечатка: совпадение (регистр hex не
// важен, пробелы по краям срезаются), расхождение (в сообщении оба
// значения), пустой ожидаемый отпечаток — ошибка.
func TestVerifySigningKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tnp-rec.jks")
	content := []byte("tnp-test-key-material")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	fp := hex.EncodeToString(sum[:])

	t.Run("совпал", func(t *testing.T) {
		if err := verifySigningKey(path, fp); err != nil {
			t.Fatalf("совпадающий отпечаток отвергнут: %v", err)
		}
	})
	t.Run("совпал — верхний регистр и пробелы", func(t *testing.T) {
		if err := verifySigningKey(path, "  "+strings.ToUpper(fp)+"\n"); err != nil {
			t.Fatalf("отпечаток в верхнем регистре отвергнут: %v", err)
		}
	})
	t.Run("не совпал", func(t *testing.T) {
		err := verifySigningKey(path, strings.Repeat("0", 64))
		if err == nil {
			t.Fatal("чужой отпечаток обязан отвергаться")
		}
		for _, want := range []string{"не совпал", fp, strings.Repeat("0", 64)} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("сообщение %q: нет %q", err, want)
			}
		}
	})
	t.Run("отпечаток не задан", func(t *testing.T) {
		err := verifySigningKey(path, "   ")
		if err == nil || !strings.Contains(err.Error(), "TNP_SIGNING_KEY_SHA256") {
			t.Fatalf("пустой отпечаток обязан отвергаться, got: %v", err)
		}
	})
	t.Run("файла нет", func(t *testing.T) {
		if err := verifySigningKey(filepath.Join(dir, "нет-такого.jks"), fp); err == nil {
			t.Fatal("отпечаток отсутствующего файла обязан быть ошибкой")
		}
	})
}

// TestCheckSigningPerms — права файлов ключей не шире 0600 (только rw
// владельца): 0600/0400 проходят, доступ группы и остальных (0644, 0640,
// 0604, 0660, 0700) отвергается с подсказкой chmod 600.
func TestCheckSigningPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.txt")
	if err := os.WriteFile(path, []byte("std.key_alias=tnp-std\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, perm := range []os.FileMode{0o600, 0o400} {
		if err := os.Chmod(path, perm); err != nil {
			t.Fatal(err)
		}
		if err := checkSigningPerms(path); err != nil {
			t.Fatalf("права %04o отвергнуты: %v", perm, err)
		}
	}
	for _, perm := range []os.FileMode{0o644, 0o640, 0o604, 0o660, 0o700, 0o777} {
		if err := os.Chmod(path, perm); err != nil {
			t.Fatal(err)
		}
		err := checkSigningPerms(path)
		if err == nil {
			t.Fatalf("права %04o обязаны отвергаться", perm)
		}
		if !strings.Contains(err.Error(), "chmod 600") {
			t.Fatalf("сообщение для %04o: %q (нет подсказки chmod 600)", perm, err)
		}
	}
	if err := checkSigningPerms(filepath.Join(dir, "нет-такого.jks")); err == nil {
		t.Fatal("права отсутствующего файла обязаны быть ошибкой")
	}
}

// TestSetupSigningFingerprint — сверка отпечатка на точке конфигурации
// подписи: совпадение с TNP_SIGNING_KEY_SHA256 — signing-конфиг собран,
// расхождение — ok=false с явной диагностикой, пустой env — проверки нет.
func TestSetupSigningFingerprint(t *testing.T) {
	dir := fakeSigningDir(t)
	fp, err := signingKeySHA256(filepath.Join(dir, "tnp-rec.jks"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("отпечаток совпал", func(t *testing.T) {
		t.Setenv("TNP_SIGNING_KEY_SHA256", strings.ToUpper(fp))
		sgn := setupSigning(dir, true)
		if !sgn.ok {
			t.Fatalf("ok=false: %s", sgn.diag)
		}
		if sgn.kind != "rec" {
			t.Fatalf("вид ключа %q (want rec)", sgn.kind)
		}
		if sgn.fingerprint != fp {
			t.Fatalf("отпечаток %q (want %q)", sgn.fingerprint, fp)
		}
		if sgn.props != filepath.Join(dir, "signing.properties") {
			t.Fatalf("signing.properties: %q", sgn.props)
		}
	})

	t.Run("отпечаток не совпал", func(t *testing.T) {
		t.Setenv("TNP_SIGNING_KEY_SHA256", strings.Repeat("0", 64))
		sgn := setupSigning(dir, true)
		if sgn.ok {
			t.Fatal("чужой отпечаток обязан отвергаться")
		}
		if !strings.Contains(sgn.diag, "отпечаток ключа не совпал") {
			t.Fatalf("диагностика: %q", sgn.diag)
		}
		if sgn.fingerprint != fp {
			t.Fatalf("фактический отпечаток %q (want %q)", sgn.fingerprint, fp)
		}
	})

	t.Run("отпечаток не задан — проверки нет", func(t *testing.T) {
		sgn := setupSigning(dir, true)
		if !sgn.ok {
			t.Fatalf("ok=false без TNP_SIGNING_KEY_SHA256: %s", sgn.diag)
		}
		if sgn.fingerprint != fp {
			t.Fatalf("отпечаток %q (want %q)", sgn.fingerprint, fp)
		}
	})
}

// TestSetupSigningPermsRejected — права шире 0600 у файла ключей
// отвергают signing-конфиг с явной диагностикой (release-сборка будет
// отвергнута градлом, genesis-артефакты собираются).
func TestSetupSigningPermsRejected(t *testing.T) {
	for _, name := range []string{"credentials.txt", "tnp-std.jks", "tnp-rec.jks"} {
		t.Run(name, func(t *testing.T) {
			dir := fakeSigningDir(t)
			if err := os.Chmod(filepath.Join(dir, name), 0o644); err != nil {
				t.Fatal(err)
			}
			sgn := setupSigning(dir, false)
			if sgn.ok {
				t.Fatalf("права 0644 у %s обязаны отвергаться", name)
			}
			for _, want := range []string{name, "chmod 600", "release-сборка будет отвергнута"} {
				if !strings.Contains(sgn.diag, want) {
					t.Fatalf("диагностика %q: нет %q", sgn.diag, want)
				}
			}
		})
	}
}

// TestWriteSigningPropertiesChmod — права уже существующего
// signing.properties исправляются на 0600 (os.WriteFile задаёт права
// только при создании; скопированный файл сохраняет чужие).
func TestWriteSigningPropertiesChmod(t *testing.T) {
	dir := fakeSigningDir(t)
	props := filepath.Join(dir, "signing.properties")
	if err := os.WriteFile(props, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(props, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSigningProperties(dir); err != nil {
		t.Fatalf("signing.properties: %v", err)
	}
	fi, err := os.Stat(props)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("права signing.properties %04o (want 0600)", fi.Mode().Perm())
	}
}

// TestConfigureSigningFingerprintEnv — configure с заданным
// TNP_SIGNING_KEY_SHA256: совпадение — signing_properties в android.json,
// расхождение — конфига нет (release-сборка будет отвергнута).
func TestConfigureSigningFingerprintEnv(t *testing.T) {
	sgn := fakeSigningDir(t)
	fp, err := signingKeySHA256(filepath.Join(sgn, "tnp-rec.jks"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TNP_SIGNING_KEY_SHA256", fp)
	_, aj, _ := configureRun(t, "-slug", "monte-cloud", "-signing-dir", sgn)
	if got, _ := aj["signing_properties"].(string); got != filepath.Join(sgn, "signing.properties") {
		t.Fatalf("совпадающий отпечаток: signing_properties %v", aj["signing_properties"])
	}
	t.Setenv("TNP_SIGNING_KEY_SHA256", strings.Repeat("a", 64))
	_, aj2, _ := configureRun(t, "-slug", "monte-cloud", "-signing-dir", sgn)
	if got, ok := aj2["signing_properties"]; ok {
		t.Fatalf("чужой отпечаток: signing_properties не должен генерироваться: %v", got)
	}
}
