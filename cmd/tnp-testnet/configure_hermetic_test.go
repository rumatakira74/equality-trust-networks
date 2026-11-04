// Герметичность тестов конфигуратора (тема fix-release-keys, шаг 3; P1
// аудита Sonnet-2 §1.3): без -signing-dir configure читает реальный
// ~/.tnp-signing/credentials.txt и перезаписывает реальный
// signing.properties. Страж TestMain подставляет вместо домашнего каталога
// временный с фиктивными ключами и после прогона пакета проверяет, что
// файлы не тронуты: любой тест, забывший -signing-dir, падает здесь.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fakeSigningFiles — фиктивные боевые ключи и пароли в каталоге dir:
// файлы-заглушки (содержимое keystore configure не читает, отпечаток —
// SHA-256 файла), права 0600 — как у боевых файлов.
func fakeSigningFiles(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, kind := range []string{"std", "rec"} {
		if err := os.WriteFile(filepath.Join(dir, "tnp-"+kind+".jks"),
			[]byte("fake"), 0o600); err != nil {
			return err
		}
	}
	cred := "# фиктивные пароли (тест)\n" +
		"std.key_alias=tnp-std\nstd.store_password=fake-std-store\nstd.key_password=fake-std-key\n" +
		"rec.key_alias=tnp-rec\nrec.store_password=fake-rec-store\nrec.key_password=fake-rec-key\n"
	return os.WriteFile(filepath.Join(dir, "credentials.txt"), []byte(cred), 0o600)
}

// signingDirSnapshot — снимок каталога ключей: имя файла → SHA-256
// содержимого и права. Сверка снимков до и после прогона показывает и
// создание, и перезапись, и смену прав.
func signingDirSnapshot(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	snap := map[string]string{}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		fi, err := e.Info()
		if err != nil {
			return nil, err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		snap[e.Name()] = fmt.Sprintf("%s %04o", hex.EncodeToString(sum[:]), fi.Mode().Perm())
	}
	return snap, nil
}

// diffSnapshots — расхождения снимков в виде строк «файл: было → стало».
func diffSnapshots(before, after map[string]string) []string {
	var diff []string
	names := map[string]bool{}
	for n := range before {
		names[n] = true
	}
	for n := range after {
		names[n] = true
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	for _, n := range sorted {
		b, inBefore := before[n]
		a, inAfter := after[n]
		switch {
		case !inBefore:
			diff = append(diff, fmt.Sprintf("%s: создан", n))
		case !inAfter:
			diff = append(diff, fmt.Sprintf("%s: удалён", n))
		case b != a:
			diff = append(diff, fmt.Sprintf("%s: %s → %s", n, b, a))
		}
	}
	return diff
}

// TestMain — страж герметичности: TNP_SIGNING_DIR на время прогона пакета
// указывает на временный каталог с фиктивными ключами и «меченым»
// signing.properties. Тест, вызвавший configureCmd без -signing-dir,
// прочитает и перезапишет этот каталог — страж падает с расхождением.
func TestMain(m *testing.M) {
	guard, err := os.MkdirTemp("", "tnp-signing-guard-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "страж герметичности: временный каталог: %v\n", err)
		os.Exit(1)
	}
	if err := fakeSigningFiles(guard); err != nil {
		fmt.Fprintf(os.Stderr, "страж герметичности: фиктивные ключи: %v\n", err)
		os.RemoveAll(guard)
		os.Exit(1)
	}
	// предсуществующий signing.properties с меткой: перезапись видна
	props := filepath.Join(guard, "signing.properties")
	const sentinel = "# страж герметичности: файл не должен меняться\n"
	if err := os.WriteFile(props, []byte(sentinel), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "страж герметичности: signing.properties: %v\n", err)
		os.RemoveAll(guard)
		os.Exit(1)
	}
	before, err := signingDirSnapshot(guard)
	if err != nil {
		fmt.Fprintf(os.Stderr, "страж герметичности: снимок: %v\n", err)
		os.RemoveAll(guard)
		os.Exit(1)
	}
	if err := os.Setenv("TNP_SIGNING_DIR", guard); err != nil {
		fmt.Fprintf(os.Stderr, "страж герметичности: TNP_SIGNING_DIR: %v\n", err)
		os.RemoveAll(guard)
		os.Exit(1)
	}

	code := m.Run()

	after, err := signingDirSnapshot(guard)
	if err != nil {
		fmt.Fprintf(os.Stderr, "страж герметичности: снимок после прогона: %v\n", err)
		code = 1
	} else if diff := diffSnapshots(before, after); len(diff) != 0 {
		fmt.Fprintf(os.Stderr,
			"страж герметичности: тесты тронули signing-каталог вне t.TempDir() "+
				"(нужен -signing-dir во всех вызовах configureCmd):\n  %s\n",
			strings.Join(diff, "\n  "))
		code = 1
	}
	os.RemoveAll(guard)
	os.Exit(code)
}

// TestConfigureSigningDirWinsOverEnv — -signing-dir перекрывает
// TNP_SIGNING_DIR: путь из env (каталог по умолчанию) не читается и не
// пишется, signing.properties создаётся во временном каталоге теста.
func TestConfigureSigningDirWinsOverEnv(t *testing.T) {
	envDir := filepath.Join(t.TempDir(), "env-signing")
	if err := fakeSigningFiles(envDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TNP_SIGNING_DIR", envDir)
	before, err := signingDirSnapshot(envDir)
	if err != nil {
		t.Fatal(err)
	}
	sgn := filepath.Join(t.TempDir(), "signing")
	if err := fakeSigningFiles(sgn); err != nil {
		t.Fatal(err)
	}
	_, aj, _ := configureRun(t, "-slug", "monte-cloud", "-signing-dir", sgn)
	if got, _ := aj["signing_properties"].(string); got != filepath.Join(sgn, "signing.properties") {
		t.Fatalf("signing_properties: %v (want %s)", aj["signing_properties"],
			filepath.Join(sgn, "signing.properties"))
	}
	after, err := signingDirSnapshot(envDir)
	if err != nil {
		t.Fatal(err)
	}
	if diff := diffSnapshots(before, after); len(diff) != 0 {
		t.Fatalf("каталог из TNP_SIGNING_DIR изменён: %v", diff)
	}
}
