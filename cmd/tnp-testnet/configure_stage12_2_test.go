// Тесты конфигуратора мини-этапа 12.2 (fingerprint-манифест): посевной
// круг — публичные ключи, поставщик — fingerprint, адреса выводятся из
// финального network_id; сборка детерминирована, дубликаты fingerprint'ов
// отвергаются, ядро создаёт посевные кошельки по выведенным адресам.
package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestConfigureFingerprintManifest — сборка на новом wire: манифест
// хранит публичные ключи посевного круга и fingerprint поставщика
// (-provider-seed — удобство: fingerprint посевного кошелька);
// network_id само-сертифицирован (адресов в манифесте нет — цикл
// деривации устранён), адрес посевного выводится из финального
// network_id, ядро создаёт кошельки по выведенным адресам.
func TestConfigureFingerprintManifest(t *testing.T) {
	gp, _, _ := configureRun(t, "-slug", "monte-cloud", "-provider-seed", "V2")
	if err := wire.VerifySelfCertified(gp); err != nil {
		t.Fatalf("network_id не само-сертифицирован: %v", err)
	}
	// посевной круг: только публичные ключи (адресов в wire больше нет)
	if len(gp.Manifest.SeedCircle) != 3 {
		t.Fatalf("посевной круг: %d (нужно 3)", len(gp.Manifest.SeedCircle))
	}
	for i, m := range gp.Manifest.SeedCircle {
		if len(m.PublicKey) == 0 {
			t.Fatalf("посевной круг %d: пустой публичный ключ", i)
		}
	}
	// поставщик — fingerprint посевного кошелька V2
	wantFp := crypto.Fingerprint(crypto.MarshalPubKey(&seedKey("V2").PublicKey))
	provs := gp.Manifest.Profile.Providers
	if len(provs) != 1 || !bytes.Equal(provs[0], wantFp) {
		t.Fatalf("поставщик: %x (нужен fingerprint V2 %x)", provs, wantFp)
	}
	// адрес посевного V2 выводится из финального network_id
	wantAddr := addrOf(seedKey("V2"), gp.Genesis.NetworkID)
	gotAddr := crypto.AddressOfKey(gp.Manifest.SeedCircle[1].PublicKey, gp.Genesis.NetworkID)
	if !bytes.Equal(gotAddr, wantAddr) {
		t.Fatalf("выведенный адрес V2: %x (нужен %x)", gotAddr, wantAddr)
	}
	// ядро: кошельки посевного круга создаются по выведенным адресам
	l, err := ledger.New(gp, ledger.Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("ledger.New: %v", err)
	}
	for _, lbl := range []string{"V1", "V2", "V3"} {
		w := l.State.Wallet(addrOf(seedKey(lbl), gp.Genesis.NetworkID))
		if w == nil || !w.Seed {
			t.Fatalf("кошелёк %s не создан по выведенному адресу", lbl)
		}
	}
}

// TestConfigureDeterministicNetworkID — сборка детерминирована: две
// сборки с одинаковыми параметрами дают одинаковый network_id (адреса
// в манифесте не участвуют — предварительная деривация удалена).
func TestConfigureDeterministicNetworkID(t *testing.T) {
	gp1, _, _ := configureRun(t, "-slug", "monte-cloud", "-genesis-time", "1730000000")
	gp2, _, _ := configureRun(t, "-slug", "monte-cloud", "-genesis-time", "1730000000")
	if !bytes.Equal(gp1.Genesis.NetworkID, gp2.Genesis.NetworkID) {
		t.Fatalf("network_id недетерминирован: %x != %x",
			gp1.Genesis.NetworkID, gp2.Genesis.NetworkID)
	}
}

// TestConfigureDuplicateFingerprints — дубликаты fingerprint'ов
// отвергаются при сборке: повторная метка посевного круга и повторный
// fingerprint поставщика.
func TestConfigureDuplicateFingerprints(t *testing.T) {
	// та же метка посевного круга дважды — дубликат публичного ключа
	err := configureCmd([]string{"-template", "8", "-slug", "monte-cloud",
		"-seeds", "V1,V1", "-out", filepath.Join(t.TempDir(), "net"),
		"-signing-dir", filepath.Join(t.TempDir(), "signing")})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("дубликат fingerprint'а")) {
		t.Fatalf("дубликат посевного круга обязан отвергаться, got: %v", err)
	}
	// один и тот же fingerprint поставщика дважды
	fp := crypto.FingerprintHex(crypto.Fingerprint(crypto.MarshalPubKey(&seedKey("V1").PublicKey)))
	err = configureCmd([]string{"-template", "8", "-slug", "monte-cloud",
		"-providers", fp + "," + fp, "-out", filepath.Join(t.TempDir(), "net"),
		"-signing-dir", filepath.Join(t.TempDir(), "signing")})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("дубликат fingerprint'а")) {
		t.Fatalf("дубликат поставщика обязан отвергаться, got: %v", err)
	}
}
