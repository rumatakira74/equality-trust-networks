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
	"fmt"
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
	return os.WriteFile(filepath.Join(dir, "signing.properties"), []byte(b.String()), 0o600)
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

// setupSigning — точка конфигурации подписи: выбирает ключ по типу сети,
// проверяет наличие/доступность keystores и credentials, генерирует
// signing.properties. Возвращает (вид ключа, путь signing.properties,
// ok, диагностика); при ok=false genesis-артефакты собираются, но
// release-сборка будет отвергнута градлом.
func setupSigning(dir string, recurring bool) (kind, props string, ok bool, diag string) {
	kind = signingKind(recurring)
	for _, k := range []string{"std", "rec"} {
		if err := checkSigningKeystore(filepath.Join(dir, "tnp-"+k+".jks")); err != nil {
			return kind, "", false, fmt.Sprintf(
				"боевой ключ %s недоступен: %s (%v) — release-сборка будет отвергнута (см. docs/RELEASE-KEYS.md)",
				k, filepath.Join(dir, "tnp-"+k+".jks"), err)
		}
	}
	if err := writeSigningProperties(dir); err != nil {
		return kind, "", false, fmt.Sprintf(
			"signing-конфиг не сгенерирован: %v — release-сборка будет отвергнута (см. docs/RELEASE-KEYS.md)", err)
	}
	return kind, filepath.Join(dir, "signing.properties"), true, ""
}
