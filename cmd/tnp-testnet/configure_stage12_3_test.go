// Тесты мини-этапа 12.3 (решение владельца 01.11.2026): applicationId
// io.equality.<sanitized-slug>.x<hash8> — префикс «x» перед hash8
// ставится всегда: сегмент Android-пакета обязан начинаться с буквы,
// hash8 начинается с hex-цифры в 10/16 случаев (находка прогона 12.2:
// AAPT отверг io.equality.cirrus.783da646). Label Eq-<hash8>, имена
// APK и display_name не меняются.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// net783da646 — network_id сети из прогона 12.2 (slug cirrus,
// genesis-time 1790807400), на которой AAPT отверг applicationId
// io.equality.cirrus.783da646: hash8 начинается с цифры.
var net783da646 = mustHex("7cd5b877e09c4b38f566d64547df7abff18fec115d2523a09b580a9f7005c1b0")

// netE5796f9c — сеть прогона 12.2 после обхода «подбор с буквы»
// (slug cirrus, genesis-time 1788326486): hash8 начинается с буквы.
var netE5796f9c = mustHex("7aab00e60210a766f327d99fcd0a446844c686a2bf58dad7e607227597f533c9")

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// validAndroidPackage — каждый сегмент имени пакета начинается с буквы
// (правило Android, AAPT); используется для проверки applicationId.
func validAndroidPackage(pkg string) bool {
	if pkg == "" {
		return false
	}
	for _, seg := range strings.Split(pkg, ".") {
		if seg == "" || seg[0] < 'a' || seg[0] > 'z' {
			return false
		}
	}
	return true
}

// TestApplicationIDLetterPrefix — префикс «x» перед hash8: applicationId
// валиден по правилам Android и для сети с цифрой в начале hash8
// (783da646 — AAPT-находка прогона 12.2), и для сети с буквой
// (e5796f9c); детерминизм — тот же network_id даёт тот же пакет.
func TestApplicationIDLetterPrefix(t *testing.T) {
	cases := []struct {
		name string
		nid  []byte
		slug string
		want string
	}{
		{"цифра в начале hash8 (прогон 12.2)", net783da646, "monte-cloud",
			"io.equality.monte_cloud.x783da646"},
		{"буква в начале hash8", netE5796f9c, "monte-cloud",
			"io.equality.monte_cloud.xe5796f9c"},
	}
	for _, c := range cases {
		got := ApplicationID(c.nid, c.slug)
		if got != c.want {
			t.Fatalf("%s: application_id %q (want %q)", c.name, got, c.want)
		}
		if !validAndroidPackage(got) {
			t.Fatalf("%s: application_id %q невалиден по правилам Android", c.name, got)
		}
		// детерминизм: тот же network_id — тот же пакет
		if again := ApplicationID(c.nid, c.slug); again != got {
			t.Fatalf("%s: application_id недетерминирован: %q != %q", c.name, again, got)
		}
	}
	// глобальная сеть — тоже с префиксом x
	g := ApplicationID(wire.GlobalNetworkID, "global")
	if !validAndroidPackage(g) || !strings.HasPrefix(g, "io.equality.global.x") {
		t.Fatalf("application_id глобальной сети: %q", g)
	}
}

// TestApplicationIDAnyNetworkValid — валидность для любой сети: 64
// детерминированных network_id — каждый applicationId (std/rec, slug с
// дефисами и без, глобальная) проходит правило Android «сегмент с буквы».
func TestApplicationIDAnyNetworkValid(t *testing.T) {
	for i := 0; i < 64; i++ {
		nid := sha256.Sum256([]byte{byte(i), byte(i >> 8), byte(i >> 16)})
		for _, slug := range []string{"monte-cloud", "abc", "global"} {
			id := ApplicationID(nid[:], slug)
			if !validAndroidPackage(id) {
				t.Fatalf("network_id %x slug %q: application_id %q невалиден",
					nid[:], slug, id)
			}
		}
	}
}

// TestApplicationIDUnchangedArtifacts — префикс «x» живёт только в
// applicationId: label Eq-<hash8>, имя APK и display_name от префикса
// не зависят (решение 3 мандата 12.3).
func TestApplicationIDUnchangedArtifacts(t *testing.T) {
	label := wire.EqLabel(net783da646)
	if label != "Eq-783da646" || strings.Contains(label, "x783") {
		t.Fatalf("label изменился: %q", label)
	}
	apk := APKName("monte-cloud", true, "1.0.0")
	if apk != "equality-monte-cloud-rec-v1.0.0.apk" || strings.Contains(apk, "x783") {
		t.Fatalf("имя APK изменилось: %q", apk)
	}
	// hash8 в applicationId совпадает с label (без префикса)
	id := ApplicationID(net783da646, "monte-cloud")
	if !strings.HasSuffix(id, strings.TrimPrefix(label, "Eq-")) {
		t.Fatalf("hash8 разошёлся с label: %q vs %q", id, label)
	}
	// санитизация slug — регрессия 12.1: дефис → «_»
	if !strings.Contains(id, "monte_cloud") || strings.Contains(id, "monte-cloud") {
		t.Fatalf("санитизация slug сломана: %q", id)
	}
	// два разных network_id с равным slug — разные пакеты (хеш в суффиксе)
	if bytes.Equal(net783da646, netE5796f9c) ||
		ApplicationID(net783da646, "monte-cloud") == ApplicationID(netE5796f9c, "monte-cloud") {
		t.Fatalf("разные сети дают одинаковый applicationId")
	}
}
