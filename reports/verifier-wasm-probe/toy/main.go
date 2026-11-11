// Игрушечный пример WASM-пробы верификатора (предпроектная работа,
// решение владельца 11.11.2026): минимальный пакет, импортирующий
// crypto/wire-пакеты ядра как go-модульную зависимость, и собираемый
// под GOOS=js GOARCH=wasm.
//
// Это НЕ продуктовый код: игрушечный пример живёт только в
// reports/verifier-wasm-probe/ и в продуктовые пакеты не переносится.
// Проверяется сборка (компиляция под js/wasm) и работоспособность путей
// verify-контура на нативном прогоне; P2P-пакеты (net/, mobile/) в сферу
// пробы не входят.
package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/present"
	"github.com/rumatakira74/equality-trust-networks/core/smt"
	"github.com/rumatakira74/equality-trust-networks/core/threshold"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func main() {
	// 1. Ключи/адрес ядра (core/crypto): fingerprint + деривация адреса.
	k, err := crypto.GenerateKey()
	if err != nil {
		fail(err)
	}
	pub := crypto.MarshalPubKey(&k.PublicKey)
	fp := crypto.Fingerprint(pub)
	networkID := []byte("probe-network-id-32-bytes-example!!")
	addr := crypto.DeriveAddress(fp, networkID)
	fmt.Printf("address: %x\n", addr)

	// 2. Каноническое утверждение предъявления (core/wire, spec/01) и
	// roundtrip кодека листа состояния (canonical protobuf-подобный).
	stmt := &wire.AttestStatement{
		NetworkID:    networkID,
		Address:      addr,
		Status:       wire.StatusActive,
		AgeDays:      42,
		VouchDepth:   2,
		Flagged:      false,
		ManifestHash: []byte("manifest-hash-probe"),
		Epoch:        1,
		Challenge:    []byte("challenge-probe"),
		ExpiresAt:    1_800_000_000,
		AppID:        []byte("app-probe"),
	}
	h := stmt.Hash()
	fmt.Printf("statement hash: %x\n", h[:])

	leaf := &wire.StateLeaf{Seq: 1, Balance: 5, Status: wire.StatusActive, RegisteredEpoch: 1}
	leaf2, err := wire.DecodeStateLeaf(leaf.Encode())
	if err != nil {
		fail(err)
	}
	fmt.Printf("state leaf roundtrip: seq=%d status=%s\n", leaf2.Seq, leaf2.Status)

	// 3. Подпись ECDSA P-256 DER над хешем утверждения (core/crypto).
	sig, err := crypto.SignDER(k, h[:])
	if err != nil {
		fail(err)
	}
	fmt.Printf("wallet signature verifies: %v\n", crypto.VerifyDER(&k.PublicKey, h[:], sig))

	// 4. Идентификатор «один человек — одно действие» (core/present, 6.1).
	fmt.Printf("app identity: %x\n", present.AppIdentity(addr, stmt.AppID))

	// 5. Merkle-доказательство (core/smt): дерево из одного листа,
	// Prove + VerifyProof против корня — полный путь верификатора.
	tree := smt.New()
	var idx [32]byte
	copy(idx[:], addr)
	leafHash := sha256.Sum256(leaf.Encode())
	tree.SetLeaf(idx, leafHash)
	root := tree.Root()
	sibs, lh, found := tree.Prove(idx)
	fmt.Printf("smt proof found=%v verifies=%v (siblings=%d)\n",
		found, smt.VerifyProof(idx, lh, sibs, root), len(sibs))

	// 6. Чекпоинт с пороговой подписью N-из-M (cons + core/threshold):
	// подпись подписанта (cons.Signer), сборка чекпоинта, проверка
	// cons.Verify (цепочка + порог) и threshold.Verify напрямую.
	cp := &wire.Checkpoint{
		NetworkID:          networkID,
		Epoch:              1,
		StateRoot:          root[:],
		PrevCheckpointHash: make([]byte, 32),
		NetworkTime:        1_000_000 + 86_400,
		ProtocolVersion:    1,
		Participants:       [][]byte{addr},
		Threshold:          1,
	}
	share, err := (&cons.Signer{Wallet: addr, Priv: k}).Sign(cp)
	if err != nil {
		fail(err)
	}
	cp.ThresholdSig = &wire.ThresholdSignature{Signatures: []wire.SignatureShare{share}}
	pubOf := func(a []byte) (*ecdsa.PublicKey, bool) { return &k.PublicKey, true }
	err = cons.Verify(cp, nil, 1_000_000, 86_400, 600, pubOf)
	fmt.Printf("cons.Verify: %v\n", err)
	valid := threshold.Verify(cp.CheckpointHash(), cp.Participants, cp.Threshold,
		cp.ThresholdSig.Signatures, pubOf)
	fmt.Printf("threshold valid signatures: %d\n", valid)

	// 7. present.Verify — пустое предъявление отвергается (полный вход
	// требует состояния сети; для пробы достаточно пути отказа).
	if _, err := present.Verify(nil, nil, nil, nil, nil, 0, nil); err != nil {
		fmt.Printf("present.Verify(nil) rejected as expected: %v\n", err)
	}
	fmt.Println("toy probe done")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "probe failed:", err)
	os.Exit(1)
}
