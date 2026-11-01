// Этап 14, A2: синтетический корень аттестации допустим только для
// genesis с sandbox = true; не-sandbox сеть — reject в NewNode.
// Тестовый обход — AllowSyntheticRoots (unit-тесты, headless-утилиты).
package net

import (
	"testing"

	"crypto/x509"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func TestSyntheticRootOutsideSandbox(t *testing.T) {
	e := newTestEnv(t)
	base := func() *NodeConfig {
		return &NodeConfig{
			Genesis:     e.gp,
			NodeKey:     crypto.DeterministicKey("syn-root-test"),
			ListenAddrs: []string{"/ip4/127.0.0.1/udp/0/quic-v1"},
			// синтетический корень тестового окружения
			TrustedRoots: []*x509.Certificate{e.rootCert},
		}
	}
	// не-sandbox genesis + синтетический корень — reject
	if _, err := NewNode(base()); err == nil {
		t.Fatal("не-sandbox genesis с синтетическим корнем принят")
	}
	// test-only обход (headless-утилиты)
	cfg := base()
	cfg.AllowSyntheticRoots = true
	if _, err := NewNode(cfg); err != nil {
		t.Fatalf("AllowSyntheticRoots: %v", err)
	}
	// sandbox genesis + синтетический корень — прод-путь песочницы
	e.gp.Manifest.Profile.Sandbox = true
	wire.SetManifestHash(e.gp)
	if _, err := NewNode(base()); err != nil {
		t.Fatalf("sandbox genesis с синтетическим корнем отклонён: %v", err)
	}
}
