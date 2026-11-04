// Боевая release-подпись APK (тема A8): конфигуратор выбирает ключ по типу
// сети (std → tnp-std, rec → tnp-rec; глобальная сеть — std, комбинации
// global+rec не существует по genesis-валидации) и генерирует signing-конфиг
// (пути/пароли) в локальном каталоге вне git (дефолт ~/.tnp-signing/,
// флаг -signing-dir или env TNP_SIGNING_DIR). Пароли читаются из
// credentials.txt (chmod 600; ключи генерируются keytool на машине
// владельца — docs/RELEASE-KEYS.md) и в git/CI/логи не попадают.
// Отсутствие или недоступность ключей — явная диагностика: release-сборку
// без ключей отвергает градл (app/android/app/build.gradle.kts),
// debug-сборки не затронуты.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// signingKind — вид боевого ключа по типу сети (решение владельца):
// единый std для всех std-сетей (включая глобальную), отдельный rec для
// rec-сетей (коммерческая сборка, 16.2).
func signingKind(recurring bool) string {
	if recurring {
		return "rec"
	}
	return "std"
}

// signingDirDefault — каталог боевых ключей: env TNP_SIGNING_DIR или
// ~/.tnp-signing (дефолт темы A8; переопределяется флагом -signing-dir).
func signingDirDefault() string {
	if d := os.Getenv("TNP_SIGNING_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".tnp-signing"
	}
	return filepath.Join(home, ".tnp-signing")
}

// signingKeySHA256 — SHA-256 файла боевого ключа (keystore) в hex.
// Отпечаток сверяется при сборке со значением TNP_SIGNING_KEY_SHA256
// (тема fix-release-keys, P1 аудита Sonnet-2): подмена keystore — это
// подпись чужим ключом, и молча она проходить не должна.
func signingKeySHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// checkSigningPerms — права файла ключей не шире 0600 (только чтение и
// запись владельца; доступ группы, остальных и исполнение отвергаются).
// os.WriteFile(0o600) задаёт права только при создании — скопированный,
// восстановленный из бэкапа или сгенерированный keytool файл сохраняет
// чужие права, и пароли рядом с ключом становятся читаемыми.
func checkSigningPerms(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if perm := fi.Mode().Perm(); perm&^0o600 != 0 {
		return fmt.Errorf("права %04o шире 0600 (только rw владельца): %s — нужен chmod 600",
			perm, path)
	}
	return nil
}

// verifySigningKey — сверка отпечатка файла ключа с ожидаемым (hex,
// регистр не важен). Пустой ожидаемый отпечаток — ошибка: сборка без
// ожидаемого значения не подтверждает, что APK подписан своим ключом.
func verifySigningKey(path, expected string) error {
	expected = strings.ToLower(strings.TrimSpace(expected))
	if expected == "" {
		return fmt.Errorf("отпечаток ключа не задан (TNP_SIGNING_KEY_SHA256)")
	}
	got, err := signingKeySHA256(path)
	if err != nil {
		return err
	}
	if got != expected {
		return fmt.Errorf("отпечаток ключа не совпал: %s → %s, ожидался %s", path, got, expected)
	}
	return nil
}

// readSigningCredentials — пароли из credentials.txt (строки key=value,
// комментарии с «#»). Файл chmod 600, вне git.
func readSigningCredentials(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cred := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		cred[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return cred, nil
}

// writeSigningProperties — генерация signing.properties (chmod 600, вне
// git): абсолютные пути к обоим keystores + пароли из credentials.txt.
// Вид ключа (std|rec) сборка выбирает через TNP_KEY_KIND; configure
// печатает готовую команду сборки.
func writeSigningProperties(dir string) error {
	cred, err := readSigningCredentials(filepath.Join(dir, "credentials.txt"))
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Equality release signing (сгенерирован configure, тема A8)\n")
	fmt.Fprintf(&b, "# chmod 600; вне git; пароли из credentials.txt\n")
	for _, kind := range []string{"std", "rec"} {
		alias := cred[kind+".key_alias"]
		store := cred[kind+".store_password"]
		key := cred[kind+".key_password"]
		if alias == "" || store == "" || key == "" {
			return fmt.Errorf("credentials.txt: нет %s.key_alias/%s.store_password/%s.key_password",
				kind, kind, kind)
		}
		abs, err := filepath.Abs(filepath.Join(dir, "tnp-"+kind+".jks"))
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "%s.storeFile=%s\n%s.storePassword=%s\n%s.keyAlias=%s\n%s.keyPassword=%s\n",
			kind, abs, kind, store, kind, alias, kind, key)
	}
	props := filepath.Join(dir, "signing.properties")
	if err := os.WriteFile(props, []byte(b.String()), 0o600); err != nil {
		return err
	}
	// права выставляются и для уже существующего файла (тема
	// fix-release-keys, шаг 2): os.WriteFile задаёт их только при создании,
	// а восстановленный из бэкапа signing.properties сохраняет чужие права
	return os.Chmod(props, 0o600)
}

