// Тесты подписанного seed-списка (4.5, шаг 2 каскада): формат полезной
// нагрузки, мультиподпись N-из-M через core/threshold, свежесть.
package net

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
)

// signers — синтетические ключи проекта (реальные — трек «Ключи проекта»).
func signers(t *testing.T, n int) ([]*ecdsa.PrivateKey, map[string]*ecdsa.PublicKey) {
	t.Helper()
	keys := make([]*ecdsa.PrivateKey, n)
	pub := make(map[string]*ecdsa.PublicKey, n)
	for i := range keys {
		keys[i] = crypto.DeterministicKey("seed-signer:" + string(rune('A'+i)))
		pub[string(seedSignerID(keys[i]))] = &keys[i].PublicKey
	}
	return keys, pub
}

func TestSeedListVerifyOK(t *testing.T) {
	all, pubs := signers(t, 3)
	networkID := []byte("tnp-testnet-0001")
	sl := &SeedList{
		NetworkID: networkID,
		Timestamp: uint64(time.Now().Unix()),
		Peers: []peerAddrMsg{
			{ID: "12D3KooWtest1", Addrs: []string{"/ip4/10.0.0.1/tcp/4001"}},
			{ID: "12D3KooWtest2", Addrs: []string{"/ip4/10.0.0.2/udp/4001/quic-v1"}},
		},
	}
	// подписывают 2 из 3 (порог 2)
	blob := SignSeedList(sl, all[:2])
	got, err := VerifySeedList(blob, networkID, 2, pubs, 0)
	if err != nil {
		t.Fatalf("валидный список отклонён: %v", err)
	}
	if len(got.Peers) != 2 {
		t.Fatalf("пиров: %d", len(got.Peers))
	}
	if string(got.NetworkID) != string(networkID) {
		t.Fatal("network_id не совпал")
	}
}

func TestSeedListInsufficientSignatures(t *testing.T) {
	all, pubs := signers(t, 3)
	sl := &SeedList{NetworkID: []byte("tnp-testnet-0001"), Timestamp: uint64(time.Now().Unix())}
	// подписал только 1 из 3, порог 2
	blob := SignSeedList(sl, all[:1])
	if _, err := VerifySeedList(blob, sl.NetworkID, 2, pubs, 0); err == nil {
		t.Fatal("список с недостаточной мультиподписью принят")
	}
	// подпись не из набора проекта
	rogue := crypto.DeterministicKey("rogue")
	blob2 := SignSeedList(sl, []*ecdsa.PrivateKey{rogue, all[0]})
	if _, err := VerifySeedList(blob2, sl.NetworkID, 2, pubs, 0); err == nil {
		t.Fatal("подпись постороннего принята")
	}
	// дубликат подписи одного подписанта не удваивает вес
	blob3 := SignSeedList(sl, []*ecdsa.PrivateKey{all[0], all[0]})
	if _, err := VerifySeedList(blob3, sl.NetworkID, 2, pubs, 0); err == nil {
		t.Fatal("дублированная подпись принята")
	}
}

func TestSeedListTamperedAndStale(t *testing.T) {
	all, pubs := signers(t, 3)
	networkID := []byte("tnp-testnet-0001")
	sl := &SeedList{NetworkID: networkID, Timestamp: uint64(time.Now().Unix())}
	blob := SignSeedList(sl, all[:3])

	// подмена полезной нагрузки после подписания
	tampered := append([]byte(nil), blob...)
	tampered[7] ^= 0xff
	if _, err := VerifySeedList(tampered, networkID, 2, pubs, 0); err == nil {
		t.Fatal("подменённый список принят")
	}
	// устаревший список (свежесть — best-effort, 12.3)
	stale := &SeedList{NetworkID: networkID, Timestamp: uint64(time.Now().Unix() - 8*24*3600)}
	blob2 := SignSeedList(stale, all[:3])
	if _, err := VerifySeedList(blob2, networkID, 2, pubs, uint64(time.Now().Unix())); err == nil {
		t.Fatal("устаревший список принят")
	}
}
