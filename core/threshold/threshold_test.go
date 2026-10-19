// Тесты N-из-M набора подписей (решение этапа 0, spec/07 7.2).
package threshold

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func TestThresholdNOfM(t *testing.T) {
	// 5 участников, порог 3
	type member struct {
		priv *ecdsa.PrivateKey
		addr []byte
	}
	var members []member
	participants := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		priv, err := ecdsa.GenerateKey(ecdsaP256(), rand.Reader)
		if err != nil {
			t.Fatalf("key: %v", err)
		}
		m := member{priv: priv, addr: []byte{byte(i)}}
		members = append(members, m)
		participants[i] = m.addr
	}
	hash := sha256Sum([]byte("checkpoint"))
	pubOf := func(addr []byte) (*ecdsa.PublicKey, bool) {
		for _, m := range members {
			if string(m.addr) == string(addr) {
				return &m.priv.PublicKey, true
			}
		}
		return nil, false
	}
	sign := func(i int) wire.SignatureShare {
		sig, _ := crypto.SignDER(members[i].priv, hash[:])
		return wire.SignatureShare{Signer: members[i].addr, Signature: sig}
	}

	// 3 из 5 — достаточно
	sigs := []wire.SignatureShare{sign(0), sign(1), sign(2)}
	if v := Verify(hash, participants, 3, sigs, pubOf); !Meets(uint64(v), 3) {
		t.Fatalf("3 из 5 не прошли: %d", v)
	}
	// 2 из 5 — нет
	sigs = []wire.SignatureShare{sign(0), sign(1)}
	if v := Verify(hash, participants, 3, sigs, pubOf); Meets(uint64(v), 3) {
		t.Fatalf("2 из 5 прошли: %d", v)
	}
	// подписант вне набора не считается
	outsider, _ := ecdsa.GenerateKey(ecdsaP256(), rand.Reader)
	osig, _ := crypto.SignDER(outsider, hash[:])
	sigs = []wire.SignatureShare{sign(0), sign(1), {Signer: []byte{99}, Signature: osig}}
	if v := Verify(hash, participants, 3, sigs, pubOf); Meets(uint64(v), 3) {
		t.Fatalf("посторонний подписант прошёл: %d", v)
	}
	// дубликат подписанта не считается дважды
	sigs = []wire.SignatureShare{sign(0), sign(0), sign(0)}
	if v := Verify(hash, participants, 3, sigs, pubOf); v != 1 {
		t.Fatalf("дубликат посчитан: %d", v)
	}
}