// checkSigningKeystore — боевой keystore существует и читаем
// (недоступность — явная диагностика, отказ release-сборки).
func checkSigningKeystore(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return f.Close()
}

// signingSetup — результат настройки подписи: вид ключа, путь
// signing.properties, отпечаток файла ключа (SHA-256 — для
// TNP_SIGNING_KEY_SHA256), ok и диагностика. При ok=false genesis-артефакты
// собираются, но release-сборка будет отвергнута градлом.
type signingSetup struct {
	kind        string
	props       string
	fingerprint string
	ok          bool
	diag        string
}

// setupSigning — точка конфигурации подписи: выбирает ключ по типу сети,
// проверяет наличие/доступность keystores, права файлов ключей и отпечаток
// (TNP_SIGNING_KEY_SHA256, если задан), генерирует signing.properties.
func setupSigning(dir string, recurring bool) signingSetup {
	kind := signingKind(recurring)
	for _, k := range []string{"std", "rec"} {
		if err := checkSigningKeystore(filepath.Join(dir, "tnp-"+k+".jks")); err != nil {
			return signingSetup{kind: kind, diag: fmt.Sprintf(
				"боевой ключ %s недоступен: %s (%v) — release-сборка будет отвергнута (см. docs/RELEASE-KEYS.md)",
				k, filepath.Join(dir, "tnp-"+k+".jks"), err)}
		}
	}
	// права файлов ключей (шаг 2): пароли лежат рядом с keystore открытым
	// текстом — доступ группы/остальных к любому из файлов отвергается
	for _, f := range []string{"credentials.txt", "tnp-std.jks", "tnp-rec.jks"} {
		path := filepath.Join(dir, f)
		if _, statErr := os.Stat(path); statErr != nil {
			continue // отсутствие диагностируют checkSigningKeystore и writeSigningProperties
		}
		if err := checkSigningPerms(path); err != nil {
			return signingSetup{kind: kind, diag: fmt.Sprintf(
				"права файлов ключей: %v — release-сборка будет отвергнута (см. docs/RELEASE-KEYS.md)", err)}
		}
	}
	store := filepath.Join(dir, "tnp-"+kind+".jks")
	// отпечаток ключа (шаг 1): печатается для TNP_SIGNING_KEY_SHA256;
	// заданный env сверяется здесь же — расхождение видно до сборки
	fingerprint, err := signingKeySHA256(store)
	if err != nil {
		return signingSetup{kind: kind, diag: fmt.Sprintf(
			"отпечаток ключа %s не вычислен: %v — release-сборка будет отвергнута", store, err)}
	}
	if expected := os.Getenv("TNP_SIGNING_KEY_SHA256"); expected != "" {
		if err := verifySigningKey(store, expected); err != nil {
			return signingSetup{kind: kind, fingerprint: fingerprint, diag: fmt.Sprintf(
				"%v — release-сборка будет отвергнута (отпечаток печатает configure, сверьте с docs/RELEASE-KEYS.md)", err)}
		}
	}
	if err := writeSigningProperties(dir); err != nil {
		return signingSetup{kind: kind, fingerprint: fingerprint, diag: fmt.Sprintf(
			"signing-конфиг не сгенерирован: %v — release-сборка будет отвергнута (см. docs/RELEASE-KEYS.md)", err)}
	}
	return signingSetup{kind: kind, props: filepath.Join(dir, "signing.properties"),
		fingerprint: fingerprint, ok: true}
}
